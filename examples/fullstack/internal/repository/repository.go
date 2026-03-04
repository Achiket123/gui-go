// Package repository — thin database access layer.
// Each method maps 1:1 to a SQL query; no business logic here.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/achiket/taskflow/internal/db"
	"github.com/achiket/taskflow/internal/models"
)

// ─── UserRepo ─────────────────────────────────────────────────────────────────

type UserRepo struct{ db *db.DB }

func NewUserRepo(d *db.DB) *UserRepo { return &UserRepo{d} }

func (r *UserRepo) ByID(ctx context.Context, id string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, display_name, COALESCE(avatar_url,''), role, is_active, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.AvatarURL,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *UserRepo) ByWorkspace(ctx context.Context, workspaceID string) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.username, u.display_name, COALESCE(u.avatar_url,''), u.role, u.is_active, u.created_at, u.updated_at
		 FROM users u
		 JOIN workspace_members wm ON wm.user_id = u.id
		 WHERE wm.workspace_id = ? ORDER BY u.display_name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id, displayName, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET display_name=?, avatar_url=?, updated_at=NOW() WHERE id=?",
		displayName, avatarURL, id)
	return err
}

// ─── WorkspaceRepo ────────────────────────────────────────────────────────────

type WorkspaceRepo struct{ db *db.DB }

func NewWorkspaceRepo(d *db.DB) *WorkspaceRepo { return &WorkspaceRepo{d} }

func (r *WorkspaceRepo) Create(ctx context.Context, name, slug, description, ownerID string) (*models.Workspace, error) {
	ws := &models.Workspace{
		ID: uuid.NewString(), Name: name, Slug: slug,
		Description: description, OwnerID: ownerID,
	}
	return ws, r.db.Transact(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO workspaces (id, name, slug, description, owner_id) VALUES (?,?,?,?,?)",
			ws.ID, ws.Name, ws.Slug, ws.Description, ws.OwnerID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO workspace_members (workspace_id, user_id, role) VALUES (?,?,'admin')",
			ws.ID, ownerID)
		return err
	})
}

