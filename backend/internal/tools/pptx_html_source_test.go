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
				HTML:    `<div style="padding:96px"><h2 class="section-title">Operational imperative</h2><p class="body-copy">Scale requires governance, workflow fit, and measurable outcomes.</p></div>`,
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
			ThemeCSS:    `.cover-grid{display:grid;grid-template-columns:1.2fr 420px;gap:42px}.cover-stack{display:grid;gap:24px}.panel{padding:24px;border:1px solid var(--border)}.tag{display:inline-flex;padding:8px 12px}`,
			CoverHTML:   `<div class="cover-grid" style="padding:96px"><div><div class="eyebrow">FROM PILOT TO PRODUCTION</div><h1 class="display-title">AI in Healthcare Operational Rollout</h1></div><div class="panel"></div></div>`,
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
