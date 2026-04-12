package tools

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"
)

type docxRunStyle struct {
	FontSizeHalfPts int
	Bold            bool
	Italic          bool
	Color           string
}

func buildDOCX(args docxArgs) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	entries := []struct {
		name    string
		content string
	}{
		{"[Content_Types].xml", docxContentTypesXML()},
		{"_rels/.rels", docxRootRelsXML()},
		{"docProps/app.xml", docxAppXML()},
		{"docProps/core.xml", docxCoreXML(args.Title)},
		{"word/document.xml", docxDocumentXML(args)},
		{"word/styles.xml", docxStylesXML()},
		{"word/_rels/document.xml.rels", docxDocumentRelsXML()},
	}

	for _, entry := range entries {
		w, err := zw.Create(entry.name)
		if err != nil {
			return nil, fmt.Errorf("zip create %s: %w", entry.name, err)
		}
		if _, err := w.Write([]byte(entry.content)); err != nil {
			return nil, fmt.Errorf("zip write %s: %w", entry.name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func docxContentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`
}

func docxRootRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`
}

func docxAppXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
  xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>Barq Cowork</Application>
  <DocSecurity>0</DocSecurity>
  <ScaleCrop>false</ScaleCrop>
  <Company>Barq Cowork</Company>
  <LinksUpToDate>false</LinksUpToDate>
  <SharedDoc>false</SharedDoc>
  <HyperlinksChanged>false</HyperlinksChanged>
  <AppVersion>1.0</AppVersion>
</Properties>`
}

func docxCoreXML(title string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties
  xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:dcmitype="http://purl.org/dc/dcmitype/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>%s</dc:title>
  <dc:creator>Barq Cowork</dc:creator>
  <cp:lastModifiedBy>Barq Cowork</cp:lastModifiedBy>
  <dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">%s</dcterms:modified>
</cp:coreProperties>`, xmlEsc(title), now, now)
}

func docxDocumentRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
}

func docxStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/>
        <w:color w:val="1E293B"/>
        <w:sz w:val="21"/>
        <w:szCs w:val="21"/>
      </w:rPr>
    </w:rPrDefault>
  </w:docDefaults>
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
    <w:qFormat/>
    <w:rPr>
      <w:rFonts w:ascii="Calibri Light" w:hAnsi="Calibri Light"/>
      <w:b/>
      <w:color w:val="1E293B"/>
      <w:sz w:val="48"/>
      <w:szCs w:val="48"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Subtitle">
    <w:name w:val="Subtitle"/>
    <w:rPr>
      <w:i/>
      <w:color w:val="64748B"/>
      <w:sz w:val="24"/>
      <w:szCs w:val="24"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:qFormat/>
    <w:rPr>
      <w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/>
      <w:b/>
      <w:color w:val="4F46E5"/>
      <w:sz w:val="28"/>
      <w:szCs w:val="28"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:qFormat/>
    <w:rPr>
      <w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/>
      <w:b/>
      <w:color w:val="1E293B"/>
      <w:sz w:val="24"/>
      <w:szCs w:val="24"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="ListParagraph">
    <w:name w:val="List Paragraph"/>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Meta">
    <w:name w:val="Meta"/>
    <w:rPr>
      <w:color w:val="64748B"/>
      <w:sz w:val="18"/>
      <w:szCs w:val="18"/>
    </w:rPr>
  </w:style>
