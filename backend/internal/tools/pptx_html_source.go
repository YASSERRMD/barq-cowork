package tools

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	htmlScriptTagPattern    = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	htmlDangerTagPattern    = regexp.MustCompile(`(?is)</?(iframe|object|embed|link|meta)[^>]*>`)
	htmlEventAttrPattern    = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
	htmlJSHrefPattern       = regexp.MustCompile(`(?i)(href|src)\s*=\s*("javascript:[^"]*"|'javascript:[^']*')`)
	htmlInlineStylePattern  = regexp.MustCompile(`(?is)\sstyle\s*=\s*("([^"]*)"|'([^']*)')`)
	htmlStyleWrapper        = regexp.MustCompile(`(?is)</?style\b[^>]*>`)
	htmlTagPattern          = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlWhitespacePattern   = regexp.MustCompile(`\s+`)
	htmlStructurePattern    = regexp.MustCompile(`(?is)<(div|section|article|header|footer|ul|ol|table|svg|figure|aside|main)\b`)
	htmlDensityBlockPattern = regexp.MustCompile(`(?is)(stat-card|bullet-item|timeline-row|compare-col|summary-chip|panel|note-card|roadmap-row|metric-band|tag-row|summary-strip|<li\b|<tr\b|<svg\b|<table\b)`)
	cssDangerPattern        = regexp.MustCompile(`(?is)@import|expression\s*\(|javascript:|behavior\s*:`)
	cssRuleBodyPattern      = regexp.MustCompile(`\{([^{}]+)\}`)
	cssPXTokenPattern       = regexp.MustCompile(`(\d+(?:\.\d+)?)px`)
)

func sanitizeHTMLMarkup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlScriptTagPattern.ReplaceAllString(raw, "")
	raw = htmlDangerTagPattern.ReplaceAllString(raw, "")
	raw = htmlEventAttrPattern.ReplaceAllString(raw, "")
	raw = htmlJSHrefPattern.ReplaceAllString(raw, `$1="#"`)
	raw = normalizeHTMLLayoutStyles(raw)
	return strings.TrimSpace(raw)
}

func sanitizeCSSMarkup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlStyleWrapper.ReplaceAllString(raw, "")
	raw = cssDangerPattern.ReplaceAllString(raw, "")
	raw = normalizeCSSLayoutRules(raw)
	return strings.TrimSpace(raw)
}

func htmlVisibleText(raw string) string {
	raw = sanitizeHTMLMarkup(raw)
	raw = htmlTagPattern.ReplaceAllString(raw, " ")
	raw = html.UnescapeString(raw)
	raw = htmlWhitespacePattern.ReplaceAllString(strings.TrimSpace(raw), " ")
	return strings.TrimSpace(raw)
}

func htmlHasStructuredBlocks(raw string) bool {
	raw = sanitizeHTMLMarkup(raw)
	return htmlStructurePattern.MatchString(raw) && len(htmlVisibleText(raw)) >= 24
}

func htmlInformationBlockCount(raw string) int {
	raw = sanitizeHTMLMarkup(raw)
	return len(htmlDensityBlockPattern.FindAllStringIndex(raw, -1))
}

func shouldUseHTMLSource(manifest pptxPreviewManifest) bool {
	return len(htmlManifestAuthoringIssues(manifest)) == 0
}

