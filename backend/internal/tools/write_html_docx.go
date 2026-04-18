package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barq-cowork/barq-cowork/internal/generator"
)

// WriteHTMLDocxTool converts an HTML string (plus optional CSS override) to a
// professional Word document (.docx) using the Pandoc pipeline with the UAE AI
// Safety print profile. Pandoc must be installed on the host system.
type WriteHTMLDocxTool struct{}

func (WriteHTMLDocxTool) Name() string { return "write_html_docx" }

func (WriteHTMLDocxTool) Description() string {
	return "Convert an HTML document (body fragment or full page) to a high-fidelity Word (.docx) file " +
		"using the UAE AI Safety visual theme: Crimson Red headings, Dark Green sub-headings, " +
		"Inter/Montserrat typography, and proper A4 page layout. " +
		"Use this when the request explicitly involves HTML content, rich formatting, or brand-specific styling. " +
		"Pandoc must be installed on the host."
}

func (WriteHTMLDocxTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Output filename without extension, e.g. 'ai-safety-report'",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Document title used in metadata and the HTML <title> tag",
			},
			"author": map[string]any{
				"type":        "string",
				"description": "Author name stored in document properties (optional)",
			},
			"html": map[string]any{
				"type":        "string",
				"description": "HTML body content. May be a fragment (<h1>…</h1><p>…</p>) or a full document. Use .cover-page, .info-box, .warning-box classes for styled callouts.",
			},
			"css": map[string]any{
				"type":        "string",
				"description": "Custom CSS override. Leave empty to use the built-in UAE AI Safety print profile.",
			},
		},
		"required": []string{"filename", "title", "html"},
	}
}

type writeHTMLDocxArgs struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	HTML     string `json:"html"`
	CSS      string `json:"css"`
}

func (t WriteHTMLDocxTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args writeHTMLDocxArgs
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

	fname := sanitizeFilename(args.Filename, ".docx")
	relPath := filepath.Join("documents", fname+".docx")
	absPath, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Err("create documents directory: %v", err)
	}

	gen := generator.New()
	data, err := gen.ToDocx(ctx, generator.Request{
		HTML:   args.HTML,
		CSS:    args.CSS,
		Title:  args.Title,
		Author: args.Author,
	})
	if err != nil {
		return Err("docx generation failed: %v", err)
	}

	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return Err("write docx: %v", err)
	}

	return OKData(
		fmt.Sprintf("Word document written to %s (%d bytes)", relPath, len(data)),
		map[string]any{"path": relPath, "size": int64(len(data))},
	)
}

// sanitizeFilename strips a suffix (if present) and replaces non-alphanumeric
// characters with hyphens to produce a safe file base-name.
func sanitizeFilename(name, stripSuffix string) string {
	name = strings.TrimSuffix(name, stripSuffix)
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
}
