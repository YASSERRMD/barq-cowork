package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// XlsxWorkbook is the full declarative input to the builder. The LLM (or a
// thin tool wrapper) produces this struct; the builder does the rest —
// styling, widths, freeze pane, zebra stripes, charts.
type XlsxWorkbook struct {
	Title  string      `json:"title,omitempty"`
	Author string      `json:"author,omitempty"`
	Theme  XlsxTheme   `json:"theme,omitempty"`
	Sheets []XlsxSheet `json:"sheets"`
}

// XlsxSheet describes a single worksheet's data + layout preferences.
//
// ColumnOverrides lets the LLM pin a specific semantic role for a column
// (e.g. force "year" integers to render as ColumnText so Excel doesn't add
// thousand separators). An empty string means "let the builder infer".
//
// Formulas is a free-form "cell → formula" map for arbitrary spreadsheet
// math — "=SUM(B2:B10)", "=AVERAGE(C2:C10)", "=B2*C2", etc. Cells targeted
// here skip the value-coercion path; whatever excelize calculates wins.
//
// TotalsFormulas is an array aligned with Headers. Non-empty entries are
// rendered as formulas in the totals row instead of the literal values from
// Totals — so "=SUM(B2:B4)" in slot 1 becomes a real, recalculating total.
type XlsxSheet struct {
	Name             string            `json:"name"`
	Headers          []string          `json:"headers"`
	Rows             [][]any           `json:"rows"`
	ColumnOverrides  []ColumnKind      `json:"column_overrides,omitempty"`
	Totals           []any             `json:"totals,omitempty"`
	TotalsFormulas   []string          `json:"totals_formulas,omitempty"`
	Formulas         map[string]string `json:"formulas,omitempty"`
	Charts           []XlsxChart       `json:"charts,omitempty"` // populated in phase 3
	ConditionalRules []ConditionalRule `json:"conditional_rules,omitempty"`
}

// BuildWorkbook renders a workbook in-memory and returns the bytes. It is the
// single entry point for callers — do not instantiate excelize.File yourself
// elsewhere, so the styling/zebra/freeze conventions stay in one place.
func BuildWorkbook(wb XlsxWorkbook) ([]byte, error) {
	if len(wb.Sheets) == 0 {
		return nil, fmt.Errorf("workbook needs at least one sheet")
	}

	f := excelize.NewFile()
	defer f.Close()

	mapper := NewXlsxStyleMapper(f, wb.Theme)

	// excelize creates Sheet1 by default — rename it to the first declared
	// sheet so we don't leak the default name into user-facing workbooks.
	defaultName := f.GetSheetName(0)

	for i, s := range wb.Sheets {
		name := sanitizeSheetName(s.Name, i)
		if i == 0 {
			if err := f.SetSheetName(defaultName, name); err != nil {
				return nil, fmt.Errorf("rename sheet 0: %w", err)
			}
		} else {
			if _, err := f.NewSheet(name); err != nil {
				return nil, fmt.Errorf("new sheet %q: %w", name, err)
			}
		}
		if err := writeSheet(f, mapper, name, s); err != nil {
			return nil, fmt.Errorf("sheet %q: %w", name, err)
		}
	}

	f.SetActiveSheet(0)

	if wb.Title != "" || wb.Author != "" {
		_ = f.SetDocProps(&excelize.DocProperties{
			Title:   wb.Title,
			Creator: wb.Author,
		})
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}
	return buf.Bytes(), nil
}

