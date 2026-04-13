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

type pptxResolvedDeckDesign struct {
	Composition   string
	Density       string
	ShapeLanguage string
	AccentMode    string
	HeroLayout    string
}

type pptxResolvedSlideDesign struct {
	LayoutStyle string
	PanelStyle  string
	AccentMode  string
	Density     string
	VisualFocus string
}

type pptxSlideShell struct {
	FrameX int
	FrameY int
	FrameW int
	FrameH int
	BodyX  int
	BodyY  int
	BodyW  int
	BodyH  int
}

const (
	pptxHeadingFont = "Arial"
	pptxBodyFont    = "Arial"
)

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
	switch resolveDeckDesign(deck).Composition {
	case "frame":
		return 1
	case "asym":
		return 2
	case "band":
		return 3
	case "float":
		return 4
	case "gallery":
		return 5
	default:
		return 0
	}
}

func renderDeckSlide(s pptxSlide, pal pptxPalette, deck pptxDeckContext, slideIndex int) string {
	layout := effectivePPTXLayout(s)
	variant := slideVisualVariant(deck.Title, s.Heading, layout, slideIndex)
	if s.Design != nil {
		switch strings.ToLower(strings.TrimSpace(s.Design.LayoutStyle)) {
		case "rail", "stack", "stage":
			variant = 1
		case "split", "grid", "matrix", "spotlight":
			variant = 2
		}
	}

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

func resolveDeckDesign(deck pptxDeckContext) pptxResolvedDeckDesign {
	design := deck.DeckPlan.Design
	resolved := pptxResolvedDeckDesign{
		Composition:   strings.ToLower(strings.TrimSpace(firstNonEmpty(design.Composition, "split"))),
		Density:       strings.ToLower(strings.TrimSpace(firstNonEmpty(design.Density, "balanced"))),
		ShapeLanguage: strings.ToLower(strings.TrimSpace(firstNonEmpty(design.ShapeLanguage, "mixed"))),
		AccentMode:    strings.ToLower(strings.TrimSpace(firstNonEmpty(design.AccentMode, "rail"))),
		HeroLayout:    strings.ToLower(strings.TrimSpace(firstNonEmpty(design.HeroLayout, "motif"))),
	}
	return resolved
}

func resolveSlideDesign(deck pptxDeckContext, s pptxSlide) pptxResolvedSlideDesign {
	base := resolveDeckDesign(deck)
	design := pptxResolvedSlideDesign{
		LayoutStyle: strings.ToLower(strings.TrimSpace(firstNonEmpty(base.Composition, "split"))),
		PanelStyle:  "soft",
		AccentMode:  strings.ToLower(strings.TrimSpace(firstNonEmpty(base.AccentMode, "rail"))),
		Density:     strings.ToLower(strings.TrimSpace(firstNonEmpty(base.Density, "balanced"))),
		VisualFocus: "text",
	}
	if s.Design == nil {
		return design
	}
	design.LayoutStyle = strings.ToLower(strings.TrimSpace(firstNonEmpty(s.Design.LayoutStyle, design.LayoutStyle)))
	design.PanelStyle = strings.ToLower(strings.TrimSpace(firstNonEmpty(s.Design.PanelStyle, design.PanelStyle)))
	design.AccentMode = strings.ToLower(strings.TrimSpace(firstNonEmpty(s.Design.AccentMode, design.AccentMode)))
	design.Density = strings.ToLower(strings.TrimSpace(firstNonEmpty(s.Design.Density, design.Density)))
	design.VisualFocus = strings.ToLower(strings.TrimSpace(firstNonEmpty(s.Design.VisualFocus, design.VisualFocus)))
	return design
}

func slideCornerRadius(deck pptxDeckContext, large bool) int {
	shape := resolveDeckDesign(deck).ShapeLanguage
	switch shape {
	case "crisp":
		if large {
			return 6
		}
		return 4
	case "soft":
		if large {
			return 24
		}
		return 18
	default:
		if large {
			return 16
		}
		return 10
	}
}

func renderSlideShell(g *idg, sb *strings.Builder, pal pptxPalette, deck pptxDeckContext, s pptxSlide, slideIndex int) pptxSlideShell {
	design := resolveSlideDesign(deck, s)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))

	frameX, frameY := 420000, 420000
	frameW, frameH := 8304000, 5980000
	frameFill := pal.card
	if design.PanelStyle == "tint" || design.PanelStyle == "glass" {
		frameFill = deckSurface(pal)
	}
	sb.WriteString(spRoundRect(g, "frame", frameX, frameY, frameW, frameH, frameFill, deckBorder(pal), slideCornerRadius(deck, true)))

	switch design.AccentMode {
	case "band":
		sb.WriteString(spRect(g, "accentBand", frameX, frameY, frameW, 48000, pal.accent))
	case "block":
		sb.WriteString(spRect(g, "accentBlock", frameX+frameW-760000, frameY, 760000, frameH, deckAccentWash(pal, 0.1)))
	case "glow":
		sb.WriteString(spEllipse(g, "accentGlow", frameX+frameW-1200000, frameY-240000, 1500000, 1500000, deckAccentWash(pal, 0.18), 100, "", 0, 0))
	default:
		sb.WriteString(spRect(g, "accentRail", frameX, frameY, 50000, frameH, pal.accent))
	}

	pillW := 1440000
	if design.AccentMode == "chip" || design.AccentMode == "marker" {
		pillW = 1600000
	}
	pillY := frameY + 200000
	pillFill := deckAccentWash(pal, 0.14)
	if design.AccentMode == "band" {
		pillFill = deckAccent2Wash(pal, 0.16)
	}
	sb.WriteString(spRoundRect(g, "metaPill", frameX+260000, pillY, pillW, 250000, pillFill, "", 0))
	sb.WriteString(spText(g, "metaText", frameX+260000, pillY, pillW, 250000, slideLabel(deck, slideIndex, effectivePPTXLayout(s)), pal.accent, 980, true, "ctr", pptxBodyFont))

	headingX := frameX + 260000
	headingY := pillY + 360000
	headingW := frameW - 520000
	headingH := 560000
	if design.Density == "dense" {
		headingH = 660000
	}
	sb.WriteString(spTextLeft(g, "heading", headingX, headingY, headingW, headingH, slideHeadingOrFallback(s, "Untitled Slide"), pal.text, 2500, true, "t", pptxHeadingFont))

	bodyY := headingY + headingH + 180000
	bodyH := frameH - (bodyY - frameY) - 240000
	return pptxSlideShell{
		FrameX: frameX,
		FrameY: frameY,
		FrameW: frameW,
		FrameH: frameH,
		BodyX:  frameX + 260000,
		BodyY:  bodyY,
		BodyW:  frameW - 520000,
		BodyH:  bodyH,
	}
}

