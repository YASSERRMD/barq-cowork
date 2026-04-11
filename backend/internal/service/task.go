package service

import (
	"context"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/google/uuid"
)

// TaskService handles task lifecycle operations.
type TaskService struct {
	tasks    domain.TaskRepository
	projects domain.ProjectRepository
}

// NewTaskService creates a TaskService.
func NewTaskService(tasks domain.TaskRepository, projects domain.ProjectRepository) *TaskService {
	return &TaskService{tasks: tasks, projects: projects}
}

// Create validates, checks project existence, and persists a new task.
func (s *TaskService) Create(ctx context.Context, projectID, title, description, providerID string) (*domain.Task, error) {
	// Verify project exists.
	if _, err := s.projects.GetByID(ctx, projectID); err != nil {
		return nil, fmt.Errorf("project %s: %w", projectID, err)
	}

	now := time.Now().UTC()
	t := &domain.Task{
		ID:          uuid.NewString(),
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      domain.TaskStatusPending,
		ProviderID:  providerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("task service create: %w", err)
	}
	return t, nil
}

// Get retrieves a task by ID.
func (s *TaskService) Get(ctx context.Context, id string) (*domain.Task, error) {
	return s.tasks.GetByID(ctx, id)
}

// ListByProject returns all tasks for a project, newest first.
func (s *TaskService) ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	return s.tasks.ListByProject(ctx, projectID)
}

// UpdateStatus transitions a task to a new status, enforcing valid transitions.
func (s *TaskService) UpdateStatus(ctx context.Context, id string, next domain.TaskStatus) (*domain.Task, error) {
	t, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !t.CanTransitionTo(next) {
		return nil, &domain.ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("cannot transition from %s to %s", t.Status, next),
		}
	}
	if err := s.tasks.UpdateStatus(ctx, id, next, time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("task status update: %w", err)
	}
	return s.tasks.GetByID(ctx, id)
}

// Update replaces the mutable fields of a task.
func (s *TaskService) Update(ctx context.Context, id, title, description, providerID string) (*domain.Task, error) {
	t, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Title = title
	t.Description = description
	t.ProviderID = providerID
	t.UpdatedAt = time.Now().UTC()
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("task service update: %w", err)
	}
	return t, nil
}

// Delete removes a task.
func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.tasks.Delete(ctx, id)
}
