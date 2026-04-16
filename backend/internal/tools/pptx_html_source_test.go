package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

// The preview renderer now composes every slide from structured fields and
// parseComposition() — it intentionally ignores LLM-authored HTML so the
// preview matches the final .pptx geometry. These tests assert the Reveal.js
// shell plus a few layout branches.

func TestBuildPPTXPreviewManifest_EmitsRevealDocumentFromStructuredFields(t *testing.T) {
	planned := planPPTXPresentation(
		"AI in Healthcare Operational Rollout",
		"Operating plan for enterprise-scale clinical deployment",
		[]pptxSlide{
			{
				Heading: "Operational imperative",
				Type:    "bullets",
				Points: []string{
					"Governance: clear ownership for each deployment gate.",
					"Workflow: integration into existing clinical routines.",
					"Measurement: KPIs that leaders can trust.",
				},
				Design: &pptxSlideDesign{
					LayoutStyle: "split wide",
					PanelStyle:  "filled",
					AccentMode:  "band",
					Density:     "balanced",
				},
				// Intentionally supply LLM HTML that should be ignored.
				HTML: `<div class="legacy-llm-html">this must not appear in the preview</div>`,
			},
		},
		pptxDeckDesignInput{
			Subject:    "AI in Healthcare Operational Rollout",
			Audience:   "healthcare executives",
			Narrative:  "Imperative -> capability -> roadmap -> decision",
			Theme:      "healthcare",
			CoverStyle: "editorial",
			ColorStory: "cool clinical depth",
			Motif:      "health",
			Kicker:     "From pilot to production",
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

	manifestBytes, err := buildPPTXPreviewManifest(
		"AI in Healthcare Operational Rollout",
		"Operating plan for enterprise-scale clinical deployment",
		planned,
	)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	var manifest pptxPreviewManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	doc := manifest.HTMLDocument
	if doc == "" {
		t.Fatalf("expected html_document to be populated")
	}

	// Reveal.js shell
	for _, want := range []string{
		`<div class="reveal">`,
		`<div class="slides">`,
		`reveal.js@5`,
		`new Reveal(`,
		`width: 1280`,
		`height: 720`,
		`margin: 0`,
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected reveal shell token %q, got %s", want, doc)
		}
	}

	// Structured content from manifest fields, not the LLM HTML payload
	for _, want := range []string{
		`From pilot to production`,
		`AI in Healthcare Operational Rollout`,
		`Governance`,
		`Workflow`,
		`Measurement`,
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected structured content %q in preview, got %s", want, doc)
		}
	}

	// LLM HTML must never be rendered
	if strings.Contains(doc, "legacy-llm-html") || strings.Contains(doc, "this must not appear") {
		t.Fatalf("expected LLM-authored HTML to be ignored in preview, got %s", doc)
	}

	// Composition-driven classes for a split+band bullets slide
	for _, want := range []string{
		`class="barq-slide barq-accent-band"`,
		`class="barq-split"`,
		`barq-panel-filled`,
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected composition class %q for split wide + band, got %s", want, doc)
		}
	}

	// Palette variables wired from manifest.palette
	for _, want := range []string{
		`--bg: #0F172A`,
		`--accent: #14B8A6`,
		`--text: #F8FAFC`,
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected palette variable %q, got %s", want, doc)
		}
	}
}

func TestRenderPPTXPreviewManifest_PrefersEmbeddedHTMLDocument(t *testing.T) {
	input := `<!DOCTYPE html><html><body><main>custom preview</main></body></html>`
	got := renderPPTXPreviewManifest(pptxPreviewManifest{HTMLDocument: input})
	if got != input {
		t.Fatalf("expected preview to return embedded html document")
	}
}

func TestValidatePlannedHTMLDeckSource_AcceptsStructuredFallbackContent(t *testing.T) {
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
			Subject:    "Operational rollout",
			Audience:   "operations leaders",
			Narrative:  "Imperative -> roadmap -> decision",
			Theme:      "healthcare",
			CoverStyle: "editorial",
			ColorStory: "cool clinical depth",
			Motif:      "health",
			Kicker:     "Operational briefing",
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

	if err := validatePlannedHTMLDeckSource(planned); err != nil {
		t.Fatalf("expected structured fallback to pass html deck validation: %v", err)
	}
}

