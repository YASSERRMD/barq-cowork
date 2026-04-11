// Package service contains application services that orchestrate domain logic
// and repository operations. Services are the boundary between HTTP handlers
// and the domain/storage layers.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/google/uuid"
)

// WorkspaceService handles workspace lifecycle operations.
type WorkspaceService struct {
	repo domain.WorkspaceRepository
}

// NewWorkspaceService creates a WorkspaceService backed by repo.
func NewWorkspaceService(repo domain.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

// Create validates and persists a new workspace, returning it with generated ID.
func (s *WorkspaceService) Create(ctx context.Context, name, description, rootPath string) (*domain.Workspace, error) {
	w := &domain.Workspace{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		RootPath:    rootPath,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("workspace service create: %w", err)
	}
	return w, nil
}

// Get retrieves a workspace by ID.
func (s *WorkspaceService) Get(ctx context.Context, id string) (*domain.Workspace, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all workspaces ordered by creation time descending.
func (s *WorkspaceService) List(ctx context.Context) ([]*domain.Workspace, error) {
	return s.repo.List(ctx)
}

// Update replaces the mutable fields of a workspace.
func (s *WorkspaceService) Update(ctx context.Context, id, name, description, rootPath string) (*domain.Workspace, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	w.Name = name
	w.Description = description
	w.RootPath = rootPath
	w.UpdatedAt = time.Now().UTC()
	if err := w.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, w); err != nil {
		return nil, fmt.Errorf("workspace service update: %w", err)
	}
	return w, nil
}

// Delete removes a workspace and all its projects + tasks (cascade).
func (s *WorkspaceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
