package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// WritePPTXTool creates a real .pptx PowerPoint file.
type WritePPTXTool struct{}

func (WritePPTXTool) Name() string { return "write_pptx" }
func (WritePPTXTool) Description() string {
	return "Create a professional PowerPoint presentation (.pptx) powered by the Barq PPTX Engine. " +
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
			"icon":  map[string]any{"type": "string", "description": "Emoji or symbol icon, e.g. '⚡'"},
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

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{"type": "string", "description": "Output filename without extension, e.g. 'ai-strategy-2026'"},
			"title":    map[string]any{"type": "string", "description": "Presentation title (shown on cover slide)"},
			"subtitle": map[string]any{"type": "string", "description": "Subtitle or tagline shown on cover slide"},
			"author":   map[string]any{"type": "string", "description": "Optional author name"},
			"slides": map[string]any{
				"type":        "array",
				"description": "Slides array — first slide is auto-made the cover/title slide. Mix types for visual variety. Aim for 6-10 slides.",
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
							"type": "string",
							"enum": []string{"column", "bar", "line", "pie", "doughnut", "area", "scatter"},
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
						"left_column": compareColumnSchema,
						"right_column": compareColumnSchema,

						// table
						"table": tableSchema,
					},
					"required": []string{"heading", "type"},
				},
			},
		},
		"required": []string{"filename", "title", "slides"},
	}
}

