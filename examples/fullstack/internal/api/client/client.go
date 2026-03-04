// Package client — HTTP client that wraps every API endpoint.
// Used exclusively by the GUI application.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/achiket/taskflow/internal/models"
)

// Client talks to the TaskFlow REST API.
type Client struct {
	base        string
	http        *http.Client
	mu          sync.RWMutex
	accessToken  string
	refreshToken string

	// OnTokenRefreshed is called after a successful token refresh.
	OnTokenRefreshed func()
}

// New creates a Client targeting baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// SetTokens stores the current access + refresh tokens.
func (c *Client) SetTokens(access, refresh string) {
	c.mu.Lock()
	c.accessToken = access
	c.refreshToken = refresh
	c.mu.Unlock()
}

// IsAuthenticated returns true when an access token is stored.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	ok := c.accessToken != ""
	c.mu.RUnlock()
	return ok
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

type LoginResponse struct {
	Tokens *models.TokenPair `json:"tokens"`
	User   *models.User      `json:"user"`
}

func (c *Client) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	var resp LoginResponse
	if err := c.post(ctx, "/api/auth/login", map[string]string{
		"email": email, "password": password,
	}, &resp, false); err != nil {
		return nil, err
	}
	c.SetTokens(resp.Tokens.AccessToken, resp.Tokens.RefreshToken)
	return &resp, nil
}

func (c *Client) Register(ctx context.Context, email, username, displayName, password string) (*LoginResponse, error) {
	var resp LoginResponse
	if err := c.post(ctx, "/api/auth/register", map[string]string{
		"email": email, "username": username,
		"display_name": displayName, "password": password,
	}, &resp, false); err != nil {
		return nil, err
	}
	c.SetTokens(resp.Tokens.AccessToken, resp.Tokens.RefreshToken)
	return &resp, nil
}

func (c *Client) Logout(ctx context.Context) error {
	c.mu.RLock()
	rt := c.refreshToken
	c.mu.RUnlock()
	_ = c.post(ctx, "/api/auth/logout", map[string]string{"refresh_token": rt}, nil, true)
	c.SetTokens("", "")
	return nil
}

func (c *Client) RefreshTokens(ctx context.Context) error {
	c.mu.RLock()
	rt := c.refreshToken
	c.mu.RUnlock()
	var pair models.TokenPair
	if err := c.post(ctx, "/api/auth/refresh", map[string]string{"refresh_token": rt}, &pair, false); err != nil {
		return err
	}
	c.SetTokens(pair.AccessToken, pair.RefreshToken)
	if c.OnTokenRefreshed != nil {
		c.OnTokenRefreshed()
	}
	return nil
}

// ─── Workspaces ───────────────────────────────────────────────────────────────

func (c *Client) ListWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	var out []models.Workspace
	return out, c.get(ctx, "/api/workspaces", nil, &out)
}

func (c *Client) CreateWorkspace(ctx context.Context, name, description string) (*models.Workspace, error) {
	var out models.Workspace
	return &out, c.post(ctx, "/api/workspaces", map[string]string{
		"name": name, "description": description,
	}, &out, true)
}

// ─── Projects ─────────────────────────────────────────────────────────────────

func (c *Client) ListProjects(ctx context.Context, workspaceID string) ([]models.Project, error) {
	var out []models.Project
	return out, c.get(ctx, "/api/workspaces/"+workspaceID+"/projects", nil, &out)
}

func (c *Client) CreateProject(ctx context.Context, workspaceID, name, description, color string) (*models.Project, error) {
	var out models.Project
	return &out, c.post(ctx, "/api/workspaces/"+workspaceID+"/projects", map[string]string{
		"name": name, "description": description, "color": color,
	}, &out, true)
}

func (c *Client) GetProject(ctx context.Context, id string) (*models.Project, error) {
	var out models.Project
	return &out, c.get(ctx, "/api/projects/"+id, nil, &out)
}

func (c *Client) UpdateProject(ctx context.Context, id, name, description, color string) (*models.Project, error) {
	var out models.Project
	return &out, c.put(ctx, "/api/projects/"+id, map[string]string{
		"name": name, "description": description, "color": color,
	}, &out)
}

func (c *Client) ArchiveProject(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/projects/"+id)
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func (c *Client) ListTasks(ctx context.Context, projectID string, params url.Values) (*models.PageResult[models.Task], error) {
	var out models.PageResult[models.Task]
	path := "/api/projects/" + projectID + "/tasks"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return &out, c.get(ctx, path, nil, &out)
}

func (c *Client) CreateTask(ctx context.Context, projectID, workspaceID, title, description string,
	status models.TaskStatus, priority models.TaskPriority, assigneeID string) (*models.Task, error) {
	var out models.Task
	return &out, c.post(ctx, "/api/projects/"+projectID+"/tasks", map[string]string{
		"title": title, "description": description,
		"status":       string(status),
		"priority":     string(priority),
		"assignee_id":  assigneeID,
		"workspace_id": workspaceID,
	}, &out, true)
}

func (c *Client) GetTask(ctx context.Context, id string) (*models.Task, error) {
	var out models.Task
	return &out, c.get(ctx, "/api/tasks/"+id, nil, &out)
}

func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, workspaceID string, status models.TaskStatus) error {
	return c.patch(ctx, "/api/tasks/"+taskID+"/status", map[string]string{
		"status": string(status), "workspace_id": workspaceID,
	}, nil)
}

func (c *Client) DeleteTask(ctx context.Context, taskID, workspaceID string) error {
	return c.delete(ctx, "/api/tasks/"+taskID+"?workspace_id="+workspaceID)
}

// ─── Comments ─────────────────────────────────────────────────────────────────

func (c *Client) ListComments(ctx context.Context, taskID string) ([]models.Comment, error) {
	var out []models.Comment
	return out, c.get(ctx, "/api/tasks/"+taskID+"/comments", nil, &out)
}

func (c *Client) AddComment(ctx context.Context, taskID, workspaceID, body string) (*models.Comment, error) {
	var out models.Comment
	return &out, c.post(ctx, "/api/tasks/"+taskID+"/comments", map[string]string{
		"body": body, "workspace_id": workspaceID,
	}, &out, true)
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

func (c *Client) Dashboard(ctx context.Context, workspaceID string) (map[string]any, error) {
	var out map[string]any
	return out, c.get(ctx, "/api/workspaces/"+workspaceID+"/dashboard", nil, &out)
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	if params != nil {
		path += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out, true)
}

func (c *Client) post(ctx context.Context, path string, body any, out any, auth bool) error {
	return c.bodyRequest(ctx, http.MethodPost, path, body, out, auth)
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	return c.bodyRequest(ctx, http.MethodPut, path, body, out, true)
}

func (c *Client) patch(ctx context.Context, path string, body any, out any) error {
	return c.bodyRequest(ctx, http.MethodPatch, path, body, out, true)
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil, true)
}

func (c *Client) bodyRequest(ctx context.Context, method, path string, body any, out any, withAuth bool) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out, withAuth)
}

func (c *Client) do(req *http.Request, out any, withAuth bool) error {
	if withAuth {
		c.mu.RLock()
		token := c.accessToken
		c.mu.RUnlock()
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	// Auto-refresh on 401.
	if resp.StatusCode == http.StatusUnauthorized && withAuth {
		if err := c.RefreshTokens(context.Background()); err == nil {
			// Retry once with new token.
			c.mu.RLock()
			req.Header.Set("Authorization", "Bearer "+c.accessToken)
			c.mu.RUnlock()
			resp.Body.Close()
			resp, err = c.http.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
		}
	}

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}
