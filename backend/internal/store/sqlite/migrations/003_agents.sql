-- Migration 003: sub-agents for parallel execution

CREATE TABLE IF NOT EXISTS sub_agents (
    id             TEXT PRIMARY KEY,
    parent_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role           TEXT NOT NULL DEFAULT 'custom',
    title          TEXT NOT NULL,
    instructions   TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending',
    plan_id        TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL,
    started_at     TEXT,
    completed_at   TEXT
);

CREATE INDEX IF NOT EXISTS idx_sub_agents_parent_task_id ON sub_agents(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_sub_agents_status         ON sub_agents(status);
