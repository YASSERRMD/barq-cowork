package v1

import (
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// DTOs are the wire representations. They are intentionally separate from
// domain types so the HTTP API can evolve independently.

type workspaceDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RootPath    string    `json:"root_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toWorkspaceDTO(w *domain.Workspace) *workspaceDTO {
	return &workspaceDTO{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		RootPath:    w.RootPath,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

type projectDTO struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Instructions string    `json:"instructions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toProjectDTO(p *domain.Project) *projectDTO {
	return &projectDTO{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		Description:  p.Description,
		Instructions: p.Instructions,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

type taskDTO struct {
	ID          string             `json:"id"`
	ProjectID   string             `json:"project_id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Status      domain.TaskStatus  `json:"status"`
	ProviderID  string             `json:"provider_id"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

func toTaskDTO(t *domain.Task) *taskDTO {
	return &taskDTO{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		ProviderID:  t.ProviderID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
	}
}
