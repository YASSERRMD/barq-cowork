package generator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// htmlToDocx converts the HTML content in req to a .docx byte slice using
// Pandoc. A UAE AI Safety reference.docx is generated on-the-fly and passed
// via --reference-doc so all Word paragraph styles are applied automatically.
//
// Pandoc must be installed and present in PATH.
func htmlToDocx(ctx context.Context, req Request) ([]byte, error) {
	if err := checkPandoc(); err != nil {
		return nil, err
	}

	// ── temp directory ────────────────────────────────────────────────────
	dir, err := os.MkdirTemp("", "barq-docx-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// ── write input HTML ──────────────────────────────────────────────────
	htmlPath := filepath.Join(dir, "input.html")
	if err := os.WriteFile(htmlPath, []byte(buildFullHTML(req)), 0o600); err != nil {
		return nil, fmt.Errorf("write input html: %w", err)
	}

	// ── generate reference DOCX ───────────────────────────────────────────
	refBytes, err := buildReferenceDocx()
	if err != nil {
		return nil, fmt.Errorf("build reference docx: %w", err)
	}
	refPath := filepath.Join(dir, "reference.docx")
	if err := os.WriteFile(refPath, refBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write reference docx: %w", err)
	}

	// ── output path ───────────────────────────────────────────────────────
	outPath := filepath.Join(dir, "output.docx")

	// ── run Pandoc ────────────────────────────────────────────────────────
	//   --from html+raw_html  preserves <div>, custom classes, etc.
	//   --to docx             OOXML output
	//   --reference-doc       applies our UAE AI Safety Word styles
	cmd := exec.CommandContext(ctx, "pandoc",
		"--from", "html+raw_html",
		"--to", "docx",
		"--reference-doc", refPath,
		"--output", outPath,
		htmlPath,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pandoc: %w\n%s", err, string(out))
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read pandoc output: %w", err)
	}
	return data, nil
}

// checkPandoc returns a human-friendly error when pandoc is not in PATH.
func checkPandoc() error {
	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf(
			"pandoc not found in PATH — install it from https://pandoc.org/installing.html",
		)
	}
	return nil
}