</w:styles>`
}

func docxDocumentXML(args docxArgs) string {
	var body strings.Builder

	date := time.Now().UTC().Format("January 2, 2006")
	body.WriteString(docxAccentBarParagraph())
	body.WriteString(docxSpacerParagraph(320))
	body.WriteString(docxStyledParagraph("Title", args.Title, docxRunStyle{FontSizeHalfPts: 48, Bold: true, Color: "1E293B"}, 0, 280, "left", "", false))
	if args.Subtitle != "" {
		body.WriteString(docxStyledParagraph("Subtitle", args.Subtitle, docxRunStyle{FontSizeHalfPts: 24, Italic: true, Color: "64748B"}, 0, 420, "left", "", false))
	}
	if args.Author != "" {
		body.WriteString(docxMetaParagraph("Prepared by", args.Author))
	}
	body.WriteString(docxMetaParagraph("Date", date))
	body.WriteString(docxPageBreakParagraph())

	for _, sec := range args.Sections {
		level := sec.Level
		if level <= 0 {
			level = 1
		}
		if sec.Heading != "" {
			if level == 1 {
				body.WriteString(docxStyledParagraph("Heading1", strings.ToUpper(sec.Heading), docxRunStyle{FontSizeHalfPts: 28, Bold: true, Color: "4F46E5"}, 340, 140, "left", "6366F1", false))
			} else {
				body.WriteString(docxStyledParagraph("Heading2", sec.Heading, docxRunStyle{FontSizeHalfPts: 24, Bold: true, Color: "1E293B"}, 260, 100, "left", "A5B4FC", false))
			}
		}

		for _, line := range splitDocxContent(sec.Content) {
			switch {
			case line == "":
				continue
			case strings.HasPrefix(line, "• ") || strings.HasPrefix(line, "- "):
				body.WriteString(docxBulletParagraph(strings.TrimSpace(line[2:])))
			default:
				body.WriteString(docxStyledParagraph("Normal", line, docxRunStyle{FontSizeHalfPts: 21, Color: "1E293B"}, 80, 110, "left", "", false))
			}
		}

		if sec.Table != nil && len(sec.Table.Headers) > 0 {
			body.WriteString(docxTableXML(sec.Table))
			body.WriteString(docxSpacerParagraph(140))
		}
	}

	body.WriteString(docxSectionProps())
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document
  xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas"
  xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
  xmlns:o="urn:schemas-microsoft-com:office:office"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
  xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"
  xmlns:v="urn:schemas-microsoft-com:vml"
  xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing"
  xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
  xmlns:w10="urn:schemas-microsoft-com:office:word"
  xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
  xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml"
  xmlns:wpg="http://schemas.microsoft.com/office/word/2010/wordprocessingGroup"
  xmlns:wpi="http://schemas.microsoft.com/office/word/2010/wordprocessingInk"
  xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml"
  xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape"
  mc:Ignorable="w14 wp14">
  <w:body>` + body.String() + `</w:body>
</w:document>`
}

func splitDocxContent(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.Split(content, "\n")
}

func docxAccentBarParagraph() string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:before="0" w:after="0"/>
    <w:shd w:val="clear" w:color="auto" w:fill="6366F1"/>
  </w:pPr>
  <w:r>
    <w:rPr><w:sz w:val="18"/><w:szCs w:val="18"/></w:rPr>
    <w:t xml:space="preserve"> </w:t>
  </w:r>
</w:p>`
}

func docxSpacerParagraph(after int) string {
	return fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="0" w:after="%d"/></w:pPr></w:p>`, after)
}

func docxStyledParagraph(styleID, text string, style docxRunStyle, before, after int, align, bottomBorderColor string, keepNext bool) string {
	alignVal := ""
	switch strings.ToLower(align) {
	case "center":
		alignVal = `<w:jc w:val="center"/>`
	case "right":
		alignVal = `<w:jc w:val="right"/>`
	default:
		alignVal = `<w:jc w:val="left"/>`
	}
	keep := ""
	if keepNext {
		keep = `<w:keepNext/>`
	}
	border := ""
	if bottomBorderColor != "" {
		border = fmt.Sprintf(`<w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="%s"/></w:pBdr>`, bottomBorderColor)
	}
	return fmt.Sprintf(`<w:p>
  <w:pPr>
    <w:pStyle w:val="%s"/>
    %s
    %s
    <w:spacing w:before="%d" w:after="%d"/>
    %s
  </w:pPr>
  <w:r>%s<w:t>%s</w:t></w:r>
</w:p>`, styleID, keep, alignVal, before, after, border, docxRunProps(style), xmlEsc(text))
}

func docxMetaParagraph(label, value string) string {
	return fmt.Sprintf(`<w:p>
  <w:pPr>
    <w:pStyle w:val="Meta"/>
    <w:spacing w:before="0" w:after="80"/>
  </w:pPr>
  <w:r>%s<w:t>%s: </w:t></w:r>
  <w:r>%s<w:t>%s</w:t></w:r>
</w:p>`,
		docxRunProps(docxRunStyle{FontSizeHalfPts: 18, Color: "64748B"}),
		xmlEsc(label),
		docxRunProps(docxRunStyle{FontSizeHalfPts: 18, Bold: true, Color: "1E293B"}),
		xmlEsc(value),
	)
}

