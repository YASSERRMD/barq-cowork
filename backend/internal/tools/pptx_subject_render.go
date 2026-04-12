package tools

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type pptxDeckContext struct {
	Title        string
	Subtitle     string
	ThemeName    string
	DeckPlan     plannedPPTXDeckPlan
	SlideCount   int
	CoverVariant int
}

func newPPTXDeckContext(title, subtitle string, planned plannedPPTXPresentation) pptxDeckContext {
	deck := pptxDeckContext{
		Title:      title,
		Subtitle:   subtitle,
		ThemeName:  planned.ThemeName,
		DeckPlan:   planned.DeckPlan,
		SlideCount: len(planned.Slides),
	}
	deck.CoverVariant = coverVisualVariant(deck)
	return deck
}

func coverVisualVariant(deck pptxDeckContext) int {
	switch coverStyleKey(deck) {
	case "orbit":
		return 1
	case "mosaic":
		return 2
	case "poster":
		return 3
	case "playful":
		return 4
	default:
		sum := 0
		for _, ch := range strings.ToLower(deck.Title + "|" + deck.ThemeName + "|" + deck.DeckPlan.Subject) {
			sum += int(ch)
		}
		if sum < 0 {
			sum = -sum
		}
		return sum % 5
	}
}

func renderDeckSlide(s pptxSlide, pal pptxPalette, deck pptxDeckContext, slideIndex int) string {
	layout := effectivePPTXLayout(s)
	variant := slideVisualVariant(deck.Title, s.Heading, layout, slideIndex)

	switch layout {
	case "title":
		return renderSectionSlide(s, pal, variant, slideIndex)
	case "stats":
		return renderStatsDeckSlide(s, pal, variant, deck, slideIndex)
	case "steps":
		return renderStepsDeckSlide(s, pal, variant, deck, slideIndex)
	case "cards":
		return renderCardsDeckSlide(s, pal, variant, deck, slideIndex)
	case "chart":
		return renderChartDeckSlide(s, pal, variant, deck, slideIndex)
	case "timeline":
		return renderTimelineDeckSlide(s, pal, variant, deck, slideIndex)
	case "compare":
		return renderCompareDeckSlide(s, pal, variant, deck, slideIndex)
	case "table":
		return renderTableDeckSlide(s, pal, variant, deck, slideIndex)
	case "blank":
		return renderBlankDeckSlide(s, pal, variant, slideIndex)
	default:
		return renderBulletsDeckSlide(s, pal, variant, deck, slideIndex)
	}
}

func effectivePPTXLayout(s pptxSlide) string {
	if kind := normalizeSlideKind(s.Type); kind != "" {
		return kind
	}
	if kind := normalizeSlideKind(s.Layout); kind != "" {
		return kind
	}
	switch {
	case len(s.ChartSeries) > 0 || len(s.ChartCategories) > 0:
		return "chart"
	case len(s.Timeline) > 0:
		return "timeline"
	case s.LeftColumn != nil || s.RightColumn != nil:
		return "compare"
	case s.Table != nil && len(s.Table.Headers) > 0:
		return "table"
	case len(s.Cards) > 0:
		return "cards"
	case len(s.Steps) > 0:
		return "steps"
	case len(s.Stats) > 0:
		return "stats"
	case len(s.Points) > 0:
		return autoLayout(s.Heading, s.Points)
	default:
		return "blank"
	}
}

func normalizeSlideKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bullets", "stats", "steps", "cards", "chart", "timeline", "compare", "table", "title", "blank":
		return strings.ToLower(strings.TrimSpace(raw))
	case "bullets_slide":
		return "bullets"
	case "stats_slide":
		return "stats"
	default:
		return ""
	}
}

func slideVisualVariant(deckTitle, heading, layout string, slideIndex int) int {
	sum := slideIndex*17 + len(layout)*11
	for _, ch := range strings.ToLower(deckTitle + "|" + heading + "|" + layout) {
		sum += int(ch)
	}
	if sum < 0 {
		sum = -sum
	}
	return sum % 3
}

func slideHeadingOrFallback(s pptxSlide, fallback string) string {
	if strings.TrimSpace(s.Heading) != "" {
		return strings.TrimSpace(s.Heading)
	}
	return fallback
}

func renderPPTXCoverSlide(deck pptxDeckContext, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder

	switch deck.CoverVariant {
	case 1:
		renderCoverOrbitFocus(g, &sb, deck, pal)
	case 2:
		renderCoverMosaicGrid(g, &sb, deck, pal)
	case 3:
		renderCoverStudioPoster(g, &sb, deck, pal)
	case 4:
		renderCoverPlayfulCanvas(g, &sb, deck, pal)
	default:
		renderCoverEditorialSplit(g, &sb, deck, pal)
	}

	return wrapSlide(sb.String())
}

func renderCoverEditorialSplit(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "accentRail", 0, 0, 9144000, 38100, pal.accent))
	sb.WriteString(spRect(g, "rightPanel", 6360000, 0, 2784000, 6858000, pal.card))
	sb.WriteString(spEllipse(g, "heroOrb", 6840000, -360000, 2340000, 2340000, pal.accent, 14, "", 0, 0))
	sb.WriteString(spEllipse(g, "footOrb", 7120000, 4680000, 1500000, 1500000, pal.accent2, 12, "", 0, 0))
	renderCoverMotif(g, sb, "editorialMotif", 7000000, 1180000, 1180000, pal, coverMotifToken(deck), 22)
	sb.WriteString(spTextLeft(g, "titleKicker", 540000, 880000, 2300000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 1080, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "title", 540000, 1480000, 5200000, 1820000, firstNonEmpty(deck.Title, "Presentation"), pal.text, 4200, true, "t", "Calibri Light"))
	if subtitle := coverLead(deck); subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 540000, 3720000, 4500000, 360000, subtitle, pal.accent2, 1600, true, "t", "Calibri"))
	}
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 540000, 4160000, 4800000, 360000, support, pal.muted, 1220, false, "t", "Calibri"))
	}
	sb.WriteString(spRect(g, "titleRule", 540000, 5200000, 2100000, 25400, pal.accent))
	sb.WriteString(spTextLeft(g, "subjectLine", 540000, 5360000, 3000000, 220000, coverSubjectLine(deck), pal.muted, 980, true, "t", "Calibri"))
}

