package tools

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type plannedPPTXPresentation struct {
	ThemeName string
	DeckPlan  plannedPPTXDeckPlan
	Slides    []plannedPPTXSlide
}

type plannedPPTXDeckPlan struct {
	Subject         string   `json:"subject"`
	Audience        string   `json:"audience"`
	NarrativeArc    string   `json:"narrative_arc"`
	VisualDirection string   `json:"visual_direction"`
	DominantNeed    string   `json:"dominant_need"`
	LayoutMix       []string `json:"layout_mix,omitempty"`
}

type plannedPPTXSlide struct {
	Slide   pptxSlide
	Layout  string
	Variant int
	Plan    plannedPPTXSlidePlan
	Audit   plannedPPTXSlideAudit
}

type plannedPPTXSlidePlan struct {
	Purpose       string `json:"purpose"`
	Visual        string `json:"visual"`
	ContentSource string `json:"content_source"`
}

type plannedPPTXSlideAudit struct {
	ContentFit bool     `json:"content_fit"`
	LayoutFit  bool     `json:"layout_fit"`
	VisualFit  bool     `json:"visual_fit"`
	Notes      []string `json:"notes,omitempty"`
}

func planPPTXPresentation(title, subtitle string, slides []pptxSlide, themeName string) plannedPPTXPresentation {
	if strings.TrimSpace(themeName) == "" {
		themeName = pickThemeName(title, subtitle)
	}

	deckPlan := deriveDeckPlan(title, subtitle, slides, themeName)
	planned := plannedPPTXPresentation{
		ThemeName: themeName,
		DeckPlan:  deckPlan,
		Slides:    make([]plannedPPTXSlide, 0, len(slides)),
	}

	for i, slide := range slides {
		planned.Slides = append(planned.Slides, planPPTXSlide(slide, deckPlan, title, i))
	}

	planned.DeckPlan.LayoutMix = plannedLayoutMix(planned.Slides)
	planned.DeckPlan.NarrativeArc = deriveNarrativeArc(planned.Slides)
	return planned
}

func validatePPTXPresentation(planned plannedPPTXPresentation) error {
	if len(planned.Slides) == 0 {
		return fmt.Errorf("presentation requires at least one content slide")
	}

	var problems []string
	for i, slide := range planned.Slides {
		if slide.Audit.ContentFit && slide.Audit.LayoutFit && slide.Audit.VisualFit {
			continue
		}
		label := firstNonEmpty(slide.Slide.Heading, fmt.Sprintf("Slide %d", i+2))
		note := strings.Join(slide.Audit.Notes, "; ")
		if note == "" {
			note = "failed slide audit"
		}
		problems = append(problems, fmt.Sprintf("%s: %s", label, note))
	}
	if len(problems) > 0 {
		return fmt.Errorf(strings.Join(problems, " | "))
	}
	return nil
}