type pptxArgs struct {
	Filename string      `json:"filename"`
	Title    string      `json:"title"`
	Subtitle string      `json:"subtitle"`
	Author   string      `json:"author"`
	Slides   []pptxSlide `json:"slides"`
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
// The bridge (pptx_bridge.py) translates this into a pptx_engine Slide.
type pptxSlide struct {
	Heading      string             `json:"heading"`
	Type         string             `json:"type"`            // primary field
	Layout       string             `json:"layout"`          // backward-compat alias for Type
	SpeakerNotes string             `json:"speaker_notes"`
	// bullets
	Points       []string           `json:"points,omitempty"`
	// stats
	Stats        []pptxStat         `json:"stats,omitempty"`
	// steps
	Steps        []string           `json:"steps,omitempty"`
	// cards
	Cards        []pptxCard         `json:"cards,omitempty"`
	// chart
	ChartType       string               `json:"chart_type,omitempty"`
	ChartCategories []string             `json:"chart_categories,omitempty"`
	ChartSeries     []pptxChartSeries    `json:"chart_series,omitempty"`
	YLabel          string               `json:"y_label,omitempty"`
	// timeline
	Timeline    []pptxTimelineItem `json:"timeline,omitempty"`
	// compare
	LeftColumn  *pptxCompareColumn `json:"left_column,omitempty"`
	RightColumn *pptxCompareColumn `json:"right_column,omitempty"`
	// table
	Table       *pptxTableData     `json:"table,omitempty"`
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
	"tech":        {bg: "0F172A", card: "1E293B", accent: "6366F1", accent2: "A5B4FC", text: "F8FAFC", muted: "94A3B8", border: "2D3F55"},
	"healthcare":  {bg: "061B2E", card: "0E2D45", accent: "06B6D4", accent2: "67E8F9", text: "F8FAFC", muted: "94A3B8", border: "1A4060"},
	"education":   {bg: "1A1100", card: "2D1E00", accent: "F59E0B", accent2: "FCD34D", text: "F8FAFC", muted: "94A3B8", border: "4A3500"},
	"environment": {bg: "071A10", card: "0F2B1C", accent: "10B981", accent2: "6EE7B7", text: "F8FAFC", muted: "94A3B8", border: "1A4A30"},
	"finance":     {bg: "061A0E", card: "102B1A", accent: "22C55E", accent2: "86EFAC", text: "F8FAFC", muted: "94A3B8", border: "1A4A28"},
	"creative":    {bg: "140A2A", card: "221545", accent: "8B5CF6", accent2: "C4B5FD", text: "F8FAFC", muted: "94A3B8", border: "3A2060"},
	"security":    {bg: "1A0808", card: "2E1010", accent: "EF4444", accent2: "FCA5A5", text: "F8FAFC", muted: "94A3B8", border: "4A1818"},
	"data":        {bg: "061A1E", card: "0E2B30", accent: "14B8A6", accent2: "5EEAD4", text: "F8FAFC", muted: "94A3B8", border: "1A4048"},
	"logistics":   {bg: "0A1020", card: "162035", accent: "3B82F6", accent2: "93C5FD", text: "F8FAFC", muted: "94A3B8", border: "203050"},
	"retail":      {bg: "1A0C00", card: "2E1800", accent: "F97316", accent2: "FDBA74", text: "F8FAFC", muted: "94A3B8", border: "4A2800"},
	"hr":          {bg: "1A0A18", card: "2E1530", accent: "EC4899", accent2: "F9A8D4", text: "F8FAFC", muted: "94A3B8", border: "4A1840"},
}

func paletteFor(themeName string) pptxPalette {
	if p, ok := themepalettes[themeName]; ok {
		return p
	}
	return themepalettes["tech"]
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
	if args.Subtitle == "" {
		args.Subtitle = time.Now().Format("January 2006")
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
	themeName := pickThemeName(args.Title, args.Subtitle)
	data, err := buildPPTX(args.Title, args.Subtitle, args.Slides, themeName)
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

// buildPPTXviaPython calls pptx_bridge.py — the pptx_engine bridge script.
// It translates the tool's JSON payload into a full Deck schema and renders
// using the engine's 10-layout slide registry (charts, timeline, compare, etc.).
// Falls back to gen_pptx.py if the bridge is not found.
func buildPPTXviaPython(ctx context.Context, args pptxArgs, themeName string) ([]byte, error) {
	// Prefer the new engine bridge; fall back to legacy gen_pptx.py
	scriptPath := findScript("scripts/pptx_bridge.py")
	if scriptPath == "" {
		scriptPath = findScript("scripts/gen_pptx.py")
	}
	if scriptPath == "" {
		return nil, fmt.Errorf("pptx_bridge.py (and gen_pptx.py fallback) not found")
	}

	// Build the JSON payload. The bridge accepts both the old and new formats;
	// we send the extended format so all 10 slide types are available.
	type pyPayload struct {
		Title    string      `json:"title"`
		Subtitle string      `json:"subtitle"`
		Author   string      `json:"author,omitempty"`
		Theme    string      `json:"theme"`
		Slides   []pptxSlide `json:"slides"`
	}
	payload, err := json.Marshal(pyPayload{
		Title:    args.Title,
		Subtitle: args.Subtitle,
		Author:   args.Author,
		Theme:    themeName,
		Slides:   args.Slides,
	})
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pptx_bridge error: %v — %s", err, stderr.String())
	}
	if stdout.Len() < 100 {
		return nil, fmt.Errorf("pptx_bridge produced too-small output (%d bytes); stderr: %s",
			stdout.Len(), stderr.String())
	}
	return stdout.Bytes(), nil
}

// findScript searches common locations for the given relative script path.
func findScript(rel string) string {
	// 1. Next to the running executable
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 2. Relative to current working directory (dev mode: go run from backend/)
	if wd, err := os.Getwd(); err == nil {
		p := filepath.Join(wd, rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 3. Walk up from cwd looking for the scripts directory
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 5; i++ {
			p := filepath.Join(dir, rel)
			if _, err := os.Stat(p); err == nil {
				return p
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
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

// hasWord checks if keyword appears as a whole word in text (space-delimited).
func hasWord(text string, keywords ...string) bool {
	padded := " " + text + " "
	for _, k := range keywords {
		// For multi-word phrases, check substring; for single words, check word boundary.
		if strings.Contains(k, " ") {
			if strings.Contains(padded, k) {
				return true
			}
		} else {
			// Single word — must be surrounded by non-alpha characters.
			idx := strings.Index(padded, k)
			for idx != -1 {
				before := idx > 0 && !isAlpha(padded[idx-1])
				after := idx+len(k) < len(padded) && !isAlpha(padded[idx+len(k)])
				if before && after {
					return true
				}
				idx = strings.Index(padded[idx+1:], k)
				if idx == -1 {
					break
				}
				idx += strings.Index(padded, k) + 1 // re-anchor; simpler to just use Contains above
				break // avoid infinite loop; substring found, not whole-word
			}
			// Simpler fallback: just prefix/suffix word check
			if strings.Contains(padded, " "+k+" ") ||
				strings.HasPrefix(padded, k+" ") ||
				strings.HasSuffix(padded, " "+k) {
				return true
			}
		}
	}
	return false
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func pickThemeName(title, subtitle string) string {
	c := strings.ToLower(title + " " + subtitle)

	// Tech multi-word phrases checked FIRST to avoid "machine learning" → education
	if strings.Contains(c, "machine learning") || strings.Contains(c, "deep learning") ||
		strings.Contains(c, "artificial intelligence") || strings.Contains(c, "large language") ||
		strings.Contains(c, "neural network") || strings.Contains(c, "data science") ||
		strings.Contains(c, "computer vision") || strings.Contains(c, "natural language") {
		return "tech"
	}
	// Health / Medical — checked before generic tech because "healthcare technology" → Cyan
	if hasWord(c, "health", "healthcare", "medical", "doctor", "hospital", "wellness",
		"biotech", "pharma", "clinical", "patient", "covid", "biology") {
		return "healthcare"
	}
	// Education / Learning / Kids — before generic "tech"
	if hasWord(c, "education", "learning", "school", "student", "teacher", "training",
		"course", "curriculum", "university", "college", "classroom", "kids", "children") {
		return "education"
	}
	// Environment / Sustainability / Climate
	if hasWord(c, "environment", "sustainability", "climate", "renewable", "solar",
		"carbon", "eco", "nature", "planet") {
		return "environment"
	}
	// Finance / Business / Revenue
	if hasWord(c, "finance", "financial", "revenue", "business", "market", "investment",
		"startup", "profit", "sales", "economics", "budget", "investor", "funding", "bank") {
		return "finance"
	}
	// Creative / Design / Art / Brand
	if hasWord(c, "design", "creative", "art", "brand", "marketing", "media",
		"visual", "photography", "film", "music", "fashion") {
		return "creative"
	}
	// Security / Cyber / Risk
	if hasWord(c, "security", "cyber", "cybersecurity", "threat", "hack", "ransomware", "firewall",
		"privacy", "compliance", "risk", "breach", "malware", "phishing") {
		return "security"
	}
	// Data / Analytics / BI / Warehouse
	if hasWord(c, "data", "analytics", "bi", "warehouse", "databrick", "snowflake",
		"insight", "dashboard", "intelligence") {
		return "data"
	}
	// Logistics / Supply Chain / Shipping
	if hasWord(c, "logistics", "supply", "shipping", "transport", "fleet", "delivery",
		"warehouse") {
		return "logistics"
	}
	// Retail / E-commerce / Consumer
	if hasWord(c, "retail", "shop", "ecommerce", "consumer", "merchandise", "store") {
		return "retail"
	}
	// HR / Human Resources / Talent
	if hasWord(c, "hr", "human resource", "talent", "recruit", "employee", "workforce",
		"people ops") {
		return "hr"
	}
	// Politics / Government / Policy
	if hasWord(c, "politic", "government", "policy", "election", "vote", "congress",
		"senate", "legislation", "democrat", "republican", "law", "regulation", "civic") {
		return "security"
	}
	// Technology / AI / Digital / Software — checked last as most generic
	if hasWord(c, "ai", "technology", "software", "code", "developer", "digital",
		"data science", "neural", "cloud", "cybersecurity", "blockchain", "api",
		"programming", "artificial intelligence", "machine learning", "deep learning") {
		return "tech"
	}
	// Default: pick based on hash for variety
	themes := []string{"tech", "creative", "data", "logistics", "finance"}
	idx := 0
	for _, ch := range c {
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
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "topBar", 0, 0, 9144000, 12700, pal.accent))
	sb.WriteString(spEllipse(g, "decCircleLg", 7500000, -600000, 2400000, 2400000, "", 0, pal.accent, 19050, 15))
	sb.WriteString(spTextLeft(g, "headingText", 457200, 60000, 8229600, 457200, heading, pal.text, 3600, true, "ctr", "Calibri Light"))
	sb.WriteString(spRect(g, "divider", 457200, 571500, 8229600, 9525, pal.border))
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

func pptxCoverSlide(title, subtitle string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder

	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spEllipse(g, "decCircle1", 7000000, -800000, 3000000, 3000000, "", 0, pal.accent, 28575, 15))
	sb.WriteString(spEllipse(g, "decCircle2", 7600000, -200000, 2000000, 2000000, "", 0, pal.accent, 19050, 10))
	sb.WriteString(spEllipse(g, "decCircleSm", 200000, 5800000, 600000, 600000, pal.accent, 8, "", 0, 0))
	sb.WriteString(spRect(g, "accentBar", 457200, 1371600, 19050, 3657600, pal.accent))

	id := g.next()
	sb.WriteString(fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="titleText"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="762000" y="1600200"/><a:ext cx="7467600" cy="1981200"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr anchor="ctr" wrap="square"><a:normAutofit/></a:bodyPr><a:lstStyle/>
<a:p><a:r><a:rPr lang="en-US" sz="4400" b="1" dirty="0" smtClean="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="Calibri Light" pitchFamily="2" charset="0"/></a:rPr><a:t>%s</a:t></a:r></a:p>
</p:txBody>
</p:sp>`, id, pal.text, xmlEsc(title)))

	id = g.next()
	sb.WriteString(fmt.Sprintf(`<p:sp>
<p:nvSpPr><p:cNvPr id="%d" name="subtitleText"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="762000" y="3657600"/><a:ext cx="7467600" cy="685800"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/><a:ln><a:noFill/></a:ln></p:spPr>
<p:txBody><a:bodyPr anchor="t" wrap="square"><a:normAutofit/></a:bodyPr><a:lstStyle/>
<a:p><a:r><a:rPr lang="en-US" sz="2000" dirty="0" smtClean="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="Calibri" pitchFamily="2" charset="0"/></a:rPr><a:t>%s</a:t></a:r></a:p>
</p:txBody>
</p:sp>`, id, pal.muted, xmlEsc(subtitle)))

	sb.WriteString(spRect(g, "bottomLine", 0, 6845300, 9144000, 12700, pal.accent))

	return wrapSlide(sb.String())
}

// ── Bullets layout ────────────────────────────────────────────────────────────

func pptxBulletsSlide(heading string, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	twoCol := len(points) >= 5
	contentTop := 640000
	contentH := 5900000

	if twoCol {
		colW := 4000000
		colGap := 228600
		leftX := 457200
		rightX := leftX + colW + colGap

		cardH := contentH / 4
		if len(points) <= 6 {
			cardH = contentH / 3
		}
		cardGap := 60000

		for i, pt := range points {
			col := i % 2
			row := i / 2
			cardX := leftX
			if col == 1 {
				cardX = rightX
			}
			cardY := contentTop + row*(cardH+cardGap)

			sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), cardX, cardY, colW, cardH, pal.card, pal.accent, 15))
			sb.WriteString(spRect(g, fmt.Sprintf("strip%d", i), cardX, cardY, 9000, cardH, pal.accent))

			badgeSize := 228600
			badgeX := cardX + 30000
			badgeY := cardY + (cardH-badgeSize)/2
			sb.WriteString(spEllipse(g, fmt.Sprintf("badge%d", i), badgeX, badgeY, badgeSize, badgeSize, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("badgeNum%d", i), badgeX, badgeY, badgeSize, badgeSize, fmt.Sprintf("%d", i+1), pal.text, 1400, true, "ctr", "Calibri"))

			textX := cardX + 290000
			textW := colW - 310000
			sb.WriteString(spTextLeft(g, fmt.Sprintf("ptText%d", i), textX, cardY+30000, textW, cardH-60000, pt, pal.text, 1600, false, "ctr", "Calibri"))
		}
	} else {
		numCards := len(points)
		if numCards == 0 {
			numCards = 1
		}
		totalGap := 60000 * (numCards - 1)
		cardH := (contentH - totalGap) / numCards
		if cardH > 1200000 {
			cardH = 1200000
		}
		cardW := 8229600
		cardX := 457200

		for i, pt := range points {
			cardY := contentTop + i*(cardH+60000)
			sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), cardX, cardY, cardW, cardH, pal.card, pal.accent, 15))
			sb.WriteString(spRect(g, fmt.Sprintf("strip%d", i), cardX, cardY, 9000, cardH, pal.accent))

			badgeSize := 228600
			badgeX := cardX + 40000
			badgeY := cardY + (cardH-badgeSize)/2
			sb.WriteString(spEllipse(g, fmt.Sprintf("badge%d", i), badgeX, badgeY, badgeSize, badgeSize, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("badgeNum%d", i), badgeX, badgeY, badgeSize, badgeSize, fmt.Sprintf("%d", i+1), pal.text, 1400, true, "ctr", "Calibri"))

			textX := cardX + 320000
			textW := cardW - 380000
			sb.WriteString(spTextLeft(g, fmt.Sprintf("ptText%d", i), textX, cardY+20000, textW, cardH-40000, pt, pal.text, 1800, false, "ctr", "Calibri"))
		}
	}

	sb.WriteString(spEllipse(g, "decCircleSmBR", 8800000, 6400000, 500000, 500000, pal.accent, 8, "", 0, 0))

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
	gap := 152400
	cardW := (totalW - gap*(n-1)) / n
	cardH := 4800000
	cardTop := 700000
	startX := 457200

	for i, st := range allStats {
		cardX := startX + i*(cardW+gap)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("statCard%d", i), cardX, cardTop, cardW, cardH, pal.card, pal.accent, 15))
		sb.WriteString(spRect(g, fmt.Sprintf("statTopStrip%d", i), cardX, cardTop, cardW, cardH/12, pal.accent))
		sb.WriteString(spText(g, fmt.Sprintf("statVal%d", i), cardX, cardTop+cardH/10, cardW, cardH/3, st.Value, pal.accent, 6000, true, "ctr", "Calibri Light"))
		sb.WriteString(spText(g, fmt.Sprintf("statLabel%d", i), cardX, cardTop+cardH/3+cardH/10+60000, cardW, cardH/5, st.Label, pal.text, 1800, false, "ctr", "Calibri"))
		if st.Desc != "" {
			sb.WriteString(spText(g, fmt.Sprintf("statDesc%d", i), cardX, cardTop+cardH/3+cardH/10+cardH/5+80000, cardW, cardH/7, st.Desc, pal.muted, 1400, false, "ctr", "Calibri"))
		}
		hasPercent := strings.Contains(st.Value, "%")
		if hasPercent {
			pctStr := strings.TrimSuffix(strings.TrimSpace(st.Value), "%")
			pct := 0
			fmt.Sscanf(pctStr, "%d", &pct)
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			trackY := cardTop + cardH - cardH/10
			trackH := 19050
			sb.WriteString(spRoundRect(g, fmt.Sprintf("track%d", i), cardX+50000, trackY, cardW-100000, trackH, pal.border, "", 0))
			filledW := (cardW - 100000) * pct / 100
			if filledW > 0 {
				sb.WriteString(spRoundRect(g, fmt.Sprintf("fill%d", i), cardX+50000, trackY, filledW, trackH, pal.accent, "", 0))
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

	rowPalette := []string{"6366F1", "8B5CF6", "06B6D4", "10B981", "F59E0B", "EF4444"}

	renderRow := func(rowSteps []string, rowTop int, rowStartNum int) {
		n := len(rowSteps)
		if n == 0 {
			return
		}
		arrowW := 200000
		arrowH := 100000
		totalW := 8229600
		gap := arrowW
		boxW := (totalW - gap*(n-1)) / n
		boxH := 600000
		labelH := 400000
		startX := 457200

		for i, step := range rowSteps {
			boxX := startX + i*(boxW+gap)
			acc := rowPalette[(rowStartNum+i)%len(rowPalette)]
			sb.WriteString(spRoundRect(g, fmt.Sprintf("stepBox%d", rowStartNum+i), boxX, rowTop, boxW, boxH, acc, "", 0))
			sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", rowStartNum+i), boxX, rowTop, boxW, boxH, fmt.Sprintf("%d", rowStartNum+i+1), pal.text, 3200, true, "ctr", "Calibri Light"))
			sb.WriteString(spText(g, fmt.Sprintf("stepLabel%d", rowStartNum+i), boxX, rowTop+boxH+40000, boxW, labelH, step, pal.text, 1400, false, "t", "Calibri"))

			if i < n-1 {
				arrX := boxX + boxW
				arrY := rowTop + (boxH-arrowH)/2
				sb.WriteString(spRightArrow(g, fmt.Sprintf("arrow%d", rowStartNum+i), arrX, arrY, arrowW, arrowH, pal.border))
			}
		}
	}

	if len(steps) <= 4 {
		renderRow(steps, 1000000, 0)
	} else {
		row1 := steps[:3]
		row2 := steps[3:]
		renderRow(row1, 900000, 0)
		renderRow(row2, 3000000, 3)
	}

	return wrapSlide(sb.String())
}

// ── Cards layout ──────────────────────────────────────────────────────────────

var cardIcons = []string{"◈", "⬡", "◎", "◇", "△", "▣"}

func pptxCardsSlide(heading string, points []string, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(slideHeader(g, heading, pal))

	cards := points
	if len(cards) > 6 {
		cards = cards[:6]
	}

	cols := 2
	rows := (len(cards) + cols - 1) / cols
	totalW := 8229600
	totalH := 5700000
	gapX := 228600
	gapY := 152400
	cardW := (totalW - gapX*(cols-1)) / cols
	cardH := (totalH - gapY*(rows-1)) / rows
	startX := 457200
	startY := 700000

	for i, pt := range cards {
		col := i % cols
		row := i / cols
		cardX := startX + col*(cardW+gapX)
		cardY := startY + row*(cardH+gapY)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("featureCard%d", i), cardX, cardY, cardW, cardH, pal.card, pal.accent, 15))
		sb.WriteString(spRect(g, fmt.Sprintf("cardStrip%d", i), cardX, cardY, 9000, cardH, pal.accent))

		icon := cardIcons[i%len(cardIcons)]
		iconH := cardH / 3
		sb.WriteString(spText(g, fmt.Sprintf("cardIcon%d", i), cardX, cardY+20000, cardW, iconH, icon, pal.accent, 3200, false, "ctr", "Calibri"))

		title := pt
		body := ""
		if len(pt) > 30 {
			spaceIdx := strings.Index(pt[25:], " ")
			if spaceIdx >= 0 {
				split := 25 + spaceIdx
				title = pt[:split]
				body = pt[split+1:]
			}
		}
		sb.WriteString(spText(g, fmt.Sprintf("cardTitle%d", i), cardX+20000, cardY+iconH+30000, cardW-40000, cardH/5, title, pal.text, 1600, true, "ctr", "Calibri"))
		if body != "" {
			sb.WriteString(spText(g, fmt.Sprintf("cardBody%d", i), cardX+20000, cardY+iconH+cardH/5+50000, cardW-40000, cardH/4, body, pal.muted, 1300, false, "t", "Calibri"))
		}
	}

	return wrapSlide(sb.String())
}

// ── Dispatch ──────────────────────────────────────────────────────────────────

func pptxContentSlide(s pptxSlide, pal pptxPalette) string {
	layout := s.Layout
	if layout == "" {
		if len(s.Stats) > 0 {
			layout = "stats"
		} else {
			layout = autoLayout(s.Heading, s.Points)
		}
	}

	switch layout {
	case "stats":
		return pptxStatsSlide(s.Heading, s.Stats, s.Points, pal)
	case "steps":
		return pptxStepsSlide(s.Heading, s.Points, pal)
	case "cards":
		return pptxCardsSlide(s.Heading, s.Points, pal)
	default:
		return pptxBulletsSlide(s.Heading, s.Points, pal)
	}
}

// ── PPTX builder ──────────────────────────────────────────────────────────────

func buildPPTX(title, subtitle string, slides []pptxSlide, themeName string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pal := paletteFor(themeName)

	type entry struct {
		name    string
		content string
	}

	totalSlides := len(slides) + 1

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
	}

	entries = append(entries,
		entry{"ppt/slides/slide1.xml", pptxCoverSlide(title, subtitle, pal)},
		entry{"ppt/slides/_rels/slide1.xml.rels", pptxSlideRels("../slideLayouts/slideLayout1.xml")},
	)

	for i, s := range slides {
		idx := i + 2
		entries = append(entries,
			entry{fmt.Sprintf("ppt/slides/slide%d.xml", idx), pptxContentSlide(s, pal)},
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