func renderCoverOrbitFocus(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spEllipse(g, "orbitOne", 800000, 720000, 4200000, 4200000, "", 0, pal.border, 6350, 42))
	sb.WriteString(spEllipse(g, "orbitTwo", 1260000, 1180000, 3340000, 3340000, "", 0, pal.border, 6350, 42))
	sb.WriteString(spEllipse(g, "orbitThree", 1740000, 1660000, 2380000, 2380000, "", 0, pal.border, 6350, 42))
	sb.WriteString(spEllipse(g, "accentGlow", 1280000, 2140000, 1400000, 1400000, pal.accent, 18, "", 0, 0))
	renderCoverMotif(g, sb, "orbitMotif", 2140000, 2600000, 900000, pal, coverMotifToken(deck), 26)
	sb.WriteString(spRoundRect(g, "titlePanel", 4300000, 1120000, 3660000, 3920000, pal.card, pal.border, 12))
	sb.WriteString(spRect(g, "titleAccent", 4300000, 1120000, 3660000, 38100, pal.accent))
	sb.WriteString(spTextLeft(g, "titleKicker", 4620000, 1480000, 2400000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 1080, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "title", 4620000, 1980000, 3020000, 1500000, firstNonEmpty(deck.Title, "Presentation"), pal.text, 3600, true, "t", "Calibri Light"))
	if subtitle := coverLead(deck); subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 4620000, 3560000, 2780000, 320000, subtitle, pal.accent2, 1480, true, "t", "Calibri"))
	}
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 4620000, 3980000, 2780000, 420000, support, pal.muted, 1160, false, "t", "Calibri"))
	}
	sb.WriteString(spTextLeft(g, "subjectLine", 4620000, 4700000, 2200000, 180000, coverSubjectLine(deck), pal.muted, 960, true, "t", "Calibri"))
}

func renderCoverMosaicGrid(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3200)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRoundRect(g, "titlePanel", 620000, 3720000, 3600000, 1980000, pal.card, pal.border, 12))
	sb.WriteString(spRect(g, "panelAccent", 620000, 3720000, 3600000, 38100, pal.accent))
	tileXs := []int{4680000, 6440000, 4680000, 6440000, 7320000}
	tileYs := []int{920000, 920000, 2660000, 2660000, 4380000}
	tileWs := []int{1540000, 1540000, 1540000, 1540000, 980000}
	tileHs := []int{1400000, 1400000, 1400000, 1400000, 1180000}
	for i := range tileXs {
		fill := pal.card
		if i == 1 || i == 3 {
			fill = pal.accent
		}
		sb.WriteString(spRoundRect(g, fmt.Sprintf("tile%d", i), tileXs[i], tileYs[i], tileWs[i], tileHs[i], fill, pal.border, 12))
	}
	renderCoverMotif(g, sb, "mosaicMotif", 6620000, 1060000, 1160000, pal, coverMotifToken(deck), 22)
	renderCoverMotif(g, sb, "mosaicMotifSmall", 4860000, 2840000, 760000, pal, coverMotifToken(deck), 18)
	sb.WriteString(spTextLeft(g, "titleKicker", 900000, 4040000, 2200000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 1080, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "title", 900000, 4520000, 2820000, titleHeight+140000, firstNonEmpty(deck.Title, "Presentation"), pal.text, titleSize, true, "t", "Calibri Light"))
	subtitleY := 4520000 + titleHeight + 180000
	if subtitle := coverLead(deck); subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 900000, subtitleY, 2500000, 260000, subtitle, pal.accent2, 1320, true, "t", "Calibri"))
	}
	supportY := subtitleY + 320000
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 4680000, supportY, 2400000, 220000, support, pal.muted, 980, false, "t", "Calibri"))
	}
	sb.WriteString(spTextLeft(g, "subjectLine", 4680000, supportY+340000, 2200000, 180000, coverSubjectLine(deck), pal.muted, 900, true, "t", "Calibri"))
}

func renderCoverStudioPoster(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3600)
	subtitle := coverLead(deck)
	subtitleSize, subtitleHeight := coverSubtitleMetrics(subtitle, 1500)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "posterBand", 0, 0, 2480000, 6858000, pal.accent))
	sb.WriteString(spRect(g, "posterRail", 2480000, 0, 300000, 6858000, pal.accent2))
	for i := 0; i < 3; i++ {
		y := 980000 + i*1460000
		sb.WriteString(spRoundRect(g, fmt.Sprintf("posterStrip%d", i), 6460000, y, 1980000, 1080000, pal.card, pal.border, 10))
	}
	renderCoverMotif(g, sb, "posterMotif", 6740000, 1160000, 960000, pal, coverMotifToken(deck), 24)
	sb.WriteString(spTextLeft(g, "titleKicker", 2980000, 900000, 2600000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent2, 1120, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "title", 2980000, 1460000, 3200000, titleHeight, firstNonEmpty(deck.Title, "Presentation"), pal.text, titleSize, true, "t", "Calibri Light"))
	subtitleY := 1460000 + titleHeight + 220000
	if subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 2980000, subtitleY, 2800000, subtitleHeight, subtitle, pal.accent2, subtitleSize, true, "t", "Calibri"))
	}
	supportY := subtitleY + subtitleHeight + 140000
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 2980000, supportY, 2600000, 420000, support, pal.muted, 1180, false, "t", "Calibri"))
	}
	sb.WriteString(spTextLeft(g, "subjectLine", 2980000, 5860000, 2200000, 180000, coverSubjectLine(deck), pal.muted, 960, true, "t", "Calibri"))
}

