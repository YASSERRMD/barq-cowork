package tools

import (
	"fmt"
	"html"
	"math"
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
	htmlDensityBlockPattern = regexp.MustCompile(`(?is)(stat-card|bullet-item|timeline-row|compare-col|summary-chip|panel|note-card|roadmap-row|metric-band|tag-row|summary-strip|\bcard\b|card-body|card-title|list-group|list-group-item|\brow\b|\bcol-(?:auto|\d{1,2})\b|\bbadge\b|\bbi\b|data-bi=|data-bootstrap-icon=|<li\b|<tr\b|<i\b|<svg\b|<table\b)`)
	cssDangerPattern        = regexp.MustCompile(`(?is)@import|expression\s*\(|javascript:|behavior\s*:`)
	cssRuleBodyPattern      = regexp.MustCompile(`\{([^{}]+)\}`)
	cssPXTokenPattern       = regexp.MustCompile(`(\d+(?:\.\d+)?)px`)
)

// --- Sanitization helpers (still used by pptx_subject_plan.go for slide.HTML
// input sanitization; the preview renderer no longer trusts that content
// visually). ---

func sanitizeHTMLMarkup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlScriptTagPattern.ReplaceAllString(raw, "")
	raw = htmlDangerTagPattern.ReplaceAllString(raw, "")
	raw = htmlEventAttrPattern.ReplaceAllString(raw, "")
	raw = htmlJSHrefPattern.ReplaceAllString(raw, `$1="#"`)
	raw = stripEmojiGlyphs(raw)
	raw = normalizeInlineStyleDeclarations(raw)
	return strings.TrimSpace(raw)
}

func stripEmojiGlyphs(raw string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF:
			return -1
		case r >= 0x2600 && r <= 0x27BF:
			return -1
		case r == 0xFE0F || r == 0x200D:
			return -1
		default:
			return r
		}
	}, raw)
}

func sanitizeCSSMarkup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlStyleWrapper.ReplaceAllString(raw, "")
	raw = cssDangerPattern.ReplaceAllString(raw, "")
	raw = normalizeCSSRuleBodies(raw)
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

// --- HTML authoring readiness checks (still used by validation + tests;
// they no longer gate the preview render, which always uses structured
// fields). ---

func validatePlannedHTMLDeckSource(planned plannedPPTXPresentation) error {
	var issues []string
	for i, slide := range planned.Slides {
		if !plannedSlideHasFallbackHTMLContent(slide.Slide) &&
			!htmlSlideContentReady(slide.Slide.HTML) {
			issues = append(issues, fmt.Sprintf("slides[%d]", i))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(
			"write_pptx requires structured slide content; missing or invalid: %s",
			strings.Join(issues, ", "),
		)
	}
	return nil
}

func plannedSlideHasFallbackHTMLContent(slide pptxSlide) bool {
	return len(slide.Points) >= 2 ||
		len(slide.Stats) >= 2 ||
		len(slide.Steps) >= 2 ||
		len(slide.Cards) >= 2 ||
		len(slide.Timeline) >= 2 ||
		compareHasContent(slide.LeftColumn, slide.RightColumn) ||
		tableHasContent(slide.Table)
}

func previewSlideHasFallbackHTMLContent(slide pptxPreviewSlide) bool {
	return len(slide.Points) >= 2 ||
		len(slide.Stats) >= 2 ||
		len(slide.Steps) >= 2 ||
		len(slide.Cards) >= 2 ||
		len(slide.Timeline) >= 2 ||
		compareHasContent(slide.LeftColumn, slide.RightColumn) ||
		tableHasContent(slide.Table)
}

func compareHasContent(left, right *pptxCompareColumn) bool {
	return left != nil && right != nil && len(left.Points) >= 1 && len(right.Points) >= 1
}

func tableHasContent(table *pptxTableData) bool {
	return table != nil && len(table.Headers) > 0 && len(table.Rows) > 0
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
		strings.Contains(lower, "lead") ||
		strings.Contains(lower, "tag") ||
		strings.Contains(lower, "panel") ||
		strings.Contains(lower, "card") ||
		strings.Contains(lower, "badge") ||
		strings.Contains(lower, "summary-chip") ||
		strings.Contains(lower, `class="bi`) ||
		strings.Contains(lower, `class='bi`) ||
		strings.Contains(lower, "data-bi=") ||
		strings.Contains(lower, "data-bootstrap-icon=") ||
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

// -----------------------------------------------------------------------------
// New Reveal.js preview renderer. Drives layout from structured fields +
// parseComposition() so the preview geometry mirrors the PptxGenJS renderer.
// LLM-authored HTML (slide.HTML, DeckPlan.CoverHTML) is intentionally ignored.
// -----------------------------------------------------------------------------

// buildPPTXHTMLDocument emits a Reveal.js-format HTML document that visually
// mirrors the final .pptx output. The slide dimensions are fixed to 1280x720
// (16:9) to match the PptxGenJS renderer's logical canvas.
func buildPPTXHTMLDocument(manifest pptxPreviewManifest) string {
	pal := previewManifestPalette(manifest)
	title := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Title), "Presentation"))

	var body strings.Builder
	body.WriteString(revealCoverSection(manifest, pal))
	for _, slide := range manifest.Slides {
		body.WriteString(revealContentSection(slide, pal))
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + title + `</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reset.css">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reveal.css">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/theme/white.css">
<style>` + revealBaseCSS(pal) + `</style>
</head>
<body>
<div class="reveal">
<div class="slides">
` + body.String() + `
</div>
</div>
<script type="module">
import Reveal from 'https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reveal.esm.js';
const deck = new Reveal({
  width: 1280,
  height: 720,
  margin: 0,
  minScale: 0.2,
  maxScale: 2.0,
  hash: true,
  slideNumber: 'c/t',
  transition: 'slide',
  backgroundTransition: 'fade',
  controls: true,
  progress: true,
  center: false,
  plugins: [],
});
deck.initialize();
</script>
</body>
</html>`
}