// writeSheet populates a single worksheet: header row, data rows with
// alternating zebra stripes and per-column styles, optional totals row,
// per-column widths, frozen header pane, and auto-filter on the header.
func writeSheet(f *excelize.File, m *XlsxStyleMapper, sheet string, s XlsxSheet) error {
	if len(s.Headers) == 0 {
		return fmt.Errorf("headers are required")
	}

	// Resolve the kind of each column — caller override wins over inference.
	kinds := InferColumnKinds(s.Rows, len(s.Headers), 200)
	for i, over := range s.ColumnOverrides {
		if i >= len(kinds) {
			break
		}
		if over >= 0 {
			kinds[i] = over
		}
	}

	// Header row.
	headerID, err := m.ID(StyleHeader)
	if err != nil {
		return err
	}
	for c, h := range s.Headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return err
		}
	}
	if err := applyRowStyle(f, sheet, 1, len(s.Headers), headerID); err != nil {
		return err
	}

	// Data rows.
	for r, row := range s.Rows {
		excelRow := r + 2 // 1-based; header is row 1
		zebra := r%2 == 1
		for c := 0; c < len(s.Headers); c++ {
			var v any
			if c < len(row) {
				v = row[c]
			}
			cell, _ := excelize.CoordinatesToCellName(c+1, excelRow)
			if err := f.SetCellValue(sheet, cell, coerceValue(v, kinds[c])); err != nil {
				return err
			}
			styleID, err := m.ID(bodyStyleFor(kinds[c], zebra))
			if err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, cell, cell, styleID); err != nil {
				return err
			}
		}
	}

	// Optional totals row. TotalsFormulas beats Totals on a per-column basis,
	// so "=SUM(B2:B4)" in slot 1 renders as a live formula while slot 0 still
	// says the literal "Total".
	if len(s.Totals) > 0 || len(s.TotalsFormulas) > 0 {
		totalsRow := len(s.Rows) + 2
		totalsID, err := m.ID(StyleTotals)
		if err != nil {
			return err
		}
		for c := 0; c < len(s.Headers); c++ {
			cell, _ := excelize.CoordinatesToCellName(c+1, totalsRow)
			if c < len(s.TotalsFormulas) && s.TotalsFormulas[c] != "" {
				if looksLikeFormula(s.TotalsFormulas[c]) {
					if err := f.SetCellFormula(sheet, cell, normaliseFormula(s.TotalsFormulas[c])); err != nil {
						return fmt.Errorf("totals formula %s: %w", cell, err)
					}
				} else if err := f.SetCellValue(sheet, cell, s.TotalsFormulas[c]); err != nil {
					return err
				}
			} else if c < len(s.Totals) && s.Totals[c] != nil {
				if err := f.SetCellValue(sheet, cell, coerceValue(s.Totals[c], kinds[c])); err != nil {
					return err
				}
			} else {
				continue // skip styling for cells we didn't touch
			}
			if err := f.SetCellStyle(sheet, cell, cell, totalsID); err != nil {
				return err
			}
		}
	}

	// Cell-level formulas take priority over any value that was laid down at
	// the same coordinate. Styling on those cells is left as-is (the body /
	// zebra style picked by column kind still applies).
	for cell, formula := range s.Formulas {
		if strings.TrimSpace(formula) == "" {
			continue
		}
		if err := f.SetCellFormula(sheet, cell, normaliseFormula(formula)); err != nil {
			return fmt.Errorf("formula %s: %w", cell, err)
		}
	}

	// Column widths.
	widths := ComputeColumnWidths(s.Headers, s.Rows, 200)
	for c, w := range widths {
		colLetter, _ := excelize.ColumnNumberToName(c + 1)
		if err := f.SetColWidth(sheet, colLetter, colLetter, w); err != nil {
			return err
		}
	}

	// Freeze the header row so headers stay visible as the user scrolls.
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return err
	}

	// Auto-filter across the full header range.
	lastCell, _ := excelize.CoordinatesToCellName(len(s.Headers), 1)
	if err := f.AutoFilter(sheet, "A1:"+lastCell, nil); err != nil {
		return err
	}

	// Conditional formats attach to the data region; must run after values +
	// formulas are in place so ranges like B2:B5 resolve to real cells.
	if err := applyConditionalRules(f, sheet, s, m); err != nil {
		return err
	}

	// Charts render last so they can reference the final data region.
	if err := applyCharts(f, sheet, s); err != nil {
		return err
	}

	return nil
}

// applyRowStyle applies one style to every cell in a single row, inclusive.
func applyRowStyle(f *excelize.File, sheet string, row, cols, styleID int) error {
	left, _ := excelize.CoordinatesToCellName(1, row)
	right, _ := excelize.CoordinatesToCellName(cols, row)
	return f.SetCellStyle(sheet, left, right, styleID)
}

