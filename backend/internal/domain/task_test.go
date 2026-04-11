package domain_test

import (
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    domain.Task
		wantErr bool
	}{
		{"valid task with project", domain.Task{Title: "Do X", ProjectID: "p1"}, false},
		{"valid direct task no project", domain.Task{Title: "Do X"}, false}, // project-less tasks are allowed
		{"missing title", domain.Task{ProjectID: "p1"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.task.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestTask_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from domain.TaskStatus
		to   domain.TaskStatus
		want bool
	}{
		{domain.TaskStatusPending, domain.TaskStatusPlanning, true},
		{domain.TaskStatusPending, domain.TaskStatusRunning, false},
		{domain.TaskStatusPlanning, domain.TaskStatusRunning, true},
		{domain.TaskStatusRunning, domain.TaskStatusCompleted, true},
		{domain.TaskStatusRunning, domain.TaskStatusFailed, true},
		{domain.TaskStatusRunning, domain.TaskStatusCancelled, true},
		{domain.TaskStatusCompleted, domain.TaskStatusRunning, false},
		{domain.TaskStatusFailed, domain.TaskStatusRunning, false},
		{domain.TaskStatusCancelled, domain.TaskStatusPending, false},
	}
	for _, tc := range tests {
		task := domain.Task{Status: tc.from}
		got := task.CanTransitionTo(tc.to)
		if got != tc.want {
			t.Errorf("CanTransitionTo(%s → %s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestWorkspace_Validate(t *testing.T) {
	if err := (&domain.Workspace{Name: "ok"}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := (&domain.Workspace{}).Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestProject_Validate(t *testing.T) {
	if err := (&domain.Project{Name: "ok", WorkspaceID: "w1"}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := (&domain.Project{WorkspaceID: "w1"}).Validate(); err == nil {
		t.Error("expected error for empty name")
	}
	if err := (&domain.Project{Name: "ok"}).Validate(); err == nil {
		t.Error("expected error for empty workspace_id")
	}
}
