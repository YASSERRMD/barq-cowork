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

func TestWritePPTXTool_CreatesMixedDeck(t *testing.T) {
	ictx, dir := newTestCtx(t)
	result := tools.WritePPTXTool{}.Execute(context.Background(), ictx, `{
		"filename":"subject-driven-deck",
		"title":"AI in Healthcare: Operational Rollout",
		"subtitle":"Executive presentation",
		"deck":{
			"audience":"operations leaders",
			"theme":"healthcare",
			"visual_style":"editorial clinical",
			"cover_style":"editorial",
			"motif":"health",
			"kicker":"Operational care briefing"
		},
		"slides":[
			{"heading":"Current pressure points","type":"bullets","points":["Fragmented data across teams","Manual triage slows response","Leaders lack real-time visibility"]},
			{"heading":"Impact snapshot","type":"stats","stats":[{"value":"92%","label":"Retention","desc":"Clinician workflow adoption"},{"value":"3.2x","label":"ROI","desc":"Operational efficiency gain"}]},
			{"heading":"Adoption trend","type":"chart","chart_type":"column","chart_categories":["Q1","Q2","Q3","Q4"],"chart_series":[{"name":"Adoption","values":[18,34,51,72]}]},
			{"heading":"Implementation roadmap","type":"timeline","timeline":[{"date":"Q1","title":"Pilot","desc":"Initial deployment"},{"date":"Q2","title":"Expand","desc":"Add more clinics"},{"date":"Q3","title":"Standardize","desc":"Roll out best practices"}]},
			{"heading":"Legacy vs target","type":"compare","left_column":{"heading":"Legacy","points":["Manual handoffs","Limited context"]},"right_column":{"heading":"Target","points":["Automated routing","Unified view"]}},
			{"heading":"Capability matrix","type":"table","table":{"headers":["Capability","Today","Target"],"rows":[["Routing","Manual","Automated"],["Insights","Weekly","Real-time"]]}}
		]
	}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("write_pptx failed: %s", result.Error)
	}

	pptxPath := filepath.Join(dir, "slides", "subject-driven-deck.pptx")
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
	if slideCount != 7 {
		t.Fatalf("expected 7 slide xml files including cover, got %d", slideCount)
	}
	if !contains(entries, "customXml/barq-presentation.json") {
		t.Fatalf("pptx missing structured preview manifest: %v", entries)
	}
	manifestBytes := unzipEntryData(t, data, "customXml/barq-presentation.json")
	var manifest struct {
		Theme   string `json:"theme"`
		Palette struct {
			Background string `json:"background"`
			Accent     string `json:"accent"`
		} `json:"palette"`
		DeckPlan struct {
			Subject         string `json:"subject"`
			Audience        string `json:"audience"`
			VisualDirection string `json:"visual_direction"`
			CoverStyle      string `json:"cover_style"`
			Motif           string `json:"motif"`
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
	if manifest.Palette.Background == "" || manifest.Palette.Accent == "" {
		t.Fatalf("expected palette in manifest, got %+v", manifest.Palette)
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
	if !strings.Contains(html, "Impact snapshot") || !strings.Contains(html, "Implementation roadmap") {
		t.Fatalf("pptx preview missing expected headings: %s", html)
	}
	if strings.Contains(html, "Audience:") || strings.Contains(html, "AUDITED") {
		t.Fatalf("pptx preview leaked planning metadata: %s", html)
	}
	if !strings.Contains(html, `data-layout="chart"`) || !strings.Contains(html, `data-layout="timeline"`) || !strings.Contains(html, `barq-preview-card-icon`) {
		t.Fatalf("pptx preview did not render structured slide layouts: %s", html)
	}
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