// revealBaseCSS defines palette variables and utility classes used by the
// structured slide templates. All type sizes are tuned for 1280x720 slides.
func revealBaseCSS(pal pptxPalette) string {
	return `
.reveal .slides { text-align: left; }
.reveal .slides section { padding: 0 !important; font-size: 20px; }
.reveal, .reveal .slides, .reveal .slides section {
  --bg: ` + hexColor(pal.bg) + `;
  --card: ` + hexColor(pal.card) + `;
  --accent: ` + hexColor(pal.accent) + `;
  --accent-2: ` + hexColor(pal.accent2) + `;
  --text: ` + hexColor(pal.text) + `;
  --muted: ` + hexColor(pal.muted) + `;
  --border: ` + hexColor(pal.border) + `;
  --accent-rgba-12: ` + hexRGBA(pal.accent, 0.12) + `;
  --accent-rgba-18: ` + hexRGBA(pal.accent, 0.18) + `;
  --accent-rgba-06: ` + hexRGBA(pal.accent, 0.06) + `;
  --border-rgba-38: ` + hexRGBA(pal.border, 0.38) + `;
  font-family: "Helvetica Neue", Arial, sans-serif;
  color: var(--text);
}
.barq-slide {
  position: relative;
  width: 1280px;
  height: 720px;
  background: var(--bg);
  color: var(--text);
  overflow: hidden;
  box-sizing: border-box;
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 16px;
  padding: 36px 44px;
}
.barq-cover { padding: 56px 64px; }
.barq-cover-inner {
  display: grid;
  gap: 20px;
  align-content: start;
  max-width: 1100px;
}
.barq-kicker-chip {
  display: inline-flex;
  align-items: center;
  align-self: start;
  padding: 6px 14px;
  border-radius: 999px;
  background: var(--accent-rgba-12);
  color: var(--accent);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-size: 14px;
  border: 1px solid var(--accent);
}
.barq-cover-title {
  font-size: 64px;
  line-height: 1.03;
  font-weight: 800;
  letter-spacing: -0.03em;
  color: var(--text);
  margin: 0;
}
.barq-cover-subtitle {
  font-size: 26px;
  line-height: 1.32;
  color: var(--muted);
  margin: 0;
  max-width: 980px;
}
.barq-cover-meta {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 12px;
}
.barq-cover-meta-tag {
  padding: 6px 14px;
  border-radius: 999px;
  background: var(--card);
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  border: 1px solid var(--border);
}
.barq-cover-rule {
  width: 120px;
  height: 5px;
  border-radius: 999px;
  background: var(--accent);
}
.barq-slide-header {
  display: grid;
  gap: 6px;
  grid-template-columns: 1fr auto;
  align-items: end;
}
.barq-slide-eyebrow {
  font-size: 14px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--muted);
  font-weight: 700;
}
.barq-slide-title {
  font-size: 40px;
  line-height: 1.08;
  font-weight: 800;
  color: var(--text);
  margin: 0;
  letter-spacing: -0.02em;
}
.barq-slide-body { display: grid; min-height: 0; }
/* Composition utility classes */
.barq-split { display: grid; gap: 24px; min-height: 0; }
.barq-hero-first { display: grid; gap: 20px; min-height: 0; }
.barq-cols-1 { display: grid; gap: 16px; grid-template-columns: 1fr; }
.barq-cols-2 { display: grid; gap: 16px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.barq-cols-3 { display: grid; gap: 16px; grid-template-columns: repeat(3, minmax(0, 1fr)); }
.barq-horizontal { display: grid; gap: 16px; }
.barq-panel-filled {
  background: var(--accent);
  color: #ffffff;
  border-radius: 14px;
}
.barq-panel-filled .barq-panel-text,
.barq-panel-filled .barq-panel-kicker { color: #ffffff; }
.barq-panel-outline {
  background: transparent;
  color: var(--text);
  border: 2px solid var(--accent);
  border-radius: 14px;
}
.barq-panel-glass {
  background: var(--accent-rgba-12);
  color: var(--text);
  border: 1px solid var(--accent-rgba-18);
  border-radius: 14px;
}
.barq-panel { padding: 24px 26px; min-width: 0; display: grid; align-content: start; gap: 10px; }
.barq-panel-kicker {
  font-size: 14px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  font-weight: 700;
  color: var(--muted);
}
.barq-panel-lead {
  font-size: 28px;
  line-height: 1.24;
  font-weight: 700;
  color: var(--text);
}
.barq-panel-text {
  font-size: 20px;
  line-height: 1.46;
  color: var(--text);
}
.barq-accent-rail::before {
  content: "";
  position: absolute;
  top: 0; left: 0;
  width: 8px;
  height: 100%;
  background: var(--accent);
}
.barq-accent-band::before {
  content: "";
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 6px;
  background: var(--accent);
}
.barq-accent-chip {
  position: absolute;
  top: 22px;
  right: 26px;
  width: 38px;
  height: 38px;
  border-radius: 999px;
  background: var(--accent);
  box-shadow: 0 6px 20px rgba(0,0,0,0.12);
}
.barq-accent-glow {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(60% 40% at 20% 10%, var(--accent-rgba-18), transparent 70%);
}
/* Stat tiles */
.barq-stat-tile {
  display: grid;
  gap: 6px;
  align-content: start;
  padding: 20px 22px;
  background: var(--card);
  border: 1px solid var(--border);
  border-top: 6px solid var(--accent);
  border-radius: 14px;
  color: var(--text);
  min-width: 0;
}
.barq-stat-value {
  font-size: 44px;
  line-height: 1;
  font-weight: 800;
  color: var(--accent);
}
.barq-stat-value-hero {
  font-size: 76px;
  font-weight: 800;
  line-height: 1;
  color: inherit;
}
.barq-stat-label {
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted);
}
.barq-stat-label-hero { color: inherit; opacity: 0.85; }
.barq-stat-desc {
  font-size: 18px;
  line-height: 1.42;
  color: var(--muted);
}
.barq-stat-desc-hero { color: inherit; opacity: 0.9; }
/* Bullet point rows */
.barq-points { display: grid; gap: 12px; margin: 0; padding: 0; list-style: none; min-width: 0; }
.barq-point {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 12px 16px;
  background: var(--card);
  border: 1px solid var(--border);
  border-left: 6px solid var(--accent);
  border-radius: 12px;
  color: var(--text);
  min-width: 0;
}
.barq-point-dot {
  width: 10px;
  height: 10px;
  margin-top: 8px;
  border-radius: 999px;
  background: var(--accent);
}
.barq-point-title {
  font-size: 20px;
  line-height: 1.28;
  font-weight: 700;
  color: var(--text);
}
.barq-point-desc {
  margin-top: 4px;
  font-size: 18px;
  line-height: 1.44;
  color: var(--muted);
}
/* Cards */
.barq-card {
  padding: 20px 22px;
  background: var(--card);
  border: 1px solid var(--border);
  border-top: 6px solid var(--accent);
  border-radius: 14px;
  display: grid;
  gap: 10px;
  align-content: start;
  min-width: 0;
}
.barq-card-title {
  font-size: 22px;
  line-height: 1.2;
  font-weight: 700;
  color: var(--text);
}
.barq-card-desc {
  font-size: 18px;
  line-height: 1.44;
  color: var(--muted);
}
.barq-card-horizontal {
  display: grid;
  grid-template-columns: 68px 200px 1fr;
  gap: 16px;
  align-items: center;
  padding: 16px 20px;
  background: var(--card);
  border: 1px solid var(--border);
  border-left: 6px solid var(--accent);
  border-radius: 14px;
  min-width: 0;
}
.barq-card-icon {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  background: var(--accent-rgba-12);
  display: grid;
  place-items: center;
  color: var(--accent);
  font-weight: 800;
  font-size: 22px;
}
/* Steps */
.barq-steps-vertical { display: grid; gap: 12px; }
.barq-step-row {
  display: grid;
  grid-template-columns: 64px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 20px;
  background: var(--card);
  border: 1px solid var(--border);
  border-left: 6px solid var(--accent);
  border-radius: 14px;
  min-width: 0;
}
.barq-step-num {
  font-size: 26px;
  font-weight: 800;
  color: var(--accent);
}
.barq-step-title {
  font-size: 20px;
  line-height: 1.22;
  font-weight: 700;
  color: var(--text);
}
.barq-step-desc {
  font-size: 17px;
  line-height: 1.4;
  color: var(--muted);
  margin-top: 4px;
}
.barq-step-chip {
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted);
}
.barq-steps-horizontal {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(0, 1fr);
  gap: 12px;
  position: relative;
}
.barq-steps-horizontal::before {
  content: "";
  position: absolute;
  top: 30px;
  left: 4%;
  right: 4%;
  height: 3px;
  background: var(--border-rgba-38);
  z-index: 0;
}
.barq-step-node {
  position: relative;
  display: grid;
  gap: 8px;
  justify-items: center;
  text-align: center;
  padding: 0 6px;
  z-index: 1;
}
.barq-step-node-circle {
  width: 60px;
  height: 60px;
  border-radius: 999px;
  background: var(--accent);
  color: #ffffff;
  display: grid;
  place-items: center;
  font-size: 22px;
  font-weight: 800;
}
.barq-step-node-card {
  padding: 12px 14px;
  background: var(--card);
  border: 1px solid var(--border);
  border-top: 4px solid var(--accent);
  border-radius: 12px;
  display: grid;
  gap: 6px;
  text-align: left;
  width: 100%;
  min-width: 0;
}
/* Timeline */
.barq-timeline { display: grid; gap: 12px; }
.barq-timeline-row {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 16px;
  align-items: start;
  padding: 14px 18px;
  background: var(--card);
  border: 1px solid var(--border);
  border-left: 6px solid var(--accent);
  border-radius: 12px;
}
.barq-timeline-date {
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent);
}
.barq-timeline-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
  line-height: 1.24;
}
.barq-timeline-desc {
  margin-top: 4px;
  font-size: 17px;
  line-height: 1.42;
  color: var(--muted);
}
/* Compare */
.barq-compare {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  min-height: 0;
}
.barq-compare-col {
  padding: 20px 22px;
  background: var(--card);
  border: 1px solid var(--border);
  border-top: 6px solid var(--accent);
  border-radius: 14px;
  display: grid;
  gap: 10px;
  align-content: start;
}
.barq-compare-heading {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
}
.barq-compare-list { margin: 0; padding: 0; list-style: none; display: grid; gap: 8px; }
.barq-compare-list li {
  font-size: 18px;
  line-height: 1.42;
  color: var(--text);
  padding-left: 20px;
  position: relative;
}
.barq-compare-list li::before {
  content: "";
  position: absolute;
  left: 0;
  top: 10px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--accent);
}
/* Table */
.barq-table-wrap {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 14px;
  overflow: hidden;
}
.barq-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 17px;
  line-height: 1.4;
}
.barq-table th, .barq-table td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
  color: var(--text);
}
.barq-table thead th {
  background: var(--accent-rgba-06);
  color: var(--text);
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  font-size: 14px;
}
.barq-table tbody tr:last-child th, .barq-table tbody tr:last-child td { border-bottom: 0; }
/* Chart */
.barq-chart-placeholder {
  display: grid;
  place-items: center;
  background: var(--card);
  border: 1px dashed var(--border);
  border-radius: 14px;
  padding: 28px;
  color: var(--muted);
  font-size: 18px;
  min-height: 360px;
}
.barq-chart-bars {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(0, 1fr);
  gap: 14px;
  align-items: end;
  height: 100%;
}
.barq-chart-bar {
  display: grid;
  grid-template-rows: 1fr auto;
  gap: 6px;
  height: 100%;
}
.barq-chart-bar-fill {
  align-self: end;
  background: linear-gradient(180deg, var(--accent), var(--accent-2));
  border-radius: 8px 8px 0 0;
  min-height: 8px;
}
.barq-chart-bar-label {
  font-size: 13px;
  color: var(--muted);
  text-align: center;
}
`
}

