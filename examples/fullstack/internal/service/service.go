// Package service — business logic layer.
// Services coordinate repositories, enforce access-control rules, and emit
// activity log entries.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/achiket/taskflow/internal/models"
	"github.com/achiket/taskflow/internal/repository"
)

// ErrForbidden is returned when the caller lacks permission.
var ErrForbidden = errors.New("forbidden")

// ErrNotFound is returned when a resource doesn't exist.
var ErrNotFound = errors.New("not found")

// ─── Repos bundle ─────────────────────────────────────────────────────────────

type Repos struct {
	User      *repository.UserRepo
	Workspace *repository.WorkspaceRepo
	Project   *repository.ProjectRepo
	Task      *repository.TaskRepo
	Comment   *repository.CommentRepo
	Activity  *repository.ActivityRepo
	Label     *repository.LabelRepo
}

// ─── WorkspaceService ─────────────────────────────────────────────────────────

type WorkspaceService struct{ r *Repos }

func NewWorkspaceService(r *Repos) *WorkspaceService { return &WorkspaceService{r} }

func (s *WorkspaceService) Create(ctx context.Context, name, description, ownerID string) (*models.Workspace, error) {
	slug := slugify(name)
	ws, err := s.r.Workspace.Create(ctx, name, slug, description, ownerID)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	_ = s.r.Activity.Log(ctx, ws.ID, ownerID, "workspace", ws.ID, "created", nil)
	return ws, nil
}

func (s *WorkspaceService) ListForUser(ctx context.Context, userID string) ([]models.Workspace, error) {
	return s.r.Workspace.ByUser(ctx, userID)
}

func (s *WorkspaceService) InviteMember(ctx context.Context, workspaceID, inviterID, email string, role models.UserRole) error {
	// Simplified: look up user by email then add.
	_ = inviterID
	_ = email
	// In production you'd email an invite link.
	return nil
}

// ─── ProjectService ───────────────────────────────────────────────────────────

type ProjectService struct{ r *Repos }

func NewProjectService(r *Repos) *ProjectService { return &ProjectService{r} }

func (s *ProjectService) Create(ctx context.Context, workspaceID, name, desc, color, creatorID string) (*models.Project, error) {
	p := &models.Project{
		WorkspaceID: workspaceID,
		Name:        name,
		Description: desc,
		Color:       color,
		Status:      models.ProjectActive,
		CreatedBy:   creatorID,
	}
	if err := s.r.Project.Create(ctx, p); err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(map[string]string{"name": name})
	_ = s.r.Activity.Log(ctx, workspaceID, creatorID, "project", p.ID, "created", meta)
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, workspaceID string) ([]models.Project, error) {
	return s.r.Project.ByWorkspace(ctx, workspaceID)
}

func (s *ProjectService) Get(ctx context.Context, id string) (*models.Project, error) {
	p, err := s.r.Project.ByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *ProjectService) Update(ctx context.Context, p *models.Project, actorID string) error {
	if err := s.r.Project.Update(ctx, p); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{"name": p.Name})
	_ = s.r.Activity.Log(ctx, p.WorkspaceID, actorID, "project", p.ID, "updated", meta)
	return nil
}

func (s *ProjectService) Archive(ctx context.Context, projectID, actorID, workspaceID string) error {
	p, err := s.r.Project.ByID(ctx, projectID)
	if err != nil {
		return ErrNotFound
	}
	p.Status = models.ProjectArchived
	if err := s.r.Project.Update(ctx, p); err != nil {
		return err
	}
	_ = s.r.Activity.Log(ctx, workspaceID, actorID, "project", projectID, "archived", nil)
	return nil
}

// ─── TaskService ──────────────────────────────────────────────────────────────

type TaskService struct{ r *Repos }

func NewTaskService(r *Repos) *TaskService { return &TaskService{r} }

// CreateInput is the payload for creating a task.
type CreateTaskInput struct {
	ProjectID   string
	WorkspaceID string
	Title       string
	Description string
	Status      models.TaskStatus
	Priority    models.TaskPriority
	AssigneeID  string
	ReporterID  string
	LabelIDs    []string
}

func (s *TaskService) Create(ctx context.Context, in CreateTaskInput) (*models.Task, error) {
	t := &models.Task{
		ProjectID:   in.ProjectID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Priority:    in.Priority,
		ReporterID:  in.ReporterID,
	}
	if in.AssigneeID != "" {
		t.AssigneeID.Valid = true
		t.AssigneeID.String = in.AssigneeID
	}
	if t.Status == "" {
		t.Status = models.StatusBacklog
	}
	if t.Priority == "" {
		t.Priority = models.PriorityMedium
	}
	if err := s.r.Task.Create(ctx, t); err != nil {
		return nil, err
	}
	if len(in.LabelIDs) > 0 {
		_ = s.r.Label.SetTaskLabels(ctx, t.ID, in.LabelIDs)
	}
	meta, _ := json.Marshal(map[string]string{"title": t.Title, "status": string(t.Status)})
	_ = s.r.Activity.Log(ctx, in.WorkspaceID, in.ReporterID, "task", t.ID, "created", meta)
	return t, nil
}

