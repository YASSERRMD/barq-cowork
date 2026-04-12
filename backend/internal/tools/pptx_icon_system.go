package tools

import (
	"fmt"
	"strings"
)

var legacyEmojiIcons = map[string]string{
	"⚡":  "automation",
	"🔒":  "shield",
	"📊":  "chart",
	"📈":  "growth",
	"🧩":  "integration",
	"🧠":  "strategy",
	"🌐":  "integration",
	"👥":  "people",
	"🧭":  "strategy",
	"📚":  "learning",
	"❤️": "health",
	"🌿":  "leaf",
}

func inferCardIcon(card pptxCard, index int) string {
	if icon := normalizeIconToken(card.Icon); icon != "" {
		return icon
	}
	if icon := inferIconFromText(card.Title + " " + card.Desc); icon != "" {
		return icon
	}
	fallback := []string{"automation", "shield", "chart", "integration", "people", "strategy"}
	return fallback[index%len(fallback)]
}

func inferIconFromText(text string) string {
	return normalizeIconToken(text)
}

func normalizeIconToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if token, ok := legacyEmojiIcons[raw]; ok {
		return token
	}

	text := normalizeThemeText(raw)
	switch {
	case hasWord(text, "shield", "lock", "security", "secure", "control", "governance", "compliance", "privacy", "risk"):
		return "shield"
	case hasWord(text, "chart", "graph", "analytics", "data", "insight", "metric", "dashboard"):
		return "chart"
	case hasWord(text, "growth", "revenue", "sales", "finance", "market", "profit"):
		return "growth"
	case hasWord(text, "automation", "workflow", "process", "ops", "operations", "speed", "efficiency", "flow"):
		return "automation"
	case hasWord(text, "integration", "connect", "connection", "platform", "system", "api", "network"):
		return "integration"
	case hasWord(text, "people", "team", "customer", "talent", "user", "community"):
		return "people"
	case hasWord(text, "strategy", "roadmap", "plan", "planning", "direction", "target", "goal"):
		return "strategy"
	case hasWord(text, "learning", "education", "school", "student", "teacher", "classroom", "kid", "kids", "children"):
		return "learning"
	case hasWord(text, "health", "medical", "patient", "clinical", "care"):
		return "health"
	case hasWord(text, "environment", "climate", "green", "sustainability", "carbon", "eco", "nature"):
		return "leaf"
	case hasWord(text, "logistics", "delivery", "fleet", "shipping", "warehouse", "transport", "supply"):
		return "logistics"
	case hasWord(text, "creative", "design", "idea", "brand", "innovation", "spark"):
		return "spark"
	default:
		return ""
	}
}

func renderCardIconBadge(g *idg, sb *strings.Builder, name string, x, y, size int, pal pptxPalette, token string, variant int) {
	token = firstNonEmpty(normalizeIconToken(token), "strategy")
	switch variant % 3 {
	case 1:
		sb.WriteString(spRoundRect(g, name+"Bg", x, y, size, size, pal.accent, pal.accent2, 12))
	default:
		sb.WriteString(spEllipse(g, name+"Bg", x, y, size, size, pal.accent, 100, "", 0, 0))
	}
	renderPPTXIconGlyph(g, sb, name, x, y, size, pal, token)
}

