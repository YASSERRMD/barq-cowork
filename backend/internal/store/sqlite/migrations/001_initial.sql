-- Migration 001: initial schema
-- All timestamps stored as ISO8601 UTC text for portability.

CREATE TABLE IF NOT EXISTS workspaces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    root_path   TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_workspace_id ON projects(workspace_id);

CREATE TABLE IF NOT EXISTS tasks (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    provider_id  TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    started_at   TEXT,
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks(status);

CREATE TABLE IF NOT EXISTS plans (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_task_id ON plans(task_id);

CREATE TABLE IF NOT EXISTS plan_steps (
    id           TEXT PRIMARY KEY,
    plan_id      TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    step_order   INTEGER NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    tool_name    TEXT NOT NULL DEFAULT '',
    tool_input   TEXT NOT NULL DEFAULT '',
    tool_output  TEXT NOT NULL DEFAULT '',
    started_at   TEXT,
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_plan_steps_plan_id ON plan_steps(plan_id);

CREATE TABLE IF NOT EXISTS artifacts (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    project_id     TEXT NOT NULL,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL,
    content_path   TEXT NOT NULL DEFAULT '',
    content_inline TEXT NOT NULL DEFAULT '',
    size           INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_task_id    ON artifacts(task_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_project_id ON artifacts(project_id);

CREATE TABLE IF NOT EXISTS events (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL,
    type       TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id);

CREATE TABLE IF NOT EXISTS approval_requests (
    id          TEXT PRIMARY KEY,
    task_id     TEXT NOT NULL,
    tool_name   TEXT NOT NULL,
    action      TEXT NOT NULL,
    payload     TEXT NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'pending',
    resolution  TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_approvals_task_id ON approval_requests(task_id);
CREATE INDEX IF NOT EXISTS idx_approvals_status  ON approval_requests(status);

CREATE TABLE IF NOT EXISTS provider_profiles (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    base_url      TEXT NOT NULL,
    api_key_env   TEXT NOT NULL,
    model         TEXT NOT NULL,
    timeout_sec   INTEGER NOT NULL DEFAULT 120,
    is_default    INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
