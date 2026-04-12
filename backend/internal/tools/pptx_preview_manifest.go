package tools

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

const pptxPreviewManifestPath = "customXml/barq-presentation.json"

type pptxPreviewManifest struct {
	Version   int                 `json:"version"`
	Title     string              `json:"title"`
	Subtitle  string              `json:"subtitle,omitempty"`
	Theme     string              `json:"theme"`
	DeckPlan  pptxPreviewDeckPlan `json:"deck_plan"`
	Narrative string              `json:"narrative"`
	LayoutMix []string            `json:"layout_mix,omitempty"`
	Slides    []pptxPreviewSlide  `json:"slides"`
}

type pptxPreviewDeckPlan struct {
	Subject         string   `json:"subject"`
	Audience        string   `json:"audience"`
	NarrativeArc    string   `json:"narrative_arc"`
	VisualDirection string   `json:"visual_direction"`
	DominantNeed    string   `json:"dominant_need"`
	LayoutMix       []string `json:"layout_mix,omitempty"`
}

type pptxPreviewSlide struct {
	Number          int                   `json:"number"`
	Heading         string                `json:"heading"`
	Layout          string                `json:"layout"`
	Variant         int                   `json:"variant"`
	Purpose         string                `json:"purpose,omitempty"`
	Visual          string                `json:"visual,omitempty"`
	ContentSource   string                `json:"content_source,omitempty"`
	Audit           plannedPPTXSlideAudit `json:"audit"`
	SpeakerNotes    string                `json:"speaker_notes,omitempty"`
	Points          []string              `json:"points,omitempty"`
	Stats           []pptxStat            `json:"stats,omitempty"`
	Steps           []string              `json:"steps,omitempty"`
	Cards           []pptxCard            `json:"cards,omitempty"`
	ChartType       string                `json:"chart_type,omitempty"`
	ChartCategories []string              `json:"chart_categories,omitempty"`
	ChartSeries     []pptxChartSeries     `json:"chart_series,omitempty"`
	YLabel          string                `json:"y_label,omitempty"`
	Timeline        []pptxTimelineItem    `json:"timeline,omitempty"`
	LeftColumn      *pptxCompareColumn    `json:"left_column,omitempty"`
	RightColumn     *pptxCompareColumn    `json:"right_column,omitempty"`
	Table           *pptxTableData        `json:"table,omitempty"`
}

func buildPPTXPreviewManifest(title, subtitle string, planned plannedPPTXPresentation) ([]byte, error) {
	manifest := pptxPreviewManifest{
		Version:  1,
		Title:    strings.TrimSpace(title),
		Subtitle: strings.TrimSpace(subtitle),
		Theme:    planned.ThemeName,
		DeckPlan: pptxPreviewDeckPlan{
			Subject:         planned.DeckPlan.Subject,
			Audience:        planned.DeckPlan.Audience,
			NarrativeArc:    planned.DeckPlan.NarrativeArc,
			VisualDirection: planned.DeckPlan.VisualDirection,
			DominantNeed:    planned.DeckPlan.DominantNeed,
			LayoutMix:       append([]string(nil), planned.DeckPlan.LayoutMix...),
		},
		Narrative: firstNonEmpty(strings.TrimSpace(planned.DeckPlan.NarrativeArc), previewNarrative(planned.Slides)),
		LayoutMix: append([]string(nil), planned.DeckPlan.LayoutMix...),
		Slides:    make([]pptxPreviewSlide, 0, len(planned.Slides)),
	}

	for i, slide := range planned.Slides {
		manifest.Slides = append(manifest.Slides, pptxPreviewSlide{
			Number:          i + 2,
			Heading:         slide.Slide.Heading,
			Layout:          slide.Layout,
			Variant:         slide.Variant,
			Purpose:         firstNonEmpty(slide.Plan.Purpose, previewPurpose(slide.Layout)),
			Visual:          firstNonEmpty(slide.Plan.Visual, previewVisual(slide.Layout)),
			ContentSource:   slide.Plan.ContentSource,
			Audit:           slide.Audit,
			SpeakerNotes:    strings.TrimSpace(slide.Slide.SpeakerNotes),
			Points:          append([]string(nil), slide.Slide.Points...),
			Stats:           append([]pptxStat(nil), slide.Slide.Stats...),
			Steps:           append([]string(nil), slide.Slide.Steps...),
			Cards:           append([]pptxCard(nil), slide.Slide.Cards...),
			ChartType:       slide.Slide.ChartType,
			ChartCategories: append([]string(nil), slide.Slide.ChartCategories...),
			ChartSeries:     append([]pptxChartSeries(nil), slide.Slide.ChartSeries...),
			YLabel:          slide.Slide.YLabel,
			Timeline:        append([]pptxTimelineItem(nil), slide.Slide.Timeline...),
			LeftColumn:      cloneCompareColumn(slide.Slide.LeftColumn),
			RightColumn:     cloneCompareColumn(slide.Slide.RightColumn),
			Table:           cloneTable(slide.Slide.Table),
		})
	}

	return json.MarshalIndent(manifest, "", "  ")
}