func validatePlannedHTMLDeckSource(planned plannedPPTXPresentation) error {
	var issues []string
	if cssRuleCount(planned.DeckPlan.ThemeCSS) < 4 {
		issues = append(issues, "deck.theme_css")
	}
	if !htmlCoverContentReady(planned.DeckPlan.CoverHTML) {
		issues = append(issues, "deck.cover_html")
	}
	for i, slide := range planned.Slides {
		field := fmt.Sprintf("slides[%d].html", i)
		if !htmlSlideContentReady(slide.Slide.HTML) {
			issues = append(issues, field)
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(
			"write_pptx requires a fully authored HTML/CSS deck; missing or invalid: %s",
			strings.Join(issues, ", "),
		)
	}
	return nil
}

func htmlManifestAuthoringIssues(manifest pptxPreviewManifest) []string {
	var issues []string
	if cssRuleCount(manifest.DeckPlan.ThemeCSS) < 4 {
		issues = append(issues, "deck.theme_css")
	}
	if !htmlCoverContentReady(manifest.DeckPlan.CoverHTML) {
		issues = append(issues, "deck.cover_html")
	}
	for i, slide := range manifest.Slides {
		field := fmt.Sprintf("slides[%d].html", i)
		if !htmlSlideContentReady(slide.HTML) {
			issues = append(issues, field)
		}
	}
	return issues
}

func cssRuleCount(raw string) int {
	return strings.Count(sanitizeCSSMarkup(raw), "{")
}

func htmlStructuredContentReady(raw string, minVisibleChars int) bool {
	textLen := len(htmlVisibleText(raw))
	blocks := htmlInformationBlockCount(raw)
	return htmlHasStructuredBlocks(raw) &&
		textLen >= minVisibleChars &&
		(textLen >= minVisibleChars+60 || blocks >= 3)
}

func htmlCoverContentReady(raw string) bool {
	return htmlStructuredContentReady(raw, 96) && htmlInformationBlockCount(raw) >= 4
}

func htmlSlideContentReady(raw string) bool {
	return htmlStructuredContentReady(raw, 90)
}

func buildPPTXHTMLDocument(manifest pptxPreviewManifest) string {
	pal := previewManifestPalette(manifest)
	var slides strings.Builder
	cover := strings.TrimSpace(manifest.DeckPlan.CoverHTML)
	if cover == "" {
		cover = fallbackHTMLCover(manifest)
	}
	slides.WriteString(`<section class="barq-pptx-slide barq-pptx-cover" data-slide-kind="cover"><div class="barq-pptx-canvas">` + cover + `</div></section>`)
	for _, slide := range manifest.Slides {
		body := strings.TrimSpace(slide.HTML)
		if body == "" {
			body = fallbackHTMLSlideMarkup(slide)
		}
		slides.WriteString(`<section class="barq-pptx-slide" data-slide-kind="content" data-layout="` + html.EscapeString(slide.Layout) + `"><div class="barq-pptx-canvas">` + body + `</div></section>`)
	}

	css := pptxHTMLBaseCSS(pal)
	if extra := strings.TrimSpace(manifest.DeckPlan.ThemeCSS); extra != "" {
		css += "\n" + extra
	}
	css += "\n" + pptxHTMLGuardrailCSS()
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(firstNonEmpty(manifest.Title, "Presentation")) + `</title>
<style>` + css + `</style>
</head>
<body>
<main class="barq-pptx-deck">` + slides.String() + `</main>
</body>
</html>`
}

func pptxHTMLGuardrailCSS() string {
	return `
.barq-pptx-cover .cover-shell,
.barq-pptx-cover .cover-grid {
  align-content: start !important;
  align-items: center !important;
}
.barq-pptx-cover .cover-stack,
.barq-pptx-cover .cover-aside,
.barq-pptx-slide .slide-shell,
.barq-pptx-slide .content-shell {
  align-content: start !important;
}
.barq-pptx-cover .cover-shell {
  padding-top: 72px !important;
  padding-bottom: 64px !important;
}
.barq-pptx-slide .summary-strip {
  align-items: stretch;
}
`
}

func normalizeHTMLLayoutStyles(raw string) string {
	return htmlInlineStylePattern.ReplaceAllStringFunc(raw, func(match string) string {
		submatches := htmlInlineStylePattern.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		quoted := submatches[1]
		value := submatches[2]
		if value == "" {
			value = submatches[3]
		}
		normalized := normalizeStyleDeclarationBlock(value)
		quote := `"`
		if strings.Contains(normalized, `"`) {
			quote = `'`
		} else if strings.HasPrefix(quoted, `'`) {
			quote = `'`
		}
		return ` style=` + quote + normalized + quote
	})
}

func normalizeCSSLayoutRules(raw string) string {
	return cssRuleBodyPattern.ReplaceAllStringFunc(raw, func(match string) string {
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}"))
		return "{" + normalizeStyleDeclarationBlock(body) + "}"
	})
}

func normalizeStyleDeclarationBlock(block string) string {
	parts := strings.Split(block, ";")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			normalized = append(normalized, part)
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(key))
		val := strings.TrimSpace(value)
		switch prop {
		case "padding", "padding-top", "padding-right", "padding-bottom", "padding-left":
			val = clampPXValues(val, 78)
		case "margin", "margin-top", "margin-right", "margin-bottom", "margin-left":
			val = clampPXValues(val, 42)
		case "gap", "grid-gap", "row-gap", "column-gap":
			val = clampPXValues(val, 24)
		case "font-size":
			val = clampPXValues(val, 68)
		case "border-radius":
			val = clampPXValues(val, 24)
		case "line-height":
			val = clampLineHeightValue(val, 1.45)
		case "font-family":
			val = normalizeFontFamilyValue(val)
		}
		normalized = append(normalized, strings.TrimSpace(key)+": "+val)
	}
	return strings.Join(normalized, "; ")
}

