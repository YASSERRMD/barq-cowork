package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/barq-cowork/barq-cowork/internal/generator"
)

// WriteXlsxTool renders a styled Excel (.xlsx) workbook from an LLM-supplied
// JSON spec — data, theme colors, and optional chart descriptions. It is a
// thin wrapper around generator.BuildWorkbook; all rendering rules (auto
// widths, zebra stripes, freeze header, chart placement) live in the
// generator package so the REST endpoint can share them.
type WriteXlsxTool struct{}

func (WriteXlsxTool) Name() string { return "write_xlsx" }

func (WriteXlsxTool) Description() string {
	return "Generate a styled Excel workbook (.xlsx) from data rows, a theme, and optional chart " +
		"descriptions. The renderer auto-sizes columns, alternates zebra stripes, freezes the " +
		"header row, and adds auto-filter. Supply `charts` to inject bar/column/line/pie/3D " +
		"charts positioned next to the data. Design the theme around the subject — don't reuse " +
		"a template. Use this tool for any tabular deliverable the user asks to be in Excel, " +
		"not CSV."
}

func (WriteXlsxTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Output filename without extension, e.g. 'q1-sales'",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Workbook title used in document metadata (optional)",
			},
			"author": map[string]any{
				"type":        "string",
				"description": "Creator stored in document properties (optional)",
			},
			"theme": map[string]any{
				"type":        "object",
				"description": "Visual theme. Colors are 6-digit hex without '#'. All fields optional; blanks use neutral defaults. Design around the subject — muted palette for finance, warmer for marketing, etc.",
				"properties": map[string]any{
					"name":               map[string]any{"type": "string"},
					"heading_font":       map[string]any{"type": "string"},
					"body_font":          map[string]any{"type": "string"},
					"mono_font":          map[string]any{"type": "string"},
					"header_fill_color":  map[string]any{"type": "string"},
					"header_text_color":  map[string]any{"type": "string"},
					"body_text_color":    map[string]any{"type": "string"},
					"zebra_fill_color":   map[string]any{"type": "string"},
					"accent_color":       map[string]any{"type": "string"},
					"border_color":       map[string]any{"type": "string"},
					"totals_fill_color":  map[string]any{"type": "string"},
					"totals_text_color":  map[string]any{"type": "string"},
					"negative_color":     map[string]any{"type": "string"},
					"positive_color":     map[string]any{"type": "string"},
				},
			},
			"sheets": map[string]any{
				"type":        "array",
				"description": "One or more worksheets rendered in order. The first becomes the active sheet.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":    map[string]any{"type": "string", "description": "Tab name (sanitised to Excel's 31-char rules)"},
						"headers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"rows": map[string]any{
							"type":        "array",
							"description": "Data rows — each row is an array aligned with headers.",
							"items":       map[string]any{"type": "array"},
						},
						"column_overrides": map[string]any{
							"type":        "array",
							"description": "Optional: force each column's kind. 0=text 1=integer 2=float 3=currency 4=percent 5=date. Omit a value or pass -1 to auto-infer.",
							"items":       map[string]any{"type": "integer"},
						},
						"totals": map[string]any{
							"type":        "array",
							"description": "Optional totals row. Array aligned with headers; use null for skipped columns.",
						},
						"totals_formulas": map[string]any{
							"type":        "array",
							"description": "Optional: array aligned with headers. Non-empty entries render as live formulas in the totals row, overriding the matching 'totals' slot. Example: ['', '=SUM(B2:B10)', '=AVERAGE(C2:C10)']. Leading '=' is optional.",
							"items":       map[string]any{"type": "string"},
						},
						"formulas": map[string]any{
							"type":                 "object",
							"description":          "Optional: cell-address -> formula map for arbitrary spreadsheet math. Example: {\"D2\": \"=B2*C2\", \"E10\": \"=AVERAGE(E2:E9)\"}. Evaluated by Excel when the workbook opens; leading '=' optional. Use for derived columns, cross-sheet refs, IF/VLOOKUP, etc.",
							"additionalProperties": map[string]any{"type": "string"},
						},
						"conditional_rules": map[string]any{
							"type":        "array",
							"description": "Optional conditional-format rules applied after data layout. Each rule targets a sheet-relative range (e.g. 'B2:B20'). Rule types: greater_than, less_than, equal_to, not_equal_to, greater_or_equal, less_or_equal, between, not_between (value/value2 numeric); top_n, bottom_n (value=count, percent=true for %); above_average, below_average; duplicate, unique; blanks, no_blanks, errors, no_errors; text_contains, text_not_contains, text_begins_with, text_ends_with (value=needle); color_scale_2, color_scale_3 (min_color/mid_color/max_color hex); data_bar (bar_color hex). fill_color / text_color / bold style matched cells on the cell/text/top/average/duplicate/unique/blank/error families.",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"range":      map[string]any{"type": "string", "description": "Cell range like 'B2:B20'"},
									"type":       map[string]any{"type": "string"},
									"value":      map[string]any{"type": "string"},
									"value2":     map[string]any{"type": "string"},
									"fill_color": map[string]any{"type": "string"},
									"text_color": map[string]any{"type": "string"},
									"bold":       map[string]any{"type": "boolean"},
									"min_color":  map[string]any{"type": "string"},
									"mid_color":  map[string]any{"type": "string"},
									"max_color":  map[string]any{"type": "string"},
									"bar_color":  map[string]any{"type": "string"},
									"percent":    map[string]any{"type": "boolean"},
								},
								"required": []string{"range", "type"},
							},
						},
						"charts": map[string]any{
							"type":        "array",
							"description": "Optional chart descriptors; rendered after data is laid out.",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"title":          map[string]any{"type": "string"},
									"type":           map[string]any{"type": "string", "description": "One of: bar, bar_stacked, bar3d, column/col, col_stacked, column3d, line, line3d, pie, pie3d, doughnut, area, area_stacked, area3d, scatter, radar. Unknown names fall back to column."},
									"categories_col": map[string]any{"type": "integer", "description": "0-based column index supplying x-axis labels (dates or names)"},
									"series_cols":   map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "0-based column indices, one per series"},
									"series_names": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional per-series legend labels; falls back to header text"},
									"anchor_cell":  map[string]any{"type": "string", "description": "Top-left cell for the chart, e.g. 'G2'. Omit to auto-place right of the data."},
								},
								"required": []string{"type", "series_cols"},
							},
						},
					},
					"required": []string{"name", "headers", "rows"},
				},
			},
		},
		"required": []string{"filename", "sheets"},
	}
}

type writeXlsxArgs struct {
	Filename string                 `json:"filename"`
	Title    string                 `json:"title"`
	Author   string                 `json:"author"`
	Theme    generator.XlsxTheme    `json:"theme"`
	Sheets   []generator.XlsxSheet  `json:"sheets"`
}

func (t WriteXlsxTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args writeXlsxArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	if len(args.Sheets) == 0 {
		return Err("at least one sheet is required")
	}

	fname := sanitizeFilename(args.Filename, ".xlsx")
	relPath := filepath.Join("spreadsheets", fname+".xlsx")
	absPath, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Err("create spreadsheets directory: %v", err)
	}

	data, err := generator.BuildWorkbook(generator.XlsxWorkbook{
		Title:  args.Title,
		Author: args.Author,
		Theme:  args.Theme,
		Sheets: args.Sheets,
	})
	if err != nil {
		return Err("xlsx generation failed: %v", err)
	}
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return Err("write xlsx: %v", err)
	}

	return OKData(
		fmt.Sprintf("Excel workbook written to %s (%d bytes)", relPath, len(data)),
		map[string]any{"path": relPath, "size": int64(len(data))},
	)
}