func loadPPTXPreviewManifest(data []byte) (pptxPreviewManifest, bool, error) {
	manifestBytes, err := officeZipRead(data, pptxPreviewManifestPath)
	if err != nil {
		if strings.Contains(err.Error(), "zip entry not found") {
			return pptxPreviewManifest{}, false, nil
		}
		return pptxPreviewManifest{}, false, err
	}

	var manifest pptxPreviewManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return pptxPreviewManifest{}, false, fmt.Errorf("parse pptx preview manifest: %w", err)
	}
	return manifest, true, nil
}

func renderPPTXPreviewManifest(manifest pptxPreviewManifest) string {
	pal := paletteFor(manifest.Theme)
	var body strings.Builder

	body.WriteString(renderPPTXPreviewCover(manifest, pal))
	for _, slide := range manifest.Slides {
		body.WriteString(renderPPTXPreviewSlide(slide, pal, len(manifest.Slides)+1))
	}

	return pptxPreviewHTMLShell(body.String(), manifest, pal)
}

func renderPPTXPreviewCover(manifest pptxPreviewManifest, pal pptxPalette) string {
	layoutChips := ""
	if len(manifest.LayoutMix) > 0 {
		var chips strings.Builder
		for _, layout := range manifest.LayoutMix {
			chips.WriteString(`<span class="barq-preview-chip">` + html.EscapeString(strings.ToUpper(layout)) + `</span>`)
		}
		layoutChips = `<div class="barq-preview-chip-row">` + chips.String() + `</div>`
	}

	narrative := firstNonEmpty(
		strings.TrimSpace(manifest.DeckPlan.NarrativeArc),
		strings.TrimSpace(manifest.Narrative),
		"Subject-led deck planned and rendered in Go.",
	)

	audience := ""
	if strings.TrimSpace(manifest.DeckPlan.Audience) != "" {
		audience = `<p class="barq-preview-audience">Audience: ` + html.EscapeString(manifest.DeckPlan.Audience) + `</p>`
	}

	return `<section class="barq-preview-cover">
  <div class="barq-preview-cover-panel">
    <p class="barq-preview-eyebrow">PowerPoint Preview</p>
    <h1>` + html.EscapeString(firstNonEmpty(manifest.Title, "Presentation")) + `</h1>
    <p class="barq-preview-subtitle">` + html.EscapeString(firstNonEmpty(manifest.Subtitle, "Deck preview")) + `</p>
    <p class="barq-preview-narrative">` + html.EscapeString(narrative) + `</p>
    ` + audience + `
    ` + layoutChips + `
  </div>
  <div class="barq-preview-cover-accent" style="background:` + hexColor(pal.accent) + `"></div>
</section>`
}

