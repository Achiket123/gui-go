// Package models — domain types that map to database rows.
package models

import (
	"database/sql"
	"time"
)

// ─── User ─────────────────────────────────────────────────────────────────────

type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

type User struct {
	ID           string       `json:"id"`
	Email        string       `json:"email"`
	Username     string       `json:"username"`
	DisplayName  string       `json:"display_name"`
	PasswordHash string       `json:"-"`
	AvatarURL    string       `json:"avatar_url,omitempty"`
	Role         UserRole     `json:"role"`
	IsActive     bool         `json:"is_active"`
	LastLogin    sql.NullTime `json:"last_login,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ─── Workspace ────────────────────────────────────────────────────────────────

type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Eager-loaded.
	Owner   *User  `json:"owner,omitempty"`
	Members []User `json:"members,omitempty"`
}

// ─── Project ──────────────────────────────────────────────────────────────────

type ProjectStatus string

const (
	ProjectActive   ProjectStatus = "active"
	ProjectArchived ProjectStatus = "archived"
	ProjectDeleted  ProjectStatus = "deleted"
)

type Project struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Color       string         `json:"color"`
	Status      ProjectStatus  `json:"status"`
	LeadID      sql.NullString `json:"lead_id,omitempty"`
	StartDate   sql.NullTime   `json:"start_date,omitempty"`
	TargetDate  sql.NullTime   `json:"target_date,omitempty"`
	CreatedBy   string         `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Computed.
	TaskCount  int `json:"task_count,omitempty"`
	DoneCount  int `json:"done_count,omitempty"`
	Lead       *User `json:"lead,omitempty"`
}

// ─── Task ─────────────────────────────────────────────────────────────────────

type TaskStatus string

const (
	StatusBacklog     TaskStatus = "backlog"
	StatusTodo        TaskStatus = "todo"
	StatusInProgress  TaskStatus = "in_progress"
	StatusInReview    TaskStatus = "in_review"
	StatusDone        TaskStatus = "done"
	StatusCancelled   TaskStatus = "cancelled"
)

var TaskStatusOrder = map[TaskStatus]int{
	StatusBacklog:    0,
	StatusTodo:       1,
	StatusInProgress: 2,
	StatusInReview:   3,
	StatusDone:       4,
	StatusCancelled:  5,
}

var TaskStatusLabel = map[TaskStatus]string{
	StatusBacklog:    "Backlog",
	StatusTodo:       "To Do",
	StatusInProgress: "In Progress",
	StatusInReview:   "In Review",
	StatusDone:       "Done",
	StatusCancelled:  "Cancelled",
}

type TaskPriority string

const (
	PriorityUrgent TaskPriority = "urgent"
	PriorityHigh   TaskPriority = "high"
	PriorityMedium TaskPriority = "medium"
	PriorityLow    TaskPriority = "low"
	PriorityNone   TaskPriority = "none"
)

var TaskPriorityLabel = map[TaskPriority]string{
	PriorityUrgent: "Urgent",
	PriorityHigh:   "High",
	PriorityMedium: "Medium",
	PriorityLow:    "Low",
	PriorityNone:   "None",
}

type Task struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	ParentID    sql.NullString `json:"parent_id,omitempty"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      TaskStatus     `json:"status"`
	Priority    TaskPriority   `json:"priority"`
	AssigneeID  sql.NullString `json:"assignee_id,omitempty"`
	ReporterID  string         `json:"reporter_id"`
	Position    float64        `json:"position"`
	EstimateH   sql.NullFloat64 `json:"estimate_h,omitempty"`
	DueDate     sql.NullTime   `json:"due_date,omitempty"`
	CompletedAt sql.NullTime   `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Eager-loaded.
	Assignee *User    `json:"assignee,omitempty"`
	Reporter *User    `json:"reporter,omitempty"`
	Labels   []Label  `json:"labels,omitempty"`
	SubTasks []Task   `json:"sub_tasks,omitempty"`
	Comments []Comment `json:"comments,omitempty"`
}

// ─── Label ────────────────────────────────────────────────────────────────────

type Label struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Comment ──────────────────────────────────────────────────────────────────

type Comment struct {
	ID        string       `json:"id"`
	TaskID    string       `json:"task_id"`
	AuthorID  string       `json:"author_id"`
	Body      string       `json:"body"`
	EditedAt  sql.NullTime `json:"edited_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`

	Author *User `json:"author,omitempty"`
}

// ─── Activity ─────────────────────────────────────────────────────────────────

type Activity struct {
	ID          int64          `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	ActorID     sql.NullString `json:"actor_id"`
	EntityType  string         `json:"entity_type"`
	EntityID    string         `json:"entity_id"`
	Action      string         `json:"action"`
	Meta        []byte         `json:"meta,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`

	Actor *User `json:"actor,omitempty"`
}

// ─── Pagination ───────────────────────────────────────────────────────────────

type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type PageResult[T any] struct {
	Items []T  `json:"items"`
	Total int  `json:"total"`
	Page  Page `json:"page"`
}

// ─── Filter ───────────────────────────────────────────────────────────────────

type TaskFilter struct {
	ProjectID  string
	AssigneeID string
	Status     []TaskStatus
	Priority   []TaskPriority
	Search     string
	Page       Page
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

type Claims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	Role        UserRole `json:"role"`
	WorkspaceID string   `json:"wid,omitempty"`
}
