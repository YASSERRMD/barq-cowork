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
		(textLen >= minVisibleChars+24 || blocks >= 2)
}

func htmlCoverContentReady(raw string) bool {
	raw = sanitizeHTMLMarkup(raw)
	if !htmlHasStructuredBlocks(raw) || len(htmlVisibleText(raw)) < 40 {
		return false
	}
	lower := strings.ToLower(raw)
	hasHeading := strings.Contains(lower, "<h1") ||
		strings.Contains(lower, "display-title") ||
		strings.Contains(lower, "section-title")
	hasSupport := strings.Contains(lower, "<p") ||
		strings.Contains(lower, "lede") ||
		strings.Contains(lower, "tag") ||
		strings.Contains(lower, "panel") ||
		strings.Contains(lower, "summary-chip") ||
		strings.Contains(lower, "<svg")
	return hasHeading && hasSupport
}

func htmlSlideContentReady(raw string) bool {
	raw = sanitizeHTMLMarkup(raw)
	if !htmlHasStructuredBlocks(raw) || len(htmlVisibleText(raw)) < 56 {
		return false
	}
	lower := strings.ToLower(raw)
	hasHeading := strings.Contains(lower, "<h2") ||
		strings.Contains(lower, "<h3") ||
		strings.Contains(lower, "section-title")
	return hasHeading || htmlInformationBlockCount(raw) >= 2
}

