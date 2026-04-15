package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPPTXPreviewManifest_EmbedsHTMLDocument(t *testing.T) {
	planned := planPPTXPresentation(
		"AI in Healthcare Operational Rollout",
		"Operating plan for enterprise-scale clinical deployment",
		[]pptxSlide{
			{
				Heading: "Operational imperative",
				Type:    "html",
				HTML:    `<div style="padding:96px;display:grid;gap:22px"><div class="eyebrow">EXECUTIVE SUMMARY</div><h2 class="section-title">Operational imperative</h2><p class="body-copy">Scale requires governance, workflow fit, measurable outcomes, accountable ownership, and one operating rhythm that leaders can trust.</p><div class="grid-2"><div class="panel-light">Governance, workflow, and KPI ownership must move together.</div><div class="panel-light">The rollout succeeds when measurement is operational, not decorative.</div></div></div>`,
			},
		},
		pptxDeckDesignInput{
			Subject:     "AI in Healthcare Operational Rollout",
			Audience:    "healthcare executives",
			Narrative:   "Imperative -> capability -> roadmap -> decision",
			Theme:       "healthcare",
			VisualStyle: "editorial operating proposal",
			CoverStyle:  "editorial",
			ColorStory:  "cool clinical depth",
			Motif:       "health",
			Kicker:      "From pilot to production",
			ThemeCSS:    `.cover-grid{display:grid;grid-template-columns:1.2fr 420px;gap:42px}.cover-stack{display:grid;gap:24px}.summary-strip{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.summary-chip{padding:14px;border:1px solid var(--border)}.panel{padding:24px;border:1px solid var(--border)}.tag{display:inline-flex;padding:8px 12px}`,
			CoverHTML:   `<div class="cover-grid" style="padding:96px"><div class="cover-stack"><div class="eyebrow">FROM PILOT TO PRODUCTION</div><h1 class="display-title">AI in Healthcare Operational Rollout</h1><p class="lede">Enterprise operating plan for safe, measurable clinical scale-up across governance, workflow, and adoption.</p><div class="tag-row"><span class="tag">Clinical operations</span><span class="tag">Governance</span><span class="tag">Measurement</span></div></div><div style="display:grid;gap:16px"><div class="panel">Audience: healthcare executives and clinical operations leaders.</div><div class="summary-strip"><div class="summary-chip">Scope: rollout governance</div><div class="summary-chip">Horizon: 12 months</div><div class="summary-chip">Goal: measured scale-up</div></div></div></div>`,
			Palette: &pptxPaletteInput{
				Background: "0F172A",
				Card:       "172033",
				Accent:     "14B8A6",
				Accent2:    "60A5FA",
				Text:       "F8FAFC",
				Muted:      "CBD5E1",
				Border:     "334155",
			},
		},
	)

	manifestBytes, err := buildPPTXPreviewManifest("AI in Healthcare Operational Rollout", "Operating plan for enterprise-scale clinical deployment", planned)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	var manifest pptxPreviewManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if manifest.HTMLDocument == "" {
		t.Fatalf("expected html_document to be populated")
	}
	if !strings.Contains(manifest.HTMLDocument, "barq-pptx-slide") || !strings.Contains(manifest.HTMLDocument, "FROM PILOT TO PRODUCTION") {
		t.Fatalf("expected html document to contain slide markup, got %s", manifest.HTMLDocument)
	}
	if !strings.Contains(manifest.HTMLDocument, `class="cover-shell slide"`) {
		t.Fatalf("expected cover html to be wrapped in a shell, got %s", manifest.HTMLDocument)
	}
	if !strings.Contains(manifest.HTMLDocument, `class="slide-shell slide content-shell"`) {
		t.Fatalf("expected content html to be wrapped in a shell, got %s", manifest.HTMLDocument)
	}
}