func slidePanelFill(pal pptxPalette, design pptxResolvedSlideDesign, emphasis int) string {
	switch design.PanelStyle {
	case "solid":
		return pal.card
	case "outline":
		return mixHex(pal.bg, pal.card, 0.92)
	case "tint":
		if emphasis%2 == 0 {
			return deckAccentWash(pal, 0.12)
		}
		return deckAccent2Wash(pal, 0.12)
	case "glass":
		return mixHex(pal.bg, pal.card, 0.82)
	default:
		if emphasis%2 == 0 {
			return deckSurface(pal)
		}
		return mixHex(pal.bg, pal.card, 0.92)
	}
}

func slidePanelBorder(pal pptxPalette, design pptxResolvedSlideDesign) string {
	if design.PanelStyle == "solid" {
		return pal.border
	}
	return deckBorder(pal)
}

func renderPPTXCoverSlide(deck pptxDeckContext, pal pptxPalette) string {
	g := &idg{}
	var sb strings.Builder
	renderPreviewAlignedCover(g, &sb, deck, pal)

	return wrapSlide(sb.String())
}

func renderPreviewAlignedCover(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	design := resolveDeckDesign(deck)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))

	cardX, cardY := 360000, 500000
	cardW, cardH := 8424000, 5600000
	sb.WriteString(spRoundRect(g, "coverCard", cardX, cardY, cardW, cardH, pal.card, deckBorder(pal), slideCornerRadius(deck, true)))

	panelX, panelY := cardX+260000, cardY+260000
	panelW, panelH := 4700000, cardH-520000
	figureX := cardX + cardW - 2500000
	figureY := panelY
	figureW := 2240000
	figureH := panelH
	if design.Composition == "band" {
		sb.WriteString(spRect(g, "coverBand", cardX, cardY, 1380000, cardH, deckAccentWash(pal, 0.18)))
		panelX = cardX + 1660000
		panelW = 3800000
		figureX = cardX + cardW - 2200000
		figureW = 1880000
	}
	if design.Composition == "frame" {
		panelW = 4300000
		figureX = cardX + 5440000
		figureW = 2100000
	}
	if design.Composition == "float" || design.Composition == "gallery" {
		panelW = 4200000
		figureX = cardX + 5580000
		figureW = 2000000
	}

	accentRuleW := 1740000
	if design.AccentMode == "band" {
		accentRuleW = 2200000
	}
	sb.WriteString(spRect(g, "coverRule", panelX, panelY+500000, accentRuleW, 32000, pal.accent))
	sb.WriteString(spTextLeft(g, "coverKicker", panelX, panelY, 3000000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 960, true, "t", pptxBodyFont))

	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3300, 15)
	if design.Density == "airy" {
		titleSize += 120
	}
	titleY := panelY + 840000
	titleW := panelW - 400000
	if design.Composition == "band" {
		titleW = panelW - 240000
	}
	sb.WriteString(spTextLeft(g, "coverTitle", panelX, titleY, titleW, titleHeight, firstNonEmpty(deck.Title, "Presentation"), pal.text, titleSize, true, "t", pptxHeadingFont))

	subtitle := coverLead(deck)
	subtitleSize, subtitleHeight := coverSubtitleMetrics(subtitle, 1460, 36)
	subtitleY := titleY + titleHeight + 160000
	if subtitle != "" {
		sb.WriteString(spTextLeft(g, "coverSubtitle", panelX, subtitleY, titleW, subtitleHeight, subtitle, pal.accent2, subtitleSize, true, "t", pptxBodyFont))
	}
	supportY := subtitleY + subtitleHeight + 120000
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "coverSupport", panelX, supportY, titleW, 320000, support, pal.muted, 1080, false, "t", pptxBodyFont))
	}
	subjectY := cardY + cardH - 520000
	sb.WriteString(spTextLeft(g, "coverSubject", panelX, subjectY, titleW, 220000, coverSubjectLine(deck), pal.muted, 920, true, "t", pptxBodyFont))

	renderCoverFigurePanel(g, sb, figureX, figureY, figureW, figureH, deck, pal)
}

