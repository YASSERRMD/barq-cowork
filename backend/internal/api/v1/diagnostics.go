package v1

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
)

// DiagnosticsHandler exposes endpoints that help operators understand the
// runtime state of barq-coworkd. The /bundle endpoint downloads a ZIP
// containing system info, recent events, and recent artifact metadata.
type DiagnosticsHandler struct {
	events    EventQuerier
	artifacts ArtifactQuerier
	version   string
}

// NewDiagnosticsHandler creates a DiagnosticsHandler.
// It reuses the EventQuerier and ArtifactQuerier interfaces defined in
// execution.go so no extra ports are needed.
func NewDiagnosticsHandler(
	events EventQuerier,
	artifacts ArtifactQuerier,
	version string,
) *DiagnosticsHandler {
	return &DiagnosticsHandler{events: events, artifacts: artifacts, version: version}
}

// Register mounts diagnostics routes on r.
func (h *DiagnosticsHandler) Register(r chi.Router) {
	r.Get("/diagnostics/bundle", h.exportBundle)
	r.Get("/diagnostics/info", h.systemInfo)
}

// ─────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────

// exportBundle GET /api/v1/diagnostics/bundle
// Streams a ZIP file with system.json, events.json, and artifacts.json.
func (h *DiagnosticsHandler) exportBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buf := &bytes.Buffer{}
	zw  := zip.NewWriter(buf)

	// ── system.json ──────────────────────────────────────────────────
	if err := addJSONToZip(zw, "system.json", h.buildSystemInfo()); err != nil {
		http.Error(w, "bundle error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ── events.json ──────────────────────────────────────────────────
	events, _ := h.events.ListRecent(ctx, 500)
	evDTOs := make([]*eventDTO, 0, len(events))
	for _, ev := range events {
		evDTOs = append(evDTOs, toEventDTO(ev))
	}
	if err := addJSONToZip(zw, "events.json", evDTOs); err != nil {
		http.Error(w, "bundle error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ── artifacts.json ───────────────────────────────────────────────
	artifacts, _ := h.artifacts.ListRecent(ctx, 200)
	artDTOs := make([]*artifactDTO, 0, len(artifacts))
	for _, a := range artifacts {
		artDTOs = append(artDTOs, toArtifactDTO(a))
	}
	if err := addJSONToZip(zw, "artifacts.json", artDTOs); err != nil {
		http.Error(w, "bundle error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := zw.Close(); err != nil {
		http.Error(w, "zip close error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("barq-cowork-diag-%s.zip", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// systemInfo GET /api/v1/diagnostics/info
// Returns runtime and build information as JSON.
func (h *DiagnosticsHandler) systemInfo(w http.ResponseWriter, _ *http.Request) {
	jsonOK(w, h.buildSystemInfo())
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

type systemInfoDTO struct {
	GeneratedAt     string            `json:"generated_at"`
	Version         string            `json:"version"`
	GoVersion       string            `json:"go_version"`
	OS              string            `json:"os"`
	Arch            string            `json:"arch"`
	NumCPU          int               `json:"num_cpu"`
	NumGoroutine    int               `json:"num_goroutine"`
	MemAllocMB      float64           `json:"mem_alloc_mb"`
	MemTotalAllocMB float64           `json:"mem_total_alloc_mb"`
	BuildInfo       map[string]string `json:"build_info,omitempty"`
}

func (h *DiagnosticsHandler) buildSystemInfo() systemInfoDTO {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	info := systemInfoDTO{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Version:         h.version,
		GoVersion:       runtime.Version(),
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		NumCPU:          runtime.NumCPU(),
		NumGoroutine:    runtime.NumGoroutine(),
		MemAllocMB:      float64(ms.Alloc) / (1 << 20),
		MemTotalAllocMB: float64(ms.TotalAlloc) / (1 << 20),
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		info.BuildInfo = map[string]string{
			"path":       bi.Path,
			"go_version": bi.GoVersion,
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision", "vcs.time", "CGO_ENABLED":
				info.BuildInfo[s.Key] = s.Value
			}
		}
	}

	return info
}

func addJSONToZip(zw *zip.Writer, name string, v any) error {
	fw, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fw)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
