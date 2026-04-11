-- Migration 005: schedules for recurring tasks
CREATE TABLE IF NOT EXISTS schedules (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cron_expr   TEXT NOT NULL,
    task_title  TEXT NOT NULL,
    task_desc   TEXT NOT NULL DEFAULT '',
    provider_id TEXT NOT NULL DEFAULT '',
    enabled     INTEGER NOT NULL DEFAULT 1,
    last_run_at TEXT,
    next_run_at TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedules_project_id ON schedules(project_id);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled    ON schedules(enabled);
