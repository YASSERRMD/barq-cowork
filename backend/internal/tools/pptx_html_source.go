package tools

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	htmlScriptTagPattern  = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	htmlDangerTagPattern  = regexp.MustCompile(`(?is)</?(iframe|object|embed|link|meta)[^>]*>`)
	htmlEventAttrPattern  = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
	htmlJSHrefPattern     = regexp.MustCompile(`(?i)(href|src)\s*=\s*("javascript:[^"]*"|'javascript:[^']*')`)
	htmlStyleWrapper      = regexp.MustCompile(`(?is)</?style\b[^>]*>`)
	htmlTagPattern        = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlWhitespacePattern = regexp.MustCompile(`\s+`)
	htmlStructurePattern  = regexp.MustCompile(`(?is)<(div|section|article|header|footer|ul|ol|table|svg|figure|aside|main)\b`)
	cssDangerPattern      = regexp.MustCompile(`(?is)@import|expression\s*\(|javascript:|behavior\s*:`)
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
	return strings.TrimSpace(raw)
}

func sanitizeCSSMarkup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlStyleWrapper.ReplaceAllString(raw, "")
	raw = cssDangerPattern.ReplaceAllString(raw, "")
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

func shouldUseHTMLSource(manifest pptxPreviewManifest) bool {
	return len(htmlManifestAuthoringIssues(manifest)) == 0
}

