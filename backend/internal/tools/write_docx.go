package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DocxTool generates a professional Word document via a Python subprocess (python-docx).
type DocxTool struct{}

func (DocxTool) Name() string        { return "write_docx" }
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
	Filename string       `json:"filename"`
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle"`
	Author   string       `json:"author"`
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

	// Build payload for Python script
	payload, err := json.Marshal(map[string]any{
		"title":    args.Title,
		"subtitle": args.Subtitle,
		"author":   args.Author,
		"date":     time.Now().UTC().Format("January 2, 2006"),
		"sections": args.Sections,
	})
	if err != nil {
		return Err("marshal payload: %v", err)
	}

	// Try Python subprocess first
	data, pyErr := buildDocxViaPython(ctx, payload)
	if pyErr != nil {
		return Err("docx generation failed: %v", pyErr)
	}

	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return Err("write docx: %v", err)
	}

	return OKData(
		fmt.Sprintf("Word document written to %s (%d bytes)", relPath, len(data)),
		map[string]any{"path": relPath, "size": int64(len(data))},
	)
}

func buildDocxViaPython(ctx context.Context, payload []byte) ([]byte, error) {
	scriptPath, err := findDocxScript()
	if err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("python error: %v — %s", err, stderr.String())
	}
	if len(out) < 100 {
		return nil, fmt.Errorf("python returned too-small output (%d bytes); stderr: %s", len(out), stderr.String())
	}
	return out, nil
}

func findDocxScript() (string, error) {
	exe, _ := os.Executable()
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "scripts", "gen_docx.py"),
		filepath.Join("scripts", "gen_docx.py"),
	}
	// Walk up directories looking for scripts/gen_docx.py
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		candidates = append(candidates, filepath.Join(dir, "scripts", "gen_docx.py"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("gen_docx.py not found in any candidate path")
}