// -----------------------------------------------------------------------------
// Cover
// -----------------------------------------------------------------------------

func revealCoverSection(manifest pptxPreviewManifest, pal pptxPalette) string {
	_ = pal
	title := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.Title), "Presentation"))
	subtitle := strings.TrimSpace(manifest.Subtitle)
	kicker := strings.TrimSpace(manifest.DeckPlan.Kicker)
	colorStory := strings.TrimSpace(manifest.DeckPlan.ColorStory)

	var inner strings.Builder
	if kicker != "" {
		inner.WriteString(`<span class="barq-kicker-chip">` + html.EscapeString(kicker) + `</span>`)
	}
	inner.WriteString(`<h1 class="barq-cover-title">` + title + `</h1>`)
	if subtitle != "" {
		inner.WriteString(`<p class="barq-cover-subtitle">` + html.EscapeString(subtitle) + `</p>`)
	}
	inner.WriteString(`<div class="barq-cover-rule"></div>`)

	var meta []string
	if audience := strings.TrimSpace(manifest.DeckPlan.Audience); audience != "" {
		meta = append(meta, `<span class="barq-cover-meta-tag">For `+html.EscapeString(audience)+`</span>`)
	}
	if theme := strings.TrimSpace(manifest.Theme); theme != "" {
		meta = append(meta, `<span class="barq-cover-meta-tag">`+html.EscapeString(theme)+`</span>`)
	}
	if colorStory != "" {
		meta = append(meta, `<span class="barq-cover-meta-tag">`+html.EscapeString(colorStory)+`</span>`)
	}
	if len(meta) > 0 {
		inner.WriteString(`<div class="barq-cover-meta">` + strings.Join(meta, "") + `</div>`)
	}

	return `<section data-slide-kind="cover">
  <div class="barq-slide barq-cover">
    <div class="barq-cover-inner">` + inner.String() + `</div>
  </div>
</section>
`
}

