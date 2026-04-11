// Package v1 contains HTTP handlers for the /api/v1 REST API.
package v1

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// envelope wraps API responses for consistent shape.
type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

func jsonCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

func jsonNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(envelope{Error: msg})
}

// handleErr maps domain errors to HTTP status codes.
func handleErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		jsonError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrValidation):
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrConflict):
		jsonError(w, http.StatusConflict, err.Error())
	default:
		slog.Error("internal error", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
	}
}

// decode reads and JSON-decodes r.Body into dst, returning false on error.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// encodeJSON writes v as JSON to w; errors are swallowed (already writing response).
func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
