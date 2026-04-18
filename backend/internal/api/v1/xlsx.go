package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/barq-cowork/barq-cowork/internal/generator"
	"github.com/go-chi/chi/v5"
)

// XlsxHandler streams a dynamically generated .xlsx workbook straight to the
// caller. It is the public counterpart to the write_xlsx agent tool: the
// frontend (or any HTTP client) can POST a JSON spec and get the bytes back
// as a file download — no workspace, no disk write, no job.
type XlsxHandler struct{}

// NewXlsxHandler returns the zero-value handler. Kept function-shaped so the
// router registration follows the same style as the rest of v1.
func NewXlsxHandler() *XlsxHandler { return &XlsxHandler{} }

// Register mounts the xlsx routes on r.
func (h *XlsxHandler) Register(r chi.Router) {
	r.Post("/xlsx/generate", h.generate)
	r.Post("/xlsx/recommend-charts", h.recommendCharts)
}

// xlsxGenerateRequest mirrors generator.XlsxWorkbook but exposes a `filename`
// field the handler uses for the Content-Disposition header (the generator
// itself doesn't care).
type xlsxGenerateRequest struct {
	Filename string                `json:"filename"`
	Title    string                `json:"title,omitempty"`
	Author   string                `json:"author,omitempty"`
	Theme    generator.XlsxTheme   `json:"theme,omitempty"`
	Sheets   []generator.XlsxSheet `json:"sheets"`
}

// generate POST /api/v1/xlsx/generate
//
// Accepts the same JSON shape as the write_xlsx tool and streams the
// resulting .xlsx as a binary download. Errors before any byte is written
// are returned as JSON; once streaming starts, errors are swallowed (the
// HTTP response headers are already gone).
func (h *XlsxHandler) generate(w http.ResponseWriter, r *http.Request) {
	var req xlsxGenerateRequest
	if !decode(w, r, &req) {
		return
	}
	if len(req.Sheets) == 0 {
		jsonError(w, http.StatusBadRequest, "at least one sheet is required")
		return
	}

	data, err := generator.BuildWorkbook(generator.XlsxWorkbook{
		Title:  req.Title,
		Author: req.Author,
		Theme:  req.Theme,
		Sheets: req.Sheets,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	fname := sanitizeDownloadFilename(req.Filename)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, fname))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

// xlsxRecommendRequest is a single-sheet subset — the recommender only needs
// headers + rows to do its job.
type xlsxRecommendRequest struct {
	Sheet generator.XlsxSheet `json:"sheet"`
}

// recommendCharts POST /api/v1/xlsx/recommend-charts
//
// Returns a plausible chart spec for the sheet, or `[]` when nothing is
// plottable. Lets the frontend (or an upstream agent) preview what the
// builder would do before committing to the full generate call.
func (h *XlsxHandler) recommendCharts(w http.ResponseWriter, r *http.Request) {
	var req xlsxRecommendRequest
	if !decode(w, r, &req) {
		return
	}
	if len(req.Sheet.Headers) == 0 {
		jsonError(w, http.StatusBadRequest, "sheet.headers is required")
		return
	}
	charts := generator.RecommendCharts(req.Sheet)
	if charts == nil {
		charts = []generator.XlsxChart{}
	}
	// Use encoder directly — the envelope shape elsewhere wraps under `data`,
	// which matches the rest of v1 so frontend clients can reuse the same
	// unwrap helper.
	jsonOK(w, charts)
}

// sanitizeDownloadFilename keeps the filename header safe for browsers. It
// strips path separators, collapses whitespace, and guarantees an .xlsx
// extension. Empty input falls back to 'workbook.xlsx'.
func sanitizeDownloadFilename(in string) string {
	base := strings.TrimSpace(in)
	if base == "" {
		return "workbook.xlsx"
	}
	// Keep only letters, digits, dot, hyphen, underscore, space.
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.', r == ' ':
			return r
		}
		return '-'
	}, base)
	if !strings.HasSuffix(strings.ToLower(out), ".xlsx") {
		out = strings.TrimSuffix(out, ".") + ".xlsx"
	}
	return out
}

