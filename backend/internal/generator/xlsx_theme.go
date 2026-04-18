package generator

// XlsxTheme is the LLM-supplied visual identity for a workbook. All fields are
// optional; missing values fall back to neutral defaults inside the style
// mapper so callers can omit whatever they don't care about.
//
// Colors are 6-digit hex strings WITHOUT a leading '#' (e.g. "BE123C"), which
// is the representation excelize's fill/font/border APIs expect. Font names
// are the literal family strings Excel writes into the workbook.
type XlsxTheme struct {
	Name string `json:"name,omitempty"`

	HeadingFont string `json:"heading_font,omitempty"`
	BodyFont    string `json:"body_font,omitempty"`
	MonoFont    string `json:"mono_font,omitempty"`

	HeaderFillColor  string `json:"header_fill_color,omitempty"`
	HeaderTextColor  string `json:"header_text_color,omitempty"`
	BodyTextColor    string `json:"body_text_color,omitempty"`
	ZebraFillColor   string `json:"zebra_fill_color,omitempty"`
	AccentColor      string `json:"accent_color,omitempty"`
	BorderColor      string `json:"border_color,omitempty"`
	TotalsFillColor  string `json:"totals_fill_color,omitempty"`
	TotalsTextColor  string `json:"totals_text_color,omitempty"`
	NegativeColor    string `json:"negative_color,omitempty"`
	PositiveColor    string `json:"positive_color,omitempty"`
}

// defaultXlsxTheme is the neutral fallback applied field-by-field when the
// caller leaves a slot blank. It is intentionally muted so user-supplied
// accents stand out when present.
func defaultXlsxTheme() XlsxTheme {
	return XlsxTheme{
		Name:            "neutral",
		HeadingFont:     "Calibri",
		BodyFont:        "Calibri",
		MonoFont:        "Consolas",
		HeaderFillColor: "1F2937",
		HeaderTextColor: "FFFFFF",
		BodyTextColor:   "111827",
		ZebraFillColor:  "F3F4F6",
		AccentColor:     "2563EB",
		BorderColor:     "D1D5DB",
		TotalsFillColor: "E5E7EB",
		TotalsTextColor: "111827",
		NegativeColor:   "B91C1C",
		PositiveColor:   "15803D",
	}
}

// mergedTheme returns t with every blank field replaced by the corresponding
// default. The receiver is copied — callers keep their original values.
func (t XlsxTheme) mergedTheme() XlsxTheme {
	d := defaultXlsxTheme()
	pick := func(v, fallback string) string {
		if v == "" {
			return fallback
		}
		return v
	}
	return XlsxTheme{
		Name:            pick(t.Name, d.Name),
		HeadingFont:     pick(t.HeadingFont, d.HeadingFont),
		BodyFont:        pick(t.BodyFont, d.BodyFont),
		MonoFont:        pick(t.MonoFont, d.MonoFont),
		HeaderFillColor: pick(t.HeaderFillColor, d.HeaderFillColor),
		HeaderTextColor: pick(t.HeaderTextColor, d.HeaderTextColor),
		BodyTextColor:   pick(t.BodyTextColor, d.BodyTextColor),
		ZebraFillColor:  pick(t.ZebraFillColor, d.ZebraFillColor),
		AccentColor:     pick(t.AccentColor, d.AccentColor),
		BorderColor:     pick(t.BorderColor, d.BorderColor),
		TotalsFillColor: pick(t.TotalsFillColor, d.TotalsFillColor),
		TotalsTextColor: pick(t.TotalsTextColor, d.TotalsTextColor),
		NegativeColor:   pick(t.NegativeColor, d.NegativeColor),
		PositiveColor:   pick(t.PositiveColor, d.PositiveColor),
	}
}