func renderCoverPlayfulCanvas(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spEllipse(g, "playOrbOne", 540000, 720000, 1400000, 1400000, pal.accent, 18, "", 0, 0))
	sb.WriteString(spEllipse(g, "playOrbTwo", 6780000, 820000, 1640000, 1640000, pal.accent2, 18, "", 0, 0))
	sb.WriteString(spEllipse(g, "playOrbThree", 7340000, 4640000, 1280000, 1280000, pal.accent, 14, "", 0, 0))
	sb.WriteString(spRoundRect(g, "titleCard", 860000, 1820000, 4480000, 2800000, pal.card, pal.border, 12))
	sb.WriteString(spRect(g, "titleAccent", 860000, 1820000, 4480000, 38100, pal.accent))
	sb.WriteString(spRoundRect(g, "stickerOne", 1120000, 1240000, 2200000, 320000, pal.accent, "", 0))
	sb.WriteString(spText(g, "titleKicker", 1120000, 1240000, 2200000, 320000, strings.ToUpper(coverKicker(deck)), pal.text, 1120, true, "ctr", "Calibri"))
	renderCoverMotif(g, sb, "playMotifOne", 5640000, 1880000, 1080000, pal, coverMotifToken(deck), 24)
	renderCoverMotif(g, sb, "playMotifTwo", 7040000, 3200000, 820000, pal, coverMotifToken(deck), 20)
	sb.WriteString(spTextLeft(g, "title", 1240000, 2320000, 3840000, 1360000, firstNonEmpty(deck.Title, "Presentation"), pal.text, 3420, true, "t", "Calibri Light"))
	if subtitle := coverLead(deck); subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 1240000, 3820000, 3300000, 300000, subtitle, pal.accent, 1440, true, "t", "Calibri"))
	}
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 1240000, 4200000, 3300000, 320000, support, pal.muted, 1100, false, "t", "Calibri"))
	}
}

func coverStyleKey(deck pptxDeckContext) string {
	if explicit := normalizeCoverStyleToken(deck.DeckPlan.CoverStyle); explicit != "" {
		return explicit
	}
	text := strings.ToLower(strings.Join([]string{
		deck.DeckPlan.VisualDirection,
		deck.DeckPlan.ColorStory,
		deck.DeckPlan.Audience,
	}, " "))
	switch {
	case containsAny(text, "playful", "kids", "children", "young learners", "classroom"):
		return "playful"
	case containsAny(text, "orbit", "radial", "tech", "signal", "data"):
		return "orbit"
	case containsAny(text, "mosaic", "grid", "collage", "modular"):
		return "mosaic"
	case containsAny(text, "poster", "studio", "campaign", "showcase"):
		return "poster"
	default:
		return "editorial"
	}
}

func normalizeCoverStyleToken(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "editorial", "orbit", "mosaic", "poster", "playful":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func coverLead(deck pptxDeckContext) string {
	return coverCompactText(strings.TrimSpace(deck.Subtitle), 62)
}

func coverSupportLine(deck pptxDeckContext) string {
	if audience := strings.TrimSpace(deck.DeckPlan.Audience); audience != "" {
		return coverCompactText("For "+audience, 56)
	}
	if story := strings.TrimSpace(deck.DeckPlan.NarrativeArc); story != "" && !strings.Contains(story, "->") {
		return coverCompactText(story, 56)
	}
	return ""
}

func coverKicker(deck pptxDeckContext) string {
	return coverCompactText(firstNonEmpty(deck.DeckPlan.Kicker, "Subject-specific presentation"), 46)
}

func coverSubjectLine(deck pptxDeckContext) string {
	return coverCompactText(firstNonEmpty(deck.DeckPlan.Subject, deck.Title), 36)
}

func coverMotifToken(deck pptxDeckContext) string {
	return firstNonEmpty(normalizeIconToken(deck.DeckPlan.Motif), defaultMotif(deck.ThemeName, deck.DeckPlan.Audience), "spark")
}

func renderCoverMotif(g *idg, sb *strings.Builder, name string, x, y, size int, pal pptxPalette, token string, fillAlpha int) {
	sb.WriteString(spEllipse(g, name+"Bg", x, y, size, size, pal.accent, fillAlpha, pal.accent2, 6350, minInt(fillAlpha+20, 100)))
	renderPPTXIconGlyph(g, sb, name+"Glyph", x, y, size, pal, token)
}

func coverCompactText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(value)) <= limit || limit <= 0 {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maxInt(0, limit-1)])) + "…"
}

func coverTitleMetrics(title string, base int) (int, int) {
	length := len([]rune(strings.TrimSpace(title)))
	switch {
	case length > 40:
		return base - 800, 1380000
	case length > 28:
		return base - 450, 1120000
	default:
		return base, 860000
	}
}

func coverSubtitleMetrics(subtitle string, base int) (int, int) {
	length := len([]rune(strings.TrimSpace(subtitle)))
	switch {
	case length > 72:
		return base - 260, 820000
	case length > 48:
		return base - 140, 620000
	default:
		return base, 340000
	}
}

func renderBackdrop(g *idg, sb *strings.Builder, pal pptxPalette, variant int) int {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))

	switch variant {
	case 0:
		sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 38100, pal.accent))
		sb.WriteString(spEllipse(g, "orbTop", 7100000, -700000, 2500000, 2500000, pal.accent, 7, "", 0, 0))
		sb.WriteString(spEllipse(g, "orbBottom", -300000, 5600000, 1200000, 1200000, pal.accent2, 6, "", 0, 0))
	case 1:
		sb.WriteString(spRect(g, "leftRail", 0, 0, 260000, 6858000, pal.accent))
		sb.WriteString(spRoundRect(g, "haloCard", 6640000, 440000, 1900000, 720000, pal.card, pal.border, 10))
		sb.WriteString(spEllipse(g, "halo", 7420000, 5180000, 1400000, 1400000, pal.accent2, 8, "", 0, 0))
	default:
		sb.WriteString(spRect(g, "rightPanel", 7480000, 0, 1664000, 6858000, pal.card))
		sb.WriteString(spEllipse(g, "topGlow", 6400000, -500000, 1800000, 1800000, pal.accent, 10, "", 0, 0))
		sb.WriteString(spEllipse(g, "midGlow", 7800000, 2300000, 900000, 900000, pal.accent2, 14, "", 0, 0))
	}

	return 840000
}