func renderPPTXPreviewSlide(slide pptxPreviewSlide, pal pptxPalette, totalSlides int) string {
	meta := `<div class="barq-preview-meta">
  <span class="barq-preview-kicker">Slide ` + fmt.Sprintf("%d of %d", slide.Number, totalSlides) + `</span>
  <span class="barq-preview-kicker">` + html.EscapeString(strings.ToUpper(firstNonEmpty(slide.Layout, "bullets"))) + `</span>
  <span class="barq-preview-kicker">` + html.EscapeString(firstNonEmpty(slide.Visual, "planned visual")) + `</span>`
	if strings.TrimSpace(slide.ContentSource) != "" {
		meta += `
  <span class="barq-preview-kicker">` + html.EscapeString(strings.ToUpper(slide.ContentSource)) + `</span>`
	}
	if slide.Audit.ContentFit && slide.Audit.LayoutFit && slide.Audit.VisualFit {
		meta += `
  <span class="barq-preview-kicker">AUDITED</span>`
	}
	meta += `
</div>`

	notes := ""
	if text := strings.TrimSpace(slide.SpeakerNotes); text != "" {
		notes = `<div class="barq-preview-notes"><strong>Speaker notes:</strong> ` + html.EscapeString(text) + `</div>`
	}

	body := renderPPTXPreviewBody(slide, pal)
	return `<section class="barq-preview-slide" data-layout="` + html.EscapeString(slide.Layout) + `">
  ` + meta + `
  <h2>` + html.EscapeString(firstNonEmpty(slide.Heading, "Untitled Slide")) + `</h2>
  <p class="barq-preview-purpose">` + html.EscapeString(firstNonEmpty(slide.Purpose, "Slide-specific narrative block.")) + `</p>
  ` + body + notes + `
</section>`
}

func renderPPTXPreviewBody(slide pptxPreviewSlide, pal pptxPalette) string {
	switch slide.Layout {
	case "stats":
		return renderPPTXPreviewStats(slide)
	case "steps":
		return renderPPTXPreviewSteps(slide)
	case "cards":
		return renderPPTXPreviewCards(slide, pal)
	case "chart":
		return renderPPTXPreviewChart(slide, pal)
	case "timeline":
		return renderPPTXPreviewTimeline(slide)
	case "compare":
		return renderPPTXPreviewCompare(slide)
	case "table":
		return renderPPTXPreviewTable(slide)
	case "title":
		return renderPPTXPreviewSection(slide)
	case "blank":
		return renderPPTXPreviewBlank(slide)
	default:
		return renderPPTXPreviewBullets(slide)
	}
}

func renderPPTXPreviewBullets(slide pptxPreviewSlide) string {
	var items strings.Builder
	for _, point := range safePoints(slide.Points, 6) {
		items.WriteString(`<li>` + html.EscapeString(point) + `</li>`)
	}
	return `<ul class="barq-preview-list">` + items.String() + `</ul>`
}

func renderPPTXPreviewStats(slide pptxPreviewSlide) string {
	stats := slide.Stats
	if len(stats) == 0 {
		stats = effectiveStats(pptxSlide{Points: slide.Points})
	}

	var cards strings.Builder
	for _, stat := range stats {
		cards.WriteString(`<article class="barq-preview-stat">
  <div class="barq-preview-stat-value">` + html.EscapeString(stat.Value) + `</div>
  <div class="barq-preview-stat-label">` + html.EscapeString(stat.Label) + `</div>
  <div class="barq-preview-stat-desc">` + html.EscapeString(firstNonEmpty(stat.Desc, stat.Label)) + `</div>
</article>`)
	}
	return `<div class="barq-preview-stats-grid">` + cards.String() + `</div>`
}

func renderPPTXPreviewSteps(slide pptxPreviewSlide) string {
	steps := slide.Steps
	if len(steps) == 0 {
		steps = slide.Points
	}

	var items strings.Builder
	for i, step := range safePoints(steps, 6) {
		items.WriteString(`<div class="barq-preview-step">
  <span class="barq-preview-step-num">` + fmt.Sprintf("%d", i+1) + `</span>
  <span class="barq-preview-step-text">` + html.EscapeString(step) + `</span>
</div>`)
	}
	return `<div class="barq-preview-steps">` + items.String() + `</div>`
}

func renderPPTXPreviewCards(slide pptxPreviewSlide, pal pptxPalette) string {
	cards := slide.Cards
	if len(cards) == 0 {
		cards = effectiveCards(pptxSlide{Points: slide.Points})
	}

	var items strings.Builder
	for i, card := range cards {
		items.WriteString(`<article class="barq-preview-card">
  <div class="barq-preview-card-icon">` + previewCardIconSVG(inferCardIcon(card, i), pal) + `</div>
  <h3>` + html.EscapeString(card.Title) + `</h3>
  <p>` + html.EscapeString(firstNonEmpty(card.Desc, card.Title)) + `</p>
</article>`)
	}
	return `<div class="barq-preview-cards-grid">` + items.String() + `</div>`
}

