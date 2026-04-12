package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// WritePPTXTool creates a real .pptx PowerPoint file.
type WritePPTXTool struct{}

func (WritePPTXTool) Name() string { return "write_pptx" }
func (WritePPTXTool) Description() string {
	return "Create a professional PowerPoint presentation (.pptx) powered by the Barq PPTX Engine. " +
		"Requires a complete deck design brief chosen by the model, then audits each slide for content fit, layout fit, and visual fit before rendering. " +
		"Supports 10 rich slide types: bullets, stats, steps, cards, chart, timeline, compare, table, title, blank. " +
		"Use this for ALL presentation, slides, deck, or slideshow requests. " +
		"Saves to slides/<filename>.pptx."
}
func (WritePPTXTool) InputSchema() map[string]any {
	statItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"value": map[string]any{"type": "string", "description": "Large display value, e.g. '75%' or '$2.4M'"},
			"label": map[string]any{"type": "string", "description": "Metric name"},
			"desc":  map[string]any{"type": "string", "description": "Short description shown below label"},
		},
		"required": []string{"value", "label"},
	}
	cardItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"icon":  map[string]any{"type": "string", "description": "Semantic icon name, e.g. 'automation', 'shield', 'chart', or 'people'"},
			"title": map[string]any{"type": "string", "description": "Card title (short, under 40 chars)"},
			"desc":  map[string]any{"type": "string", "description": "One-sentence description"},
		},
		"required": []string{"icon", "title"},
	}
	timelineItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"date":  map[string]any{"type": "string", "description": "Date label, e.g. 'Q1 2026' or 'Phase 1'"},
			"title": map[string]any{"type": "string", "description": "Milestone title"},
			"desc":  map[string]any{"type": "string", "description": "Optional short description"},
		},
		"required": []string{"date", "title"},
	}
	chartSeriesSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":   map[string]any{"type": "string", "description": "Series name shown in legend"},
			"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}, "description": "Numeric data values"},
			"color":  map[string]any{"type": "string", "description": "Optional hex color override, e.g. '6366F1'"},
		},
		"required": []string{"name", "values"},
	}
	compareColumnSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"heading": map[string]any{"type": "string", "description": "Column heading text"},
			"points":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Comparison points"},
		},
		"required": []string{"heading", "points"},
	}
	tableSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"headers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Column header labels"},
			"rows":    map[string]any{"type": "array", "items": map[string]any{"type": "array"}, "description": "Table data rows (arrays of strings)"},
		},
		"required": []string{"headers", "rows"},
	}
	paletteSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"background": map[string]any{"type": "string", "description": "Optional custom hex background color, e.g. '#F6F1E8'"},
			"card":       map[string]any{"type": "string", "description": "Optional custom hex surface color"},
			"accent":     map[string]any{"type": "string", "description": "Optional custom primary accent hex color"},
			"accent2":    map[string]any{"type": "string", "description": "Optional custom secondary accent hex color"},
			"text":       map[string]any{"type": "string", "description": "Optional custom primary text hex color"},
			"muted":      map[string]any{"type": "string", "description": "Optional custom muted text hex color"},
			"border":     map[string]any{"type": "string", "description": "Optional custom border hex color"},
		},
		"required": []string{"background", "card", "accent", "accent2", "text", "muted", "border"},
	}
	deckSchema := map[string]any{
		"type":        "object",
		"description": "Required deck-level design plan chosen by the model. Fill this completely so the final presentation theme, cover treatment, and palette follow the subject instead of falling back to generic defaults.",
		"properties": map[string]any{
			"subject":      map[string]any{"type": "string", "description": "Explicit subject framing if it should differ from the title"},
			"audience":     map[string]any{"type": "string", "description": "Who this presentation is for, e.g. 'parents and educators' or 'board members'"},
			"narrative":    map[string]any{"type": "string", "description": "High-level story arc for the deck"},
			"theme":        map[string]any{"type": "string", "description": "Domain/theme choice such as tech, education, healthcare, finance, environment, creative, security, data, logistics, retail, or hr"},
			"visual_style": map[string]any{"type": "string", "description": "Overall visual direction, e.g. 'playful classroom collage', 'editorial minimal', 'executive signal-led', or 'bold studio poster'"},
			"cover_style":  map[string]any{"type": "string", "description": "Cover composition style such as editorial, orbit, mosaic, poster, or playful"},
			"color_story":  map[string]any{"type": "string", "description": "Color mood like 'warm daylight', 'cool clinical', 'soft paper', or 'dark command center'"},
			"motif":        map[string]any{"type": "string", "description": "Visual motif or semantic icon token for the cover, e.g. learning, spark, shield, leaf, chart"},
			"kicker":       map[string]any{"type": "string", "description": "Optional short cover kicker shown above the title"},
			"palette":      paletteSchema,
		},
		"required": []string{"subject", "audience", "narrative", "theme", "visual_style", "cover_style", "color_story", "motif", "palette"},
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{"type": "string", "description": "Output filename without extension, e.g. 'ai-strategy-2026'"},
			"title":    map[string]any{"type": "string", "description": "Presentation title (shown on cover slide)"},
			"subtitle": map[string]any{"type": "string", "description": "Subtitle or tagline shown on cover slide"},
			"author":   map[string]any{"type": "string", "description": "Optional author name"},
			"deck":     deckSchema,
			"slides": map[string]any{
				"type":        "array",
				"description": "Slides array — first slide is auto-made the cover/title slide. Plan the overall narrative and vary slide types based on the subject instead of using a fixed template. Aim for 6-10 slides.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"heading": map[string]any{"type": "string", "description": "Slide heading (max 60 chars)"},
						"type": map[string]any{
							"type": "string",
							"enum": []string{"bullets", "stats", "steps", "cards", "chart", "timeline", "compare", "table", "blank"},
							"description": "Slide layout type. " +
								"bullets=text list; stats=KPI metrics; steps=process flow; cards=feature grid; " +
								"chart=data visualisation; timeline=milestone roadmap; compare=two-column; table=data table; blank=empty.",
						},
						"speaker_notes": map[string]any{"type": "string", "description": "Optional speaker notes for this slide"},

						// bullets
						"points": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": "[bullets] Bullet point strings (3-6 recommended)",
						},

						// stats
						"stats": map[string]any{
							"type": "array", "items": statItemSchema,
							"description": "[stats] KPI items (2-4 recommended)",
						},

						// steps
						"steps": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": "[steps] Ordered process step descriptions (3-6 recommended)",
						},

						// cards
						"cards": map[string]any{
							"type": "array", "items": cardItemSchema,
							"description": "[cards] Feature/benefit cards with icon, title, desc (4-6 recommended)",
						},

						// chart
						"chart_type": map[string]any{
							"type":        "string",
							"enum":        []string{"column", "bar", "line", "pie", "doughnut", "area", "scatter"},
							"description": "[chart] Chart type",
						},
						"chart_categories": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": "[chart] Category axis labels",
						},
						"chart_series": map[string]any{
							"type": "array", "items": chartSeriesSchema,
							"description": "[chart] Data series array",
						},
						"y_label": map[string]any{
							"type": "string", "description": "[chart] Optional Y-axis label",
						},

						// timeline
						"timeline": map[string]any{
							"type": "array", "items": timelineItemSchema,
							"description": "[timeline] Milestone items with date, title, desc (3-6 recommended)",
						},

						// compare
						"left_column":  compareColumnSchema,
						"right_column": compareColumnSchema,

						// table
						"table": tableSchema,
					},
					"required": []string{"heading", "type"},
				},
			},
		},
		"required": []string{"filename", "title", "deck", "slides"},
	}
}