func TestRenderPPTXPreviewManifest_PrefersEmbeddedHTMLDocument(t *testing.T) {
	input := `<!DOCTYPE html><html><body><main>custom preview</main></body></html>`
	got := renderPPTXPreviewManifest(pptxPreviewManifest{HTMLDocument: input})
	if got != input {
		t.Fatalf("expected preview to return embedded html document")
	}
}

func TestValidatePlannedHTMLDeckSource_RejectsStructuredFallback(t *testing.T) {
	planned := planPPTXPresentation(
		"Operational rollout",
		"",
		[]pptxSlide{
			{
				Heading: "Pressure points",
				Type:    "bullets",
				Points:  []string{"Fragmented pilots", "Weak governance", "No shared scorecard"},
			},
		},
		pptxDeckDesignInput{
			Subject:     "Operational rollout",
			Audience:    "operations leaders",
			Narrative:   "Imperative -> roadmap -> decision",
			Theme:       "healthcare",
			VisualStyle: "editorial clinical",
			CoverStyle:  "editorial",
			ColorStory:  "cool clinical depth",
			Motif:       "health",
			Kicker:      "Operational briefing",
			ThemeCSS:    `.deck-shell{display:grid;gap:20px}.panel{padding:24px;border:1px solid var(--border)}.grid-2{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}.tag{display:inline-flex;padding:8px 12px}`,
			CoverHTML:   `<div class="deck-shell" style="padding:96px"><div class="eyebrow">OPERATIONAL BRIEFING</div><h1 class="display-title">Operational rollout</h1><div class="panel">Governance, workflow design, and measurement</div></div>`,
			Palette: &pptxPaletteInput{
				Background: "F5FAFE",
				Card:       "FFFFFF",
				Accent:     "0EA5E9",
				Accent2:    "67E8F9",
				Text:       "0F172A",
				Muted:      "64748B",
				Border:     "D6EAF4",
			},
		},
	)

	if err := validatePlannedHTMLDeckSource(planned); err == nil {
		t.Fatalf("expected structured fallback to fail html deck validation")
	}
}

func TestSanitizeHTMLMarkup_NormalizesOversizedInlineStyles(t *testing.T) {
	got := sanitizeHTMLMarkup(`<div style="padding: 96px 104px 88px; gap: 36px; font-family: Inter, system-ui, sans-serif; font-size: 84px; line-height: 1.8">Slide</div>`)
	for _, want := range []string{
		`padding: 78px 78px 78px`,
		`gap: 24px`,
		`font-family: "Aptos", "Helvetica Neue", Arial, sans-serif`,
		`font-size: 68px`,
		`line-height: 1.45`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in sanitized html, got %s", want, got)
		}
	}
}

func TestSanitizeCSSMarkup_NormalizesThemeTypographyAndSpacing(t *testing.T) {
	got := sanitizeCSSMarkup(`.cover{padding:110px 96px 88px;gap:32px;font-family:"Avenir Next","SF Pro Display",sans-serif;font-size:72px}.body{margin-top:64px;line-height:1.7}`)
	for _, want := range []string{
		`padding: 78px 78px 78px`,
		`gap: 24px`,
		`font-family: "Aptos", "Helvetica Neue", Arial, sans-serif`,
		`font-size: 68px`,
		`margin-top: 42px`,
		`line-height: 1.45`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in sanitized css, got %s", want, got)
		}
	}
}

func TestHTMLCoverContentReady_AcceptsConciseCover(t *testing.T) {
	raw := `<div style="padding:80px;background-color:#F5F0E8"><h1 style="font-size:48px">Islamic Parenting</h1><p style="font-size:20px">Raising righteous children with faith, love, and wisdom.</p></div>`
	if !htmlCoverContentReady(raw) {
		t.Fatalf("expected concise but well-formed cover html to be accepted")
	}
}

func TestWrapHTMLSlideShell_DoesNotDoubleWrapExistingShell(t *testing.T) {
	raw := `<div class="slide-shell slide"><h2 class="section-title">Signal</h2></div>`
	got := wrapHTMLSlideShell(raw, false)
	if got != raw {
		t.Fatalf("expected existing shell to remain unchanged, got %s", got)
	}
}