func renderHeadline(g *idg, sb *strings.Builder, pal pptxPalette, heading, kicker string, variant int) int {
	switch variant {
	case 0:
		sb.WriteString(spTextLeft(g, "heading", 457200, 140000, 7200000, 420000, heading, pal.text, 3000, true, "ctr", "Calibri Light"))
		sb.WriteString(spRect(g, "headingLine", 457200, 610000, 1600000, 22225, pal.accent))
		if kicker != "" {
			sb.WriteString(spTextLeft(g, "kicker", 457200, 650000, 2400000, 180000, strings.ToUpper(kicker), pal.muted, 1100, true, "t", "Calibri"))
		}
		return 900000
	case 1:
		sb.WriteString(spRoundRect(g, "headingCard", 520000, 160000, 6100000, 560000, pal.card, pal.border, 10))
		sb.WriteString(spRect(g, "headingAccent", 520000, 160000, 90000, 560000, pal.accent))
		sb.WriteString(spTextLeft(g, "heading", 700000, 230000, 5600000, 300000, heading, pal.text, 2800, true, "ctr", "Calibri Light"))
		if kicker != "" {
			sb.WriteString(spTextLeft(g, "kicker", 700000, 500000, 2600000, 120000, kicker, pal.muted, 1050, false, "t", "Calibri"))
		}
		return 980000
	default:
		sb.WriteString(spTextLeft(g, "heading", 457200, 180000, 6400000, 430000, heading, pal.text, 3100, true, "ctr", "Calibri Light"))
		sb.WriteString(spRoundRect(g, "badge", 457200, 620000, 1400000, 220000, pal.card, pal.border, 10))
		sb.WriteString(spText(g, "kicker", 457200, 620000, 1400000, 220000, strings.ToUpper(firstNonEmpty(kicker, "Key Slide")), pal.accent2, 1100, true, "ctr", "Calibri"))
		return 980000
	}
}

func renderBulletsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Key Points"), slideLabel(deck, slideIndex, "narrative"), variant)
	points := safePoints(s.Points, 6)
	if len(points) == 0 {
		points = []string{"Key insight", "Supporting evidence", "Operational implication"}
	}

	if variant == 2 && len(points) >= 4 {
		colW := 3860000
		gap := 260000
		cardH := 940000
		for i, pt := range points {
			col := i % 2
			row := i / 2
			x := 457200 + col*(colW+gap)
			y := top + row*(cardH+160000)
			sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), x, y, colW, cardH, pal.card, pal.border, 10))
			sb.WriteString(spEllipse(g, fmt.Sprintf("pointBadge%d", i), x+60000, y+140000, 260000, 260000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("pointNum%d", i), x+60000, y+140000, 260000, 260000, fmt.Sprintf("%d", i+1), pal.text, 1500, true, "ctr", "Calibri"))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), x+360000, y+100000, colW-440000, cardH-180000, pt, pal.text, 1500, false, "t", "Calibri"))
		}
		return wrapSlide(sb.String())
	}

	cardW := 8229600
	cardH := 730000
	if variant == 1 {
		cardH = 620000
	}
	for i, pt := range points {
		y := top + i*(cardH+110000)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), 457200, y, cardW, cardH, pal.card, pal.border, 10))
		sb.WriteString(spRect(g, fmt.Sprintf("pointAccent%d", i), 457200, y+30000, 38100, cardH-60000, pal.accent))
		sb.WriteString(spText(g, fmt.Sprintf("pointNum%d", i), 520000, y, 280000, cardH, fmt.Sprintf("%02d", i+1), pal.accent2, 1200, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), 860000, y+40000, 7600000, cardH-80000, pt, pal.text, 1500, false, "ctr", "Calibri"))
	}
	return wrapSlide(sb.String())
}

func renderStatsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Performance Metrics"), slideLabel(deck, slideIndex, "metrics"), variant)
	stats := effectiveStats(s)
	if len(stats) == 0 {
		stats = []pptxStat{
			{Value: "92%", Label: "Retention", Desc: "Strong repeat usage"},
			{Value: "3.1x", Label: "ROI", Desc: "Efficiency improvement"},
			{Value: "14d", Label: "Payback", Desc: "Fast adoption curve"},
		}
	}
	if len(stats) > 4 {
		stats = stats[:4]
	}

	cols := len(stats)
	if cols > 2 && variant == 1 {
		cols = 2
	}
	rows := (len(stats) + cols - 1) / cols
	totalW := 8229600
	gapX := 200000
	gapY := 180000
	cardW := (totalW - gapX*(cols-1)) / cols
	cardH := 1600000
	if rows == 2 {
		cardH = 1360000
	}

	for i, stat := range stats {
		col := i % cols
		row := i / cols
		x := 457200 + col*(cardW+gapX)
		y := top + row*(cardH+gapY)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("statCard%d", i), x, y, cardW, cardH, pal.card, pal.border, 10))
		sb.WriteString(spRect(g, fmt.Sprintf("statAccent%d", i), x+50000, y, cardW-100000, 28575, pal.accent))
		sb.WriteString(spText(g, fmt.Sprintf("statValue%d", i), x+20000, y+170000, cardW-40000, 460000, stat.Value, pal.accent2, 3600, true, "ctr", "Calibri Light"))
		sb.WriteString(spText(g, fmt.Sprintf("statLabel%d", i), x+20000, y+690000, cardW-40000, 220000, stat.Label, pal.text, 1500, true, "ctr", "Calibri"))
		if stat.Desc != "" {
			sb.WriteString(spText(g, fmt.Sprintf("statDesc%d", i), x+40000, y+980000, cardW-80000, 220000, stat.Desc, pal.muted, 1100, false, "ctr", "Calibri"))
		}
		if strings.Contains(stat.Value, "%") {
			pct := parsePercent(stat.Value)
			barW := (cardW - 140000) * pct / 100
			sb.WriteString(spRoundRect(g, fmt.Sprintf("track%d", i), x+70000, y+cardH-180000, cardW-140000, 50000, pal.border, "", 0))
			if barW > 0 {
				sb.WriteString(spRoundRect(g, fmt.Sprintf("fill%d", i), x+70000, y+cardH-180000, barW, 50000, pal.accent, "", 0))
			}
		}
	}

	return wrapSlide(sb.String())
}

func renderStepsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	steps := effectiveSteps(s)
	if len(steps) == 0 {
		steps = []string{"Discover the problem", "Design the solution", "Ship the workflow", "Measure adoption"}
	}

	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "How It Works"), slideLabel(deck, slideIndex, "workflow"), variant)

	if variant == 1 {
		lineX := 900000
		sb.WriteString(spRect(g, "roadmapLine", lineX, top, 16000, 4300000, pal.border))
		for i, step := range steps {
			y := top + i*900000
			sb.WriteString(spEllipse(g, fmt.Sprintf("stepNode%d", i), lineX-180000, y, 380000, 380000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", i), lineX-180000, y, 380000, 380000, fmt.Sprintf("%d", i+1), pal.text, 1700, true, "ctr", "Calibri"))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("stepCard%d", i), 1400000, y-60000, 6600000, 500000, pal.card, pal.border, 10))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("stepText%d", i), 1600000, y+20000, 6200000, 360000, step, pal.text, 1450, false, "t", "Calibri"))
		}
		return wrapSlide(sb.String())
	}

	stepCount := len(steps)
	if stepCount > 5 {
		stepCount = 5
		steps = steps[:5]
	}
	totalW := 7900000
	gap := 120000
	boxW := (totalW - gap*(stepCount-1)) / stepCount
	for i, step := range steps {
		x := 620000 + i*(boxW+gap)
		y := top + 500000
		sb.WriteString(spRightArrow(g, fmt.Sprintf("stepShape%d", i), x, y, boxW, 1300000, pal.card))
		sb.WriteString(spEllipse(g, fmt.Sprintf("stepBadge%d", i), x+50000, y+90000, 260000, 260000, pal.accent, 100, "", 0, 0))
		sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", i), x+50000, y+90000, 260000, 260000, fmt.Sprintf("%d", i+1), pal.text, 1400, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("stepText%d", i), x+340000, y+180000, boxW-420000, 880000, step, pal.text, 1350, false, "t", "Calibri"))
	}
	return wrapSlide(sb.String())
}

func renderCardsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Capabilities"), slideLabel(deck, slideIndex, "capabilities"), variant)
	cards := effectiveCards(s)
	if len(cards) == 0 {
		cards = []pptxCard{
			{Icon: "automation", Title: "Speed", Desc: "Faster execution"},
			{Icon: "shield", Title: "Control", Desc: "Safer operations"},
			{Icon: "chart", Title: "Insight", Desc: "Measurable outcomes"},
			{Icon: "integration", Title: "Flexibility", Desc: "Fits the workflow"},
		}
	}
	if len(cards) > 6 {
		cards = cards[:6]
	}

	cols := 3
	if len(cards) <= 4 {
		cols = 2
	}
	rows := (len(cards) + cols - 1) / cols
	cardW := (8229600 - 180000*(cols-1)) / cols
	cardH := (4600000 - 180000*(rows-1)) / rows
	startY := top + 120000
	for i, card := range cards {
		col := i % cols
		row := i / cols
		x := 457200 + col*(cardW+180000)
		y := startY + row*(cardH+180000)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), x, y, cardW, cardH, pal.card, pal.border, 10))
		sb.WriteString(spRect(g, fmt.Sprintf("cardAccent%d", i), x, y, cardW, 28575, pal.accent))
		renderCardIconBadge(g, &sb, fmt.Sprintf("cardIcon%d", i), x+cardW/2-180000, y+120000, 360000, pal, inferCardIcon(card, i), variant)
		sb.WriteString(spText(g, fmt.Sprintf("cardTitle%d", i), x+40000, y+560000, cardW-80000, 260000, card.Title, pal.text, 1500, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("cardDesc%d", i), x+50000, y+860000, cardW-100000, cardH-960000, firstNonEmpty(card.Desc, card.Title), pal.muted, 1150, false, "t", "Calibri"))
	}
	return wrapSlide(sb.String())
}

func renderChartDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Trend View"), slideLabel(deck, slideIndex, "data story"), variant)
	chartType := strings.ToLower(strings.TrimSpace(firstNonEmpty(s.ChartType, "column")))
	switch chartType {
	case "pie", "doughnut":
		renderShareChart(g, &sb, s, pal, top)
	default:
		renderSeriesChart(g, &sb, s, pal, top, variant)
	}
	return wrapSlide(sb.String())
}

func renderTimelineDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	items := s.Timeline
	if len(items) == 0 {
		items = []pptxTimelineItem{
			{Date: "Q1", Title: "Discovery", Desc: "Define the scope"},
			{Date: "Q2", Title: "Build", Desc: "Ship the first release"},
			{Date: "Q3", Title: "Adopt", Desc: "Operational rollout"},
			{Date: "Q4", Title: "Scale", Desc: "Expand to more teams"},
		}
	}
	if len(items) > 5 {
		items = items[:5]
	}

	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Roadmap"), slideLabel(deck, slideIndex, "timeline"), variant)

	if variant == 1 {
		x := 900000
		sb.WriteString(spRect(g, "timelineVertical", x, top, 22000, 4300000, pal.border))
		for i, item := range items {
			y := top + i*900000
			sb.WriteString(spEllipse(g, fmt.Sprintf("milestone%d", i), x-160000, y, 320000, 320000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("milestoneCard%d", i), 1300000, y-100000, 6500000, 520000, pal.card, pal.border, 10))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneDate%d", i), 1500000, y-40000, 1200000, 180000, item.Date, pal.accent2, 1100, true, "t", "Calibri"))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneTitle%d", i), 1500000, y+110000, 2200000, 180000, item.Title, pal.text, 1400, true, "t", "Calibri"))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneDesc%d", i), 3800000, y+100000, 3700000, 200000, item.Desc, pal.muted, 1050, false, "t", "Calibri"))
		}
		return wrapSlide(sb.String())
	}

	lineY := top + 1700000
	sb.WriteString(spRect(g, "timelineLine", 700000, lineY, 7600000, 22000, pal.border))
	nodeGap := 7600000 / len(items)
	for i, item := range items {
		cx := 700000 + i*nodeGap + nodeGap/2
		sb.WriteString(spEllipse(g, fmt.Sprintf("timelineNode%d", i), cx-180000, lineY-170000, 360000, 360000, pal.accent, 100, "", 0, 0))
		cardY := lineY - 1100000
		if i%2 == 1 {
			cardY = lineY + 240000
		}
		sb.WriteString(spRoundRect(g, fmt.Sprintf("timelineCard%d", i), cx-650000, cardY, 1300000, 620000, pal.card, pal.border, 10))
		sb.WriteString(spText(g, fmt.Sprintf("timelineDate%d", i), cx-600000, cardY+60000, 1200000, 140000, item.Date, pal.accent2, 1050, true, "ctr", "Calibri"))
		sb.WriteString(spText(g, fmt.Sprintf("timelineTitle%d", i), cx-600000, cardY+220000, 1200000, 160000, item.Title, pal.text, 1200, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("timelineDesc%d", i), cx-590000, cardY+380000, 1180000, 160000, item.Desc, pal.muted, 900, false, "t", "Calibri"))
	}
	return wrapSlide(sb.String())
}

func renderCompareDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	left, right := effectiveCompareColumns(s)
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Comparison"), slideLabel(deck, slideIndex, "decision view"), variant)

	leftX, rightX := 457200, 4720000
	colW, colH := 3600000, 4300000
	sb.WriteString(spRoundRect(g, "leftCol", leftX, top+100000, colW, colH, pal.card, pal.border, 10))
	sb.WriteString(spRoundRect(g, "rightCol", rightX, top+100000, colW, colH, pal.card, pal.border, 10))
	sb.WriteString(spRect(g, "leftHead", leftX, top+100000, colW, 320000, pal.border))
	sb.WriteString(spRect(g, "rightHead", rightX, top+100000, colW, 320000, pal.accent))
	sb.WriteString(spText(g, "leftTitle", leftX, top+100000, colW, 320000, firstNonEmpty(left.Heading, "Before"), pal.text, 1500, true, "ctr", "Calibri"))
	sb.WriteString(spText(g, "rightTitle", rightX, top+100000, colW, 320000, firstNonEmpty(right.Heading, "After"), pal.text, 1500, true, "ctr", "Calibri"))
	sb.WriteString(spRightArrow(g, "compareArrow", 4200000, top+1700000, 700000, 300000, pal.accent2))

	renderComparePoints(g, &sb, left.Points, leftX, top+520000, colW, pal, false)
	renderComparePoints(g, &sb, right.Points, rightX, top+520000, colW, pal, true)
	return wrapSlide(sb.String())
}

func renderTableDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	top := renderHeadline(g, &sb, pal, slideHeadingOrFallback(s, "Structured View"), slideLabel(deck, slideIndex, "matrix"), variant)
	table := s.Table
	if table == nil || len(table.Headers) == 0 {
		table = &pptxTableData{
			Headers: []string{"Option", "Time", "Cost"},
			Rows: [][]string{
				{"Manual", "5 days", "$80k"},
				{"Hybrid", "2 days", "$35k"},
				{"Automated", "4 hours", "$12k"},
			},
		}
	}

	cols := len(table.Headers)
	if cols == 0 {
		return wrapSlide(sb.String())
	}

	tableX, tableY := 457200, top+120000
	tableW := 6000000
	if variant == 2 {
		tableW = 5400000
	}
	colW := tableW / cols
	rowH := 480000
	for i, header := range table.Headers {
		x := tableX + i*colW
		sb.WriteString(spRect(g, fmt.Sprintf("hdr%d", i), x, tableY, colW, rowH, pal.accent))
		sb.WriteString(spText(g, fmt.Sprintf("hdrText%d", i), x, tableY, colW, rowH, header, pal.text, 1300, true, "ctr", "Calibri"))
	}
	for r, row := range table.Rows {
		fill := pal.card
		if r%2 == 1 {
			fill = pal.border
		}
		for c, value := range row {
			x := tableX + c*colW
			y := tableY + rowH + r*rowH
			sb.WriteString(spRect(g, fmt.Sprintf("cell%d_%d", r, c), x, y, colW, rowH, fill))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("cellText%d_%d", r, c), x+40000, y+40000, colW-80000, rowH-80000, value, pal.text, 1100, false, "ctr", "Calibri"))
		}
	}

	if variant == 2 {
		panelX := tableX + tableW + 220000
		sb.WriteString(spRoundRect(g, "tablePanel", panelX, tableY, 1950000, 2000000, pal.card, pal.border, 10))
		sb.WriteString(spText(g, "panelTitle", panelX+50000, tableY+120000, 1850000, 220000, "Decision Signal", pal.accent2, 1400, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, "panelBody", panelX+80000, tableY+500000, 1790000, 1200000, summarizeTable(table), pal.muted, 1050, false, "t", "Calibri"))
	}

	return wrapSlide(sb.String())
}

func renderSectionSlide(s pptxSlide, pal pptxPalette, variant int, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	heading := slideHeadingOrFallback(s, "Section")
	sb.WriteString(spRect(g, "sectionAccent", 0, 1500000, 9144000, 180000, pal.accent))
	sb.WriteString(spTextLeft(g, "sectionBadge", 457200, 1280000, 1800000, 180000, fmt.Sprintf("Section %d", slideIndex+1), pal.accent2, 1100, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "sectionHeading", 457200, 1850000, 6200000, 1200000, heading, pal.text, 4200, true, "t", "Calibri Light"))
	if len(s.Points) > 0 {
		sb.WriteString(spTextLeft(g, "sectionBody", 457200, 3600000, 4300000, 700000, s.Points[0], pal.muted, 1600, false, "t", "Calibri"))
	}
	return wrapSlide(sb.String())
}

