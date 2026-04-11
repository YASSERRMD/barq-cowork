package domain

import (
	"context"
	"time"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusPlanning  TaskStatus = "planning"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ValidTaskStatuses is the set of allowed task status values.
var ValidTaskStatuses = map[TaskStatus]bool{
	TaskStatusPending:   true,
	TaskStatusPlanning:  true,
	TaskStatusRunning:   true,
	TaskStatusCompleted: true,
	TaskStatusFailed:    true,
	TaskStatusCancelled: true,
}

// Task is the primary unit of work. A task is submitted by the user,
// planned by the agent runtime, and executed step-by-step.
type Task struct {
	ID          string
	ProjectID   string
	Title       string
	Description string
	Status      TaskStatus
	ProviderID  string // references a ProviderProfile.ID; empty = use project/workspace default
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// Validate returns a ValidationError if required fields are missing.
func (t *Task) Validate() error {
	if t.Title == "" {
		return &ValidationError{Field: "title", Message: "required"}
	}
	if t.ProjectID == "" {
		return &ValidationError{Field: "project_id", Message: "required"}
	}
	return nil
}

// CanTransitionTo returns true if the task can legally move to next.
func (t *Task) CanTransitionTo(next TaskStatus) bool {
	allowed := map[TaskStatus][]TaskStatus{
		TaskStatusPending:  {TaskStatusPlanning, TaskStatusCancelled},
		TaskStatusPlanning: {TaskStatusRunning, TaskStatusFailed, TaskStatusCancelled},
		TaskStatusRunning:  {TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled},
		// terminal states — no transitions allowed
		TaskStatusCompleted: {},
		TaskStatusFailed:    {},
		TaskStatusCancelled: {},
	}
	for _, s := range allowed[t.Status] {
		if s == next {
			return true
		}
	}
	return false
}

// TaskRepository is the storage port for tasks.
type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	ListByProject(ctx context.Context, projectID string) ([]*Task, error)
	Update(ctx context.Context, t *Task) error
	UpdateStatus(ctx context.Context, id string, status TaskStatus, now time.Time) error
	Delete(ctx context.Context, id string) error
}
