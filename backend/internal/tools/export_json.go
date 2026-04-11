package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONExportTool writes structured data as a pretty-printed JSON file.
type JSONExportTool struct{}

func (JSONExportTool) Name() string        { return "export_json" }
func (JSONExportTool) Description() string {
	return "Write structured data as a pretty-printed JSON file to the workspace artifacts directory. " +
		"Saves to exports/<filename>.json."
}
func (JSONExportTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{"type": "string", "description": "Output filename without extension"},
			"data":     map[string]any{"type": "object", "description": "The JSON data to export"},
		},
		"required": []string{"filename", "data"},
	}
}

type exportJSONArgs struct {
	Filename string `json:"filename"`
	Data     any    `json:"data"`
}

func (t JSONExportTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args exportJSONArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}

	pretty, err := json.MarshalIndent(args.Data, "", "  ")
	if err != nil {
		return Err("marshal data: %v", err)
	}

	relPath := filepath.Join("exports", args.Filename+".json")
	abs, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Err("create exports directory: %v", err)
	}
	if err := os.WriteFile(abs, pretty, 0o644); err != nil {
		return Err("write json: %v", err)
	}

	return OKData(
		fmt.Sprintf("JSON exported to %s (%d bytes)", relPath, len(pretty)),
		map[string]any{"path": relPath, "size": len(pretty)},
	)
}