func renderBlankDeckSlide(s pptxSlide, pal pptxPalette, variant int, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	renderBackdrop(g, &sb, pal, variant)
	body := slideHeadingOrFallback(s, "Section Break")
	if len(s.Points) > 0 && strings.TrimSpace(s.Points[0]) != "" {
		body = s.Points[0]
	}
	sb.WriteString(spRoundRect(g, "blankCard", 1200000, 1800000, 6744000, 2600000, pal.card, pal.border, 10))
	sb.WriteString(spText(g, "blankMark", 1200000, 2100000, 6744000, 400000, "PAUSE / TRANSITION", pal.accent2, 1200, true, "ctr", "Calibri"))
	sb.WriteString(spText(g, "blankBody", 1500000, 2700000, 6144000, 1200000, body, pal.text, 2600, true, "ctr", "Calibri Light"))
	return wrapSlide(sb.String())
}

func renderShareChart(g *idg, sb *strings.Builder, s pptxSlide, pal pptxPalette, top int) {
	categories := chartCategoriesOrFallback(s)
	series := effectiveChartSeries(s)
	if len(series) == 0 {
		series = []pptxChartSeries{{Name: "Share", Values: []float64{38, 27, 19, 16}}}
	}
	values := normalizedChartValues(series[0].Values, len(categories))
	total := sumFloats(values)
	if total <= 0 {
		total = 1
	}

	startY := top + 240000
	for i, cat := range categories {
		y := startY + i*820000
		value := values[i]
		pct := int((value / total) * 100)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("shareCard%d", i), 700000, y, 7600000, 560000, pal.card, pal.border, 10))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("shareName%d", i), 860000, y+80000, 2200000, 180000, cat, pal.text, 1350, true, "t", "Calibri"))
		sb.WriteString(spRoundRect(g, fmt.Sprintf("shareTrack%d", i), 860000, y+300000, 5000000, 50000, pal.border, "", 0))
		sb.WriteString(spRoundRect(g, fmt.Sprintf("shareFill%d", i), 860000, y+300000, maxInt(400000, 5000000*pct/100), 50000, pal.accent, "", 0))
		sb.WriteString(spText(g, fmt.Sprintf("sharePct%d", i), 6000000, y+140000, 1000000, 160000, fmt.Sprintf("%d%%", pct), pal.accent2, 1400, true, "ctr", "Calibri"))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("shareValue%d", i), 7060000, y+130000, 500000, 200000, formatChartNumber(value), pal.muted, 1050, false, "t", "Calibri"))
	}
}

func renderSeriesChart(g *idg, sb *strings.Builder, s pptxSlide, pal pptxPalette, top int, variant int) {
	categories := chartCategoriesOrFallback(s)
	series := effectiveChartSeries(s)
	if len(series) == 0 {
		series = []pptxChartSeries{{Name: "Series", Values: []float64{28, 44, 36, 58}}}
	}
	values := normalizedChartValues(series[0].Values, len(categories))
	second := []float64{}
	if len(series) > 1 {
		second = normalizedChartValues(series[1].Values, len(categories))
	}

	chartX, chartY := 760000, top+360000
	chartW, chartH := 6200000, 3000000
	sb.WriteString(spRoundRect(g, "chartPanel", chartX-140000, chartY-160000, chartW+280000, chartH+520000, pal.card, pal.border, 10))
	for i := 0; i < 5; i++ {
		y := chartY + i*(chartH/4)
		sb.WriteString(spRect(g, fmt.Sprintf("grid%d", i), chartX, y, chartW, 6350, pal.border))
	}

	maxValue := maxFloat(values)
	if len(second) > 0 {
		maxValue = maxFloat(append(values[:0:0], append(values, second...)...))
	}
	if maxValue <= 0 {
		maxValue = 1
	}

	barGap := 180000
	barW := (chartW - barGap*(len(categories)-1)) / len(categories)
	for i, category := range categories {
		x := chartX + i*(barW+barGap)
		h := int((values[i] / maxValue) * float64(chartH-300000))
		sb.WriteString(spRoundRect(g, fmt.Sprintf("bar%d", i), x, chartY+chartH-h, barW, h, pal.accent, "", 0))
		if len(second) > i {
			h2 := int((second[i] / maxValue) * float64(chartH-300000))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("bar2_%d", i), x+barW/3, chartY+chartH-h2, barW/3, h2, pal.accent2, "", 0))
		}
		sb.WriteString(spText(g, fmt.Sprintf("barVal%d", i), x, chartY+chartH-h-180000, barW, 140000, formatChartNumber(values[i]), pal.text, 900, true, "ctr", "Calibri"))
		sb.WriteString(spText(g, fmt.Sprintf("barLabel%d", i), x-60000, chartY+chartH+60000, barW+120000, 160000, category, pal.muted, 900, false, "ctr", "Calibri"))
	}

	insightX := 7160000
	insightH := ternary(variant == 2, 2300000, 1800000)
	sb.WriteString(spRoundRect(g, "insightCard", insightX, chartY, 1500000, insightH, pal.card, pal.border, 10))
	sb.WriteString(spText(g, "insightTitle", insightX, chartY+100000, 1500000, 180000, firstNonEmpty(series[0].Name, "Signal"), pal.accent2, 1250, true, "ctr", "Calibri"))
	sb.WriteString(spTextLeft(g, "insightBody", insightX+70000, chartY+420000, 1360000, insightH-500000, chartInsight(categories, values), pal.muted, 980, false, "t", "Calibri"))
}

func renderComparePoints(g *idg, sb *strings.Builder, points []string, x, y, width int, pal pptxPalette, positive bool) {
	for i, point := range safePoints(points, 5) {
		py := y + i*650000
		accent := pal.border
		if positive {
			accent = pal.accent
		}
		sb.WriteString(spEllipse(g, fmt.Sprintf("cmpDot%d", i), x+70000, py+80000, 180000, 180000, accent, 100, "", 0, 0))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("cmpText%d", i), x+320000, py, width-420000, 280000, point, pal.text, 1150, false, "t", "Calibri"))
	}
}

