package domain

import (
	"context"
	"time"
)

// Workspace is the top-level container. Each workspace maps to a filesystem
// root that agent tools are scoped to.
type Workspace struct {
	ID          string
	Name        string
	Description string
	RootPath    string // filesystem root; tools cannot access paths outside this
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate returns a ValidationError if required fields are missing.
func (w *Workspace) Validate() error {
	if w.Name == "" {
		return &ValidationError{Field: "name", Message: "required"}
	}
	return nil
}

// WorkspaceRepository is the storage port for workspaces.
// Implementations live in internal/store/*.
type WorkspaceRepository interface {
	Create(ctx context.Context, w *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	List(ctx context.Context) ([]*Workspace, error)
	Update(ctx context.Context, w *Workspace) error
	Delete(ctx context.Context, id string) error
}
