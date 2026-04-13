package tools

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
)

const pptxPreviewManifestPath = "customXml/barq-presentation.json"

type pptxPreviewManifest struct {
	Version   int                 `json:"version"`
	Title     string              `json:"title"`
	Subtitle  string              `json:"subtitle,omitempty"`
	Theme     string              `json:"theme"`
	Palette   pptxPreviewPalette  `json:"palette"`
	DeckPlan  pptxPreviewDeckPlan `json:"deck_plan"`
	Narrative string              `json:"narrative"`
	LayoutMix []string            `json:"layout_mix,omitempty"`
	Slides    []pptxPreviewSlide  `json:"slides"`
}

type pptxPreviewPalette struct {
	Background string `json:"background"`
	Card       string `json:"card"`
	Accent     string `json:"accent"`
	Accent2    string `json:"accent2"`
	Text       string `json:"text"`
	Muted      string `json:"muted"`
	Border     string `json:"border"`
}

type pptxPreviewDeckPlan struct {
	Subject         string         `json:"subject"`
	Audience        string         `json:"audience"`
	NarrativeArc    string         `json:"narrative_arc"`
	VisualDirection string         `json:"visual_direction"`
	DominantNeed    string         `json:"dominant_need"`
	CoverStyle      string         `json:"cover_style,omitempty"`
	ColorStory      string         `json:"color_story,omitempty"`
	Motif           string         `json:"motif,omitempty"`
	Kicker          string         `json:"kicker,omitempty"`
	Design          pptxDeckDesign `json:"design,omitempty"`
	LayoutMix       []string       `json:"layout_mix,omitempty"`
}

type pptxPreviewSlide struct {
	Number          int                   `json:"number"`
	Heading         string                `json:"heading"`
	Layout          string                `json:"layout"`
	Variant         int                   `json:"variant"`
	Purpose         string                `json:"purpose,omitempty"`
	Visual          string                `json:"visual,omitempty"`
	ContentSource   string                `json:"content_source,omitempty"`
	Design          *pptxSlideDesign      `json:"design,omitempty"`
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
		Palette: pptxPreviewPalette{
			Background: planned.Palette.bg,
			Card:       planned.Palette.card,
			Accent:     planned.Palette.accent,
			Accent2:    planned.Palette.accent2,
			Text:       planned.Palette.text,
			Muted:      planned.Palette.muted,
			Border:     planned.Palette.border,
		},
		DeckPlan: pptxPreviewDeckPlan{
			Subject:         planned.DeckPlan.Subject,
			Audience:        planned.DeckPlan.Audience,
			NarrativeArc:    planned.DeckPlan.NarrativeArc,
			VisualDirection: planned.DeckPlan.VisualDirection,
			DominantNeed:    planned.DeckPlan.DominantNeed,
			CoverStyle:      planned.DeckPlan.CoverStyle,
			ColorStory:      planned.DeckPlan.ColorStory,
			Motif:           planned.DeckPlan.Motif,
			Kicker:          planned.DeckPlan.Kicker,
			Design:          planned.DeckPlan.Design,
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
			Design:          cloneSlideDesign(slide.Slide.Design),
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

func cloneSlideDesign(design *pptxSlideDesign) *pptxSlideDesign {
	if design == nil {
		return nil
	}
	copy := *design
	return &copy
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
	pal := previewManifestPalette(manifest)
	family := previewDeckRenderFamily(manifest)
	var body strings.Builder

	body.WriteString(renderPPTXPreviewCover(manifest, pal))
	for _, slide := range manifest.Slides {
		body.WriteString(renderPPTXPreviewSlide(slide, pal, len(manifest.Slides)+1, family))
	}

	return pptxPreviewHTMLShell(body.String(), manifest, pal)
}

func renderPPTXPreviewCover(manifest pptxPreviewManifest, pal pptxPalette) string {
	if previewDeckRenderFamily(manifest) == "proposal" {
		return renderPPTXPreviewProposalCover(manifest, pal)
	}
	kicker := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Kicker), "Subject-specific presentation"))
	subtitle := ""
	if text := strings.TrimSpace(manifest.Subtitle); text != "" {
		subtitle = `<p class="barq-preview-subtitle">` + html.EscapeString(text) + `</p>`
	}
	support := ""
	if audience := strings.TrimSpace(manifest.DeckPlan.Audience); audience != "" {
		support = `<p class="barq-preview-support">For ` + html.EscapeString(audience) + `</p>`
	}
	subjectLine := ""
	if subject := strings.TrimSpace(manifest.DeckPlan.Subject); subject != "" {
		subjectLine = `<p class="barq-preview-subject">` + html.EscapeString(subject) + `</p>`
	}
	coverStyle := html.EscapeString(previewCoverStyle(manifest))
	design := previewDeckRenderDesign(manifest)

	return `<section class="barq-preview-cover" data-cover-style="` + coverStyle + `" data-composition="` + html.EscapeString(design.Composition) + `" data-accent-mode="` + html.EscapeString(design.AccentMode) + `">
  <div class="barq-preview-cover-stage">
    <div class="barq-preview-cover-panel">
      <p class="barq-preview-eyebrow">` + kicker + `</p>
      <div class="barq-preview-cover-rule"></div>
      <h1>` + html.EscapeString(firstNonEmpty(manifest.Title, "Presentation")) + `</h1>
      ` + subtitle + `
      ` + support + `
      ` + subjectLine + `
    </div>
    <div class="barq-preview-cover-figure" data-composition="` + html.EscapeString(design.Composition) + `">` + renderPPTXPreviewCoverFigure(manifest, pal, design) + `</div>
  </div>
</section>`
}

