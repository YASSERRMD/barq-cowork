package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// a4WidthIn and a4HeightIn are A4 paper dimensions in inches, as required by
// the Chrome DevTools Protocol PrintToPDF parameters.
const (
	a4WidthIn  = 8.2677 // 210 mm
	a4HeightIn = 11.693 // 297 mm
)

// htmlToPDF renders req.HTML to a PDF byte slice using headless Chromium.
// Chromium (or Google Chrome) must be installed; chromedp will locate it
// automatically via the standard exec-allocator search paths.
//
// The CSS @page { margin: 0 } rule combined with position:fixed header/footer
// bars in AISafetyCSS ensures the coloured bars and watermark appear on every
// printed page.
func htmlToPDF(ctx context.Context, req Request) ([]byte, error) {
	// ── write full HTML to a temp file ────────────────────────────────────
	// Using file:// is more reliable than data: URIs for large documents and
	// ensures relative asset paths (if any) resolve correctly.
	dir, err := os.MkdirTemp("", "barq-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	htmlPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(buildFullHTML(req)), 0o600); err != nil {
		return nil, fmt.Errorf("write html: %w", err)
	}
	fileURL := "file://" + htmlPath

	// ── chromedp exec allocator ───────────────────────────────────────────
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-software-rasterizer", true),
		// Allow loading local file:// resources from the temp dir
		chromedp.Flag("allow-file-access-from-files", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// ── navigate and print ────────────────────────────────────────────────
	var pdfBuf []byte

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(fileURL),
		// Wait until the DOM (including fixed bars) is fully ready.
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var runErr error
			pdfBuf, _, runErr = page.PrintToPDF().
				WithPaperWidth(a4WidthIn).
				WithPaperHeight(a4HeightIn).
				WithPrintBackground(true).    // required for coloured bars + watermark
				WithMarginTop(0).             // @page margin:0 owns all spacing
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				WithPreferCSSPageSize(true).  // honour @page { size: A4 }
				Do(ctx)
			return runErr
		}),
	); err != nil {
		return nil, fmt.Errorf("chromedp print-to-pdf: %w", err)
	}

	return pdfBuf, nil
}