func docxBulletParagraph(text string) string {
	return fmt.Sprintf(`<w:p>
  <w:pPr>
    <w:pStyle w:val="ListParagraph"/>
    <w:spacing w:before="40" w:after="40" w:line="276" w:lineRule="auto"/>
    <w:ind w:left="360" w:hanging="180"/>
  </w:pPr>
  <w:r>%s<w:t>▸ </w:t></w:r>
  <w:r>%s<w:t>%s</w:t></w:r>
</w:p>`,
		docxRunProps(docxRunStyle{FontSizeHalfPts: 20, Color: "6366F1"}),
		docxRunProps(docxRunStyle{FontSizeHalfPts: 21, Color: "1E293B"}),
		xmlEsc(text),
	)
}

func docxTableXML(table *docxTable) string {
	colCount := len(table.Headers)
	if colCount == 0 {
		return ""
	}

	var grid strings.Builder
	for range table.Headers {
		grid.WriteString(`<w:gridCol w:w="2800"/>`)
	}

	var rows strings.Builder
	rows.WriteString(docxTableRowXML(table.Headers, true, 0))
	for i, row := range table.Rows {
		rows.WriteString(docxTableRowXML(row, false, i))
	}

	return `<w:tbl>
  <w:tblPr>
    <w:tblW w:w="0" w:type="auto"/>
    <w:jc w:val="left"/>
    <w:tblBorders>
      <w:top w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
      <w:left w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
      <w:bottom w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
      <w:right w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
      <w:insideH w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
      <w:insideV w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
    </w:tblBorders>
  </w:tblPr>
  <w:tblGrid>` + grid.String() + `</w:tblGrid>
  ` + rows.String() + `
</w:tbl>`
}

func docxTableRowXML(values []string, header bool, rowIndex int) string {
	fill := "FFFFFF"
	textColor := "1E293B"
	bold := false
	if header {
		fill = "6366F1"
		textColor = "FFFFFF"
		bold = true
	} else if rowIndex%2 == 1 {
		fill = "F8FAFC"
	}

	var cells strings.Builder
	for _, value := range values {
		cells.WriteString(fmt.Sprintf(`<w:tc>
  <w:tcPr>
    <w:tcW w:w="2800" w:type="dxa"/>
    <w:shd w:val="clear" w:color="auto" w:fill="%s"/>
    <w:vAlign w:val="center"/>
  </w:tcPr>
  <w:p>
    <w:pPr><w:jc w:val="%s"/><w:spacing w:before="60" w:after="60"/></w:pPr>
    <w:r>%s<w:t>%s</w:t></w:r>
  </w:p>
</w:tc>`, fill, ternary(header, "center", "left"), docxRunProps(docxRunStyle{FontSizeHalfPts: 20, Bold: bold, Color: textColor}), xmlEsc(value)))
	}
	return `<w:tr>` + cells.String() + `</w:tr>`
}

func docxPageBreakParagraph() string {
	return `<w:p><w:r><w:br w:type="page"/></w:r></w:p>`
}

func docxSectionProps() string {
	return `<w:sectPr>
  <w:pgSz w:w="12240" w:h="15840"/>
  <w:pgMar w:top="1440" w:right="1656" w:bottom="1440" w:left="1656" w:header="708" w:footer="708" w:gutter="0"/>
</w:sectPr>`
}

func docxRunProps(style docxRunStyle) string {
	size := style.FontSizeHalfPts
	if size <= 0 {
		size = 21
	}
	color := style.Color
	if color == "" {
		color = "1E293B"
	}

	var sb strings.Builder
	sb.WriteString(`<w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/>`)
	if style.Bold {
		sb.WriteString(`<w:b/>`)
	}
	if style.Italic {
		sb.WriteString(`<w:i/>`)
	}
	sb.WriteString(fmt.Sprintf(`<w:color w:val="%s"/><w:sz w:val="%d"/><w:szCs w:val="%d"/></w:rPr>`, color, size, size))
	return sb.String()
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