func renderCoverFigurePanel(g *idg, sb *strings.Builder, x, y, w, h int, deck pptxDeckContext, pal pptxPalette) {
	design := resolveDeckDesign(deck)
	figureFill := deckAccentWash(pal, 0.1)
	switch normalizeCoverStyleToken(deck.DeckPlan.CoverStyle) {
	case "poster":
		figureFill = deckAccentWash(pal, 0.18)
	case "playful":
		figureFill = deckAccent2Wash(pal, 0.16)
	case "mosaic":
		figureFill = deckAccentWash(pal, 0.12)
	}
	sb.WriteString(spRoundRect(g, "coverFigure", x, y, w, h, figureFill, deckBorder(pal), slideCornerRadius(deck, true)))

	switch design.AccentMode {
	case "glow":
		sb.WriteString(spEllipse(g, "figureGlow", x+w-860000, y-180000, 980000, 980000, deckAccentWash(pal, 0.22), 100, "", 0, 0))
	case "chip":
		sb.WriteString(spRoundRect(g, "figureChip", x+180000, y+180000, 760000, 220000, deckAccentWash(pal, 0.22), "", 0))
	}

	switch design.Composition {
	case "gallery", "asym":
		renderCoverFigureGallery(g, sb, x, y, w, h, pal)
	case "float":
		renderCoverFigureFloat(g, sb, x, y, w, h, deck, pal)
	default:
		renderCoverFigureFocus(g, sb, x, y, w, h, deck, pal)
	}
}

func renderCoverFigureFocus(g *idg, sb *strings.Builder, x, y, w, h int, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spEllipse(g, "focusOrbA", x+w-820000, y+220000, 680000, 680000, deckAccentWash(pal, 0.16), 100, "", 0, 0))
	sb.WriteString(spEllipse(g, "focusOrbB", x+220000, y+h-980000, 760000, 760000, deckAccent2Wash(pal, 0.18), 100, "", 0, 0))
	renderCoverMotif(g, sb, "coverMotif", x+w/2-320000, y+h/2-420000, 640000, pal, coverMotifToken(deck), 22)
}

func renderCoverFigureFloat(g *idg, sb *strings.Builder, x, y, w, h int, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spEllipse(g, "floatOrbA", x+240000, y+260000, 880000, 880000, deckAccentWash(pal, 0.18), 100, "", 0, 0))
	sb.WriteString(spEllipse(g, "floatOrbB", x+w-1020000, y+h-1200000, 760000, 760000, deckAccent2Wash(pal, 0.18), 100, "", 0, 0))
	renderCoverMotif(g, sb, "floatMotifA", x+w/2-420000, y+680000, 520000, pal, coverMotifToken(deck), 24)
	renderCoverMotif(g, sb, "floatMotifB", x+w/2-260000, y+h/2+280000, 420000, pal, coverMotifToken(deck), 20)
}

func renderCoverFigureGallery(g *idg, sb *strings.Builder, x, y, w, h int, pal pptxPalette) {
	tileW := (w - 280000) / 2
	tileH := (h - 360000) / 3
	for row := 0; row < 3; row++ {
		for col := 0; col < 2; col++ {
			tileX := x + col*(tileW+140000)
			tileY := y + row*(tileH+120000)
			fill := deckSurface(pal)
			if (row+col)%2 == 1 {
				fill = deckAccentWash(pal, 0.18)
			}
			if row == 2 && col == 1 {
				fill = deckAccent2Wash(pal, 0.2)
			}
			sb.WriteString(spRoundRect(g, fmt.Sprintf("galleryTile%d_%d", row, col), tileX, tileY, tileW, tileH, fill, deckBorder(pal), 14))
		}
	}
}

func renderCoverEditorialSplit(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "accentRail", 0, 0, 9144000, 38100, pal.accent))
	sb.WriteString(spRect(g, "rightPanel", 6360000, 0, 2784000, 6858000, pal.card))
	sb.WriteString(spEllipse(g, "heroOrb", 6840000, -360000, 2340000, 2340000, deckAccentWash(pal, 0.16), 100, "", 0, 0))
	sb.WriteString(spEllipse(g, "footOrb", 7120000, 4680000, 1500000, 1500000, deckAccent2Wash(pal, 0.14), 100, "", 0, 0))
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
	sb.WriteString(spEllipse(g, "orbitOne", 800000, 720000, 4200000, 4200000, "", 0, deckBorder(pal), 6350, 100))
	sb.WriteString(spEllipse(g, "orbitTwo", 1260000, 1180000, 3340000, 3340000, "", 0, deckBorder(pal), 6350, 100))
	sb.WriteString(spEllipse(g, "orbitThree", 1740000, 1660000, 2380000, 2380000, "", 0, deckBorder(pal), 6350, 100))
	sb.WriteString(spEllipse(g, "accentGlow", 1280000, 2140000, 1400000, 1400000, deckAccentWash(pal, 0.18), 100, "", 0, 0))
	renderCoverMotif(g, sb, "orbitMotif", 2140000, 2600000, 900000, pal, coverMotifToken(deck), 30)
	sb.WriteString(spRoundRect(g, "titlePanel", 4300000, 1120000, 3660000, 3920000, deckSurface(pal), pal.border, 18))
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
	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3300, 12)
	subtitle := coverLead(deck)
	subtitleSize, subtitleHeight := coverSubtitleMetrics(subtitle, 1380, 30)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "topRule", 620000, 980000, 1960000, 38100, pal.accent))
	sb.WriteString(spRoundRect(g, "titlePanel", 620000, 1240000, 3900000, 4260000, deckSurface(pal), pal.border, 18))
	sb.WriteString(spRect(g, "panelAccent", 620000, 1240000, 3900000, 38100, pal.accent))
	tileXs := []int{5060000, 6620000, 5060000, 6620000, 7480000}
	tileYs := []int{1040000, 860000, 2700000, 2700000, 4720000}
	tileWs := []int{1340000, 1820000, 1340000, 1820000, 940000}
	tileHs := []int{1320000, 1540000, 1320000, 1700000, 1060000}
	for i := range tileXs {
		fill := deckSurface(pal)
		switch i {
		case 1:
			fill = deckAccentWash(pal, 0.84)
		case 3:
			fill = deckAccent2Wash(pal, 0.78)
		}
		sb.WriteString(spRoundRect(g, fmt.Sprintf("tile%d", i), tileXs[i], tileYs[i], tileWs[i], tileHs[i], fill, pal.border, 18))
	}
	renderCoverMotif(g, sb, "mosaicMotif", 7060000, 1160000, 980000, pal, coverMotifToken(deck), 30)
	renderCoverMotif(g, sb, "mosaicMotifSmall", 5440000, 3040000, 620000, pal, coverMotifToken(deck), 24)
	sb.WriteString(spTextLeft(g, "titleKicker", 920000, 1600000, 2600000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 1080, true, "t", "Calibri"))
	sb.WriteString(spTextLeft(g, "title", 920000, 2100000, 3120000, titleHeight, firstNonEmpty(deck.Title, "Presentation"), pal.text, titleSize, true, "t", "Calibri Light"))
	subtitleY := 2100000 + titleHeight + 180000
	if subtitle != "" {
		sb.WriteString(spTextLeft(g, "subtitle", 920000, subtitleY, 3100000, subtitleHeight, subtitle, pal.accent2, subtitleSize, true, "t", "Calibri"))
	}
	supportY := subtitleY + subtitleHeight + 160000
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "support", 920000, supportY, 3080000, 500000, support, pal.muted, 1080, false, "t", "Calibri"))
	}
	sb.WriteString(spRect(g, "footerRule", 920000, 5020000, 1680000, 19050, pal.accent2))
	sb.WriteString(spTextLeft(g, "subjectLine", 920000, 5140000, 3080000, 260000, coverSubjectLine(deck), pal.muted, 920, true, "t", "Calibri"))
}