func planPPTXSlide(s pptxSlide, deck plannedPPTXDeckPlan, deckTitle string, index int) plannedPPTXSlide {
	layout := effectivePPTXLayout(s)
	planned := s
	planned.Type = layout
	planned.Layout = layout

	if strings.TrimSpace(planned.Heading) == "" {
		planned.Heading = defaultSlideHeading(layout, deckTitle, index)
	}

	contentSource := "explicit"

	switch layout {
	case "stats":
		stats := effectiveStats(planned)
		switch {
		case len(s.Stats) > 0:
			contentSource = "explicit"
		case len(stats) > 0:
			contentSource = "derived"
		default:
			contentSource = "subject-default"
			stats = themedFallbackStats(deck, planned.Heading)
		}
		planned.Stats = stats
	case "steps":
		steps := effectiveSteps(planned)
		switch {
		case len(s.Steps) > 0:
			contentSource = "explicit"
		case len(s.Points) > 0:
			contentSource = "derived"
		default:
			contentSource = "subject-default"
			steps = themedFallbackSteps(deck, planned.Heading)
		}
		planned.Steps = steps
	case "cards":
		cards := effectiveCards(planned)
		switch {
		case len(s.Cards) > 0:
			contentSource = "explicit"
		case len(s.Points) > 0:
			contentSource = "derived"
		default:
			contentSource = "subject-default"
			cards = themedFallbackCards(deck, planned.Heading)
		}
		for i := range cards {
			cards[i].Icon = inferCardIcon(cards[i], i)
		}
		planned.Cards = cards
	case "chart":
		categories := chartCategoriesOrFallback(planned)
		series := effectiveChartSeries(planned)
		switch {
		case len(s.ChartSeries) > 0:
			contentSource = "explicit"
		case len(series) > 0:
			contentSource = "derived"
		default:
			contentSource = "subject-default"
			categories, series = themedFallbackChart(deck, planned.Heading)
		}
		planned.ChartCategories = trimAndPadLabels(categories, len(categories))
		planned.ChartSeries = series
		planned.ChartType = firstNonEmpty(strings.TrimSpace(planned.ChartType), themedChartType(deck))
		if strings.TrimSpace(planned.YLabel) == "" {
			planned.YLabel = themedYAxisLabel(deck)
		}
	case "timeline":
		items := planned.Timeline
		if len(items) == 0 {
			contentSource = "subject-default"
			items = themedFallbackTimeline(deck, planned.Heading)
		}
		planned.Timeline = items
	case "compare":
		left, right := effectiveCompareColumns(planned)
		if s.LeftColumn == nil && s.RightColumn == nil {
			contentSource = "subject-default"
			left, right = themedFallbackCompare(deck, planned.Heading)
		}
		planned.LeftColumn = &left
		planned.RightColumn = &right
	case "table":
		table := planned.Table
		if table == nil || len(table.Headers) == 0 {
			contentSource = "subject-default"
			table = themedFallbackTable(deck, planned.Heading)
		}
		planned.Table = table
	case "title":
		if len(planned.Points) == 0 {
			contentSource = "subject-default"
			planned.Points = []string{sectionTransitionCopy(deck, planned.Heading)}
		}
	case "blank":
		if len(planned.Points) == 0 {
			contentSource = "subject-default"
			planned.Points = []string{sectionTransitionCopy(deck, planned.Heading)}
		}
	default:
		points := safePoints(planned.Points, 6)
		if len(points) == 0 {
			contentSource = "subject-default"
			points = themedFallbackBullets(deck, planned.Heading)
		}
		planned.Points = points
	}

	plan := plannedPPTXSlidePlan{
		Purpose:       slidePurpose(deck, layout, planned.Heading),
		Visual:        slideVisual(layout, planned.Heading),
		ContentSource: contentSource,
	}
	if strings.TrimSpace(planned.SpeakerNotes) == "" {
		planned.SpeakerNotes = generatedSpeakerNotes(planned, plan)
	}

	variant := slideVisualVariant(deckTitle, planned.Heading, layout, index)
	audit := auditPlannedSlide(planned, layout)

	return plannedPPTXSlide{
		Slide:   planned,
		Layout:  layout,
		Variant: variant,
		Plan:    plan,
		Audit:   audit,
	}
}

func deriveDeckPlan(title, subtitle string, slides []pptxSlide, themeName string) plannedPPTXDeckPlan {
	subject := strings.TrimSpace(firstNonEmpty(title, subtitle, "Presentation"))
	return plannedPPTXDeckPlan{
		Subject:         subject,
		Audience:        deriveAudience(subtitle, title),
		NarrativeArc:    deriveNarrativeArcFromInputs(slides),
		VisualDirection: themeVisualDirection(themeName),
		DominantNeed:    themeDominantNeed(themeName),
		LayoutMix:       plannedLayoutMixFromInputs(slides),
	}
}

func deriveAudience(subtitle, title string) string {
	text := strings.ToLower(strings.TrimSpace(title + " " + subtitle))
	switch {
	case containsAny(text, "executive", "board", "leadership", "steering", "investor"):
		return "executive stakeholders"
	case containsAny(text, "sales", "go to market", "customer", "prospect"):
		return "commercial stakeholders"
	case containsAny(text, "training", "enablement", "workshop", "team"):
		return "operational teams"
	case containsAny(text, "product", "engineering", "platform", "delivery"):
		return "product and delivery teams"
	default:
		return "mixed business audience"
	}
}