func clampPXValues(raw string, max float64) string {
	return cssPXTokenPattern.ReplaceAllStringFunc(raw, func(match string) string {
		valueText := strings.TrimSuffix(match, "px")
		value, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return match
		}
		if value > max {
			value = max
		}
		if value < 0 {
			value = 0
		}
		if value == float64(int(value)) {
			return strconv.Itoa(int(value)) + "px"
		}
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 1, 64), "0"), ".") + "px"
	})
}

func clampLineHeightValue(raw string, max float64) string {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return raw
	}
	if value > max {
		value = max
	}
	if value < 1.05 {
		value = 1.05
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

func normalizeFontFamilyValue(value string) string {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "sans-serif") {
		return `"Aptos", "Helvetica Neue", Arial, sans-serif`
	}
	serifHints := []string{"georgia", "times", "palatino", "garamond"}
	for _, hint := range serifHints {
		if strings.Contains(lower, hint) {
			return `"Georgia", "Times New Roman", serif`
		}
	}
	if strings.Contains(lower, "serif") {
		return `"Georgia", "Times New Roman", serif`
	}
	sansHints := []string{
		"avenir", "sf pro", "inter", "system-ui", "manrope", "poppins", "sora",
		"space grotesk", "ibm plex sans", "segoe", "helvetica", "arial", "aptos", "calibri",
	}
	for _, hint := range sansHints {
		if strings.Contains(lower, hint) {
			return `"Aptos", "Helvetica Neue", Arial, sans-serif`
		}
	}
	return value
}

func pptxHTMLBaseCSS(pal pptxPalette) string {
	return `
:root {
  --bg: ` + hexColor(pal.bg) + `;
  --card: ` + hexColor(pal.card) + `;
  --accent: ` + hexColor(pal.accent) + `;
  --accent-2: ` + hexColor(pal.accent2) + `;
  --text: ` + hexColor(pal.text) + `;
  --muted: ` + hexColor(pal.muted) + `;
  --border: ` + hexColor(pal.border) + `;
  --surface-soft: rgba(255,255,255,0.08);
  --surface-strong: rgba(15,23,42,0.76);
  --shadow: 0 18px 42px rgba(15,23,42,0.16);
}
* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; background: #0b1020; color: var(--text); font-family: "Aptos", "Helvetica Neue", Arial, sans-serif; }
body { padding: 24px; }
.barq-pptx-deck { display: flex; flex-direction: column; gap: 28px; align-items: center; }
.barq-pptx-slide {
  width: 1920px;
  height: 1080px;
  position: relative;
  overflow: hidden;
  background: var(--bg);
  color: var(--text);
  box-shadow: var(--shadow);
}
.barq-pptx-canvas {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.barq-pptx-slide h1, .barq-pptx-slide h2, .barq-pptx-slide h3, .barq-pptx-slide p { margin: 0; }
.barq-pptx-slide ul, .barq-pptx-slide ol { margin: 0; padding: 0; list-style: none; }
.barq-pptx-canvas > .cover-shell, .barq-pptx-canvas > .slide-shell, .barq-pptx-canvas > .content-shell { min-height: 100%; }
.eyebrow { font-size: 17px; font-weight: 700; letter-spacing: 0.12em; text-transform: uppercase; color: var(--muted); }
.display-title { font-size: 60px; line-height: 1.02; font-weight: 800; letter-spacing: -0.04em; }
.section-title { font-size: 38px; line-height: 1.08; font-weight: 800; letter-spacing: -0.03em; }
.lede { font-size: 22px; line-height: 1.34; color: var(--muted); }
.body-copy { font-size: 20px; line-height: 1.42; color: var(--text); }
.muted-copy { color: var(--muted); }
.rule { width: 180px; height: 4px; border-radius: 999px; background: var(--accent); }
.meta-row, .tag-row { display: flex; flex-wrap: wrap; gap: 10px; }
.stack-tight { display: grid; gap: 12px; }
.stack-regular { display: grid; gap: 18px; }
.summary-strip { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.summary-chip {
  display: grid;
  gap: 6px;
  padding: 14px 16px;
  border: 1px solid rgba(255,255,255,0.10);
  background: rgba(255,255,255,0.06);
}
.tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.12);
  background: rgba(255,255,255,0.08);
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}
.panel {
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.10);
  padding: 20px 22px;
}
.panel-light {
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border: 1px solid rgba(15,23,42,0.08);
}
.grid-2, .grid-3, .grid-4 { display: grid; gap: 16px; }
.grid-2 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.grid-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.grid-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.stat-card { padding: 20px 22px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 6px solid var(--accent); }
.stat-value { font-size: 38px; line-height: 1; font-weight: 800; color: var(--accent); margin-bottom: 8px; }
.stat-label { font-size: 16px; line-height: 1.2; font-weight: 800; margin-bottom: 6px; text-transform: uppercase; letter-spacing: 0.03em; }
.stat-desc { font-size: 16px; line-height: 1.42; color: #475569; }
.bullet-list { display: grid; gap: 14px; }
.bullet-item {
  padding: 16px 18px 16px 24px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 5px solid var(--accent);
}
.bullet-item strong { display: block; margin-bottom: 6px; font-size: 20px; }
.timeline-list { display: grid; gap: 14px; }
.timeline-row {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  padding: 16px 18px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 5px solid var(--accent);
}
.timeline-date { font-size: 13px; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; color: var(--accent); }
.compare-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.compare-col { padding: 18px 20px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 6px solid var(--accent); }
.compare-col h3 { font-size: 22px; margin-bottom: 10px; }
.compare-col li { font-size: 18px; line-height: 1.4; margin-top: 8px; }
table { width: 100%; border-collapse: collapse; background: rgba(255,255,255,0.98); color: #0f172a; }
thead th { background: rgba(15,23,42,0.08); font-size: 13px; font-weight: 800; text-transform: uppercase; letter-spacing: 0.08em; }
th, td { border: 1px solid rgba(15,23,42,0.10); padding: 10px 12px; text-align: left; vertical-align: top; font-size: 16px; line-height: 1.35; }
svg { display: block; max-width: 100%; max-height: 100%; }
`
}