func renderCoverStudioPoster(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3600, 18)
	subtitle := coverLead(deck)
	subtitleSize, subtitleHeight := coverSubtitleMetrics(subtitle, 1500, 34)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	sb.WriteString(spRect(g, "posterBand", 0, 0, 2480000, 6858000, pal.accent))
	sb.WriteString(spRect(g, "posterRail", 2480000, 0, 300000, 6858000, pal.accent2))
	for i := 0; i < 3; i++ {
		y := 980000 + i*1460000
		sb.WriteString(spRoundRect(g, fmt.Sprintf("posterStrip%d", i), 6460000, y, 1980000, 1080000, deckSurface(pal), pal.border, 18))
	}
	renderCoverMotif(g, sb, "posterMotif", 6740000, 1160000, 960000, pal, coverMotifToken(deck), 30)
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
	sb.WriteString(spEllipse(g, "playOrbOne", 540000, 720000, 1400000, 1400000, deckAccentWash(pal, 0.16), 100, "", 0, 0))
	sb.WriteString(spEllipse(g, "playOrbTwo", 6780000, 820000, 1640000, 1640000, deckAccent2Wash(pal, 0.18), 100, "", 0, 0))
	sb.WriteString(spEllipse(g, "playOrbThree", 7340000, 4640000, 1280000, 1280000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
	sb.WriteString(spRoundRect(g, "titleCard", 860000, 1820000, 4480000, 2800000, deckSurface(pal), pal.border, 18))
	sb.WriteString(spRect(g, "titleAccent", 860000, 1820000, 4480000, 38100, pal.accent))
	sb.WriteString(spRoundRect(g, "stickerOne", 1120000, 1240000, 2200000, 320000, pal.accent, "", 0))
	sb.WriteString(spText(g, "titleKicker", 1120000, 1240000, 2200000, 320000, strings.ToUpper(coverKicker(deck)), pal.text, 1120, true, "ctr", "Calibri"))
	renderCoverMotif(g, sb, "playMotifOne", 5640000, 1880000, 1080000, pal, coverMotifToken(deck), 30)
	renderCoverMotif(g, sb, "playMotifTwo", 7040000, 3200000, 820000, pal, coverMotifToken(deck), 26)
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
	sb.WriteString(spEllipse(g, name+"Bg", x, y, size, size, deckAccentWash(pal, float64(fillAlpha)/100), 100, deckAccent2Wash(pal, 0.62), 6350, 100))
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

func coverTitleMetrics(title string, base, charsPerLine int) (int, int) {
	lines := estimateCoverLines(title, charsPerLine)
	size := base
	switch {
	case lines >= 4:
		size -= 700
	case lines == 3:
		size -= 380
	case lines == 2:
		size -= 120
	}
	return size, maxInt(860000, lines*440000+140000)
}

func coverSubtitleMetrics(subtitle string, base, charsPerLine int) (int, int) {
	lines := estimateCoverLines(subtitle, charsPerLine)
	size := base
	switch {
	case lines >= 3:
		size -= 180
	case lines == 2:
		size -= 80
	}
	return size, maxInt(340000, lines*220000+80000)
}

func estimateCoverLines(text string, charsPerLine int) int {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" || charsPerLine <= 0 {
		return 1
	}
	words := strings.Fields(text)
	lines := 1
	lineLen := 0
	for _, word := range words {
		wordLen := len([]rune(word))
		if wordLen > charsPerLine {
			if lineLen > 0 {
				lines++
				lineLen = 0
			}
			lines += (wordLen - 1) / charsPerLine
			continue
		}
		if lineLen == 0 {
			lineLen = wordLen
			continue
		}
		if lineLen+1+wordLen > charsPerLine {
			lines++
			lineLen = wordLen
			continue
		}
		lineLen += 1 + wordLen
	}
	return maxInt(lines, 1)
}

func deckSurface(pal pptxPalette) string {
	return mixHex(pal.bg, pal.card, 0.88)
}

func deckBorder(pal pptxPalette) string {
	return mixHex(pal.bg, pal.border, 0.72)
}

func deckAccentWash(pal pptxPalette, strength float64) string {
	return mixHex(pal.bg, pal.accent, strength)
}

func deckAccent2Wash(pal pptxPalette, strength float64) string {
	return mixHex(pal.bg, pal.accent2, strength)
}

func renderBackdrop(g *idg, sb *strings.Builder, pal pptxPalette, variant int) int {
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))

	switch variant {
	case 0:
		sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 38100, pal.accent))
		sb.WriteString(spEllipse(g, "orbTop", 7100000, -700000, 2500000, 2500000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
		sb.WriteString(spEllipse(g, "orbBottom", -300000, 5600000, 1200000, 1200000, deckAccent2Wash(pal, 0.12), 100, "", 0, 0))
	case 1:
		sb.WriteString(spRect(g, "leftRail", 0, 0, 260000, 6858000, pal.accent))
		sb.WriteString(spRoundRect(g, "haloCard", 6640000, 440000, 1900000, 720000, deckSurface(pal), pal.border, 18))
		sb.WriteString(spEllipse(g, "halo", 7420000, 5180000, 1400000, 1400000, deckAccent2Wash(pal, 0.14), 100, "", 0, 0))
	default:
		sb.WriteString(spRect(g, "rightPanel", 7480000, 0, 1664000, 6858000, pal.card))
		sb.WriteString(spEllipse(g, "topGlow", 6400000, -500000, 1800000, 1800000, deckAccentWash(pal, 0.14), 100, "", 0, 0))
		sb.WriteString(spEllipse(g, "midGlow", 7800000, 2300000, 900000, 900000, deckAccent2Wash(pal, 0.16), 100, "", 0, 0))
	}

	return 840000
}

