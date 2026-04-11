package domain

import (
	"context"
	"time"
)

// Project belongs to a Workspace. It carries persistent instructions that are
// prepended to every task prompt within the project.
type Project struct {
	ID           string
	WorkspaceID  string
	Name         string
	Description  string
	Instructions string // system-level instructions for every task in this project
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate returns a ValidationError if required fields are missing.
func (p *Project) Validate() error {
	if p.Name == "" {
		return &ValidationError{Field: "name", Message: "required"}
	}
	if p.WorkspaceID == "" {
		return &ValidationError{Field: "workspace_id", Message: "required"}
	}
	return nil
}

// ProjectRepository is the storage port for projects.
type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Project, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id string) error
}
