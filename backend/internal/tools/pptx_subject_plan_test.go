package tools

import "testing"

func TestPlanPPTXPresentation_BuildsDeckStrategyAndAuditsSlides(t *testing.T) {
	planned := planPPTXPresentation(
		"AI in Healthcare: Operational Rollout",
		"Executive steering update",
		[]pptxSlide{
			{Heading: "Current pressure points", Type: "bullets", Points: []string{"Fragmented data across clinics", "Manual triage slows response", "Leaders lack real-time visibility"}},
			{Heading: "Impact snapshot", Type: "stats", Stats: []pptxStat{{Value: "92%", Label: "Adoption", Desc: "Clinician usage"}, {Value: "3.2x", Label: "ROI", Desc: "Operational improvement"}}},
			{Heading: "Adoption trend", Type: "chart", ChartType: "column", ChartCategories: []string{"Q1", "Q2", "Q3", "Q4"}, ChartSeries: []pptxChartSeries{{Name: "Adoption", Values: []float64{18, 34, 51, 72}}}},
			{Heading: "Implementation roadmap", Type: "timeline", Timeline: []pptxTimelineItem{{Date: "Q1", Title: "Pilot", Desc: "Initial deployment"}, {Date: "Q2", Title: "Expand", Desc: "Add more clinics"}, {Date: "Q3", Title: "Standardize", Desc: "Roll out best practices"}}},
		},
		pptxDeckDesignInput{},
	)

	if planned.ThemeName != "healthcare" {
		t.Fatalf("expected healthcare theme, got %q", planned.ThemeName)
	}
	if planned.DeckPlan.Audience == "" || planned.DeckPlan.NarrativeArc == "" || planned.DeckPlan.VisualDirection == "" {
		t.Fatalf("expected populated deck plan, got %+v", planned.DeckPlan)
	}
	if len(planned.DeckPlan.LayoutMix) < 3 {
		t.Fatalf("expected mixed layouts, got %+v", planned.DeckPlan.LayoutMix)
	}

	for _, slide := range planned.Slides {
		if slide.Plan.Purpose == "" || slide.Plan.Visual == "" || slide.Plan.ContentSource == "" {
			t.Fatalf("expected populated slide plan, got %+v", slide.Plan)
		}
		if !slide.Audit.ContentFit || !slide.Audit.LayoutFit || !slide.Audit.VisualFit {
			t.Fatalf("expected passing audit, got %+v", slide.Audit)
		}
		if slide.Variant < 0 || slide.Variant > 2 {
			t.Fatalf("unexpected slide variant %d", slide.Variant)
		}
	}
}

func TestPlanPPTXPresentation_FillsSubjectAwareFallbacks(t *testing.T) {
	planned := planPPTXPresentation(
		"Supply Chain Visibility Platform",
		"Operations review",
		[]pptxSlide{
			{Heading: "Rollout path", Type: "steps"},
			{Heading: "Capability story", Type: "cards"},
			{Heading: "Decision matrix", Type: "table"},
		},
		pptxDeckDesignInput{},
	)

	if err := validatePPTXPresentation(planned); err != nil {
		t.Fatalf("expected valid planned deck, got %v", err)
	}
	if len(planned.Slides[0].Slide.Steps) < 3 {
		t.Fatalf("expected fallback steps, got %+v", planned.Slides[0].Slide.Steps)
	}
	if len(planned.Slides[1].Slide.Cards) < 3 {
		t.Fatalf("expected fallback cards, got %+v", planned.Slides[1].Slide.Cards)
	}
	if planned.Slides[2].Slide.Table == nil || len(planned.Slides[2].Slide.Table.Headers) < 2 {
		t.Fatalf("expected fallback table, got %+v", planned.Slides[2].Slide.Table)
	}
}

func TestPickThemeName_PrefersEducationForKidsAudience(t *testing.T) {
	theme := pickThemeName(
		"Generative AI: Amazing Creativity for Kids!",
		"Classroom introduction for children",
	)
	if theme != "education" {
		t.Fatalf("expected education theme, got %q", theme)
	}
}

func TestPlanPPTXPresentation_PrefersExplicitDeckDesign(t *testing.T) {
	planned := planPPTXPresentation(
		"Kids and AI",
		"Guide for families",
		[]pptxSlide{{Heading: "Why it matters", Type: "bullets", Points: []string{"Creativity tools are everywhere", "Families need practical guidance", "Children need safe exploration"}}},
		pptxDeckDesignInput{
			Subject:     "Kids and AI",
			Audience:    "parents and educators",
			Narrative:   "Context -> opportunities -> safeguards",
			Theme:       "education",
			VisualStyle: "playful classroom collage",
			CoverStyle:  "playful",
			ColorStory:  "bright classroom tones with soft contrast",
			Motif:       "learning",
			Kicker:      "A visual guide for curious learners",
			Palette: &pptxPaletteInput{
				Background: "FFF6E5",
				Card:       "FFFDF7",
				Accent:     "F59E0B",
				Accent2:    "FCD34D",
				Text:       "1F2937",
				Muted:      "6B7280",
				Border:     "E8D8B8",
			},
		},
	)

	if planned.ThemeName != "education" {
		t.Fatalf("expected explicit education theme, got %q", planned.ThemeName)
	}
	if planned.DeckPlan.CoverStyle != "playful" || planned.DeckPlan.Motif != "learning" {
		t.Fatalf("expected explicit deck design to be preserved, got %+v", planned.DeckPlan)
	}
	if planned.Palette.bg != "FFF6E5" || planned.Palette.accent != "F59E0B" {
		t.Fatalf("expected explicit palette override, got %+v", planned.Palette)
	}
}

func TestResolveDeckPalette_KeepsChosenVisualFamilyAccent(t *testing.T) {
	palette := resolveDeckPalette("education", pptxDeckDesignInput{
		VisualStyle: "bold studio poster",
		CoverStyle:  "poster",
		ColorStory:  "electric cobalt and rose",
	}, "creative students")

	if palette.accent != "4F46E5" || palette.accent2 != "A5B4FC" {
		t.Fatalf("expected studio-light palette accents, got %+v", palette)
	}
}

func TestInferCardIcon_NormalizesLegacyEmojiToSemanticToken(t *testing.T) {
	icon := inferCardIcon(pptxCard{Icon: "📊", Title: "Insights", Desc: "Data visibility"}, 0)
	if icon != "chart" {
		t.Fatalf("expected chart token, got %q", icon)
	}
}

func TestDeriveAudience_DetectsYoungLearners(t *testing.T) {
	audience := deriveAudience("Discover how computers can draw and create", "Generative AI for Kids")
	if audience != "young learners" {
		t.Fatalf("expected young learners, got %q", audience)
	}
}
