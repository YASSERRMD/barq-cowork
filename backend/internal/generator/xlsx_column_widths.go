package generator

import "unicode/utf8"

// Column-width constants calibrated against Excel's Calibri 11 rendering.
// `excelize.SetColWidth` uses "number of characters at the default font", so
// these are roughly character counts with a small padding built in.
const (
	minColumnWidth = 8.0
	maxColumnWidth = 60.0
	widthPadding   = 2.0
)

// ComputeColumnWidths returns one width (in Excel character units) per column,
// sized so the widest header / cell text fits — clamped between min and max so
// pathological single-cell novels don't blow the page layout. Multibyte
// characters are counted by rune, not by byte.
func ComputeColumnWidths(headers []string, rows [][]any, sampleLimit int) []float64 {
	cols := len(headers)
	widths := make([]float64, cols)
	for c := 0; c < cols; c++ {
		widths[c] = float64(utf8.RuneCountInString(headers[c]))
	}

	scanned := 0
	for _, row := range rows {
		if scanned >= sampleLimit {
			break
		}
		scanned++
		for c, v := range row {
			if c >= cols {
				break
			}
			if rc := float64(utf8.RuneCountInString(stringify(v))); rc > widths[c] {
				widths[c] = rc
			}
		}
	}

	for c := range widths {
		widths[c] = clampWidth(widths[c] + widthPadding)
	}
	return widths
}

func clampWidth(w float64) float64 {
	if w < minColumnWidth {
		return minColumnWidth
	}
	if w > maxColumnWidth {
		return maxColumnWidth
	}
	return w
}
