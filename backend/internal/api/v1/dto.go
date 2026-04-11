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

// ─────────────────────────────────────────────
// Plan / Step DTOs
// ─────────────────────────────────────────────

type planStepDTO struct {
	ID          string            `json:"id"`
	PlanID      string            `json:"plan_id"`
	Order       int               `json:"order"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.StepStatus `json:"status"`
	ToolName    string            `json:"tool_name"`
	ToolInput   string            `json:"tool_input"`
	ToolOutput  string            `json:"tool_output"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

type planDTO struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	Steps     []*planStepDTO `json:"steps"`
	CreatedAt time.Time      `json:"created_at"`
}

func toPlanDTO(p *domain.Plan) *planDTO {
	steps := make([]*planStepDTO, len(p.Steps))
	for i, s := range p.Steps {
		steps[i] = &planStepDTO{
			ID:          s.ID,
			PlanID:      s.PlanID,
			Order:       s.Order,
			Title:       s.Title,
			Description: s.Description,
			Status:      s.Status,
			ToolName:    s.ToolName,
			ToolInput:   s.ToolInput,
			ToolOutput:  s.ToolOutput,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
		}
	}
	return &planDTO{
		ID:        p.ID,
		TaskID:    p.TaskID,
		Steps:     steps,
		CreatedAt: p.CreatedAt,
	}
}

// ─────────────────────────────────────────────
// Artifact DTOs
// ─────────────────────────────────────────────

type artifactDTO struct {
	ID            string              `json:"id"`
	TaskID        string              `json:"task_id"`
	ProjectID     string              `json:"project_id"`
	Name          string              `json:"name"`
	Type          domain.ArtifactType `json:"type"`
	ContentPath   string              `json:"content_path"`
	ContentInline string              `json:"content_inline,omitempty"`
	Size          int64               `json:"size"`
	CreatedAt     time.Time           `json:"created_at"`
}

func toArtifactDTO(a *domain.Artifact) *artifactDTO {
	return &artifactDTO{
		ID:            a.ID,
		TaskID:        a.TaskID,
		ProjectID:     a.ProjectID,
		Name:          a.Name,
		Type:          a.Type,
		ContentPath:   a.ContentPath,
		ContentInline: a.ContentInline,
		Size:          a.Size,
		CreatedAt:     a.CreatedAt,
	}
}

// ─────────────────────────────────────────────
// Event DTOs
// ─────────────────────────────────────────────

type eventDTO struct {
	ID        string          `json:"id"`
	TaskID    string          `json:"task_id"`
	Type      domain.EventType `json:"type"`
	Payload   string          `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func toEventDTO(e *domain.Event) *eventDTO {
	return &eventDTO{
		ID:        e.ID,
		TaskID:    e.TaskID,
		Type:      e.Type,
		Payload:   e.Payload,
		CreatedAt: e.CreatedAt,
	}
}

// ─────────────────────────────────────────────
// Memory DTOs
// ─────────────────────────────────────────────

type contextFileDTO struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	FilePath    string    `json:"file_path"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toContextFileDTO(cf *domain.ContextFile) *contextFileDTO {
	return &contextFileDTO{
		ID:          cf.ID,
		ProjectID:   cf.ProjectID,
		Name:        cf.Name,
		FilePath:    cf.FilePath,
		Content:     cf.Content,
		Description: cf.Description,
		CreatedAt:   cf.CreatedAt,
		UpdatedAt:   cf.UpdatedAt,
	}
}

type taskTemplateDTO struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ProviderID  string    `json:"provider_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toTaskTemplateDTO(t *domain.TaskTemplate) *taskTemplateDTO {
	return &taskTemplateDTO{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
		ProviderID:  t.ProviderID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
