package generator

import (
	"strconv"
	"strings"
	"time"
)

// ColumnKind is the inferred data shape of a single column. The sheet builder
// uses it to pick a body / zebra style pair (right-align numbers, format
// percents, parse ISO dates, …) without the LLM having to specify one.
type ColumnKind int

const (
	ColumnText ColumnKind = iota
	ColumnInteger
	ColumnFloat
	ColumnCurrency
	ColumnPercent
	ColumnDate
)

// InferColumnKinds looks at up to `sampleLimit` rows per column and returns the
// best-fit ColumnKind for each. A column is classified as numeric only if
// EVERY non-empty sample parses as a number; one stray string pins it to text.
// This is conservative on purpose — wrong-numeric mis-formatting (e.g. "N/A"
// rendered as a date serial) looks much worse than a text column.
func InferColumnKinds(rows [][]any, columnCount, sampleLimit int) []ColumnKind {
	kinds := make([]ColumnKind, columnCount)
	for c := 0; c < columnCount; c++ {
		kinds[c] = inferColumn(rows, c, sampleLimit)
	}
	return kinds
}

func inferColumn(rows [][]any, col, sampleLimit int) ColumnKind {
	sampled := 0
	allInt := true
	allFloat := true
	allPercent := true
	allCurrency := true
	allDate := true
	sawAnyNumeric := false
	sawAnyDate := false

	for i := 0; i < len(rows) && sampled < sampleLimit; i++ {
		if col >= len(rows[i]) {
			continue
		}
		raw := rows[i][col]
		s := strings.TrimSpace(stringify(raw))
		if s == "" {
			continue
		}
		sampled++

		if _, ok := parseInteger(raw, s); !ok {
			allInt = false
		} else {
			sawAnyNumeric = true
		}
		if _, ok := parseFloat(raw, s); !ok {
			allFloat = false
		} else {
			sawAnyNumeric = true
		}
		if !looksLikePercent(s) {
			allPercent = false
		}
		if !looksLikeCurrency(s) {
			allCurrency = false
		}
		if _, ok := parseISODate(s); !ok {
			allDate = false
		} else {
			sawAnyDate = true
		}
	}

	if sampled == 0 {
		return ColumnText
	}

	switch {
	case allPercent:
		return ColumnPercent
	case allCurrency:
		return ColumnCurrency
	case allDate && sawAnyDate:
		return ColumnDate
	case allInt && sawAnyNumeric:
		return ColumnInteger
	case allFloat && sawAnyNumeric:
		return ColumnFloat
	default:
		return ColumnText
	}
}

// looksLikePercent accepts "12%", "12.5 %", "-3.14%"; rejects anything else.
func looksLikePercent(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "%") {
		return false
	}
	num := strings.TrimSpace(strings.TrimSuffix(s, "%"))
	_, err := strconv.ParseFloat(num, 64)
	return err == nil
}

// currencySymbols is the small, explicit set of recognised leading/trailing
// symbols. Anything else (letters, punctuation) will NOT be treated as a
// currency prefix/suffix — the classifier is intentionally narrow so text like
// "Jan 2026" doesn't get misread as a dollar figure.
var currencySymbols = []string{"$", "€", "£", "¥", "₹", "₱", "¢", "₽", "₩"}

// currencyCodes is the small set of trailing 3-letter ISO codes we accept.
// Extend if real datasets need more — the cost of a miss is minor (the column
// falls back to text), while the cost of a false positive is a garbled sheet.
var currencyCodes = []string{"USD", "EUR", "GBP", "JPY", "INR", "CNY", "AUD", "CAD"}

// looksLikeCurrency accepts a leading currency symbol followed by a number
// ("$1,234.56", "€12.50"), OR a number followed by a trailing ISO code with
// optional whitespace ("1234.56 USD"). Anything else is rejected.
func looksLikeCurrency(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Leading symbol.
	for _, sym := range currencySymbols {
		if strings.HasPrefix(s, sym) {
			rest := strings.TrimSpace(strings.TrimPrefix(s, sym))
			return parsesAsNumber(rest)
		}
	}
	// Trailing ISO code.
	upper := strings.ToUpper(s)
	for _, code := range currencyCodes {
		if strings.HasSuffix(upper, code) {
			head := strings.TrimSpace(upper[:len(upper)-len(code)])
			if head == "" {
				return false
			}
			return parsesAsNumber(head)
		}
	}
	return false
}

// parsesAsNumber strips grouping commas and checks plain Go float parse.
func parsesAsNumber(s string) bool {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// parseISODate recognises the date formats the LLM is most likely to emit:
// "YYYY-MM-DD" (primary), "YYYY/MM/DD", "MM/DD/YYYY". Returns the parsed
// time.Time plus ok.
func parseISODate(s string) (time.Time, bool) {
	for _, layout := range []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseInteger accepts either a typed int/int64/float-with-integer-value or a
// textual integer ("1234", "-5"). The textual form tolerates grouping commas.
func parseInteger(v any, s string) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case float64:
		if x == float64(int64(x)) {
			return int64(x), true
		}
	case float32:
		if x == float32(int32(x)) {
			return int64(x), true
		}
	}
	stripped := strings.ReplaceAll(s, ",", "")
	n, err := strconv.ParseInt(stripped, 10, 64)
	return n, err == nil
}

// parseFloat accepts typed floats or textual decimals with grouping commas.
func parseFloat(v any, s string) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	stripped := strings.ReplaceAll(s, ",", "")
	f, err := strconv.ParseFloat(stripped, 64)
	return f, err == nil
}

// stringify renders v as a string without using fmt, which would keep the
// import surface small if this file grows.
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case time.Time:
		return x.Format("2006-01-02")
	}
	return ""
}
