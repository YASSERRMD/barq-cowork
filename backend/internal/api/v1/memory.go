package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Port interfaces
// ─────────────────────────────────────────────

// ContextFileStore is the persistence port for context files.
type ContextFileStore interface {
	Create(ctx context.Context, cf *domain.ContextFile) error
	GetByID(ctx context.Context, id string) (*domain.ContextFile, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.ContextFile, error)
	Update(ctx context.Context, cf *domain.ContextFile) error
	Delete(ctx context.Context, id string) error
}

// TaskTemplateStore is the persistence port for task templates.
type TaskTemplateStore interface {
	Create(ctx context.Context, t *domain.TaskTemplate) error
	GetByID(ctx context.Context, id string) (*domain.TaskTemplate, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.TaskTemplate, error)
	Update(ctx context.Context, t *domain.TaskTemplate) error
	Delete(ctx context.Context, id string) error
}

// ─────────────────────────────────────────────
// MemoryHandler
// ─────────────────────────────────────────────

// MemoryHandler mounts project memory routes (context files + templates).
type MemoryHandler struct {
	ctxFiles  ContextFileStore
	templates TaskTemplateStore
}

// NewMemoryHandler creates a MemoryHandler.
func NewMemoryHandler(cf ContextFileStore, tt TaskTemplateStore) *MemoryHandler {
	return &MemoryHandler{ctxFiles: cf, templates: tt}
}

// Register mounts routes on the given /api/v1 sub-router.
func (h *MemoryHandler) Register(r chi.Router) {
	// Context files (per project)
	r.Get("/projects/{projectID}/context-files", h.listContextFiles)
	r.Post("/projects/{projectID}/context-files", h.createContextFile)
	r.Get("/context-files/{id}", h.getContextFile)
	r.Put("/context-files/{id}", h.updateContextFile)
	r.Delete("/context-files/{id}", h.deleteContextFile)

	// Task templates (per project)
	r.Get("/projects/{projectID}/templates", h.listTemplates)
	r.Post("/projects/{projectID}/templates", h.createTemplate)
	r.Get("/templates/{id}", h.getTemplate)
	r.Put("/templates/{id}", h.updateTemplate)
	r.Delete("/templates/{id}", h.deleteTemplate)
}

// ─────────────────────────────────────────────
// Context file handlers
// ─────────────────────────────────────────────

func (h *MemoryHandler) listContextFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.ctxFiles.ListByProject(r.Context(), chi.URLParam(r, "projectID"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*contextFileDTO, len(files))
	for i, cf := range files {
		out[i] = toContextFileDTO(cf)
	}
	if out == nil {
		out = []*contextFileDTO{}
	}
	jsonOK(w, out)
}

func (h *MemoryHandler) createContextFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		FilePath    string `json:"file_path"`
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Name == "" {
		jsonError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	now := time.Now().UTC()
	cf := &domain.ContextFile{
		ID:          uuid.NewString(),
		ProjectID:   chi.URLParam(r, "projectID"),
		Name:        req.Name,
		FilePath:    req.FilePath,
		Content:     req.Content,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.ctxFiles.Create(r.Context(), cf); err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toContextFileDTO(cf))
}

func (h *MemoryHandler) getContextFile(w http.ResponseWriter, r *http.Request) {
	cf, err := h.ctxFiles.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toContextFileDTO(cf))
}

func (h *MemoryHandler) updateContextFile(w http.ResponseWriter, r *http.Request) {
	cf, err := h.ctxFiles.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	var req struct {
		Name        string `json:"name"`
		FilePath    string `json:"file_path"`
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if !decode(w, r, &req) {
		return
	}
	cf.Name = req.Name
	cf.FilePath = req.FilePath
	cf.Content = req.Content
	cf.Description = req.Description
	if err := h.ctxFiles.Update(r.Context(), cf); err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toContextFileDTO(cf))
}

func (h *MemoryHandler) deleteContextFile(w http.ResponseWriter, r *http.Request) {
	if err := h.ctxFiles.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}

// ─────────────────────────────────────────────
// Task template handlers
// ─────────────────────────────────────────────

func (h *MemoryHandler) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templates.ListByProject(r.Context(), chi.URLParam(r, "projectID"))
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*taskTemplateDTO, len(templates))
	for i, t := range templates {
		out[i] = toTaskTemplateDTO(t)
	}
	if out == nil {
		out = []*taskTemplateDTO{}
	}
	jsonOK(w, out)
}

func (h *MemoryHandler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ProviderID  string `json:"provider_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Name == "" || req.Title == "" {
		jsonError(w, http.StatusUnprocessableEntity, "name and title are required")
		return
	}
	now := time.Now().UTC()
	t := &domain.TaskTemplate{
		ID:          uuid.NewString(),
		ProjectID:   chi.URLParam(r, "projectID"),
		Name:        req.Name,
		Title:       req.Title,
		Description: req.Description,
		ProviderID:  req.ProviderID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.templates.Create(r.Context(), t); err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toTaskTemplateDTO(t))
}

func (h *MemoryHandler) getTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := h.templates.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toTaskTemplateDTO(t))
}

func (h *MemoryHandler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := h.templates.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ProviderID  string `json:"provider_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	t.Name = req.Name
	t.Title = req.Title
	t.Description = req.Description
	t.ProviderID = req.ProviderID
	if err := h.templates.Update(r.Context(), t); err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toTaskTemplateDTO(t))
}

func (h *MemoryHandler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}