func fallbackHTMLCover(manifest pptxPreviewManifest) string {
	title := html.EscapeString(firstNonEmpty(manifest.Title, "Presentation"))
	subtitle := strings.TrimSpace(manifest.Subtitle)
	subtitleHTML := ""
	if subtitle != "" {
		subtitleHTML = `<p class="lede" style="max-width:820px;">` + html.EscapeString(subtitle) + `</p>`
	}
	meta := []string{}
	if subject := strings.TrimSpace(manifest.DeckPlan.Subject); subject != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(subject)+`</span>`)
	}
	if audience := strings.TrimSpace(manifest.DeckPlan.Audience); audience != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(audience)+`</span>`)
	}
	kicker := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Kicker), "Presentation"))
	narrative := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Narrative), strings.TrimSpace(manifest.DeckPlan.NarrativeArc), "Structured decision-ready narrative"))
	colorStory := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.ColorStory), "Deliberate contemporary system"))
	return `<div style="position:absolute;inset:0;background:
linear-gradient(135deg, rgba(15,23,42,0.28), transparent 42%),
radial-gradient(circle at top right, rgba(255,255,255,0.08), transparent 28%),
var(--bg);"></div>
<div class="cover-shell" style="position:absolute;left:78px;top:76px;right:78px;bottom:72px;display:grid;grid-template-columns:minmax(0,1.12fr) 360px;gap:30px;align-items:stretch;">
  <div class="stack-regular" style="align-content:end;">
    <div class="eyebrow">` + kicker + `</div>
    <div class="rule"></div>
    <h1 class="display-title" style="max-width:860px;">` + title + `</h1>
    ` + subtitleHTML + `
    <div class="meta-row">` + strings.Join(meta, "") + `</div>
  </div>
  <div class="stack-tight" style="align-content:end;">
    <div class="panel">
      <div class="eyebrow" style="margin-bottom:10px;">Narrative</div>
      <p class="body-copy">` + narrative + `</p>
    </div>
    <div class="summary-strip">
      <div class="summary-chip"><span class="eyebrow">Audience</span><span class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Audience), "Decision-makers")) + `</span></div>
      <div class="summary-chip"><span class="eyebrow">Theme</span><span class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Theme), "Presentation")) + `</span></div>
      <div class="summary-chip"><span class="eyebrow">Mood</span><span class="body-copy">` + colorStory + `</span></div>
    </div>
  </div>
</div>`
}

