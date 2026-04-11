package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MoveFileTool moves or renames a file/directory within the workspace.
// This is a DESTRUCTIVE action — it requires user approval.
type MoveFileTool struct{}

func (MoveFileTool) Name() string        { return "move_file" }
func (MoveFileTool) Description() string {
	return "Move or rename a file or directory within the workspace. " +
		"Requires approval because this operation modifies the filesystem layout."
}
func (MoveFileTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source":      map[string]any{"type": "string", "description": "Relative source path"},
			"destination": map[string]any{"type": "string", "description": "Relative destination path"},
		},
		"required": []string{"source", "destination"},
	}
}

type moveFileArgs struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

func (t MoveFileTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args moveFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}

	srcAbs, err := scopedPath(ictx.WorkspaceRoot, args.Source)
	if err != nil {
		return Err("source: %v", err)
	}
	dstAbs, err := scopedPath(ictx.WorkspaceRoot, args.Destination)
	if err != nil {
		return Err("destination: %v", err)
	}

	// Destructive — require approval before acting.
	action := fmt.Sprintf("Move %s → %s", args.Source, args.Destination)
	if ictx.RequireApproval != nil && !ictx.RequireApproval(ctx, action, argsJSON) {
		return Denied(action)
	}

	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		return Err("create destination directories: %v", err)
	}

	if err := os.Rename(srcAbs, dstAbs); err != nil {
		return Err("move file: %v", err)
	}

	return OK(fmt.Sprintf("Moved %s → %s", args.Source, args.Destination))
}
