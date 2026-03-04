// Package api — HTTP REST API handlers using only stdlib net/http.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/achiket/taskflow/internal/auth"
	"github.com/achiket/taskflow/internal/models"
	"github.com/achiket/taskflow/internal/service"
)

// ─── Server ───────────────────────────────────────────────────────────────────

type Server struct {
	mux         *http.ServeMux
	authSvc     *auth.Service
	wsService   *service.WorkspaceService
	projService *service.ProjectService
	taskService *service.TaskService
	cmtService  *service.CommentService
	dashService *service.DashboardService
}

func NewServer(
	authSvc *auth.Service,
	ws *service.WorkspaceService,
	proj *service.ProjectService,
	task *service.TaskService,
	cmt *service.CommentService,
	dash *service.DashboardService,
) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		authSvc:     authSvc,
		wsService:   ws,
		projService: proj,
		taskService: task,
		cmtService:  cmt,
		dashService: dash,
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// HTTPServer returns a configured *http.Server ready to call ListenAndServe.
func (s *Server) HTTPServer(addr string) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      cors(s),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// ─── Routes ───────────────────────────────────────────────────────────────────

func (s *Server) registerRoutes() {
	m := s.mux

	// Auth.
	m.HandleFunc("POST /api/auth/register", s.handleRegister)
	m.HandleFunc("POST /api/auth/login",    s.handleLogin)
	m.HandleFunc("POST /api/auth/refresh",  s.handleRefresh)
	m.HandleFunc("POST /api/auth/logout",   s.auth(s.handleLogout))

	// Me.
	m.HandleFunc("GET  /api/me",            s.auth(s.handleMe))
	m.HandleFunc("PUT  /api/me/password",   s.auth(s.handleChangePassword))

	// Workspaces.
	m.HandleFunc("GET  /api/workspaces",        s.auth(s.handleListWorkspaces))
	m.HandleFunc("POST /api/workspaces",        s.auth(s.handleCreateWorkspace))
	m.HandleFunc("GET  /api/workspaces/{id}",   s.auth(s.handleGetWorkspace))

	// Projects.
	m.HandleFunc("GET  /api/workspaces/{wsID}/projects",       s.auth(s.handleListProjects))
	m.HandleFunc("POST /api/workspaces/{wsID}/projects",       s.auth(s.handleCreateProject))
	m.HandleFunc("GET  /api/projects/{id}",                    s.auth(s.handleGetProject))
	m.HandleFunc("PUT  /api/projects/{id}",                    s.auth(s.handleUpdateProject))
	m.HandleFunc("DELETE /api/projects/{id}",                  s.auth(s.handleArchiveProject))

	// Tasks.
	m.HandleFunc("GET  /api/projects/{projID}/tasks",  s.auth(s.handleListTasks))
	m.HandleFunc("POST /api/projects/{projID}/tasks",  s.auth(s.handleCreateTask))
	m.HandleFunc("GET  /api/tasks/{id}",               s.auth(s.handleGetTask))
	m.HandleFunc("PUT  /api/tasks/{id}",               s.auth(s.handleUpdateTask))
	m.HandleFunc("PATCH /api/tasks/{id}/status",       s.auth(s.handleUpdateTaskStatus))
	m.HandleFunc("PATCH /api/tasks/{id}/reorder",      s.auth(s.handleReorderTask))
	m.HandleFunc("DELETE /api/tasks/{id}",             s.auth(s.handleDeleteTask))

	// Comments.
	m.HandleFunc("GET  /api/tasks/{taskID}/comments",      s.auth(s.handleListComments))
	m.HandleFunc("POST /api/tasks/{taskID}/comments",      s.auth(s.handleAddComment))
	m.HandleFunc("DELETE /api/comments/{id}",              s.auth(s.handleDeleteComment))

	// Dashboard.
	m.HandleFunc("GET /api/workspaces/{wsID}/dashboard", s.auth(s.handleDashboard))

	// Health.
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
	})
}

