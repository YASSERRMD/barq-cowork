package tools

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func PreviewOfficeArtifact(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pptx":
		return previewPPTX(path)
	case ".docx":
		return previewDOCX(path)
	default:
		return "", fmt.Errorf("preview not available for %s", filepath.Ext(path))
	}
}

type previewWordDocument struct {
	XMLName xml.Name        `xml:"document"`
	Body    previewWordBody `xml:"body"`
}

type previewWordBody struct {
	Paragraphs []previewWordParagraph `xml:"p"`
	Tables     []previewWordTable     `xml:"tbl"`
}

type previewWordParagraph struct {
	Properties previewWordParagraphProperties `xml:"pPr"`
	Runs       []previewWordRun               `xml:"r"`
}

type previewWordParagraphProperties struct {
	Style *previewWordStyle `xml:"pStyle"`
}

type previewWordStyle struct {
	Val string `xml:"val,attr"`
}

type previewWordRun struct {
	Texts []string `xml:"t"`
}

type previewWordTable struct {
	Rows []previewWordRow `xml:"tr"`
}

type previewWordRow struct {
	Cells []previewWordCell `xml:"tc"`
}

type previewWordCell struct {
	Paragraphs []previewWordParagraph `xml:"p"`
}

func previewDOCX(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	documentXML, err := officeZipRead(data, "word/document.xml")
	if err != nil {
		return "", err
	}

	var doc previewWordDocument
	if err := xml.Unmarshal(documentXML, &doc); err != nil {
		return "", fmt.Errorf("parse docx XML: %w", err)
	}

	var sections strings.Builder
	for _, p := range doc.Body.Paragraphs {
		text := strings.TrimSpace(p.text())
		if text == "" {
			continue
		}

		style := strings.ToLower(strings.TrimSpace(p.style()))
		switch style {
		case "title":
			sections.WriteString(`<h1 style="color:#4F46E5;font-size:32px;font-weight:700;margin:24px 0 12px;letter-spacing:-0.02em">` + html.EscapeString(text) + `</h1>`)
		case "subtitle":
			sections.WriteString(`<p style="color:#64748B;font-size:17px;font-style:italic;margin:0 0 18px">` + html.EscapeString(text) + `</p>`)
		case "heading1":
			sections.WriteString(`<h2 style="color:#312E81;font-size:22px;font-weight:700;margin:24px 0 10px;padding-bottom:8px;border-bottom:2px solid #C7D2FE">` + html.EscapeString(text) + `</h2>`)
		case "heading2":
			sections.WriteString(`<h3 style="color:#1E293B;font-size:18px;font-weight:700;margin:20px 0 8px">` + html.EscapeString(text) + `</h3>`)
		case "listparagraph":
			trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "▸"), "•"))
			sections.WriteString(`<div style="display:flex;gap:10px;margin:6px 0 6px 14px;color:#475569;font-size:14px;line-height:1.6"><span style="color:#6366F1;font-weight:700">▸</span><span>` + html.EscapeString(trimmed) + `</span></div>`)
		case "meta":
			sections.WriteString(`<p style="color:#64748B;font-size:13px;margin:0 0 6px">` + html.EscapeString(text) + `</p>`)
		default:
			sections.WriteString(`<p style="color:#334155;font-size:14px;line-height:1.7;margin:8px 0">` + html.EscapeString(text) + `</p>`)
		}
	}

	for _, table := range doc.Body.Tables {
		sections.WriteString(previewWordTableHTML(table))
	}

	return officePreviewShell(sections.String(), "#F8FAFC", "#0F172A"), nil
}

func previewWordTableHTML(table previewWordTable) string {
	var rows strings.Builder
	for rowIndex, row := range table.Rows {
		tag := "td"
		if rowIndex == 0 {
			tag = "th"
		}
		rows.WriteString("<tr>")
		for _, cell := range row.Cells {
			text := html.EscapeString(strings.TrimSpace(cell.text()))
			if tag == "th" {
				rows.WriteString(`<th style="padding:10px 12px;background:#4F46E5;color:#FFFFFF;font-size:13px;font-weight:700;border:1px solid #C7D2FE">` + text + `</th>`)
			} else {
				rows.WriteString(`<td style="padding:10px 12px;background:#FFFFFF;color:#334155;font-size:13px;border:1px solid #E2E8F0">` + text + `</td>`)
			}
		}
		rows.WriteString("</tr>")
	}

	return `<table style="width:100%;border-collapse:collapse;margin:18px 0 22px">` + rows.String() + `</table>`
}

