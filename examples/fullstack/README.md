# TaskFlow — Full-Stack Go Project Management App

A complete, production-grade project management SaaS application built entirely in Go.
Backend REST API + MySQL + JWT auth + native desktop GUI using `gui-go`.

---

## Architecture

```
taskflow/
├── cmd/
│   ├── server/main.go          ← REST API server (net/http, no framework)
│   └── app/main.go             ← Native desktop GUI app
│
├── internal/
│   ├── auth/auth.go            ← JWT access/refresh tokens, bcrypt passwords
│   ├── db/db.go                ← MySQL connection pool (database/sql)
│   ├── models/models.go        ← Domain types (User, Workspace, Project, Task…)
│   ├── repository/repository.go← Database access layer (raw SQL)
│   ├── service/service.go      ← Business logic (Workspace/Project/Task/Dashboard)
│   └── api/
│       ├── api.go              ← HTTP handlers + JWT middleware + CORS
│       └── client.go           ← HTTP client used by the GUI frontend
│
├── ui/
│   ├── state/state.go          ← Reactive app state (Signals, EventBus)
│   ├── styles/styles.go        ← Design tokens, helpers, dark theme
│   └── screens/
│       ├── auth.go             ← Login + Register screens
│       ← workspace.go          ← Workspace picker, Dashboard, Sidebar, MainScreen
│       └── board.go            ← Kanban board, Task drawer, Create Project panel
│
└── schema.sql                  ← MySQL DDL (users, workspaces, projects, tasks…)
```

---

## Features

### Backend
- **Authentication** — JWT access tokens (15 min) + refresh tokens (7 days), bcrypt password hashing
- **Workspaces** — multi-tenant; users belong to multiple workspaces with roles (admin/member/viewer)
- **Projects** — per-workspace; color-coded, lead assignment, progress tracking
- **Tasks** — full Kanban workflow (Backlog → Todo → In Progress → In Review → Done → Cancelled), priority levels, assignees, due dates, estimated hours, sub-tasks, drag-and-drop reordering via `position` float
- **Comments** — per-task threaded comments
- **Labels** — workspace-scoped labels attached to tasks
- **Activity Log** — every state change is recorded with actor + JSON metadata
- **Full-text search** — MySQL FULLTEXT index on task title + description

### Frontend (native GUI via `gui-go`)
- **Reactive state** — `Signal[T]`, `Stream[T]`, `Effect`, `EventBus` for zero-boilerplate UI updates
- **Screen navigation** — `Navigator` stack (push/pop/replace)
- **Login / Register** — form validation, error display, JWT storage
- **Workspace picker** — list, create workspace
- **Sidebar** — project list with task counts, active-project highlight, logout
- **Dashboard** — stat cards (total/done/in-progress/members/projects), recent activity feed
- **Kanban board** — one scrollable column per status, task cards with priority dots, assignee avatars
- **Task drawer** — slide-in create/edit panel: title, description, status dropdown, priority dropdown, comments
- **Create project panel** — name, description, accent-color picker with live preview swatch
- **Global toast** — error/success notification bar

---

## Quick Start

### 1. Database

```bash
mysql -u root -p -e "CREATE DATABASE taskflow CHARACTER SET utf8mb4;"
mysql -u root -p taskflow < schema.sql
```

Default seed account: `admin@taskflow.io` / `Admin@1234`

### 2. Backend server

```bash
go run ./cmd/server \
  -addr :8080 \
  -db-host 127.0.0.1 \
  -db-port 3306 \
  -db-name taskflow \
  -db-user root \
  -db-pass "" \
  -jwt-secret "change-me-in-production"
```

The server logs `[server] listening on :8080` when ready.

### 3. Desktop app

```bash
go run ./cmd/app -api http://localhost:8080
```

---

## API Reference (REST)

All authenticated routes require `Authorization: Bearer <access_token>`.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Sign up → returns `{tokens, user}` |
| POST | `/api/auth/login` | Sign in → returns `{tokens, user}` |
| POST | `/api/auth/refresh` | Rotate tokens |
| POST | `/api/auth/logout` | Revoke refresh token |
| GET | `/api/me` | Current user claims |
| PUT | `/api/me/password` | Change password |
| GET | `/api/workspaces` | List user's workspaces |
| POST | `/api/workspaces` | Create workspace |
| GET | `/api/workspaces/{id}/projects` | List projects |
| POST | `/api/workspaces/{wsID}/projects` | Create project |
| GET | `/api/projects/{id}` | Get project |
| PUT | `/api/projects/{id}` | Update project |
| DELETE | `/api/projects/{id}` | Archive project |
| GET | `/api/projects/{projID}/tasks` | List tasks (filter by status/priority/assignee/search) |
| POST | `/api/projects/{projID}/tasks` | Create task |
| GET | `/api/tasks/{id}` | Get task (with comments + assignee) |
| PUT | `/api/tasks/{id}` | Update task |
| PATCH | `/api/tasks/{id}/status` | Change task status |
| PATCH | `/api/tasks/{id}/reorder` | Update sort position |
| DELETE | `/api/tasks/{id}` | Delete task |
| GET | `/api/tasks/{taskID}/comments` | List comments |
| POST | `/api/tasks/{taskID}/comments` | Add comment |
| DELETE | `/api/comments/{id}` | Delete own comment |
| GET | `/api/workspaces/{wsID}/dashboard` | Dashboard stats |
| GET | `/api/health` | Health check |

---

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | Server listen address |
| `-db-host` | `127.0.0.1` | MySQL host |
| `-db-port` | `3306` | MySQL port |
| `-db-name` | `taskflow` | MySQL database |
| `-db-user` | `root` | MySQL user |
| `-db-pass` | `` | MySQL password |
| `-jwt-secret` | `super-secret-key` | **Change in production!** |
| `-api` (app) | `http://localhost:8080` | API base URL |

---

## Security Notes

- JWT access tokens expire in **15 minutes**; the GUI client auto-refreshes using the 7-day refresh token
- Refresh tokens are **rotated** on every use (old token revoked immediately)
- Passwords are hashed with **bcrypt cost 12**
- All refresh tokens are stored as **SHA-256 hashes** in the DB — raw tokens never persist
- SQL queries use **parameterised statements** only — no string interpolation
- CORS is open (`*`) for development; restrict in production

---

## Dependencies

```
github.com/go-sql-driver/mysql v1.8.1    — MySQL driver
github.com/golang-jwt/jwt/v5   v5.2.1    — JWT tokens
github.com/google/uuid         v1.6.0    — UUID generation
golang.org/x/crypto            v0.22.0   — bcrypt
github.com/achiket/gui-go      (local)   — native GUI library
```