// -----------------------------------------------------------------------------
// Content slides
// -----------------------------------------------------------------------------

func revealContentSection(slide pptxPreviewSlide, pal pptxPalette) string {
	cp := parseComposition(slide.Design)
	body := revealSlideBody(slide, cp, pal)

	accentExtras := ""
	bodyClasses := "barq-slide"
	switch {
	case cp.AccentBand:
		bodyClasses += " barq-accent-band"
	case cp.AccentChip:
		accentExtras = `<div class="barq-accent-chip"></div>`
	case cp.AccentGlow:
		accentExtras = `<div class="barq-accent-glow"></div>`
	default:
		bodyClasses += " barq-accent-rail"
	}

	header := revealSlideHeader(slide)

	return `<section data-slide-kind="content" data-layout="` + html.EscapeString(slide.Layout) + `">
  <div class="` + bodyClasses + `">
    ` + accentExtras + `
    ` + header + `
    <div class="barq-slide-body">` + body + `</div>
  </div>
</section>
`
}

func revealSlideHeader(slide pptxPreviewSlide) string {
	title := html.EscapeString(firstNonEmpty(strings.TrimSpace(slide.Heading), "Slide"))
	eyebrow := strings.TrimSpace(slide.Purpose)
	var sb strings.Builder
	sb.WriteString(`<div class="barq-slide-header"><div>`)
	if eyebrow != "" {
		sb.WriteString(`<div class="barq-slide-eyebrow">` + html.EscapeString(eyebrow) + `</div>`)
	}
	sb.WriteString(`<h2 class="barq-slide-title">` + title + `</h2>`)
	sb.WriteString(`</div><div class="barq-slide-number">` + html.EscapeString(strconv.Itoa(slide.Number)) + `</div></div>`)
	return sb.String()
}