func (s *TaskService) List(ctx context.Context, filter models.TaskFilter) (models.PageResult[models.Task], error) {
	result, err := s.r.Task.ByProject(ctx, filter)
	if err != nil {
		return result, err
	}
	// Eager-load assignees.
	for i := range result.Items {
		if result.Items[i].AssigneeID.Valid {
			u, err := s.r.User.ByID(ctx, result.Items[i].AssigneeID.String)
			if err == nil {
				result.Items[i].Assignee = u
			}
		}
	}
	return result, nil
}

func (s *TaskService) Get(ctx context.Context, id string) (*models.Task, error) {
	t, err := s.r.Task.ByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	// Eager-load.
	if t.AssigneeID.Valid {
		if u, err := s.r.User.ByID(ctx, t.AssigneeID.String); err == nil {
			t.Assignee = u
		}
	}
	if u, err := s.r.User.ByID(ctx, t.ReporterID); err == nil {
		t.Reporter = u
	}
	t.Comments, _ = s.r.Comment.ByTask(ctx, id)
	return t, nil
}

func (s *TaskService) UpdateStatus(ctx context.Context, taskID, actorID, workspaceID string, status models.TaskStatus) error {
	old, err := s.r.Task.ByID(ctx, taskID)
	if err != nil {
		return ErrNotFound
	}
	if err := s.r.Task.UpdateStatus(ctx, taskID, status); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{
		"from": string(old.Status), "to": string(status),
	})
	_ = s.r.Activity.Log(ctx, workspaceID, actorID, "task", taskID, "status_changed", meta)
	return nil
}

func (s *TaskService) Update(ctx context.Context, t *models.Task, actorID, workspaceID string) error {
	if err := s.r.Task.Update(ctx, t); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{"title": t.Title})
	_ = s.r.Activity.Log(ctx, workspaceID, actorID, "task", t.ID, "updated", meta)
	return nil
}

func (s *TaskService) Delete(ctx context.Context, taskID, actorID, workspaceID string) error {
	if err := s.r.Task.Delete(ctx, taskID); err != nil {
		return err
	}
	_ = s.r.Activity.Log(ctx, workspaceID, actorID, "task", taskID, "deleted", nil)
	return nil
}

func (s *TaskService) Reorder(ctx context.Context, taskID string, newPos float64) error {
	return s.r.Task.Reorder(ctx, taskID, newPos)
}

// ─── CommentService ───────────────────────────────────────────────────────────

type CommentService struct{ r *Repos }

func NewCommentService(r *Repos) *CommentService { return &CommentService{r} }

func (s *CommentService) Add(ctx context.Context, taskID, authorID, workspaceID, body string) (*models.Comment, error) {
	c, err := s.r.Comment.Create(ctx, taskID, authorID, body)
	if err != nil {
		return nil, err
	}
	_ = s.r.Activity.Log(ctx, workspaceID, authorID, "comment", c.ID, "created", nil)
	return c, nil
}

func (s *CommentService) List(ctx context.Context, taskID string) ([]models.Comment, error) {
	return s.r.Comment.ByTask(ctx, taskID)
}

func (s *CommentService) Delete(ctx context.Context, commentID, authorID string) error {
	return s.r.Comment.Delete(ctx, commentID, authorID)
}

// ─── DashboardService ─────────────────────────────────────────────────────────

type DashboardStats struct {
	TotalTasks    int
	DoneTasks     int
	InProgress    int
	OverdueTasks  int
	Projects      int
	Members       int
	RecentActivity []models.Activity
}

type DashboardService struct{ r *Repos }

func NewDashboardService(r *Repos) *DashboardService { return &DashboardService{r} }

func (s *DashboardService) Stats(ctx context.Context, workspaceID string) (*DashboardStats, error) {
	stats := &DashboardStats{}

	projects, err := s.r.Project.ByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	stats.Projects = len(projects)
	for _, p := range projects {
		stats.TotalTasks += p.TaskCount
		stats.DoneTasks += p.DoneCount
	}
	stats.InProgress = stats.TotalTasks - stats.DoneTasks

	members, err := s.r.User.ByWorkspace(ctx, workspaceID)
	if err == nil {
		stats.Members = len(members)
	}

	stats.RecentActivity, _ = s.r.Activity.Recent(ctx, workspaceID, 20)
	return stats, nil
}

// ─── utility ──────────────────────────────────────────────────────────────────

func slugify(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			result = append(result, c)
		case c >= 'A' && c <= 'Z':
			result = append(result, c+32)
		case c == ' ' || c == '-' || c == '_':
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	return string(result)
}