func previewPPTX(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if manifest, ok, err := loadPPTXPreviewManifest(data); err != nil {
		return "", err
	} else if ok {
		return renderPPTXPreviewManifest(manifest), nil
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pptx zip: %w", err)
	}

	type slideFile struct {
		Name  string
		Index int
	}
	var slides []slideFile
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "ppt/slides/slide") && strings.HasSuffix(file.Name, ".xml") {
			base := strings.TrimSuffix(filepath.Base(file.Name), ".xml")
			num := strings.TrimPrefix(base, "slide")
			index, _ := strconv.Atoi(num)
			slides = append(slides, slideFile{Name: file.Name, Index: index})
		}
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].Index < slides[j].Index })

	var cards strings.Builder
	for i, slide := range slides {
		xmlBytes, err := officeZipRead(data, slide.Name)
		if err != nil {
			return "", err
		}
		lines := extractXMLText(xmlBytes, "t")
		if len(lines) == 0 {
			lines = []string{"Empty slide"}
		}

		var content strings.Builder
		for j, line := range dedupeStrings(lines) {
			if line == "" {
				continue
			}
			switch {
			case j == 0:
				content.WriteString(`<h2 style="color:#FFFFFF;font-size:24px;font-weight:700;margin:0 0 10px">` + html.EscapeString(line) + `</h2>`)
			case looksLikeMetric(line):
				content.WriteString(`<div style="display:inline-block;margin:6px 10px 6px 0;padding:8px 12px;border-radius:999px;background:rgba(99,102,241,0.18);color:#C7D2FE;font-size:13px;font-weight:700">` + html.EscapeString(line) + `</div>`)
			default:
				content.WriteString(`<p style="color:#CBD5E1;font-size:14px;line-height:1.6;margin:6px 0">` + html.EscapeString(line) + `</p>`)
			}
		}

		cards.WriteString(`<section style="background:#0F172A;border:1px solid rgba(148,163,184,0.18);border-radius:18px;padding:24px 28px;margin:0 0 16px;box-shadow:0 20px 45px rgba(15,23,42,0.22)">
  <div style="display:flex;align-items:center;gap:10px;margin:0 0 16px">
    <span style="display:inline-flex;align-items:center;justify-content:center;min-width:30px;height:30px;padding:0 10px;border-radius:999px;background:#4F46E5;color:#FFFFFF;font-size:12px;font-weight:700">` + strconv.Itoa(i+1) + `</span>
    <span style="color:#94A3B8;font-size:12px;letter-spacing:0.08em;text-transform:uppercase">Slide ` + strconv.Itoa(i+1) + ` of ` + strconv.Itoa(len(slides)) + `</span>
  </div>` + content.String() + `</section>`)
	}

	return officePreviewShell(cards.String(), "#020617", "#F8FAFC"), nil
}

func officePreviewShell(content, background, text string) string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Preview</title></head><body style="margin:0;background:` + background + `;color:` + text + `;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif"><main style="max-width:980px;margin:0 auto;padding:28px 20px 40px">` + content + `</main></body></html>`
}

func officeZipRead(data []byte, target string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		if file.Name != target {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return readAll(rc)
	}
	return nil, fmt.Errorf("zip entry not found: %s", target)
}

func extractXMLText(data []byte, localName string) []string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var values []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != localName {
			continue
		}
		var text string
		if err := dec.DecodeElement(&text, &start); err == nil {
			text = strings.TrimSpace(text)
			if text != "" {
				values = append(values, text)
			}
		}
	}
	return values
}

func dedupeStrings(values []string) []string {
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

func looksLikeMetric(value string) bool {
	return strings.ContainsAny(value, "%$0123456789")
}

func (p previewWordParagraph) text() string {
	var parts []string
	for _, run := range p.Runs {
		for _, text := range run.Texts {
			text = strings.TrimSpace(text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "")
}

func (p previewWordParagraph) style() string {
	if p.Properties.Style == nil {
		return ""
	}
	return p.Properties.Style.Val
}

func (c previewWordCell) text() string {
	var parts []string
	for _, p := range c.Paragraphs {
		if text := strings.TrimSpace(p.text()); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func readAll(rc interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