func effectiveStats(s pptxSlide) []pptxStat {
	if len(s.Stats) > 0 {
		return s.Stats
	}
	var stats []pptxStat
	numRe := regexp.MustCompile(`^\s*[\d$€£>~%]`)
	for _, point := range s.Points {
		parts := strings.SplitN(point, ":", 2)
		if len(parts) == 2 && numRe.MatchString(strings.TrimSpace(parts[0])) {
			stats = append(stats, pptxStat{
				Value: strings.TrimSpace(parts[0]),
				Label: strings.TrimSpace(parts[1]),
			})
			continue
		}
	}
	return stats
}

func effectiveSteps(s pptxSlide) []string {
	if len(s.Steps) > 0 {
		return safePoints(s.Steps, 6)
	}
	return safePoints(s.Points, 6)
}

func effectiveCards(s pptxSlide) []pptxCard {
	if len(s.Cards) > 0 {
		return s.Cards
	}
	var cards []pptxCard
	for i, point := range safePoints(s.Points, 6) {
		title, desc := splitCardText(point)
		cards = append(cards, pptxCard{Icon: inferIconFromText(title), Title: title, Desc: desc})
		if i == 5 {
			break
		}
	}
	return cards
}

func effectiveCompareColumns(s pptxSlide) (pptxCompareColumn, pptxCompareColumn) {
	left := pptxCompareColumn{Heading: "Current State", Points: []string{"Manual steps", "Slow turnaround", "Low visibility"}}
	right := pptxCompareColumn{Heading: "Future State", Points: []string{"Automated flow", "Faster delivery", "Clear metrics"}}
	if s.LeftColumn != nil {
		left = *s.LeftColumn
	}
	if s.RightColumn != nil {
		right = *s.RightColumn
	}
	return left, right
}

func chartCategoriesOrFallback(s pptxSlide) []string {
	if len(s.ChartCategories) > 0 {
		return trimAndPadLabels(s.ChartCategories, len(s.ChartCategories))
	}
	if len(s.Timeline) > 0 {
		var labels []string
		for _, item := range s.Timeline {
			labels = append(labels, item.Date)
		}
		return trimAndPadLabels(labels, len(labels))
	}
	return []string{"Q1", "Q2", "Q3", "Q4"}
}

func effectiveChartSeries(s pptxSlide) []pptxChartSeries {
	if len(s.ChartSeries) > 0 {
		return s.ChartSeries
	}
	stats := effectiveStats(s)
	if len(stats) > 0 {
		values := make([]float64, 0, len(stats))
		labels := make([]string, 0, len(stats))
		for _, stat := range stats {
			values = append(values, parseNumericValue(stat.Value))
			labels = append(labels, stat.Label)
		}
		s.ChartCategories = labels
		return []pptxChartSeries{{Name: "Metrics", Values: values}}
	}
	return nil
}

func splitCardText(point string) (string, string) {
	if idx := strings.Index(point, ":"); idx > 0 && idx < 44 {
		return strings.TrimSpace(point[:idx]), strings.TrimSpace(point[idx+1:])
	}
	parts := strings.Fields(point)
	if len(parts) <= 5 {
		return point, ""
	}
	return strings.Join(parts[:minInt(4, len(parts))], " "), strings.Join(parts[minInt(4, len(parts)):], " ")
}

func slideLabel(deck pptxDeckContext, slideIndex int, fallback string) string {
	return fmt.Sprintf("%s • %d/%d", firstNonEmpty(fallback, "slide"), slideIndex+2, deck.SlideCount+1)
}

func safePoints(points []string, limit int) []string {
	var out []string
	for _, point := range points {
		point = strings.TrimSpace(point)
		if point == "" {
			continue
		}
		out = append(out, point)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func trimAndPadLabels(values []string, count int) []string {
	if count <= 0 {
		count = len(values)
	}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	for len(out) < count {
		out = append(out, fmt.Sprintf("Item %d", len(out)+1))
	}
	return out[:count]
}

func normalizedChartValues(values []float64, count int) []float64 {
	out := make([]float64, 0, count)
	for _, value := range values {
		out = append(out, value)
		if len(out) == count {
			return out
		}
	}
	for len(out) < count {
		out = append(out, 0)
	}
	return out
}

func parsePercent(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	n, _ := strconv.Atoi(value)
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func parseNumericValue(value string) float64 {
	value = strings.TrimSpace(strings.ToUpper(value))
	value = strings.TrimPrefix(value, "$")
	value = strings.TrimSuffix(value, "%")
	mult := 1.0
	switch {
	case strings.HasSuffix(value, "K"):
		mult = 1_000
		value = strings.TrimSuffix(value, "K")
	case strings.HasSuffix(value, "M"):
		mult = 1_000_000
		value = strings.TrimSuffix(value, "M")
	case strings.HasSuffix(value, "B"):
		mult = 1_000_000_000
		value = strings.TrimSuffix(value, "B")
	case strings.HasSuffix(value, "X"):
		value = strings.TrimSuffix(value, "X")
	}
	n, _ := strconv.ParseFloat(value, 64)
	return n * mult
}

func formatChartNumber(value float64) string {
	switch {
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", value/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", value/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.0fK", value/1_000)
	case value == float64(int64(value)):
		return fmt.Sprintf("%.0f", value)
	default:
		return fmt.Sprintf("%.1f", value)
	}
}

func chartInsight(categories []string, values []float64) string {
	if len(categories) == 0 || len(values) == 0 {
		return "No data available."
	}
	bestIdx := 0
	for i := 1; i < len(values); i++ {
		if values[i] > values[bestIdx] {
			bestIdx = i
		}
	}
	return fmt.Sprintf("Peak category: %s\nTop value: %s\nUse this slide to explain the driver behind the leading segment.", categories[bestIdx], formatChartNumber(values[bestIdx]))
}

func summarizeTable(table *pptxTableData) string {
	if table == nil || len(table.Rows) == 0 {
		return "Summarize the comparison and recommend the best option for the audience."
	}
	best := table.Rows[len(table.Rows)-1]
	return fmt.Sprintf("The strongest option appears in the final row: %s. Use the matrix to explain why that option outperforms the rest.", strings.Join(best, " • "))
}

func sumFloats(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func maxFloat(values []float64) float64 {
	max := 0.0
	for i, value := range values {
		if i == 0 || value > max {
			max = value
		}
	}
	return max
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