func renderPPTXPreviewChart(slide pptxPreviewSlide, pal pptxPalette) string {
	chartType := strings.ToLower(strings.TrimSpace(firstNonEmpty(slide.ChartType, "column")))
	switch chartType {
	case "pie", "doughnut":
		return renderPPTXPreviewShareChart(slide, pal)
	default:
		return renderPPTXPreviewSeriesChart(slide, pal)
	}
}

func renderPPTXPreviewShareChart(slide pptxPreviewSlide, pal pptxPalette) string {
	categories := chartCategoriesOrFallback(pptxSlide{ChartCategories: slide.ChartCategories})
	series := slide.ChartSeries
	if len(series) == 0 {
		series = []pptxChartSeries{{Name: "Share", Values: []float64{40, 26, 19, 15}}}
	}
	values := normalizedChartValues(series[0].Values, len(categories))
	total := sumFloats(values)
	if total <= 0 {
		total = 1
	}

	var rows strings.Builder
	for i, category := range categories {
		pct := int((values[i] / total) * 100)
		rows.WriteString(`<div class="barq-preview-share-row">
  <div class="barq-preview-share-label">` + html.EscapeString(category) + `</div>
  <div class="barq-preview-share-track"><span style="width:` + fmt.Sprintf("%d%%", pct) + `;background:` + hexColor(pal.accent) + `"></span></div>
  <div class="barq-preview-share-value">` + fmt.Sprintf("%d%%", pct) + `</div>
</div>`)
	}
	return `<div class="barq-preview-share-chart">` + rows.String() + `</div>`
}

func renderPPTXPreviewSeriesChart(slide pptxPreviewSlide, pal pptxPalette) string {
	categories := chartCategoriesOrFallback(pptxSlide{ChartCategories: slide.ChartCategories})
	series := slide.ChartSeries
	if len(series) == 0 {
		series = []pptxChartSeries{{Name: "Series", Values: []float64{28, 44, 36, 58}}}
	}
	values := normalizedChartValues(series[0].Values, len(categories))
	maxValue := maxFloat(values)
	if maxValue <= 0 {
		maxValue = 1
	}

	var bars strings.Builder
	for i, category := range categories {
		height := int((values[i] / maxValue) * 100)
		if height < 8 {
			height = 8
		}
		bars.WriteString(`<div class="barq-preview-bar-group">
  <div class="barq-preview-bar-value">` + html.EscapeString(formatChartNumber(values[i])) + `</div>
  <div class="barq-preview-bar" style="height:` + fmt.Sprintf("%d%%", height) + `;background:` + hexColor(pal.accent) + `"></div>
  <div class="barq-preview-bar-label">` + html.EscapeString(category) + `</div>
</div>`)
	}

	yLabel := ""
	if label := strings.TrimSpace(slide.YLabel); label != "" {
		yLabel = `<div class="barq-preview-chart-axis">` + html.EscapeString(label) + `</div>`
	}

	return `<div class="barq-preview-chart-block">` + yLabel + `<div class="barq-preview-bars">` + bars.String() + `</div></div>`
}

func renderPPTXPreviewTimeline(slide pptxPreviewSlide) string {
	var items strings.Builder
	for _, item := range slide.Timeline {
		items.WriteString(`<article class="barq-preview-timeline-item">
  <div class="barq-preview-timeline-date">` + html.EscapeString(item.Date) + `</div>
  <div class="barq-preview-timeline-body">
    <h3>` + html.EscapeString(item.Title) + `</h3>
    <p>` + html.EscapeString(firstNonEmpty(item.Desc, item.Title)) + `</p>
  </div>
</article>`)
	}
	return `<div class="barq-preview-timeline">` + items.String() + `</div>`
}

func renderPPTXPreviewCompare(slide pptxPreviewSlide) string {
	left, right := effectiveCompareColumns(pptxSlide{
		LeftColumn:  cloneCompareColumn(slide.LeftColumn),
		RightColumn: cloneCompareColumn(slide.RightColumn),
	})
	return `<div class="barq-preview-compare">
  ` + renderPPTXPreviewCompareColumn(left, "Before") + `
  ` + renderPPTXPreviewCompareColumn(right, "After") + `
</div>`
}

