package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const maxReadBytes = 1024 * 1024 // 1 MB safety cap

// ReadFileTool reads the content of a file within the workspace.
type ReadFileTool struct{}

func (ReadFileTool) Name() string        { return "read_file" }
func (ReadFileTool) Description() string { return "Read the text content of a file within the workspace. Returns an error for binary files exceeding 1 MB." }
func (ReadFileTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Relative file path within workspace"},
		},
		"required": []string{"path"},
	}
}

type readFileArgs struct {
	Path string `json:"path"`
}

func (t ReadFileTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args readFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}

	abs, err := scopedPath(ictx.WorkspaceRoot, args.Path)
	if err != nil {
		return Err("%v", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return Err("file not found: %s", args.Path)
	}
	if info.IsDir() {
		return Err("%s is a directory, not a file", args.Path)
	}
	if info.Size() > maxReadBytes {
		return Err("file too large (%d bytes > 1MB limit): %s", info.Size(), args.Path)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return Err("read file: %v", err)
	}

	content := string(data)
	summary := fmt.Sprintf("Read %d bytes from %s", len(data), args.Path)
	return OKData(summary, map[string]any{"path": args.Path, "content": content, "size": len(data)})
}
