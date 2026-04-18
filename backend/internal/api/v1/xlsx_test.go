package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestXlsx_Generate_StreamsBytes posts a minimal spec and expects the handler
// to respond 200 with the xlsx content-type and a non-trivial body. It also
// sanity-checks the Content-Disposition header so browser downloads pick up
// the right filename.
func TestXlsx_Generate_StreamsBytes(t *testing.T) {
	r := chi.NewRouter()
	NewXlsxHandler().Register(r)

	body := map[string]any{
		"filename": "demo",
		"sheets": []map[string]any{{
			"name":    "Sheet",
			"headers": []string{"a", "b"},
			"rows": [][]any{
				{"one", 1},
				{"two", 2},
			},
		}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/xlsx/generate", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/vnd.openxmlformats") {
		t.Errorf("content-type: %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "demo.xlsx") {
		t.Errorf("content-disposition: %q", cd)
	}
	// XLSX is a zip — the magic bytes start with "PK\x03\x04".
	bodyBytes := w.Body.Bytes()
	if len(bodyBytes) < 4 || bodyBytes[0] != 'P' || bodyBytes[1] != 'K' {
		t.Fatalf("expected zip magic bytes, got prefix %v (len %d)", bodyBytes[:min(4, len(bodyBytes))], len(bodyBytes))
	}
}

func TestXlsx_Generate_RejectsEmptySheets(t *testing.T) {
	r := chi.NewRouter()
	NewXlsxHandler().Register(r)

	req := httptest.NewRequest(http.MethodPost, "/xlsx/generate", strings.NewReader(`{"sheets":[]}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestXlsx_RecommendCharts_ReturnsLineForDatedNumerics(t *testing.T) {
	r := chi.NewRouter()
	NewXlsxHandler().Register(r)

	body := map[string]any{
		"sheet": map[string]any{
			"name":    "Trend",
			"headers": []string{"Date", "Revenue"},
			"rows": [][]any{
				{"2026-01-01", 1000},
				{"2026-02-01", 1500},
			},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/xlsx/recommend-charts", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"type":"line"`) {
		t.Errorf("expected recommendation with type=line, got %s", w.Body.String())
	}
}

func TestSanitizeDownloadFilename(t *testing.T) {
	cases := map[string]string{
		"":              "workbook.xlsx",
		"demo":          "demo.xlsx",
		"demo.xlsx":     "demo.xlsx",
		"Q1 Report":     "Q1 Report.xlsx",
		"../escape.sh":  "..-escape.sh.xlsx", // / stripped, dots kept (safe in filename)
	}
	for in, want := range cases {
		if got := sanitizeDownloadFilename(in); got != want {
			t.Errorf("sanitizeDownloadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
