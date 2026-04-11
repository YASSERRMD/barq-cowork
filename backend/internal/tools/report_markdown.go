package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownReportTool writes a formatted markdown report to the workspace.
type MarkdownReportTool struct{}

func (MarkdownReportTool) Name() string        { return "write_markdown_report" }
func (MarkdownReportTool) Description() string {
	return "Write a structured markdown report file to the workspace artifacts directory. " +
		"Automatically adds a title, timestamp header, and saves to reports/<filename>.md."
}
func (MarkdownReportTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{"type": "string", "description": "Output filename without extension, e.g. 'analysis-report'"},
			"title":    map[string]any{"type": "string", "description": "Report title"},
			"content":  map[string]any{"type": "string", "description": "Full markdown content body"},
		},
		"required": []string{"filename", "title", "content"},
	}
}

type markdownReportArgs struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

func (t MarkdownReportTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args markdownReportArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	// Auto-derive title from filename if not provided
	if args.Title == "" {
		args.Title = strings.ReplaceAll(strings.TrimSuffix(args.Filename, ".md"), "-", " ")
		args.Title = strings.ReplaceAll(args.Title, "_", " ")
	}

	// Build report with header
	ts := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	full := fmt.Sprintf("# %s\n\n_Generated: %s_\n\n---\n\n%s\n", args.Title, ts, args.Content)

	relPath := filepath.Join("reports", args.Filename+".md")
	abs, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Err("create reports directory: %v", err)
	}
	if err := os.WriteFile(abs, []byte(full), 0o644); err != nil {
		return Err("write report: %v", err)
	}

	return OKData(
		fmt.Sprintf("Report written to %s (%d bytes)", relPath, len(full)),
		map[string]any{"path": relPath, "size": len(full)},
	)
}
