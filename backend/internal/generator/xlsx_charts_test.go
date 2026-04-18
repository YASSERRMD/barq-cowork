package generator

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestResolveChartType_FallsBackToCol(t *testing.T) {
	ct, resolved := resolveChartType("totally-unknown-shape")
	if ct != excelize.Col {
		t.Errorf("want Col fallback, got %v", ct)
	}
	if resolved != "col" {
		t.Errorf("want resolved=col, got %q", resolved)
	}
}

func TestResolveChartType_AliasesCaseInsensitive(t *testing.T) {
	ct, _ := resolveChartType("  LINE3D  ")
	if ct != excelize.Line3D {
		t.Errorf("want Line3D, got %v", ct)
	}
}

func TestCellRangeForColumn(t *testing.T) {
	r, err := cellRangeForColumn("Sales", 1, 2, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r != "Sales!$B$2:$B$10" {
		t.Errorf("got %q", r)
	}
	// Spaces in the sheet name should force single-quote wrapping.
	r2, _ := cellRangeForColumn("Q1 Sales", 0, 2, 5)
	if !strings.HasPrefix(r2, "'Q1 Sales'!") {
		t.Errorf("expected quoted sheet ref, got %q", r2)
	}
}

func TestRecommendCharts_PicksDateAndNumerics(t *testing.T) {
	s := XlsxSheet{
		Name:    "Trend",
		Headers: []string{"Date", "Region", "Revenue", "Units"},
		Rows: [][]any{
			{"2026-01-01", "EMEA", 1000, 42},
			{"2026-02-01", "APAC", 1500, 51},
			{"2026-03-01", "AMER", 2200, 60},
		},
	}
	got := RecommendCharts(s)
	if len(got) != 1 {
		t.Fatalf("want 1 chart, got %d", len(got))
	}
	if got[0].CategoriesCol != 0 {
		t.Errorf("want date col=0, got %d", got[0].CategoriesCol)
	}
	if got[0].Type != "line" {
		t.Errorf("want type=line, got %q", got[0].Type)
	}
	if len(got[0].SeriesCols) != 2 {
		t.Errorf("want 2 numeric series, got %d", len(got[0].SeriesCols))
	}
}

func TestRecommendCharts_NoNumericsReturnsNil(t *testing.T) {
	s := XlsxSheet{
		Name:    "Labels",
		Headers: []string{"Name", "Tag"},
		Rows:    [][]any{{"a", "x"}, {"b", "y"}},
	}
	if got := RecommendCharts(s); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

// TestBuildWorkbook_WithChart emits a workbook with an explicit chart spec and
// asserts that the resulting .xlsx zip contains a chart XML part. Excelize
// writes charts under xl/charts/; if AddChart silently no-ops, this test
// fails.
func TestBuildWorkbook_WithChart(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Trend",
			Headers: []string{"Date", "Revenue"},
			Rows: [][]any{
				{"2026-01-01", 1000},
				{"2026-02-01", 1500},
				{"2026-03-01", 2200},
			},
			Charts: []XlsxChart{{
				Title:         "Monthly revenue",
				Type:          "line",
				CategoriesCol: 0,
				SeriesCols:    []int{1},
			}},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}
	sawChart := false
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/charts/chart") {
			sawChart = true
			break
		}
	}
	if !sawChart {
		t.Error("expected a chart part under xl/charts/, found none")
	}
}