func renderPPTXIconGlyph(g *idg, sb *strings.Builder, name string, x, y, size int, pal pptxPalette, token string) {
	fg := pal.text
	pad := size / 5
	innerX := x + pad
	innerY := y + pad
	innerW := size - pad*2
	innerH := size - pad*2

	switch token {
	case "shield":
		sb.WriteString(spPresetShape(g, name+"Shield", "hexagon", innerX, innerY, innerW, innerH, "", 0, fg, 12700, 100))
		sb.WriteString(spRect(g, name+"ShieldBar", x+size/2-22000, y+size/2-90000, 44000, 180000, fg))
		sb.WriteString(spRect(g, name+"ShieldArm", x+size/2-90000, y+size/2-22000, 180000, 44000, fg))
	case "chart":
		barW := size / 8
		baseY := y + size - pad
		xs := []int{x + pad + 20000, x + size/2 - barW/2, x + size - pad - barW - 20000}
		heights := []int{size / 4, size / 2, size * 3 / 5}
		for i := range xs {
			sb.WriteString(spRoundRect(g, fmt.Sprintf("%sBar%d", name, i), xs[i], baseY-heights[i], barW, heights[i], fg, "", 0))
		}
		sb.WriteString(spRect(g, name+"ChartBase", x+pad, baseY, size-pad*2, 18000, fg))
	case "growth":
		barW := size / 8
		baseY := y + size - pad
		xs := []int{x + pad + 20000, x + size/2 - barW/2, x + size - pad - barW - 20000}
		heights := []int{size / 5, size / 3, size / 2}
		for i := range xs {
			sb.WriteString(spRoundRect(g, fmt.Sprintf("%sGrowthBar%d", name, i), xs[i], baseY-heights[i], barW, heights[i], fg, "", 0))
		}
		sb.WriteString(spRightArrow(g, name+"GrowthArrow", x+size/2-70000, y+pad, 180000, 90000, fg))
	case "automation":
		sb.WriteString(spRightArrow(g, name+"Flow1", x+pad-20000, y+size/2-90000, size/2, 180000, fg))
		sb.WriteString(spRightArrow(g, name+"Flow2", x+size/2-40000, y+size/2-90000, size/2-pad+20000, 180000, fg))
		sb.WriteString(spEllipse(g, name+"FlowDot", x+size-pad-90000, y+size/2-90000, 180000, 180000, pal.accent2, 100, fg, 6350, 100))
	case "integration":
		ring := size / 3
		sb.WriteString(spEllipse(g, name+"RingLeft", x+pad, y+size/2-ring/2, ring, ring, "", 0, fg, 12700, 100))
		sb.WriteString(spEllipse(g, name+"RingRight", x+size-pad-ring, y+size/2-ring/2, ring, ring, "", 0, fg, 12700, 100))
		sb.WriteString(spRect(g, name+"RingBridge", x+size/2-50000, y+size/2-22000, 100000, 44000, fg))
	case "people":
		head := size / 5
		sb.WriteString(spEllipse(g, name+"Head1", x+size/2-head-30000, y+pad, head, head, fg, 100, "", 0, 0))
		sb.WriteString(spEllipse(g, name+"Head2", x+size/2+30000, y+pad, head, head, fg, 100, "", 0, 0))
		sb.WriteString(spRoundRect(g, name+"Body1", x+size/2-head-50000, y+pad+head+30000, head+70000, size/3, fg, "", 0))
		sb.WriteString(spRoundRect(g, name+"Body2", x+size/2, y+pad+head+30000, head+70000, size/3, fg, "", 0))
	case "learning":
		pageW := innerW/2 - 30000
		sb.WriteString(spPresetShape(g, name+"PageLeft", "roundRect", innerX, innerY+40000, pageW, innerH-80000, "", 0, fg, 12700, 100))
		sb.WriteString(spPresetShape(g, name+"PageRight", "roundRect", innerX+pageW+60000, innerY+40000, pageW, innerH-80000, "", 0, fg, 12700, 100))
		sb.WriteString(spRect(g, name+"Spine", x+size/2-15000, innerY+50000, 30000, innerH-100000, fg))
	case "health":
		sb.WriteString(spRect(g, name+"HealthV", x+size/2-32000, y+pad, 64000, size-pad*2, fg))
		sb.WriteString(spRect(g, name+"HealthH", x+pad, y+size/2-32000, size-pad*2, 64000, fg))
	case "leaf":
		sb.WriteString(spEllipse(g, name+"LeafLeft", x+size/2-140000, y+pad+40000, 180000, 240000, fg, 100, "", 0, 0))
		sb.WriteString(spEllipse(g, name+"LeafRight", x+size/2-20000, y+pad, 180000, 240000, fg, 100, "", 0, 0))
		sb.WriteString(spRect(g, name+"LeafStem", x+size/2-12000, y+size/2+20000, 24000, size/3, fg))
	case "logistics":
		sb.WriteString(spPresetShape(g, name+"Box", "roundRect", innerX, y+size/2-100000, innerW-90000, 200000, "", 0, fg, 12700, 100))
		sb.WriteString(spRect(g, name+"BoxTape", x+size/2-18000, y+size/2-100000, 36000, 200000, fg))
		sb.WriteString(spRightArrow(g, name+"BoxArrow", x+size-size/3, y+pad+40000, size/4, 110000, fg))
	case "spark":
		sb.WriteString(spPresetShape(g, name+"SparkDiamond", "diamond", x+size/2-70000, y+pad, 140000, 140000, fg, 100, "", 0, 0))
		sb.WriteString(spPresetShape(g, name+"SparkDiamond2", "diamond", x+pad+30000, y+size/2-30000, 110000, 110000, fg, 100, "", 0, 0))
		sb.WriteString(spPresetShape(g, name+"SparkDiamond3", "diamond", x+size-pad-130000, y+size/2+20000, 110000, 110000, fg, 100, "", 0, 0))
	default:
		sb.WriteString(spEllipse(g, name+"TargetOuter", innerX, innerY, innerW, innerH, "", 0, fg, 12700, 100))
		sb.WriteString(spEllipse(g, name+"TargetInner", x+size/2-70000, y+size/2-70000, 140000, 140000, fg, 100, "", 0, 0))
	}
}

