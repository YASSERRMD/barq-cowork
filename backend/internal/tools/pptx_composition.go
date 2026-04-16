package tools

import (
	"regexp"
	"strings"
)

// compositionParams mirrors the TypeScript CompositionParams in
// backend/pptx_renderer/src/render-pptx.ts so the Go preview renders layouts
// identical to the final .pptx output.
type compositionParams struct {
	// Layout geometry
	SplitRatio float64 // 0 = no split; 0.32, 0.40, 0.52 = left panel width ratio
	HeroFirst  bool    // hero/lead block appears before content (top or left)
	Columns    int     // content columns: 1, 2, 3
	Horizontal bool    // content flows left-right (not top-bottom)

	// Surface
	PanelFilled  bool // lead/hero panel has solid fill
	PanelOutline bool // lead panel has outline only
	PanelGlass   bool // lead panel has semi-transparent fill

	// Accent placement
	AccentRail bool // left vertical accent bar
	AccentBand bool // top horizontal accent bar
	AccentChip bool // small badge accent
	AccentGlow bool // ambient glow effect

	// Density
	Density float64 // 0.75=airy, 1.0=balanced, 1.28=dense (scales font/spacing)
}

// Regex patterns mirror the TypeScript source exactly. Keep them synchronized
// with parseComposition() in backend/pptx_renderer/src/render-pptx.ts.
var (
	compSplitPattern      = regexp.MustCompile(`split|side|panel|aside|left|right`)
	compSplitWidePattern  = regexp.MustCompile(`wide|broad|major|60|65`)
	compSplitNarrowPattrn = regexp.MustCompile(`narrow|minor|30|35`)
	compCols3Pattern      = regexp.MustCompile(`3.col|three.col|3col|triple`)
	compCols2Pattern      = regexp.MustCompile(`2.col|two.col|2col|dual|double|split|grid`)
	compHeroPattern       = regexp.MustCompile(`hero|spotlight|focus|stage|lead|featured|highlight|banner|above|top`)
	compHorizPattern      = regexp.MustCompile(`rail|horizontal|row|h.?flow`)
	compPanelFilledPs     = regexp.MustCompile(`solid|filled|block|dark|bold`)
	compPanelFilledLs     = regexp.MustCompile(`solid|block`)
	compPanelOutlinePs    = regexp.MustCompile(`outline|border|wire|ghost`)
	compAccentBandPattern = regexp.MustCompile(`band|top|bar|stripe`)
	compAccentChipPattern = regexp.MustCompile(`chip|badge|dot|pill`)
	compAccentGlowPattern = regexp.MustCompile(`glow|ambient|soft`)
	compDensityAiryPttn   = regexp.MustCompile(`airy|sparse|open`)
	compDensityDensePttn  = regexp.MustCompile(`dense|compact|tight`)
)

// parseComposition mirrors parseComposition() from the TypeScript renderer.
func parseComposition(design *pptxSlideDesign) compositionParams {
	ls := ""
	ps := "soft"
	am := "rail"
	dn := "balanced"
	if design != nil {
		if v := strings.TrimSpace(design.LayoutStyle); v != "" {
			ls = v
		}
		if v := strings.TrimSpace(design.PanelStyle); v != "" {
			ps = v
		}
		if v := strings.TrimSpace(design.AccentMode); v != "" {
			am = v
		}
		if v := strings.TrimSpace(design.Density); v != "" {
			dn = v
		}
	}
	ls = strings.ToLower(ls)
	ps = strings.ToLower(ps)
	am = strings.ToLower(am)
	dn = strings.ToLower(dn)

	splitRatio := 0.0
	if compSplitPattern.MatchString(ls) {
		switch {
		case compSplitWidePattern.MatchString(ls):
			splitRatio = 0.52
		case compSplitNarrowPattrn.MatchString(ls):
			splitRatio = 0.32
		default:
			splitRatio = 0.40
		}
	}

	columns := 1
	switch {
	case compCols3Pattern.MatchString(ls):
		columns = 3
	case compCols2Pattern.MatchString(ls):
		columns = 2
	}

	heroFirst := compHeroPattern.MatchString(ls)
	horizontal := compHorizPattern.MatchString(ls)

	panelFilled := compPanelFilledPs.MatchString(ps) || compPanelFilledLs.MatchString(ls)
	panelOutline := compPanelOutlinePs.MatchString(ps)

	accentBand := compAccentBandPattern.MatchString(am)
	accentChip := compAccentChipPattern.MatchString(am)
	accentGlow := compAccentGlowPattern.MatchString(am)
	accentRail := !accentBand && !accentChip && !accentGlow

	density := 1.0
	switch {
	case compDensityAiryPttn.MatchString(dn):
		density = 0.75
	case compDensityDensePttn.MatchString(dn):
		density = 1.28
	}

	return compositionParams{
		SplitRatio:   splitRatio,
		HeroFirst:    heroFirst,
		Columns:      columns,
		Horizontal:   horizontal,
		PanelFilled:  panelFilled,
		PanelOutline: panelOutline,
		PanelGlass:   !panelFilled && !panelOutline,
		AccentRail:   accentRail,
		AccentBand:   accentBand,
		AccentChip:   accentChip,
		AccentGlow:   accentGlow,
		Density:      density,
	}
}
