-- ============================================================
-- TaskFlow — MySQL schema
-- Run once against a fresh database:
--   mysql -u root -p taskflow < schema.sql
-- ============================================================

CREATE DATABASE IF NOT EXISTS taskflow CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE taskflow;

-- ── users ─────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    id           CHAR(36)     NOT NULL PRIMARY KEY,
    email        VARCHAR(255) NOT NULL UNIQUE,
    username     VARCHAR(60)  NOT NULL UNIQUE,
    display_name VARCHAR(120) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url   VARCHAR(512),
    role         ENUM('admin','member','viewer') NOT NULL DEFAULT 'member',
    is_active    TINYINT(1)   NOT NULL DEFAULT 1,
    last_login   DATETIME,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_email (email),
    INDEX idx_users_username (username)
) ENGINE=InnoDB;

-- ── workspaces ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS workspaces (
    id          CHAR(36)     NOT NULL PRIMARY KEY,
    name        VARCHAR(120) NOT NULL,
    slug        VARCHAR(60)  NOT NULL UNIQUE,
    description TEXT,
    owner_id    CHAR(36)     NOT NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT,
    INDEX idx_ws_owner (owner_id)
) ENGINE=InnoDB;

-- ── workspace_members ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id CHAR(36) NOT NULL,
    user_id      CHAR(36) NOT NULL,
    role         ENUM('admin','member','viewer') NOT NULL DEFAULT 'member',
    joined_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id),
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id)      REFERENCES users(id)      ON DELETE CASCADE
) ENGINE=InnoDB;

-- ── projects ──────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS projects (
    id           CHAR(36)     NOT NULL PRIMARY KEY,
    workspace_id CHAR(36)     NOT NULL,
    name         VARCHAR(120) NOT NULL,
    description  TEXT,
    color        CHAR(7)      NOT NULL DEFAULT '#6366F1',
    status       ENUM('active','archived','deleted') NOT NULL DEFAULT 'active',
    lead_id      CHAR(36),
    start_date   DATE,
    target_date  DATE,
    created_by   CHAR(36)     NOT NULL,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (lead_id)      REFERENCES users(id)      ON DELETE SET NULL,
    FOREIGN KEY (created_by)   REFERENCES users(id)      ON DELETE RESTRICT,
    INDEX idx_projects_workspace (workspace_id),
    INDEX idx_projects_status   (status)
) ENGINE=InnoDB;

-- ── labels ────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS labels (
    id           CHAR(36)    NOT NULL PRIMARY KEY,
    workspace_id CHAR(36)    NOT NULL,
    name         VARCHAR(60) NOT NULL,
    color        CHAR(7)     NOT NULL DEFAULT '#94A3B8',
    created_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    UNIQUE KEY uq_label_ws_name (workspace_id, name)
) ENGINE=InnoDB;

-- ── tasks ─────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS tasks (
    id           CHAR(36)     NOT NULL PRIMARY KEY,
    project_id   CHAR(36)     NOT NULL,
    parent_id    CHAR(36),                          -- sub-task support
    title        VARCHAR(512) NOT NULL,
    description  TEXT,
    status       ENUM('backlog','todo','in_progress','in_review','done','cancelled')
                              NOT NULL DEFAULT 'backlog',
    priority     ENUM('urgent','high','medium','low','none')
                              NOT NULL DEFAULT 'medium',
    assignee_id  CHAR(36),
    reporter_id  CHAR(36)     NOT NULL,
    position     FLOAT        NOT NULL DEFAULT 0,   -- for drag-and-drop ordering
    estimate_h   FLOAT,                             -- estimated hours
    due_date     DATE,
    completed_at DATETIME,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id)  REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id)   REFERENCES tasks(id)    ON DELETE SET NULL,
    FOREIGN KEY (assignee_id) REFERENCES users(id)    ON DELETE SET NULL,
    FOREIGN KEY (reporter_id) REFERENCES users(id)    ON DELETE RESTRICT,
    INDEX idx_tasks_project  (project_id),
    INDEX idx_tasks_assignee (assignee_id),
    INDEX idx_tasks_status   (status),
    INDEX idx_tasks_priority (priority),
    FULLTEXT  ft_tasks_title_desc (title, description)
) ENGINE=InnoDB;

-- ── task_labels ───────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS task_labels (
    task_id  CHAR(36) NOT NULL,
    label_id CHAR(36) NOT NULL,
    PRIMARY KEY (task_id, label_id),
    FOREIGN KEY (task_id)  REFERENCES tasks(id)  ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- ── comments ──────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS comments (
    id         CHAR(36)  NOT NULL PRIMARY KEY,
    task_id    CHAR(36)  NOT NULL,
    author_id  CHAR(36)  NOT NULL,
    body       TEXT      NOT NULL,
    edited_at  DATETIME,
    created_at DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id)   REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT,
    INDEX idx_comments_task (task_id)
) ENGINE=InnoDB;

-- ── activity_log ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS activity_log (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36)        NOT NULL,
    actor_id     CHAR(36),
    entity_type  VARCHAR(40)     NOT NULL,   -- 'task', 'project', 'member'
    entity_id    CHAR(36)        NOT NULL,
    action       VARCHAR(60)     NOT NULL,   -- 'created', 'status_changed', etc.
    meta         JSON,
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    INDEX idx_activity_workspace (workspace_id),
    INDEX idx_activity_entity    (entity_type, entity_id),
    INDEX idx_activity_actor     (actor_id),
    INDEX idx_activity_time      (created_at)
) ENGINE=InnoDB;

-- ── refresh_tokens ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         CHAR(36)    NOT NULL PRIMARY KEY,
    user_id    CHAR(36)    NOT NULL,
    token_hash CHAR(64)    NOT NULL UNIQUE,   -- SHA-256 hex
    expires_at DATETIME    NOT NULL,
    revoked    TINYINT(1)  NOT NULL DEFAULT 0,
    user_agent VARCHAR(255),
    ip_address VARCHAR(45),
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_rt_user    (user_id),
    INDEX idx_rt_expires (expires_at)
) ENGINE=InnoDB;

-- ── seed: default admin ───────────────────────────────────────────────────────
-- Password: Admin@1234  (bcrypt hash below — change in production!)

INSERT IGNORE INTO users (id, email, username, display_name, password_hash, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@taskflow.io',
    'admin',
    'TaskFlow Admin',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj4J/qqIYFpu',
    'admin'
);