func TestWrapHTMLSlideShell_AddsCoverComposeClassForGenericCovers(t *testing.T) {
	raw := `<div style="display:flex"><div>left</div><div>right</div></div>`
	got := wrapHTMLSlideShell(raw, true)
	if !strings.Contains(got, `cover-shell--compose`) {
		t.Fatalf("expected generic cover to receive compose shell, got %s", got)
	}
}

func TestPreferredHTMLCover_PreservesValidAuthoredCover(t *testing.T) {
	manifest := pptxPreviewManifest{
		Title: "Islamic Parenting",
		Theme: "education",
		DeckPlan: pptxPreviewDeckPlan{
			Subject:      "Islamic Parenting",
			Audience:     "parents",
			NarrativeArc: "trust -> guidance -> daily practice",
			ColorStory:   "warm earth tones",
			Kicker:       "Faith-centered guidance",
			CoverHTML:    `<div style="display:flex"><div><h1 class="display-title">Islamic Parenting</h1><p class="lede">Raise children with faith.</p></div><div><svg viewBox="0 0 24 24"></svg></div></div>`,
		},
		Palette: pptxPreviewPalette{
			Background: "F5F0E8",
			Card:       "FFFFFF",
			Accent:     "2D6A4F",
			Accent2:    "C9A84C",
			Text:       "1B1B1B",
			Muted:      "7A7266",
			Border:     "E0D8CC",
		},
	}

	got := preferredHTMLCover(manifest)
	if !strings.Contains(got, `<div style="display:flex">`) {
		t.Fatalf("expected valid authored cover to be preserved, got %s", got)
	}
	if strings.Contains(got, `cover-grid`) || strings.Contains(got, `Narrative`) {
		t.Fatalf("expected authored cover, not deterministic fallback, got %s", got)
	}
}

func TestPreferredHTMLCover_FallsBackOnlyForInvalidCover(t *testing.T) {
	manifest := pptxPreviewManifest{
		Title: "Islamic Parenting",
		Theme: "education",
		DeckPlan: pptxPreviewDeckPlan{
			Subject:      "Islamic Parenting",
			Audience:     "parents",
			NarrativeArc: "trust -> guidance -> daily practice",
			ColorStory:   "warm earth tones",
			Kicker:       "Faith-centered guidance",
			CoverHTML:    `<div><span>Thin</span></div>`,
		},
		Palette: pptxPreviewPalette{
			Background: "F5F0E8",
			Card:       "FFFFFF",
			Accent:     "2D6A4F",
			Accent2:    "C9A84C",
			Text:       "1B1B1B",
			Muted:      "7A7266",
			Border:     "E0D8CC",
		},
	}

	got := preferredHTMLCover(manifest)
	if !strings.Contains(got, `cover-grid`) || !strings.Contains(got, `Narrative`) {
		t.Fatalf("expected invalid cover to fall back to structured cover, got %s", got)
	}
}

func TestPreferredHTMLSlideMarkup_FallsBackForUnderstructuredSlide(t *testing.T) {
	slide := pptxPreviewSlide{
		Heading: "Operating signal",
		Layout:  "bullets",
		HTML:    `<div style="display:flex;flex-direction:column;gap:18px"><h2 class="section-title">Operating signal</h2><p class="body-copy">One paragraph and one heading are not enough for a client-facing slide that needs real density.</p></div>`,
		Points: []string{
			"Connect governance to workflow ownership.",
			"Treat scorecards as operating controls, not decoration.",
			"Close with one funding and accountability decision.",
		},
	}

	got := preferredHTMLSlideMarkup(slide)
	if strings.Contains(got, `display:flex`) {
		t.Fatalf("expected weak authored slide to fall back to structured markup, got %s", got)
	}
	if !strings.Contains(got, `bullet-list`) || !strings.Contains(got, `Operating signal`) {
		t.Fatalf("expected fallback bullet layout, got %s", got)
	}
}