func renderPPTXPreviewSlide(slide pptxPreviewSlide, pal pptxPalette, totalSlides int, family string) string {
	if family == "proposal" {
		return renderPPTXPreviewProposalSlide(slide, pal, totalSlides)
	}
	design := previewSlideRenderDesign(slide)
	meta := `<div class="barq-preview-meta">
  <span class="barq-preview-kicker">Slide ` + fmt.Sprintf("%d of %d", slide.Number, totalSlides) + `</span>
</div>`

	body := renderPPTXPreviewBody(slide, pal)
	return `<section class="barq-preview-slide" data-layout="` + html.EscapeString(slide.Layout) + `" data-layout-style="` + html.EscapeString(design.LayoutStyle) + `" data-accent-mode="` + html.EscapeString(design.AccentMode) + `" data-density="` + html.EscapeString(design.Density) + `" data-panel-style="` + html.EscapeString(design.PanelStyle) + `">
  <div class="barq-preview-slide-stage">
    ` + meta + `
    <h2>` + html.EscapeString(firstNonEmpty(slide.Heading, "Untitled Slide")) + `</h2>
    <div class="barq-preview-slide-body">` + body + `</div>
  </div>
</section>`
}

func previewDeckRenderDesign(manifest pptxPreviewManifest) pptxDeckDesign {
	design := manifest.DeckPlan.Design
	if strings.TrimSpace(design.Composition) == "" {
		design.Composition = "split"
	}
	if strings.TrimSpace(design.AccentMode) == "" {
		design.AccentMode = "rail"
	}
	if strings.TrimSpace(design.Density) == "" {
		design.Density = "balanced"
	}
	if strings.TrimSpace(design.ShapeLanguage) == "" {
		design.ShapeLanguage = "mixed"
	}
	if strings.TrimSpace(design.HeroLayout) == "" {
		design.HeroLayout = "motif"
	}
	return design
}

func previewSlideRenderDesign(slide pptxPreviewSlide) pptxSlideDesign {
	if slide.Design != nil {
		return *slide.Design
	}
	return pptxSlideDesign{
		LayoutStyle: "stack",
		PanelStyle:  "soft",
		AccentMode:  "rail",
		Density:     "balanced",
		VisualFocus: "text",
	}
}

