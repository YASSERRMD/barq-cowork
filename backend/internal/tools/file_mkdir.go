package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// CreateFolderTool creates a directory (and parents) within the workspace.
type CreateFolderTool struct{}

func (CreateFolderTool) Name() string        { return "create_folder" }
func (CreateFolderTool) Description() string { return "Create a directory (and any missing parent directories) within the workspace." }
func (CreateFolderTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Relative path of the directory to create"},
		},
		"required": []string{"path"},
	}
}

type mkdirArgs struct {
	Path string `json:"path"`
}

func (t CreateFolderTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args mkdirArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}

	abs, err := scopedPath(ictx.WorkspaceRoot, args.Path)
	if err != nil {
		return Err("%v", err)
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return Err("create folder: %v", err)
	}

	return OK(fmt.Sprintf("Created folder: %s", args.Path))
}