type pptxArgs struct {
	Filename string              `json:"filename"`
	Title    string              `json:"title"`
	Subtitle string              `json:"subtitle"`
	Author   string              `json:"author"`
	Deck     pptxDeckDesignInput `json:"deck"`
	Slides   []pptxSlide         `json:"slides"`
}

type pptxPaletteInput struct {
	Background string `json:"background,omitempty"`
	Card       string `json:"card,omitempty"`
	Accent     string `json:"accent,omitempty"`
	Accent2    string `json:"accent2,omitempty"`
	Text       string `json:"text,omitempty"`
	Muted      string `json:"muted,omitempty"`
	Border     string `json:"border,omitempty"`
}

type pptxDeckDesignInput struct {
	Subject     string            `json:"subject,omitempty"`
	Audience    string            `json:"audience,omitempty"`
	Narrative   string            `json:"narrative,omitempty"`
	Theme       string            `json:"theme,omitempty"`
	VisualStyle string            `json:"visual_style,omitempty"`
	CoverStyle  string            `json:"cover_style,omitempty"`
	ColorStory  string            `json:"color_story,omitempty"`
	Motif       string            `json:"motif,omitempty"`
	Kicker      string            `json:"kicker,omitempty"`
	Palette     *pptxPaletteInput `json:"palette,omitempty"`
}

// pptxStat is a KPI metric item for the stats layout.
type pptxStat struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// pptxCard is a feature/benefit card for the cards layout.
type pptxCard struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// pptxTimelineItem is a milestone entry for the timeline layout.
type pptxTimelineItem struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// pptxChartSeries is one data series for the chart layout.
type pptxChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

// pptxCompareColumn is one column for the compare layout.
type pptxCompareColumn struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
}

// pptxTable is tabular data for the table layout.
type pptxTableData struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// pptxSlide is the full slide definition accepted by the write_pptx tool.
type pptxSlide struct {
	Heading      string `json:"heading"`
	Type         string `json:"type"`   // primary field
	Layout       string `json:"layout"` // backward-compat alias for Type
	SpeakerNotes string `json:"speaker_notes"`
	// bullets
	Points []string `json:"points,omitempty"`
	// stats
	Stats []pptxStat `json:"stats,omitempty"`
	// steps
	Steps []string `json:"steps,omitempty"`
	// cards
	Cards []pptxCard `json:"cards,omitempty"`
	// chart
	ChartType       string            `json:"chart_type,omitempty"`
	ChartCategories []string          `json:"chart_categories,omitempty"`
	ChartSeries     []pptxChartSeries `json:"chart_series,omitempty"`
	YLabel          string            `json:"y_label,omitempty"`
	// timeline
	Timeline []pptxTimelineItem `json:"timeline,omitempty"`
	// compare
	LeftColumn  *pptxCompareColumn `json:"left_column,omitempty"`
	RightColumn *pptxCompareColumn `json:"right_column,omitempty"`
	// table
	Table *pptxTableData `json:"table,omitempty"`
}

// pptxPalette holds the full color palette for a presentation theme.
type pptxPalette struct {
	bg      string // background color hex (no #)
	card    string // surface/card color hex
	accent  string // accent color hex
	accent2 string // lighter accent
	text    string // primary text color
	muted   string // muted text color
	border  string // border color hex
}

var themepalettes = map[string]pptxPalette{
	"editorial-light": {bg: "F7F2E8", card: "FFFDF9", accent: "4F46E5", accent2: "A5B4FC", text: "1F2937", muted: "6B7280", border: "D8CFC0"},
	"studio-light":    {bg: "EEF4FF", card: "FFFFFF", accent: "4F46E5", accent2: "A5B4FC", text: "111827", muted: "6B7280", border: "D7DDED"},
	"playful-light":   {bg: "FFF6E5", card: "FFFDF7", accent: "F59E0B", accent2: "FCD34D", text: "1F2937", muted: "6B7280", border: "E8D8B8"},
	"earth-light":     {bg: "F2F1EA", card: "FBFAF6", accent: "10B981", accent2: "6EE7B7", text: "1F2937", muted: "667085", border: "D3CFC1"},
	"executive-dark":  {bg: "0F172A", card: "162033", accent: "4F46E5", accent2: "A5B4FC", text: "F8FAFC", muted: "CBD5E1", border: "334155"},
	"signal-dark":     {bg: "111827", card: "1F2937", accent: "14B8A6", accent2: "5EEAD4", text: "F9FAFB", muted: "D1D5DB", border: "374151"},
}

func paletteFor(themeName string) pptxPalette {
	return resolveDeckPalette(themeName, pptxDeckDesignInput{}, "")
}

func resolveDeckPalette(themeName string, design pptxDeckDesignInput, audience string) pptxPalette {
	family := pickPaletteFamily(themeName, design, audience)
	pal, ok := themepalettes[family]
	if !ok {
		pal = themepalettes["editorial-light"]
	}
	pal = overlayPaletteInput(pal, design.Palette)
	return pal
}

func pickPaletteFamily(themeName string, design pptxDeckDesignInput, audience string) string {
	text := strings.ToLower(strings.Join([]string{
		themeName,
		audience,
		design.VisualStyle,
		design.CoverStyle,
		design.ColorStory,
		design.Kicker,
	}, " "))

	switch {
	case containsAny(text, "playful", "kids", "kid", "children", "classroom", "young learners", "storybook", "collage"):
		return "playful-light"
	case containsAny(text, "editorial", "magazine", "minimal", "clean", "story-led", "refined", "paper"):
		return "editorial-light"
	case containsAny(text, "poster", "studio", "bold", "vibrant", "campaign", "gallery", "showcase"):
		return "studio-light"
	case containsAny(text, "earth", "organic", "natural", "sustainable", "calm", "warm neutral"):
		return "earth-light"
	case containsAny(text, "executive", "board", "premium", "midnight", "dark", "dramatic"):
		return "executive-dark"
	case containsAny(text, "signal", "command center", "dashboard", "operations", "system", "control"):
		return "signal-dark"
	}

	switch themeName {
	case "education", "retail", "hr":
		return "playful-light"
	case "environment":
		return "earth-light"
	case "healthcare", "data", "finance", "logistics":
		return "editorial-light"
	case "security", "tech":
		return "executive-dark"
	case "creative":
		return "studio-light"
	default:
		return "editorial-light"
	}
}