func renderHeadline(g *idg, sb *strings.Builder, pal pptxPalette, heading, kicker string, variant int) int {
	switch variant {
	case 0:
		sb.WriteString(spTextLeft(g, "heading", 457200, 140000, 7200000, 420000, heading, pal.text, 3000, true, "ctr", "Calibri Light"))
		sb.WriteString(spRect(g, "headingLine", 457200, 610000, 1600000, 22225, pal.accent))
		if kicker != "" {
			sb.WriteString(spRoundRect(g, "kickerPill", 457200, 640000, 1700000, 220000, deckAccentWash(pal, 0.16), "", 0))
			sb.WriteString(spText(g, "kicker", 457200, 640000, 1700000, 220000, kicker, pal.accent, 1040, true, "ctr", "Calibri"))
		}
		return 900000
	case 1:
		sb.WriteString(spRoundRect(g, "headingCard", 520000, 160000, 6100000, 560000, deckSurface(pal), pal.border, 18))
		sb.WriteString(spRect(g, "headingAccent", 520000, 160000, 90000, 560000, pal.accent))
		sb.WriteString(spTextLeft(g, "heading", 700000, 230000, 5600000, 300000, heading, pal.text, 2800, true, "ctr", "Calibri Light"))
		if kicker != "" {
			sb.WriteString(spTextLeft(g, "kicker", 700000, 500000, 2600000, 120000, kicker, pal.muted, 1050, false, "t", "Calibri"))
		}
		return 980000
	default:
		sb.WriteString(spTextLeft(g, "heading", 457200, 180000, 6400000, 430000, heading, pal.text, 3100, true, "ctr", "Calibri Light"))
		sb.WriteString(spRoundRect(g, "badge", 457200, 620000, 1600000, 220000, deckAccentWash(pal, 0.14), "", 0))
		sb.WriteString(spText(g, "kicker", 457200, 620000, 1600000, 220000, firstNonEmpty(kicker, "Slide"), pal.accent, 1080, true, "ctr", "Calibri"))
		return 980000
	}
}

func renderBulletsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
	points := safePoints(s.Points, 6)
	if len(points) == 0 {
		points = []string{"Key insight", "Supporting evidence", "Operational implication"}
	}

	if design.LayoutStyle == "split" && len(points) >= 4 {
		colW := (shell.BodyW - 180000) / 2
		gap := 180000
		cardH := 700000
		for i, pt := range points {
			col := i % 2
			row := i / 2
			x := shell.BodyX + col*(colW+gap)
			y := shell.BodyY + row*(cardH+140000)
			sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), x, y, colW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
			sb.WriteString(spEllipse(g, fmt.Sprintf("pointBadge%d", i), x+50000, y+150000, 220000, 220000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("pointNum%d", i), x+50000, y+150000, 220000, 220000, fmt.Sprintf("%d", i+1), pal.text, 1100, true, "ctr", pptxBodyFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), x+320000, y+90000, colW-400000, cardH-180000, pt, pal.text, 1240, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}

	cardW := shell.BodyW
	cardH := 560000
	for i, pt := range points {
		y := shell.BodyY + i*(cardH+100000)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), shell.BodyX, y, cardW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spEllipse(g, fmt.Sprintf("pointBadge%d", i), shell.BodyX+50000, y+170000, 200000, 200000, pal.accent, 100, "", 0, 0))
		sb.WriteString(spText(g, fmt.Sprintf("pointNum%d", i), shell.BodyX+50000, y+170000, 200000, 200000, fmt.Sprintf("%d", i+1), pal.text, 1050, true, "ctr", pptxBodyFont))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), shell.BodyX+320000, y+100000, cardW-420000, cardH-160000, pt, pal.text, 1260, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderStatsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
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
	totalW := shell.BodyW
	gapX := 200000
	gapY := 180000
	cardW := (totalW - gapX*(cols-1)) / cols
	cardH := (shell.BodyH - gapY*(rows-1)) / rows
	if rows == 2 {
		cardH = minInt(cardH, 1260000)
	}

	for i, stat := range stats {
		col := i % cols
		row := i / cols
		x := shell.BodyX + col*(cardW+gapX)
		y := shell.BodyY + row*(cardH+gapY)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("statCard%d", i), x, y, cardW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spText(g, fmt.Sprintf("statValue%d", i), x+20000, y+130000, cardW-40000, 420000, stat.Value, pal.accent2, 3000, true, "ctr", pptxHeadingFont))
		sb.WriteString(spText(g, fmt.Sprintf("statLabel%d", i), x+20000, y+560000, cardW-40000, 200000, stat.Label, pal.text, 1300, true, "ctr", pptxBodyFont))
		if stat.Desc != "" {
			sb.WriteString(spTextLeft(g, fmt.Sprintf("statDesc%d", i), x+50000, y+820000, cardW-100000, 260000, stat.Desc, pal.muted, 1020, false, "t", pptxBodyFont))
		}
		if strings.Contains(stat.Value, "%") {
			pct := parsePercent(stat.Value)
			barW := (cardW - 140000) * pct / 100
			sb.WriteString(spRoundRect(g, fmt.Sprintf("track%d", i), x+70000, y+cardH-160000, cardW-140000, 50000, mixHex(pal.bg, pal.border, 0.18), "", 0))
			if barW > 0 {
				sb.WriteString(spRoundRect(g, fmt.Sprintf("fill%d", i), x+70000, y+cardH-160000, barW, 50000, pal.accent, "", 0))
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
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)

	if design.LayoutStyle == "rail" {
		lineX := shell.BodyX + 180000
		sb.WriteString(spRect(g, "roadmapLine", lineX, shell.BodyY, 16000, shell.BodyH, deckBorder(pal)))
		for i, step := range steps {
			y := shell.BodyY + i*820000
			sb.WriteString(spEllipse(g, fmt.Sprintf("stepNode%d", i), lineX-130000, y+140000, 260000, 260000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", i), lineX-130000, y+140000, 260000, 260000, fmt.Sprintf("%d", i+1), pal.text, 1100, true, "ctr", pptxBodyFont))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("stepCard%d", i), shell.BodyX+420000, y, shell.BodyW-420000, 520000, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("stepText%d", i), shell.BodyX+580000, y+100000, shell.BodyW-640000, 300000, step, pal.text, 1240, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}

	stepCount := len(steps)
	if stepCount > 5 {
		stepCount = 5
		steps = steps[:5]
	}
	totalW := shell.BodyW
	gap := 120000
	boxW := (totalW - gap*(stepCount-1)) / stepCount
	for i, step := range steps {
		x := shell.BodyX + i*(boxW+gap)
		y := shell.BodyY + 320000
		sb.WriteString(spRoundRect(g, fmt.Sprintf("stepShape%d", i), x, y, boxW, 1180000, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spEllipse(g, fmt.Sprintf("stepBadge%d", i), x+boxW/2-120000, y+120000, 240000, 240000, pal.accent, 100, "", 0, 0))
		sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", i), x+boxW/2-120000, y+120000, 240000, 240000, fmt.Sprintf("%d", i+1), pal.text, 1100, true, "ctr", pptxBodyFont))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("stepText%d", i), x+80000, y+420000, boxW-160000, 540000, step, pal.text, 1180, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderCardsDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
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
	cardW := (shell.BodyW - 180000*(cols-1)) / cols
	cardH := (shell.BodyH - 180000*(rows-1)) / rows
	startY := shell.BodyY
	for i, card := range cards {
		col := i % cols
		row := i / cols
		x := shell.BodyX + col*(cardW+180000)
		y := startY + row*(cardH+180000)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("card%d", i), x, y, cardW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		renderCardIconBadge(g, &sb, fmt.Sprintf("cardIcon%d", i), x+70000, y+70000, 320000, pal, inferCardIcon(card, i), variant)
		sb.WriteString(spTextLeft(g, fmt.Sprintf("cardTitle%d", i), x+70000, y+460000, cardW-140000, 220000, card.Title, pal.text, 1300, true, "t", pptxHeadingFont))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("cardDesc%d", i), x+70000, y+740000, cardW-140000, cardH-820000, firstNonEmpty(card.Desc, card.Title), pal.muted, 1020, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderChartDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
	categories := chartCategoriesOrFallback(s)
	series := effectiveChartSeries(s)
	if len(series) == 0 {
		series = []pptxChartSeries{{Name: "Series", Values: []float64{28, 44, 36, 58}}}
	}
	chartType := strings.ToLower(strings.TrimSpace(firstNonEmpty(s.ChartType, "column")))
	chartX, chartY := shell.BodyX, shell.BodyY
	chartW, chartH := shell.BodyW, shell.BodyH
	panelW := 0
	if chartType != "pie" && chartType != "doughnut" && design.LayoutStyle == "split" {
		panelW = 1760000
		chartW -= panelW + 180000
	}
	sb.WriteString(spRoundRect(g, "chartPanel", chartX, chartY, chartW, chartH, slidePanelFill(pal, design, 0), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
	if chartType == "pie" || chartType == "doughnut" {
		total := sumFloats(normalizedChartValues(series[0].Values, len(categories)))
		if total <= 0 {
			total = 1
		}
		for i, category := range categories {
			rowY := chartY + 240000 + i*620000
			pct := int((normalizedChartValues(series[0].Values, len(categories))[i] / total) * 100)
			sb.WriteString(spTextLeft(g, fmt.Sprintf("shareLabel%d", i), chartX+180000, rowY, 1600000, 180000, category, pal.text, 1200, true, "t", pptxBodyFont))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("shareTrack%d", i), chartX+180000, rowY+240000, chartW-760000, 50000, mixHex(pal.bg, pal.border, 0.18), "", 0))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("shareFill%d", i), chartX+180000, rowY+240000, (chartW-760000)*pct/100, 50000, pal.accent, "", 0))
			sb.WriteString(spText(g, fmt.Sprintf("sharePct%d", i), chartX+chartW-420000, rowY+140000, 240000, 180000, fmt.Sprintf("%d%%", pct), pal.muted, 980, false, "ctr", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}
	values := normalizedChartValues(series[0].Values, len(categories))
	second := []float64{}
	if len(series) > 1 {
		second = normalizedChartValues(series[1].Values, len(categories))
	}
	maxValue := maxFloat(values)
	if len(second) > 0 {
		maxValue = maxFloat(append(values[:0:0], append(values, second...)...))
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	innerX, innerY := chartX+180000, chartY+220000
	innerW, innerH := chartW-320000, chartH-540000
	for i := 0; i < 4; i++ {
		gridY := innerY + i*(innerH/4)
		sb.WriteString(spRect(g, fmt.Sprintf("chartGrid%d", i), innerX, gridY, innerW, 6350, mixHex(pal.bg, pal.border, 0.24)))
	}
	barGap := 120000
	barW := (innerW - barGap*(len(categories)-1)) / len(categories)
	for i, category := range categories {
		x := innerX + i*(barW+barGap)
		h := int((values[i] / maxValue) * float64(innerH-220000))
		if h < 80000 {
			h = 80000
		}
		sb.WriteString(spRoundRect(g, fmt.Sprintf("bar%d", i), x, innerY+innerH-h, barW, h, pal.accent, "", 0))
		if len(second) > i {
			h2 := int((second[i] / maxValue) * float64(innerH-220000))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("barAlt%d", i), x+barW/2, innerY+innerH-h2, barW/3, h2, pal.accent2, "", 0))
		}
		sb.WriteString(spText(g, fmt.Sprintf("barLabel%d", i), x-30000, innerY+innerH+50000, barW+60000, 160000, category, pal.muted, 880, false, "ctr", pptxBodyFont))
	}
	if panelW > 0 {
		panelX := chartX + chartW + 180000
		sb.WriteString(spRoundRect(g, "chartInsightPanel", panelX, chartY, panelW, chartH, slidePanelFill(pal, design, 1), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spTextLeft(g, "chartSeriesName", panelX+90000, chartY+120000, panelW-180000, 180000, firstNonEmpty(series[0].Name, "Signal"), pal.accent2, 1100, true, "t", pptxBodyFont))
		if strings.TrimSpace(s.YLabel) != "" {
			sb.WriteString(spTextLeft(g, "chartYLabel", panelX+90000, chartY+340000, panelW-180000, 180000, s.YLabel, pal.muted, 900, true, "t", pptxBodyFont))
		}
		sb.WriteString(spTextLeft(g, "chartInsight", panelX+90000, chartY+660000, panelW-180000, chartH-760000, chartInsight(categories, values), pal.text, 980, false, "t", pptxBodyFont))
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
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
	if design.LayoutStyle == "rail" {
		x := shell.BodyX + 220000
		sb.WriteString(spRect(g, "timelineVertical", x, shell.BodyY, 22000, shell.BodyH, deckBorder(pal)))
		for i, item := range items {
			y := shell.BodyY + i*820000
			sb.WriteString(spEllipse(g, fmt.Sprintf("milestone%d", i), x-110000, y+150000, 220000, 220000, pal.accent, 100, "", 0, 0))
			sb.WriteString(spRoundRect(g, fmt.Sprintf("milestoneCard%d", i), shell.BodyX+420000, y, shell.BodyW-420000, 540000, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneDate%d", i), shell.BodyX+560000, y+90000, 1100000, 180000, item.Date, pal.accent2, 980, true, "t", pptxBodyFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneTitle%d", i), shell.BodyX+560000, y+240000, 2200000, 180000, item.Title, pal.text, 1220, true, "t", pptxHeadingFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("milestoneDesc%d", i), shell.BodyX+2800000, y+230000, shell.BodyW-3000000, 200000, item.Desc, pal.muted, 960, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}
	lineY := shell.BodyY + shell.BodyH/2
	sb.WriteString(spRect(g, "timelineLine", shell.BodyX+200000, lineY, shell.BodyW-400000, 22000, deckBorder(pal)))
	nodeGap := (shell.BodyW - 400000) / len(items)
	for i, item := range items {
		cx := shell.BodyX + 200000 + i*nodeGap + nodeGap/2
		sb.WriteString(spEllipse(g, fmt.Sprintf("timelineNode%d", i), cx-180000, lineY-170000, 360000, 360000, pal.accent, 100, "", 0, 0))
		cardY := lineY - 1100000
		if i%2 == 1 {
			cardY = lineY + 240000
		}
		sb.WriteString(spRoundRect(g, fmt.Sprintf("timelineCard%d", i), cx-650000, cardY, 1300000, 620000, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spText(g, fmt.Sprintf("timelineDate%d", i), cx-600000, cardY+60000, 1200000, 140000, item.Date, pal.accent2, 980, true, "ctr", pptxBodyFont))
		sb.WriteString(spText(g, fmt.Sprintf("timelineTitle%d", i), cx-600000, cardY+220000, 1200000, 160000, item.Title, pal.text, 1120, true, "ctr", pptxHeadingFont))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("timelineDesc%d", i), cx-590000, cardY+380000, 1180000, 160000, item.Desc, pal.muted, 860, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderCompareDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	left, right := effectiveCompareColumns(s)
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
	colW := (shell.BodyW - 180000) / 2
	colH := shell.BodyH
	leftX, rightX := shell.BodyX, shell.BodyX+colW+180000
	leftFill := slidePanelFill(pal, design, 0)
	rightFill := slidePanelFill(pal, design, 1)
	sb.WriteString(spRoundRect(g, "leftCol", leftX, shell.BodyY, colW, colH, leftFill, slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
	sb.WriteString(spRoundRect(g, "rightCol", rightX, shell.BodyY, colW, colH, rightFill, slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
	sb.WriteString(spTextLeft(g, "leftTitle", leftX+80000, shell.BodyY+100000, colW-160000, 180000, firstNonEmpty(left.Heading, "Before"), pal.text, 1260, true, "t", pptxHeadingFont))
	sb.WriteString(spTextLeft(g, "rightTitle", rightX+80000, shell.BodyY+100000, colW-160000, 180000, firstNonEmpty(right.Heading, "After"), pal.text, 1260, true, "t", pptxHeadingFont))
	renderComparePoints(g, &sb, left.Points, leftX, shell.BodyY+360000, colW, pal, false)
	renderComparePoints(g, &sb, right.Points, rightX, shell.BodyY+360000, colW, pal, true)
	return wrapSlide(sb.String())
}

func renderTableDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	shell := renderSlideShell(g, &sb, pal, deck, s, slideIndex)
	design := resolveSlideDesign(deck, s)
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

	tableX, tableY := shell.BodyX, shell.BodyY
	tableW := shell.BodyW
	sidePanelW := 0
	if design.LayoutStyle == "split" || variant == 2 {
		sidePanelW = 1700000
		tableW -= sidePanelW + 180000
	}
	colW := tableW / cols
	rowH := 480000
	for i, header := range table.Headers {
		x := tableX + i*colW
		sb.WriteString(spRoundRect(g, fmt.Sprintf("hdr%d", i), x, tableY, colW, rowH, deckAccentWash(pal, 0.16), slidePanelBorder(pal, design), 6))
		sb.WriteString(spText(g, fmt.Sprintf("hdrText%d", i), x, tableY, colW, rowH, header, pal.accent, 1100, true, "ctr", pptxBodyFont))
	}
	for r, row := range table.Rows {
		fill := slidePanelFill(pal, design, r)
		for c, value := range row {
			x := tableX + c*colW
			y := tableY + rowH + r*rowH
			sb.WriteString(spRoundRect(g, fmt.Sprintf("cell%d_%d", r, c), x, y, colW, rowH, fill, slidePanelBorder(pal, design), 4))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("cellText%d_%d", r, c), x+40000, y+80000, colW-80000, rowH-120000, value, pal.text, 960, false, "t", pptxBodyFont))
		}
	}

	if sidePanelW > 0 {
		panelX := tableX + tableW + 180000
		sb.WriteString(spRoundRect(g, "tablePanel", panelX, tableY, sidePanelW, shell.BodyH, slidePanelFill(pal, design, 1), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spTextLeft(g, "panelTitle", panelX+80000, tableY+120000, sidePanelW-160000, 220000, "Decision Signal", pal.accent2, 1180, true, "t", pptxBodyFont))
		sb.WriteString(spTextLeft(g, "panelBody", panelX+80000, tableY+420000, sidePanelW-160000, shell.BodyH-520000, summarizeTable(table), pal.text, 980, false, "t", pptxBodyFont))
	}

	return wrapSlide(sb.String())
}

func renderSectionSlide(s pptxSlide, pal pptxPalette, variant int, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	heading := slideHeadingOrFallback(s, "Section")
	sb.WriteString(spRoundRect(g, "sectionCard", 1080000, 1600000, 6984000, 3200000, pal.card, deckBorder(pal), 18))
	sb.WriteString(spRoundRect(g, "sectionBadgeBg", 1360000, 1900000, 1600000, 240000, deckAccentWash(pal, 0.14), "", 0))
	sb.WriteString(spText(g, "sectionBadge", 1360000, 1900000, 1600000, 240000, fmt.Sprintf("Section %d", slideIndex+1), pal.accent, 980, true, "ctr", pptxBodyFont))
	sb.WriteString(spTextLeft(g, "sectionHeading", 1360000, 2480000, 6000000, 820000, heading, pal.text, 3200, true, "t", pptxHeadingFont))
	if len(s.Points) > 0 {
		sb.WriteString(spTextLeft(g, "sectionBody", 1360000, 3480000, 5000000, 600000, s.Points[0], pal.muted, 1300, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderBlankDeckSlide(s pptxSlide, pal pptxPalette, variant int, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	body := slideHeadingOrFallback(s, "Section Break")
	if len(s.Points) > 0 && strings.TrimSpace(s.Points[0]) != "" {
		body = s.Points[0]
	}
	sb.WriteString(spRoundRect(g, "blankCard", 1200000, 1900000, 6744000, 2300000, pal.card, deckBorder(pal), 18))
	sb.WriteString(spText(g, "blankMark", 1200000, 2200000, 6744000, 260000, "TRANSITION", pal.accent2, 1000, true, "ctr", pptxBodyFont))
	sb.WriteString(spText(g, "blankBody", 1500000, 2820000, 6144000, 760000, body, pal.text, 2200, true, "ctr", pptxHeadingFont))
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
		py := y + i*520000
		accent := deckBorder(pal)
		if positive {
			accent = pal.accent
		}
		sb.WriteString(spEllipse(g, fmt.Sprintf("cmpDot%d", i), x+80000, py+70000, 160000, 160000, accent, 100, "", 0, 0))
		sb.WriteString(spTextLeft(g, fmt.Sprintf("cmpText%d", i), x+300000, py, width-380000, 260000, point, pal.text, 1020, false, "t", pptxBodyFont))
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
	return fmt.Sprintf("Slide %d of %d", slideIndex+2, deck.SlideCount+1)
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
