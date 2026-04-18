package generator

import (
	"fmt"
	"strings"
)

// buildFullHTML wraps an HTML fragment (or a full document body) in the
// standard UAE AI Safety print shell, injecting the header/footer bars and
// the watermark overlay. If req.CSS is non-empty it replaces AISafetyCSS.
func buildFullHTML(req Request) string {
	css := req.CSS
	if css == "" {
		css = AISafetyCSS
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1.0"/>
<title>%s</title>
<style>
%s
</style>
</head>
<body>
<div class="page-header" aria-hidden="true"></div>
<div class="page-footer" aria-hidden="true"></div>
<div class="watermark"   aria-hidden="true"></div>
<main class="document-content">
%s
</main>
</body>
</html>`, htmlEsc(req.Title), css, req.HTML)
}

// htmlEsc escapes the minimal set of characters required in an HTML title.
func htmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}