func themeAccentColors(themeName string) (string, string) {
	switch themeName {
	case "healthcare":
		return "0EA5E9", "67E8F9"
	case "education":
		return "F59E0B", "FCD34D"
	case "environment":
		return "10B981", "6EE7B7"
	case "finance":
		return "16A34A", "86EFAC"
	case "creative":
		return "D946EF", "F0ABFC"
	case "security":
		return "EF4444", "FCA5A5"
	case "data":
		return "14B8A6", "5EEAD4"
	case "logistics":
		return "2563EB", "93C5FD"
	case "retail":
		return "EA580C", "FDBA74"
	case "hr":
		return "EC4899", "F9A8D4"
	default:
		return "4F46E5", "A5B4FC"
	}
}

func overlayPaletteInput(base pptxPalette, input *pptxPaletteInput) pptxPalette {
	if input == nil {
		return base
	}
	if color := normalizePaletteHex(input.Background); color != "" {
		base.bg = color
	}
	if color := normalizePaletteHex(input.Card); color != "" {
		base.card = color
	}
	if color := normalizePaletteHex(input.Accent); color != "" {
		base.accent = color
	}
	if color := normalizePaletteHex(input.Accent2); color != "" {
		base.accent2 = color
	}
	if color := normalizePaletteHex(input.Text); color != "" {
		base.text = color
	}
	if color := normalizePaletteHex(input.Muted); color != "" {
		base.muted = color
	}
	if color := normalizePaletteHex(input.Border); color != "" {
		base.border = color
	}
	return base
}

func normalizePaletteHex(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) == 3 {
		value = strings.Repeat(string(value[0]), 2) + strings.Repeat(string(value[1]), 2) + strings.Repeat(string(value[2]), 2)
	}
	if matched, _ := regexp.MatchString(`(?i)^[0-9a-f]{6}$`, value); !matched {
		return ""
	}
	return strings.ToUpper(value)
}

func missingDeckDesignFields(deck pptxDeckDesignInput) []string {
	var missing []string
	if strings.TrimSpace(deck.Subject) == "" {
		missing = append(missing, "deck.subject")
	}
	if strings.TrimSpace(deck.Audience) == "" {
		missing = append(missing, "deck.audience")
	}
	if strings.TrimSpace(deck.Narrative) == "" {
		missing = append(missing, "deck.narrative")
	}
	if strings.TrimSpace(deck.Theme) == "" {
		missing = append(missing, "deck.theme")
	}
	if strings.TrimSpace(deck.VisualStyle) == "" {
		missing = append(missing, "deck.visual_style")
	}
	if strings.TrimSpace(deck.CoverStyle) == "" {
		missing = append(missing, "deck.cover_style")
	}
	if strings.TrimSpace(deck.ColorStory) == "" {
		missing = append(missing, "deck.color_story")
	}
	if strings.TrimSpace(deck.Motif) == "" {
		missing = append(missing, "deck.motif")
	}
	if deck.Palette == nil {
		missing = append(missing, "deck.palette")
	} else {
		missing = append(missing, missingPaletteFields(*deck.Palette)...)
	}
	return missing
}

func missingPaletteFields(palette pptxPaletteInput) []string {
	var missing []string
	fields := map[string]string{
		"background": palette.Background,
		"card":       palette.Card,
		"accent":     palette.Accent,
		"accent2":    palette.Accent2,
		"text":       palette.Text,
		"muted":      palette.Muted,
		"border":     palette.Border,
	}
	for key, value := range fields {
		if normalizePaletteHex(value) == "" {
			missing = append(missing, "deck.palette."+key)
		}
	}
	return missing
}

func (t WritePPTXTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args pptxArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	if args.Title == "" {
		args.Title = strings.ReplaceAll(args.Filename, "-", " ")
	}
	if missing := missingDeckDesignFields(args.Deck); len(missing) > 0 {
		return Err(
			"deck design brief is required for write_pptx; missing or invalid: %s",
			strings.Join(missing, ", "),
		)
	}

	relPath := filepath.Join("slides", args.Filename+".pptx")
	abs, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Err("create slides dir: %v", err)
	}

	// Pure Go PPTX engine — no Python dependency.
	planned := planPPTXPresentation(args.Title, args.Subtitle, args.Slides, args.Deck)
	if err := validatePPTXPresentation(planned); err != nil {
		return Err("plan pptx: %v", err)
	}
	data, err := buildPPTX(args.Title, args.Subtitle, planned)
	if err != nil {
		return Err("build pptx: %v", err)
	}

	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return Err("write pptx: %v", err)
	}

	return OKData(
		fmt.Sprintf("PowerPoint presentation written to %s (%d slides, %d bytes)", relPath, len(args.Slides)+1, len(data)),
		map[string]any{"path": relPath, "size": int64(len(data))},
	)
}

// ── XML helpers ───────────────────────────────────────────────────────────────

func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// ── ID generator ──────────────────────────────────────────────────────────────

type idg struct{ n int }

func (g *idg) next() int { g.n++; return g.n }

// ── Theme accent ──────────────────────────────────────────────────────────────
// pickThemeName selects a coordinated color theme for the entire presentation
// based on keywords in the title/subtitle. Returns a theme name string.

func hasWord(text string, keywords ...string) bool {
	padded := normalizeThemeText(text)
	for _, k := range keywords {
		if containsThemePhrase(padded, k) {
			return true
		}
	}
	return false
}

type themeKeyword struct {
	phrase string
	weight int
}

func normalizeThemeText(text string) string {
	var b strings.Builder
	needsSpace := false
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			needsSpace = true
			continue
		}
		if needsSpace {
			b.WriteByte(' ')
			needsSpace = false
		}
	}
	return " " + strings.Join(strings.Fields(b.String()), " ") + " "
}

func containsThemePhrase(text, phrase string) bool {
	normalizedPhrase := strings.TrimSpace(normalizeThemeText(phrase))
	if normalizedPhrase == "" {
		return false
	}
	return strings.Contains(text, " "+normalizedPhrase+" ")
}

func scoreTheme(text string, keywords []themeKeyword) int {
	score := 0
	for _, keyword := range keywords {
		if containsThemePhrase(text, keyword.phrase) {
			score += keyword.weight
		}
	}
	return score
}

