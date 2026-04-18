package generator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"
)

// buildReferenceDocx creates a minimal .docx whose styles.xml carries the
// UAE AI Safety theme. Pandoc reads this as its --reference-doc, so the
// paragraph and character styles are applied to every generated paragraph
// without touching the content.
func buildReferenceDocx() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	entries := []struct {
		name    string
		content string
	}{
		{"[Content_Types].xml", refContentTypesXML()},
		{"_rels/.rels", refRootRelsXML()},
		{"docProps/app.xml", refAppXML()},
		{"docProps/core.xml", refCoreXML()},
		{"word/document.xml", refDocumentXML()},
		{"word/styles.xml", refStylesXML()},
		{"word/settings.xml", refSettingsXML()},
		{"word/theme/theme1.xml", refThemeXML()},
		{"word/_rels/document.xml.rels", refDocumentRelsXML()},
	}

	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			return nil, fmt.Errorf("zip create %s: %w", e.name, err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			return nil, fmt.Errorf("zip write %s: %w", e.name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func refContentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml"  ContentType="application/xml"/>
  <Override PartName="/word/document.xml"       ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml"         ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/settings.xml"       ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>
  <Override PartName="/word/theme/theme1.xml"   ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
  <Override PartName="/docProps/core.xml"       ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml"        ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`
}

func refRootRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"          Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"       Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties"     Target="docProps/app.xml"/>
</Relationships>`
}

func refAppXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">
  <Application>Barq Cowork – UAE AI Safety</Application>
  <AppVersion>1.0</AppVersion>
</Properties>`
}

func refCoreXML() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties
  xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>Barq Cowork Reference</dc:title>
  <dc:creator>Barq Cowork</dc:creator>
  <dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">%s</dcterms:modified>
</cp:coreProperties>`, now, now)
}

// refDocumentXML is a minimal body — Pandoc only reads the styles, not content.
func refDocumentXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Normal"/></w:pPr></w:p>
    <w:sectPr>
      <w:pgSz w:w="11906" w:h="16838"/>
      <w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134"
               w:header="709" w:footer="709" w:gutter="0"/>
    </w:sectPr>
  </w:body>
</w:document>`
}

func refDocumentRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles"   Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"    Target="theme/theme1.xml"/>
</Relationships>`
}

func refSettingsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:defaultTabStop w:val="709"/>
  <w:compat><w:compatSetting w:name="compatibilityMode" w:uri="http://schemas.microsoft.com/office/word" w:val="15"/></w:compat>
</w:settings>`
}

