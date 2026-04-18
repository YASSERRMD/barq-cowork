package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// htmlToDocx converts an HTML document to .docx using headless LibreOffice.
// LibreOffice (`soffice`) must be installed on the host system. The HTML is
// written to a temporary file together with the UAE AI Safety print CSS and
// converted via `--convert-to docx:"MS Word 2007 XML"`.
func htmlToDocx(ctx context.Context, req Request) ([]byte, error) {
	fullHTML := buildFullHTML(req)

	workDir, err := os.MkdirTemp("", "barq-docx-*")
	if err != nil {
		return nil, fmt.Errorf("tempdir: %w", err)
	}
	defer os.RemoveAll(workDir)

	htmlPath := filepath.Join(workDir, "source.html")
	if err := os.WriteFile(htmlPath, []byte(fullHTML), 0o644); err != nil {
		return nil, fmt.Errorf("write html: %w", err)
	}

	if err := runLibreOffice(ctx, htmlPath, workDir, "docx:MS Word 2007 XML"); err != nil {
		return nil, err
	}

	outPath := filepath.Join(workDir, "source.docx")
	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read docx output: %w", err)
	}
	return data, nil
}
