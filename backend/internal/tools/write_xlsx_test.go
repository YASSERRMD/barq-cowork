package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteXlsxTool_EndToEnd(t *testing.T) {
	dir := t.TempDir()

	args := map[string]any{
		"filename": "q1-sales",
		"title":    "Q1 Sales",
		"author":   "barq",
		"theme": map[string]any{
			"header_fill_color": "BE123C",
			"accent_color":      "BE123C",
		},
		"sheets": []map[string]any{{
			"name":    "Revenue",
			"headers": []string{"Region", "Revenue", "Closed"},
			"rows": [][]any{
				{"EMEA", 12340.5, "2026-01-15"},
				{"APAC", 9820.75, "2026-02-02"},
				{"AMER", 15500.0, "2026-03-11"},
			},
			"totals": []any{"Total", 37661.25, nil},
			"charts": []map[string]any{{
				"title":          "Revenue by region",
				"type":           "column",
				"categories_col": 0,
				"series_cols":    []int{1},
			}},
		}},
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}

	res := WriteXlsxTool{}.Execute(
		context.Background(),
		InvocationContext{WorkspaceRoot: dir},
		string(argsJSON),
	)
	if res.Status != ResultOK {
		t.Fatalf("tool failed: %s", res.Error)
	}

	out := filepath.Join(dir, "spreadsheets", "q1-sales.xlsx")
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() < 1000 {
		t.Fatalf("output suspiciously small: %d bytes", info.Size())
	}
}

func TestWriteXlsxTool_SanitizesFilename(t *testing.T) {
	// ../escape should be neutralised into a hyphenated in-workspace name,
	// never allowed to traverse out.
	dir := t.TempDir()
	args := `{"filename":"../escape","sheets":[{"name":"a","headers":["h"],"rows":[["v"]]}]}`
	res := WriteXlsxTool{}.Execute(
		context.Background(),
		InvocationContext{WorkspaceRoot: dir},
		args,
	)
	if res.Status != ResultOK {
		t.Fatalf("expected sanitised write to succeed, got %q", res.Error)
	}
	// Nothing should exist outside the workspace.
	if _, err := os.Stat(filepath.Join(dir, "..", "escape.xlsx")); err == nil {
		t.Fatal("output escaped the workspace root")
	}
}

func TestWriteXlsxTool_ValidatesRequired(t *testing.T) {
	res := WriteXlsxTool{}.Execute(
		context.Background(),
		InvocationContext{WorkspaceRoot: t.TempDir()},
		`{"filename":"x"}`,
	)
	if res.Status == ResultOK {
		t.Fatal("expected error for missing sheets")
	}
}
