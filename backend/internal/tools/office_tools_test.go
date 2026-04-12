package tools_test

import (
	"archive/zip"
	"bytes"
	"context"
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

	html, err := tools.PreviewOfficeArtifact(pptxPath)
	if err != nil {
		t.Fatalf("preview pptx: %v", err)
	}
	if !strings.Contains(html, "Impact snapshot") || !strings.Contains(html, "Implementation roadmap") {
		t.Fatalf("pptx preview missing expected headings: %s", html)
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
