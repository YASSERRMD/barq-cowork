package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ListFilesTool lists files and directories within a workspace-scoped path.
type ListFilesTool struct{}

func (ListFilesTool) Name() string { return "list_files" }
func (ListFilesTool) Description() string {
	return "List files and directories at a given path within the workspace. " +
		"Set recursive=true to walk subdirectories."
}
func (ListFilesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":      map[string]any{"type": "string", "description": "Relative path within workspace (default: '.')"},
			"recursive": map[string]any{"type": "boolean", "description": "Walk subdirectories (default: false)"},
		},
		"required": []string{},
	}
}

type listFilesArgs struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
}

type fileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

func (t ListFilesTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args listFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Path == "" {
		args.Path = "."
	}

	abs, err := scopedPath(ictx.WorkspaceRoot, args.Path)
	if err != nil {
		return Err("%v", err)
	}

	var entries []fileEntry

	if args.Recursive {
		err = filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
			if err != nil || p == abs {
				return err
			}
			rel, _ := filepath.Rel(ictx.WorkspaceRoot, p)
			info, _ := d.Info()
			var size int64
			if info != nil && !d.IsDir() {
				size = info.Size()
			}
			entries = append(entries, fileEntry{Name: d.Name(), Path: rel, IsDir: d.IsDir(), Size: size})
			return nil
		})
	} else {
		dirEntries, e := os.ReadDir(abs)
		err = e
		for _, d := range dirEntries {
			rel, _ := filepath.Rel(ictx.WorkspaceRoot, filepath.Join(abs, d.Name()))
			info, _ := d.Info()
			var size int64
			if info != nil && !d.IsDir() {
				size = info.Size()
			}
			entries = append(entries, fileEntry{Name: d.Name(), Path: rel, IsDir: d.IsDir(), Size: size})
		}
	}

	if err != nil {
		return Err("list files: %v", err)
	}
	if entries == nil {
		entries = []fileEntry{}
	}

	summary := fmt.Sprintf("Listed %d entries in %s", len(entries), args.Path)
	return OKData(summary, entries)
}
