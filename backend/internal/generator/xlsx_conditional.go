package generator

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ConditionalRule is the LLM-facing spec for a single conditional-format
// rule applied to a cell range. We intentionally flatten excelize's richer
// ConditionalFormatOptions — the LLM rarely needs the esoteric knobs, and a
// small, documented surface is easier to prompt against.
//
// Type values:
//
//	greater_than, less_than, equal_to, not_equal_to,
//	greater_or_equal, less_or_equal,
//	between, not_between,
//	top_n, bottom_n,
//	above_average, below_average,
//	duplicate, unique,
//	text_contains, text_not_contains, text_begins_with, text_ends_with,
//	color_scale_2, color_scale_3,
//	data_bar,
//	blanks, no_blanks, errors, no_errors
type ConditionalRule struct {
	Range string `json:"range"` // e.g. "B2:B10" (sheet-relative, no prefix)
	Type  string `json:"type"`

	// Value / Value2 are used by the numeric + text comparison types. For
	// `between`/`not_between`, both must be set. For `top_n`/`bottom_n`, set
	// Value to the count (e.g. "3").
	Value  string `json:"value,omitempty"`
	Value2 string `json:"value2,omitempty"`

	// FillColor / TextColor target the matching cells (numeric / text /
	// top-bottom / average / duplicate / unique / blank / error rules).
	// Hex without '#'. Either or both may be set.
	FillColor string `json:"fill_color,omitempty"`
	TextColor string `json:"text_color,omitempty"`

	// Bold makes matched cells bold (numeric / text family rules).
	Bold bool `json:"bold,omitempty"`

	// Color-scale + data-bar params. Colors default to a heatmap if omitted.
	MinColor string `json:"min_color,omitempty"`
	MidColor string `json:"mid_color,omitempty"`
	MaxColor string `json:"max_color,omitempty"`
	BarColor string `json:"bar_color,omitempty"`

	// Percent: for top_n / bottom_n, interpret Value as "top N %".
	Percent bool `json:"percent,omitempty"`
}

// applyConditionalRules realises every rule on the sheet. Errors from
// individual rules abort the whole sheet — a broken rule usually means the
// LLM supplied a typoed range or type, and silently dropping it would hide
// the bug behind a confusingly unformatted output.
func applyConditionalRules(f *excelize.File, sheet string, s XlsxSheet, m *XlsxStyleMapper) error {
	for i, r := range s.ConditionalRules {
		if err := applyOneConditionalRule(f, sheet, r, m); err != nil {
			return fmt.Errorf("conditional rule %d (%s, %s): %w", i, r.Type, r.Range, err)
		}
	}
	return nil
}

func applyOneConditionalRule(f *excelize.File, sheet string, r ConditionalRule, m *XlsxStyleMapper) error {
	rangeRef := strings.TrimSpace(r.Range)
	if rangeRef == "" {
		return fmt.Errorf("range is required")
	}

	opts, err := buildConditionalOpts(f, r, m)
	if err != nil {
		return err
	}
	return f.SetConditionalFormat(sheet, rangeRef, opts)
}