func deriveNarrativeArcFromInputs(slides []pptxSlide) string {
	if len(slides) == 0 {
		return "Context -> Evidence -> Recommendation"
	}
	layouts := plannedLayoutMixFromInputs(slides)
	if len(layouts) == 0 {
		return "Context -> Evidence -> Recommendation"
	}
	var stages []string
	for _, layout := range layouts {
		switch layout {
		case "title", "blank", "bullets":
			stages = append(stages, "Context")
		case "stats", "chart":
			stages = append(stages, "Evidence")
		case "steps", "timeline":
			stages = append(stages, "Execution")
		case "cards":
			stages = append(stages, "Capabilities")
		case "compare", "table":
			stages = append(stages, "Decision")
		}
	}
	return strings.Join(uniqueStrings(stages), " -> ")
}

func deriveNarrativeArc(slides []plannedPPTXSlide) string {
	if len(slides) == 0 {
		return "Context -> Evidence -> Recommendation"
	}
	var stages []string
	for _, slide := range slides {
		stages = append(stages, slide.Plan.Purpose)
	}
	return strings.Join(uniqueStrings(stages), " -> ")
}

func themeVisualDirection(themeName string) string {
	switch themeName {
	case "healthcare":
		return "calm clinical credibility with clean data emphasis"
	case "education":
		return "structured learning flow with warm instructional cues"
	case "environment":
		return "impact-focused sustainability story with milestone framing"
	case "finance":
		return "measured business rigor with signal-first visuals"
	case "creative":
		return "high-contrast storytelling with expressive section breaks"
	case "security":
		return "controlled risk narrative with strong contrast and auditability"
	case "data":
		return "analysis-led narrative with charts and structured comparisons"
	case "logistics":
		return "operational clarity with process and timeline visuals"
	case "retail":
		return "commercial performance story with growth signals"
	case "hr":
		return "people-centric change narrative with adoption emphasis"
	default:
		return "modern product narrative with varied layouts and clear hierarchy"
	}
}

func themeDominantNeed(themeName string) string {
	switch themeName {
	case "security", "finance":
		return "governance"
	case "logistics", "data", "tech":
		return "operations"
	case "retail", "creative":
		return "growth"
	case "education", "hr":
		return "adoption"
	case "environment", "healthcare":
		return "impact"
	default:
		return "execution"
	}
}

func plannedLayoutMix(slides []plannedPPTXSlide) []string {
	var layouts []string
	for _, slide := range slides {
		if slide.Layout != "" {
			layouts = append(layouts, slide.Layout)
		}
	}
	return uniqueStrings(layouts)
}

func plannedLayoutMixFromInputs(slides []pptxSlide) []string {
	var layouts []string
	for _, slide := range slides {
		if layout := effectivePPTXLayout(slide); layout != "" {
			layouts = append(layouts, layout)
		}
	}
	return uniqueStrings(layouts)
}

func slidePurpose(deck plannedPPTXDeckPlan, layout, heading string) string {
	subject := shortSubject(deck.Subject, heading)
	switch layout {
	case "stats":
		return "Prove the scale of " + subject + " with headline metrics."
	case "steps":
		return "Explain how the " + subject + " rollout should execute."
	case "cards":
		return "Show the core capabilities that support " + subject + "."
	case "chart":
		return "Quantify the trend behind " + subject + "."
	case "timeline":
		return "Sequence the milestones for " + subject + "."
	case "compare":
		return "Contrast the current and target state for " + subject + "."
	case "table":
		return "Give a structured decision reference for " + subject + "."
	case "title":
		return "Reset the narrative around the next section."
	case "blank":
		return "Pause before the next section of the story."
	default:
		return "Frame the key points the audience must remember about " + subject + "."
	}
}