func pickThemeName(title, subtitle string) string {
	c := normalizeThemeText(title + " " + subtitle)
	themeSignals := map[string][]themeKeyword{
		"healthcare": {
			{phrase: "health", weight: 2}, {phrase: "healthcare", weight: 3}, {phrase: "medical", weight: 3},
			{phrase: "doctor", weight: 2}, {phrase: "hospital", weight: 3}, {phrase: "wellness", weight: 2},
			{phrase: "biotech", weight: 2}, {phrase: "pharma", weight: 2}, {phrase: "clinical", weight: 3},
			{phrase: "patient", weight: 3}, {phrase: "biology", weight: 2},
		},
		"education": {
			{phrase: "education", weight: 3}, {phrase: "learning", weight: 2}, {phrase: "school", weight: 3},
			{phrase: "student", weight: 3}, {phrase: "teacher", weight: 3}, {phrase: "training", weight: 2},
			{phrase: "course", weight: 2}, {phrase: "curriculum", weight: 3}, {phrase: "university", weight: 2},
			{phrase: "college", weight: 2}, {phrase: "classroom", weight: 3}, {phrase: "kid", weight: 4},
			{phrase: "kids", weight: 4}, {phrase: "child", weight: 4}, {phrase: "children", weight: 4},
			{phrase: "teen", weight: 3}, {phrase: "teens", weight: 3},
		},
		"environment": {
			{phrase: "environment", weight: 3}, {phrase: "sustainability", weight: 3}, {phrase: "climate", weight: 3},
			{phrase: "renewable", weight: 3}, {phrase: "solar", weight: 2}, {phrase: "carbon", weight: 2},
			{phrase: "eco", weight: 1}, {phrase: "nature", weight: 2}, {phrase: "planet", weight: 2},
		},
		"finance": {
			{phrase: "finance", weight: 3}, {phrase: "financial", weight: 3}, {phrase: "revenue", weight: 2},
			{phrase: "business", weight: 2}, {phrase: "market", weight: 2}, {phrase: "investment", weight: 3},
			{phrase: "startup", weight: 2}, {phrase: "profit", weight: 2}, {phrase: "sales", weight: 2},
			{phrase: "economics", weight: 2}, {phrase: "budget", weight: 2}, {phrase: "investor", weight: 3},
			{phrase: "funding", weight: 2}, {phrase: "bank", weight: 3},
		},
		"creative": {
			{phrase: "design", weight: 3}, {phrase: "creative", weight: 3}, {phrase: "art", weight: 2},
			{phrase: "brand", weight: 2}, {phrase: "marketing", weight: 2}, {phrase: "media", weight: 2},
			{phrase: "visual", weight: 2}, {phrase: "photography", weight: 2}, {phrase: "film", weight: 2},
			{phrase: "music", weight: 2}, {phrase: "fashion", weight: 2},
		},
		"security": {
			{phrase: "security", weight: 3}, {phrase: "cyber", weight: 3}, {phrase: "cybersecurity", weight: 4},
			{phrase: "threat", weight: 2}, {phrase: "hack", weight: 2}, {phrase: "ransomware", weight: 4},
			{phrase: "firewall", weight: 3}, {phrase: "privacy", weight: 2}, {phrase: "compliance", weight: 2},
			{phrase: "risk", weight: 2}, {phrase: "breach", weight: 3}, {phrase: "malware", weight: 4},
			{phrase: "phishing", weight: 4}, {phrase: "government", weight: 1}, {phrase: "policy", weight: 1},
			{phrase: "law", weight: 1}, {phrase: "regulation", weight: 2}, {phrase: "civic", weight: 1},
		},
		"data": {
			{phrase: "data", weight: 3}, {phrase: "analytics", weight: 3}, {phrase: "bi", weight: 2},
			{phrase: "warehouse", weight: 2}, {phrase: "databricks", weight: 3}, {phrase: "snowflake", weight: 3},
			{phrase: "insight", weight: 2}, {phrase: "dashboard", weight: 2}, {phrase: "intelligence", weight: 2},
		},
		"logistics": {
			{phrase: "logistics", weight: 3}, {phrase: "supply chain", weight: 4}, {phrase: "supply", weight: 2},
			{phrase: "shipping", weight: 3}, {phrase: "transport", weight: 2}, {phrase: "fleet", weight: 3},
			{phrase: "delivery", weight: 2}, {phrase: "warehouse", weight: 2},
		},
		"retail": {
			{phrase: "retail", weight: 3}, {phrase: "shop", weight: 2}, {phrase: "ecommerce", weight: 3},
			{phrase: "consumer", weight: 2}, {phrase: "merchandise", weight: 2}, {phrase: "store", weight: 2},
		},
		"hr": {
			{phrase: "hr", weight: 3}, {phrase: "human resources", weight: 4}, {phrase: "human resource", weight: 4},
			{phrase: "talent", weight: 3}, {phrase: "recruit", weight: 3}, {phrase: "employee", weight: 2},
			{phrase: "workforce", weight: 3}, {phrase: "people ops", weight: 4},
		},
		"tech": {
			{phrase: "ai", weight: 2}, {phrase: "technology", weight: 2}, {phrase: "software", weight: 3},
			{phrase: "code", weight: 2}, {phrase: "developer", weight: 2}, {phrase: "digital", weight: 2},
			{phrase: "data science", weight: 3}, {phrase: "neural", weight: 3}, {phrase: "cloud", weight: 2},
			{phrase: "blockchain", weight: 3}, {phrase: "api", weight: 2}, {phrase: "programming", weight: 2},
			{phrase: "artificial intelligence", weight: 4}, {phrase: "machine learning", weight: 4},
			{phrase: "deep learning", weight: 4}, {phrase: "large language", weight: 4},
			{phrase: "computer vision", weight: 4}, {phrase: "natural language", weight: 4},
		},
	}
	themePriority := []string{"healthcare", "education", "environment", "finance", "creative", "security", "data", "logistics", "retail", "hr", "tech"}

	bestTheme := ""
	bestScore := 0
	for _, theme := range themePriority {
		score := scoreTheme(c, themeSignals[theme])
		if score > bestScore {
			bestScore = score
			bestTheme = theme
		}
	}
	if bestTheme != "" {
		return bestTheme
	}

	// Default: pick based on hash for variety
	themes := []string{"tech", "creative", "data", "logistics", "finance"}
	idx := 0
	for _, ch := range strings.TrimSpace(c) {
		idx += int(ch)
	}
	return themes[idx%len(themes)]
}

// ── Auto-layout detection ─────────────────────────────────────────────────────

func autoLayout(heading string, points []string) string {
	h := strings.ToLower(heading)
	for _, k := range []string{"step", "process", "how to", "roadmap", "workflow", "phase", "stage", "pipeline", "journey"} {
		if strings.Contains(h, k) && len(points) >= 2 {
			return "steps"
		}
	}
	numRe := regexp.MustCompile(`^\s*[\d$€£>~]`)
	numCount := 0
	for _, p := range points {
		if numRe.MatchString(p) {
			numCount++
		}
	}
	if numCount >= (len(points)+1)/2 && len(points) >= 2 {
		return "stats"
	}
	if len(points) >= 4 && len(points) <= 6 {
		avg := 0
		for _, p := range points {
			avg += len(p)
		}
		if len(points) > 0 && avg/len(points) < 80 {
			return "cards"
		}
	}
	return "bullets"
}

// ── Shape builders ────────────────────────────────────────────────────────────

func spRect(g *idg, name string, x, y, w, h int, fill string) string {
	id := g.next()
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, fill)
}

