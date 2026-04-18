package generator

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// StyleRole is a semantic label the LLM (or the builder itself) can ask for
// instead of hand-rolling an excelize.Style. The mapper resolves the role to
// a concrete excelize style ID using the current theme, and caches the ID so
// repeated lookups on large sheets are cheap.
type StyleRole string

const (
	StyleHeader       StyleRole = "header"
	StyleBody         StyleRole = "body"
	StyleZebra        StyleRole = "zebra"
	StyleTotals       StyleRole = "totals"
	StyleAccent       StyleRole = "accent"
	StyleNegative     StyleRole = "negative"
	StylePositive     StyleRole = "positive"
	StyleBodyNumber   StyleRole = "body_number"
	StyleZebraNumber  StyleRole = "zebra_number"
	StyleBodyCurrency StyleRole = "body_currency"
	StyleBodyPercent  StyleRole = "body_percent"
	StyleBodyDate     StyleRole = "body_date"
	StyleMono         StyleRole = "mono"
)

// XlsxStyleMapper builds and memoises excelize style IDs for a given theme +
// file. One mapper is scoped to one *excelize.File; do not share across files.
type XlsxStyleMapper struct {
	file  *excelize.File
	theme XlsxTheme
	cache map[StyleRole]int
}

// NewXlsxStyleMapper returns a mapper bound to f. Missing theme fields are
// filled from the neutral default, so callers can pass a zero-value theme.
func NewXlsxStyleMapper(f *excelize.File, theme XlsxTheme) *XlsxStyleMapper {
	return &XlsxStyleMapper{
		file:  f,
		theme: theme.mergedTheme(),
		cache: make(map[StyleRole]int),
	}
}

// Theme returns the merged theme the mapper is using. Useful for callers that
// need the same color values elsewhere (chart styling, conditional formats).
func (m *XlsxStyleMapper) Theme() XlsxTheme { return m.theme }

// ID resolves a role to the cached excelize style index, building it on first
// request. Returns an error if excelize rejects the style definition.
func (m *XlsxStyleMapper) ID(role StyleRole) (int, error) {
	if id, ok := m.cache[role]; ok {
		return id, nil
	}
	style, err := m.buildStyle(role)
	if err != nil {
		return 0, err
	}
	id, err := m.file.NewStyle(style)
	if err != nil {
		return 0, fmt.Errorf("style %q: %w", role, err)
	}
	m.cache[role] = id
	return id, nil
}

// MustID panics on error — only use during setup when the style set is known
// good. Tests rely on it; production callers should prefer ID.
func (m *XlsxStyleMapper) MustID(role StyleRole) int {
	id, err := m.ID(role)
	if err != nil {
		panic(err)
	}
	return id
}

// buildStyle composes the excelize.Style for a role out of theme parts.
// Unrecognised roles return an error — that is preferable to silently drawing
// an unstyled cell the user expected to be highlighted.
func (m *XlsxStyleMapper) buildStyle(role StyleRole) (*excelize.Style, error) {
	t := m.theme
	border := borderAll(t.BorderColor, 1)

	switch role {
	case StyleHeader:
		return &excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Family: t.HeadingFont,
				Size:   11,
				Color:  hashHex(t.HeaderTextColor),
			},
			Fill: solidFill(t.HeaderFillColor),
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
				WrapText:   true,
			},
			Border: border,
		}, nil

	case StyleBody:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
			Border:    border,
		}, nil

	case StyleZebra:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			Fill:      solidFill(t.ZebraFillColor),
			Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
			Border:    border,
		}, nil

	case StyleTotals:
		return &excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Family: t.HeadingFont,
				Size:   10,
				Color:  hashHex(t.TotalsTextColor),
			},
			Fill:      solidFill(t.TotalsFillColor),
			Alignment: &excelize.Alignment{Vertical: "center"},
			Border:    border,
		}, nil

	case StyleAccent:
		return &excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.AccentColor),
			},
			Alignment: &excelize.Alignment{Vertical: "center"},
			Border:    border,
		}, nil

	case StyleNegative:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.NegativeColor),
			},
			NumFmt:    2, // 0.00
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StylePositive:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.PositiveColor),
			},
			NumFmt:    2,
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleBodyNumber:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			NumFmt:    3, // #,##0
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleZebraNumber:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			Fill:      solidFill(t.ZebraFillColor),
			NumFmt:    3,
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleBodyCurrency:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			NumFmt:    4, // #,##0.00
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleBodyPercent:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			NumFmt:    10, // 0.00%
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleBodyDate:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.BodyFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			NumFmt:    14, // m/d/yy
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			Border:    border,
		}, nil

	case StyleMono:
		return &excelize.Style{
			Font: &excelize.Font{
				Family: t.MonoFont,
				Size:   10,
				Color:  hashHex(t.BodyTextColor),
			},
			Alignment: &excelize.Alignment{Vertical: "center"},
			Border:    border,
		}, nil
	}
	return nil, fmt.Errorf("unknown style role %q", role)
}

// solidFill builds a pattern-1 solid fill with the given hex color.
func solidFill(hex string) excelize.Fill {
	return excelize.Fill{
		Type:    "pattern",
		Pattern: 1,
		Color:   []string{hashHex(hex)},
	}
}

// borderAll returns the four-sided thin border used by every data cell.
func borderAll(color string, style int) []excelize.Border {
	c := hashHex(color)
	return []excelize.Border{
		{Type: "left", Color: c, Style: style},
		{Type: "right", Color: c, Style: style},
		{Type: "top", Color: c, Style: style},
		{Type: "bottom", Color: c, Style: style},
	}
}

// hashHex ensures a hex color is prefixed with '#', which excelize's newer
// APIs accept for font/border colors. Fills use the raw form elsewhere, but
// using the hashed form uniformly is safe.
func hashHex(hex string) string {
	if hex == "" {
		return "#000000"
	}
	if hex[0] == '#' {
		return hex
	}
	return "#" + hex
}
