package v1

import (
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/service"
	"github.com/go-chi/chi/v5"
)

// ProviderHandler wires provider-related HTTP routes.
type ProviderHandler struct {
	svc *service.ProviderService
}

// NewProviderHandler creates a ProviderHandler.
func NewProviderHandler(svc *service.ProviderService) *ProviderHandler {
	return &ProviderHandler{svc: svc}
}

// Register mounts the provider routes on r.
func (h *ProviderHandler) Register(r chi.Router) {
	// Available providers (from registry + config)
	r.Get("/providers", h.listAvailable)

	// Test a connection without saving
	r.Post("/providers/test", h.testConnection)

	// Saved provider profiles (persisted in SQLite)
	r.Get("/provider-profiles", h.listProfiles)
	r.Post("/provider-profiles", h.createProfile)
	r.Get("/provider-profiles/{id}", h.getProfile)
	r.Put("/provider-profiles/{id}", h.updateProfile)
	r.Delete("/provider-profiles/{id}", h.deleteProfile)
	r.Post("/provider-profiles/{id}/test", h.testProfile)
}

// listAvailable GET /api/v1/providers
func (h *ProviderHandler) listAvailable(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, h.svc.ListAvailable())
}

// testConnection POST /api/v1/providers/test
// Body: { provider_name, base_url, api_key_env, model }
func (h *ProviderHandler) testConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProviderName string `json:"provider_name"`
		BaseURL      string `json:"base_url"`
		APIKeyEnv    string `json:"api_key_env"`
		Model        string `json:"model"`
	}
	if !decode(w, r, &req) {
		return
	}
	result := h.svc.TestConnection(r.Context(), req.ProviderName, req.BaseURL, req.APIKeyEnv, req.Model)
	jsonOK(w, result)
}

// listProfiles GET /api/v1/provider-profiles
func (h *ProviderHandler) listProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.svc.List(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	out := make([]*providerProfileDTO, len(profiles))
	for i, p := range profiles {
		out[i] = toProviderProfileDTO(p)
	}
	if out == nil {
		out = []*providerProfileDTO{}
	}
	jsonOK(w, out)
}

// createProfile POST /api/v1/provider-profiles
func (h *ProviderHandler) createProfile(w http.ResponseWriter, r *http.Request) {
	var req profileInput
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Create(r.Context(),
		req.Name, req.ProviderName, req.BaseURL, req.APIKeyEnv,
		req.Model, req.TimeoutSec, req.IsDefault,
	)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonCreated(w, toProviderProfileDTO(p))
}

// getProfile GET /api/v1/provider-profiles/{id}
func (h *ProviderHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toProviderProfileDTO(p))
}

// updateProfile PUT /api/v1/provider-profiles/{id}
func (h *ProviderHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileInput
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Update(r.Context(),
		chi.URLParam(r, "id"),
		req.Name, req.ProviderName, req.BaseURL, req.APIKeyEnv,
		req.Model, req.TimeoutSec, req.IsDefault,
	)
	if err != nil {
		handleErr(w, err)
		return
	}
	jsonOK(w, toProviderProfileDTO(p))
}

// deleteProfile DELETE /api/v1/provider-profiles/{id}
func (h *ProviderHandler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleErr(w, err)
		return
	}
	jsonNoContent(w)
}

// testProfile POST /api/v1/provider-profiles/{id}/test
func (h *ProviderHandler) testProfile(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	result := h.svc.TestConnection(r.Context(), p.ProviderName, p.BaseURL, p.APIKeyEnv, p.Model)
	jsonOK(w, result)
}
