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

// ─────────────────────────────────────────────
// Provider profile DTOs
// ─────────────────────────────────────────────

type providerProfileDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ProviderName string    `json:"provider_name"`
	BaseURL      string    `json:"base_url"`
	APIKeyEnv    string    `json:"api_key_env"` // env var name only, never the value
	Model        string    `json:"model"`
	TimeoutSec   int       `json:"timeout_sec"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toProviderProfileDTO(p *domain.ProviderProfile) *providerProfileDTO {
	return &providerProfileDTO{
		ID:           p.ID,
		Name:         p.Name,
		ProviderName: p.ProviderName,
		BaseURL:      p.BaseURL,
		APIKeyEnv:    p.APIKeyEnv,
		Model:        p.Model,
		TimeoutSec:   p.TimeoutSec,
		IsDefault:    p.IsDefault,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// profileInput is the shared request body for create/update provider profile.
type profileInput struct {
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	BaseURL      string `json:"base_url"`
	APIKeyEnv    string `json:"api_key_env"`
	Model        string `json:"model"`
	TimeoutSec   int    `json:"timeout_sec"`
	IsDefault    bool   `json:"is_default"`
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