func renderPPTXPreviewCompareColumn(column pptxCompareColumn, fallback string) string {
	var items strings.Builder
	for _, point := range safePoints(column.Points, 5) {
		items.WriteString(`<li>` + html.EscapeString(point) + `</li>`)
	}
	return `<article class="barq-preview-compare-col">
  <h3>` + html.EscapeString(firstNonEmpty(column.Heading, fallback)) + `</h3>
  <ul class="barq-preview-list">` + items.String() + `</ul>
</article>`
}

func renderPPTXPreviewTable(slide pptxPreviewSlide) string {
	if slide.Table == nil {
		return ``
	}

	var head strings.Builder
	for _, value := range slide.Table.Headers {
		head.WriteString(`<th>` + html.EscapeString(value) + `</th>`)
	}

	var rows strings.Builder
	for _, row := range slide.Table.Rows {
		rows.WriteString("<tr>")
		for _, value := range row {
			rows.WriteString(`<td>` + html.EscapeString(value) + `</td>`)
		}
		rows.WriteString("</tr>")
	}

	return `<div class="barq-preview-table-wrap">
  <table class="barq-preview-table">
    <thead><tr>` + head.String() + `</tr></thead>
    <tbody>` + rows.String() + `</tbody>
  </table>
</div>`
}

func renderPPTXPreviewSection(slide pptxPreviewSlide) string {
	body := ""
	if len(slide.Points) > 0 {
		body = `<p class="barq-preview-section-copy">` + html.EscapeString(slide.Points[0]) + `</p>`
	}
	return `<div class="barq-preview-section-break">
  <div class="barq-preview-section-badge">Section Divider</div>
  ` + body + `
</div>`
}

func renderPPTXPreviewBlank(slide pptxPreviewSlide) string {
	body := firstNonEmpty(firstSlidePoint(slide.Points), slide.Heading, "Transition")
	return `<div class="barq-preview-blank">
  <div class="barq-preview-blank-label">Transition</div>
  <p>` + html.EscapeString(body) + `</p>
</div>`
}