func spRoundRect(g *idg, name string, x, y, w, h int, fill, borderCol string, borderAlpha int) string {
	id := g.next()
	borderFill := ""
	if borderCol != "" && borderAlpha > 0 {
		borderFill = fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill>`, borderCol, borderAlpha*1000)
	} else {
		borderFill = `<a:noFill/>`
	}
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="roundRect"><a:avLst><a:gd name="adj" fmla="val 30000"/></a:avLst></a:prstGeom><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln w="9525">%s</a:ln></p:spPr>
<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, fill, borderFill)
}

func spEllipse(g *idg, name string, x, y, w, h int, fill string, fillAlpha int, strokeCol string, strokeW, strokeAlpha int) string {
	id := g.next()
	var fillStr string
	if fill != "" && fillAlpha > 0 {
		fillStr = fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill>`, fill, fillAlpha*1000)
	} else {
		fillStr = `<a:noFill/>`
	}
	var lnStr string
	if strokeCol != "" && strokeW > 0 && strokeAlpha > 0 {
		lnStr = fmt.Sprintf(`<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill></a:ln>`, strokeW, strokeCol, strokeAlpha*1000)
	} else {
		lnStr = `<a:ln><a:noFill/></a:ln>`
	}
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="ellipse"><a:avLst/></a:prstGeom>%s%s</p:spPr>
<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, fillStr, lnStr)
}

func spPresetShape(g *idg, name, prst string, x, y, w, h int, fill string, fillAlpha int, strokeCol string, strokeW, strokeAlpha int) string {
	id := g.next()
	var fillStr string
	if fill != "" && fillAlpha > 0 {
		fillStr = fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill>`, fill, fillAlpha*1000)
	} else {
		fillStr = `<a:noFill/>`
	}
	var lnStr string
	if strokeCol != "" && strokeW > 0 && strokeAlpha > 0 {
		lnStr = fmt.Sprintf(`<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill></a:ln>`, strokeW, strokeCol, strokeAlpha*1000)
	} else {
		lnStr = `<a:ln><a:noFill/></a:ln>`
	}
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="%s"><a:avLst/></a:prstGeom>%s%s</p:spPr>
<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, xmlEsc(prst), fillStr, lnStr)
}

func spText(g *idg, name string, x, y, w, h int, text, color string, sz int, bold bool, anchor string, font string) string {
	id := g.next()
	bAttr := "0"
	if bold {
		bAttr = "1"
	}
	if anchor == "" {
		anchor = "ctr"
	}
	if font == "" {
		font = "Calibri"
	}
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr anchor="%s" wrap="square"><a:normAutofit/></a:bodyPr><a:lstStyle/>
<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="en-US" sz="%d" b="%s" dirty="0" smtClean="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="%s" pitchFamily="2" charset="0"/></a:rPr><a:t>%s</a:t></a:r></a:p>
</p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, anchor, sz, bAttr, color, font, xmlEsc(text))
}

func spTextLeft(g *idg, name string, x, y, w, h int, text, color string, sz int, bold bool, anchor string, font string) string {
	id := g.next()
	bAttr := "0"
	if bold {
		bAttr = "1"
	}
	if anchor == "" {
		anchor = "ctr"
	}
	if font == "" {
		font = "Calibri"
	}
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr anchor="%s" wrap="square"><a:normAutofit/></a:bodyPr><a:lstStyle/>
<a:p><a:r><a:rPr lang="en-US" sz="%d" b="%s" dirty="0" smtClean="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="%s" pitchFamily="2" charset="0"/></a:rPr><a:t>%s</a:t></a:r></a:p>
</p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, anchor, sz, bAttr, color, font, xmlEsc(text))
}

func spRightArrow(g *idg, name string, x, y, w, h int, fill string) string {
	id := g.next()
	return fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rightArrow"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>
</p:sp>`, id, xmlEsc(name), x, y, w, h, fill)
}

// ── Common slide header ───────────────────────────────────────────────────────

func slideHeader(g *idg, heading string, pal pptxPalette) string {
	var sb strings.Builder
	// Gradient-style bg: main bg + subtle gradient overlay
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	// Accent top strip — thicker for visual weight
	sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 38100, pal.accent))
	// Subtle accent glow circle in top-right
	sb.WriteString(spEllipse(g, "glowTR", 7200000, -800000, 2800000, 2800000, pal.accent, 4, "", 0, 0))
	// Heading text — large, bold
	sb.WriteString(spTextLeft(g, "headingText", 457200, 100000, 7500000, 500000, heading, pal.text, 3200, true, "ctr", "Calibri Light"))
	// Accent underline bar beneath heading
	sb.WriteString(spRect(g, "accentLine", 457200, 600000, 1600000, 28575, pal.accent))
	// Faint horizontal rule across full width
	sb.WriteString(spRect(g, "divider", 457200, 640000, 8229600, 6350, pal.border))
	return sb.String()
}

// ── Slide wrappers ────────────────────────────────────────────────────────────

