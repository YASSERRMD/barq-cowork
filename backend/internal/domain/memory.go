package domain

import "time"

// ─────────────────────────────────────────────
// ContextFile — a reference document attached to a project.
// The planner reads all context files before generating a plan.
// ─────────────────────────────────────────────

// ContextFile is a named snippet of text (or a workspace file path) attached
// to a project that is injected into every planning prompt automatically.
type ContextFile struct {
	ID          string
	ProjectID   string
	Name        string // short human label, e.g. "API schema" or "coding conventions"
	FilePath    string // optional: path relative to workspace root
	Content     string // inline text content; used when FilePath is empty
	Description string // one-line explanation shown in the UI
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ─────────────────────────────────────────────
// TaskTemplate — a reusable task blueprint per project.
// ─────────────────────────────────────────────

// TaskTemplate stores a pre-filled task that users can load into the task
// creation form. Templates belong to a project and carry optional
// provider preferences.
type TaskTemplate struct {
	ID          string
	ProjectID   string
	Name        string // human label, e.g. "Weekly summary report"
	Title       string // task title pre-fill
	Description string // task description pre-fill
	ProviderID  string // optional preferred provider profile
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ─────────────────────────────────────────────
// WorkspaceMemory — abstraction for project context retrieval.
// The simple implementation loads all ContextFiles by project ID.
// A future implementation can add semantic / vector retrieval.
// ─────────────────────────────────────────────

// WorkspaceMemory retrieves contextual information for a project.
// Implementations must be safe for concurrent use.
type WorkspaceMemory interface {
	// Recall returns relevant context snippets for the given project.
	// The query parameter is reserved for future semantic search;
	// the simple implementation ignores it and returns all context files.
	Recall(projectID string, query string) ([]MemoryEntry, error)
}

// MemoryEntry is a single piece of retrieved context.
type MemoryEntry struct {
	Source  string // e.g. "context_file:uuid" or "artifact:uuid"
	Label   string // human-readable name
	Content string // text to inject into the prompt
	Score   float64 // relevance score (0–1); 1 = exact, lower = heuristic
}