func (r *WorkspaceRepo) ByUser(ctx context.Context, userID string) ([]models.Workspace, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT w.id, w.name, w.slug, COALESCE(w.description,''), w.owner_id, w.created_at, w.updated_at
		 FROM workspaces w
		 JOIN workspace_members wm ON wm.workspace_id = w.id
		 WHERE wm.user_id = ? ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Workspace
	for rows.Next() {
		var ws models.Workspace
		if err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.Description,
			&ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

func (r *WorkspaceRepo) ByID(ctx context.Context, id string) (*models.Workspace, error) {
	ws := &models.Workspace{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, COALESCE(description,''), owner_id, created_at, updated_at
		 FROM workspaces WHERE id = ?`, id,
	).Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt)
	return ws, err
}

func (r *WorkspaceRepo) AddMember(ctx context.Context, workspaceID, userID string, role models.UserRole) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT IGNORE INTO workspace_members (workspace_id, user_id, role) VALUES (?,?,?)",
		workspaceID, userID, role)
	return err
}

// ─── ProjectRepo ──────────────────────────────────────────────────────────────

type ProjectRepo struct{ db *db.DB }

func NewProjectRepo(d *db.DB) *ProjectRepo { return &ProjectRepo{d} }

func (r *ProjectRepo) Create(ctx context.Context, p *models.Project) error {
	p.ID = uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, description, color, status, lead_id, created_by)
		 VALUES (?,?,?,?,?,?,?,?)`,
		p.ID, p.WorkspaceID, p.Name, p.Description, p.Color, p.Status,
		p.LeadID, p.CreatedBy)
	return err
}

func (r *ProjectRepo) ByWorkspace(ctx context.Context, workspaceID string) ([]models.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.workspace_id, p.name, COALESCE(p.description,''), p.color, p.status,
		        p.lead_id, p.created_by, p.created_at, p.updated_at,
		        COUNT(t.id) AS task_count,
		        SUM(t.status = 'done') AS done_count
		 FROM projects p
		 LEFT JOIN tasks t ON t.project_id = p.id AND t.status != 'cancelled'
		 WHERE p.workspace_id = ? AND p.status != 'deleted'
		 GROUP BY p.id ORDER BY p.name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description,
			&p.Color, &p.Status, &p.LeadID, &p.CreatedBy,
			&p.CreatedAt, &p.UpdatedAt, &p.TaskCount, &p.DoneCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) ByID(ctx context.Context, id string) (*models.Project, error) {
	p := &models.Project{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, name, COALESCE(description,''), color, status,
		        lead_id, created_by, created_at, updated_at
		 FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.Color, &p.Status,
		&p.LeadID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *ProjectRepo) Update(ctx context.Context, p *models.Project) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name=?, description=?, color=?, status=?, lead_id=?, updated_at=NOW() WHERE id=?`,
		p.Name, p.Description, p.Color, p.Status, p.LeadID, p.ID)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE projects SET status='deleted', updated_at=NOW() WHERE id=?", id)
	return err
}

// ─── TaskRepo ─────────────────────────────────────────────────────────────────

type TaskRepo struct{ db *db.DB }

func NewTaskRepo(d *db.DB) *TaskRepo { return &TaskRepo{d} }

const taskSelectCols = `
	t.id, t.project_id, t.parent_id, t.title, COALESCE(t.description,''),
	t.status, t.priority, t.assignee_id, t.reporter_id, t.position,
	t.estimate_h, t.due_date, t.completed_at, t.created_at, t.updated_at`

func scanTask(row interface {
	Scan(...any) error
}) (*models.Task, error) {
	t := &models.Task{}
	return t, row.Scan(
		&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.AssigneeID, &t.ReporterID, &t.Position,
		&t.EstimateH, &t.DueDate, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
}

func (r *TaskRepo) Create(ctx context.Context, t *models.Task) error {
	t.ID = uuid.NewString()
	// Set position to the end of the project's current tasks.
	var maxPos sql.NullFloat64
	_ = r.db.QueryRowContext(ctx,
		"SELECT MAX(position) FROM tasks WHERE project_id = ?", t.ProjectID).Scan(&maxPos)
	if maxPos.Valid {
		t.Position = maxPos.Float64 + 1000
	} else {
		t.Position = 1000
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (id, project_id, parent_id, title, description, status, priority,
		                    assignee_id, reporter_id, position, estimate_h, due_date)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.ProjectID, t.ParentID, t.Title, t.Description,
		t.Status, t.Priority, t.AssigneeID, t.ReporterID,
		t.Position, t.EstimateH, t.DueDate)
	return err
}

func (r *TaskRepo) ByProject(ctx context.Context, filter models.TaskFilter) (models.PageResult[models.Task], error) {
	var result models.PageResult[models.Task]

	where := []string{"t.project_id = ?"}
	args := []any{filter.ProjectID}

	if filter.AssigneeID != "" {
		where = append(where, "t.assignee_id = ?")
		args = append(args, filter.AssigneeID)
	}
	if len(filter.Status) > 0 {
		ph := strings.Repeat("?,", len(filter.Status))
		ph = ph[:len(ph)-1]
		where = append(where, "t.status IN ("+ph+")")
		for _, s := range filter.Status {
			args = append(args, s)
		}
	}
	if len(filter.Priority) > 0 {
		ph := strings.Repeat("?,", len(filter.Priority))
		ph = ph[:len(ph)-1]
		where = append(where, "t.priority IN ("+ph+")")
		for _, p := range filter.Priority {
			args = append(args, p)
		}
	}
	if filter.Search != "" {
		where = append(where, "MATCH(t.title, t.description) AGAINST (? IN BOOLEAN MODE)")
		args = append(args, filter.Search+"*")
	}

	whereStr := strings.Join(where, " AND ")

	// Count.
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	_ = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks t WHERE "+whereStr, countArgs...).Scan(&result.Total)

	// Fetch.
	limit := filter.Page.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, filter.Page.Offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s FROM tasks t WHERE %s ORDER BY t.position, t.created_at LIMIT ? OFFSET ?`,
			taskSelectCols, whereStr), args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, *t)
	}
	result.Page = filter.Page
	return result, rows.Err()
}

func (r *TaskRepo) ByID(ctx context.Context, id string) (*models.Task, error) {
	return scanTask(r.db.QueryRowContext(ctx,
		"SELECT "+taskSelectCols+" FROM tasks t WHERE t.id = ?", id))
}