func pptxPreviewHTMLShell(content string, manifest pptxPreviewManifest, pal pptxPalette) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(firstNonEmpty(manifest.Title, "Presentation Preview")) + `</title>
<style>
  :root {
    --bg: ` + hexColor(pal.bg) + `;
    --card: ` + hexColor(pal.card) + `;
    --accent: ` + hexColor(pal.accent) + `;
    --accent-2: ` + hexColor(pal.accent2) + `;
    --text: ` + hexColor(pal.text) + `;
    --muted: ` + hexColor(pal.muted) + `;
    --border: ` + hexColor(pal.border) + `;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    background: linear-gradient(180deg, #020617 0%%, var(--bg) 100%%);
    color: var(--text);
    font-family: "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
  }
  main { max-width: 1080px; margin: 0 auto; padding: 28px 20px 48px; }
  .barq-preview-cover, .barq-preview-slide {
    border-radius: 24px;
    border: 1px solid rgba(148, 163, 184, 0.18);
    background: linear-gradient(180deg, rgba(15, 23, 42, 0.92), rgba(15, 23, 42, 0.74));
    box-shadow: 0 24px 60px rgba(2, 6, 23, 0.32);
    margin-bottom: 18px;
  }
  .barq-preview-cover {
    position: relative;
    overflow: hidden;
    min-height: 260px;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 180px;
  }
  .barq-preview-cover-panel { padding: 34px 34px 30px; }
  .barq-preview-cover-accent { opacity: 0.95; }
  .barq-preview-eyebrow {
    margin: 0 0 12px;
    color: var(--accent-2);
    font-size: 12px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    font-weight: 700;
  }
  .barq-preview-cover h1 {
    margin: 0 0 10px;
    font-size: clamp(2rem, 3vw, 3rem);
    line-height: 1.08;
    letter-spacing: -0.03em;
  }
  .barq-preview-subtitle, .barq-preview-narrative {
    margin: 0 0 12px;
    color: var(--muted);
    font-size: 15px;
    line-height: 1.65;
  }
  .barq-preview-audience {
    margin: 0 0 12px;
    color: var(--accent-2);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-chip-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 18px;
  }
  .barq-preview-chip, .barq-preview-kicker {
    display: inline-flex;
    align-items: center;
    padding: 6px 10px;
    border-radius: 999px;
    background: rgba(99, 102, 241, 0.14);
    color: var(--accent-2);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-slide { padding: 24px 26px 26px; }
  .barq-preview-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 14px;
  }
  .barq-preview-slide h2 {
    margin: 0 0 8px;
    font-size: 28px;
    line-height: 1.15;
  }
  .barq-preview-purpose {
    margin: 0 0 18px;
    color: var(--muted);
    font-size: 14px;
  }
  .barq-preview-list {
    margin: 0;
    padding-left: 18px;
    display: grid;
    gap: 10px;
    color: var(--text);
    line-height: 1.6;
  }
  .barq-preview-stats-grid, .barq-preview-cards-grid, .barq-preview-compare {
    display: grid;
    gap: 14px;
  }
  .barq-preview-stats-grid { grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
  .barq-preview-cards-grid { grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); }
  .barq-preview-stat, .barq-preview-card, .barq-preview-compare-col, .barq-preview-blank, .barq-preview-section-break {
    background: rgba(15, 23, 42, 0.55);
    border: 1px solid rgba(148, 163, 184, 0.16);
    border-radius: 18px;
    padding: 16px;
  }
  .barq-preview-stat-value {
    font-size: 28px;
    font-weight: 800;
    color: var(--accent-2);
  }
  .barq-preview-stat-label {
    margin-top: 8px;
    font-size: 14px;
    font-weight: 700;
  }
  .barq-preview-stat-desc, .barq-preview-card p, .barq-preview-timeline-item p, .barq-preview-notes {
    margin-top: 6px;
    color: var(--muted);
    font-size: 13px;
    line-height: 1.6;
  }
  .barq-preview-steps {
    display: grid;
    gap: 12px;
  }
  .barq-preview-step {
    display: grid;
    grid-template-columns: 36px minmax(0, 1fr);
    gap: 12px;
    align-items: start;
    padding: 12px 14px;
    border-radius: 16px;
    border: 1px solid rgba(148, 163, 184, 0.16);
    background: rgba(15, 23, 42, 0.52);
  }
  .barq-preview-step-num, .barq-preview-card-icon {
    width: 36px;
    height: 36px;
    border-radius: 999px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-weight: 700;
  }
  .barq-preview-card-icon svg {
    width: 100%;
    height: 100%;
    display: block;
  }
  .barq-preview-step-text { padding-top: 6px; line-height: 1.55; }
  .barq-preview-card h3, .barq-preview-compare-col h3, .barq-preview-timeline-item h3 {
    margin: 12px 0 4px;
    font-size: 16px;
  }
  .barq-preview-chart-block { display: grid; gap: 14px; }
  .barq-preview-chart-axis {
    color: var(--muted);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .barq-preview-bars {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(84px, 1fr));
    gap: 14px;
    align-items: end;
    min-height: 240px;
    padding: 16px;
    border-radius: 18px;
    background: rgba(15, 23, 42, 0.55);
    border: 1px solid rgba(148, 163, 184, 0.16);
  }
  .barq-preview-bar-group {
    min-height: 210px;
    display: flex;
    flex-direction: column;
    justify-content: end;
    gap: 8px;
    align-items: stretch;
  }
  .barq-preview-bar-value, .barq-preview-bar-label, .barq-preview-share-value {
    font-size: 12px;
    text-align: center;
    color: var(--muted);
  }
  .barq-preview-bar {
    border-radius: 12px 12px 4px 4px;
    min-height: 18px;
  }
  .barq-preview-share-chart, .barq-preview-timeline, .barq-preview-table-wrap {
    display: grid;
    gap: 12px;
  }
  .barq-preview-share-row {
    display: grid;
    grid-template-columns: minmax(120px, 180px) minmax(0, 1fr) 58px;
    gap: 12px;
    align-items: center;
  }
  .barq-preview-share-track {
    height: 12px;
    border-radius: 999px;
    background: rgba(148, 163, 184, 0.14);
    overflow: hidden;
  }
  .barq-preview-share-track span {
    display: block;
    height: 100%%;
    border-radius: inherit;
  }
  .barq-preview-timeline-item {
    display: grid;
    grid-template-columns: 110px minmax(0, 1fr);
    gap: 14px;
    align-items: start;
    padding: 14px 16px;
    border-radius: 18px;
    background: rgba(15, 23, 42, 0.52);
    border: 1px solid rgba(148, 163, 184, 0.16);
  }
  .barq-preview-timeline-date {
    color: var(--accent-2);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    padding-top: 4px;
  }
  .barq-preview-compare { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .barq-preview-table {
    width: 100%%;
    border-collapse: collapse;
    overflow: hidden;
    border-radius: 18px;
  }
  .barq-preview-table thead th {
    background: rgba(99, 102, 241, 0.16);
    color: var(--accent-2);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .barq-preview-table th, .barq-preview-table td {
    padding: 12px 14px;
    border: 1px solid rgba(148, 163, 184, 0.16);
    text-align: left;
    font-size: 13px;
  }
  .barq-preview-table td { background: rgba(15, 23, 42, 0.52); }
  .barq-preview-section-break, .barq-preview-blank { text-align: center; }
  .barq-preview-section-badge, .barq-preview-blank-label {
    color: var(--accent-2);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    margin-bottom: 10px;
  }
  .barq-preview-section-copy, .barq-preview-blank p {
    margin: 0;
    color: var(--muted);
    line-height: 1.6;
  }
  .barq-preview-notes {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px dashed rgba(148, 163, 184, 0.18);
  }
  @media (max-width: 720px) {
    main { padding: 18px 14px 28px; }
    .barq-preview-cover { grid-template-columns: 1fr; }
    .barq-preview-cover-accent { min-height: 12px; }
    .barq-preview-compare { grid-template-columns: 1fr; }
    .barq-preview-timeline-item, .barq-preview-share-row {
      grid-template-columns: 1fr;
    }
    .barq-preview-bar-group { min-height: 170px; }
  }
</style>
</head>
<body>
<main>` + content + `</main>
</body>
</html>`
}

