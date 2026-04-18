package generator

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// htmlToDocx converts HTML into a fully-styled Office Open XML (.docx) file.
// This is a first-class pure-Go implementation — no external tools (Pandoc,
// LibreOffice) are invoked and no pre-rendered templates are used. The
// converter walks the parsed HTML DOM and emits OOXML paragraphs, tables,
// list items, hyperlinks, and embedded images directly, inheriting the UAE
// AI Safety paragraph and character styles from styles.xml.
func htmlToDocx(ctx context.Context, req Request) ([]byte, error) {
	doc, err := html.Parse(strings.NewReader(req.HTML))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	start := findBody(doc)
	if start == nil {
		start = doc
	}

	dw := newDocxWriter(ctx)
	dw.theme = resolveDocxTheme(req.Theme)
	dw.chrome = req.Chrome
	dw.walkBlocks(start)

	sectPrChrome := ""
	if dw.chrome != nil {
		sectPrChrome = `
      <w:headerReference w:type="default" r:id="rId5"/>
      <w:footerReference w:type="default" r:id="rId6"/>
      <w:headerReference w:type="first" r:id="rId7"/>
      <w:footerReference w:type="first" r:id="rId8"/>
      <w:titlePg/>`
	}

	documentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
  <w:body>
%s
    <w:sectPr>%s
      <w:pgSz w:w="11906" w:h="16838"/>
      <w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134" w:header="709" w:footer="709" w:gutter="0"/>
    </w:sectPr>
  </w:body>
</w:document>`, dw.body.String(), sectPrChrome)

	return dw.assemble(documentXML, req.Title)
}

// ─── Writer state ───────────────────────────────────────────────────────────

type docxImage struct {
	id        string
	part      string
	mime      string
	data      []byte
	widthEMU  int
	heightEMU int
}

type docxHyperlink struct {
	id     string
	target string
}

type docxWriter struct {
	ctx        context.Context
	body       *strings.Builder
	images     []docxImage
	hyperlinks []docxHyperlink
	nextRelID  int
	drawingID  int
	theme      DocxTheme
	chrome     *DocxChrome
}

func newDocxWriter(ctx context.Context) *docxWriter {
	return &docxWriter{ctx: ctx, body: &strings.Builder{}, nextRelID: 100, drawingID: 1}
}

// captureBlocks walks n's block children into a temporary buffer and returns
// the resulting OOXML. Used by styled-box renderers so they can wrap inner
// content in tables/borders/shading without polluting the main body stream.
func (w *docxWriter) captureBlocks(walk func()) string {
	saved := w.body
	local := &strings.Builder{}
	w.body = local
	walk()
	w.body = saved
	return local.String()
}

func (w *docxWriter) relID() string {
	id := fmt.Sprintf("rId%d", w.nextRelID)
	w.nextRelID++
	return id
}

// ─── DOCX packaging ─────────────────────────────────────────────────────────

func (w *docxWriter) assemble(documentXML, title string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, content string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}

	hasChrome := w.chrome != nil
	parts := []struct{ name, content string }{
		{"[Content_Types].xml", contentTypesXML(w.images, hasChrome)},
		{"_rels/.rels", rootRelsXML()},
		{"docProps/app.xml", appXML()},
		{"docProps/core.xml", coreXML(title)},
		{"word/document.xml", documentXML},
		{"word/styles.xml", stylesXML(w.theme)},
		{"word/settings.xml", settingsXML()},
		{"word/numbering.xml", numberingXML(w.theme)},
		{"word/theme/theme1.xml", themeXML(w.theme)},
		{"word/_rels/document.xml.rels", documentRelsXML(w.images, w.hyperlinks, hasChrome)},
	}
	if hasChrome {
		parts = append(parts,
			struct{ name, content string }{"word/header1.xml", headerXML(*w.chrome, w.theme)},
			struct{ name, content string }{"word/footer1.xml", footerXML(*w.chrome, w.theme)},
			struct{ name, content string }{"word/header2.xml", emptyHeaderXML()},
			struct{ name, content string }{"word/footer2.xml", emptyFooterXML()},
		)
	}
	for _, p := range parts {
		if err := add(p.name, p.content); err != nil {
			return nil, fmt.Errorf("zip %s: %w", p.name, err)
		}
	}

	for _, img := range w.images {
		f, err := zw.Create("word/" + img.part)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(img.data); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── HTML traversal ─────────────────────────────────────────────────────────

func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == atom.Body {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBody(c); b != nil {
			return b
		}
	}
	return nil
}

func (w *docxWriter) walkBlocks(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.emitBlock(c)
	}
}

func (w *docxWriter) emitBlock(c *html.Node) {
	if c.Type == html.TextNode {
		if strings.TrimSpace(c.Data) != "" {
			w.writeParagraph(c, "Normal", "", "")
		}
		return
	}
	if c.Type != html.ElementNode {
		return
	}

	switch c.DataAtom {
	case atom.H1:
		w.writeParagraph(c, "Heading1", "", "")
	case atom.H2:
		w.writeParagraph(c, "Heading2", "", "")
	case atom.H3:
		w.writeParagraph(c, "Heading3", "", "")
	case atom.H4, atom.H5, atom.H6:
		w.writeParagraph(c, "Heading4", "", "")

	case atom.P:
		w.writeParagraph(c, "Normal", "", paragraphAlignment(c))

	case atom.Blockquote:
		w.walkBlocksAs(c, "BlockText")

	case atom.Ul:
		w.emitList(c, 1, 0)
	case atom.Ol:
		w.emitList(c, 2, 0)

	case atom.Pre:
		w.writeParagraph(c, "SourceCode", "", "")

	case atom.Hr:
		cls := getAttr(c, "class")
		switch {
		case strings.Contains(cls, "pagebreak") || strings.Contains(cls, "page-break"):
			w.body.WriteString(pageBreakParagraph())
		case strings.Contains(cls, "divider-dots") || strings.Contains(cls, "dots"):
			w.body.WriteString(w.renderDividerDots())
		default:
			w.body.WriteString(w.hrParagraph())
		}

	case atom.Table:
		w.body.WriteString(w.tableFromHTML(c))

	case atom.Dl:
		w.walkBlocks(c)
	case atom.Dt:
		w.writeParagraph(c, "Heading4", "", "")
	case atom.Dd:
		w.writeParagraph(c, "Normal", `<w:ind w:left="480"/>`, "")

	case atom.Figure:
		w.walkBlocks(c)
	case atom.Figcaption:
		w.writeParagraph(c, "Caption", "", "center")

	case atom.Div, atom.Section, atom.Article, atom.Main, atom.Header, atom.Footer, atom.Aside:
		class := getAttr(c, "class")
		switch {
		case strings.Contains(class, "cover-page"):
			w.walkBlocks(c)
			w.body.WriteString(pageBreakParagraph())
		case strings.Contains(class, "page-header") ||
			strings.Contains(class, "page-footer") ||
			strings.Contains(class, "watermark"):
			// Decorative elements from the PDF print shell — skip in DOCX.
		default:
			if w.emitStyledBlockIfClass(c) {
				return
			}
			if strings.Contains(class, "info-box") || strings.Contains(class, "warning-box") {
				w.walkBlocksAs(c, "BlockText")
				return
			}
			w.walkBlocks(c)
		}

	default:
		w.walkBlocks(c)
	}
}

func (w *docxWriter) walkBlocksAs(n *html.Node, styleID string) {
	if !hasBlockChildren(n) {
		w.writeParagraph(n, styleID, "", "")
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.ElementNode && c.DataAtom == atom.P:
			w.writeParagraph(c, styleID, "", "")
		case c.Type == html.ElementNode && isBlockAtom(c.DataAtom):
			w.emitBlock(c)
		case c.Type == html.TextNode && strings.TrimSpace(c.Data) != "":
			w.writeParagraph(c, styleID, "", "")
		case c.Type == html.ElementNode:
			w.walkBlocksAs(c, styleID)
		}
	}
}

func (w *docxWriter) emitList(listEl *html.Node, numID, depth int) {
	for c := listEl.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.DataAtom != atom.Li {
			continue
		}
		w.writeParagraph(c, "ListParagraph", fmt.Sprintf(
			`<w:numPr><w:ilvl w:val="%d"/><w:numId w:val="%d"/></w:numPr>`, depth, numID), "")

		for nested := c.FirstChild; nested != nil; nested = nested.NextSibling {
			if nested.Type == html.ElementNode &&
				(nested.DataAtom == atom.Ul || nested.DataAtom == atom.Ol) {
				nID := 1
				if nested.DataAtom == atom.Ol {
					nID = 2
				}
				w.emitList(nested, nID, depth+1)
			}
		}
	}
}

// ─── Inline handling ────────────────────────────────────────────────────────

type runStyle struct {
	bold      bool
	italic    bool
	underline bool
	code      bool
	strike    bool
	hyperlink string
	linkColor string // hex without '#', set when hyperlink is active
}

func (w *docxWriter) writeParagraph(n *html.Node, styleID, extraPPr, align string) {
	alignXML := ""
	switch align {
	case "center":
		alignXML = `<w:jc w:val="center"/>`
	case "right":
		alignXML = `<w:jc w:val="right"/>`
	case "justify":
		alignXML = `<w:jc w:val="both"/>`
	}

	var runs strings.Builder
	w.collectInlines(n, runStyle{}, &runs)

	w.body.WriteString(fmt.Sprintf(
		"    <w:p><w:pPr><w:pStyle w:val=\"%s\"/>%s%s</w:pPr>%s</w:p>\n",
		styleID, extraPPr, alignXML, runs.String()))
}

func (w *docxWriter) collectInlines(n *html.Node, st runStyle, out *strings.Builder) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if c.Data == "" {
				continue
			}
			text := normalizeWhitespace(c.Data)
			if text == "" {
				continue
			}
			if st.hyperlink != "" {
				linkStyle := runStyle{
					bold: st.bold, italic: st.italic, underline: true,
					code: st.code, strike: st.strike, hyperlink: st.hyperlink,
					linkColor: st.linkColor,
				}
				out.WriteString(fmt.Sprintf(`<w:hyperlink r:id="%s">%s</w:hyperlink>`,
					st.hyperlink, runXML(text, linkStyle)))
			} else {
				out.WriteString(runXML(text, st))
			}

		case html.ElementNode:
			ns := st
			switch c.DataAtom {
			case atom.B, atom.Strong:
				ns.bold = true
			case atom.I, atom.Em:
				ns.italic = true
			case atom.U:
				ns.underline = true
			case atom.Code:
				ns.code = true
			case atom.S, atom.Strike, atom.Del:
				ns.strike = true

			case atom.Br:
				out.WriteString(`<w:r><w:br/></w:r>`)
				continue

			case atom.A:
				if href := getAttr(c, "href"); href != "" {
					rid := w.relID()
					w.hyperlinks = append(w.hyperlinks, docxHyperlink{id: rid, target: href})
					ns.hyperlink = rid
					ns.linkColor = w.theme.LinkColor
				}

			case atom.Img:
				if xml := w.embedImage(c); xml != "" {
					out.WriteString(xml)
				}
				continue
			}
			w.collectInlines(c, ns, out)
		}
	}
}

func runXML(text string, st runStyle) string {
	var rpr strings.Builder
	rpr.WriteString("<w:rPr>")
	if st.code {
		rpr.WriteString(`<w:rStyle w:val="VerbatimChar"/>`)
	}
	if st.bold {
		rpr.WriteString(`<w:b/>`)
	}
	if st.italic {
		rpr.WriteString(`<w:i/>`)
	}
	if st.underline {
		rpr.WriteString(`<w:u w:val="single"/>`)
	}
	if st.strike {
		rpr.WriteString(`<w:strike/>`)
	}
	if st.hyperlink != "" {
		linkColor := st.linkColor
		if linkColor == "" {
			linkColor = "1D4ED8"
		}
		fmt.Fprintf(&rpr, `<w:color w:val="%s"/>`, linkColor)
	}
	rpr.WriteString("</w:rPr>")

	return fmt.Sprintf(`<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r>`,
		rpr.String(), xmlEscape(text))
}

// ─── Images ─────────────────────────────────────────────────────────────────

var dataURIRegex = regexp.MustCompile(`^data:([a-zA-Z0-9+/.-]+);base64,(.*)$`)

func (w *docxWriter) embedImage(n *html.Node) string {
	src := getAttr(n, "src")
	if src == "" {
		return ""
	}

	var (
		data []byte
		mime string
		err  error
	)

	if m := dataURIRegex.FindStringSubmatch(src); m != nil {
		mime = m[1]
		data, err = base64.StdEncoding.DecodeString(m[2])
		if err != nil {
			return ""
		}
	} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		data, mime, err = fetchRemoteImage(w.ctx, src)
		if err != nil {
			return ""
		}
	} else {
		return ""
	}

	ext := imageExt(mime)
	if ext == "" {
		return ""
	}

	idx := len(w.images) + 1
	part := fmt.Sprintf("media/image%d.%s", idx, ext)
	rid := w.relID()

	img := docxImage{
		id:        rid,
		part:      part,
		mime:      mime,
		data:      data,
		widthEMU:  3810000, // ~4.17 in
		heightEMU: 2857500, // ~3.12 in
	}
	w.images = append(w.images, img)

	alt := getAttr(n, "alt")
	if alt == "" {
		alt = fmt.Sprintf("Figure %d", idx)
	}
	dID := w.drawingID
	w.drawingID++

	return fmt.Sprintf(`<w:r><w:drawing>
  <wp:inline distT="0" distB="0" distL="0" distR="0">
    <wp:extent cx="%d" cy="%d"/>
    <wp:effectExtent l="0" t="0" r="0" b="0"/>
    <wp:docPr id="%d" name="%s" descr="%s"/>
    <wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>
    <a:graphic>
      <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">
        <pic:pic>
          <pic:nvPicPr>
            <pic:cNvPr id="%d" name="%s"/>
            <pic:cNvPicPr/>
          </pic:nvPicPr>
          <pic:blipFill>
            <a:blip r:embed="%s"/>
            <a:stretch><a:fillRect/></a:stretch>
          </pic:blipFill>
          <pic:spPr>
            <a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>
            <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          </pic:spPr>
        </pic:pic>
      </a:graphicData>
    </a:graphic>
  </wp:inline>
</w:drawing></w:r>`,
		img.widthEMU, img.heightEMU,
		dID, xmlEscape(alt), xmlEscape(alt),
		dID, xmlEscape(alt),
		rid,
		img.widthEMU, img.heightEMU)
}

func fetchRemoteImage(ctx context.Context, url string) ([]byte, string, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("fetch image: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = http.DetectContentType(data)
	}
	return data, strings.Split(mime, ";")[0], nil
}

func imageExt(mime string) string {
	switch strings.ToLower(mime) {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return ""
}

// ─── Paragraph helpers ──────────────────────────────────────────────────────

func pageBreakParagraph() string {
	return "    <w:p><w:r><w:br w:type=\"page\"/></w:r></w:p>\n"
}

func (w *docxWriter) hrParagraph() string {
	color := lightenHex(w.theme.AccentColor, 0.7)
	return fmt.Sprintf(`    <w:p>
      <w:pPr>
        <w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="%s"/></w:pBdr>
        <w:spacing w:before="120" w:after="120"/>
      </w:pPr>
    </w:p>
`, color)
}

func paragraphAlignment(n *html.Node) string {
	class := strings.ToLower(getAttr(n, "class"))
	style := strings.ToLower(getAttr(n, "style"))
	switch {
	case strings.Contains(class, "text-center") || strings.Contains(style, "text-align:center"):
		return "center"
	case strings.Contains(class, "text-right") || strings.Contains(style, "text-align:right"):
		return "right"
	case strings.Contains(class, "text-justify") || strings.Contains(style, "text-align:justify"):
		return "justify"
	}
	return ""
}

// ─── Tables ─────────────────────────────────────────────────────────────────

func (w *docxWriter) tableFromHTML(tbl *html.Node) string {
	rows := collectTableRows(tbl)
	if len(rows) == 0 {
		return ""
	}
	cols := 0
	for _, r := range rows {
		if len(r.cells) > cols {
			cols = len(r.cells)
		}
	}
	if cols == 0 {
		return ""
	}

	colWidth := 9000 / cols
	if colWidth < 600 {
		colWidth = 600
	}

	var grid strings.Builder
	for i := 0; i < cols; i++ {
		fmt.Fprintf(&grid, `<w:gridCol w:w="%d"/>`, colWidth)
	}

	var rowsXML strings.Builder
	for i, r := range rows {
		rowsXML.WriteString(w.tableRowXML(r.cells, r.header, i, colWidth))
	}

	border := lightenHex(w.theme.AccentColor, 0.75)
	return fmt.Sprintf(`    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="9000" w:type="dxa"/>
        <w:tblBorders>
          <w:top    w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
          <w:left   w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
          <w:bottom w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
          <w:right  w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
          <w:insideH w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
          <w:insideV w:val="single" w:sz="4" w:space="0" w:color="%[2]s"/>
        </w:tblBorders>
      </w:tblPr>
      <w:tblGrid>%[1]s</w:tblGrid>
%[3]s    </w:tbl>
`, grid.String(), border, rowsXML.String())
}

type htmlRow struct {
	cells  []*html.Node
	header bool
}

func collectTableRows(tbl *html.Node) []htmlRow {
	var out []htmlRow
	var walk func(n *html.Node, inHead bool)
	walk = func(n *html.Node, inHead bool) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			switch c.DataAtom {
			case atom.Thead:
				walk(c, true)
			case atom.Tbody, atom.Tfoot:
				walk(c, false)
			case atom.Tr:
				var cells []*html.Node
				isHead := inHead
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if cell.Type == html.ElementNode &&
						(cell.DataAtom == atom.Td || cell.DataAtom == atom.Th) {
						cells = append(cells, cell)
						if cell.DataAtom == atom.Th {
							isHead = true
						}
					}
				}
				if len(cells) > 0 {
					out = append(out, htmlRow{cells: cells, header: isHead})
				}
			default:
				walk(c, inHead)
			}
		}
	}
	walk(tbl, false)
	return out
}

func (w *docxWriter) tableRowXML(cells []*html.Node, header bool, rowIdx, colWidth int) string {
	fill := "FFFFFF"
	textColor := w.theme.BodyColor
	if header {
		fill = w.theme.AccentColor
		textColor = readableTextOn(fill)
	} else if rowIdx%2 == 1 {
		fill = lightenHex(w.theme.AccentColor, 0.92)
	}

	var cellsXML strings.Builder
	for _, cell := range cells {
		var runs strings.Builder
		st := runStyle{bold: header}
		w.collectInlines(cell, st, &runs)
		if runs.Len() == 0 {
			runs.WriteString(runXML("", st))
		}
		coloured := addColorToRuns(runs.String(), textColor)

		fmt.Fprintf(&cellsXML, `        <w:tc>
          <w:tcPr>
            <w:tcW w:w="%d" w:type="dxa"/>
            <w:shd w:val="clear" w:color="auto" w:fill="%s"/>
            <w:vAlign w:val="center"/>
          </w:tcPr>
          <w:p>
            <w:pPr><w:spacing w:before="60" w:after="60"/></w:pPr>%s
          </w:p>
        </w:tc>
`, colWidth, fill, coloured)
	}

	return fmt.Sprintf("      <w:tr>\n%s      </w:tr>\n", cellsXML.String())
}

func addColorToRuns(runs, color string) string {
	return strings.ReplaceAll(runs, "<w:rPr>", fmt.Sprintf(`<w:rPr><w:color w:val="%s"/>`, color))
}

// ─── Utilities ──────────────────────────────────────────────────────────────

func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func isBlockAtom(a atom.Atom) bool {
	switch a {
	case atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6,
		atom.Ul, atom.Ol, atom.Li, atom.Blockquote, atom.Pre, atom.Hr,
		atom.Table, atom.Div, atom.Section, atom.Article, atom.Main,
		atom.Dl, atom.Dt, atom.Dd, atom.Figure, atom.Figcaption:
		return true
	}
	return false
}

func hasBlockChildren(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && isBlockAtom(c.DataAtom) {
			return true
		}
	}
	return false
}

func normalizeWhitespace(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
