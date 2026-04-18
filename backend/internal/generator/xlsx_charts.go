package generator

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// chartTypeAliases maps the lowercase chart name the LLM (or the caller) uses
// in JSON onto the excelize ChartType constant. We keep the set small and
// presentation-focused — obscure variants like Bar3DConePercentStacked are
// reachable via the raw Go API but excluded here to keep the LLM-facing
// surface simple and self-documenting.
var chartTypeAliases = map[string]excelize.ChartType{
	"bar":          excelize.Bar,
	"bar_stacked":  excelize.BarStacked,
	"bar3d":        excelize.Bar3DClustered,
	"column":       excelize.Col,
	"col":          excelize.Col,
	"col_stacked":  excelize.ColStacked,
	"column3d":     excelize.Col3D,
	"col3d":        excelize.Col3D,
	"line":         excelize.Line,
	"line3d":       excelize.Line3D,
	"pie":          excelize.Pie,
	"pie3d":        excelize.Pie3D,
	"doughnut":     excelize.Doughnut,
	"area":         excelize.Area,
	"area_stacked": excelize.AreaStacked,
	"area3d":       excelize.Area3D,
	"scatter":      excelize.Scatter,
	"radar":        excelize.Radar,
}

// resolveChartType returns the excelize constant for a caller-supplied name,
// tolerating case and whitespace variance. Falls back to Col (the most
// universally readable shape) when the name is unrecognised — we prefer a
// chart over a hard error since charts are an enhancement, not a contract.
func resolveChartType(name string) (excelize.ChartType, string) {
	key := strings.ToLower(strings.TrimSpace(name))
	if ct, ok := chartTypeAliases[key]; ok {
		return ct, key
	}
	return excelize.Col, "col"
}

// applyCharts renders every chart declared on the sheet. It reads from the
// already-laid-out data, so callers MUST invoke this AFTER writing headers +
// rows (the builder guarantees this ordering).
func applyCharts(f *excelize.File, sheet string, s XlsxSheet) error {
	for i, c := range s.Charts {
		if err := applyOneChart(f, sheet, s, c); err != nil {
			return fmt.Errorf("chart %d (%q): %w", i, c.Title, err)
		}
	}
	return nil
}

func applyOneChart(f *excelize.File, sheet string, s XlsxSheet, c XlsxChart) error {
	if len(s.Rows) == 0 {
		return fmt.Errorf("no rows to chart")
	}
	if len(c.SeriesCols) == 0 {
		return fmt.Errorf("at least one series column is required")
	}

	ct, resolved := resolveChartType(c.Type)

	// Categories come from c.CategoriesCol — the x-axis labels (dates, names).
	catRange, err := cellRangeForColumn(sheet, c.CategoriesCol, 2, len(s.Rows)+1)
	if err != nil {
		return err
	}

	series := make([]excelize.ChartSeries, 0, len(c.SeriesCols))
	for i, col := range c.SeriesCols {
		valRange, err := cellRangeForColumn(sheet, col, 2, len(s.Rows)+1)
		if err != nil {
			return err
		}
		name := ""
		if i < len(c.SeriesNames) && c.SeriesNames[i] != "" {
			name = c.SeriesNames[i]
		} else if col >= 0 && col < len(s.Headers) {
			name = s.Headers[col]
		}
		series = append(series, excelize.ChartSeries{
			Name:       name,
			Categories: catRange,
			Values:     valRange,
		})
	}

	anchor := strings.TrimSpace(c.AnchorCell)
	if anchor == "" {
		anchor = defaultChartAnchor(s)
	}

	chart := &excelize.Chart{
		Type:      ct,
		Series:    series,
		Dimension: excelize.ChartDimension{Width: 640, Height: 360},
		Title:     []excelize.RichTextRun{{Text: c.Title}},
		Legend:    excelize.ChartLegend{Position: "bottom"},
		PlotArea: excelize.ChartPlotArea{
			ShowCatName:  false,
			ShowSerName:  true,
			ShowVal:      false,
			ShowPercent:  ct == excelize.Pie || ct == excelize.Pie3D || ct == excelize.Doughnut,
		},
	}
	_ = resolved // retained for future telemetry / tooling
	return f.AddChart(sheet, anchor, chart)
}

// cellRangeForColumn builds a sheet-prefixed absolute range like
// "Sheet1!$B$2:$B$10" for a 0-based column index and inclusive 1-based row
// bounds. Returns an error if col is out of range.
func cellRangeForColumn(sheet string, col, startRow, endRow int) (string, error) {
	if col < 0 {
		return "", fmt.Errorf("column index %d is negative", col)
	}
	startCell, err := excelize.CoordinatesToCellName(col+1, startRow, true)
	if err != nil {
		return "", err
	}
	endCell, err := excelize.CoordinatesToCellName(col+1, endRow, true)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s!%s:%s", sheetRef(sheet), startCell, endCell), nil
}

// sheetRef quotes sheet names that contain spaces or punctuation. Excel's
// range syntax wraps such names in single quotes.
func sheetRef(sheet string) string {
	if strings.ContainsAny(sheet, " '-") {
		return "'" + strings.ReplaceAll(sheet, "'", "''") + "'"
	}
	return sheet
}

// defaultChartAnchor places the chart two columns to the right of the data
// region, one row below the header — close enough to read together, far enough
// not to overlap the last column on narrow sheets.
func defaultChartAnchor(s XlsxSheet) string {
	col := len(s.Headers) + 2
	cell, _ := excelize.CoordinatesToCellName(col, 2)
	return cell
}

// RecommendCharts surveys a sheet and returns a single-entry chart spec
// suitable for a quick default visualisation. Strategy:
//   - If there is exactly one ColumnDate / category-like text column, use it
//     as categories and every numeric column as a series → line chart.
//   - Otherwise pick the first text column as categories and the first numeric
//     column as the only series → column chart.
//   - If the sheet has no usable combination, return nil.
//
// This is a suggestion, not a mandate: the LLM can accept, tweak, or replace
// the returned spec before passing it into BuildWorkbook.
func RecommendCharts(s XlsxSheet) []XlsxChart {
	kinds := InferColumnKinds(s.Rows, len(s.Headers), 200)

	catCol := -1
	dateCol := -1
	numCols := []int{}
	for i, k := range kinds {
		switch k {
		case ColumnDate:
			if dateCol == -1 {
				dateCol = i
			}
		case ColumnText:
			if catCol == -1 {
				catCol = i
			}
		case ColumnInteger, ColumnFloat, ColumnCurrency, ColumnPercent:
			numCols = append(numCols, i)
		}
	}

	chosenCat := dateCol
	chartType := "line"
	if chosenCat == -1 {
		chosenCat = catCol
		chartType = "column"
	}
	if chosenCat == -1 || len(numCols) == 0 {
		return nil
	}

	title := s.Name
	if title == "" {
		title = "Summary"
	}
	return []XlsxChart{{
		Title:         title,
		Type:          chartType,
		CategoriesCol: chosenCat,
		SeriesCols:    numCols,
	}}
}
