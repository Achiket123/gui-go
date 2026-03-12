// Package auth — JWT token management, password hashing, and session handling.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/achiket123/taskflow/internal/db"
	"github.com/achiket123/taskflow/internal/models"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	bcryptCost      = 12
)

// ErrInvalidCredentials is returned when email/password don't match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrTokenExpired is returned when a JWT is expired.
var ErrTokenExpired = errors.New("token expired")

// ErrTokenInvalid is returned for malformed or tampered tokens.
var ErrTokenInvalid = errors.New("token invalid")

// Service handles all authentication operations.
type Service struct {
	db        *db.DB
	jwtSecret []byte
}

// NewService creates an auth Service.
func NewService(database *db.DB, jwtSecret string) *Service {
	return &Service{db: database, jwtSecret: []byte(jwtSecret)}
}

// ─── Password ─────────────────────────────────────────────────────────────────

// HashPassword returns a bcrypt hash of password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(b), nil
}

// CheckPassword returns nil when password matches hash.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ─── JWT Claims ───────────────────────────────────────────────────────────────

type jwtClaims struct {
	UserID      string          `json:"uid"`
	Email       string          `json:"email"`
	Role        models.UserRole `json:"role"`
	WorkspaceID string          `json:"wid,omitempty"`
	jwt.RegisteredClaims
}

// ─── Login ────────────────────────────────────────────────────────────────────

// LoginRequest carries credentials.
type LoginRequest struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// Login authenticates a user and issues a token pair.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*models.TokenPair, *models.User, error) {
	user, err := s.userByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.IsActive {
		return nil, nil, ErrInvalidCredentials
	}
	if err := CheckPassword(user.PasswordHash, req.Password); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user, req.UserAgent, req.IPAddress)
	if err != nil {
		return nil, nil, err
	}

	// Update last_login.
	_, _ = s.db.ExecContext(ctx,
		"UPDATE users SET last_login = NOW() WHERE id = ?", user.ID)

	return pair, user, nil
}

// ─── Register ─────────────────────────────────────────────────────────────────

// RegisterRequest carries sign-up data.
type RegisterRequest struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	UserAgent   string
	IPAddress   string
}

// Register creates a new user and returns a token pair.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*models.TokenPair, *models.User, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, nil, err
	}
	user := &models.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		PasswordHash: hash,
		Role:         models.RoleMember,
		IsActive:     true,
	}
	if err := s.insertUser(ctx, user); err != nil {
		return nil, nil, err
	}
	pair, err := s.issueTokenPair(ctx, user, req.UserAgent, req.IPAddress)
	if err != nil {
		return nil, nil, err
	}
	return pair, user, nil
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

// Refresh exchanges a valid refresh token for a new token pair.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken, userAgent, ip string) (*models.TokenPair, error) {
	hash := hashToken(rawRefreshToken)

	var userID string
	var expires time.Time
	var revoked bool
	err := s.db.QueryRowContext(ctx,
		"SELECT user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash = ?", hash,
	).Scan(&userID, &expires, &revoked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenInvalid
		}
		return nil, err
	}
	if revoked || time.Now().After(expires) {
		return nil, ErrTokenExpired
	}

	// Rotate: revoke old token.
	if _, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?", hash); err != nil {
		return nil, err
	}

	user, err := s.userByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, user, userAgent, ip)
}

// ─── Validate ─────────────────────────────────────────────────────────────────

// ValidateAccessToken parses and validates an access JWT.
func (s *Service) ValidateAccessToken(tokenStr string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrTokenInvalid
			}
			return s.jwtSecret, nil
		},
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return &models.Claims{
		UserID: c.UserID,
		Email:  c.Email,
		Role:   c.Role,
	}, nil
}

// ─── Logout ───────────────────────────────────────────────────────────────────

// Logout revokes a specific refresh token.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := hashToken(rawRefreshToken)
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?", hash)
	return err
}

// LogoutAll revokes all refresh tokens for a user.
func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ?", userID)
	return err
}

// ─── Change password ──────────────────────────────────────────────────────────

// ChangePassword verifies the old password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, userID, oldPw, newPw string) error {
	var hash string
	if err := s.db.QueryRowContext(ctx,
		"SELECT password_hash FROM users WHERE id = ?", userID).Scan(&hash); err != nil {
		return err
	}
	if err := CheckPassword(hash, oldPw); err != nil {
		return ErrInvalidCredentials
	}
	newHash, err := HashPassword(newPw)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?", newHash, userID)
	if err != nil {
		return err
	}
	// Invalidate all sessions.
	return s.LogoutAll(ctx, userID)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *Service) issueTokenPair(ctx context.Context, user *models.User, userAgent, ip string) (*models.TokenPair, error) {
	now := time.Now()

	// Access token.
	claims := &jwtClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			Issuer:    "taskflow",
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token.
	rawRefresh, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	hash := hashToken(rawRefresh)
	refreshID := uuid.NewString()
	expires := now.Add(refreshTokenTTL)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, user_agent, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		refreshID, user.ID, hash, expires, userAgent, ip)
	if err != nil {
		return nil, fmt.Errorf("insert refresh token: %w", err)
	}

	// Purge expired tokens for this user (best-effort).
	go func() {
		if _, err := s.db.Exec(
			"DELETE FROM refresh_tokens WHERE user_id = ? AND (expires_at < NOW() OR revoked = 1)",
			user.ID); err != nil {
			log.Printf("[auth] purge expired tokens: %v", err)
		}
	}()

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) userByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, username, display_name, password_hash, COALESCE(avatar_url,''),
		        role, is_active, created_at, updated_at
		 FROM users WHERE email = ? LIMIT 1`, email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash,
		&u.AvatarURL, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) userByID(ctx context.Context, id string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, username, display_name, password_hash, COALESCE(avatar_url,''),
		        role, is_active, created_at, updated_at
		 FROM users WHERE id = ? LIMIT 1`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash,
		&u.AvatarURL, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) insertUser(ctx context.Context, u *models.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, username, display_name, password_hash, role, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Username, u.DisplayName, u.PasswordHash, u.Role, u.IsActive)
	return err
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