func wrapSlide(body string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
` + body + `
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr>
</p:sld>`
}

// ── Cover slide ───────────────────────────────────────────────────────────────

func pptxCoverSlide(deck pptxDeckContext, pal pptxPalette) string {
	return renderPPTXCoverSlide(deck, pal)
}

// ── Bullets layout ────────────────────────────────────────────────────────────

func pptxBulletsSlide(heading string, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	contentTop := 720000
	availH := 5900000
	leftMargin := 457200

	twoCol := len(points) >= 5
	if twoCol {
		colW := 3900000
		colGap := 400000
		rightX := leftMargin + colW + colGap
		itemH := 700000
		itemGap := 120000

		for i, pt := range points {
			col := i % 2
			row := i / 2
			x := leftMargin
			if col == 1 {
				x = rightX
			}
			y := contentTop + row*(itemH+itemGap)

			// Card background with accent left border
			sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), x, y, colW, itemH, pal.card, pal.border, 10))
			sb.WriteString(spRect(g, fmt.Sprintf("accent%d", i), x, y+50000, 38100, itemH-100000, pal.accent))
			// Text
			sb.WriteString(spTextLeft(g, fmt.Sprintf("text%d", i), x+120000, y+20000, colW-160000, itemH-40000, pt, pal.text, 1500, false, "ctr", "Calibri"))
		}
	} else {
		n := len(points)
		if n == 0 {
			n = 1
		}
		itemGap := 100000
		totalGap := itemGap * (n - 1)
		itemH := (availH - totalGap) / n
		if itemH > 900000 {
			itemH = 900000
		}

		for i, pt := range points {
			y := contentTop + i*(itemH+itemGap)

			// Full-width card with accent left border
			sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), leftMargin, y, 8229600, itemH, pal.card, pal.border, 8))
			// Thick accent left bar
			sb.WriteString(spRect(g, fmt.Sprintf("accent%d", i), leftMargin, y+30000, 38100, itemH-60000, pal.accent))
			// Bullet text
			sb.WriteString(spTextLeft(g, fmt.Sprintf("text%d", i), leftMargin+150000, y+20000, 7800000, itemH-40000, pt, pal.text, 1700, false, "ctr", "Calibri"))
		}
	}

	// Bottom-right accent dot
	sb.WriteString(spEllipse(g, "decDot", 8600000, 6400000, 300000, 300000, pal.accent, 10, "", 0, 0))

	return wrapSlide(sb.String())
}

// ── Stats layout ──────────────────────────────────────────────────────────────

func pptxStatsSlide(heading string, stats []pptxStat, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	allStats := make([]pptxStat, 0, len(stats)+len(points))
	allStats = append(allStats, stats...)
	if len(allStats) == 0 {
		numRe := regexp.MustCompile(`^\s*[\d$€£>~%]`)
		for _, p := range points {
			parts := strings.SplitN(p, ":", 2)
			val := strings.TrimSpace(parts[0])
			label := ""
			desc := ""
			if len(parts) == 2 {
				label = strings.TrimSpace(parts[1])
			}
			if !numRe.MatchString(val) {
				words := strings.Fields(val)
				if len(words) > 0 {
					val = words[0]
					label = strings.Join(words[1:], " ")
				}
			}
			allStats = append(allStats, pptxStat{Value: val, Label: label, Desc: desc})
		}
	}
	if len(allStats) > 4 {
		allStats = allStats[:4]
	}
	n := len(allStats)
	if n == 0 {
		return wrapSlide(sb.String())
	}

	totalW := 8229600
	gap := 200000
	cardW := (totalW - gap*(n-1)) / n
	cardH := 4600000
	cardTop := 800000
	startX := 457200

	for i, st := range allStats {
		cardX := startX + i*(cardW+gap)
		// Card with top accent border
		sb.WriteString(spRoundRect(g, fmt.Sprintf("statCard%d", i), cardX, cardTop, cardW, cardH, pal.card, pal.border, 8))
		// Top accent bar
		sb.WriteString(spRect(g, fmt.Sprintf("topAccent%d", i), cardX+50000, cardTop, cardW-100000, 38100, pal.accent))
		// Big stat value — prominent accent color
		sb.WriteString(spText(g, fmt.Sprintf("statVal%d", i), cardX, cardTop+300000, cardW, cardH/3, st.Value, pal.accent, 5400, true, "ctr", "Calibri Light"))
		// Divider line
		sb.WriteString(spRect(g, fmt.Sprintf("statDiv%d", i), cardX+cardW/4, cardTop+300000+cardH/3, cardW/2, 6350, pal.border))
		// Label
		sb.WriteString(spText(g, fmt.Sprintf("statLbl%d", i), cardX+20000, cardTop+400000+cardH/3, cardW-40000, cardH/5, st.Label, pal.text, 1600, true, "ctr", "Calibri"))
		// Description
		if st.Desc != "" {
			sb.WriteString(spText(g, fmt.Sprintf("statDesc%d", i), cardX+20000, cardTop+500000+cardH/3+cardH/5, cardW-40000, cardH/6, st.Desc, pal.muted, 1200, false, "ctr", "Calibri"))
		}
		// Progress bar for percentages
		if strings.Contains(st.Value, "%") {
			pctStr := strings.TrimSuffix(strings.TrimSpace(st.Value), "%")
			pct := 0
			fmt.Sscanf(pctStr, "%d", &pct)
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			trackY := cardTop + cardH - 200000
			trackH := 38100
			barMargin := 80000
			sb.WriteString(spRoundRect(g, fmt.Sprintf("track%d", i), cardX+barMargin, trackY, cardW-barMargin*2, trackH, pal.border, "", 0))
			filledW := (cardW - barMargin*2) * pct / 100
			if filledW > 0 {
				sb.WriteString(spRoundRect(g, fmt.Sprintf("fill%d", i), cardX+barMargin, trackY, filledW, trackH, pal.accent, "", 0))
			}
		}
	}

	return wrapSlide(sb.String())
}

// ── Steps layout ──────────────────────────────────────────────────────────────

func pptxStepsSlide(heading string, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	steps := points
	if len(steps) == 0 {
		return wrapSlide(sb.String())
	}

	// Horizontal connector line across full width
	lineY := 1600000
	sb.WriteString(spRect(g, "connector", 457200, lineY+100000, 8229600, 6350, pal.border))

	renderRow := func(rowSteps []string, rowTop int, rowStartNum int) {
		n := len(rowSteps)
		if n == 0 {
			return
		}
		totalW := 8229600
		gap := 80000
		boxW := (totalW - gap*(n-1)) / n
		circleSize := 400000
		startX := 457200

		for i, step := range rowSteps {
			boxX := startX + i*(boxW+gap)
			centerX := boxX + boxW/2

			// Step number circle on the connector line
			circleX := centerX - circleSize/2
			circleY := rowTop - circleSize/2 + 100000
			sb.WriteString(spEllipse(g, fmt.Sprintf("circle%d", rowStartNum+i), circleX, circleY, circleSize, circleSize, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("num%d", rowStartNum+i), circleX, circleY, circleSize, circleSize, fmt.Sprintf("%d", rowStartNum+i+1), pal.text, 2000, true, "ctr", "Calibri"))

			// Card below the circle
			cardTop := rowTop + circleSize/2 + 200000
			cardH := 2400000
			sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", rowStartNum+i), boxX, cardTop, boxW, cardH, pal.card, pal.border, 8))
			// Step text
			sb.WriteString(spText(g, fmt.Sprintf("label%d", rowStartNum+i), boxX+30000, cardTop+40000, boxW-60000, cardH-80000, step, pal.text, 1400, false, "t", "Calibri"))

			// Arrow between steps
			if i < n-1 {
				arrX := boxX + boxW + gap/4
				arrY := rowTop + 50000
				sb.WriteString(spRightArrow(g, fmt.Sprintf("arrow%d", rowStartNum+i), arrX, arrY, gap/2, 100000, pal.accent))
			}
		}
	}

	if len(steps) <= 4 {
		renderRow(steps, lineY, 0)
	} else {
		// Second connector line
		sb.WriteString(spRect(g, "connector2", 457200, 3800000+100000, 8229600, 6350, pal.border))
		renderRow(steps[:3], lineY, 0)
		renderRow(steps[3:], 3800000, 3)
	}

	return wrapSlide(sb.String())
}

// ── Cards layout ──────────────────────────────────────────────────────────────

func pptxCardsSlide(heading string, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	cards := points
	if len(cards) > 6 {
		cards = cards[:6]
	}

	n := len(cards)
	cols := 3
	if n <= 2 {
		cols = 2
	} else if n <= 3 {
		cols = 3
	}
	rows := (n + cols - 1) / cols
	totalW := 8229600
	gapX := 200000
	gapY := 200000
	cardW := (totalW - gapX*(cols-1)) / cols
	availH := 5600000
	cardH := (availH - gapY*(rows-1)) / rows
	if cardH > 2800000 {
		cardH = 2800000
	}
	startX := 457200
	startY := 800000

	for i, pt := range cards {
		col := i % cols
		row := i / cols
		cardX := startX + col*(cardW+gapX)
		cardY := startY + row*(cardH+gapY)

		// Card with accent top border
		sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), cardX, cardY, cardW, cardH, pal.card, pal.border, 8))
		sb.WriteString(spRect(g, fmt.Sprintf("topAccent%d", i), cardX+40000, cardY, cardW-80000, 28575, pal.accent))

		// Number indicator (not a circle — a rounded rectangle badge)
		badgeW := 280000
		badgeH := 280000
		sb.WriteString(spRoundRect(g, fmt.Sprintf("badge%d", i), cardX+cardW/2-badgeW/2, cardY+120000, badgeW, badgeH, pal.accent, "", 0))
		sb.WriteString(spText(g, fmt.Sprintf("badgeNum%d", i), cardX+cardW/2-badgeW/2, cardY+120000, badgeW, badgeH, fmt.Sprintf("%d", i+1), pal.text, 1600, true, "ctr", "Calibri"))

		// Title/content text
		title := pt
		body := ""
		if idx := strings.Index(pt, ":"); idx > 0 && idx < 40 {
			title = strings.TrimSpace(pt[:idx])
			body = strings.TrimSpace(pt[idx+1:])
		} else if len(pt) > 35 {
			if spaceIdx := strings.Index(pt[30:], " "); spaceIdx >= 0 {
				title = pt[:30+spaceIdx]
				body = pt[30+spaceIdx+1:]
			}
		}

		titleY := cardY + 120000 + badgeH + 100000
		sb.WriteString(spText(g, fmt.Sprintf("title%d", i), cardX+20000, titleY, cardW-40000, cardH/4, title, pal.text, 1500, true, "t", "Calibri"))
		if body != "" {
			bodyY := titleY + cardH/4 + 20000
			sb.WriteString(spText(g, fmt.Sprintf("body%d", i), cardX+30000, bodyY, cardW-60000, cardH/3, body, pal.muted, 1200, false, "t", "Calibri"))
		}
	}

	return wrapSlide(sb.String())
}

// ── Dispatch ──────────────────────────────────────────────────────────────────

func pptxContentSlide(s pptxSlide, pal pptxPalette) string {
	return renderDeckSlide(s, pal, pptxDeckContext{}, 0)
}

// ── PPTX builder ──────────────────────────────────────────────────────────────

func buildPPTX(title, subtitle string, planned plannedPPTXPresentation) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pal := planned.Palette
	if pal.bg == "" {
		pal = paletteFor(planned.ThemeName)
	}
	deck := newPPTXDeckContext(title, subtitle, planned)
	manifest, err := buildPPTXPreviewManifest(title, subtitle, planned)
	if err != nil {
		return nil, fmt.Errorf("build pptx manifest: %w", err)
	}

	type entry struct {
		name    string
		content string
	}

	totalSlides := len(planned.Slides) + 1

	entries := []entry{
		{"[Content_Types].xml", pptxContentTypes(totalSlides)},
		{"_rels/.rels", pptxRootRels()},
		{"docProps/app.xml", pptxAppXML()},
		{"docProps/core.xml", pptxCoreXML(title)},
		{"ppt/presentation.xml", pptxPresentationXML(totalSlides)},
		{"ppt/_rels/presentation.xml.rels", pptxPresentationRels(totalSlides)},
		{"ppt/theme/theme1.xml", pptxThemeXML(pal)},
		{"ppt/slideMasters/slideMaster1.xml", pptxSlideMasterXML(pal)},
		{"ppt/slideMasters/_rels/slideMaster1.xml.rels", pptxSlideMasterRels()},
		{"ppt/slideLayouts/slideLayout1.xml", pptxLayoutTitleXML()},
		{"ppt/slideLayouts/_rels/slideLayout1.xml.rels", pptxLayoutRels()},
		{"ppt/slideLayouts/slideLayout2.xml", pptxLayoutContentXML()},
		{"ppt/slideLayouts/_rels/slideLayout2.xml.rels", pptxLayoutRels()},
		{pptxPreviewManifestPath, string(manifest)},
	}

	entries = append(entries,
		entry{"ppt/slides/slide1.xml", pptxCoverSlide(deck, pal)},
		entry{"ppt/slides/_rels/slide1.xml.rels", pptxSlideRels("../slideLayouts/slideLayout1.xml")},
	)

	for i, slide := range planned.Slides {
		idx := i + 2
		entries = append(entries,
			entry{fmt.Sprintf("ppt/slides/slide%d.xml", idx), renderDeckSlide(slide.Slide, pal, deck, i)},
			entry{fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", idx), pptxSlideRels("../slideLayouts/slideLayout2.xml")},
		)
	}

	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			return nil, fmt.Errorf("zip create %s: %w", e.name, err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			return nil, fmt.Errorf("zip write %s: %w", e.name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ── Content Types ─────────────────────────────────────────────────────────────

func pptxContentTypes(numSlides int) string {
	var overrides strings.Builder
	for i := 1; i <= numSlides; i++ {
		overrides.WriteString(fmt.Sprintf(
			`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`,
			i,
		))
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="json" ContentType="application/json"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout2.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  ` + overrides.String() + `
</Types>`
}