func revealSlideBody(slide pptxPreviewSlide, cp compositionParams, pal pptxPalette) string {
	switch strings.ToLower(strings.TrimSpace(slide.Layout)) {
	case "stats":
		return revealStats(slide, cp, pal)
	case "cards":
		return revealCards(slide, cp, pal)
	case "steps":
		return revealSteps(slide, cp, pal)
	case "timeline":
		return revealTimeline(slide, pal)
	case "compare":
		return revealCompare(slide, pal)
	case "table":
		return revealTable(slide, pal)
	case "chart":
		return revealChart(slide, pal)
	default:
		return revealBullets(slide, cp, pal)
	}
}

// --- renderStats port ---
func revealStats(slide pptxPreviewSlide, cp compositionParams, pal pptxPalette) string {
	stats := slide.Stats
	if len(stats) == 0 {
		return revealBullets(slide, cp, pal)
	}
	if cp.SplitRatio > 0 && len(stats) >= 2 {
		hero := stats[0]
		rest := stats[1:]
		if len(rest) > 3 {
			rest = rest[:3]
		}
		leftPct := cp.SplitRatio * 100
		rightPct := 100 - leftPct
		panelClass := panelClassFor(cp)

		var heroBody strings.Builder
		heroBody.WriteString(`<div class="barq-stat-value barq-stat-value-hero">` + html.EscapeString(hero.Value) + `</div>`)
		heroBody.WriteString(`<div class="barq-stat-label barq-stat-label-hero">` + html.EscapeString(hero.Label) + `</div>`)
		if d := strings.TrimSpace(hero.Desc); d != "" {
			heroBody.WriteString(`<div class="barq-stat-desc barq-stat-desc-hero">` + html.EscapeString(d) + `</div>`)
		}

		var tiles strings.Builder
		for _, s := range rest {
			tiles.WriteString(renderStatTile(s))
		}

		return `<div class="barq-split" style="` + densityStyle(cp, "gap") + `grid-template-columns: ` + floatPct(leftPct) + ` ` + floatPct(rightPct) + `;">
  <div class="` + panelClass + ` barq-panel" style="` + densityPanelStyle(cp) + `">` + heroBody.String() + `</div>
  <div class="barq-cols-1" style="gap: 12px;">` + tiles.String() + `</div>
</div>`
	}

	// DEFAULT GRID: 2-col or 4-up
	cols := 2
	if len(stats) == 1 {
		cols = 1
	}
	if len(stats) >= 3 {
		cols = 2 // TS caps columns at 2, stacks to rows of 2
	}
	gridClass := gridColsClass(cols)
	var tiles strings.Builder
	for _, s := range stats {
		if len(tiles.String()) > 0 {
			// keep appending
		}
		tiles.WriteString(renderStatTile(s))
	}
	_ = gridClass
	return `<div class="` + gridClass + `" style="` + densityStyle(cp, "gap") + `">` + tiles.String() + `</div>`
}