// bodyStyleFor picks the right StyleRole for a (kind, zebra) combo. Non-zebra
// rows use the plain body/number variants; zebra rows use the tinted ones so
// alternating stripes are visible without any caller configuration.
func bodyStyleFor(kind ColumnKind, zebra bool) StyleRole {
	switch kind {
	case ColumnInteger, ColumnFloat:
		if zebra {
			return StyleZebraNumber
		}
		return StyleBodyNumber
	case ColumnCurrency:
		return StyleBodyCurrency
	case ColumnPercent:
		return StyleBodyPercent
	case ColumnDate:
		return StyleBodyDate
	}
	if zebra {
		return StyleZebra
	}
	return StyleBody
}

// coerceValue converts loose any-values into the form excelize stores best for
// a given kind. Numeric text is parsed so Excel formats it natively; dates
// become time.Time so the numfmt renders correctly. Anything unrecognised is
// passed through as-is.
func coerceValue(v any, kind ColumnKind) any {
	if v == nil {
		return ""
	}
	s := stringify(v)
	switch kind {
	case ColumnInteger:
		if n, ok := parseInteger(v, s); ok {
			return n
		}
	case ColumnFloat, ColumnCurrency:
		if f, ok := parseFloat(v, strings.TrimSpace(stripCurrencyChrome(s))); ok {
			return f
		}
	case ColumnPercent:
		trim := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
		if f, ok := parseFloat(v, trim); ok {
			return f / 100.0
		}
	case ColumnDate:
		if t, ok := parseISODate(s); ok {
			return t
		}
	}
	return v
}

// stripCurrencyChrome removes recognised currency symbols / ISO codes and
// grouping commas from s so parseFloat can accept the bare number. It is the
// mirror of the looksLikeCurrency classifier.
func stripCurrencyChrome(s string) string {
	out := strings.TrimSpace(s)
	for _, sym := range currencySymbols {
		out = strings.TrimPrefix(out, sym)
	}
	upper := strings.ToUpper(out)
	for _, code := range currencyCodes {
		if strings.HasSuffix(upper, code) {
			out = strings.TrimSpace(out[:len(out)-len(code)])
			break
		}
	}
	out = strings.ReplaceAll(out, ",", "")
	return strings.TrimSpace(out)
}

// normaliseFormula strips a leading '=' if present — excelize's
// SetCellFormula expects the bare expression ("SUM(B2:B4)"), but the LLM (and
// humans) naturally write "=SUM(B2:B4)". Tolerating both keeps the API kind
// to callers who forget which convention this library uses.
func normaliseFormula(f string) string {
	f = strings.TrimSpace(f)
	return strings.TrimPrefix(f, "=")
}

func looksLikeFormula(f string) bool {
	trimmed := strings.TrimSpace(f)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "=") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "=")) != ""
	}
	// Literal labels such as "TOTAL" and "GRAND TOTAL" are common in LLM output.
	// Only route strings through SetCellFormula when they contain formula syntax.
	return strings.ContainsAny(trimmed, "()+-*/^&=<>!:,$")
}

// sanitizeSheetName enforces Excel's sheet-name constraints: non-empty, at
// most 31 chars, no []:*?/\ characters. Falls back to "Sheet{N}" when blank.
func sanitizeSheetName(name string, index int) string {
	forbidden := []string{"[", "]", ":", "*", "?", "/", "\\"}
	out := strings.TrimSpace(name)
	for _, f := range forbidden {
		out = strings.ReplaceAll(out, f, "-")
	}
	if out == "" {
		out = fmt.Sprintf("Sheet%d", index+1)
	}
	if len(out) > 31 {
		out = out[:31]
	}
	return out
}

// XlsxChart is declared here (as an empty placeholder) so the sheet builder
// can reference it without depending on a later phase's file. Phase 3 fills
// in the real fields and the AddChart logic.
type XlsxChart struct {
	Title         string   `json:"title,omitempty"`
	Type          string   `json:"type,omitempty"`
	CategoriesCol int      `json:"categories_col,omitempty"`
	SeriesCols    []int    `json:"series_cols,omitempty"`
	AnchorCell    string   `json:"anchor_cell,omitempty"`
	SeriesNames   []string `json:"series_names,omitempty"`
}
