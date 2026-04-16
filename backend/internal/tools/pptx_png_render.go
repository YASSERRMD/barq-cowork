package tools

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// pptxPNGRenderer converts a .pptx file into one PNG per slide using LibreOffice
// headless (PPTX → PDF) and pdftoppm (PDF → PNGs). The result is the list of
// absolute PNG file paths in slide order.
//
// Caller is expected to have soffice and pdftoppm on PATH. If either is absent,
// the function returns an error and the caller should fall back to HTML preview.
//
// Rendered pages are cached in <workspaceCache>/pptx-previews/<sha1>/ keyed by a
// hash of the PPTX bytes. A subsequent request for the same .pptx reuses the
// cached images rather than re-running LibreOffice.

// renderPPTXToPNGs converts a .pptx at pptxPath to PNGs and returns their paths.
// cacheRoot is the directory under which converted images will be stored
// (typically ~/.barq-cowork/pptx-previews). It is created if missing.
// dpi is the rasterization density (100 = fast preview, 150 = sharper).
func renderPPTXToPNGs(pptxPath, cacheRoot string, dpi int) ([]string, error) {
	if dpi <= 0 {
		dpi = 120
	}
	data, err := os.ReadFile(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("read pptx: %w", err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:16])

	outDir := filepath.Join(cacheRoot, digest)
	if pngs, ok := cachedPNGPages(outDir); ok {
		return pngs, nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	sofficeBin, err := findSofficeBinary()
	if err != nil {
		return nil, err
	}

	// LibreOffice uses a user profile directory. Give it a private one so parallel
	// runs don't collide.
	profileDir := filepath.Join(outDir, ".lo-profile")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, fmt.Errorf("create libreoffice profile dir: %w", err)
	}

	pdfPath := filepath.Join(outDir, "deck.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		convCmd := exec.Command(
			sofficeBin,
			"--headless",
			"--norestore",
			"--nologo",
			"--nofirststartwizard",
			"-env:UserInstallation=file://"+profileDir,
			"--convert-to", "pdf",
			"--outdir", outDir,
			pptxPath,
		)
		convCmd.Env = append(os.Environ(), "HOME="+profileDir)
		output, err := runWithTimeout(convCmd, 60*time.Second)
		if err != nil {
			return nil, fmt.Errorf("libreoffice convert: %w (output: %s)", err, string(output))
		}

		// LibreOffice names the output by the input basename.
		candidate := filepath.Join(outDir, strings.TrimSuffix(filepath.Base(pptxPath), filepath.Ext(pptxPath))+".pdf")
		if _, err := os.Stat(candidate); err == nil && candidate != pdfPath {
			if err := os.Rename(candidate, pdfPath); err != nil {
				return nil, fmt.Errorf("rename converted pdf: %w", err)
			}
		}
		if _, err := os.Stat(pdfPath); err != nil {
			return nil, fmt.Errorf("libreoffice did not produce a PDF at %s (output: %s)", pdfPath, string(output))
		}
	}

	rasterizeCmd := exec.Command(
		"pdftoppm",
		"-png",
		"-r", fmt.Sprintf("%d", dpi),
		pdfPath,
		filepath.Join(outDir, "slide"),
	)
	if output, err := runWithTimeout(rasterizeCmd, 60*time.Second); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (output: %s)", err, string(output))
	}

	pngs, ok := cachedPNGPages(outDir)
	if !ok {
		return nil, fmt.Errorf("no PNG pages produced in %s", outDir)
	}
	return pngs, nil
}

func cachedPNGPages(dir string) ([]string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	var pngs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "slide-") || !strings.HasSuffix(name, ".png") {
			continue
		}
		pngs = append(pngs, filepath.Join(dir, name))
	}
	if len(pngs) == 0 {
		return nil, false
	}
	// pdftoppm names slides slide-1.png, slide-2.png, ... slide-10.png.
	// Lexicographic order handles 1..9 correctly but breaks at 10, so sort by
	// parsed page number instead.
	sort.SliceStable(pngs, func(i, j int) bool {
		return pageNumberFromPNG(pngs[i]) < pageNumberFromPNG(pngs[j])
	})
	return pngs, true
}

func pageNumberFromPNG(path string) int {
	base := filepath.Base(path)
	trimmed := strings.TrimSuffix(strings.TrimPrefix(base, "slide-"), ".png")
	var n int
	fmt.Sscanf(trimmed, "%d", &n)
	return n
}

func findSofficeBinary() (string, error) {
	for _, candidate := range []string{
		"soffice",
		"/opt/homebrew/bin/soffice",
		"/usr/local/bin/soffice",
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
	} {
		if candidate == "soffice" {
			if path, err := exec.LookPath("soffice"); err == nil {
				return path, nil
			}
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("libreoffice (soffice) not found on PATH or standard install locations")
}

// previewPPTXAsPNGStack converts a .pptx to PNGs via LibreOffice + pdftoppm and
// returns a self-contained HTML page that stacks each slide image full-width.
// Returns ("", false, nil) when LibreOffice/pdftoppm aren't available so callers
// can fall back to the Reveal manifest preview. Any other error is returned as
// ("", false, err).
func previewPPTXAsPNGStack(pptxPath, cacheRoot string) (string, bool, error) {
	pngs, err := renderPPTXToPNGs(pptxPath, cacheRoot, 140)
	if err != nil {
		// Missing binaries should degrade gracefully; real failures propagate.
		msg := err.Error()
		if strings.Contains(msg, "soffice") || strings.Contains(msg, "pdftoppm") || strings.Contains(msg, "libreoffice") {
			return "", false, nil
		}
		return "", false, err
	}
	if len(pngs) == 0 {
		return "", false, nil
	}
	var sections strings.Builder
	for i, png := range pngs {
		data, err := os.ReadFile(png)
		if err != nil {
			continue
		}
		b64 := base64.StdEncoding.EncodeToString(data)
		sections.WriteString(fmt.Sprintf(
			`<figure style="margin:0 0 18px;display:flex;justify-content:center"><img src="data:image/png;base64,%s" alt="Slide %d" style="width:min(100%%,1440px);height:auto;display:block;border-radius:18px;box-shadow:0 28px 80px rgba(2,6,23,0.35);background:#000" loading="lazy"/></figure>`,
			b64, i+1,
		))
	}
	doc := `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Presentation preview</title><style>html,body{margin:0;padding:0;background:#07111f;min-height:100%;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,sans-serif;color:#f8fafc}main{max-width:1480px;margin:0 auto;padding:18px}</style></head><body><main>` + sections.String() + `</main></body></html>`
	return doc, true, nil
}

// defaultPPTXPreviewCacheRoot returns ~/.barq-cowork/pptx-previews, or
// /tmp/barq-cowork/pptx-previews if the home dir cannot be determined. The
// directory is lazily created by renderPPTXToPNGs.
func defaultPPTXPreviewCacheRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".barq-cowork", "pptx-previews")
	}
	return filepath.Join(os.TempDir(), "barq-cowork", "pptx-previews")
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	done := make(chan struct{})
	var (
		out []byte
		err error
	)
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
		return out, err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return out, fmt.Errorf("command timed out after %s", timeout)
	}
}