func (r *TaskRepo) Update(ctx context.Context, t *models.Task) error {
	var completedAt *time.Time
	if t.Status == models.StatusDone && !t.CompletedAt.Valid {
		now := time.Now()
		completedAt = &now
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET title=?, description=?, status=?, priority=?,
		 assignee_id=?, position=?, estimate_h=?, due_date=?, completed_at=?, updated_at=NOW()
		 WHERE id=?`,
		t.Title, t.Description, t.Status, t.Priority,
		t.AssigneeID, t.Position, t.EstimateH, t.DueDate, completedAt, t.ID)
	return err
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error {
	var completedAt *time.Time
	if status == models.StatusDone {
		now := time.Now()
		completedAt = &now
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE tasks SET status=?, completed_at=?, updated_at=NOW() WHERE id=?",
		status, completedAt, id)
	return err
}

func (r *TaskRepo) Reorder(ctx context.Context, id string, newPosition float64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE tasks SET position=?, updated_at=NOW() WHERE id=?", newPosition, id)
	return err
}

func (r *TaskRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id=?", id)
	return err
}

// ─── CommentRepo ──────────────────────────────────────────────────────────────

type CommentRepo struct{ db *db.DB }

func NewCommentRepo(d *db.DB) *CommentRepo { return &CommentRepo{d} }

func (r *CommentRepo) ByTask(ctx context.Context, taskID string) ([]models.Comment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.task_id, c.author_id, c.body, c.edited_at, c.created_at,
		        u.id, u.display_name, COALESCE(u.avatar_url,'')
		 FROM comments c
		 JOIN users u ON u.id = c.author_id
		 WHERE c.task_id = ? ORDER BY c.created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Comment
	for rows.Next() {
		var c models.Comment
		c.Author = &models.User{}
		if err := rows.Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.Body, &c.EditedAt, &c.CreatedAt,
			&c.Author.ID, &c.Author.DisplayName, &c.Author.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CommentRepo) Create(ctx context.Context, taskID, authorID, body string) (*models.Comment, error) {
	c := &models.Comment{
		ID: uuid.NewString(), TaskID: taskID, AuthorID: authorID, Body: body,
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO comments (id, task_id, author_id, body) VALUES (?,?,?,?)",
		c.ID, c.TaskID, c.AuthorID, c.Body)
	return c, err
}

func (r *CommentRepo) Delete(ctx context.Context, id, authorID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM comments WHERE id=? AND author_id=?", id, authorID)
	return err
}

// ─── ActivityRepo ─────────────────────────────────────────────────────────────

type ActivityRepo struct{ db *db.DB }

func NewActivityRepo(d *db.DB) *ActivityRepo { return &ActivityRepo{d} }

func (r *ActivityRepo) Log(ctx context.Context, workspaceID, actorID, entityType, entityID, action string, meta []byte) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO activity_log (workspace_id, actor_id, entity_type, entity_id, action, meta)
		 VALUES (?,?,?,?,?,?)`,
		workspaceID, actorID, entityType, entityID, action, meta)
	return err
}

func (r *ActivityRepo) Recent(ctx context.Context, workspaceID string, limit int) ([]models.Activity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.workspace_id, a.actor_id, a.entity_type, a.entity_id, a.action, a.meta, a.created_at,
		        u.id, u.display_name, COALESCE(u.avatar_url,'')
		 FROM activity_log a
		 LEFT JOIN users u ON u.id = a.actor_id
		 WHERE a.workspace_id = ? ORDER BY a.created_at DESC LIMIT ?`,
		workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Activity
	for rows.Next() {
		var act models.Activity
		var uID, uName, uAvatar sql.NullString
		if err := rows.Scan(&act.ID, &act.WorkspaceID, &act.ActorID,
			&act.EntityType, &act.EntityID, &act.Action, &act.Meta, &act.CreatedAt,
			&uID, &uName, &uAvatar); err != nil {
			return nil, err
		}
		if uID.Valid {
			act.Actor = &models.User{
				ID: uID.String, DisplayName: uName.String, AvatarURL: uAvatar.String,
			}
		}
		out = append(out, act)
	}
	return out, rows.Err()
}

// ─── Label helpers ────────────────────────────────────────────────────────────

type LabelRepo struct{ db *db.DB }

func NewLabelRepo(d *db.DB) *LabelRepo { return &LabelRepo{d} }

func (r *LabelRepo) ByWorkspace(ctx context.Context, workspaceID string) ([]models.Label, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, workspace_id, name, color, created_at FROM labels WHERE workspace_id=? ORDER BY name",
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Label
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.WorkspaceID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LabelRepo) Create(ctx context.Context, workspaceID, name, color string) (*models.Label, error) {
	l := &models.Label{ID: uuid.NewString(), WorkspaceID: workspaceID, Name: name, Color: color}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO labels (id, workspace_id, name, color) VALUES (?,?,?,?)",
		l.ID, l.WorkspaceID, l.Name, l.Color)
	return l, err
}

func (r *LabelRepo) SetTaskLabels(ctx context.Context, taskID string, labelIDs []string) error {
	return r.db.Transact(context.Background(), func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM task_labels WHERE task_id=?", taskID); err != nil {
			return err
		}
		for _, lid := range labelIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT IGNORE INTO task_labels (task_id, label_id) VALUES (?,?)", taskID, lid); err != nil {
				return err
			}
		}
		return nil
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func scanUsers(rows *sql.Rows) ([]models.User, error) {
	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName,
			&u.AvatarURL, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
