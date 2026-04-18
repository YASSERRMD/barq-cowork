package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// soffice candidates on macOS (Homebrew), Linux, and Windows installs.
var sofficeCandidates = []string{
	"soffice",
	"libreoffice",
	"/Applications/LibreOffice.app/Contents/MacOS/soffice",
	"/opt/homebrew/bin/soffice",
	"/usr/local/bin/soffice",
	"/usr/bin/soffice",
	"/usr/bin/libreoffice",
}

// resolveSoffice returns the path to the LibreOffice CLI or an error listing
// where it was searched.
func resolveSoffice() (string, error) {
	for _, c := range sofficeCandidates {
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	return "", errors.New("LibreOffice not found — install it (brew install --cask libreoffice) " +
		"or add soffice to PATH")
}

// runLibreOffice invokes `soffice --headless --convert-to <filter> <input>`
// writing the output into outDir. A 120s timeout is applied.
func runLibreOffice(ctx context.Context, inputPath, outDir, filter string) error {
	bin, err := resolveSoffice()
	if err != nil {
		return err
	}

	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin,
		"--headless",
		"--norestore",
		"--nolockcheck",
		"--nofirststartwizard",
		"--convert-to", filter,
		"--outdir", outDir,
		inputPath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("libreoffice convert (%s) failed: %v — stderr: %s",
			filter, err, stderr.String())
	}
	return nil
}
