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
