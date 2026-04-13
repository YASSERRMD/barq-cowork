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
	pptxHeadingFont = "Avenir Next"
	pptxBodyFont    = "Avenir Next"
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
		return renderSectionSlide(s, pal, variant, deck, slideIndex)
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
		return renderBlankDeckSlide(s, pal, variant, deck, slideIndex)
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
	if deckRenderFamily(deck) == "proposal" {
		if large {
			return 6
		}
		return 3
	}
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

func deckRenderFamily(deck pptxDeckContext) string {
	text := strings.ToLower(strings.Join([]string{
		deck.Title,
		deck.Subtitle,
		deck.ThemeName,
		deck.DeckPlan.Subject,
		deck.DeckPlan.Audience,
		deck.DeckPlan.NarrativeArc,
		deck.DeckPlan.VisualDirection,
		deck.DeckPlan.ColorStory,
		deck.DeckPlan.DominantNeed,
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

func proposalCanvasFill(pal pptxPalette) string {
	return mixHex(pal.bg, "F0F4F8", 0.76)
}

func proposalHeaderFill(pal pptxPalette) string {
	return mixHex("0D1B2A", pal.text, 0.32)
}

func proposalFooterFill(pal pptxPalette) string {
	return mixHex(proposalHeaderFill(pal), pal.card, 0.12)
}

func proposalMutedOnDark(pal pptxPalette) string {
	return mixHex(proposalHeaderFill(pal), "FFFFFF", 0.66)
}

func proposalMutedOnLight(pal pptxPalette) string {
	return mixHex(proposalCanvasFill(pal), pal.text, 0.54)
}

func proposalAccentColor(pal pptxPalette, index int) string {
	switch index % 4 {
	case 1:
		return pal.accent2
	case 2:
		return mixHex(pal.text, pal.border, 0.34)
	case 3:
		return mixHex(pal.accent, pal.accent2, 0.5)
	default:
		return pal.accent
	}
}

func proposalRowFill(pal pptxPalette, index int) string {
	if index%2 == 1 {
		return mixHex(proposalCanvasFill(pal), pal.card, 0.74)
	}
	return pal.card
}

func proposalRowBorder(pal pptxPalette) string {
	return mixHex(proposalCanvasFill(pal), pal.border, 0.84)
}

func renderProposalPanel(g *idg, sb *strings.Builder, name string, x, y, w, h int, fill, border string) {
	sb.WriteString(spRect(g, name, x, y, w, h, fill))
	if border == "" {
		return
	}
	const stroke = 4500
	sb.WriteString(spRect(g, name+"TopBorder", x, y, w, stroke, border))
	sb.WriteString(spRect(g, name+"BottomBorder", x, y+h-stroke, w, stroke, border))
	sb.WriteString(spRect(g, name+"LeftBorder", x, y, stroke, h, border))
	sb.WriteString(spRect(g, name+"RightBorder", x+w-stroke, y, stroke, h, border))
}

func renderProposalTopAccent(g *idg, sb *strings.Builder, name string, x, y, w int, fill string) {
	sb.WriteString(spRect(g, name, x, y, w, 32000, fill))
}

func renderProposalSideAccent(g *idg, sb *strings.Builder, name string, x, y, h int, fill string) {
	sb.WriteString(spRect(g, name, x, y, 32000, h, fill))
}

func renderProposalMetricTile(g *idg, sb *strings.Builder, name string, x, y, w, h int, value, label, detail, accent string, pal pptxPalette, dark bool) {
	fill := pal.card
	border := proposalRowBorder(pal)
	labelColor := pal.text
	bodyColor := proposalMutedOnLight(pal)
	if dark {
		fill = proposalHeaderFill(pal)
		border = mixHex(fill, pal.card, 0.18)
		labelColor = proposalMutedOnDark(pal)
		bodyColor = proposalMutedOnDark(pal)
	}
	renderProposalPanel(g, sb, name+"Panel", x, y, w, h, fill, border)
	renderProposalTopAccent(g, sb, name+"Accent", x, y, w, accent)
	sb.WriteString(spTextLeft(g, name+"Label", x+90000, y+90000, w-180000, 150000, strings.ToUpper(label), labelColor, 820, true, "t", pptxBodyFont))
	sb.WriteString(spTextLeft(g, name+"Value", x+90000, y+270000, w-180000, 260000, value, accent, 1820, true, "t", pptxHeadingFont))
	if detail != "" {
		sb.WriteString(spTextLeft(g, name+"Detail", x+90000, y+500000, w-180000, h-570000, detail, bodyColor, 820, false, "t", pptxBodyFont))
	}
}

func proposalSlideTitle(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= 28 {
		return strings.ToUpper(text)
	}
	return text
}

func proposalSlideIconToken(s pptxSlide) string {
	layout := effectivePPTXLayout(s)
	switch layout {
	case "stats", "chart":
		return "chart"
	case "steps", "timeline":
		return "strategy"
	case "compare":
		return "integration"
	case "table":
		return "shield"
	case "cards":
		if len(s.Cards) > 0 {
			return inferCardIcon(s.Cards[0], 0)
		}
		return "spark"
	default:
		return firstNonEmpty(normalizeIconToken(s.Heading), inferIconFromText(s.Heading+" "+layout), "spark")
	}
}

func renderProposalHeaderIcon(g *idg, sb *strings.Builder, name string, x, y, size int, pal pptxPalette, token string) {
	fill := mixHex(proposalHeaderFill(pal), pal.accent, 0.22)
	renderProposalPanel(g, sb, name+"Bg", x, y, size, size, fill, mixHex(fill, pal.card, 0.18))
	iconPal := pal
	iconPal.text = "FFFFFF"
	iconPal.accent2 = "FFFFFF"
	renderPPTXIconGlyph(g, sb, name+"Glyph", x+36000, y+36000, size-72000, iconPal, token)
}

func renderProposalCover(g *idg, sb *strings.Builder, deck pptxDeckContext, pal pptxPalette) {
	bg := proposalHeaderFill(pal)
	muted := proposalMutedOnDark(pal)
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, bg))
	sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 42000, pal.accent))

	panelX := 720000
	panelW := 6120000
	sb.WriteString(spTextLeft(g, "coverKicker", panelX, 780000, 2800000, 220000, strings.ToUpper(coverKicker(deck)), pal.accent, 980, true, "t", pptxBodyFont))
	sb.WriteString(spRect(g, "coverRule", panelX, 1200000, 1840000, 28000, pal.accent2))

	titleSize, titleHeight := coverTitleMetrics(deck.Title, 3880, 17)
	sb.WriteString(spTextLeft(g, "coverTitle", panelX, 1440000, panelW, titleHeight, firstNonEmpty(deck.Title, "Presentation"), "FFFFFF", titleSize, true, "t", pptxHeadingFont))

	subtitle := coverLead(deck)
	subtitleSize, subtitleHeight := coverSubtitleMetrics(subtitle, 1540, 38)
	if subtitle != "" {
		sb.WriteString(spTextLeft(g, "coverSubtitle", panelX, 1440000+titleHeight+220000, panelW, subtitleHeight, subtitle, muted, subtitleSize, false, "t", pptxBodyFont))
	}
	if support := coverSupportLine(deck); support != "" {
		sb.WriteString(spTextLeft(g, "coverSupport", panelX, 1440000+titleHeight+subtitleHeight+420000, panelW, 320000, support, muted, 1120, false, "t", pptxBodyFont))
	}

	sb.WriteString(spRect(g, "footerBar", 0, 6400000, 9144000, 360000, proposalFooterFill(pal)))
	sb.WriteString(spTextLeft(g, "footerText", panelX, 6480000, 3200000, 180000, coverSubjectLine(deck), muted, 920, false, "t", pptxBodyFont))
}