func buildPPTXHTMLDocument(manifest pptxPreviewManifest) string {
	pal := previewManifestPalette(manifest)
	var slides strings.Builder
	cover := preferredHTMLCover(manifest)
	slides.WriteString(`<section class="barq-pptx-slide barq-pptx-cover" data-slide-kind="cover"><div class="barq-pptx-canvas">` + wrapHTMLSlideShell(cover, true) + `</div></section>`)
	for _, slide := range manifest.Slides {
		body := preferredHTMLSlideMarkup(slide)
		slides.WriteString(`<section class="barq-pptx-slide" data-slide-kind="content" data-layout="` + html.EscapeString(slide.Layout) + `"><div class="barq-pptx-canvas">` + wrapHTMLSlideShell(body, false) + `</div></section>`)
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
<main class="barq-pptx-deck" data-theme="` + html.EscapeString(strings.ToLower(strings.TrimSpace(manifest.Theme))) + `" data-cover-style="` + html.EscapeString(strings.ToLower(strings.TrimSpace(manifest.DeckPlan.CoverStyle))) + `" data-composition="` + html.EscapeString(strings.ToLower(strings.TrimSpace(manifest.DeckPlan.Design.Composition))) + `" data-density="` + html.EscapeString(strings.ToLower(strings.TrimSpace(manifest.DeckPlan.Design.Density))) + `" data-accent-mode="` + html.EscapeString(strings.ToLower(strings.TrimSpace(manifest.DeckPlan.Design.AccentMode))) + `">` + slides.String() + `</main>
</body>
</html>`
}

func preferredHTMLCover(manifest pptxPreviewManifest) string {
	cover := strings.TrimSpace(manifest.DeckPlan.CoverHTML)
	if cover == "" {
		return fallbackHTMLCover(manifest)
	}
	if htmlCoverNeedsFallback(cover) {
		return fallbackHTMLCover(manifest)
	}
	return cover
}

func htmlCoverNeedsFallback(raw string) bool {
	lower := strings.ToLower(sanitizeHTMLMarkup(raw))
	if strings.Contains(lower, "cover-grid") ||
		strings.Contains(lower, "cover-stack") ||
		strings.Contains(lower, "cover-aside") {
		return false
	}
	signals := 0
	for _, token := range []string{
		"summary-strip",
		"summary-chip",
		"panel",
		"meta-row",
		"tag-row",
		"grid-2",
		"grid-3",
		"grid-4",
	} {
		if strings.Contains(lower, token) {
			signals++
		}
	}
	return signals < 2 || htmlInformationBlockCount(raw) < 5
}

func preferredHTMLSlideMarkup(slide pptxPreviewSlide) string {
	body := strings.TrimSpace(slide.HTML)
	if body == "" || htmlSlideNeedsFallback(body) {
		return fallbackHTMLSlideMarkup(slide)
	}
	return body
}

func htmlSlideNeedsFallback(raw string) bool {
	raw = sanitizeHTMLMarkup(raw)
	if raw == "" {
		return true
	}
	lower := strings.ToLower(raw)
	blocks := htmlInformationBlockCount(raw)
	textLen := len(htmlVisibleText(raw))
	hasDenseGrid := (strings.Contains(lower, "grid-2") ||
		strings.Contains(lower, "grid-3") ||
		strings.Contains(lower, "grid-4") ||
		strings.Contains(lower, "slide-grid") ||
		strings.Contains(lower, "compare-grid") ||
		strings.Contains(lower, "summary-strip")) && blocks >= 2
	hasLayoutSignal := strings.Contains(lower, "slide-grid") ||
		strings.Contains(lower, "grid-2") ||
		strings.Contains(lower, "grid-3") ||
		strings.Contains(lower, "grid-4") ||
		strings.Contains(lower, "summary-strip") ||
		strings.Contains(lower, "panel") ||
		strings.Contains(lower, "stat-card") ||
		strings.Contains(lower, "bullet-item") ||
		strings.Contains(lower, "timeline-row") ||
		strings.Contains(lower, "compare-col") ||
		strings.Contains(lower, "<table")
	if !hasLayoutSignal {
		return true
	}
	if hasDenseGrid {
		return false
	}
	if strings.Contains(lower, "display:flex") && blocks < 5 {
		return true
	}
	if textLen >= 220 && blocks < 4 {
		return true
	}
	if textLen >= 420 && blocks < 6 {
		return true
	}
	if strings.Contains(lower, "<h2") && blocks < 3 && textLen < 220 {
		return true
	}
	return false
}

func pptxHTMLGuardrailCSS() string {
	return `
.barq-pptx-cover .cover-shell,
.barq-pptx-slide .slide-shell,
.barq-pptx-slide .content-shell {
  align-content: start !important;
}
.barq-pptx-cover .cover-shell {
  padding-top: 40px !important;
  padding-bottom: 36px !important;
  grid-template-rows: auto !important;
  align-content: start !important;
}
.barq-pptx-cover .cover-grid {
  align-items: start !important;
  min-height: 100% !important;
}
.barq-pptx-cover .cover-shell--compose {
  display: grid !important;
  align-content: start !important;
}
.barq-pptx-cover .cover-shell--compose > div:first-child {
  display: grid !important;
  grid-template-columns: minmax(0, 1.14fr) minmax(380px, 0.86fr) !important;
  gap: 24px !important;
  min-height: 100% !important;
  align-items: start !important;
  padding: 0 !important;
}
.barq-pptx-cover .cover-shell--compose > div:first-child > div:first-child {
  display: grid !important;
  gap: 14px !important;
  align-content: start !important;
  padding: 0 !important;
  min-width: 0 !important;
}
.barq-pptx-cover .cover-shell--compose > div:first-child > div:last-child {
  display: grid !important;
  gap: 12px !important;
  align-content: start !important;
  justify-items: stretch !important;
  padding: 0 0 0 10px !important;
  min-width: 0 !important;
}
.barq-pptx-cover .cover-shell--compose .display-title {
  max-width: 780px !important;
  font-size: 74px !important;
}
.barq-pptx-cover .cover-shell--compose .lede {
  max-width: 700px !important;
  font-size: 24px !important;
}
.barq-pptx-cover .cover-shell--compose .panel-light {
  padding: 18px 20px !important;
  border-radius: 14px !important;
}
.barq-pptx-cover .cover-stack > .panel-light {
  min-height: 186px !important;
}
.barq-pptx-cover .cover-aside > .panel-light {
  min-height: 126px !important;
}
.barq-pptx-cover .cover-shell--compose .grid-3 {
  align-items: stretch !important;
  grid-auto-rows: 1fr !important;
}
.barq-pptx-cover .cover-shell--compose .grid-3 > .panel-light {
  min-height: 142px !important;
}
.barq-pptx-cover .meta-row {
  display: flex !important;
  flex-wrap: wrap !important;
  gap: 10px !important;
}
.barq-pptx-cover .tag {
  min-height: 36px !important;
  padding: 0 12px !important;
  font-size: 14px !important;
  letter-spacing: 0.08em !important;
  border-radius: 999px !important;
  border: 1px solid color-mix(in srgb, var(--accent) 18%, white) !important;
  background: color-mix(in srgb, var(--accent) 6%, white) !important;
}
.barq-pptx-slide .slide-shell,
.barq-pptx-slide .content-shell {
  display: grid !important;
  gap: 12px !important;
  padding: 36px 42px 30px !important;
  min-height: 100% !important;
  align-content: start !important;
  grid-template-rows: auto auto auto minmax(0, 1fr) !important;
}
.barq-pptx-slide .slide-root {
  display: grid !important;
  gap: 12px !important;
  padding: 0 !important;
  min-height: 100% !important;
  align-content: start !important;
  grid-template-rows: auto auto auto minmax(0, 1fr) !important;
}
.barq-pptx-slide .slide-root > .lede,
.barq-pptx-slide .slide-shell > .lede,
.barq-pptx-slide .content-shell > .lede {
  max-width: 1120px !important;
  margin-bottom: 0 !important;
  font-size: 28px !important;
  line-height: 1.34 !important;
}
.barq-pptx-slide .slide-root > :last-child,
.barq-pptx-slide .slide-shell > :last-child,
.barq-pptx-slide .content-shell > :last-child {
  align-self: stretch !important;
}
.barq-pptx-slide .slide-root > .rule,
.barq-pptx-slide .slide-shell > .rule,
.barq-pptx-slide .content-shell > .rule {
  width: 136px !important;
  margin: 0 !important;
}
.barq-pptx-slide .slide-root > .grid-2,
.barq-pptx-slide .slide-root > .grid-3,
.barq-pptx-slide .slide-root > .grid-4,
.barq-pptx-slide .slide-shell > .grid-2,
.barq-pptx-slide .slide-shell > .grid-3,
.barq-pptx-slide .slide-shell > .grid-4,
.barq-pptx-slide .content-shell > .grid-2,
.barq-pptx-slide .content-shell > .grid-3,
.barq-pptx-slide .content-shell > .grid-4 {
  align-self: stretch !important;
  grid-auto-rows: 1fr !important;
}
.barq-pptx-slide .summary-strip,
.barq-pptx-slide .grid-2,
.barq-pptx-slide .grid-3,
.barq-pptx-slide .grid-4,
.barq-pptx-slide .steps-flow {
  align-items: stretch !important;
  gap: 12px !important;
}
.barq-pptx-slide .slide-root > .grid-4,
.barq-pptx-slide .slide-shell > .grid-4,
.barq-pptx-slide .content-shell > .grid-4 {
  grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
  min-height: 468px !important;
}
.barq-pptx-slide .slide-root > .grid-3,
.barq-pptx-slide .slide-shell > .grid-3,
.barq-pptx-slide .content-shell > .grid-3 {
  min-height: 244px !important;
}
.barq-pptx-slide .slide-root > .grid-2,
.barq-pptx-slide .slide-shell > .grid-2,
.barq-pptx-slide .content-shell > .grid-2 {
  min-height: 612px !important;
}
.barq-pptx-slide .slide-root > .grid-2 > div,
.barq-pptx-slide .slide-shell > .grid-2 > div,
.barq-pptx-slide .content-shell > .grid-2 > div {
  display: flex !important;
  flex-direction: column !important;
  gap: 12px !important;
  min-height: 100% !important;
}
.barq-pptx-slide .slide-root > .grid-2 > div > .bullet-list,
.barq-pptx-slide .slide-shell > .grid-2 > div > .bullet-list,
.barq-pptx-slide .content-shell > .grid-2 > div > .bullet-list {
  display: grid !important;
  gap: 10px !important;
  grid-auto-rows: 1fr !important;
  flex: 1 1 auto !important;
}
.barq-pptx-slide .slide-root > .grid-2 > div > .bullet-list > .bullet-item,
.barq-pptx-slide .slide-shell > .grid-2 > div > .bullet-list > .bullet-item,
.barq-pptx-slide .content-shell > .grid-2 > div > .bullet-list > .bullet-item {
  min-height: 0 !important;
}
.barq-pptx-slide .slide-root > .grid-2 > div > .panel,
.barq-pptx-slide .slide-shell > .grid-2 > div > .panel,
.barq-pptx-slide .content-shell > .grid-2 > div > .panel,
.barq-pptx-slide .slide-root > .grid-2 > div > .panel-light,
.barq-pptx-slide .slide-shell > .grid-2 > div > .panel-light,
.barq-pptx-slide .content-shell > .grid-2 > div > .panel-light {
  flex: 1 1 0 !important;
  margin-bottom: 0 !important;
}
.barq-pptx-slide .slide-root > div[style*="grid-template-columns: 1fr 1fr 1fr"],
.barq-pptx-slide .slide-root > div[style*="grid-template-columns:1fr 1fr 1fr"],
.barq-pptx-slide .slide-shell > div[style*="grid-template-columns: 1fr 1fr 1fr"],
.barq-pptx-slide .slide-shell > div[style*="grid-template-columns:1fr 1fr 1fr"],
.barq-pptx-slide .content-shell > div[style*="grid-template-columns: 1fr 1fr 1fr"],
.barq-pptx-slide .content-shell > div[style*="grid-template-columns:1fr 1fr 1fr"] {
  display: grid !important;
  grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
  gap: 14px !important;
  min-height: 252px !important;
  align-self: stretch !important;
}
.barq-pptx-slide .slide-root > div[style*="grid-template-columns: 1fr 1fr 1fr"] > .stat-card,
.barq-pptx-slide .slide-root > div[style*="grid-template-columns:1fr 1fr 1fr"] > .stat-card,
.barq-pptx-slide .slide-shell > div[style*="grid-template-columns: 1fr 1fr 1fr"] > .stat-card,
.barq-pptx-slide .slide-shell > div[style*="grid-template-columns:1fr 1fr 1fr"] > .stat-card,
.barq-pptx-slide .content-shell > div[style*="grid-template-columns: 1fr 1fr 1fr"] > .stat-card,
.barq-pptx-slide .content-shell > div[style*="grid-template-columns:1fr 1fr 1fr"] > .stat-card {
  min-height: 100% !important;
}
.barq-pptx-slide .step-item,
.barq-pptx-slide .stat-card {
  min-height: 100% !important;
}
.barq-pptx-slide .grid-2 > .panel,
.barq-pptx-slide .grid-3 > .panel,
.barq-pptx-slide .grid-4 > .panel,
.barq-pptx-slide .grid-2 > .panel-light,
.barq-pptx-slide .grid-3 > .panel-light,
.barq-pptx-slide .grid-4 > .panel-light {
  min-height: 100% !important;
}
.barq-pptx-slide .grid-4 > .panel {
  min-height: 220px !important;
}
.barq-pptx-slide .grid-3 > .stat-card {
  min-height: 212px !important;
}
.barq-pptx-slide .grid-2 > .panel-light {
  min-height: 148px !important;
}
.barq-pptx-slide .panel,
.barq-pptx-slide .panel-light {
  padding: 18px 20px !important;
  border-radius: 12px !important;
}
.barq-pptx-slide .section-title {
  font-size: 54px !important;
  line-height: 1.04 !important;
}
.barq-pptx-slide .eyebrow {
  font-size: 17px !important;
  letter-spacing: 0.14em !important;
}
.barq-pptx-slide .panel {
  background: color-mix(in srgb, var(--card) 94%, white) !important;
  border: 1px solid color-mix(in srgb, var(--border) 78%, white) !important;
}
.barq-pptx-slide .panel-light {
  background: color-mix(in srgb, var(--accent) 6%, white) !important;
  border: 1px solid color-mix(in srgb, var(--accent) 18%, white) !important;
}
.barq-pptx-slide .stat-card {
  display: grid !important;
  gap: 5px !important;
  align-content: start !important;
  min-height: 178px !important;
  padding: 18px 18px !important;
  border-radius: 12px !important;
  background: color-mix(in srgb, var(--card) 96%, white) !important;
  border: 1px solid color-mix(in srgb, var(--border) 72%, white) !important;
  border-top: 6px solid var(--accent) !important;
  text-align: left !important;
}
.barq-pptx-slide .stat-value {
  font-size: 48px !important;
  line-height: 1 !important;
}
.barq-pptx-slide .stat-label {
  font-size: 18px !important;
  font-weight: 800 !important;
  line-height: 1.2 !important;
}
.barq-pptx-slide .stat-desc {
  font-size: 17px !important;
  line-height: 1.42 !important;
}
.barq-pptx-slide .bullet-list {
  display: grid !important;
  gap: 10px !important;
}
.barq-pptx-slide .bullet-item {
  display: grid !important;
  grid-template-columns: 14px minmax(0, 1fr) !important;
  gap: 10px !important;
  align-items: start !important;
  padding: 12px 14px 12px 16px !important;
  background: color-mix(in srgb, var(--card) 97%, white) !important;
  border: 1px solid color-mix(in srgb, var(--border) 72%, white) !important;
  border-left: 5px solid var(--accent) !important;
  border-radius: 10px !important;
}
.barq-pptx-slide .bullet-item:last-child {
  border-bottom: 1px solid color-mix(in srgb, var(--border) 72%, white) !important;
}
.barq-pptx-slide .bullet-item strong {
  display: block !important;
  margin-bottom: 4px !important;
  font-size: 20px !important;
}
.barq-pptx-slide .bullet-item,
.barq-pptx-slide .bullet-item > div,
.barq-pptx-slide .body-copy,
.barq-pptx-slide li,
.barq-pptx-slide td {
  font-size: 20px !important;
  line-height: 1.42 !important;
}
.barq-pptx-slide .bullet-marker {
  width: 10px !important;
  height: 10px !important;
  margin-top: 8px !important;
}
.barq-pptx-slide .timeline-row,
.barq-pptx-slide .compare-col {
  background: color-mix(in srgb, var(--card) 97%, white) !important;
  border: 1px solid color-mix(in srgb, var(--border) 72%, white) !important;
  border-radius: 14px !important;
}
.barq-pptx-slide .icon-circle {
  width: 42px !important;
  height: 42px !important;
  margin-bottom: 12px !important;
}
.barq-pptx-slide .card-title {
  font-size: 22px !important;
  line-height: 1.2 !important;
}
.barq-pptx-slide .card-desc {
  font-size: 18px !important;
  line-height: 1.48 !important;
}
.barq-pptx-slide .summary-strip {
  grid-template-columns: repeat(auto-fit, minmax(0, 1fr)) !important;
}
.barq-pptx-deck[data-density="balanced"] .barq-pptx-slide .slide-root,
.barq-pptx-deck[data-density="compact"] .barq-pptx-slide .slide-root,
.barq-pptx-deck[data-density="balanced"] .barq-pptx-slide .slide-shell,
.barq-pptx-deck[data-density="compact"] .barq-pptx-slide .slide-shell {
  padding: 32px 38px 28px !important;
}
.barq-pptx-deck[data-cover-style="editorial"] .barq-pptx-cover .cover-shell,
.barq-pptx-deck[data-cover-style="poster"] .barq-pptx-cover .cover-shell {
  padding-top: 34px !important;
  padding-bottom: 30px !important;
}
.barq-pptx-deck[data-theme="education"] .barq-pptx-slide .panel,
.barq-pptx-deck[data-theme="education"] .barq-pptx-slide .stat-card,
.barq-pptx-deck[data-theme="education"] .barq-pptx-slide .bullet-item {
  border-radius: 10px !important;
}
`
}

func wrapHTMLSlideShell(raw string, cover bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if cover && strings.Contains(lower, "cover-shell") {
		return raw
	}
	if !cover && (strings.Contains(lower, "slide-shell") || strings.Contains(lower, "content-shell")) {
		return raw
	}
	classes := "slide-shell slide content-shell"
	if cover {
		classes = "cover-shell slide"
		if !strings.Contains(lower, "cover-grid") && !strings.Contains(lower, "cover-stack") {
			classes += " cover-shell--compose"
		}
	}
	return `<div class="` + classes + `">` + raw + `</div>`
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
.cover-shell,
.slide-shell {
  min-height: 100%;
  width: 100%;
}
.cover-shell {
  display: grid;
  padding: 60px 68px 54px;
  gap: 20px;
  align-content: start;
}
.cover-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.14fr) minmax(340px, 0.86fr);
  gap: 26px;
  min-height: 100%;
}
.cover-stack,
.cover-aside {
  display: grid;
  align-content: center;
}
.cover-stack {
  gap: 18px;
}
.cover-aside {
  gap: 16px;
}
.slide-shell {
  display: grid;
  padding: 50px 58px 46px;
  gap: 16px;
  align-content: start;
}
.slide-head {
  display: grid;
  gap: 10px;
  max-width: 1120px;
}
.slide-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.74fr);
  gap: 18px;
  align-items: start;
}
.slide-main,
.slide-side {
  display: grid;
  gap: 14px;
  align-content: start;
}
.barq-pptx-slide h1, .barq-pptx-slide h2, .barq-pptx-slide h3, .barq-pptx-slide p { margin: 0; }
.barq-pptx-slide ul, .barq-pptx-slide ol { margin: 0; padding: 0; list-style: none; }
.barq-pptx-canvas > .cover-shell, .barq-pptx-canvas > .slide-shell, .barq-pptx-canvas > .content-shell { min-height: 100%; }
.eyebrow { font-size: 18px; font-weight: 700; letter-spacing: 0.12em; text-transform: uppercase; color: var(--muted); }
.display-title { font-size: 68px; line-height: 1.01; font-weight: 800; letter-spacing: -0.045em; }
.section-title { font-size: 46px; line-height: 1.06; font-weight: 800; letter-spacing: -0.035em; }
.lede { font-size: 26px; line-height: 1.32; color: var(--muted); }
.body-copy { font-size: 22px; line-height: 1.4; color: var(--text); }
.muted-copy { color: var(--muted); }
.rule { width: 160px; height: 5px; border-radius: 999px; background: var(--accent); }
.meta-row, .tag-row { display: flex; flex-wrap: wrap; gap: 10px; }
.stack-tight { display: grid; gap: 12px; }
.stack-regular { display: grid; gap: 16px; }
.summary-strip { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.summary-chip {
  display: grid;
  gap: 6px;
  padding: 12px 14px;
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
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
}
.panel {
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.10);
  padding: 18px 20px;
  border-radius: 18px;
}
.panel-light {
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border: 1px solid rgba(15,23,42,0.08);
  border-radius: 18px;
}
.grid-2, .grid-3, .grid-4 { display: grid; gap: 14px; }
.grid-2 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.grid-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.grid-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.stat-card { padding: 18px 20px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 6px solid var(--accent); border-radius: 18px; }
.stat-value { font-size: 42px; line-height: 1; font-weight: 800; color: var(--accent); margin-bottom: 6px; }
.stat-label { font-size: 17px; line-height: 1.2; font-weight: 800; margin-bottom: 6px; text-transform: uppercase; letter-spacing: 0.03em; }
.stat-desc { font-size: 17px; line-height: 1.38; color: #475569; }
.bullet-list { display: grid; gap: 12px; }
.bullet-item {
  padding: 14px 16px 14px 20px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 5px solid var(--accent);
  border-radius: 16px;
}
.bullet-item strong { display: block; margin-bottom: 6px; font-size: 22px; }
.steps-flow { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; align-items: stretch; }
.step-item {
  display: grid;
  gap: 8px;
  align-content: start;
  padding: 16px 15px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-top: 5px solid var(--accent);
  border-radius: 18px;
}
.step-num { font-size: 34px; line-height: 1; font-weight: 800; color: var(--accent); }
.step-title { font-size: 19px; line-height: 1.24; font-weight: 700; color: #0f172a; }
.step-desc { font-size: 16px; line-height: 1.46; color: #475569; }
.step-arrow { display: none; }
.timeline-list { display: grid; gap: 12px; }
.timeline-row {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
  padding: 14px 16px;
  background: rgba(255,255,255,0.96);
  color: #0f172a;
  border-left: 5px solid var(--accent);
  border-radius: 16px;
}
.timeline-date { font-size: 14px; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; color: var(--accent); }
.compare-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.compare-col { padding: 16px 18px; background: rgba(255,255,255,0.96); color: #0f172a; border-top: 6px solid var(--accent); border-radius: 18px; }
.compare-col h3 { font-size: 24px; margin-bottom: 8px; }
.compare-col li { font-size: 19px; line-height: 1.38; margin-top: 8px; }
table { width: 100%; border-collapse: collapse; background: rgba(255,255,255,0.98); color: #0f172a; }
thead th { background: rgba(15,23,42,0.08); font-size: 14px; font-weight: 800; text-transform: uppercase; letter-spacing: 0.08em; }
th, td { border: 1px solid rgba(15,23,42,0.10); padding: 10px 12px; text-align: left; vertical-align: top; font-size: 17px; line-height: 1.35; }
svg { display: block; max-width: 100%; max-height: 100%; }
`
}

func fallbackHTMLCover(manifest pptxPreviewManifest) string {
	title := html.EscapeString(firstNonEmpty(manifest.Title, "Presentation"))
	subtitle := strings.TrimSpace(manifest.Subtitle)
	subtitleHTML := ""
	if subtitle != "" {
		subtitleHTML = `<p class="lede" style="max-width:760px;">` + html.EscapeString(subtitle) + `</p>`
	}
	meta := []string{}
	if theme := strings.TrimSpace(manifest.Theme); theme != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(theme)+`</span>`)
	}
	if need := strings.TrimSpace(manifest.DeckPlan.DominantNeed); need != "" {
		meta = append(meta, `<span class="tag">`+html.EscapeString(need)+`</span>`)
	}
	if slideCount := len(manifest.Slides) + 1; slideCount > 0 {
		meta = append(meta, `<span class="tag">`+html.EscapeString(fmt.Sprintf("%d slides", slideCount))+`</span>`)
	}
	kicker := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Kicker), "Presentation"))
	narrative := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Narrative), strings.TrimSpace(manifest.DeckPlan.NarrativeArc), "Structured decision-ready narrative"))
	colorStory := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.ColorStory), "Deliberate contemporary system"))
	chapterCards := fallbackCoverChapterCards(manifest)
	composition := strings.ToLower(strings.TrimSpace(manifest.DeckPlan.Design.Composition))
	heroLayout := strings.ToLower(strings.TrimSpace(manifest.DeckPlan.Design.HeroLayout))
	switch {
	case composition == "stack" || heroLayout == "statement":
		return `<div style="position:absolute;left:0;top:0;right:0;height:12px;background:linear-gradient(90deg, var(--accent), var(--accent-2));"></div>
<div class="cover-shell" style="position:absolute;left:76px;top:74px;right:76px;bottom:70px;">
  <div class="cover-stack" style="gap:22px;align-content:start;max-width:1240px;">
    <div class="eyebrow">` + kicker + `</div>
    <h1 class="display-title" style="max-width:1080px;">` + title + `</h1>
    ` + subtitleHTML + `
    <div class="summary-strip" style="grid-template-columns:1.1fr 0.9fr 0.9fr;gap:12px;margin-top:8px;">
      <div class="summary-chip" style="background:rgba(255,255,255,0.76);border-color:rgba(15,23,42,0.08);">
        <span class="eyebrow">Narrative</span>
        <span class="body-copy">` + narrative + `</span>
      </div>
      <div class="summary-chip" style="background:rgba(255,255,255,0.76);border-color:rgba(15,23,42,0.08);">
        <span class="eyebrow">Audience</span>
        <span class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Audience), "Decision-makers")) + `</span>
      </div>
      <div class="summary-chip" style="background:rgba(255,255,255,0.76);border-color:rgba(15,23,42,0.08);">
        <span class="eyebrow">Mood</span>
        <span class="body-copy">` + colorStory + `</span>
      </div>
    </div>
  </div>
</div>`
	case composition == "mosaic" || heroLayout == "module":
		return `<div style="position:absolute;inset:0;background:
radial-gradient(circle at top left, rgba(45,106,79,0.09), transparent 32%),
linear-gradient(180deg, rgba(255,255,255,0.16), rgba(255,255,255,0)),
var(--bg);"></div>
<div class="cover-shell" style="position:absolute;left:74px;top:70px;right:74px;bottom:68px;">
  <div class="cover-grid" style="grid-template-columns:minmax(0,1.04fr) 1fr;gap:22px;align-items:stretch;">
    <div class="cover-stack" style="gap:18px;align-content:start;">
      <div class="eyebrow">` + kicker + `</div>
      <h1 class="display-title" style="max-width:760px;">` + title + `</h1>
      ` + subtitleHTML + `
      <div class="rule"></div>
      <div class="panel-light" style="padding:20px 22px;">
        <div class="eyebrow" style="margin-bottom:10px;">Narrative</div>
        <p class="body-copy" style="font-size:19px;line-height:1.48;">` + narrative + `</p>
      </div>
    </div>
    <div class="grid-2" style="grid-template-columns:1fr 1fr;grid-auto-rows:minmax(0,1fr);gap:14px;">
      <div class="panel-light"><span class="eyebrow">Audience</span><p class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Audience), "Decision-makers")) + `</p></div>
      <div class="panel-light"><span class="eyebrow">Theme</span><p class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Theme), "Presentation")) + `</p></div>
      <div class="panel-light"><span class="eyebrow">Mood</span><p class="body-copy">` + colorStory + `</p></div>
      <div class="panel-light"><span class="eyebrow">Subject</span><p class="body-copy">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Subject), "Presentation")) + `</p></div>
    </div>
  </div>
</div>`
	}
	return `<div style="position:absolute;inset:0;background:
radial-gradient(circle at top left, rgba(45,106,79,0.08), transparent 34%),
linear-gradient(180deg, rgba(255,255,255,0.24), rgba(255,255,255,0)),
var(--bg);"></div>
<div style="position:absolute;left:0;top:0;right:0;height:10px;background:linear-gradient(90deg, var(--accent), var(--accent-2));"></div>
<div class="cover-shell" style="position:absolute;left:60px;top:42px;right:60px;bottom:40px;">
  <div class="cover-grid" style="grid-template-columns:minmax(0,1.12fr) 390px;gap:20px;align-items:start;">
    <div class="cover-stack" style="align-content:start;gap:14px;padding-top:0;">
      <div class="eyebrow">` + kicker + `</div>
      <h1 class="display-title" style="max-width:860px;">` + title + `</h1>
      ` + subtitleHTML + `
      <div class="rule"></div>
      <div class="meta-row">` + strings.Join(meta, "") + `</div>
      <div class="panel-light" style="padding:20px 22px;">
        <div class="eyebrow" style="margin-bottom:10px;">Narrative</div>
        <p class="body-copy" style="font-size:20px;line-height:1.46;">` + narrative + `</p>
      </div>
      ` + chapterCards + `
    </div>
    <div class="cover-aside" style="align-content:start;gap:12px;padding-top:2px;">
      <div class="panel-light" style="padding:16px 18px;">
        <div class="eyebrow" style="margin-bottom:8px;">Audience</div>
        <p class="body-copy" style="font-size:17px;line-height:1.42;">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Audience), "Decision-makers")) + `</p>
      </div>
      <div class="panel-light" style="padding:16px 18px;">
        <div class="eyebrow" style="margin-bottom:8px;">Mood</div>
        <p class="body-copy" style="font-size:17px;line-height:1.42;">` + colorStory + `</p>
      </div>
      <div class="panel-light" style="padding:16px 18px;">
        <div class="eyebrow" style="margin-bottom:8px;">Focus</div>
        <p class="body-copy" style="font-size:17px;line-height:1.42;">` + html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.DominantNeed), strings.TrimSpace(manifest.Theme), "Subject framing")) + `</p>
      </div>
    </div>
  </div>