func previewDeckRenderFamily(manifest pptxPreviewManifest) string {
	text := strings.ToLower(strings.Join([]string{
		manifest.Title,
		manifest.Subtitle,
		manifest.Theme,
		manifest.DeckPlan.Subject,
		manifest.DeckPlan.Audience,
		manifest.DeckPlan.NarrativeArc,
		manifest.DeckPlan.VisualDirection,
		manifest.DeckPlan.ColorStory,
		manifest.DeckPlan.DominantNeed,
	}, " "))

	switch {
	case containsAny(text, "playful", "storybook", "collage", "cartoon", "fun") &&
		!containsAny(text, "refined", "structured", "proposal", "report", "executive", "premium"):
		return "playful"
	case containsAny(text, "poster", "campaign", "showcase", "gallery", "bold studio") &&
		!containsAny(text, "proposal", "report", "summary", "brief", "guide"):
		return "studio"
	default:
		return "proposal"
	}
}

func previewProposalSlideTitle(title string) string {
	title = strings.TrimSpace(title)
	if len([]rune(title)) <= 28 {
		return strings.ToUpper(title)
	}
	return title
}

func previewProposalSlideIconToken(slide pptxPreviewSlide) string {
	switch slide.Layout {
	case "stats", "chart":
		return "chart"
	case "steps", "timeline":
		return "strategy"
	case "compare":
		return "integration"
	case "table":
		return "shield"
	case "cards":
		if len(slide.Cards) > 0 {
			return inferCardIcon(slide.Cards[0], 0)
		}
		return "spark"
	default:
		return firstNonEmpty(normalizeIconToken(slide.Heading), inferIconFromText(slide.Heading+" "+slide.Layout), "spark")
	}
}

func renderPPTXPreviewProposalCover(manifest pptxPreviewManifest, pal pptxPalette) string {
	kicker := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Kicker), "Subject-specific presentation"))
	subtitle := ""
	if text := strings.TrimSpace(manifest.Subtitle); text != "" {
		subtitle = `<p class="barq-preview-subtitle">` + html.EscapeString(text) + `</p>`
	}
	support := ""
	if audience := strings.TrimSpace(manifest.DeckPlan.Audience); audience != "" {
		support = `<p class="barq-preview-support">For ` + html.EscapeString(audience) + `</p>`
	}
	footer := html.EscapeString(firstNonEmpty(strings.TrimSpace(manifest.DeckPlan.Subject), manifest.Title, "Presentation"))
	return `<section class="barq-preview-cover" data-family="proposal">
  <div class="barq-preview-cover-stage">
    <div class="barq-preview-proposal-cover-panel">
      <p class="barq-preview-eyebrow">` + kicker + `</p>
      <div class="barq-preview-cover-rule"></div>
      <h1>` + html.EscapeString(firstNonEmpty(manifest.Title, "Presentation")) + `</h1>
      ` + subtitle + `
      ` + support + `
    </div>
    <div class="barq-preview-proposal-footer">` + footer + `</div>
  </div>
</section>`
}

func renderPPTXPreviewProposalSlide(slide pptxPreviewSlide, pal pptxPalette, totalSlides int) string {
	iconPal := pal
	iconPal.text = "FFFFFF"
	iconPal.accent = mixHex(proposalHeaderFill(pal), pal.accent, 0.22)
	body := renderPPTXPreviewBody(slide, pal)
	return `<section class="barq-preview-slide" data-family="proposal" data-layout="` + html.EscapeString(slide.Layout) + `">
  <div class="barq-preview-slide-stage">
    <div class="barq-preview-proposal-head">
      <div class="barq-preview-proposal-icon">` + previewCardIconSVG(previewProposalSlideIconToken(slide), iconPal) + `</div>
      <h2>` + html.EscapeString(previewProposalSlideTitle(firstNonEmpty(slide.Heading, "Untitled Slide"))) + `</h2>
      <span class="barq-preview-proposal-meta">Slide ` + fmt.Sprintf("%d of %d", slide.Number, totalSlides) + `</span>
    </div>
    <div class="barq-preview-slide-body">` + body + `</div>
  </div>
</section>`
}

