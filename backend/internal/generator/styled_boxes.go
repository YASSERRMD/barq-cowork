package generator

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// styled_boxes.go maps semantic HTML class names emitted by the LLM (pullquote,
// callout, callout-info, callout-tip, callout-warn, keyidea, definition,
// statbox, sidebar, factbox, stat-value/stat-label, divider-dots) into OOXML
// constructs: single-cell shaded tables with colored left borders, centered
// stat cards, decorative divider rows. All colors are derived from the active
// theme — nothing is hardcoded. These are the building blocks that make a
// generated document look like a real publication rather than plain prose.

// warnAccentColor is the fallback amber used for callout-warn when the theme
// has no dedicated warning hue. Amber is a universal "caution" signal.
const warnAccentColor = "D97706"

// classList returns a space-separated class list as a slice of tokens.
func classList(n *html.Node) []string {
	raw := strings.TrimSpace(getAttr(n, "class"))
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range classList(n) {
		if c == class {
			return true
		}
	}
	return false
}

// hasAnyClass returns the first matching class from candidates (or "").
func matchedClass(n *html.Node, candidates ...string) string {
	set := map[string]bool{}
	for _, c := range classList(n) {
		set[c] = true
	}
	for _, want := range candidates {
		if set[want] {
			return want
		}
	}
	return ""
}

// emitStyledBlockIfClass inspects the class list on n and, if it matches a
// known styled-box class, renders n as that styled box and returns true.
// Returns false when no class matches and the caller should fall back to
// default block handling.
func (w *docxWriter) emitStyledBlockIfClass(n *html.Node) bool {
	// Guard against the LLM emitting empty styled divs like
	// <div class="callout"></div> — rendering the frame around nothing
	// produces a visible empty highlighted block, which looks like broken
	// layout. Skip the whole node silently when it carries no text.
	if matchedClass(n,
		"pullquote", "callout-info", "callout-tip", "callout-warn", "callout",
		"keyidea", "key-idea", "definition", "statbox", "stat-box",
		"sidebar", "factbox", "fact-box",
	) != "" && !nodeHasVisibleText(n) {
		return true
	}
	// Order matters: check the most specific variants first so callout-info
	// doesn't get swallowed by a plain callout match.
	switch matchedClass(n,
		"pullquote",
		"callout-info", "callout-tip", "callout-warn", "callout",
		"keyidea", "key-idea",
		"definition",
		"statbox", "stat-box",
		"sidebar",
		"factbox", "fact-box",
	) {
	case "pullquote":
		w.body.WriteString(w.renderPullQuote(n))
		return true
	case "callout-info":
		w.body.WriteString(w.renderCallout(n, w.theme.LinkColor, "Note"))
		return true
	case "callout-tip":
		w.body.WriteString(w.renderCallout(n, w.theme.SecondaryColor, "Tip"))
		return true
	case "callout-warn":
		w.body.WriteString(w.renderCallout(n, warnAccentColor, "Caution"))
		return true
	case "callout":
		w.body.WriteString(w.renderCallout(n, w.theme.AccentColor, ""))
		return true
	case "keyidea", "key-idea":
		w.body.WriteString(w.renderKeyIdea(n))
		return true
	case "definition":
		w.body.WriteString(w.renderDefinition(n))
		return true
	case "statbox", "stat-box":
		w.body.WriteString(w.renderStatBox(n))
		return true
	case "sidebar":
		w.body.WriteString(w.renderSidebar(n))
		return true
	case "factbox", "fact-box":
		// Factbox is a labelled wrapper around a <table>. Render the table
		// children through the normal pipeline but wrap in a subtle heading
		// paragraph first.
		w.body.WriteString(w.renderFactBox(n))
		return true
	}
	return false
}

