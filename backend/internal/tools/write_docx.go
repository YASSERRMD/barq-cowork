package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocxTool generates a professional Word document via a native Go OOXML builder.
type DocxTool struct{}

func (DocxTool) Name() string { return "write_docx" }
func (DocxTool) Description() string {
	return "Create a professional Word document (.docx) with headings, body text, bullet lists, and tables. " +
		"Use this for reports, proposals, briefs, papers, and any formal document request."
}

func (DocxTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Output filename without extension, e.g. 'market-analysis'",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Document title shown on the cover page",
			},
			"subtitle": map[string]any{
				"type":        "string",
				"description": "Optional subtitle shown below the title",
			},
			"author": map[string]any{
				"type":        "string",
				"description": "Author name shown on the cover page",
			},
			"sections": map[string]any{
				"type":        "array",
				"description": "Document sections in order",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"heading": map[string]any{
							"type":        "string",
							"description": "Section heading text",
						},
						"level": map[string]any{
							"type":        "integer",
							"description": "Heading level: 1 (major section) or 2 (sub-section)",
							"default":     1,
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Section body text. Use '• item' prefix for bullet list items (one per line).",
						},
						"table": map[string]any{
							"type":        "object",
							"description": "Optional table for this section",
							"properties": map[string]any{
								"headers": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "string"},
								},
								"rows": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "array"},
								},
							},
						},
					},
					"required": []string{"heading", "content"},
				},
			},
		},
		"required": []string{"filename", "title", "sections"},
	}
}

type docxArgs struct {
	Filename string        `json:"filename"`
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle"`
	Author   string        `json:"author"`
	Sections []docxSection `json:"sections"`
}

type docxSection struct {
	Heading string     `json:"heading"`
	Level   int        `json:"level"`
	Content string     `json:"content"`
	Table   *docxTable `json:"table,omitempty"`
}

type docxTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

func (t DocxTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args docxArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	if args.Title == "" {
		return Err("title is required")
	}
	if len(args.Sections) == 0 {
		return Err("sections are required")
	}

	// Default level 1 for sections missing it
	for i := range args.Sections {
		if args.Sections[i].Level == 0 {
			args.Sections[i].Level = 1
		}
	}

	// Sanitize filename
	fname := strings.TrimSuffix(args.Filename, ".docx")
	fname = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, fname)

	relPath := filepath.Join("documents", fname+".docx")
	absPath, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Err("create documents directory: %v", err)
	}

	data, err := buildDOCX(args)
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
