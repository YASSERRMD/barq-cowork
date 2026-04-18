package generator

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// readSheetXML pulls the raw xl/worksheets/sheet1.xml out of an .xlsx zip for
// string-level assertions. We use text inspection instead of a second
// excelize round-trip because the library's Get* helpers for conditional
// formats lag behind the generator features we actually emit.
func readSheetXML(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open %s: %v", name, err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			return string(b)
		}
	}
	t.Fatalf("%s not found in xlsx", name)
	return ""
}

func TestConditional_NumericCellRules(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Sales",
			Headers: []string{"Rep", "Revenue"},
			Rows: [][]any{
				{"Alice", 1200},
				{"Bob", 800},
				{"Carol", 2400},
			},
			ConditionalRules: []ConditionalRule{
				{Range: "B2:B4", Type: "greater_than", Value: "1500", FillColor: "C6EFCE", TextColor: "006100"},
				{Range: "B2:B4", Type: "less_than", Value: "1000"},
				{Range: "B2:B4", Type: "between", Value: "1000", Value2: "1500"},
			},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}
	xml := readSheetXML(t, data, "xl/worksheets/sheet1.xml")
	if !strings.Contains(xml, "<conditionalFormatting") {
		t.Fatal("no <conditionalFormatting> element in sheet xml")
	}
	for _, op := range []string{"greaterThan", "lessThan", "between"} {
		if !strings.Contains(xml, op) {
			t.Errorf("expected operator %q in sheet xml", op)
		}
	}
}

func TestConditional_ColorScalesAndDataBar(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Heat",
			Headers: []string{"Bucket", "Value"},
			Rows: [][]any{
				{"A", 10}, {"B", 20}, {"C", 30}, {"D", 40}, {"E", 50},
			},
			ConditionalRules: []ConditionalRule{
				{Range: "B2:B6", Type: "color_scale_3"},
				{Range: "B2:B6", Type: "data_bar", BarColor: "638EC6"},
			},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}
	xml := readSheetXML(t, data, "xl/worksheets/sheet1.xml")
	if !strings.Contains(xml, "colorScale") {
		t.Error("expected colorScale element")
	}
	if !strings.Contains(xml, "dataBar") {
		t.Error("expected dataBar element")
	}
}

func TestConditional_TextAndAggregate(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:    "Ops",
			Headers: []string{"Status", "Score"},
			Rows: [][]any{
				{"OPEN", 10}, {"WIP", 20}, {"DONE", 30},
			},
			ConditionalRules: []ConditionalRule{
				{Range: "A2:A4", Type: "text_contains", Value: "WIP"},
				{Range: "B2:B4", Type: "top_n", Value: "1"},
				{Range: "B2:B4", Type: "above_average"},
				{Range: "A2:A4", Type: "duplicate"},
			},
		}},
	}
	data, err := BuildWorkbook(wb)
	if err != nil {
		t.Fatalf("BuildWorkbook: %v", err)
	}
	xml := readSheetXML(t, data, "xl/worksheets/sheet1.xml")
	for _, frag := range []string{"containsText", "top10", "aboveAverage", "duplicateValues"} {
		if !strings.Contains(xml, frag) {
			t.Errorf("expected %q in sheet xml", frag)
		}
	}
}

func TestConditional_UnknownTypeErrors(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:             "X",
			Headers:          []string{"a"},
			Rows:             [][]any{{1}},
			ConditionalRules: []ConditionalRule{{Range: "A1:A1", Type: "totally-made-up"}},
		}},
	}
	if _, err := BuildWorkbook(wb); err == nil {
		t.Fatal("expected error for unknown rule type")
	}
}

func TestConditional_MissingRangeErrors(t *testing.T) {
	wb := XlsxWorkbook{
		Sheets: []XlsxSheet{{
			Name:             "X",
			Headers:          []string{"a"},
			Rows:             [][]any{{1}},
			ConditionalRules: []ConditionalRule{{Type: "greater_than", Value: "5"}},
		}},
	}
	if _, err := BuildWorkbook(wb); err == nil {
		t.Fatal("expected error for empty range")
	}
}