</div>`
}

func fallbackCoverChapterCards(manifest pptxPreviewManifest) string {
	if len(manifest.Slides) == 0 {
		return ""
	}
	cards := make([]string, 0, 3)
	limit := len(manifest.Slides)
	if limit > 3 {
		limit = 3
	}
	for i := 0; i < limit; i++ {
		cards = append(cards, `<div class="panel-light" style="padding:16px 18px;"><div class="eyebrow">`+
			html.EscapeString(fmt.Sprintf("%02d", i+1))+`</div><p class="body-copy" style="font-size:18px;line-height:1.34;">`+
			html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Slides[i].Heading), "Section"))+`</p></div>`)
	}
	extraCards := []struct {
		Label string
		Body  string
	}{
		{
			Label: "Need",
			Body:  firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.DominantNeed), strings.TrimSpace(manifest.Theme), "Clear framing"),
		},
		{
			Label: "Outcome",
			Body:  firstNonEmpty(strings.TrimSpace(manifest.Subtitle), strings.TrimSpace(manifest.DeckPlan.NarrativeArc), "Actionable takeaway"),
		},
		{
			Label: "Audience",
			Body:  firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Audience), "Decision-makers"),
		},
	}
	for _, extra := range extraCards {
		if len(cards) >= 3 {
			break
		}
		cards = append(cards, `<div class="panel-light" style="padding:16px 18px;"><div class="eyebrow">`+
			html.EscapeString(extra.Label)+`</div><p class="body-copy" style="font-size:18px;line-height:1.34;">`+
			html.EscapeString(extra.Body)+`</p></div>`)
	}
	return `<div class="grid-3" style="grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-top:6px;">` + strings.Join(cards, "") + `</div>`
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
