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

// WriteHTMLDocxTool converts an HTML body + LLM-designed theme to a Word
// (.docx) file. The theme (fonts + colors) is supplied per call by the caller,
// so the visual identity of every document is decided by the LLM rather than
// baked into the renderer.
type WriteHTMLDocxTool struct{}

func (WriteHTMLDocxTool) Name() string { return "write_html_docx" }

func (WriteHTMLDocxTool) Description() string {
	return "Convert an HTML document (body fragment or full page) to a styled Word (.docx) file. " +
		"The caller supplies a `theme` object that decides fonts and colors — headings, body, " +
		"accent, link, etc. — so every document gets a visual design chosen for the subject. " +
		"Use this tool for any HTML-backed Word document; omit theme to fall back to a neutral palette."
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
				"description": "HTML body content. May be a fragment (<h1>…</h1><p>…</p>) or a full document. Use inline styles or per-page <section class=\"page page-<kind>\">…</section> blocks for varied layouts.",
			},
			"css": map[string]any{
				"type":        "string",
				"description": "Reserved for future inline stylesheet injection. Safe to leave empty.",
			},
			"theme": map[string]any{
				"type":        "object",
				"description": "Visual theme. All fields optional; missing values fall back to neutral defaults. Colors are 6-digit hex without '#'. heading_font / body_font / mono_font are font family names. Design the theme around the document's subject — don't reuse a template.",
				"properties": map[string]any{
					"name":            map[string]any{"type": "string"},
					"heading_font":    map[string]any{"type": "string"},
					"body_font":       map[string]any{"type": "string"},
					"mono_font":       map[string]any{"type": "string"},
					"body_color":      map[string]any{"type": "string"},
					"heading1_color":  map[string]any{"type": "string"},
					"heading2_color":  map[string]any{"type": "string"},
					"heading3_color":  map[string]any{"type": "string"},
					"heading4_color":  map[string]any{"type": "string"},
					"accent_color":    map[string]any{"type": "string"},
					"secondary_color": map[string]any{"type": "string"},
					"link_color":      map[string]any{"type": "string"},
					"quote_color":     map[string]any{"type": "string"},
					"muted_color":     map[string]any{"type": "string"},
					"code_bg_color":   map[string]any{"type": "string"},
					"title_color":     map[string]any{"type": "string"},
				},
			},
		},
		"required": []string{"filename", "title", "html"},
	}
}

type writeHTMLDocxArgs struct {
	Filename string              `json:"filename"`
	Title    string              `json:"title"`
	Author   string              `json:"author"`
	HTML     string              `json:"html"`
	CSS      string              `json:"css"`
	Theme    *generator.DocxTheme `json:"theme"`
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
		Theme:  args.Theme,
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