func previewCardIconSVG(token string, pal pptxPalette) string {
	token = firstNonEmpty(normalizeIconToken(token), "strategy")
	accent := hexColor(pal.accent)
	soft := hexColor(pal.accent2)
	fg := hexColor(pal.text)

	var glyph string
	switch token {
	case "shield":
		glyph = `<polygon points="32,10 48,18 48,34 32,50 16,34 16,18" fill="none" stroke="` + fg + `" stroke-width="4"/><path d="M24 30h16M32 22v16" stroke="` + fg + `" stroke-width="4" stroke-linecap="round"/>`
	case "chart":
		glyph = `<rect x="16" y="34" width="8" height="14" rx="2" fill="` + fg + `"/><rect x="28" y="24" width="8" height="24" rx="2" fill="` + fg + `"/><rect x="40" y="18" width="8" height="30" rx="2" fill="` + fg + `"/><path d="M14 48h36" stroke="` + fg + `" stroke-width="3" stroke-linecap="round"/>`
	case "growth":
		glyph = `<rect x="16" y="38" width="8" height="10" rx="2" fill="` + fg + `"/><rect x="28" y="30" width="8" height="18" rx="2" fill="` + fg + `"/><rect x="40" y="22" width="8" height="26" rx="2" fill="` + fg + `"/><path d="M23 26l9-7 8 5 9-7" stroke="` + fg + `" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`
	case "automation":
		glyph = `<path d="M14 30h18l-6-6m6 6-6 6" stroke="` + fg + `" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"/><path d="M30 30h18l-6-6m6 6-6 6" stroke="` + fg + `" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"/><circle cx="49" cy="30" r="5" fill="` + soft + `" stroke="` + fg + `" stroke-width="2"/>`
	case "integration":
		glyph = `<circle cx="24" cy="32" r="10" fill="none" stroke="` + fg + `" stroke-width="4"/><circle cx="40" cy="32" r="10" fill="none" stroke="` + fg + `" stroke-width="4"/><path d="M28 32h8" stroke="` + fg + `" stroke-width="4" stroke-linecap="round"/>`
	case "people":
		glyph = `<circle cx="25" cy="22" r="6" fill="` + fg + `"/><circle cx="39" cy="22" r="6" fill="` + fg + `"/><rect x="18" y="31" width="14" height="15" rx="5" fill="` + fg + `"/><rect x="32" y="31" width="14" height="15" rx="5" fill="` + fg + `"/>`
	case "learning":
		glyph = `<rect x="15" y="17" width="15" height="28" rx="3" fill="none" stroke="` + fg + `" stroke-width="3.5"/><rect x="34" y="17" width="15" height="28" rx="3" fill="none" stroke="` + fg + `" stroke-width="3.5"/><path d="M32 18v28" stroke="` + fg + `" stroke-width="3"/>`
	case "health":
		glyph = `<path d="M32 16v32M16 32h32" stroke="` + fg + `" stroke-width="6" stroke-linecap="round"/>`
	case "leaf":
		glyph = `<path d="M22 38c0-11 8-18 18-20 1 10-3 20-14 23-3 1-4-1-4-3Z" fill="` + fg + `"/><path d="M25 40c6-8 12-13 20-17" stroke="` + accent + `" stroke-width="3" stroke-linecap="round"/>`
	case "logistics":
		glyph = `<rect x="16" y="26" width="24" height="18" rx="3" fill="none" stroke="` + fg + `" stroke-width="3.5"/><path d="M28 26v18" stroke="` + fg + `" stroke-width="3"/><path d="M40 22h10l-4-4m4 4-4 4" stroke="` + fg + `" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`
	case "spark":
		glyph = `<path d="M32 14l5 9 9 5-9 5-5 9-5-9-9-5 9-5 5-9Z" fill="` + fg + `"/><path d="M15 46l3 5 5 3-5 3-3 5-3-5-5-3 5-3 3-5Z" fill="` + soft + `"/>`
	default:
		glyph = `<circle cx="32" cy="32" r="16" fill="none" stroke="` + fg + `" stroke-width="4"/><circle cx="32" cy="32" r="6" fill="` + fg + `"/>`
	}

	return `<svg viewBox="0 0 64 64" aria-hidden="true" focusable="false"><circle cx="32" cy="32" r="28" fill="` + accent + `"/>` + glyph + `</svg>`
}