func validatePlannedHTMLDeckSource(planned plannedPPTXPresentation) error {
	var issues []string
	if cssRuleCount(planned.DeckPlan.ThemeCSS) < 4 {
		issues = append(issues, "deck.theme_css")
	}
	if !htmlStructuredContentReady(planned.DeckPlan.CoverHTML, 28) {
		issues = append(issues, "deck.cover_html")
	}
	for i, slide := range planned.Slides {
		field := fmt.Sprintf("slides[%d].html", i)
		if !htmlStructuredContentReady(slide.Slide.HTML, 40) {
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
	if !htmlStructuredContentReady(manifest.DeckPlan.CoverHTML, 28) {
		issues = append(issues, "deck.cover_html")
	}
	for i, slide := range manifest.Slides {
		field := fmt.Sprintf("slides[%d].html", i)
		if !htmlStructuredContentReady(slide.HTML, 40) {
			issues = append(issues, field)
		}
	}
	return issues
}

func cssRuleCount(raw string) int {
	return strings.Count(sanitizeCSSMarkup(raw), "{")
}

func htmlStructuredContentReady(raw string, minVisibleChars int) bool {
	return htmlHasStructuredBlocks(raw) && len(htmlVisibleText(raw)) >= minVisibleChars
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
  --shadow: 0 24px 60px rgba(15,23,42,0.18);
}
* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; background: #0b1020; color: var(--text); font-family: "Avenir Next", "SF Pro Display", "Inter", "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, sans-serif; }
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
.eyebrow { font-size: 30px; font-weight: 700; letter-spacing: 0.12em; text-transform: uppercase; color: var(--muted); }
.display-title { font-size: 82px; line-height: 1.02; font-weight: 800; letter-spacing: -0.04em; }
.section-title { font-size: 56px; line-height: 1.06; font-weight: 800; letter-spacing: -0.03em; }
.lede { font-size: 30px; line-height: 1.35; color: var(--muted); }
.body-copy { font-size: 28px; line-height: 1.5; color: var(--text); }
.muted-copy { color: var(--muted); }
.rule { width: 220px; height: 6px; border-radius: 999px; background: var(--accent); }
.meta-row, .tag-row { display: flex; flex-wrap: wrap; gap: 14px; }
.tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 46px;
  padding: 0 18px;
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.12);
  background: rgba(255,255,255,0.08);
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
}
.panel {
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.10);
  padding: 28px 32px;
}
.panel-light {
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border: 1px solid rgba(15,23,42,0.08);
}
.grid-2, .grid-3, .grid-4 { display: grid; gap: 22px; }
.grid-2 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.grid-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.grid-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.stat-card { padding: 28px 30px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 8px solid var(--accent); }
.stat-value { font-size: 52px; line-height: 1; font-weight: 800; color: var(--accent); margin-bottom: 14px; }
.stat-label { font-size: 22px; line-height: 1.2; font-weight: 800; margin-bottom: 10px; }
.stat-desc { font-size: 22px; line-height: 1.45; color: #475569; }
.bullet-list { display: grid; gap: 18px; }
.bullet-item {
  padding: 22px 24px 22px 34px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 8px solid var(--accent);
}
.bullet-item strong { display: block; margin-bottom: 8px; font-size: 26px; }
.timeline-list { display: grid; gap: 18px; }
.timeline-row {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
  padding: 24px 26px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 8px solid var(--accent);
}
.timeline-date { font-size: 20px; font-weight: 800; letter-spacing: 0.06em; text-transform: uppercase; color: var(--accent); }
.compare-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; }
.compare-col { padding: 28px 30px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 8px solid var(--accent); }
.compare-col h3 { font-size: 28px; margin-bottom: 14px; }
.compare-col li { font-size: 24px; line-height: 1.45; margin-top: 12px; }
table { width: 100%; border-collapse: collapse; background: rgba(255,255,255,0.98); color: #0f172a; }
thead th { background: rgba(15,23,42,0.08); font-size: 20px; font-weight: 800; text-transform: uppercase; letter-spacing: 0.06em; }
th, td { border: 1px solid rgba(15,23,42,0.10); padding: 16px 18px; text-align: left; vertical-align: top; font-size: 22px; }
svg { display: block; max-width: 100%; max-height: 100%; }
`
}

func fallbackHTMLCover(manifest pptxPreviewManifest) string {
	title := html.EscapeString(firstNonEmpty(manifest.Title, "Presentation"))
	subtitle := strings.TrimSpace(manifest.Subtitle)
	subtitleHTML := ""
	if subtitle != "" {
		subtitleHTML = `<p class="lede" style="max-width:960px;">` + html.EscapeString(subtitle) + `</p>`
	}
	meta := []string{}
	if subject := strings.TrimSpace(manifest.DeckPlan.Subject); subject != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(subject)+`</span>`)
	}
	if audience := strings.TrimSpace(manifest.DeckPlan.Audience); audience != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(audience)+`</span>`)
	}
	kicker := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Kicker), "Presentation"))
	return `<div style="position:absolute;inset:0;background:
linear-gradient(135deg, rgba(15,23,42,0.28), transparent 42%),
radial-gradient(circle at top right, rgba(255,255,255,0.08), transparent 28%),
var(--bg);"></div>
<div style="position:absolute;left:110px;top:110px;right:110px;bottom:110px;display:grid;grid-template-columns:minmax(0,1.2fr) 420px;gap:48px;align-items:end;">
  <div style="display:grid;gap:24px;align-content:end;">
    <div class="eyebrow">` + kicker + `</div>
    <div class="rule"></div>
    <h1 class="display-title" style="max-width:1000px;">` + title + `</h1>
    ` + subtitleHTML + `
    <div class="meta-row">` + strings.Join(meta, "") + `</div>
  </div>
  <div style="align-self:stretch;display:grid;gap:18px;align-content:end;">
    <div class="panel" style="min-height:220px;background:linear-gradient(180deg, rgba(255,255,255,0.08), rgba(255,255,255,0.04));"></div>
    <div class="panel" style="min-height:160px;border-left:10px solid var(--accent);"></div>
    <div class="panel" style="min-height:120px;border-left:10px solid var(--accent-2);"></div>
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
		return `<div style="padding:94px 96px 80px;display:grid;gap:28px;"><h2 class="section-title">` + title + `</h2><div class="grid-` + gridClassCount(len(cards), 4) + `">` + strings.Join(cards, "") + `</div></div>`
	case "timeline":
		var rows []string
		for _, item := range slide.Timeline {
			rows = append(rows, `<div class="timeline-row"><div class="timeline-date">`+html.EscapeString(item.Date)+`</div><div><h3 style="font-size:30px;margin:0 0 8px;">`+html.EscapeString(item.Title)+`</h3><p class="body-copy muted-copy">`+html.EscapeString(item.Desc)+`</p></div></div>`)
		}
		return `<div style="padding:94px 96px 80px;display:grid;gap:24px;"><h2 class="section-title">` + title + `</h2><div class="timeline-list">` + strings.Join(rows, "") + `</div></div>`
	case "compare":
		left := ""
		right := ""
		if slide.LeftColumn != nil {
			left = fallbackCompareColumnMarkup(*slide.LeftColumn)
		}
		if slide.RightColumn != nil {
			right = fallbackCompareColumnMarkup(*slide.RightColumn)
		}
		return `<div style="padding:94px 96px 80px;display:grid;gap:24px;"><h2 class="section-title">` + title + `</h2><div class="compare-grid">` + left + right + `</div></div>`
	case "table":
		return `<div style="padding:94px 96px 80px;display:grid;gap:24px;"><h2 class="section-title">` + title + `</h2>` + fallbackTableMarkup(slide.Table) + `</div>`
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
		return `<div style="padding:94px 96px 80px;display:grid;gap:24px;"><h2 class="section-title">` + title + `</h2><ul class="bullet-list">` + strings.Join(bullets, "") + `</ul></div>`
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