func slideVisual(layout, heading string) string {
	switch layout {
	case "stats":
		return "metric cards with quick scan emphasis"
	case "steps":
		return "process flow with ordered stage markers"
	case "cards":
		return "icon-led capability grid"
	case "chart":
		return "data chart with summary insight"
	case "timeline":
		return "milestone timeline"
	case "compare":
		return "side-by-side comparison panels"
	case "table":
		return "decision matrix"
	case "title":
		return "section divider"
	case "blank":
		return "transition slide"
	default:
		return "narrative bullet cards"
	}
}

func generatedSpeakerNotes(slide pptxSlide, plan plannedPPTXSlidePlan) string {
	anchor := firstNonEmpty(slide.Heading, "this topic")
	switch slide.Type {
	case "stats":
		return fmt.Sprintf("Lead with %s and explain the few numbers that matter most. Use this slide to %s", anchor, strings.ToLower(plan.Purpose))
	case "chart":
		return fmt.Sprintf("Use %s to call out the leading signal, then explain what is driving the trend.", anchor)
	case "compare":
		return fmt.Sprintf("Walk left to right on %s and explain why the target state is the preferred choice.", anchor)
	default:
		return fmt.Sprintf("Use %s to %s", anchor, strings.ToLower(plan.Purpose))
	}
}

func auditPlannedSlide(slide pptxSlide, layout string) plannedPPTXSlideAudit {
	audit := plannedPPTXSlideAudit{}
	switch layout {
	case "stats":
		audit.ContentFit = len(slide.Stats) >= 2 && len(slide.Stats) <= 4
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = statsVisualReady(slide.Stats)
	case "steps":
		audit.ContentFit = len(slide.Steps) >= 3 && len(slide.Steps) <= 6
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = len(slide.Steps) > 0
	case "cards":
		audit.ContentFit = len(slide.Cards) >= 3 && len(slide.Cards) <= 6
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = cardsVisualReady(slide.Cards)
	case "chart":
		audit.ContentFit = len(slide.ChartCategories) >= 2 && len(slide.ChartSeries) > 0
		audit.LayoutFit = chartLayoutReady(slide.ChartCategories, slide.ChartSeries)
		audit.VisualFit = strings.TrimSpace(slide.ChartType) != ""
	case "timeline":
		audit.ContentFit = len(slide.Timeline) >= 3 && len(slide.Timeline) <= 5
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = timelineVisualReady(slide.Timeline)
	case "compare":
		audit.ContentFit = slide.LeftColumn != nil && slide.RightColumn != nil
		audit.LayoutFit = compareLayoutReady(slide.LeftColumn, slide.RightColumn)
		audit.VisualFit = audit.LayoutFit
	case "table":
		audit.ContentFit = slide.Table != nil && len(slide.Table.Headers) >= 2 && len(slide.Table.Rows) >= 1
		audit.LayoutFit = tableLayoutReady(slide.Table)
		audit.VisualFit = audit.LayoutFit
	case "title", "blank":
		audit.ContentFit = strings.TrimSpace(slide.Heading) != "" || len(slide.Points) > 0
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = true
	default:
		audit.ContentFit = len(slide.Points) >= 2 && len(slide.Points) <= 6
		audit.LayoutFit = audit.ContentFit
		audit.VisualFit = true
	}

	if !audit.ContentFit {
		audit.Notes = append(audit.Notes, "content density does not fit the selected slide type")
	}
	if !audit.LayoutFit {
		audit.Notes = append(audit.Notes, "content structure does not fit the selected layout")
	}
	if !audit.VisualFit {
		audit.Notes = append(audit.Notes, "visual elements are missing or incomplete")
	}
	return audit
}

func defaultSlideHeading(layout, deckTitle string, index int) string {
	switch layout {
	case "title":
		return "Section"
	case "stats":
		return "Key Metrics"
	case "steps":
		return "How It Works"
	case "cards":
		return "Core Capabilities"
	case "chart":
		return "Trend Analysis"
	case "timeline":
		return "Roadmap"
	case "compare":
		return "Comparison"
	case "table":
		return "Decision Matrix"
	case "blank":
		return deckTitle
	default:
		return "Slide " + strconv.Itoa(index+2)
	}
}