func renderSlideShell(g *idg, sb *strings.Builder, pal pptxPalette, deck pptxDeckContext, s pptxSlide, slideIndex int) pptxSlideShell {
	design := resolveSlideDesign(deck, s)
	deckDesign := resolveDeckDesign(deck)
	if deckRenderFamily(deck) == "proposal" {
		bg := proposalCanvasFill(pal)
		headerBG := proposalHeaderFill(pal)
		sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, bg))
		sb.WriteString(spRect(g, "headerBar", 0, 0, 9144000, 620000, headerBG))
		renderProposalHeaderIcon(g, sb, "headerIcon", 420000, 130000, 280000, pal, proposalSlideIconToken(s))
		sb.WriteString(spTextLeft(g, "headerTitle", 780000, 164000, 5600000, 220000, proposalSlideTitle(slideHeadingOrFallback(s, "Untitled Slide")), "FFFFFF", 1440, true, "ctr", pptxHeadingFont))
		renderProposalPanel(g, sb, "headerMeta", 7420000, 134000, 1280000, 280000, mixHex(headerBG, pal.accent2, 0.52), mixHex(headerBG, pal.card, 0.16))
		sb.WriteString(spText(g, "headerMetaText", 7420000, 134000, 1280000, 280000, slideLabel(deck, slideIndex, effectivePPTXLayout(s)), "FFFFFF", 900, true, "ctr", pptxBodyFont))
		return pptxSlideShell{
			FrameX: 420000,
			FrameY: 820000,
			FrameW: 8304000,
			FrameH: 5440000,
			BodyX:  420000,
			BodyY:  820000,
			BodyW:  8304000,
			BodyH:  5440000,
		}
	}

	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))

	mode := "editorial"
	switch {
	case design.LayoutStyle == "stage" || deckDesign.Composition == "float" || deckDesign.Composition == "band":
		mode = "float"
	case design.LayoutStyle == "grid" || design.LayoutStyle == "matrix" || deckDesign.Composition == "gallery" || deckDesign.Composition == "asym":
		mode = "gallery"
	case design.LayoutStyle == "split" || design.LayoutStyle == "spotlight" || deckDesign.Composition == "split" || deckDesign.Composition == "frame":
		mode = "split"
	}

	frameX, frameY := 520000, 520000
	frameW, frameH := 7420000, 5820000
	frameFill := pal.card
	if design.PanelStyle == "tint" || design.PanelStyle == "glass" {
		frameFill = deckSurface(pal)
	}
	switch mode {
	case "split":
		frameX, frameY = 520000, 520000
		frameW, frameH = 5500000, 5820000
		sb.WriteString(spRoundRect(g, "frameUnderlay", frameX+180000, frameY+180000, frameW, frameH, deckAccentWash(pal, 0.08), "", 0))
		sb.WriteString(spRoundRect(g, "frame", frameX, frameY, frameW, frameH, frameFill, deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spRoundRect(g, "figurePanelTall", 6380000, 760000, 1920000, 2220000, deckAccentWash(pal, 0.12), deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spRoundRect(g, "figurePanelWide", 6220000, 3260000, 2140000, 1680000, deckAccent2Wash(pal, 0.12), deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spEllipse(g, "figureOrb", 7340000, 5060000, 980000, 980000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
	case "gallery":
		frameX, frameY = 560000, 560000
		frameW, frameH = 7080000, 5740000
		sb.WriteString(spRoundRect(g, "galleryBackplate", frameX+160000, frameY+160000, frameW, frameH, deckAccentWash(pal, 0.08), "", 0))
		sb.WriteString(spRoundRect(g, "frame", frameX, frameY, frameW, frameH, frameFill, deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spRoundRect(g, "galleryTileA", 7040000, 760000, 1260000, 1120000, deckSurface(pal), deckBorder(pal), slideCornerRadius(deck, false)))
		sb.WriteString(spRoundRect(g, "galleryTileB", 6800000, 2060000, 1500000, 980000, deckAccentWash(pal, 0.16), deckBorder(pal), slideCornerRadius(deck, false)))
		sb.WriteString(spRoundRect(g, "galleryTileC", 7180000, 3320000, 1120000, 1480000, deckAccent2Wash(pal, 0.14), deckBorder(pal), slideCornerRadius(deck, false)))
		sb.WriteString(spEllipse(g, "galleryOrb", 7480000, 5300000, 720000, 720000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
	case "float":
		frameX, frameY = 760000, 760000
		frameW, frameH = 7060000, 5200000
		sb.WriteString(spEllipse(g, "floatOrbA", 6800000, 240000, 1600000, 1600000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
		sb.WriteString(spEllipse(g, "floatOrbB", 1200000, 5340000, 1200000, 1200000, deckAccent2Wash(pal, 0.12), 100, "", 0, 0))
		sb.WriteString(spRoundRect(g, "frame", frameX, frameY, frameW, frameH, frameFill, deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spRoundRect(g, "floatTag", 6420000, 5660000, 1540000, 520000, deckSurface(pal), deckBorder(pal), slideCornerRadius(deck, false)))
	default:
		frameX, frameY = 600000, 520000
		frameW, frameH = 7300000, 5820000
		sb.WriteString(spEllipse(g, "editorialGlow", 6940000, 220000, 1380000, 1380000, deckAccentWash(pal, 0.1), 100, "", 0, 0))
		sb.WriteString(spRoundRect(g, "frame", frameX, frameY, frameW, frameH, frameFill, deckBorder(pal), slideCornerRadius(deck, true)))
		sb.WriteString(spRect(g, "editorialRule", frameX+240000, frameY+180000, 1220000, 28000, pal.accent))
	}

	switch design.AccentMode {
	case "band":
		sb.WriteString(spRect(g, "accentBand", frameX, frameY, frameW, 32000, pal.accent))
	case "block":
		sb.WriteString(spRoundRect(g, "accentBlock", frameX+frameW-640000, frameY+160000, 420000, 420000, deckAccentWash(pal, 0.14), "", 0))
	case "glow":
		sb.WriteString(spEllipse(g, "accentGlow", frameX+frameW-760000, frameY-180000, 920000, 920000, deckAccentWash(pal, 0.18), 100, "", 0, 0))
	case "chip":
		sb.WriteString(spRoundRect(g, "accentChipMain", frameX+frameW-1060000, frameY+180000, 680000, 120000, deckAccentWash(pal, 0.22), "", 0))
		sb.WriteString(spRoundRect(g, "accentChipDot", frameX+frameW-300000, frameY+180000, 110000, 120000, deckAccent2Wash(pal, 0.22), "", 0))
	case "ribbon":
		sb.WriteString(spRect(g, "accentRibbon", frameX+220000, frameY+170000, 1100000, 28000, pal.accent))
		sb.WriteString(spRect(g, "accentRibbonTail", frameX+1380000, frameY+170000, 220000, 28000, deckAccent2Wash(pal, 0.72)))
	case "marker":
		sb.WriteString(spRect(g, "accentMarker", frameX+220000, frameY+820000, 24000, 380000, pal.accent))
	default:
		sb.WriteString(spRect(g, "accentRail", frameX, frameY, 32000, frameH, pal.accent))
	}

	pillW := 1240000
	if design.AccentMode == "chip" || design.AccentMode == "marker" || mode == "float" {
		pillW = 1440000
	}
	pillY := frameY + 180000
	pillFill := deckAccentWash(pal, 0.14)
	if design.AccentMode == "band" {
		pillFill = deckAccent2Wash(pal, 0.16)
	}
	sb.WriteString(spRoundRect(g, "metaPill", frameX+260000, pillY, pillW, 250000, pillFill, "", 0))
	sb.WriteString(spText(g, "metaText", frameX+260000, pillY, pillW, 250000, slideLabel(deck, slideIndex, effectivePPTXLayout(s)), pal.accent, 980, true, "ctr", pptxBodyFont))

	headingX := frameX + 260000
	headingY := pillY + 360000
	headingW := frameW - 700000
	if mode == "gallery" {
		headingW = frameW - 520000
	}
	headingH := 560000
	if design.Density == "dense" {
		headingH = 660000
	} else if design.Density == "airy" {
		headingH = 500000
	}
	sb.WriteString(spTextLeft(g, "heading", headingX, headingY, headingW, headingH, slideHeadingOrFallback(s, "Untitled Slide"), pal.text, 2500, true, "t", pptxHeadingFont))

	bodyY := headingY + headingH + 170000
	bodyH := frameH - (bodyY - frameY) - 220000
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
	if deckRenderFamily(deck) == "proposal" {
		renderProposalCover(g, &sb, deck, pal)
	} else {
		renderPreviewAlignedCover(g, &sb, deck, pal)
	}

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

	if deckRenderFamily(deck) == "proposal" {
		bodyY := shell.BodyY
		stats := effectiveStats(s)
		if len(stats) > 0 {
			if len(stats) > 4 {
				stats = stats[:4]
			}
			gap := 140000
			cardW := (shell.BodyW - gap*(len(stats)-1)) / len(stats)
			for i, stat := range stats {
				renderProposalMetricTile(g, &sb, fmt.Sprintf("detailMetric%d", i), shell.BodyX+i*(cardW+gap), bodyY, cardW, 640000, stat.Value, stat.Label, stat.Desc, proposalAccentColor(pal, i), pal, false)
			}
			bodyY += 820000
		}
		lead := proposalLeadText(s, points)
		if lead != "" {
			sb.WriteString(spTextLeft(g, "detailLead", shell.BodyX, bodyY, shell.BodyW, 320000, lead, pal.muted, 920, false, "t", pptxBodyFont))
			bodyY += 420000
		}
		sb.WriteString(spTextLeft(g, "detailLabel", shell.BodyX, bodyY, 2600000, 180000, proposalSectionLabel(s), pal.text, 960, true, "t", pptxBodyFont))
		bodyY += 240000
		cols := 1
		if len(points) >= 4 {
			cols = 2
		}
		colGap := 260000
		colW := shell.BodyW
		if cols == 2 {
			colW = (shell.BodyW - colGap) / 2
		}
		rowH := 260000
		for i, pt := range points {
			col := 0
			row := i
			if cols == 2 {
				col = i % 2
				row = i / 2
			}
			title, desc := splitCardText(pt)
			x := shell.BodyX + col*(colW+colGap)
			y := bodyY + row*rowH
			accent := proposalAccentColor(pal, i)
			sb.WriteString(spEllipse(g, fmt.Sprintf("pointDot%d", i), x, y+45000, 90000, 90000, accent, 100, "", 0, 0))
			if desc == "" {
				sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), x+140000, y, colW-140000, 180000, title, pal.text, 900, false, "t", pptxBodyFont))
			} else {
				sb.WriteString(spTextLeft(g, fmt.Sprintf("pointTitle%d", i), x+140000, y, colW-140000, 150000, title, pal.text, 900, true, "t", pptxBodyFont))
				sb.WriteString(spTextLeft(g, fmt.Sprintf("pointDesc%d", i), x+140000, y+130000, colW-140000, 150000, desc, proposalMutedOnLight(pal), 820, false, "t", pptxBodyFont))
			}
		}
		return wrapSlide(sb.String())
	}

	if design.LayoutStyle == "split" && len(points) >= 4 {
		colW := (shell.BodyW - 180000) / 2
		gap := 180000
		cardH := 760000
		for i, pt := range points {
			title, desc := splitCardText(pt)
			col := i % 2
			row := i / 2
			x := shell.BodyX + col*(colW+gap)
			y := shell.BodyY + row*(cardH+140000)
			sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), x, y, colW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
			sb.WriteString(spRect(g, fmt.Sprintf("pointRule%d", i), x+70000, y+110000, 340000, 22000, pal.accent))
			sb.WriteString(spEllipse(g, fmt.Sprintf("pointDot%d", i), x+70000, y+190000, 90000, 90000, pal.accent2, 100, "", 0, 0))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointTitle%d", i), x+200000, y+160000, colW-300000, 220000, title, pal.text, 1240, true, "t", pptxHeadingFont))
			if desc != "" {
				sb.WriteString(spTextLeft(g, fmt.Sprintf("pointDesc%d", i), x+200000, y+390000, colW-300000, cardH-470000, desc, pal.muted, 980, false, "t", pptxBodyFont))
			}
		}
		return wrapSlide(sb.String())
	}

	cardW := shell.BodyW
	cardH := 520000
	for i, pt := range points {
		title, desc := splitCardText(pt)
		y := shell.BodyY + i*(cardH+100000)
		sb.WriteString(spRoundRect(g, fmt.Sprintf("pointCard%d", i), shell.BodyX, y, cardW, cardH, slidePanelFill(pal, design, i), slidePanelBorder(pal, design), slideCornerRadius(deck, false)))
		sb.WriteString(spEllipse(g, fmt.Sprintf("pointDot%d", i), shell.BodyX+76000, y+170000, 90000, 90000, pal.accent, 100, "", 0, 0))
		if desc != "" {
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointTitle%d", i), shell.BodyX+220000, y+90000, cardW-320000, 180000, title, pal.text, 1180, true, "t", pptxHeadingFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointDesc%d", i), shell.BodyX+220000, y+260000, cardW-320000, cardH-300000, desc, pal.muted, 960, false, "t", pptxBodyFont))
		} else {
			sb.WriteString(spTextLeft(g, fmt.Sprintf("pointText%d", i), shell.BodyX+220000, y+120000, cardW-320000, cardH-180000, title, pal.text, 1220, false, "t", pptxBodyFont))
		}
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

	if deckRenderFamily(deck) == "proposal" {
		gapX := 170000
		heroW := (shell.BodyW - gapX) / 2
		heroH := minInt(1640000, shell.BodyH)
		heroY := shell.BodyY
		for i := 0; i < minInt(2, len(stats)); i++ {
			stat := stats[i]
			x := shell.BodyX + i*(heroW+gapX)
			fill := proposalHeaderFill(pal)
			if i == 1 {
				fill = mixHex(proposalHeaderFill(pal), pal.card, 0.18)
			}
			accent := proposalAccentColor(pal, i)
			renderProposalPanel(g, &sb, fmt.Sprintf("heroStatCard%d", i), x, heroY, heroW, heroH, fill, mixHex(fill, pal.card, 0.18))
			renderProposalTopAccent(g, &sb, fmt.Sprintf("heroStatAccent%d", i), x, heroY, heroW, accent)
			sb.WriteString(spTextLeft(g, fmt.Sprintf("heroStatLabel%d", i), x+110000, heroY+120000, heroW-220000, 170000, strings.ToUpper(stat.Label), proposalMutedOnDark(pal), 860, true, "t", pptxBodyFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("heroStatValue%d", i), x+110000, heroY+360000, heroW-220000, 420000, stat.Value, accent, 2720, true, "t", pptxHeadingFont))
			if stat.Desc != "" {
				sb.WriteString(spTextLeft(g, fmt.Sprintf("heroStatDesc%d", i), x+110000, heroY+840000, heroW-220000, heroH-930000, stat.Desc, proposalMutedOnDark(pal), 960, false, "t", pptxBodyFont))
			}
		}

		if len(stats) > 2 {
			lowerY := heroY + heroH + 190000
			lowerH := minInt(980000, shell.BodyY+shell.BodyH-lowerY-420000)
			miniCount := len(stats) - 2
			miniGap := 150000
			miniW := shell.BodyW
			if miniCount > 1 {
				miniW = (shell.BodyW - miniGap*(miniCount-1)) / miniCount
			}
			for i := 2; i < len(stats); i++ {
				stat := stats[i]
				slot := i - 2
				x := shell.BodyX + slot*(miniW+miniGap)
				accent := proposalAccentColor(pal, i)
				renderProposalPanel(g, &sb, fmt.Sprintf("miniStatCard%d", i), x, lowerY, miniW, lowerH, pal.card, proposalRowBorder(pal))
				renderProposalTopAccent(g, &sb, fmt.Sprintf("miniStatAccent%d", i), x, lowerY, miniW, accent)
				sb.WriteString(spTextLeft(g, fmt.Sprintf("miniStatLabel%d", i), x+90000, lowerY+100000, miniW-180000, 140000, strings.ToUpper(stat.Label), pal.text, 820, true, "t", pptxBodyFont))
				sb.WriteString(spTextLeft(g, fmt.Sprintf("miniStatValue%d", i), x+90000, lowerY+300000, miniW-180000, 280000, stat.Value, accent, 1880, true, "t", pptxHeadingFont))
				if stat.Desc != "" {
					sb.WriteString(spTextLeft(g, fmt.Sprintf("miniStatDesc%d", i), x+90000, lowerY+570000, miniW-180000, lowerH-640000, stat.Desc, proposalMutedOnLight(pal), 840, false, "t", pptxBodyFont))
				}
			}

			bandY := lowerY + lowerH + 160000
			bandH := minInt(360000, shell.BodyY+shell.BodyH-bandY)
			if bandH > 160000 {
				sb.WriteString(spRect(g, "statsBand", shell.BodyX, bandY, shell.BodyW, bandH, mixHex(proposalCanvasFill(pal), pal.border, 0.18)))
				sb.WriteString(spTextLeft(g, "statsBandText", shell.BodyX+90000, bandY+70000, shell.BodyW-180000, bandH-120000, proposalStatsTakeaway(stats), pal.text, 840, true, "ctr", pptxBodyFont))
			}
		}
		return wrapSlide(sb.String())
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

	if deckRenderFamily(deck) == "proposal" {
		rowH := 460000
		for i, step := range steps {
			title, desc := splitCardText(step)
			y := shell.BodyY + i*(rowH+84000)
			accent := proposalAccentColor(pal, i)
			renderProposalPanel(g, &sb, fmt.Sprintf("stepRow%d", i), shell.BodyX+320000, y, shell.BodyW-320000, rowH, proposalRowFill(pal, i), proposalRowBorder(pal))
			renderProposalSideAccent(g, &sb, fmt.Sprintf("stepAccent%d", i), shell.BodyX+320000, y, rowH, accent)
			sb.WriteString(spEllipse(g, fmt.Sprintf("stepBadge%d", i), shell.BodyX, y+70000, 220000, 220000, proposalHeaderFill(pal), 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("stepNum%d", i), shell.BodyX, y+70000, 220000, 220000, fmt.Sprintf("%d", i+1), "FFFFFF", 980, true, "ctr", pptxBodyFont))
			renderProposalHeaderIcon(g, &sb, fmt.Sprintf("stepIcon%d", i), shell.BodyX+240000, y+102000, 180000, pal, firstNonEmpty(inferIconFromText(title), "strategy"))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("stepTitle%d", i), shell.BodyX+620000, y+76000, shell.BodyW-1880000, 160000, title, pal.text, 980, true, "t", pptxBodyFont))
			if desc != "" {
				sb.WriteString(spTextLeft(g, fmt.Sprintf("stepDesc%d", i), shell.BodyX+620000, y+214000, shell.BodyW-1880000, 130000, desc, proposalMutedOnLight(pal), 820, false, "t", pptxBodyFont))
			}
			renderProposalPanel(g, &sb, fmt.Sprintf("stepTag%d", i), shell.BodyX+shell.BodyW-1080000, y+90000, 900000, 220000, mixHex(proposalCanvasFill(pal), accent, 0.14), "")
			sb.WriteString(spText(g, fmt.Sprintf("stepTagText%d", i), shell.BodyX+shell.BodyW-1080000, y+90000, 900000, 220000, fmt.Sprintf("STEP %02d", i+1), accent, 820, true, "ctr", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}

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

	if deckRenderFamily(deck) == "proposal" {
		cols := 2
		rows := (len(cards) + cols - 1) / cols
		cardW := (shell.BodyW - 180000) / cols
		cardH := (shell.BodyH - 120000*(rows-1)) / rows
		for i, card := range cards {
			col := i % cols
			row := i / cols
			x := shell.BodyX + col*(cardW+180000)
			y := shell.BodyY + row*(cardH+120000)
			accent := proposalAccentColor(pal, i)
			renderProposalPanel(g, &sb, fmt.Sprintf("card%d", i), x, y, cardW, cardH, proposalRowFill(pal, i), proposalRowBorder(pal))
			renderProposalTopAccent(g, &sb, fmt.Sprintf("cardAccent%d", i), x, y, cardW, accent)
			renderProposalHeaderIcon(g, &sb, fmt.Sprintf("cardIcon%d", i), x+80000, y+80000, 220000, pal, inferCardIcon(card, i))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("cardTitle%d", i), x+360000, y+94000, cardW-440000, 180000, card.Title, pal.text, 1020, true, "t", pptxBodyFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("cardDesc%d", i), x+90000, y+360000, cardW-180000, cardH-440000, firstNonEmpty(card.Desc, card.Title), proposalMutedOnLight(pal), 860, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
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

	if deckRenderFamily(deck) == "proposal" {
		mainW := shell.BodyW
		sideW := 0
		if chartType != "pie" && chartType != "doughnut" {
			sideW = 2140000
			mainW -= sideW + 160000
		}
		mainX, mainY := shell.BodyX, shell.BodyY
		mainH := shell.BodyH
		mainAccent := proposalAccentColor(pal, 0)
		renderProposalPanel(g, &sb, "chartProposalPanel", mainX, mainY, mainW, mainH, pal.card, proposalRowBorder(pal))
		renderProposalTopAccent(g, &sb, "chartProposalAccent", mainX, mainY, mainW, mainAccent)

		if chartType == "pie" || chartType == "doughnut" {
			values := normalizedChartValues(series[0].Values, len(categories))
			total := sumFloats(values)
			if total <= 0 {
				total = 1
			}
			for i, category := range categories {
				rowY := mainY + 220000 + i*640000
				pct := int((values[i] / total) * 100)
				rowAccent := proposalAccentColor(pal, i)
				sb.WriteString(spTextLeft(g, fmt.Sprintf("shareLabel%d", i), mainX+140000, rowY, 1700000, 180000, category, pal.text, 1040, true, "t", pptxBodyFont))
				sb.WriteString(spRoundRect(g, fmt.Sprintf("shareTrack%d", i), mainX+140000, rowY+240000, mainW-760000, 52000, mixHex(proposalCanvasFill(pal), pal.border, 0.24), "", 0))
				sb.WriteString(spRoundRect(g, fmt.Sprintf("shareFill%d", i), mainX+140000, rowY+240000, maxInt(140000, (mainW-760000)*pct/100), 52000, rowAccent, "", 0))
				sb.WriteString(spText(g, fmt.Sprintf("sharePct%d", i), mainX+mainW-460000, rowY+150000, 280000, 180000, fmt.Sprintf("%d%%", pct), pal.muted, 880, false, "ctr", pptxBodyFont))
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

		innerX, innerY := mainX+160000, mainY+200000
		innerW, innerH := mainW-320000, mainH-520000
		for i := 0; i < 4; i++ {
			gridY := innerY + i*(innerH/4)
			sb.WriteString(spRect(g, fmt.Sprintf("chartGrid%d", i), innerX, gridY, innerW, 6350, mixHex(proposalCanvasFill(pal), pal.border, 0.2)))
		}
		barGap := 120000
		barW := (innerW - barGap*(len(categories)-1)) / len(categories)
		for i, category := range categories {
			x := innerX + i*(barW+barGap)
			h := int((values[i] / maxValue) * float64(innerH-240000))
			if h < 80000 {
				h = 80000
			}
			fill := proposalAccentColor(pal, i)
			sb.WriteString(spRoundRect(g, fmt.Sprintf("bar%d", i), x, innerY+innerH-h, barW, h, fill, "", 0))
			if len(second) > i {
				h2 := int((second[i] / maxValue) * float64(innerH-240000))
				if h2 < 60000 {
					h2 = 60000
				}
				sb.WriteString(spRoundRect(g, fmt.Sprintf("barAlt%d", i), x+(barW*2)/3, innerY+innerH-h2, barW/3, h2, pal.accent2, "", 0))
			}
			sb.WriteString(spText(g, fmt.Sprintf("barLabel%d", i), x-40000, innerY+innerH+60000, barW+80000, 160000, category, pal.muted, 820, false, "ctr", pptxBodyFont))
		}

		if sideW > 0 {
			sideX := mainX + mainW + 160000
			sideAccent := proposalAccentColor(pal, 1)
			renderProposalPanel(g, &sb, "chartProposalSide", sideX, mainY, sideW, mainH, proposalRowFill(pal, 1), proposalRowBorder(pal))
			renderProposalSideAccent(g, &sb, "chartProposalSideAccent", sideX, mainY, mainH, sideAccent)
			renderProposalHeaderIcon(g, &sb, "chartSideIcon", sideX+90000, mainY+100000, 240000, pal, proposalSlideIconToken(s))
			sb.WriteString(spTextLeft(g, "chartSeriesName", sideX+380000, mainY+120000, sideW-460000, 180000, firstNonEmpty(series[0].Name, "Signal"), pal.text, 1100, true, "t", pptxHeadingFont))
			if strings.TrimSpace(s.YLabel) != "" {
				sb.WriteString(spTextLeft(g, "chartYLabel", sideX+380000, mainY+320000, sideW-460000, 160000, strings.ToUpper(s.YLabel), sideAccent, 840, true, "t", pptxBodyFont))
			}
			sb.WriteString(spTextLeft(g, "chartInsight", sideX+90000, mainY+620000, sideW-180000, mainH-720000, chartInsight(categories, values), proposalMutedOnLight(pal), 920, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}

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
	if deckRenderFamily(deck) == "proposal" {
		rowH := 560000
		for i, item := range items {
			y := shell.BodyY + i*(rowH+90000)
			accent := proposalAccentColor(pal, i)
			sb.WriteString(spEllipse(g, fmt.Sprintf("timelineBadge%d", i), shell.BodyX, y+90000, 220000, 220000, proposalHeaderFill(pal), 100, "", 0, 0))
			sb.WriteString(spText(g, fmt.Sprintf("timelineBadgeNum%d", i), shell.BodyX, y+90000, 220000, 220000, fmt.Sprintf("%d", i+1), "FFFFFF", 980, true, "ctr", pptxBodyFont))
			renderProposalHeaderIcon(g, &sb, fmt.Sprintf("timelineIcon%d", i), shell.BodyX+260000, y+112000, 180000, pal, firstNonEmpty(inferIconFromText(item.Title), proposalSlideIconToken(s)))
			renderProposalPanel(g, &sb, fmt.Sprintf("timelineRow%d", i), shell.BodyX+480000, y, shell.BodyW-480000, rowH, proposalRowFill(pal, i), proposalRowBorder(pal))
			renderProposalSideAccent(g, &sb, fmt.Sprintf("timelineAccent%d", i), shell.BodyX+480000, y, rowH, accent)
			sb.WriteString(spTextLeft(g, fmt.Sprintf("timelineTitle%d", i), shell.BodyX+700000, y+86000, 2480000, 160000, item.Title, pal.text, 980, true, "t", pptxBodyFont))
			sb.WriteString(spText(g, fmt.Sprintf("timelineDateText%d", i), shell.BodyX+3440000, y+120000, 1040000, 140000, item.Date, proposalMutedOnLight(pal), 820, true, "ctr", pptxBodyFont))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("timelineDesc%d", i), shell.BodyX+4420000, y+86000, shell.BodyW-4560000, 280000, firstNonEmpty(item.Desc, item.Title), proposalMutedOnLight(pal), 820, false, "t", pptxBodyFont))
		}
		return wrapSlide(sb.String())
	}
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
	if deckRenderFamily(deck) == "proposal" {
		colW := (shell.BodyW - 160000) / 2
		colH := shell.BodyH
		leftX, rightX := shell.BodyX, shell.BodyX+colW+160000
		columns := []struct {
			key    string
			col    pptxCompareColumn
			x      int
			accent string
		}{
			{key: "left", col: left, x: leftX, accent: proposalAccentColor(pal, 0)},
			{key: "right", col: right, x: rightX, accent: proposalAccentColor(pal, 1)},
		}
		for _, item := range columns {
			renderProposalPanel(g, &sb, item.key+"Col", item.x, shell.BodyY, colW, colH, proposalRowFill(pal, 0), proposalRowBorder(pal))
			renderProposalSideAccent(g, &sb, item.key+"Accent", item.x, shell.BodyY, colH, item.accent)
			renderProposalHeaderIcon(g, &sb, item.key+"Icon", item.x+90000, shell.BodyY+90000, 220000, pal, proposalSlideIconToken(s))
			sb.WriteString(spTextLeft(g, item.key+"Title", item.x+360000, shell.BodyY+110000, colW-440000, 180000, strings.ToUpper(firstNonEmpty(item.col.Heading, "Column")), item.accent, 900, true, "t", pptxBodyFont))
			for i, point := range safePoints(item.col.Points, 5) {
				rowY := shell.BodyY + 420000 + i*520000
				renderProposalPanel(g, &sb, fmt.Sprintf("%sPointRow%d", item.key, i), item.x+90000, rowY, colW-180000, 380000, mixHex(proposalCanvasFill(pal), pal.card, 0.58), proposalRowBorder(pal))
				sb.WriteString(spEllipse(g, fmt.Sprintf("%sPointDot%d", item.key, i), item.x+130000, rowY+110000, 160000, 160000, item.accent, 100, "", 0, 0))
				sb.WriteString(spTextLeft(g, fmt.Sprintf("%sPointText%d", item.key, i), item.x+340000, rowY+80000, colW-470000, 220000, point, pal.text, 980, false, "t", pptxBodyFont))
			}
		}
		return wrapSlide(sb.String())
	}
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

	if deckRenderFamily(deck) == "proposal" {
		tableX, tableY := shell.BodyX, shell.BodyY
		tableW := shell.BodyW
		sideW := 0
		if len(table.Rows) >= 2 {
			sideW = 1920000
			tableW -= sideW + 160000
		}
		renderProposalPanel(g, &sb, "tableProposalPanel", tableX, tableY, tableW, shell.BodyH, pal.card, proposalRowBorder(pal))
		renderProposalTopAccent(g, &sb, "tableProposalAccent", tableX, tableY, tableW, proposalAccentColor(pal, 0))
		colW := tableW / cols
		rowH := 480000
		headY := tableY + 100000
		for i, header := range table.Headers {
			x := tableX + i*colW
			sb.WriteString(spRect(g, fmt.Sprintf("hdr%d", i), x, headY, colW, rowH, mixHex(proposalCanvasFill(pal), pal.border, 0.12)))
			sb.WriteString(spTextLeft(g, fmt.Sprintf("hdrText%d", i), x+50000, headY+130000, colW-100000, 160000, strings.ToUpper(header), pal.text, 860, true, "t", pptxBodyFont))
		}
		for r, row := range table.Rows {
			fill := proposalRowFill(pal, r)
			for c, value := range row {
				x := tableX + c*colW
				y := headY + rowH + r*rowH
				sb.WriteString(spRect(g, fmt.Sprintf("cell%d_%d", r, c), x, y, colW, rowH, fill))
				sb.WriteString(spRect(g, fmt.Sprintf("cellBorderTop%d_%d", r, c), x, y, colW, 4500, proposalRowBorder(pal)))
				sb.WriteString(spTextLeft(g, fmt.Sprintf("cellText%d_%d", r, c), x+50000, y+120000, colW-100000, rowH-160000, value, pal.text, 920, false, "t", pptxBodyFont))
			}
		}
		if sideW > 0 {
			sideX := tableX + tableW + 160000
			sideAccent := proposalAccentColor(pal, 1)
			renderProposalPanel(g, &sb, "tableProposalSide", sideX, tableY, sideW, shell.BodyH, proposalRowFill(pal, 1), proposalRowBorder(pal))
			renderProposalSideAccent(g, &sb, "tableProposalSideAccent", sideX, tableY, shell.BodyH, sideAccent)
			renderProposalHeaderIcon(g, &sb, "tableSideIcon", sideX+90000, tableY+90000, 220000, pal, proposalSlideIconToken(s))
			sb.WriteString(spTextLeft(g, "panelTitle", sideX+360000, tableY+110000, sideW-440000, 180000, "DECISION SIGNAL", sideAccent, 900, true, "t", pptxBodyFont))
			sb.WriteString(spTextLeft(g, "panelBody", sideX+90000, tableY+460000, sideW-180000, shell.BodyH-560000, summarizeTable(table), pal.muted, 920, false, "t", pptxBodyFont))
		}
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

func renderSectionSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	if deckRenderFamily(deck) == "proposal" {
		bg := proposalHeaderFill(pal)
		muted := proposalMutedOnDark(pal)
		heading := slideHeadingOrFallback(s, "Section")
		sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, bg))
		sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 42000, pal.accent))
		sb.WriteString(spTextLeft(g, "sectionBadge", 900000, 1260000, 2400000, 220000, fmt.Sprintf("SECTION %d", slideIndex+1), pal.accent, 980, true, "t", pptxBodyFont))
		sb.WriteString(spRect(g, "sectionRule", 900000, 1660000, 1640000, 28000, pal.accent2))
		sb.WriteString(spTextLeft(g, "sectionHeading", 900000, 1940000, 5600000, 900000, heading, "FFFFFF", 3300, true, "t", pptxHeadingFont))
		if len(s.Points) > 0 {
			sb.WriteString(spTextLeft(g, "sectionBody", 900000, 3160000, 5200000, 520000, s.Points[0], muted, 1180, false, "t", pptxBodyFont))
		}
		sb.WriteString(spRect(g, "footerBar", 0, 6400000, 9144000, 360000, proposalFooterFill(pal)))
		return wrapSlide(sb.String())
	}
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	heading := slideHeadingOrFallback(s, "Section")
	sb.WriteString(spEllipse(g, "sectionOrb", 6800000, 680000, 1320000, 1320000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
	sb.WriteString(spRoundRect(g, "sectionPanel", 900000, 1460000, 6240000, 3480000, pal.card, deckBorder(pal), slideCornerRadius(deck, true)))
	sb.WriteString(spRect(g, "sectionRule", 1220000, 1860000, 1320000, 32000, pal.accent))
	sb.WriteString(spRoundRect(g, "sectionBadgeBg", 1220000, 1520000, 1480000, 220000, deckAccentWash(pal, 0.14), "", 0))
	sb.WriteString(spText(g, "sectionBadge", 1220000, 1520000, 1480000, 220000, fmt.Sprintf("Section %d", slideIndex+1), pal.accent, 940, true, "ctr", pptxBodyFont))
	sb.WriteString(spTextLeft(g, "sectionHeading", 1220000, 2180000, 5100000, 900000, heading, pal.text, 3300, true, "t", pptxHeadingFont))
	if len(s.Points) > 0 {
		sb.WriteString(spTextLeft(g, "sectionBody", 1220000, 3340000, 4740000, 760000, s.Points[0], pal.muted, 1200, false, "t", pptxBodyFont))
	}
	return wrapSlide(sb.String())
}

func renderBlankDeckSlide(s pptxSlide, pal pptxPalette, variant int, deck pptxDeckContext, slideIndex int) string {
	g := &idg{}
	var sb strings.Builder
	if deckRenderFamily(deck) == "proposal" {
		bg := proposalHeaderFill(pal)
		body := slideHeadingOrFallback(s, "Section Break")
		if len(s.Points) > 0 && strings.TrimSpace(s.Points[0]) != "" {
			body = s.Points[0]
		}
		sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, bg))
		sb.WriteString(spRect(g, "topStrip", 0, 0, 9144000, 42000, pal.accent))
		sb.WriteString(spTextLeft(g, "blankMark", 900000, 1780000, 2200000, 220000, "TRANSITION", pal.accent, 980, true, "t", pptxBodyFont))
		sb.WriteString(spRect(g, "blankRule", 900000, 2160000, 1480000, 28000, pal.accent2))
		sb.WriteString(spTextLeft(g, "blankBody", 900000, 2440000, 5400000, 900000, body, "FFFFFF", 2900, true, "t", pptxHeadingFont))
		sb.WriteString(spRect(g, "footerBar", 0, 6400000, 9144000, 360000, proposalFooterFill(pal)))
		return wrapSlide(sb.String())
	}
	sb.WriteString(spRect(g, "bg", 0, 0, 9144000, 6858000, pal.bg))
	body := slideHeadingOrFallback(s, "Section Break")
	if len(s.Points) > 0 && strings.TrimSpace(s.Points[0]) != "" {
		body = s.Points[0]
	}
	sb.WriteString(spEllipse(g, "blankOrb", 1100000, 1040000, 920000, 920000, deckAccentWash(pal, 0.12), 100, "", 0, 0))
	sb.WriteString(spRoundRect(g, "blankCard", 1380000, 1760000, 5880000, 2580000, pal.card, deckBorder(pal), slideCornerRadius(deck, true)))
	sb.WriteString(spRect(g, "blankRule", 1720000, 2060000, 1120000, 28000, pal.accent))
	sb.WriteString(spTextLeft(g, "blankMark", 1720000, 1740000, 2200000, 220000, "TRANSITION", pal.accent2, 960, true, "t", pptxBodyFont))
	sb.WriteString(spTextLeft(g, "blankBody", 1720000, 2360000, 4740000, 1180000, body, pal.text, 2500, true, "t", pptxHeadingFont))
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

func proposalStatsTakeaway(stats []pptxStat) string {
	if len(stats) == 0 {
		return "Key indicators that frame the decision."
	}
	var labels []string
	for _, stat := range stats {
		if label := strings.TrimSpace(stat.Label); label != "" {
			labels = append(labels, label)
		}
		if len(labels) == 3 {
			break
		}
	}
	if len(labels) == 0 {
		return "Key indicators that frame the decision."
	}
	if len(labels) == 1 {
		return fmt.Sprintf("Primary signal: %s.", labels[0])
	}
	if len(labels) == 2 {
		return fmt.Sprintf("Decision signals: %s and %s.", labels[0], labels[1])
	}
	return fmt.Sprintf("Decision signals: %s, %s, and %s.", labels[0], labels[1], labels[2])
}

func proposalLeadText(s pptxSlide, points []string) string {
	if notes := strings.TrimSpace(s.SpeakerNotes); notes != "" {
		return notes
	}
	for _, point := range points {
		if len([]rune(strings.TrimSpace(point))) >= 72 {
			return point
		}
	}
	if len(points) >= 2 {
		left, _ := splitCardText(points[0])
		right, _ := splitCardText(points[1])
		return fmt.Sprintf("Focus areas include %s and %s.", left, right)
	}
	return ""
}

func proposalSectionLabel(s pptxSlide) string {
	text := strings.ToLower(strings.TrimSpace(s.Heading))
	switch {
	case containsAny(text, "deliverable", "scope", "phase", "build", "implementation"):
		return "KEY DELIVERABLES"
	case containsAny(text, "assumption", "constraint", "dependency"):
		return "KEY ASSUMPTIONS"
	case containsAny(text, "risk", "issue"):
		return "KEY RISKS"
	case containsAny(text, "team", "role"):
		return "TEAM FOCUS"
	default:
		return "KEY POINTS"
	}
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
