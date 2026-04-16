package tools_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/tools"
)

func TestWritePPTXTool_CreatesHTMLDeckAndPreview(t *testing.T) {
	if !htmlPPTBrowserAvailable() {
		t.Skip("chrome/chromium not available for html-to-pptx export")
	}

	ictx, dir := newTestCtx(t)
	result := tools.WritePPTXTool{}.Execute(context.Background(), ictx, `{
		"filename":"subject-driven-html-deck",
		"title":"AI in Healthcare: Operational Rollout",
		"subtitle":"Executive presentation",
		"deck":{
			"subject":"AI in Healthcare: Operational Rollout",
			"audience":"operations leaders",
			"narrative":"Pressure points -> measured impact -> rollout decision",
			"theme":"healthcare",
			"visual_style":"editorial clinical",
			"cover_style":"editorial",
			"color_story":"cool clinical tones with calm contrast",
			"motif":"health",
			"kicker":"Operational care briefing",
			"theme_css":".cover-grid{display:grid;grid-template-columns:1.1fr 420px;gap:42px;align-items:end}.summary-grid{display:grid;grid-template-columns:1.05fr 0.95fr;gap:24px}.roadmap-list{display:grid;gap:18px}.roadmap-row{display:grid;grid-template-columns:180px minmax(0,1fr) 84px;gap:20px;align-items:start;padding:24px 26px;background:rgba(255,255,255,0.96);color:#0f172a;border-left:8px solid var(--accent)}.matrix-table td strong{display:block;margin-bottom:8px}",
			"cover_html":"<div class='cover-grid' style='position:absolute;inset:96px'><div style='display:grid;gap:24px;align-content:end'><div class='eyebrow'>OPERATIONAL CARE BRIEFING</div><div class='rule'></div><h1 class='display-title'>AI in Healthcare: Operational Rollout</h1><p class='lede'>Enterprise operating plan for safe, measurable clinical scale-up.</p><div class='tag-row'><div class='tag'>Clinical operations</div><div class='tag'>Governance</div><div class='tag'>Measured adoption</div></div></div><div style='display:grid;gap:18px'><div class='panel'><div class='eyebrow'>01</div><p class='body-copy'>Imperative and pressure points</p></div><div class='panel'><div class='eyebrow'>02</div><p class='body-copy'>Capabilities and controls</p></div><div class='panel'><div class='eyebrow'>03</div><p class='body-copy'>Roadmap and decision gate</p></div></div></div>",
			"palette":{
				"background":"F5FAFE",
				"card":"FFFFFF",
				"accent":"0EA5E9",
				"accent2":"67E8F9",
				"text":"0F172A",
				"muted":"64748B",
				"border":"D6EAF4"
			}
		},
		"slides":[
			{"heading":"Operational imperative","type":"html","html":"<div style='padding:94px 96px 80px;display:grid;gap:24px'><div class='eyebrow'>EXECUTIVE SUMMARY</div><div class='summary-grid'><div style='display:grid;gap:20px;align-content:start'><h2 class='section-title'>Operational imperative</h2><p class='body-copy muted-copy'>Clinical AI programs stall when pilots, governance, and workflow change are funded separately instead of as one operating model.</p><ul class='bullet-list'><li class='bullet-item'><strong>Fragmented pilots</strong><span class='body-copy'>Department wins never become enterprise capability when rollout governance arrives late.</span></li><li class='bullet-item'><strong>Invisible risk</strong><span class='body-copy'>Executives lack one view of safety, adoption, and ROI signals across active workflows.</span></li><li class='bullet-item'><strong>Decision lag</strong><span class='body-copy'>Manual triage and approval loops slow deployment even when technical models are ready.</span></li></ul></div><div class='grid-2'><div class='stat-card'><div class='stat-value'>90d</div><div class='stat-label'>Pilot window</div><div class='stat-desc'>Enough time to prove workflow fit and governance discipline.</div></div><div class='stat-card'><div class='stat-value'>3</div><div class='stat-label'>Core disciplines</div><div class='stat-desc'>Governance, workflow design, and measurement move together.</div></div></div></div></div>"},
			{"heading":"Capability pillars","type":"html","html":"<div style='padding:94px 96px 80px;display:grid;gap:24px'><div class='eyebrow'>CAPABILITY SYSTEM</div><h2 class='section-title'>Capability pillars</h2><div class='grid-3'><div class='panel-light'><h3 style='font-size:30px;margin:0 0 10px'>Governance spine</h3><p class='body-copy muted-copy'>Clinical review, model-risk policy, and deployment sign-off are sequenced before scale.</p></div><div class='panel-light'><h3 style='font-size:30px;margin:0 0 10px'>Workflow design</h3><p class='body-copy muted-copy'>Every use case is embedded into existing care operations, escalation rules, and staff training.</p></div><div class='panel-light'><h3 style='font-size:30px;margin:0 0 10px'>Measurement layer</h3><p class='body-copy muted-copy'>Adoption, safety, throughput, and ROI metrics are reviewed through one operating dashboard.</p></div></div></div>"},
			{"heading":"Implementation roadmap","type":"html","html":"<div style='padding:94px 96px 80px;display:grid;gap:24px'><div class='eyebrow'>ROADMAP</div><h2 class='section-title'>Implementation roadmap</h2><div class='roadmap-list'><div class='roadmap-row'><div class='timeline-date'>Q1 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Foundation and pilot</h3><p class='body-copy muted-copy'>Data readiness audit, governance charter, and first clinical workflow launch.</p></div><div class='tag'>01</div></div><div class='roadmap-row'><div class='timeline-date'>Q2 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Scale and validate</h3><p class='body-copy muted-copy'>Expand to priority departments, establish KPI reviews, and tighten model oversight.</p></div><div class='tag'>02</div></div><div class='roadmap-row'><div class='timeline-date'>Q4 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Enterprise rollout</h3><p class='body-copy muted-copy'>Embed into system workflows, leadership reporting, and budget planning.</p></div><div class='tag'>03</div></div></div></div>"},
			{"heading":"Decision matrix","type":"html","html":"<div style='padding:94px 96px 80px;display:grid;gap:24px'><div class='eyebrow'>DECISION FRAME</div><h2 class='section-title'>Decision matrix</h2><table class='matrix-table'><thead><tr><th>Decision area</th><th>Immediate action</th><th>Why it matters</th></tr></thead><tbody><tr><td><strong>Governance</strong>Launch one cross-functional review board.</td><td>Approve pilot criteria, escalation paths, and audit ownership.</td><td>Prevents fragmented approvals and inconsistent safety thresholds.</td></tr><tr><td><strong>Deployment</strong>Choose two high-volume workflows for scale-up.</td><td>Sequence rollout where operational savings and clinical benefit are measurable.</td><td>Creates a credible enterprise case before broader investment.</td></tr><tr><td><strong>Measurement</strong>Adopt one operating scorecard.</td><td>Track adoption, quality, throughput, and ROI in one forum.</td><td>Lets executives expand based on evidence instead of pilot anecdotes.</td></tr></tbody></table></div>"}
		]
	}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("write_pptx failed: %s", result.Error)
	}

	pptxPath := filepath.Join(dir, "slides", "subject-driven-html-deck.pptx")
	data, err := os.ReadFile(pptxPath)
	if err != nil {
		t.Fatalf("read pptx: %v", err)
	}

	entries := unzipEntryNames(t, data)
	slideCount := 0
	for _, name := range entries {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			slideCount++
		}
	}
	if slideCount != 5 {
		t.Fatalf("expected 5 slide xml files including cover, got %d", slideCount)
	}
	if !contains(entries, "customXml/barq-presentation.json") {
		t.Fatalf("pptx missing preview manifest: %v", entries)
	}
	manifestBytes := unzipEntryData(t, data, "customXml/barq-presentation.json")
	var manifest struct {
		Theme        string `json:"theme"`
		HTMLDocument string `json:"html_document"`
		Palette      struct {
			Background string `json:"background"`
			Accent     string `json:"accent"`
		} `json:"palette"`
		DeckPlan struct {
			Subject         string `json:"subject"`
			Audience        string `json:"audience"`
			VisualDirection string `json:"visual_direction"`
			CoverStyle      string `json:"cover_style"`
			Motif           string `json:"motif"`
			Design          struct {
				Composition string `json:"composition"`
				AccentMode  string `json:"accent_mode"`
			} `json:"design"`
		} `json:"deck_plan"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Theme != "healthcare" {
		t.Fatalf("expected healthcare manifest theme, got %q", manifest.Theme)
	}
	if manifest.DeckPlan.Subject == "" || manifest.DeckPlan.Audience == "" || manifest.DeckPlan.VisualDirection == "" {
		t.Fatalf("expected populated deck plan in manifest, got %+v", manifest.DeckPlan)
	}
	if manifest.DeckPlan.CoverStyle != "editorial" || manifest.DeckPlan.Motif != "health" {
		t.Fatalf("expected explicit deck design in manifest, got %+v", manifest.DeckPlan)
	}
	if manifest.DeckPlan.Design.Composition == "" || manifest.DeckPlan.Design.AccentMode == "" {
		t.Fatalf("expected render design metadata in manifest, got %+v", manifest.DeckPlan.Design)
	}
	if manifest.Palette.Background == "" || manifest.Palette.Accent == "" {
		t.Fatalf("expected palette in manifest, got %+v", manifest.Palette)
	}
	if !strings.Contains(manifest.HTMLDocument, `class="reveal"`) || !strings.Contains(strings.ToUpper(manifest.HTMLDocument), "IMPLEMENTATION ROADMAP") {
		t.Fatalf("expected embedded reveal.js html document in manifest, got %s", manifest.HTMLDocument)
	}
	coverXML := strings.ToUpper(string(unzipEntryData(t, data, "ppt/slides/slide1.xml")))
	if strings.Contains(coverXML, "AUDIENCE") ||
		strings.Contains(coverXML, "STORY STRUCTURE") ||
		strings.Contains(coverXML, "VISUAL DIRECTION") ||
		strings.Contains(coverXML, "PRIORITY") ||
		strings.Contains(coverXML, "BULLETS") ||
		strings.Contains(coverXML, "TIMELINE") {
		t.Fatalf("expected user-facing cover only, got %s", coverXML)
	}
	if !strings.Contains(coverXML, "AI IN HEALTHCARE") {
		t.Fatalf("expected cover title in slide XML, got %s", coverXML)
	}

	html, err := tools.PreviewOfficeArtifact(pptxPath)
	if err != nil {
		t.Fatalf("preview pptx: %v", err)
	}
	// Preview can be served as Reveal.js HTML (fast) OR as base64 PNG stack
	// (pixel-accurate via LibreOffice). Accept either, but require the slide
	// content is reachable from the payload.
	htmlUpper := strings.ToUpper(html)
	looksLikeReveal := strings.Contains(html, `class="reveal"`) || strings.Contains(html, `data:image/png;base64`)
	if !looksLikeReveal {
		t.Fatalf("pptx preview was neither Reveal HTML nor PNG stack: %.400s", html)
	}
	if strings.Contains(html, `class="reveal"`) {
		if !strings.Contains(htmlUpper, "OPERATIONAL IMPERATIVE") ||
			!strings.Contains(htmlUpper, "IMPLEMENTATION ROADMAP") ||
			!strings.Contains(htmlUpper, "CAPABILITY PILLARS") ||
			!strings.Contains(htmlUpper, "DECISION MATRIX") {
			t.Fatalf("reveal preview missing headings: %.400s", html)
		}
	}
	if strings.Contains(html, "Audience:") || strings.Contains(html, "AUDITED") {
		t.Fatalf("pptx preview leaked planning metadata: %s", html)
	}
}

func TestWritePPTXTool_CreatesHTMLAuthoredDeck(t *testing.T) {
	if !htmlPPTBrowserAvailable() {
		t.Skip("chrome/chromium not available for html-to-pptx export")
	}

	ictx, dir := newTestCtx(t)
	result := tools.WritePPTXTool{}.Execute(context.Background(), ictx, `{
		"filename":"html-authored-deck",
		"title":"AI in Healthcare Operational Rollout",
		"subtitle":"Operating plan for enterprise-scale clinical deployment",
		"deck":{
			"subject":"AI in Healthcare Operational Rollout",
			"audience":"healthcare executives",
			"narrative":"Imperative -> capabilities -> roadmap -> decision",
			"theme":"healthcare",
			"visual_style":"editorial proposal system",
			"cover_style":"editorial",
			"color_story":"cool clinical depth",
			"motif":"health",
			"kicker":"From pilot to production",
			"theme_css":".cover-grid{display:grid;grid-template-columns:1.25fr 420px;gap:42px;align-items:end}.chapter-card{padding:28px 30px;background:rgba(255,255,255,0.08);border-left:10px solid var(--accent)}.summary-strip{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.summary-chip{padding:14px 16px;border:1px solid rgba(255,255,255,0.1)}.rule{height:6px}.bullet-list{display:grid;gap:18px}.stat-card{padding:28px 30px;background:rgba(255,255,255,0.96)}",
			"cover_html":"<div class='cover-grid' style='position:absolute;inset:96px'><div style='display:grid;gap:24px;align-content:end'><div class='eyebrow'>FROM PILOT TO PRODUCTION</div><div class='rule'></div><h1 class='display-title'>AI in Healthcare Operational Rollout</h1><p class='lede'>Operating plan for enterprise-scale clinical deployment with one governance spine, one workflow model, and one scorecard.</p><div class='summary-strip'><div class='summary-chip'>Audience: healthcare executives</div><div class='summary-chip'>Scope: clinical operations</div><div class='summary-chip'>Goal: measured rollout</div></div></div><div style='display:grid;gap:18px'><div class='chapter-card'><strong>01</strong><div>Operational imperative</div></div><div class='chapter-card'><strong>02</strong><div>Capability pillars</div></div><div class='chapter-card'><strong>03</strong><div>Implementation roadmap</div></div></div></div>",
			"palette":{
				"background":"0F172A",
				"card":"172033",
				"accent":"14B8A6",
				"accent2":"60A5FA",
				"text":"F8FAFC",
				"muted":"CBD5E1",
				"border":"334155"
			}
		},
		"slides":[
			{"heading":"The operational imperative","type":"html","html":"<div style='padding:92px 96px 80px;display:grid;grid-template-columns:1.1fr 0.9fr;gap:28px'><div style='display:grid;gap:20px;align-content:start'><div class='eyebrow'>EXECUTIVE SUMMARY</div><h2 class='section-title'>The operational imperative</h2><p class='body-copy muted-copy'>Clinical AI programs fail at scale when governance, workflow design, and measurement remain disconnected.</p><ul class='bullet-list'><li class='bullet-item'><span class='body-copy'>Fragmented pilots create isolated wins instead of operational change.</span></li><li class='bullet-item'><span class='body-copy'>Governance is often bolted on after deployment instead of embedded in the operating model.</span></li><li class='bullet-item'><span class='body-copy'>Executives need measurable adoption, safety, and ROI signals before expansion.</span></li></ul></div><div style='display:grid;gap:18px'><div class='stat-card'><div class='stat-value'>3</div><div class='stat-label'>Critical disciplines</div><div class='stat-desc'>Governance, workflow fit, and measurement must move together.</div></div><div class='stat-card'><div class='stat-value'>90d</div><div class='stat-label'>Pilot window</div><div class='stat-desc'>Enough time to prove value without locking in weak operating habits.</div></div></div></div>"},
			{"heading":"Implementation roadmap","type":"html","html":"<div style='padding:92px 96px 80px;display:grid;gap:22px'><div class='eyebrow'>ROADMAP</div><h2 class='section-title'>Implementation roadmap</h2><div class='timeline-list'><div class='timeline-row'><div class='timeline-date'>Q1 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Foundation and pilot</h3><p class='body-copy muted-copy'>Data readiness, governance charter, and first clinical workflow launch.</p></div></div><div class='timeline-row'><div class='timeline-date'>Q2 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Scale and validate</h3><p class='body-copy muted-copy'>Expand to priority departments and operationalize KPI reviews.</p></div></div><div class='timeline-row'><div class='timeline-date'>Q4 2026</div><div><h3 style='font-size:30px;margin:0 0 8px'>Enterprise rollout</h3><p class='body-copy muted-copy'>Embed into system workflows, reporting, and leadership reviews.</p></div></div></div>"}
		]
	}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("write_pptx html deck failed: %s", result.Error)
	}

	pptxPath := filepath.Join(dir, "slides", "html-authored-deck.pptx")
	data, err := os.ReadFile(pptxPath)
	if err != nil {
		t.Fatalf("read pptx: %v", err)
	}

	manifestBytes := unzipEntryData(t, data, "customXml/barq-presentation.json")
	var manifest struct {
		HTMLDocument string `json:"html_document"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if !strings.Contains(manifest.HTMLDocument, `class="reveal"`) || !strings.Contains(manifest.HTMLDocument, "From pilot to production") {
		t.Fatalf("expected html-authored manifest, got %s", manifest.HTMLDocument)
	}

	coverXML := strings.ToUpper(string(unzipEntryData(t, data, "ppt/slides/slide1.xml")))
	if !strings.Contains(coverXML, "FROM PILOT TO PRODUCTION") || !strings.Contains(coverXML, "AI IN HEALTHCARE OPERATIONAL ROLLOUT") {
		t.Fatalf("expected html-authored cover text in pptx slide xml, got %s", coverXML)
	}
}

func TestWritePPTXTool_RejectsIncompleteDeckBrief(t *testing.T) {
	ictx, _ := newTestCtx(t)
	result := tools.WritePPTXTool{}.Execute(context.Background(), ictx, `{
		"filename":"missing-design-brief",
		"title":"Kids and AI",
		"slides":[
			{"heading":"Why it matters","type":"bullets","points":["Tools are everywhere","Families need guidance","Children need safe exploration"]}
		]
	}`)
	if result.Status == tools.ResultOK {
		t.Fatalf("expected incomplete deck brief to fail")
	}
	if !strings.Contains(result.Error, "deck design brief is required") {
		t.Fatalf("expected deck brief validation error, got %q", result.Error)
	}
}

func TestWritePPTXTool_AcceptsStructuredFallbackDeck(t *testing.T) {
	ictx, dir := newTestCtx(t)
	result := tools.WritePPTXTool{}.Execute(context.Background(), ictx, `{
		"filename":"structured-only-deck",
		"title":"Operational rollout",
		"deck":{
			"subject":"Operational rollout",
			"audience":"operations leaders",
			"narrative":"Imperative -> roadmap -> decision",
			"theme":"healthcare",
			"visual_style":"editorial clinical",
			"cover_style":"editorial",
			"color_story":"cool clinical tones",
			"motif":"health",
			"kicker":"Operational briefing",
			"theme_css":".deck-shell{display:grid;gap:20px}.summary-strip{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.summary-chip{padding:14px;border:1px solid var(--border)}.panel{padding:24px;border:1px solid var(--border)}.grid-2{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}.tag{display:inline-flex;padding:8px 12px}",
			"cover_html":"<div class='deck-shell' style='padding:96px'><div class='eyebrow'>OPERATIONAL BRIEFING</div><h1 class='display-title'>Operational rollout</h1><p class='lede'>Governance, workflow design, measurement ownership, and phased execution in one operating brief.</p><div class='panel'>Audience: operations leaders who need a clear decision path for rollout funding and accountability.</div><div class='summary-strip'><div class='summary-chip'>Scope: governance</div><div class='summary-chip'>Horizon: two quarters</div><div class='summary-chip'>Goal: ready-to-scale operations</div></div></div>",
			"palette":{
				"background":"F5FAFE",
				"card":"FFFFFF",
				"accent":"0EA5E9",
				"accent2":"67E8F9",
				"text":"0F172A",
				"muted":"64748B",
				"border":"D6EAF4"
			}
		},
		"slides":[
			{"heading":"Current pressure points","type":"bullets","points":["Fragmented data across teams","Manual triage slows response","Leaders lack real-time visibility"]}
		]
	}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("expected structured fallback deck to render, got %s", result.Error)
	}
	data, err := os.ReadFile(filepath.Join(dir, "slides", "structured-only-deck.pptx"))
	if err != nil {
		t.Fatalf("read pptx: %v", err)
	}
	manifestBytes := unzipEntryData(t, data, "customXml/barq-presentation.json")
	var manifest struct {
		HTMLDocument string `json:"html_document"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if !strings.Contains(manifest.HTMLDocument, "Current pressure points") || !strings.Contains(manifest.HTMLDocument, "Fragmented data across teams") {
		t.Fatalf("expected structured slide content in fallback html document, got %s", manifest.HTMLDocument)
	}
}

func htmlPPTBrowserAvailable() bool {
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium-browser",
		"/usr/bin/chromium",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

func TestWriteDocxTool_CreatesNativeDocx(t *testing.T) {
	ictx, dir := newTestCtx(t)
	result := tools.DocxTool{}.Execute(context.Background(), ictx, `{
		"filename":"ops-brief",
		"title":"Operations Brief",
		"subtitle":"Q2 review",
		"author":"YASSERRMD",
		"sections":[
			{"heading":"Executive Summary","level":1,"content":"Summary paragraph\n• First action\n• Second action"},
			{"heading":"Decision Matrix","level":2,"content":"Supporting notes","table":{"headers":["Option","Cost"],"rows":[["Manual","High"],["Automated","Low"]]}}
		]
	}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("write_docx failed: %s", result.Error)
	}

	docxPath := filepath.Join(dir, "documents", "ops-brief.docx")
	data, err := os.ReadFile(docxPath)
	if err != nil {
		t.Fatalf("read docx: %v", err)
	}

	entries := unzipEntryNames(t, data)
	if !contains(entries, "word/document.xml") || !contains(entries, "word/styles.xml") {
		t.Fatalf("docx missing core OOXML parts: %v", entries)
	}

	html, err := tools.PreviewOfficeArtifact(docxPath)
	if err != nil {
		t.Fatalf("preview docx: %v", err)
	}
	if !strings.Contains(html, "Operations Brief") || !strings.Contains(html, "EXECUTIVE SUMMARY") {
		t.Fatalf("docx preview missing expected text: %s", html)
	}
}

func unzipEntryNames(t *testing.T, data []byte) []string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	return names
}

func unzipEntryData(t *testing.T, data []byte, target string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	for _, file := range reader.File {
		if file.Name != target {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", target, err)
		}
		defer rc.Close()
		payload, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read zip entry %s: %v", target, err)
		}
		return payload
	}
	t.Fatalf("zip entry not found: %s", target)
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