// ─── Auth handlers ────────────────────────────────────────────────────────────

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Email == "" || body.Password == "" || len(body.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "email and password (min 8 chars) required")
		return
	}
	if body.DisplayName == "" {
		body.DisplayName = body.Username
	}
	pair, user, err := s.authSvc.Register(r.Context(), auth.RegisterRequest{
		Email: body.Email, Username: body.Username,
		DisplayName: body.DisplayName, Password: body.Password,
		UserAgent: r.UserAgent(), IPAddress: realIP(r),
	})
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"tokens": pair, "user": user})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	pair, user, err := s.authSvc.Login(r.Context(), auth.LoginRequest{
		Email: body.Email, Password: body.Password,
		UserAgent: r.UserAgent(), IPAddress: realIP(r),
	})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": pair, "user": user})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token required")
		return
	}
	pair, err := s.authSvc.Refresh(r.Context(), body.RefreshToken, r.UserAgent(), realIP(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token expired or invalid")
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	_ = s.authSvc.Logout(r.Context(), body.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	writeJSON(w, http.StatusOK, claims)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := s.authSvc.ChangePassword(r.Context(), claims.UserID, body.OldPassword, body.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

// ─── Workspace handlers ───────────────────────────────────────────────────────

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	list, err := s.wsService.ListForUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	ws, err := s.wsService.Create(r.Context(), body.Name, body.Description, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ws, err := s.wsService.ListForUser(r.Context(), claimsFrom(r.Context()).UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, w2 := range ws {
		if w2.ID == id {
			writeJSON(w, http.StatusOK, w2)
			return
		}
	}
	writeError(w, http.StatusNotFound, "workspace not found")
}

// ─── Project handlers ─────────────────────────────────────────────────────────

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("wsID")
	list, err := s.projService.List(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	wsID := r.PathValue("wsID")
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if body.Color == "" {
		body.Color = "#6366F1"
	}
	proj, err := s.projService.Create(r.Context(), wsID, body.Name, body.Description, body.Color, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, proj)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.projService.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	p, err := s.projService.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name != "" {
		p.Name = body.Name
	}
	if body.Description != "" {
		p.Description = body.Description
	}
	if body.Color != "" {
		p.Color = body.Color
	}
	if err := s.projService.Update(r.Context(), p, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleArchiveProject(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	p, err := s.projService.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.projService.Archive(r.Context(), p.ID, claims.UserID, p.WorkspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Task handlers ────────────────────────────────────────────────────────────

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	projID := r.PathValue("projID")
	q := r.URL.Query()
	filter := models.TaskFilter{
		ProjectID:  projID,
		AssigneeID: q.Get("assignee"),
		Search:     q.Get("q"),
		Page:       models.Page{Limit: 50},
	}
	if statuses := q["status"]; len(statuses) > 0 {
		for _, s := range statuses {
			filter.Status = append(filter.Status, models.TaskStatus(s))
		}
	}
	result, err := s.taskService.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	projID := r.PathValue("projID")
	var body struct {
		Title       string             `json:"title"`
		Description string             `json:"description"`
		Status      models.TaskStatus  `json:"status"`
		Priority    models.TaskPriority `json:"priority"`
		AssigneeID  string             `json:"assignee_id"`
		WorkspaceID string             `json:"workspace_id"`
		LabelIDs    []string           `json:"label_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		writeError(w, http.StatusBadRequest, "title required")
		return
	}
	t, err := s.taskService.Create(r.Context(), service.CreateTaskInput{
		ProjectID:   projID,
		WorkspaceID: body.WorkspaceID,
		Title:       body.Title,
		Description: body.Description,
		Status:      body.Status,
		Priority:    body.Priority,
		AssigneeID:  body.AssigneeID,
		ReporterID:  claims.UserID,
		LabelIDs:    body.LabelIDs,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	t, err := s.taskService.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	t, err := s.taskService.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Title       string              `json:"title"`
		Description string              `json:"description"`
		Status      models.TaskStatus   `json:"status"`
		Priority    models.TaskPriority `json:"priority"`
		AssigneeID  string              `json:"assignee_id"`
		WorkspaceID string              `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Title != "" {
		t.Title = body.Title
	}
	t.Description = body.Description
	if body.Status != "" {
		t.Status = body.Status
	}
	if body.Priority != "" {
		t.Priority = body.Priority
	}
	if body.AssigneeID != "" {
		t.AssigneeID.Valid = true
		t.AssigneeID.String = body.AssigneeID
	}
	if err := s.taskService.Update(r.Context(), t, claims.UserID, body.WorkspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Status      models.TaskStatus `json:"status"`
		WorkspaceID string            `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		writeError(w, http.StatusBadRequest, "status required")
		return
	}
	if err := s.taskService.UpdateStatus(r.Context(), r.PathValue("id"), claims.UserID, body.WorkspaceID, body.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(body.Status)})
}

func (s *Server) handleReorderTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Position float64 `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "position required")
		return
	}
	if err := s.taskService.Reorder(r.Context(), r.PathValue("id"), body.Position); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	wsID := r.URL.Query().Get("workspace_id")
	if err := s.taskService.Delete(r.Context(), r.PathValue("id"), claims.UserID, wsID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Comment handlers ─────────────────────────────────────────────────────────

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	list, err := s.cmtService.List(r.Context(), r.PathValue("taskID"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	taskID := r.PathValue("taskID")
	var body struct {
		Body        string `json:"body"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		writeError(w, http.StatusBadRequest, "body required")
		return
	}
	c, err := s.cmtService.Add(r.Context(), taskID, claims.UserID, body.WorkspaceID, body.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if err := s.cmtService.Delete(r.Context(), r.PathValue("id"), claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Dashboard handler ────────────────────────────────────────────────────────

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := s.dashService.Stats(r.Context(), r.PathValue("wsID"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ─── Middleware ───────────────────────────────────────────────────────────────

type contextKey string

const claimsKey contextKey = "claims"

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims, err := s.authSvc.ValidateAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				writeError(w, http.StatusUnauthorized, "token expired")
			} else {
				writeError(w, http.StatusUnauthorized, "invalid token")
			}
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func claimsFrom(ctx context.Context) *models.Claims {
	c, _ := ctx.Value(claimsKey).(*models.Claims)
	return c
}

// ─── CORS middleware ──────────────────────────────────────────────────────────

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[api] encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return fmt.Sprintf("%s", r.RemoteAddr)
}
