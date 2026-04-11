package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SearchFilesTool searches file contents within the workspace for a pattern.
type SearchFilesTool struct{}

func (SearchFilesTool) Name() string        { return "search_files" }
func (SearchFilesTool) Description() string {
	return "Search for a text pattern in files within the workspace. " +
		"Returns matching lines with file path and line number. Limit: first 200 matches."
}
func (SearchFilesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern":    map[string]any{"type": "string", "description": "Text to search for (case-insensitive substring)"},
			"path":       map[string]any{"type": "string", "description": "Relative path to search in (default: '.')"},
			"extensions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "File extensions to include, e.g. [\".go\",\".md\"] (default: all text files)"},
		},
		"required": []string{"pattern"},
	}
}

type searchFilesArgs struct {
	Pattern    string   `json:"pattern"`
	Path       string   `json:"path"`
	Extensions []string `json:"extensions"`
}

type searchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

func (t SearchFilesTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args searchFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Pattern == "" {
		return Err("pattern is required")
	}
	if args.Path == "" {
		args.Path = "."
	}

	abs, err := scopedPath(ictx.WorkspaceRoot, args.Path)
	if err != nil {
		return Err("%v", err)
	}

	extSet := make(map[string]bool, len(args.Extensions))
	for _, e := range args.Extensions {
		extSet[strings.ToLower(e)] = true
	}

	patternLower := strings.ToLower(args.Pattern)
	var matches []searchMatch
	const maxMatches = 200

	_ = filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || len(matches) >= maxMatches {
			return nil
		}
		if len(extSet) > 0 && !extSet[strings.ToLower(filepath.Ext(p))] {
			return nil
		}

		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil // skip unreadable files silently
		}

		rel, _ := filepath.Rel(ictx.WorkspaceRoot, p)
		for lineNum, line := range strings.Split(string(data), "\n") {
			if len(matches) >= maxMatches {
				break
			}
			if strings.Contains(strings.ToLower(line), patternLower) {
				matches = append(matches, searchMatch{
					File:    rel,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
			}
		}
		return nil
	})

	if matches == nil {
		matches = []searchMatch{}
	}
	summary := fmt.Sprintf("Found %d matches for %q", len(matches), args.Pattern)
	return OKData(summary, matches)
}