func themedFallbackBullets(deck plannedPPTXDeckPlan, heading string) []string {
	subject := shortSubject(deck.Subject, heading)
	switch deck.DominantNeed {
	case "governance":
		return []string{
			"Clarify the control objective behind " + subject,
			"Show where current governance friction slows execution",
			"Define the decision required to move forward with confidence",
		}
	case "growth":
		return []string{
			"Explain why " + subject + " matters commercially right now",
			"Highlight the biggest opportunity unlocked by better execution",
			"Close with the action that accelerates adoption or revenue",
		}
	case "adoption":
		return []string{
			"Frame the audience problem that " + subject + " is solving",
			"Show what makes the experience easier to adopt",
			"End with the behavior change this deck is driving",
		}
	default:
		return []string{
			"Frame the operating problem behind " + subject,
			"Show the constraint that is currently blocking progress",
			"Define the next decision or rollout action",
		}
	}
}

func themedFallbackStats(deck plannedPPTXDeckPlan, heading string) []pptxStat {
	subject := shortSubject(deck.Subject, heading)
	switch deck.DominantNeed {
	case "governance":
		return []pptxStat{
			{Value: "99.9%", Label: "Control Coverage", Desc: "Critical checks enforced for " + subject},
			{Value: "64%", Label: "Risk Reduction", Desc: "Lower exposure after standardization"},
			{Value: "2.4x", Label: "Audit Speed", Desc: "Faster review and evidence gathering"},
		}
	case "growth":
		return []pptxStat{
			{Value: "31%", Label: "Pipeline Lift", Desc: "Commercial upside tied to " + subject},
			{Value: "18%", Label: "Conversion Gain", Desc: "Higher progression through the funnel"},
			{Value: "2.1x", Label: "ROI", Desc: "Return achieved from the rollout"},
		}
	default:
		return []pptxStat{
			{Value: "92%", Label: "Adoption", Desc: "Strong usage across the intended audience"},
			{Value: "3.1x", Label: "Efficiency", Desc: "Operating improvement from better execution"},
			{Value: "14d", Label: "Payback", Desc: "Time to measurable value"},
		}
	}
}

func themedFallbackSteps(deck plannedPPTXDeckPlan, heading string) []string {
	subject := shortSubject(deck.Subject, heading)
	return []string{
		"Align the target outcome for " + subject,
		"Map the workflow and ownership model",
		"Launch the first controlled rollout",
		"Measure the signal and refine the process",
	}
}

func themedFallbackCards(deck plannedPPTXDeckPlan, heading string) []pptxCard {
	subject := shortSubject(deck.Subject, heading)
	return []pptxCard{
		{Icon: "⚡", Title: "Faster Flow", Desc: subject + " moves with less friction"},
		{Icon: "🔒", Title: "Better Control", Desc: "Clear governance and safer execution"},
		{Icon: "📊", Title: "Visible Signal", Desc: "Progress is measurable and reviewable"},
		{Icon: "🧩", Title: "Operational Fit", Desc: "Fits existing teams, systems, and decisions"},
	}
}

func themedFallbackChart(deck plannedPPTXDeckPlan, heading string) ([]string, []pptxChartSeries) {
	categories := []string{"Q1", "Q2", "Q3", "Q4"}
	name := firstNonEmpty(trimTitlePhrase(heading), shortSubject(deck.Subject, heading), "Adoption")
	return categories, []pptxChartSeries{
		{Name: name, Values: []float64{18, 33, 51, 72}},
	}
}

func themedFallbackTimeline(deck plannedPPTXDeckPlan, heading string) []pptxTimelineItem {
	subject := shortSubject(deck.Subject, heading)
	return []pptxTimelineItem{
		{Date: "Q1", Title: "Discover", Desc: "Define the scope for " + subject},
		{Date: "Q2", Title: "Pilot", Desc: "Launch the first operational release"},
		{Date: "Q3", Title: "Expand", Desc: "Roll into more teams and workflows"},
		{Date: "Q4", Title: "Scale", Desc: "Standardize the operating model"},
	}
}

