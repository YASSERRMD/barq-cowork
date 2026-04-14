package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// SkillHandler wires skill HTTP routes.
type SkillHandler struct {
	svc *service.SkillService
}

// NewSkillHandler creates a SkillHandler.
func NewSkillHandler(svc *service.SkillService) *SkillHandler {
	return &SkillHandler{svc: svc}
}

// Register mounts skill routes on r.
func (h *SkillHandler) Register(r chi.Router) {
	r.Get("/skills", h.list)
	r.Post("/skills", h.create)
	r.Get("/skills/{id}", h.get)
	r.Patch("/skills/{id}/enabled", h.updateEnabled)
	r.Delete("/skills/{id}", h.delete)
}

// list GET /api/v1/skills
func (h *SkillHandler) list(w http.ResponseWriter, r *http.Request) {
	skills, err := h.svc.List(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*skillDTO, len(skills))
	for i, s := range skills {
		out[i] = toSkillDTO(s)
	}
	if out == nil {
		out = []*skillDTO{}
	}
	jsonOK(w, out)
}

// get GET /api/v1/skills/{id}
func (h *SkillHandler) get(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if !ok {
		handleErr(w, domain.ErrNotFound)
		return
	}
	jsonOK(w, toSkillDTO(sk))
}

// create POST /api/v1/skills
func (h *SkillHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string   `json:"name"`
		Kind           string   `json:"kind"`
		Description    string   `json:"description"`
		OutputMimeType string   `json:"output_mime_type"`
		OutputFileExt  string   `json:"output_file_ext"`
		PromptTemplate string   `json:"prompt_template"`
		Tags           []string `json:"tags"`
		InputMimeTypes []string `json:"input_mime_types"`
	}
	if !decode(w, r, &req) {
		return
	}
	sk := &domain.SkillSpec{
		Name:           req.Name,
		Kind:           domain.SkillKind(req.Kind),
		Description:    req.Description,
		OutputMimeType: req.OutputMimeType,
		OutputFileExt:  req.OutputFileExt,
		PromptTemplate: req.PromptTemplate,
		Tags:           req.Tags,
		InputMimeTypes: req.InputMimeTypes,
	}
	created, err := h.svc.Create(r.Context(), sk)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toSkillDTO(created))
}

// updateEnabled PATCH /api/v1/skills/{id}/enabled
func (h *SkillHandler) updateEnabled(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &req) {
		return
	}
	sk, err := h.svc.UpdateEnabled(r.Context(), chi.URLParam(r, "id"), req.Enabled)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toSkillDTO(sk))
}

// delete DELETE /api/v1/skills/{id}
func (h *SkillHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}

// ── DTO ───────────────────────────────────────────────────────────

type skillDTO struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	OutputMimeType string   `json:"output_mime_type"`
	OutputFileExt  string   `json:"output_file_ext"`
	PromptTemplate string   `json:"prompt_template"`
	BuiltIn        bool     `json:"built_in"`
	Enabled        bool     `json:"enabled"`
	Tags           []string `json:"tags"`
	InputMimeTypes []string `json:"input_mime_types"`
}

func toSkillDTO(s *domain.SkillSpec) *skillDTO {
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	mimes := s.InputMimeTypes
	if mimes == nil {
		mimes = []string{}
	}
	return &skillDTO{
		ID:             s.ID,
		Name:           s.Name,
		Kind:           string(s.Kind),
		Description:    s.Description,
		OutputMimeType: s.OutputMimeType,
		OutputFileExt:  s.OutputFileExt,
		PromptTemplate: s.PromptTemplate,
		BuiltIn:        s.BuiltIn,
		Enabled:        s.Enabled,
		Tags:           tags,
		InputMimeTypes: mimes,
	}
}
