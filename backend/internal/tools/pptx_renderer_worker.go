package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed assets/pptx-renderer.cjs
var embeddedPPTXRenderer []byte

var (
	pptxRendererOnce sync.Once
	pptxRendererPath string
	pptxRendererErr  error
)

func buildPPTXWithRenderer(ctx context.Context, manifest []byte) ([]byte, error) {
	scriptPath, err := ensurePPTXRendererScript()
	if err != nil {
		return nil, err
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node runtime is required for pptx rendering: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "barq-pptx-render-*")
	if err != nil {
		return nil, fmt.Errorf("create renderer temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "presentation.pptx")
	cmd := exec.CommandContext(ctx, nodePath, scriptPath, "--output", outputPath)
	cmd.Stdin = bytes.NewReader(manifest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pptx renderer failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read renderer output: %w", err)
	}
	return injectPPTXPreviewManifest(data, manifest)
}

func ensurePPTXRendererScript() (string, error) {
	pptxRendererOnce.Do(func() {
		if len(embeddedPPTXRenderer) == 0 {
			pptxRendererErr = fmt.Errorf("embedded pptx renderer bundle is empty")
			return
		}
		sum := sha256.Sum256(embeddedPPTXRenderer)
		name := fmt.Sprintf("barq-pptx-renderer-%x.cjs", sum[:8])
		target := filepath.Join(os.TempDir(), name)

		current, err := os.ReadFile(target)
		if err == nil && bytes.Equal(current, embeddedPPTXRenderer) {
			pptxRendererPath = target
			return
		}
		if err := os.WriteFile(target, embeddedPPTXRenderer, 0o755); err != nil {
			pptxRendererErr = fmt.Errorf("write embedded pptx renderer: %w", err)
			return
		}
		pptxRendererPath = target
	})
	return pptxRendererPath, pptxRendererErr
}

func injectPPTXPreviewManifest(pptxData, manifest []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(pptxData), int64(len(pptxData)))
	if err != nil {
		return nil, fmt.Errorf("open pptx zip: %w", err)
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	hasManifest := false

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("open zip entry %s: %w", file.Name, err)
		}
		payload, err := readAll(rc)
		rc.Close()
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("read zip entry %s: %w", file.Name, err)
		}

		if file.Name == pptxPreviewManifestPath {
			payload = manifest
			hasManifest = true
		}
		if file.Name == "[Content_Types].xml" {
			payload = []byte(ensureJSONContentType(string(payload)))
		}

		w, err := zw.Create(file.Name)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("copy zip entry %s: %w", file.Name, err)
		}
		if _, err := w.Write(payload); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write zip entry %s: %w", file.Name, err)
		}
	}

	if !hasManifest {
		w, err := zw.Create(pptxPreviewManifestPath)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create manifest entry: %w", err)
		}
		if _, err := w.Write(manifest); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write manifest entry: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize pptx zip: %w", err)
	}
	return out.Bytes(), nil
}

func ensureJSONContentType(content string) string {
	if strings.Contains(content, `Extension="json"`) {
		return content
	}
	defaultXML := `<Default Extension="xml" ContentType="application/xml"/>`
	defaultJSON := `
  <Default Extension="json" ContentType="application/json"/>`
	if strings.Contains(content, defaultXML) {
		return strings.Replace(content, defaultXML, defaultXML+defaultJSON, 1)
	}
	if strings.Contains(content, "</Types>") {
		return strings.Replace(content, "</Types>", defaultJSON+"\n</Types>", 1)
	}
	return content
}
