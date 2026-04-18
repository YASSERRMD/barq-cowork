// Package generator converts LLM-generated HTML+CSS content into professional
// A4 Word (.docx) and PDF files with the UAE AI Safety visual identity.
//
// DOCX pipeline: HTML → Pandoc (--reference-doc) → .docx
// PDF  pipeline: HTML → headless Chromium (print-to-PDF) → .pdf
//
// External runtime dependencies:
//   - pandoc   (https://pandoc.org/installing.html) — for DOCX generation
//   - Chromium/Google Chrome                        — for PDF generation
package generator

import "context"

// Request carries the content and metadata for a document generation job.
type Request struct {
	// HTML is the document body — either a full HTML document or a fragment.
	// The generator wraps fragments in the UAE AI Safety print shell automatically.
	HTML string

	// CSS overrides AISafetyCSS when non-empty.
	// Pass an empty string to use the built-in UAE AI Safety print profile.
	CSS string

	// Title is embedded in the HTML <title> and DOCX core metadata.
	Title string

	// Author is stored in DOCX document properties (optional).
	Author string

	// Theme drives fonts and colors in styles.xml / theme1.xml / numbering.xml.
	// Nil means "use neutral defaults" — nothing about the visual identity is
	// baked into the OOXML package itself.
	Theme *DocxTheme

	// Chrome adds a running header/footer to every page after the cover.
	// Nil means "no chrome" — magazine / zine kinds use this so each page can
	// be visually unique.
	Chrome *DocxChrome

	// Background paints every page with a tinted full-page fill. Nil means
	// "plain white pages". Only set this when the user explicitly asks for
	// background graphics / tinted pages — otherwise the default white is the
	// right choice for most documents.
	Background *DocxBackground
}

// DocxBackground configures a full-page tinted background. Word renders this
// via a single <w:background> element + <w:displayBackgroundShape/> in
// settings.xml. The fill applies to every page including the cover.
type DocxBackground struct {
	Color string // 6-digit hex without '#', e.g. "F5F1E8" for warm cream
}

// DocxChrome configures the running header and footer.
// Leave fields blank to omit that half of the chrome — an empty HeaderText
// suppresses the header entirely; an empty FooterText with ShowPageNum=false
// suppresses the footer entirely. The cover page is always suppressed via
// <w:titlePg/>.
type DocxChrome struct {
	HeaderText  string // e.g. "Document Title — 2026 Report"
	FooterText  string // e.g. "Barq Cowork · Confidential"
	ShowPageNum bool   // right-aligned "Page N" in the footer
}

// Generator converts HTML requests to DOCX and PDF outputs.
// All methods are safe for concurrent use.
type Generator struct{}

// New returns a ready-to-use Generator.
func New() *Generator { return &Generator{} }

// ToDocx converts req to a .docx byte slice via Pandoc with the UAE AI Safety
// reference document. Pandoc must be installed and available in PATH.
func (g *Generator) ToDocx(ctx context.Context, req Request) ([]byte, error) {
	return htmlToDocx(ctx, req)
}

// ToPDF converts req to a PDF byte slice via headless Chromium.
// A Chromium-compatible browser must be installed on the host system.
func (g *Generator) ToPDF(ctx context.Context, req Request) ([]byte, error) {
	return htmlToPDF(ctx, req)
}
