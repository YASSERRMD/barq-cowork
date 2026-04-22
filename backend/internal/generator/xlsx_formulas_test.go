package generator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestBuildWorkbook_CellFormulas writes a sheet with explicit per-cell
// formulas AND totals-as-formulas, then reopens the workbook and calculates
// each cell. Tests prove:
//   - =SUM(B2:B4) in the totals row evaluates to the data sum.
//   - A per-cell formula (=B2*C2) in the body survives.
//   - Both leading-'=' and bare-expression inputs are accepted.
func TestBuildWorkbook_CellFormulas(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Budget",
			Headers: []string{"Line", "Unit cost", "Units", "Extended"},
			Rows: [][]any{
				{"Widgets", 10.0, 5, nil},
				{"Gadgets", 25.0, 3, nil},
				{"Gizmos", 7.5, 8, nil},
			},
			// Extended column (D) uses row-local formulas.
			Formulas: map[string]string{
				"D2": "=B2*C2", // leading '=' accepted
				"D3": "B3*C3",  // bare form accepted
				"D4": "=B4*C4",
			},
			// Totals: "Totals" literal in A, live formulas elsewhere.
			Totals:         []any{"Totals", nil, nil, nil},
			TotalsFormulas: []string{"", "=SUM(B2:B4)", "=SUM(C2:C4)", "=SUM(D2:D4)"},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer f.Close()

	// Per-row extended values.
	cases := map[string]string{
		"D2": "50",   // 10 * 5
		"D3": "75",   // 25 * 3
		"D4": "60",   // 7.5 * 8
		"B5": "42.5", // 10 + 25 + 7.5
		"C5": "16",   // 5 + 3 + 8
		"D5": "185",  // 50 + 75 + 60
	}
	for cell, want := range cases {
		got, err := f.CalcCellValue("Budget", cell)
		if err != nil {
			t.Errorf("CalcCellValue %s: %v", cell, err)
			continue
		}
		if got != want {
			t.Errorf("%s: got %q, want %q", cell, got, want)
		}
	}

	// Literal totals-slot should still render the text.
	gotA5, _ := f.GetCellValue("Budget", "A5")
	if gotA5 != "Totals" {
		t.Errorf("A5 (literal): got %q, want 'Totals'", gotA5)
	}
}

func TestNormaliseFormula(t *testing.T) {
	cases := map[string]string{
		"=SUM(A1:A3)":    "SUM(A1:A3)",
		"SUM(A1:A3)":     "SUM(A1:A3)",
		"  =AVG(B1:B3) ": "AVG(B1:B3)",
		"":               "",
	}
	for in, want := range cases {
		if got := normaliseFormula(in); got != want {
			t.Errorf("normaliseFormula(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildWorkbook_TotalsFormulaLabelsStayLiteral(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Budget",
			Headers: []string{"Line", "Amount"},
			Rows: [][]any{
				{"Rent", 1000},
				{"Food", 250},
			},
			TotalsFormulas: []string{"GRAND TOTAL", "=SUM(B2:B3)"},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer f.Close()

	got, err := f.GetCellValue("Budget", "A4")
	if err != nil {
		t.Fatalf("GetCellValue A4: %v", err)
	}
	if got != "GRAND TOTAL" {
		t.Fatalf("A4: got %q, want GRAND TOTAL", got)
	}
	if formula, err := f.GetCellFormula("Budget", "A4"); err != nil || formula != "" {
		t.Fatalf("A4 should not contain a formula, got formula=%q err=%v", formula, err)
	}
	if formula, err := f.GetCellFormula("Budget", "B4"); err != nil || formula != "SUM(B2:B3)" {
		t.Fatalf("B4 formula: got %q err=%v", formula, err)
	}

	raw := readSheetXML(t, data, "xl/worksheets/sheet1.xml")
	if strings.Contains(raw, "<f>GRAND TOTAL</f>") {
		t.Fatal("literal totals label was written as a formula")
	}
}

// TestBuildWorkbook_FormulaEmptyIgnored ensures an empty-string formula in the
// map is skipped rather than crashing excelize.
func TestBuildWorkbook_FormulaEmptyIgnored(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "X",
			Headers: []string{"a"},
			Rows:    [][]any{{1}},
			Formulas: map[string]string{
				"A1": "", // no-op
				"A2": "  ",
			},
		}},
	}
	if _, err := BuildWorkbook(wb); err != nil {
		t.Fatalf("empty-formula entries should be ignored: %v", err)
	}
}