func renderStatTile(s pptxStat) string {
	var sb strings.Builder
	sb.WriteString(`<div class="barq-stat-tile">`)
	sb.WriteString(`<div class="barq-stat-value">` + html.EscapeString(s.Value) + `</div>`)
	sb.WriteString(`<div class="barq-stat-label">` + html.EscapeString(s.Label) + `</div>`)
	if d := strings.TrimSpace(s.Desc); d != "" {
		sb.WriteString(`<div class="barq-stat-desc">` + html.EscapeString(d) + `</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// --- renderBullets port ---
func revealBullets(slide pptxPreviewSlide, cp compositionParams, pal pptxPalette) string {
	_ = pal
	points := slide.Points
	if len(points) > 8 {
		points = points[:8]
	}

	if cp.SplitRatio > 0 {
		leftPct := cp.SplitRatio * 100
		rightPct := 100 - leftPct
		lead := buildBulletsLead(slide, points)
		kicker := strings.TrimSpace(slide.Purpose)
		if kicker == "" {
			kicker = "Focus"
		}
		panelClass := panelClassFor(cp)

		var pts strings.Builder
		pts.WriteString(`<ul class="barq-points" style="` + densityStyle(cp, "gap") + `">`)
		for _, p := range points {
			pts.WriteString(renderPointListItem(p))
		}
		pts.WriteString(`</ul>`)

		cols := 1
		if cp.Columns >= 2 {
			cols = 2
		}
		if cols == 2 && len(points) >= 4 {
			left, right := splitEvenOdd(points)
			var leftList strings.Builder
			var rightList strings.Builder
			leftList.WriteString(`<ul class="barq-points">`)
			for _, p := range left {
				leftList.WriteString(renderPointListItem(p))
			}
			leftList.WriteString(`</ul>`)
			rightList.WriteString(`<ul class="barq-points">`)
			for _, p := range right {
				rightList.WriteString(renderPointListItem(p))
			}
			rightList.WriteString(`</ul>`)
			pts.Reset()
			pts.WriteString(`<div class="barq-cols-2" style="` + densityStyle(cp, "gap") + `">` + leftList.String() + rightList.String() + `</div>`)
		}

		return `<div class="barq-split" style="grid-template-columns: ` + floatPct(leftPct) + ` ` + floatPct(rightPct) + `; ` + densityStyle(cp, "gap") + `">
  <div class="` + panelClass + ` barq-panel" style="` + densityPanelStyle(cp) + `">
    <div class="barq-panel-kicker">` + html.EscapeString(kicker) + `</div>
    <div class="barq-panel-lead">` + html.EscapeString(lead) + `</div>
  </div>
  <div>` + pts.String() + `</div>
</div>`
	}

	if cp.HeroFirst && len(points) > 0 {
		lead := buildBulletsLead(slide, points)
		kicker := strings.TrimSpace(slide.Purpose)
		if kicker == "" {
			kicker = "Focus"
		}
		cols := 1
		if cp.Columns >= 2 || len(points) >= 4 {
			cols = 2
		}

		var list strings.Builder
		if cols == 2 {
			left, right := splitEvenOdd(points)
			list.WriteString(`<div class="barq-cols-2" style="` + densityStyle(cp, "gap") + `">`)
			list.WriteString(`<ul class="barq-points">`)
			for _, p := range left {
				list.WriteString(renderPointListItem(p))
			}
			list.WriteString(`</ul>`)
			list.WriteString(`<ul class="barq-points">`)
			for _, p := range right {
				list.WriteString(renderPointListItem(p))
			}
			list.WriteString(`</ul>`)
			list.WriteString(`</div>`)
		} else {
			list.WriteString(`<ul class="barq-points" style="` + densityStyle(cp, "gap") + `">`)
			for _, p := range points {
				list.WriteString(renderPointListItem(p))
			}
			list.WriteString(`</ul>`)
		}

		return `<div class="barq-hero-first" style="grid-template-rows: auto 1fr; ` + densityStyle(cp, "gap") + `">
  <div class="barq-panel-glass barq-panel" style="` + densityPanelStyle(cp) + `">
    <div class="barq-panel-kicker">` + html.EscapeString(kicker) + `</div>
    <div class="barq-panel-lead">` + html.EscapeString(lead) + `</div>
  </div>
  ` + list.String() + `
</div>`
	}

	// DEFAULT STACK — optional summary strip of stats, then points
	var sb strings.Builder
	stats := slide.Stats
	if len(stats) > 3 {
		stats = stats[:3]
	}
	if len(stats) > 0 {
		sb.WriteString(`<div class="` + gridColsClass(len(stats)) + `" style="margin-bottom: 16px;">`)
		for _, s := range stats {
			sb.WriteString(renderStatTile(s))
		}
		sb.WriteString(`</div>`)
	}
	cols := 1
	if len(points) >= 4 {
		cols = 2
	}
	if cols == 2 {
		left, right := splitEvenOdd(points)
		sb.WriteString(`<div class="barq-cols-2" style="` + densityStyle(cp, "gap") + `">`)
		sb.WriteString(`<ul class="barq-points">`)
		for _, p := range left {
			sb.WriteString(renderPointListItem(p))
		}
		sb.WriteString(`</ul>`)
		sb.WriteString(`<ul class="barq-points">`)
		for _, p := range right {
			sb.WriteString(renderPointListItem(p))
		}
		sb.WriteString(`</ul>`)
		sb.WriteString(`</div>`)
	} else {
		sb.WriteString(`<ul class="barq-points" style="` + densityStyle(cp, "gap") + `">`)
		for _, p := range points {
			sb.WriteString(renderPointListItem(p))
		}
		sb.WriteString(`</ul>`)
	}
	return sb.String()
}

func renderPointListItem(raw string) string {
	title, desc := splitCardText(raw)
	var sb strings.Builder
	sb.WriteString(`<li class="barq-point"><span class="barq-point-dot"></span><div>`)
	sb.WriteString(`<div class="barq-point-title">` + html.EscapeString(title) + `</div>`)
	if desc != "" {
		sb.WriteString(`<div class="barq-point-desc">` + html.EscapeString(desc) + `</div>`)
	}
	sb.WriteString(`</div></li>`)
	return sb.String()
}

// --- renderCards port ---
func revealCards(slide pptxPreviewSlide, cp compositionParams, pal pptxPalette) string {
	_ = pal
	cards := slide.Cards
	if len(cards) == 0 {
		return revealBullets(slide, cp, pal)
	}
	if len(cards) > 6 {
		cards = cards[:6]
	}

	if cp.Horizontal {
		var sb strings.Builder
		sb.WriteString(`<div class="barq-horizontal" style="` + densityStyle(cp, "gap") + `">`)
		for _, c := range cards {
			sb.WriteString(renderHorizontalCard(c))
		}
		sb.WriteString(`</div>`)
		return sb.String()
	}

	cols := cp.Columns
	switch {
	case cols < 2:
		if len(cards) >= 3 {
			cols = 3
		} else {
			cols = 2
		}
	case cols > 3:
		cols = 3
	}
	gridClass := gridColsClass(cols)
	var sb strings.Builder
	sb.WriteString(`<div class="` + gridClass + `" style="` + densityStyle(cp, "gap") + `">`)
	for _, c := range cards {
		sb.WriteString(renderGridCard(c))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func renderGridCard(c pptxCard) string {
	title := html.EscapeString(strings.TrimSpace(c.Title))
	desc := html.EscapeString(strings.TrimSpace(c.Desc))
	icon := html.EscapeString(cardIconLabel(c))
	var sb strings.Builder
	sb.WriteString(`<div class="barq-card">`)
	if icon != "" {
		sb.WriteString(`<div class="barq-card-icon">` + icon + `</div>`)
	}
	sb.WriteString(`<div class="barq-card-title">` + title + `</div>`)
	if desc != "" {
		sb.WriteString(`<div class="barq-card-desc">` + desc + `</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func renderHorizontalCard(c pptxCard) string {
	title := html.EscapeString(strings.TrimSpace(c.Title))
	desc := html.EscapeString(strings.TrimSpace(c.Desc))
	icon := html.EscapeString(cardIconLabel(c))
	var sb strings.Builder
	sb.WriteString(`<div class="barq-card-horizontal">`)
	sb.WriteString(`<div class="barq-card-icon">` + icon + `</div>`)
	sb.WriteString(`<div class="barq-card-title">` + title + `</div>`)
	sb.WriteString(`<div class="barq-card-desc">` + desc + `</div>`)
	sb.WriteString(`</div>`)
	return sb.String()
}

func cardIconLabel(c pptxCard) string {
	token := strings.TrimSpace(c.Icon)
	if token == "" {
		token = strings.TrimSpace(c.Title)
	}
	if token == "" {
		return ""
	}
	// Use first letter as a simple glyph; real icon rendering is PPTX-side.
	runes := []rune(token)
	return strings.ToUpper(string(runes[0:1]))
}

// --- renderSteps port ---
func revealSteps(slide pptxPreviewSlide, cp compositionParams, pal pptxPalette) string {
	_ = pal
	steps := slide.Steps
	if len(steps) == 0 {
		steps = slide.Points
	}
	if len(steps) > 6 {
		steps = steps[:6]
	}

	if cp.Horizontal {
		var sb strings.Builder
		sb.WriteString(`<div class="barq-steps-horizontal" style="` + densityStyle(cp, "gap") + `">`)
		for i, step := range steps {
			title, desc := splitCardText(step)
			sb.WriteString(`<div class="barq-step-node">`)
			sb.WriteString(`<div class="barq-step-node-circle">` + strconv.Itoa(i+1) + `</div>`)
			sb.WriteString(`<div class="barq-step-node-card">`)
			sb.WriteString(`<div class="barq-step-title">` + html.EscapeString(title) + `</div>`)
			if desc != "" {
				sb.WriteString(`<div class="barq-step-desc">` + html.EscapeString(desc) + `</div>`)
			}
			sb.WriteString(`</div></div>`)
		}
		sb.WriteString(`</div>`)
		return sb.String()
	}

	// DEFAULT VERTICAL ROWS
	var sb strings.Builder
	sb.WriteString(`<div class="barq-steps-vertical" style="` + densityStyle(cp, "gap") + `">`)
	for i, step := range steps {
		title, desc := splitCardText(step)
		sb.WriteString(`<div class="barq-step-row">`)
		sb.WriteString(`<div class="barq-step-num">` + fmt.Sprintf("%02d", i+1) + `</div>`)
		sb.WriteString(`<div>`)
		sb.WriteString(`<div class="barq-step-title">` + html.EscapeString(title) + `</div>`)
		if desc != "" {
			sb.WriteString(`<div class="barq-step-desc">` + html.EscapeString(desc) + `</div>`)
		}
		sb.WriteString(`</div>`)
		sb.WriteString(`<div class="barq-step-chip">Step ` + strconv.Itoa(i+1) + `</div>`)
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// --- Timeline, compare, table, chart (natural layouts, not composition-driven) ---

func revealTimeline(slide pptxPreviewSlide, pal pptxPalette) string {
	_ = pal
	var sb strings.Builder
	sb.WriteString(`<div class="barq-timeline">`)
	for _, item := range slide.Timeline {
		sb.WriteString(`<div class="barq-timeline-row">`)
		sb.WriteString(`<div class="barq-timeline-date">` + html.EscapeString(item.Date) + `</div>`)
		sb.WriteString(`<div>`)
		sb.WriteString(`<div class="barq-timeline-title">` + html.EscapeString(item.Title) + `</div>`)
		if d := strings.TrimSpace(item.Desc); d != "" {
			sb.WriteString(`<div class="barq-timeline-desc">` + html.EscapeString(d) + `</div>`)
		}
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func revealCompare(slide pptxPreviewSlide, pal pptxPalette) string {
	_ = pal
	left := slide.LeftColumn
	right := slide.RightColumn
	var sb strings.Builder
	sb.WriteString(`<div class="barq-compare">`)
	sb.WriteString(renderCompareColumn(left))
	sb.WriteString(renderCompareColumn(right))
	sb.WriteString(`</div>`)
	return sb.String()
}

func renderCompareColumn(col *pptxCompareColumn) string {
	if col == nil {
		return `<div class="barq-compare-col"><div class="barq-compare-heading">—</div></div>`
	}
	var sb strings.Builder
	sb.WriteString(`<div class="barq-compare-col">`)
	sb.WriteString(`<div class="barq-compare-heading">` + html.EscapeString(col.Heading) + `</div>`)
	sb.WriteString(`<ul class="barq-compare-list">`)
	for _, p := range col.Points {
		sb.WriteString(`<li>` + html.EscapeString(p) + `</li>`)
	}
	sb.WriteString(`</ul>`)
	sb.WriteString(`</div>`)
	return sb.String()
}

func revealTable(slide pptxPreviewSlide, pal pptxPalette) string {
	_ = pal
	t := slide.Table
	if t == nil {
		return `<div class="barq-chart-placeholder">No tabular data provided.</div>`
	}
	var sb strings.Builder
	sb.WriteString(`<div class="barq-table-wrap"><table class="barq-table"><thead><tr>`)
	for _, h := range t.Headers {
		sb.WriteString(`<th>` + html.EscapeString(h) + `</th>`)
	}
	sb.WriteString(`</tr></thead><tbody>`)
	for _, row := range t.Rows {
		sb.WriteString(`<tr>`)
		for _, cell := range row {
			sb.WriteString(`<td>` + html.EscapeString(cell) + `</td>`)
		}
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody></table></div>`)
	return sb.String()
}

func revealChart(slide pptxPreviewSlide, pal pptxPalette) string {
	_ = pal
	if len(slide.ChartSeries) == 0 || len(slide.ChartCategories) == 0 {
		label := strings.TrimSpace(slide.ChartType)
		if label == "" {
			label = "Chart"
		}
		return `<div class="barq-chart-placeholder">` + html.EscapeString(label) + ` — rendered in PPTX</div>`
	}
	// Simple first-series bar preview.
	series := slide.ChartSeries[0]
	maxVal := 0.0
	for _, v := range series.Values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal <= 0 {
		maxVal = 1
	}
	var bars strings.Builder
	bars.WriteString(`<div class="barq-chart-bars">`)
	for i, cat := range slide.ChartCategories {
		value := 0.0
		if i < len(series.Values) {
			value = series.Values[i]
		}
		pct := value / maxVal * 100
		bars.WriteString(`<div class="barq-chart-bar">`)
		bars.WriteString(`<div class="barq-chart-bar-fill" style="height: ` + strconv.FormatFloat(pct, 'f', 1, 64) + `%;"></div>`)
		bars.WriteString(`<div class="barq-chart-bar-label">` + html.EscapeString(cat) + `</div>`)
		bars.WriteString(`</div>`)
	}
	bars.WriteString(`</div>`)
	return `<div class="barq-chart-placeholder" style="padding: 20px;">` + bars.String() + `</div>`
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// buildBulletsLead constructs the hero/lead copy used by split and hero-first
// bullet layouts.
func buildBulletsLead(slide pptxPreviewSlide, points []string) string {
	if s := strings.TrimSpace(slide.Visual); s != "" {
		return s
	}
	if s := strings.TrimSpace(slide.Purpose); s != "" {
		return s
	}
	if len(points) > 0 {
		first := points[0]
		if t, d := splitCardText(first); d != "" {
			return t + ". " + d
		}
		return first
	}
	return strings.TrimSpace(slide.Heading)
}

// splitEvenOdd splits a points list into even-index and odd-index columns, as
// the TS pointColumns() helper does when a 2-column bullet layout is requested.
func splitEvenOdd(points []string) (left []string, right []string) {
	for i, p := range points {
		if i%2 == 0 {
			left = append(left, p)
		} else {
			right = append(right, p)
		}
	}
	return left, right
}

func panelClassFor(cp compositionParams) string {
	switch {
	case cp.PanelFilled:
		return "barq-panel-filled"
	case cp.PanelOutline:
		return "barq-panel-outline"
	default:
		return "barq-panel-glass"
	}
}

func gridColsClass(n int) string {
	switch {
	case n <= 1:
		return "barq-cols-1"
	case n == 2:
		return "barq-cols-2"
	default:
		return "barq-cols-3"
	}
}

// densityStyle emits a single CSS declaration (with trailing semicolon) scaled
// by cp.Density for gap/padding sizing.
func densityStyle(cp compositionParams, kind string) string {
	switch kind {
	case "gap":
		v := math.Round(16 * cp.Density)
		return "gap: " + strconv.FormatFloat(v, 'f', 0, 64) + "px;"
	case "padding":
		v := math.Round(24 * cp.Density)
		return "padding: " + strconv.FormatFloat(v, 'f', 0, 64) + "px;"
	}
	return ""
}

func densityPanelStyle(cp compositionParams) string {
	pad := math.Round(24 * cp.Density)
	font := math.Round(20 * cp.Density)
	return "padding: " + strconv.FormatFloat(pad, 'f', 0, 64) + "px; font-size: " +
		strconv.FormatFloat(font, 'f', 0, 64) + "px;"
}

func floatPct(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64) + "%"
}

// -----------------------------------------------------------------------------
// Style / CSS normalization helpers used by sanitize* (retained because
// pptx_subject_plan.go sanitizes slide.HTML before storing it). Kept private.
// -----------------------------------------------------------------------------

func normalizeInlineStyleDeclarations(raw string) string {
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

func normalizeCSSRuleBodies(raw string) string {
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
		return `"Helvetica Neue", Arial, sans-serif`
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
			return `"Helvetica Neue", Arial, sans-serif`
		}
	}
	return value
}
