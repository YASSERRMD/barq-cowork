package generator

import (
	"regexp"
	"strings"
)

// DocxTheme describes the fonts and colors the .docx should use.
// Every field is optional — missing or invalid values fall back to a neutral
// professional palette via normalize().
//
// The intent is that an LLM produces the theme as part of its plan and the
// renderer emits it verbatim into styles.xml / theme1.xml / numbering.xml.
// Nothing about the theme is hardcoded into the document package itself.
type DocxTheme struct {
	Name string `json:"name,omitempty"`

	HeadingFont string `json:"heading_font,omitempty"`
	BodyFont    string `json:"body_font,omitempty"`
	MonoFont    string `json:"mono_font,omitempty"`

	BodyColor      string `json:"body_color,omitempty"`
	Heading1Color  string `json:"heading1_color,omitempty"`
	Heading2Color  string `json:"heading2_color,omitempty"`
	Heading3Color  string `json:"heading3_color,omitempty"`
	Heading4Color  string `json:"heading4_color,omitempty"`
	AccentColor    string `json:"accent_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	LinkColor      string `json:"link_color,omitempty"`
	QuoteColor     string `json:"quote_color,omitempty"`
	MutedColor     string `json:"muted_color,omitempty"`
	CodeBgColor    string `json:"code_bg_color,omitempty"`
	TitleColor     string `json:"title_color,omitempty"`
}

var hex6Pattern = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// normalize returns a copy of t with blank / invalid fields replaced by the
// neutral default palette. It never mutates the receiver.
func (t DocxTheme) normalize() DocxTheme {
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = "Generated"
	}

	out := DocxTheme{
		Name:        name,
		HeadingFont: pickFont(t.HeadingFont, "Inter"),
		BodyFont:    pickFont(t.BodyFont, "Inter"),
		MonoFont:    pickFont(t.MonoFont, "Consolas"),

		BodyColor:      pickHex(t.BodyColor, "0F172A"),
		Heading1Color:  pickHex(t.Heading1Color, "0F172A"),
		Heading2Color:  pickHex(t.Heading2Color, "1E293B"),
		Heading3Color:  pickHex(t.Heading3Color, "334155"),
		Heading4Color:  pickHex(t.Heading4Color, "475569"),
		AccentColor:    pickHex(t.AccentColor, "2563EB"),
		SecondaryColor: pickHex(t.SecondaryColor, "0F766E"),
		LinkColor:      pickHex(t.LinkColor, "1D4ED8"),
		QuoteColor:     pickHex(t.QuoteColor, "475569"),
		MutedColor:     pickHex(t.MutedColor, "475569"),
		CodeBgColor:    pickHex(t.CodeBgColor, "F1F5F9"),
		TitleColor:     pickHex(t.TitleColor, ""),
	}
	if out.TitleColor == "" {
		out.TitleColor = out.Heading1Color
	}
	return out
}

func pickFont(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func pickHex(v, fallback string) string {
	v = strings.TrimSpace(strings.TrimPrefix(v, "#"))
	if v == "" {
		return fallback
	}
	if !hex6Pattern.MatchString(v) {
		return fallback
	}
	return strings.ToUpper(v)
}

// resolveDocxTheme returns a normalized DocxTheme, using defaults when req
// does not carry one.
func resolveDocxTheme(theme *DocxTheme) DocxTheme {
	if theme == nil {
		return DocxTheme{}.normalize()
	}
	return theme.normalize()
}
