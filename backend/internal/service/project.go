package service

import (
	"context"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/google/uuid"
)

// ProjectService handles project lifecycle operations.
type ProjectService struct {
	projects   domain.ProjectRepository
	workspaces domain.WorkspaceRepository
}

// NewProjectService creates a ProjectService.
func NewProjectService(projects domain.ProjectRepository, workspaces domain.WorkspaceRepository) *ProjectService {
	return &ProjectService{projects: projects, workspaces: workspaces}
}

// Create validates, checks workspace existence, and persists a new project.
func (s *ProjectService) Create(ctx context.Context, workspaceID, name, description, instructions string) (*domain.Project, error) {
	// Verify workspace exists.
	if _, err := s.workspaces.GetByID(ctx, workspaceID); err != nil {
		return nil, fmt.Errorf("workspace %s: %w", workspaceID, err)
	}

	p := &domain.Project{
		ID:           uuid.NewString(),
		WorkspaceID:  workspaceID,
		Name:         name,
		Description:  description,
		Instructions: instructions,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("project service create: %w", err)
	}
	return p, nil
}

// Get retrieves a project by ID.
func (s *ProjectService) Get(ctx context.Context, id string) (*domain.Project, error) {
	return s.projects.GetByID(ctx, id)
}

// ListByWorkspace returns all projects for a workspace.
func (s *ProjectService) ListByWorkspace(ctx context.Context, workspaceID string) ([]*domain.Project, error) {
	return s.projects.ListByWorkspace(ctx, workspaceID)
}

// ListAll returns all projects across all workspaces.
func (s *ProjectService) ListAll(ctx context.Context) ([]*domain.Project, error) {
	return s.projects.ListAll(ctx)
}

// Update replaces the mutable fields of a project.
func (s *ProjectService) Update(ctx context.Context, id, name, description, instructions string) (*domain.Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Name = name
	p.Description = description
	p.Instructions = instructions
	p.UpdatedAt = time.Now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.projects.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("project service update: %w", err)
	}
	return p, nil
}

// Delete removes a project and all its tasks (cascade).
func (s *ProjectService) Delete(ctx context.Context, id string) error {
	return s.projects.Delete(ctx, id)
}