// ── Root relationships ────────────────────────────────────────────────────────

func pptxRootRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`
}

// ── App / Core properties ─────────────────────────────────────────────────────

func pptxAppXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">
  <Application>Barq Cowork</Application>
  <Slides>0</Slides>
  <Notes>0</Notes>
  <HiddenSlides>0</HiddenSlides>
</Properties>`
}

func pptxCoreXML(title string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties
  xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>%s</dc:title>
  <cp:revision>1</cp:revision>
</cp:coreProperties>`, xmlEsc(title))
}

// ── Presentation ──────────────────────────────────────────────────────────────

func pptxPresentationXML(numSlides int) string {
	var sldIdLst strings.Builder
	for i := 1; i <= numSlides; i++ {
		sldIdLst.WriteString(fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 255+i, i+2))
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
  xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
  saveSubsetFonts="1" autoCompressPictures="0">
  <p:sldMasterIdLst>
    <p:sldMasterId id="2147483648" r:id="rId1"/>
  </p:sldMasterIdLst>
  <p:sldIdLst>` + sldIdLst.String() + `</p:sldIdLst>
  <p:sldSz cx="9144000" cy="6858000" type="screen4x3"/>
  <p:notesSz cx="6858000" cy="9144000"/>
  <p:defaultTextStyle>
    <a:defPPr><a:defRPr lang="en-US"/></a:defPPr>
  </p:defaultTextStyle>
</p:presentation>`
}

func pptxPresentationRels(numSlides int) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>`)
	for i := 1; i <= numSlides; i++ {
		sb.WriteString(fmt.Sprintf(`
  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+2, i))
	}
	sb.WriteString(`
</Relationships>`)
	return sb.String()
}