// refThemeXML encodes the UAE AI Safety colour theme for Word.
// Accent1 = Crimson Red, Accent2 = Dark Green.
func refThemeXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="UAE AI Safety">
  <a:themeElements>
    <a:clrScheme name="UAE AI Safety">
      <a:dk1><a:srgbClr val="1A1A2E"/></a:dk1>
      <a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>
      <a:dk2><a:srgbClr val="0F3460"/></a:dk2>
      <a:lt2><a:srgbClr val="F8FAFC"/></a:lt2>
      <a:accent1><a:srgbClr val="BE123C"/></a:accent1>
      <a:accent2><a:srgbClr val="064E3B"/></a:accent2>
      <a:accent3><a:srgbClr val="1D4ED8"/></a:accent3>
      <a:accent4><a:srgbClr val="7C3AED"/></a:accent4>
      <a:accent5><a:srgbClr val="0369A1"/></a:accent5>
      <a:accent6><a:srgbClr val="D97706"/></a:accent6>
      <a:hlink><a:srgbClr val="1D4ED8"/></a:hlink>
      <a:folHlink><a:srgbClr val="7C3AED"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="UAE AI Safety">
      <a:majorFont><a:latin typeface="Montserrat"/><a:ea typeface=""/><a:cs typeface=""/></a:majorFont>
      <a:minorFont><a:latin typeface="Inter"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="UAE AI Safety">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="6350" cap="flat" cmpd="sng"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="12700" cap="flat" cmpd="sng"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="19050" cap="flat" cmpd="sng"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`
}

// refStylesXML defines Word paragraph and character styles for the UAE AI
// Safety theme. Pandoc maps HTML elements to these style IDs automatically.
func refStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
          xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml"
          xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
          mc:Ignorable="w14">

  <!-- ── Document defaults ─────────────────────────────────── -->
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:rFonts w:ascii="Inter" w:hAnsi="Inter" w:cs="Inter"/>
        <w:color w:val="1A1A2E"/>
        <w:sz w:val="21"/>
        <w:szCs w:val="21"/>
        <w:lang w:val="en-US"/>
      </w:rPr>
    </w:rPrDefault>
    <w:pPrDefault>
      <w:pPr>
        <w:spacing w:after="120" w:line="288" w:lineRule="auto"/>
      </w:pPr>
    </w:pPrDefault>
  </w:docDefaults>

  <!-- ── Normal ─────────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:after="120" w:line="288" w:lineRule="auto"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Inter" w:hAnsi="Inter" w:cs="Inter"/>
      <w:color w:val="1A1A2E"/>
      <w:sz w:val="21"/>
      <w:szCs w:val="21"/>
    </w:rPr>
  </w:style>

  <!-- ── Body Text ──────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="BodyText">
    <w:name w:val="Body Text"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:spacing w:after="120" w:line="288" w:lineRule="auto"/></w:pPr>
  </w:style>

  <!-- ── First Paragraph ────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="FirstParagraph">
    <w:name w:val="First Paragraph"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:spacing w:before="0" w:after="120" w:line="288" w:lineRule="auto"/></w:pPr>
  </w:style>

  <!-- ── Title ──────────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
    <w:qFormat/>
    <w:pPr>
      <w:spacing w:before="0" w:after="360"/>
      <w:jc w:val="left"/>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Montserrat" w:hAnsi="Montserrat" w:cs="Montserrat"/>
      <w:b/>
      <w:color w:val="1A1A2E"/>
      <w:sz w:val="56"/>
      <w:szCs w:val="56"/>
    </w:rPr>
  </w:style>

  <!-- ── Subtitle ───────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="Subtitle">
    <w:name w:val="Subtitle"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:spacing w:before="0" w:after="280"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Inter" w:hAnsi="Inter" w:cs="Inter"/>
      <w:color w:val="475569"/>
      <w:sz w:val="26"/>
      <w:szCs w:val="26"/>
    </w:rPr>
  </w:style>

  <!-- ── Heading 1  (Crimson Red, 16pt, Montserrat Bold) ───── -->
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="360" w:after="120"/>
      <w:pBdr>
        <w:bottom w:val="single" w:sz="6" w:space="1" w:color="BE123C"/>
      </w:pBdr>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Montserrat" w:hAnsi="Montserrat" w:cs="Montserrat"/>
      <w:b/>
      <w:color w:val="BE123C"/>
      <w:sz w:val="32"/>
      <w:szCs w:val="32"/>
    </w:rPr>
  </w:style>

  <!-- ── Heading 2  (Dark Green, 13pt, Montserrat Bold) ────── -->
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="280" w:after="100"/>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Montserrat" w:hAnsi="Montserrat" w:cs="Montserrat"/>
      <w:b/>
      <w:color w:val="064E3B"/>
      <w:sz w:val="26"/>
      <w:szCs w:val="26"/>
    </w:rPr>
  </w:style>

  <!-- ── Heading 3  (Slate 900, 11.5pt, Inter SemiBold) ────── -->
  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr>
      <w:keepNext/>
      <w:spacing w:before="200" w:after="80"/>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Inter" w:hAnsi="Inter" w:cs="Inter"/>
      <w:b/>
      <w:color w:val="1A1A2E"/>
      <w:sz w:val="23"/>
      <w:szCs w:val="23"/>
    </w:rPr>
  </w:style>

  <!-- ── Heading 4 ──────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="Heading4">
    <w:name w:val="heading 4"/>
    <w:basedOn w:val="Normal"/>
    <w:next w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:keepNext/><w:spacing w:before="160" w:after="60"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Inter" w:hAnsi="Inter" w:cs="Inter"/>
      <w:b/>
      <w:color w:val="334155"/>
      <w:sz w:val="21"/>
      <w:szCs w:val="21"/>
    </w:rPr>
  </w:style>

  <!-- ── List Paragraph ─────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="ListParagraph">
    <w:name w:val="List Paragraph"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr>
      <w:ind w:left="440" w:hanging="220"/>
      <w:spacing w:before="40" w:after="40" w:line="276" w:lineRule="auto"/>
    </w:pPr>
  </w:style>

  <!-- ── Compact (Pandoc tight list item) ───────────────────── -->
  <w:style w:type="paragraph" w:styleId="Compact">
    <w:name w:val="Compact"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:spacing w:before="0" w:after="40" w:line="264" w:lineRule="auto"/></w:pPr>
  </w:style>

  <!-- ── Block Text (blockquote) ────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="BlockText">
    <w:name w:val="Block Text"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr>
      <w:ind w:left="440" w:right="440"/>
      <w:spacing w:before="120" w:after="120"/>
      <w:pBdr>
        <w:left w:val="single" w:sz="18" w:space="0" w:color="BE123C"/>
      </w:pBdr>
    </w:pPr>
    <w:rPr>
      <w:i/>
      <w:color w:val="4A1124"/>
    </w:rPr>
  </w:style>

  <!-- ── Source Code / Verbatim ─────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="SourceCode">
    <w:name w:val="Source Code"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr>
      <w:shd w:val="clear" w:color="auto" w:fill="0F172A"/>
      <w:spacing w:before="120" w:after="120" w:line="264" w:lineRule="auto"/>
      <w:ind w:left="220" w:right="220"/>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/>
      <w:color w:val="E2E8F0"/>
      <w:sz w:val="17"/>
      <w:szCs w:val="17"/>
    </w:rPr>
  </w:style>

  <!-- ── Verbatim Char (inline code) ───────────────────────── -->
  <w:style w:type="character" w:styleId="VerbatimChar">
    <w:name w:val="Verbatim Char"/>
    <w:rPr>
      <w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/>
      <w:color w:val="0F172A"/>
      <w:shd w:val="clear" w:color="auto" w:fill="F1F5F9"/>
      <w:sz w:val="17"/>
      <w:szCs w:val="17"/>
    </w:rPr>
  </w:style>

  <!-- ── Caption ────────────────────────────────────────────── -->
  <w:style w:type="paragraph" w:styleId="Caption">
    <w:name w:val="Caption"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:jc w:val="center"/><w:spacing w:before="60" w:after="120"/></w:pPr>
    <w:rPr>
      <w:i/>
      <w:color w:val="64748B"/>
      <w:sz w:val="17"/>
      <w:szCs w:val="17"/>
    </w:rPr>
  </w:style>

</w:styles>`
}