func renderPPTXPreviewCoverFigure(manifest pptxPreviewManifest, pal pptxPalette, design pptxDeckDesign) string {
	switch strings.ToLower(strings.TrimSpace(design.Composition)) {
	case "gallery", "asym":
		return `<div class="barq-preview-cover-gallery">
  <div class="barq-preview-cover-tile"></div>
  <div class="barq-preview-cover-tile barq-preview-cover-tile-accent"></div>
  <div class="barq-preview-cover-tile barq-preview-cover-tile-accent-2"></div>
  <div class="barq-preview-cover-tile"></div>
  <div class="barq-preview-cover-tile"></div>
  <div class="barq-preview-cover-tile barq-preview-cover-tile-warm"></div>
</div>`
	case "float":
		return `<div class="barq-preview-cover-float">
  <div class="barq-preview-cover-orb barq-preview-cover-orb-a"></div>
  <div class="barq-preview-cover-orb barq-preview-cover-orb-b"></div>
  <div class="barq-preview-cover-icon">` + previewCardIconSVG(firstNonEmpty(normalizeIconToken(manifest.DeckPlan.Motif), defaultMotif(manifest.Theme, manifest.DeckPlan.Audience)), pal) + `</div>
</div>`
	default:
		return `<div class="barq-preview-cover-focus">
  <div class="barq-preview-cover-orb barq-preview-cover-orb-a"></div>
  <div class="barq-preview-cover-orb barq-preview-cover-orb-b"></div>
  <div class="barq-preview-cover-icon">` + previewCardIconSVG(firstNonEmpty(normalizeIconToken(manifest.DeckPlan.Motif), defaultMotif(manifest.Theme, manifest.DeckPlan.Audience)), pal) + `</div>
</div>`
	}
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
    --accent-soft: ` + hexColor(mixHex(pal.bg, pal.accent, 0.16)) + `;
    --accent-soft-2: ` + hexColor(mixHex(pal.bg, pal.accent2, 0.18)) + `;
    --border-soft: ` + hexColor(mixHex(pal.bg, pal.border, 0.72)) + `;
    --surface-soft: ` + hexColor(mixHex(pal.bg, pal.card, 0.88)) + `;
    --track-soft: ` + hexColor(mixHex(pal.bg, pal.border, 0.18)) + `;
    --proposal-bg: ` + hexColor(proposalCanvasFill(pal)) + `;
    --proposal-header: ` + hexColor(proposalHeaderFill(pal)) + `;
    --proposal-footer: ` + hexColor(proposalFooterFill(pal)) + `;
    --proposal-muted: ` + hexColor(proposalMutedOnDark(pal)) + `;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    background:
      radial-gradient(circle at top right, var(--accent-soft), transparent 34%),
      linear-gradient(180deg, var(--bg), ` + hexColor(mixHex(pal.bg, pal.card, 0.3)) + `);
    color: var(--text);
    font-family: "Avenir Next", "SF Pro Display", "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
  }
  main { max-width: 1180px; margin: 0 auto; padding: 32px 22px 56px; }
  .barq-preview-cover, .barq-preview-slide {
    margin-bottom: 22px;
  }
  .barq-preview-cover-stage, .barq-preview-slide-stage {
    position: relative;
    overflow: hidden;
    border-radius: 34px;
    border: 1px solid var(--border-soft);
    background:
      linear-gradient(180deg, rgba(255,255,255,0.52), rgba(255,255,255,0.10)),
      var(--card);
    box-shadow: 0 24px 54px rgba(15, 23, 42, 0.10);
  }
  .barq-preview-cover-stage {
    min-height: 420px;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 280px;
    padding: 34px;
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-cover-stage {
    min-height: 420px;
    display: block;
    padding: 0;
    border-radius: 26px;
    border: none;
    background: var(--proposal-header);
    box-shadow: 0 28px 64px rgba(10, 20, 32, 0.28);
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-cover-stage::before {
    content: "";
    position: absolute;
    inset: 0 0 auto 0;
    height: 6px;
    background: var(--accent);
  }
  .barq-preview-proposal-cover-panel {
    padding: 60px 56px 40px;
    max-width: 720px;
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-eyebrow {
    color: var(--accent);
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-cover-rule {
    background: var(--accent-2);
    margin: 14px 0 24px;
  }
  .barq-preview-cover[data-family="proposal"] h1 {
    color: #fff;
    max-width: 13ch;
    margin-bottom: 18px;
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-subtitle,
  .barq-preview-cover[data-family="proposal"] .barq-preview-support,
  .barq-preview-cover[data-family="proposal"] .barq-preview-subject {
    color: var(--proposal-muted);
    max-width: 34ch;
  }
  .barq-preview-cover[data-family="proposal"] .barq-preview-cover-figure {
    display: none;
  }
  .barq-preview-proposal-footer {
    position: absolute;
    inset: auto 0 0 0;
    height: 42px;
    display: flex;
    align-items: center;
    padding: 0 56px;
    background: var(--proposal-footer);
    color: var(--proposal-muted);
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-cover-panel { padding: 8px 12px 8px; max-width: 560px; }
  .barq-preview-cover-rule {
    width: 196px;
    height: 3px;
    border-radius: 999px;
    background: var(--accent);
    margin: 14px 0 26px;
  }
  .barq-preview-cover-figure {
    position: relative;
    overflow: hidden;
    border-radius: 28px;
    min-height: 340px;
    border: 1px solid var(--border-soft);
    background: linear-gradient(180deg, rgba(255,255,255,0.35), transparent), var(--accent-soft);
    padding: 18px;
  }
  .barq-preview-cover-orb {
    position: absolute;
    border-radius: 999px;
    background: var(--accent);
    opacity: 0.14;
  }
  .barq-preview-cover-orb-a {
    width: 180px;
    height: 180px;
    top: 12px;
    right: 20px;
  }
  .barq-preview-cover-orb-b {
    width: 132px;
    height: 132px;
    bottom: 18px;
    left: 24px;
    background: var(--accent-2);
  }
  .barq-preview-cover-icon {
    position: relative;
    width: 92px;
    height: 92px;
    z-index: 1;
  }
  .barq-preview-cover-icon svg {
    width: 100%;
    height: 100%;
    display: block;
  }
  .barq-preview-cover-focus, .barq-preview-cover-float {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .barq-preview-cover-gallery {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    height: 100%;
  }
  .barq-preview-cover-tile {
    min-height: 86px;
    border-radius: 26px;
    background: rgba(255,255,255,0.86);
    border: 1px solid var(--border-soft);
  }
  .barq-preview-cover-tile-accent { background: rgba(74, 144, 217, 0.18); }
  .barq-preview-cover-tile-accent-2 { background: rgba(124, 198, 196, 0.18); }
  .barq-preview-cover-tile-warm { background: rgba(255, 140, 66, 0.12); }
  .barq-preview-cover[data-cover-style="poster"] {
    grid-template-columns: 1fr;
  }
  .barq-preview-cover[data-cover-style="poster"] .barq-preview-cover-stage {
    grid-template-columns: 180px minmax(0, 1fr);
  }
  .barq-preview-cover[data-composition="band"] .barq-preview-cover-stage::before {
    content: "";
    position: absolute;
    inset: 0 auto 0 0;
    width: 160px;
    background: linear-gradient(180deg, var(--accent-soft), transparent 74%);
  }
  .barq-preview-cover[data-composition="gallery"] .barq-preview-cover-figure,
  .barq-preview-cover[data-composition="asym"] .barq-preview-cover-figure {
    background: linear-gradient(180deg, rgba(255,255,255,0.32), transparent), rgba(255,140,66,0.10);
  }
  .barq-preview-eyebrow {
    margin: 0;
    color: var(--accent);
    font-size: 12px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    font-weight: 700;
  }
  .barq-preview-cover h1 {
    margin: 0 0 18px;
    font-size: clamp(2.2rem, 3.4vw, 4rem);
    line-height: 1.08;
    letter-spacing: -0.03em;
    max-width: 11ch;
  }
  .barq-preview-subtitle {
    margin: 0 0 14px;
    color: var(--accent-2);
    font-size: 16px;
    line-height: 1.45;
    font-weight: 700;
    max-width: 26ch;
  }
  .barq-preview-support,
  .barq-preview-subject {
    margin: 0 0 12px;
    color: var(--muted);
    font-size: 14px;
    line-height: 1.6;
  }
  .barq-preview-subject {
    margin-top: 24px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-kicker {
    display: inline-flex;
    align-items: center;
    padding: 7px 12px;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-slide-stage {
    padding: 28px 28px 30px;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-slide-stage {
    padding: 0 0 24px;
    border-radius: 18px;
    background: var(--proposal-bg);
    box-shadow: none;
  }
  .barq-preview-proposal-head {
    display: grid;
    grid-template-columns: 36px minmax(0, 1fr) auto;
    gap: 14px;
    align-items: center;
    padding: 12px 24px;
    background: var(--proposal-header);
    color: #fff;
  }
  .barq-preview-proposal-icon {
    width: 32px;
    height: 32px;
  }
  .barq-preview-proposal-icon svg {
    width: 100%;
    height: 100%;
    display: block;
  }
  .barq-preview-slide[data-family="proposal"] h2 {
    margin: 0;
    color: #fff;
    font-size: 20px;
    line-height: 1.2;
    letter-spacing: 0.02em;
    max-width: none;
  }
  .barq-preview-proposal-meta {
    display: inline-flex;
    align-items: center;
    padding: 7px 12px;
    border-radius: 999px;
    background: rgba(255,255,255,0.14);
    color: #fff;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-slide-body {
    padding: 18px 24px 0;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-meta {
    display: none;
  }
  .barq-preview-slide-stage::before {
    content: "";
    position: absolute;
  }
  .barq-preview-slide[data-accent-mode="rail"] .barq-preview-slide-stage::before {
    left: 0;
    top: 0;
    bottom: 0;
    width: 4px;
    background: var(--accent);
  }
  .barq-preview-slide[data-accent-mode="band"] .barq-preview-slide-stage::before {
    left: 0;
    top: 0;
    right: 0;
    height: 4px;
    background: var(--accent);
  }
  .barq-preview-slide[data-accent-mode="chip"] .barq-preview-slide-stage::before {
    right: 34px;
    top: 22px;
    width: 92px;
    height: 14px;
    border-radius: 999px;
    background: var(--accent-soft);
  }
  .barq-preview-slide[data-accent-mode="ribbon"] .barq-preview-slide-stage::before {
    left: 28px;
    top: 24px;
    width: 136px;
    height: 6px;
    border-radius: 999px;
    background: var(--accent);
  }
  .barq-preview-slide[data-accent-mode="marker"] .barq-preview-slide-stage::before {
    left: 30px;
    top: 108px;
    width: 4px;
    height: 56px;
    border-radius: 999px;
    background: var(--accent);
  }
  .barq-preview-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 14px;
  }
  .barq-preview-slide h2 {
    margin: 0 0 18px;
    font-size: clamp(2rem, 2.5vw, 2.7rem);
    line-height: 1.15;
    letter-spacing: -0.03em;
    max-width: 18ch;
  }
  .barq-preview-slide-body { display: grid; gap: 14px; }
  .barq-preview-list {
    margin: 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 10px;
    color: var(--text);
    line-height: 1.6;
  }
  .barq-preview-list li {
    position: relative;
    padding: 14px 16px 14px 34px;
    border-radius: 18px;
    background: var(--surface-soft);
    border: 1px solid var(--border-soft);
  }
  .barq-preview-list li::before {
    content: "";
    position: absolute;
    left: 14px;
    top: 20px;
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: var(--accent);
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-list li {
    padding: 14px 16px 14px 18px;
    border-radius: 8px;
    background: #fff;
    border: 1px solid var(--border-soft);
    border-left: 4px solid var(--accent);
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-list li::before {
    display: none;
  }
  .barq-preview-stats-grid, .barq-preview-cards-grid, .barq-preview-compare {
    display: grid;
    gap: 14px;
  }
  .barq-preview-stats-grid { grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
  .barq-preview-cards-grid { grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); }
  .barq-preview-stat, .barq-preview-card, .barq-preview-compare-col, .barq-preview-blank, .barq-preview-section-break {
    background: var(--surface-soft);
    border: 1px solid var(--border-soft);
    border-radius: 18px;
    padding: 16px;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-stat,
  .barq-preview-slide[data-family="proposal"] .barq-preview-card,
  .barq-preview-slide[data-family="proposal"] .barq-preview-compare-col,
  .barq-preview-slide[data-family="proposal"] .barq-preview-blank,
  .barq-preview-slide[data-family="proposal"] .barq-preview-section-break {
    background: #fff;
    border-radius: 8px;
    box-shadow: none;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-stat {
    border-top: 4px solid var(--accent);
    padding-top: 14px;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-card,
  .barq-preview-slide[data-family="proposal"] .barq-preview-compare-col {
    border-left: 4px solid var(--accent);
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
    padding: 14px 16px;
    border-radius: 16px;
    border: 1px solid var(--border-soft);
    background: var(--surface-soft);
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-step {
    border-radius: 8px;
    background: #fff;
    border-left: 4px solid var(--accent);
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
    font-size: 17px;
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
    background: var(--surface-soft);
    border: 1px solid var(--border-soft);
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-bars,
  .barq-preview-slide[data-family="proposal"] .barq-preview-timeline-item,
  .barq-preview-slide[data-family="proposal"] .barq-preview-table td,
  .barq-preview-slide[data-family="proposal"] .barq-preview-table-wrap {
    background: #fff;
    border-radius: 8px;
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
    background: var(--track-soft);
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
    background: var(--surface-soft);
    border: 1px solid var(--border-soft);
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-timeline-item {
    border-radius: 8px;
    background: #fff;
    border-left: 4px solid var(--accent);
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
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .barq-preview-slide[data-family="proposal"] .barq-preview-table thead th {
    background: rgba(15, 23, 42, 0.06);
    color: var(--text);
  }
  .barq-preview-table th, .barq-preview-table td {
    padding: 12px 14px;
    border: 1px solid var(--border-soft);
    text-align: left;
    font-size: 13px;
  }
  .barq-preview-table td { background: var(--surface-soft); }
  .barq-preview-section-break, .barq-preview-blank { text-align: center; }
  .barq-preview-section-badge, .barq-preview-blank-label {
    color: var(--accent);
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
  @media (max-width: 720px) {
    main { padding: 18px 14px 28px; }
    .barq-preview-cover-stage { grid-template-columns: 1fr; padding: 22px; }
    .barq-preview-cover-figure { min-height: 160px; }
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

func previewManifestPalette(manifest pptxPreviewManifest) pptxPalette {
	pal := paletteFor(manifest.Theme)
	if color := normalizePaletteHex(manifest.Palette.Background); color != "" {
		pal.bg = color
	}
	if color := normalizePaletteHex(manifest.Palette.Card); color != "" {
		pal.card = color
	}
	if color := normalizePaletteHex(manifest.Palette.Accent); color != "" {
		pal.accent = color
	}
	if color := normalizePaletteHex(manifest.Palette.Accent2); color != "" {
		pal.accent2 = color
	}
	if color := normalizePaletteHex(manifest.Palette.Text); color != "" {
		pal.text = color
	}
	if color := normalizePaletteHex(manifest.Palette.Muted); color != "" {
		pal.muted = color
	}
	if color := normalizePaletteHex(manifest.Palette.Border); color != "" {
		pal.border = color
	}
	return pal
}

func previewCoverStyle(manifest pptxPreviewManifest) string {
	style := strings.ToLower(strings.TrimSpace(manifest.DeckPlan.CoverStyle))
	switch {
	case containsAny(style, "poster", "studio", "showcase"):
		return "poster"
	case containsAny(style, "orbit", "radial"):
		return "orbit"
	case containsAny(style, "mosaic", "grid", "collage"):
		return "mosaic"
	case containsAny(style, "playful", "kids", "classroom"):
		return "playful"
	default:
		return "editorial"
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

func hexRGBA(value string, alpha float64) string {
	hex := normalizePaletteHex(value)
	if hex == "" {
		hex = "0F172A"
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", r, g, b, alpha)
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