// buildConditionalOpts is the LLM-type → excelize-options translator. Split
// out so the tests can exercise it without needing a live excelize.File for
// every permutation.
func buildConditionalOpts(f *excelize.File, r ConditionalRule, m *XlsxStyleMapper) ([]excelize.ConditionalFormatOptions, error) {
	fmtID := 0
	needsFormatID := conditionalNeedsFormat(r.Type)
	if needsFormatID {
		id, err := buildConditionalFormatStyle(f, r, m.Theme())
		if err != nil {
			return nil, err
		}
		fmtID = id
	}

	switch r.Type {
	case "greater_than":
		return cellRule(fmtID, "greater than", r.Value), nil
	case "less_than":
		return cellRule(fmtID, "less than", r.Value), nil
	case "equal_to":
		return cellRule(fmtID, "equal to", r.Value), nil
	case "not_equal_to":
		return cellRule(fmtID, "not equal to", r.Value), nil
	case "greater_or_equal":
		return cellRule(fmtID, "greater than or equal to", r.Value), nil
	case "less_or_equal":
		return cellRule(fmtID, "less than or equal to", r.Value), nil
	case "between":
		return []excelize.ConditionalFormatOptions{{
			Type: "cell", Criteria: "between",
			MinValue: r.Value, MaxValue: r.Value2, Format: &fmtID,
		}}, nil
	case "not_between":
		return []excelize.ConditionalFormatOptions{{
			Type: "cell", Criteria: "not between",
			MinValue: r.Value, MaxValue: r.Value2, Format: &fmtID,
		}}, nil
	case "top_n":
		return []excelize.ConditionalFormatOptions{{
			Type: "top", Criteria: "top", Value: r.Value,
			Percent: r.Percent, Format: &fmtID,
		}}, nil
	case "bottom_n":
		return []excelize.ConditionalFormatOptions{{
			Type: "bottom", Criteria: "bottom", Value: r.Value,
			Percent: r.Percent, Format: &fmtID,
		}}, nil
	case "above_average":
		return []excelize.ConditionalFormatOptions{{
			Type: "average", AboveAverage: true, Format: &fmtID,
		}}, nil
	case "below_average":
		return []excelize.ConditionalFormatOptions{{
			Type: "average", AboveAverage: false, Format: &fmtID,
		}}, nil
	case "duplicate":
		return []excelize.ConditionalFormatOptions{{Type: "duplicate", Format: &fmtID}}, nil
	case "unique":
		return []excelize.ConditionalFormatOptions{{Type: "unique", Format: &fmtID}}, nil
	case "blanks":
		return []excelize.ConditionalFormatOptions{{Type: "blanks", Format: &fmtID}}, nil
	case "no_blanks":
		return []excelize.ConditionalFormatOptions{{Type: "no_blanks", Format: &fmtID}}, nil
	case "errors":
		return []excelize.ConditionalFormatOptions{{Type: "errors", Format: &fmtID}}, nil
	case "no_errors":
		return []excelize.ConditionalFormatOptions{{Type: "no_errors", Format: &fmtID}}, nil
	case "text_contains":
		return textRule(fmtID, "containing", r.Value), nil
	case "text_not_contains":
		return textRule(fmtID, "not containing", r.Value), nil
	case "text_begins_with":
		return textRule(fmtID, "begins with", r.Value), nil
	case "text_ends_with":
		return textRule(fmtID, "ends with", r.Value), nil

	case "color_scale_2":
		minC, maxC := orDefault(r.MinColor, "F8696B"), orDefault(r.MaxColor, "63BE7B")
		return []excelize.ConditionalFormatOptions{{
			Type:     "2_color_scale",
			MinType:  "min", MinColor: hashHex(minC),
			MaxType: "max", MaxColor: hashHex(maxC),
		}}, nil
	case "color_scale_3":
		minC := orDefault(r.MinColor, "F8696B")
		midC := orDefault(r.MidColor, "FFEB84")
		maxC := orDefault(r.MaxColor, "63BE7B")
		return []excelize.ConditionalFormatOptions{{
			Type:     "3_color_scale",
			MinType:  "min", MinColor: hashHex(minC),
			MidType: "percentile", MidValue: "50", MidColor: hashHex(midC),
			MaxType: "max", MaxColor: hashHex(maxC),
		}}, nil
	case "data_bar":
		bar := orDefault(r.BarColor, "638EC6")
		return []excelize.ConditionalFormatOptions{{
			Type:     "data_bar",
			MinType:  "min",
			MaxType:  "max",
			BarColor: hashHex(bar),
		}}, nil
	}
	return nil, fmt.Errorf("unknown rule type %q", r.Type)
}

// cellRule is a shortcut for the numeric "cell" rule family that all share
// the same shape (criteria + single value + format).
func cellRule(fmtID int, criteria, value string) []excelize.ConditionalFormatOptions {
	return []excelize.ConditionalFormatOptions{{
		Type: "cell", Criteria: criteria, Value: value, Format: &fmtID,
	}}
}

// textRule wraps excelize's "text" rule family with a consistent shape.
func textRule(fmtID int, criteria, value string) []excelize.ConditionalFormatOptions {
	return []excelize.ConditionalFormatOptions{{
		Type: "text", Criteria: criteria, Value: value, Format: &fmtID,
	}}
}

// conditionalNeedsFormat reports whether the rule type consumes a Format ID.
// Color-scale / data-bar / icon-set rules are self-styling via their Min/Mid/
// Max/Bar color fields and ignore Format.
func conditionalNeedsFormat(t string) bool {
	switch t {
	case "color_scale_2", "color_scale_3", "data_bar":
		return false
	}
	return true
}

// buildConditionalFormatStyle materialises a one-off style (fill + font color
// + optional bold) for rules that need one. Defaults echo Excel's classic
// "red-on-pink" bad cell palette if the caller left fill/text blank, so the
// rule is always visible.
func buildConditionalFormatStyle(f *excelize.File, r ConditionalRule, t XlsxTheme) (int, error) {
	fill := orDefault(r.FillColor, "FFC7CE")
	text := orDefault(r.TextColor, "9C0006")
	style := &excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{hashHex(fill)}},
		Font: &excelize.Font{
			Color: hashHex(text),
			Bold:  r.Bold,
		},
	}
	_ = t // theme retained — future rules may want theme-aware defaults
	return f.NewConditionalStyle(style)
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