// renderStyledBox emits a single-cell shaded table wrapping the block contents
// of n. Left border uses borderColor at borderWidthEighths eighths-of-a-point.
// fillColor is the cell background (pass "" for none).
func (w *docxWriter) renderStyledBox(n *html.Node, borderColor, fillColor string, borderWidthEighths int) string {
	inner := w.captureBlocks(func() { w.walkBlocks(n) })
	if strings.TrimSpace(inner) == "" {
		// No block children — render the node's own inline content as a
		// single paragraph inside the box.
		inner = w.captureBlocks(func() {
			w.writeParagraph(n, "Normal", "", "")
		})
	}

	fillXML := ""
	if fillColor != "" {
		fillXML = fmt.Sprintf(`<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, fillColor)
	}

	return fmt.Sprintf(`    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="9000" w:type="dxa"/>
        <w:tblBorders>
          <w:top    w:val="nil"/>
          <w:left   w:val="single" w:sz="%d" w:space="0" w:color="%s"/>
          <w:bottom w:val="nil"/>
          <w:right  w:val="nil"/>
          <w:insideH w:val="nil"/>
          <w:insideV w:val="nil"/>
        </w:tblBorders>
        <w:tblCellMar>
          <w:top w:w="120" w:type="dxa"/>
          <w:left w:w="240" w:type="dxa"/>
          <w:bottom w:w="120" w:type="dxa"/>
          <w:right w:w="240" w:type="dxa"/>
        </w:tblCellMar>
      </w:tblPr>
      <w:tblGrid><w:gridCol w:w="9000"/></w:tblGrid>
      <w:tr>
        <w:tc>
          <w:tcPr>
            <w:tcW w:w="9000" w:type="dxa"/>
            %s
          </w:tcPr>
%s        </w:tc>
      </w:tr>
    </w:tbl>
    <w:p><w:pPr><w:spacing w:before="0" w:after="120"/></w:pPr></w:p>
`, borderWidthEighths, borderColor, fillXML, inner)
}

// renderPullQuote renders a large italic pulled quote with a thick accent bar.
func (w *docxWriter) renderPullQuote(n *html.Node) string {
	barColor := w.theme.AccentColor
	textColor := w.theme.QuoteColor

	var runs strings.Builder
	w.collectInlines(n, runStyle{italic: true, bold: false}, &runs)
	coloured := addColorToRuns(runs.String(), textColor)
	coloured = setRunSizeInRuns(coloured, 32) // 16pt (half-points)

	content := fmt.Sprintf(`          <w:p>
            <w:pPr>
              <w:spacing w:before="120" w:after="120"/>
              <w:rPr><w:i/></w:rPr>
            </w:pPr>%s
          </w:p>
`, coloured)

	return fmt.Sprintf(`    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="9000" w:type="dxa"/>
        <w:tblBorders>
          <w:left w:val="single" w:sz="36" w:space="0" w:color="%s"/>
        </w:tblBorders>
        <w:tblCellMar>
          <w:top w:w="160" w:type="dxa"/>
          <w:left w:w="360" w:type="dxa"/>
          <w:bottom w:w="160" w:type="dxa"/>
          <w:right w:w="240" w:type="dxa"/>
        </w:tblCellMar>
      </w:tblPr>
      <w:tblGrid><w:gridCol w:w="9000"/></w:tblGrid>
      <w:tr>
        <w:tc>
          <w:tcPr><w:tcW w:w="9000" w:type="dxa"/></w:tcPr>
%s        </w:tc>
      </w:tr>
    </w:tbl>
    <w:p><w:pPr><w:spacing w:before="0" w:after="120"/></w:pPr></w:p>
`, barColor, content)
}

func (w *docxWriter) renderCallout(n *html.Node, accent, _ string) string {
	fill := lightenHex(accent, 0.92)
	return w.renderStyledBox(n, accent, fill, 24)
}

func (w *docxWriter) renderKeyIdea(n *html.Node) string {
	accent := w.theme.AccentColor
	fill := lightenHex(accent, 0.88)
	return w.renderStyledBox(n, accent, fill, 40)
}

func (w *docxWriter) renderDefinition(n *html.Node) string {
	accent := w.theme.SecondaryColor
	fill := lightenHex(accent, 0.94)
	return w.renderStyledBox(n, accent, fill, 20)
}

func (w *docxWriter) renderSidebar(n *html.Node) string {
	muted := w.theme.MutedColor
	fill := lightenHex(muted, 0.9)
	return w.renderStyledBox(n, muted, fill, 12)
}

// renderFactBox wraps whatever block content n contains (usually an optional
// <strong>label</strong> plus a <table>) in a tinted shaded box.
func (w *docxWriter) renderFactBox(n *html.Node) string {
	accent := w.theme.AccentColor
	fill := lightenHex(accent, 0.94)
	return w.renderStyledBox(n, accent, fill, 16)
}

// renderStatBox draws a centered big-number + small-label card using the two
// optional spans <span class="stat-value"> and <span class="stat-label">. If
// neither is found it falls back to treating the first <p> as value and the
// rest as label.
func (w *docxWriter) renderStatBox(n *html.Node) string {
	value, label := extractStatContent(n)
	if value == "" && label == "" {
		// Fallback: treat the whole box as a short centered highlight.
		return w.renderStyledBox(n, w.theme.AccentColor, lightenHex(w.theme.AccentColor, 0.9), 16)
	}
	valueColor := w.theme.AccentColor
	labelColor := w.theme.MutedColor
	border := w.theme.AccentColor
	fill := lightenHex(border, 0.94)

	valueXML := ""
	if value != "" {
		valueXML = fmt.Sprintf(`          <w:p>
            <w:pPr>
              <w:jc w:val="center"/>
              <w:spacing w:before="80" w:after="40"/>
            </w:pPr>
            <w:r>
              <w:rPr>
                <w:rFonts w:ascii="%s" w:hAnsi="%s"/>
                <w:b/><w:sz w:val="56"/><w:color w:val="%s"/>
              </w:rPr>
              <w:t xml:space="preserve">%s</w:t>
            </w:r>
          </w:p>
`, w.theme.HeadingFont, w.theme.HeadingFont, valueColor, xmlEscape(value))
	}

	labelXML := ""
	if label != "" {
		labelXML = fmt.Sprintf(`          <w:p>
            <w:pPr>
              <w:jc w:val="center"/>
              <w:spacing w:before="0" w:after="80"/>
            </w:pPr>
            <w:r>
              <w:rPr><w:sz w:val="18"/><w:color w:val="%s"/></w:rPr>
              <w:t xml:space="preserve">%s</w:t>
            </w:r>
          </w:p>
`, labelColor, xmlEscape(label))
	}

	return fmt.Sprintf(`    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="9000" w:type="dxa"/>
        <w:tblBorders>
          <w:top w:val="single" w:sz="8" w:space="0" w:color="%[1]s"/>
          <w:bottom w:val="single" w:sz="8" w:space="0" w:color="%[1]s"/>
          <w:left w:val="single" w:sz="8" w:space="0" w:color="%[1]s"/>
          <w:right w:val="single" w:sz="8" w:space="0" w:color="%[1]s"/>
        </w:tblBorders>
      </w:tblPr>
      <w:tblGrid><w:gridCol w:w="9000"/></w:tblGrid>
      <w:tr>
        <w:tc>
          <w:tcPr>
            <w:tcW w:w="9000" w:type="dxa"/>
            <w:shd w:val="clear" w:color="auto" w:fill="%[2]s"/>
            <w:vAlign w:val="center"/>
          </w:tcPr>
%[3]s%[4]s        </w:tc>
      </w:tr>
    </w:tbl>
    <w:p><w:pPr><w:spacing w:before="0" w:after="120"/></w:pPr></w:p>
`, border, fill, valueXML, labelXML)
}

// extractStatContent pulls the text of <span class="stat-value"> and
// <span class="stat-label"> from n, if present.
func extractStatContent(n *html.Node) (value, label string) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Span {
			switch {
			case hasClass(n, "stat-value"):
				value = strings.TrimSpace(collectText(n))
				return
			case hasClass(n, "stat-label"):
				label = strings.TrimSpace(collectText(n))
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return value, label
}

// nodeHasVisibleText reports whether n contains at least one non-whitespace
// text-node descendant. Used to skip rendering styled-box frames around empty
// <div class="callout"></div> markers that would otherwise show as blank
// highlighted rectangles.
func nodeHasVisibleText(n *html.Node) bool {
	if n == nil {
		return false
	}
	var walk func(*html.Node) bool
	walk = func(node *html.Node) bool {
		if node.Type == html.TextNode && strings.TrimSpace(node.Data) != "" {
			return true
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	return walk(n)
}

func collectText(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// renderDividerDots emits a centered "• • •" row as a visual subsection
// separator. Used for <hr class="divider-dots"/>.
func (w *docxWriter) renderDividerDots() string {
	return fmt.Sprintf(`    <w:p>
      <w:pPr>
        <w:jc w:val="center"/>
        <w:spacing w:before="240" w:after="240"/>
      </w:pPr>
      <w:r>
        <w:rPr><w:color w:val="%s"/><w:sz w:val="28"/><w:spacing w:val="120"/></w:rPr>
        <w:t xml:space="preserve">•  •  •</w:t>
      </w:r>
    </w:p>
`, w.theme.MutedColor)
}

// setRunSizeInRuns rewrites every <w:rPr>…</w:rPr> to include a given size in
// half-points. Used by pullquote to bump the inline run sizes without
// re-collecting inlines.
func setRunSizeInRuns(runs string, halfPoints int) string {
	size := fmt.Sprintf(`<w:sz w:val="%d"/>`, halfPoints)
	return strings.ReplaceAll(runs, "<w:rPr>", "<w:rPr>"+size)
}
