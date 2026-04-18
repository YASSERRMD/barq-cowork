package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/barq-cowork/barq-cowork/internal/generator"
)

// WriteHTMLPDFTool converts an HTML string (plus optional CSS override) to a
// high-fidelity A4 PDF using a headless Chromium print-to-PDF pipeline.
// The UAE AI Safety print profile is applied by default: 5mm Crimson Red
// header, 5mm Dark Green footer, and a tiled shield watermark on every page.
// A Chromium-compatible browser must be installed on the host system.
type WriteHTMLPDFTool struct{}

func (WriteHTMLPDFTool) Name() string { return "write_html_pdf" }

func (WriteHTMLPDFTool) Description() string {
	return "Convert an HTML document to a pixel-perfect A4 PDF using headless Chromium. " +
		"The output preserves all CSS geometry: the Crimson Red header bar, Dark Green footer bar, " +
		"and tiled shield watermark repeat on every page. " +
		"Use this when a visually exact, print-ready PDF is required (reports, briefs, proposals). " +
		"Google Chrome or Chromium must be installed on the host."
}

func (WriteHTMLPDFTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Output filename without extension, e.g. 'quarterly-report'",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Document title embedded in the HTML <title> tag",
			},
			"author": map[string]any{
				"type":        "string",
				"description": "Author name (optional, shown in cover page metadata)",
			},
			"html": map[string]any{
				"type":        "string",
				"description": "HTML content. Wrap the first page in <div class=\"cover-page\"> for an automatic page break after the cover. Use .info-box and .warning-box for callouts.",
			},
			"css": map[string]any{
				"type":        "string",
				"description": "Custom CSS override. Leave empty to use the built-in UAE AI Safety print profile.",
			},
		},
		"required": []string{"filename", "title", "html"},
	}
}

type writeHTMLPDFArgs struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	HTML     string `json:"html"`
	CSS      string `json:"css"`
}

func (t WriteHTMLPDFTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args writeHTMLPDFArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	if args.Title == "" {
		return Err("title is required")
	}
	if args.HTML == "" {
		return Err("html is required")
	}

	fname := sanitizeFilename(args.Filename, ".pdf")
	relPath := filepath.Join("documents", fname+".pdf")
	absPath, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Err("create documents directory: %v", err)
	}

	gen := generator.New()
	data, err := gen.ToPDF(ctx, generator.Request{
		HTML:   args.HTML,
		CSS:    args.CSS,
		Title:  args.Title,
		Author: args.Author,
	})
	if err != nil {
		return Err("pdf generation failed: %v", err)
	}

	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return Err("write pdf: %v", err)
	}

	return OKData(
		fmt.Sprintf("PDF written to %s (%d bytes)", relPath, len(data)),
		map[string]any{"path": relPath, "size": int64(len(data))},
	)
}
