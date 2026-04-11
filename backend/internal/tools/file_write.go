package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileTool writes (creates or overwrites) a file within the workspace.
type WriteFileTool struct{}

func (WriteFileTool) Name() string        { return "write_file" }
func (WriteFileTool) Description() string { return "Write text content to a file within the workspace. Creates parent directories if needed. Overwrites existing files." }
func (WriteFileTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Relative file path within workspace"},
			"content": map[string]any{"type": "string", "description": "Text content to write"},
		},
		"required": []string{"path", "content"},
	}
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t WriteFileTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args writeFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}

	abs, err := scopedPath(ictx.WorkspaceRoot, args.Path)
	if err != nil {
		return Err("%v", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Err("create parent directories: %v", err)
	}

	if err := os.WriteFile(abs, []byte(args.Content), 0o644); err != nil {
		return Err("write file: %v", err)
	}

	return OK(fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), args.Path))
}