func themedFallbackCompare(deck plannedPPTXDeckPlan, heading string) (pptxCompareColumn, pptxCompareColumn) {
	subject := shortSubject(deck.Subject, heading)
	return pptxCompareColumn{
			Heading: "Current State",
			Points: []string{
				"Fragmented ownership around " + subject,
				"Manual handoffs and slow decisions",
				"Limited visibility into outcomes",
			},
		}, pptxCompareColumn{
			Heading: "Target State",
			Points: []string{
				"Clear ownership and standard workflow",
				"Faster execution with fewer manual steps",
				"Measured outcomes and stronger control",
			},
		}
}

func themedFallbackTable(deck plannedPPTXDeckPlan, heading string) *pptxTableData {
	subject := shortSubject(deck.Subject, heading)
	return &pptxTableData{
		Headers: []string{"Option", "Speed", "Control", "Fit"},
		Rows: [][]string{
			{"Manual " + subject, "Slow", "Low", "High effort"},
			{"Hybrid " + subject, "Medium", "Medium", "Moderate effort"},
			{"Automated " + subject, "Fast", "High", "Best long-term fit"},
		},
	}
}

func sectionTransitionCopy(deck plannedPPTXDeckPlan, heading string) string {
	return "Transition into " + firstNonEmpty(heading, deck.Subject, "the next section") + " with a cleaner story break."
}

func themedChartType(deck plannedPPTXDeckPlan) string {
	switch deck.DominantNeed {
	case "growth":
		return "bar"
	case "impact":
		return "line"
	default:
		return "column"
	}
}

func themedYAxisLabel(deck plannedPPTXDeckPlan) string {
	switch deck.DominantNeed {
	case "growth":
		return "Performance Index"
	case "governance":
		return "Control Coverage"
	case "impact":
		return "Impact Score"
	default:
		return "Value"
	}
}

func shortSubject(values ...string) string {
	for _, value := range values {
		value = trimTitlePhrase(value)
		if value != "" {
			return value
		}
	}
	return "the initiative"
}

func trimTitlePhrase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, sep := range []string{":", " - ", " | "} {
		if idx := strings.Index(value, sep); idx > 0 {
			value = value[:idx]
			break
		}
	}
	words := significantWords(value)
	if len(words) > 0 {
		return strings.Join(words[:minInt(3, len(words))], " ")
	}
	return value
}

func significantWords(value string) []string {
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	stop := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "from": {}, "into": {}, "about": {},
		"this": {}, "that": {}, "your": {}, "our": {}, "their": {}, "presentation": {},
		"deck": {}, "slide": {}, "slides": {}, "plan": {}, "strategy": {},
	}
	var out []string
	for _, part := range parts {
		if len(part) <= 2 {
			continue
		}
		if _, ok := stop[part]; ok {
			continue
		}
		out = append(out, strings.Title(part))
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func statsVisualReady(stats []pptxStat) bool {
	for _, stat := range stats {
		if strings.TrimSpace(stat.Value) == "" || strings.TrimSpace(stat.Label) == "" {
			return false
		}
	}
	return len(stats) > 0
}

func cardsVisualReady(cards []pptxCard) bool {
	if len(cards) == 0 {
		return false
	}
	for _, card := range cards {
		if strings.TrimSpace(card.Title) == "" || strings.TrimSpace(card.Icon) == "" {
			return false
		}
	}
	return true
}

func chartLayoutReady(categories []string, series []pptxChartSeries) bool {
	if len(categories) < 2 || len(series) == 0 {
		return false
	}
	for _, s := range series {
		if len(s.Values) < len(categories) {
			return false
		}
	}
	return true
}

func timelineVisualReady(items []pptxTimelineItem) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Date) == "" || strings.TrimSpace(item.Title) == "" {
			return false
		}
	}
	return len(items) > 0
}

func compareLayoutReady(left, right *pptxCompareColumn) bool {
	if left == nil || right == nil {
		return false
	}
	return len(safePoints(left.Points, 5)) >= 2 && len(safePoints(right.Points, 5)) >= 2
}

func tableLayoutReady(table *pptxTableData) bool {
	if table == nil || len(table.Headers) < 2 || len(table.Rows) == 0 {
		return false
	}
	for _, row := range table.Rows {
		if len(row) == 0 {
			return false
		}
	}
	return true
}