// ── Theme ─────────────────────────────────────────────────────────────────────

func pptxThemeXML(pal pptxPalette) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="BarqTheme">
  <a:themeElements>
    <a:clrScheme name="Barq">
      <a:dk1><a:srgbClr val="%s"/></a:dk1>
      <a:lt1><a:srgbClr val="%s"/></a:lt1>
      <a:dk2><a:srgbClr val="%s"/></a:dk2>
      <a:lt2><a:srgbClr val="F1F5F9"/></a:lt2>
      <a:accent1><a:srgbClr val="%s"/></a:accent1>
      <a:accent2><a:srgbClr val="%s"/></a:accent2>
      <a:accent3><a:srgbClr val="%s"/></a:accent3>
      <a:accent4><a:srgbClr val="%s"/></a:accent4>
      <a:accent5><a:srgbClr val="%s"/></a:accent5>
      <a:accent6><a:srgbClr val="%s"/></a:accent6>
      <a:hlink><a:srgbClr val="%s"/></a:hlink>
      <a:folHlink><a:srgbClr val="%s"/></a:folHlink>
    </a:clrScheme>`, pal.bg, pal.text, pal.card, pal.accent, pal.accent2, pal.accent, pal.accent2, pal.accent, pal.accent2, pal.accent, pal.accent2) + `
    <a:fontScheme name="Barq">
      <a:majorFont>
        <a:latin typeface="Calibri Light" panose="020F0302020204030204"/>
        <a:ea typeface=""/>
        <a:cs typeface=""/>
      </a:majorFont>
      <a:minorFont>
        <a:latin typeface="Calibri" panose="020F0502020204030204"/>
        <a:ea typeface=""/>
        <a:cs typeface=""/>
      </a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="Office">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:gradFill rotWithShape="1">
          <a:gsLst>
            <a:gs pos="0"><a:schemeClr val="phClr"><a:lumMod val="110000"/><a:satMod val="105000"/><a:tint val="67000"/></a:schemeClr></a:gs>
            <a:gs pos="50000"><a:schemeClr val="phClr"><a:lumMod val="105000"/><a:satMod val="103000"/><a:tint val="73000"/></a:schemeClr></a:gs>
            <a:gs pos="100000"><a:schemeClr val="phClr"><a:lumMod val="105000"/><a:satMod val="109000"/><a:tint val="81000"/></a:schemeClr></a:gs>
          </a:gsLst>
          <a:lin ang="5400000" scaled="0"/>
        </a:gradFill>
        <a:gradFill rotWithShape="1">
          <a:gsLst>
            <a:gs pos="0"><a:schemeClr val="phClr"><a:satMod val="103000"/><a:lumMod val="102000"/><a:tint val="94000"/></a:schemeClr></a:gs>
            <a:gs pos="100000"><a:schemeClr val="phClr"><a:lumMod val="99000"/><a:satMod val="120000"/><a:shade val="78000"/></a:schemeClr></a:gs>
          </a:gsLst>
          <a:lin ang="5400000" scaled="0"/>
        </a:gradFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="6350" cap="flat" cmpd="sng" algn="ctr">
          <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
          <a:prstDash val="solid"/>
          <a:miter lim="800000"/>
        </a:ln>
        <a:ln w="12700" cap="flat" cmpd="sng" algn="ctr">
          <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
          <a:prstDash val="solid"/>
          <a:miter lim="800000"/>
        </a:ln>
        <a:ln w="19050" cap="flat" cmpd="sng" algn="ctr">
          <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
          <a:prstDash val="solid"/>
          <a:miter lim="800000"/>
        </a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle>
          <a:effectLst>
            <a:outerShdw blurRad="57150" dist="19050" dir="5400000" algn="ctr" rotWithShape="0">
              <a:srgbClr val="000000"><a:alpha val="63000"/></a:srgbClr>
            </a:outerShdw>
          </a:effectLst>
        </a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"><a:tint val="95000"/><a:satMod val="170000"/></a:schemeClr></a:solidFill>
        <a:gradFill rotWithShape="1">
          <a:gsLst>
            <a:gs pos="0"><a:schemeClr val="phClr"><a:tint val="93000"/><a:satMod val="150000"/><a:shade val="98000"/><a:lumMod val="102000"/></a:schemeClr></a:gs>
            <a:gs pos="50000"><a:schemeClr val="phClr"><a:tint val="98000"/><a:satMod val="130000"/><a:shade val="90000"/><a:lumMod val="103000"/></a:schemeClr></a:gs>
            <a:gs pos="100000"><a:schemeClr val="phClr"><a:shade val="63000"/><a:satMod val="120000"/></a:schemeClr></a:gs>
          </a:gsLst>
          <a:lin ang="16200000" scaled="0"/>
        </a:gradFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`
}

// ── Slide Master ──────────────────────────────────────────────────────────────

func pptxSlideMasterXML(pal pptxPalette) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
        <a:effectLst/>
      </p:bgPr>
    </p:bg>`, pal.bg) + `
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="2147483649" r:id="rId1"/>
    <p:sldLayoutId id="2147483650" r:id="rId2"/>
  </p:sldLayoutIdLst>
  <p:hf sldNum="0" hdr="0" ftr="0" dt="0"/>
  <p:txStyles>
    <p:titleStyle>
      <a:lvl1pPr algn="l" rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1">
        <a:spcBef><a:spcPts val="0"/></a:spcBef>
        <a:buNone/>
        <a:defRPr lang="en-US" smtClean="0">
          <a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill>
          <a:latin typeface="+mj-lt"/>
        </a:defRPr>
      </a:lvl1pPr>
    </p:titleStyle>
    <p:bodyStyle>
      <a:lvl1pPr marL="342900" indent="-342900" algn="l" rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1">
        <a:spcBef><a:spcPts val="200"/></a:spcBef>
        <a:buFont typeface="Arial" charset="0"/>
        <a:buChar char="&#x2022;"/>
        <a:defRPr lang="en-US" smtClean="0">
          <a:solidFill><a:srgbClr val="E2E8F0"/></a:solidFill>
          <a:latin typeface="+mn-lt"/>
        </a:defRPr>
      </a:lvl1pPr>
    </p:bodyStyle>
    <p:otherStyle>
      <a:defPPr>
        <a:defRPr lang="en-US"/>
      </a:defPPr>
    </p:otherStyle>
  </p:txStyles>
</p:sldMaster>`
}

func pptxSlideMasterRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout2.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`
}

// ── Slide Layouts ─────────────────────────────────────────────────────────────

func pptxLayoutTitleXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
  type="title" preserve="1">
  <p:cSld name="Title Slide">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr>
</p:sldLayout>`
}

func pptxLayoutContentXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
  type="obj" preserve="1">
  <p:cSld name="Content Slide">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr>
</p:sldLayout>`
}

func pptxLayoutRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`
}

func pptxSlideRels(layoutTarget string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="%s"/>
</Relationships>`, layoutTarget)
}