func TestSanitizeHTMLMarkup_RemovesVisibleEmojiGlyphs(t *testing.T) {
	got := sanitizeHTMLMarkup(`<div><h2>Kids and AI 🤖</h2><p>Use real Bootstrap icons ✅, not emoji.</p></div>`)
	for _, blocked := range []string{"🤖", "✅"} {
		if strings.Contains(got, blocked) {
			t.Fatalf("expected emoji %q to be stripped, got %s", blocked, got)
		}
	}
	if !strings.Contains(got, "Kids and AI") || !strings.Contains(got, "Bootstrap icons") {
		t.Fatalf("expected non-emoji text to remain, got %s", got)
	}
}

func TestSanitizeHTMLMarkup_NormalizesOversizedInlineStyles(t *testing.T) {
	got := sanitizeHTMLMarkup(`<div style="padding: 96px 104px 88px; gap: 36px; font-family: Inter, system-ui, sans-serif; font-size: 84px; line-height: 1.8">Slide</div>`)
	for _, want := range []string{
		`padding: 78px 78px 78px`,
		`gap: 24px`,
		`font-family: "Helvetica Neue", Arial, sans-serif`,
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
		`font-family: "Helvetica Neue", Arial, sans-serif`,
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

func TestHTMLSlideContentReady_AcceptsBootstrapCardsAndIcons(t *testing.T) {
	raw := `<div class="container-fluid h-100">
  <div class="row g-3 align-items-stretch">
    <div class="col-6">
      <div class="card h-100">
        <div class="card-body">
          <span class="icon-badge"><i class="bi bi-shield-check" aria-hidden="true"></i></span>
          <h2 class="display-5">Governed rollout</h2>
          <p class="card-text">Operational ownership, measurement, and escalation are explicit before scale-up.</p>
        </div>
      </div>
    </div>
    <div class="col-6">
      <ul class="list-group">
        <li class="list-group-item">Clinical workflow owners approve each deployment gate.</li>
        <li class="list-group-item">KPI dashboards track adoption, risk, quality, and ROI.</li>
        <li class="list-group-item">Support model moves from pilot team to production runbook.</li>
      </ul>
    </div>
  </div>
</div>`

	if !htmlSlideContentReady(raw) {
		t.Fatalf("expected bootstrap-authored slide to pass content readiness")
	}
	if got := htmlInformationBlockCount(raw); got < 6 {
		t.Fatalf("expected bootstrap cards, list items, and icons to count as dense content, got %d", got)
	}
}

func TestParseComposition_PortsTSSemantics(t *testing.T) {
	cases := []struct {
		name string
		in   pptxSlideDesign
		want compositionParams
	}{
		{
			name: "split wide + band",
			in:   pptxSlideDesign{LayoutStyle: "split wide", AccentMode: "band"},
			want: compositionParams{SplitRatio: 0.52, Columns: 2, AccentBand: true, PanelGlass: true, Density: 1.0},
		},
		{
			// NOTE: "narrow" contains the substring "row", which in the TS
			// port matches the horizontal regex. We mirror that exactly.
			name: "split narrow + chip + airy",
			in:   pptxSlideDesign{LayoutStyle: "split narrow", AccentMode: "chip", Density: "airy"},
			want: compositionParams{SplitRatio: 0.32, Columns: 2, Horizontal: true, AccentChip: true, PanelGlass: true, Density: 0.75},
		},
		{
			name: "three-col + glow + dense",
			in:   pptxSlideDesign{LayoutStyle: "3-col", AccentMode: "glow", Density: "dense"},
			want: compositionParams{SplitRatio: 0, Columns: 3, AccentGlow: true, PanelGlass: true, Density: 1.28},
		},
		{
			name: "hero banner + filled panel",
			in:   pptxSlideDesign{LayoutStyle: "hero banner", PanelStyle: "filled"},
			want: compositionParams{SplitRatio: 0, Columns: 1, HeroFirst: true, PanelFilled: true, AccentRail: true, Density: 1.0},
		},
		{
			name: "horizontal rail + outline",
			in:   pptxSlideDesign{LayoutStyle: "horizontal rail", PanelStyle: "outline"},
			want: compositionParams{SplitRatio: 0, Columns: 1, Horizontal: true, PanelOutline: true, AccentRail: true, Density: 1.0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.in
			got := parseComposition(&d)
			if got != tc.want {
				t.Fatalf("parseComposition mismatch\n  want %+v\n  got  %+v", tc.want, got)
			}
		})
	}
}