func previewLayoutMix(slides []plannedPPTXSlide) []string {
	seen := make(map[string]struct{}, len(slides))
	var mix []string
	for _, slide := range slides {
		layout := strings.TrimSpace(slide.Layout)
		if layout == "" {
			continue
		}
		if _, ok := seen[layout]; ok {
			continue
		}
		seen[layout] = struct{}{}
		mix = append(mix, layout)
	}
	return mix
}

func previewNarrative(slides []plannedPPTXSlide) string {
	if len(slides) == 0 {
		return ""
	}

	labels := make([]string, 0, len(slides))
	for _, slide := range slides {
		labels = append(labels, previewPurpose(slide.Layout))
	}
	return strings.Join(labels, " -> ")
}

func previewPurpose(layout string) string {
	switch layout {
	case "stats":
		return "Quantify the headline metrics."
	case "steps":
		return "Walk through the process or rollout flow."
	case "cards":
		return "Show capabilities with icons and short benefit statements."
	case "chart":
		return "Visualize the key trend or comparison signal."
	case "timeline":
		return "Sequence the milestones and rollout phases."
	case "compare":
		return "Contrast the current state against the target state."
	case "table":
		return "Provide a structured reference matrix."
	case "title":
		return "Introduce the next section of the story."
	case "blank":
		return "Create a clean transition before the next topic."
	default:
		return "Explain the key narrative points clearly."
	}
}

func previewVisual(layout string) string {
	switch layout {
	case "stats":
		return "metric cards"
	case "steps":
		return "process flow"
	case "cards":
		return "icon grid"
	case "chart":
		return "data chart"
	case "timeline":
		return "milestone timeline"
	case "compare":
		return "side-by-side compare"
	case "table":
		return "decision matrix"
	case "title":
		return "section divider"
	case "blank":
		return "transition slide"
	default:
		return "narrative list"
	}
}

func hexColor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "#0F172A"
	}
	if strings.HasPrefix(value, "#") {
		return value
	}
	return "#" + value
}

func cloneCompareColumn(column *pptxCompareColumn) *pptxCompareColumn {
	if column == nil {
		return nil
	}
	copy := *column
	copy.Points = append([]string(nil), column.Points...)
	return &copy
}

func cloneTable(table *pptxTableData) *pptxTableData {
	if table == nil {
		return nil
	}
	copy := &pptxTableData{
		Headers: append([]string(nil), table.Headers...),
		Rows:    make([][]string, 0, len(table.Rows)),
	}
	for _, row := range table.Rows {
		copy.Rows = append(copy.Rows, append([]string(nil), row...))
	}
	return copy
}

func firstSlidePoint(points []string) string {
	for _, point := range points {
		if strings.TrimSpace(point) != "" {
			return point
		}
	}
	return ""
}
