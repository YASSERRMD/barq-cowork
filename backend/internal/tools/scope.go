package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// scopedPath resolves target relative to root and verifies the result
// stays within root. Returns the absolute path or an error if the path
// would escape the workspace boundary.
func scopedPath(root, target string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("workspace root is not set; cannot access filesystem")
	}

	abs := target
	if !filepath.IsAbs(target) {
		abs = filepath.Join(root, target)
	}

	// Clean to remove ./ ../ symlink tricks
	abs = filepath.Clean(abs)
	cleanRoot := filepath.Clean(root)

	if !strings.HasPrefix(abs, cleanRoot+string(filepath.Separator)) && abs != cleanRoot {
		return "", fmt.Errorf(
			"path %q is outside workspace root %q — operation rejected",
			target, root,
		)
	}
	return abs, nil
}