func fallbackHTMLSlideMarkup(slide pptxPreviewSlide) string {
	title := html.EscapeString(firstNonEmpty(slide.Heading, "Slide"))
	switch slide.Layout {
	case "stats":
		var cards []string
		for _, stat := range slide.Stats {
			cards = append(cards, `<div class="stat-card"><div class="stat-value">`+html.EscapeString(stat.Value)+`</div><div class="stat-label">`+html.EscapeString(stat.Label)+`</div><div class="stat-desc">`+html.EscapeString(stat.Desc)+`</div></div>`)
		}
		return `<div class="content-shell" style="padding:72px 78px 64px;display:grid;gap:18px;"><h2 class="section-title">` + title + `</h2><div class="grid-` + gridClassCount(len(cards), 4) + `">` + strings.Join(cards, "") + `</div></div>`
	case "timeline":
		var rows []string
		for _, item := range slide.Timeline {
			rows = append(rows, `<div class="timeline-row"><div class="timeline-date">`+html.EscapeString(item.Date)+`</div><div><h3 style="font-size:30px;margin:0 0 8px;">`+html.EscapeString(item.Title)+`</h3><p class="body-copy muted-copy">`+html.EscapeString(item.Desc)+`</p></div></div>`)
		}
		return `<div class="content-shell" style="padding:72px 78px 64px;display:grid;gap:18px;"><h2 class="section-title">` + title + `</h2><div class="timeline-list">` + strings.Join(rows, "") + `</div></div>`
	case "compare":
		left := ""
		right := ""
		if slide.LeftColumn != nil {
			left = fallbackCompareColumnMarkup(*slide.LeftColumn)
		}
		if slide.RightColumn != nil {
			right = fallbackCompareColumnMarkup(*slide.RightColumn)
		}
		return `<div class="content-shell" style="padding:72px 78px 64px;display:grid;gap:18px;"><h2 class="section-title">` + title + `</h2><div class="compare-grid">` + left + right + `</div></div>`
	case "table":
		return `<div class="content-shell" style="padding:72px 78px 64px;display:grid;gap:18px;"><h2 class="section-title">` + title + `</h2>` + fallbackTableMarkup(slide.Table) + `</div>`
	default:
		items := slide.Points
		if len(items) == 0 {
			items = slide.Steps
		}
		if len(items) == 0 {
			for _, card := range slide.Cards {
				items = append(items, card.Title+": "+card.Desc)
			}
		}
		var bullets []string
		for _, point := range items {
			bullets = append(bullets, `<li class="bullet-item"><span class="body-copy">`+html.EscapeString(point)+`</span></li>`)
		}
		return `<div class="content-shell" style="padding:72px 78px 64px;display:grid;gap:18px;"><h2 class="section-title">` + title + `</h2><ul class="bullet-list">` + strings.Join(bullets, "") + `</ul></div>`
	}
}

func gridClassCount(count, max int) string {
	if count < 2 {
		return "2"
	}
	if count > max {
		count = max
	}
	return strconv.Itoa(count)
}

func fallbackCompareColumnMarkup(col pptxCompareColumn) string {
	var items []string
	for _, point := range col.Points {
		items = append(items, `<li>`+html.EscapeString(point)+`</li>`)
	}
	return `<div class="compare-col"><h3>` + html.EscapeString(col.Heading) + `</h3><ul>` + strings.Join(items, "") + `</ul></div>`
}

func fallbackTableMarkup(table *pptxTableData) string {
	if table == nil {
		return `<table><tbody><tr><td>No structured data provided.</td></tr></tbody></table>`
	}
	var head []string
	for _, cell := range table.Headers {
		head = append(head, `<th>`+html.EscapeString(cell)+`</th>`)
	}
	var rows []string
	for _, row := range table.Rows {
		var cells []string
		for _, cell := range row {
			cells = append(cells, `<td>`+html.EscapeString(cell)+`</td>`)
		}
		rows = append(rows, `<tr>`+strings.Join(cells, "")+`</tr>`)
	}
	return `<table><thead><tr>` + strings.Join(head, "") + `</tr></thead><tbody>` + strings.Join(rows, "") + `</tbody></table>`
}
