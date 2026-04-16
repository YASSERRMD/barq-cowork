package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteSlidesTool creates a self-contained Reveal.js HTML presentation file.
type WriteSlidesTool struct{}

func (WriteSlidesTool) Name() string        { return "write_html_slides" }
func (WriteSlidesTool) Description() string {
	return "Create a beautiful self-contained HTML slide presentation powered by Reveal.js. " +
		"Supports rich content per slide: headings, bullet points, speaker notes, and multiple themes. " +
		"Saves to slides/<filename>.html and can be opened directly in any browser. " +
		"Use this for HTML/browser-based presentations. For real .pptx PowerPoint files use write_pptx instead."
}
func (WriteSlidesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "Output filename without extension, e.g. 'ai-trends-slides'",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Presentation title shown on the cover slide",
			},
			"subtitle": map[string]any{
				"type":        "string",
				"description": "Optional subtitle or tagline for the cover slide",
			},
			"theme": map[string]any{
				"type":        "string",
				"description": "Reveal.js theme name: black (default), white, league, beige, sky, night, serif, simple, solarized, moon, dracula",
				"enum":        []string{"black", "white", "league", "beige", "sky", "night", "serif", "simple", "solarized", "moon", "dracula"},
			},
			"slides": map[string]any{
				"type":        "array",
				"description": "Array of slide objects",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"heading": map[string]any{
							"type":        "string",
							"description": "Slide heading/title",
						},
						"points": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Bullet points that appear as fragments (one by one on click)",
						},
						"body": map[string]any{
							"type":        "string",
							"description": "Optional paragraph text shown below the heading",
						},
						"note": map[string]any{
							"type":        "string",
							"description": "Speaker notes for this slide (not visible to the audience)",
						},
					},
					"required": []string{"heading"},
				},
			},
		},
		"required": []string{"filename", "title", "slides"},
	}
}

type slideArgs struct {
	Filename string       `json:"filename"`
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle"`
	Theme    string       `json:"theme"`
	Slides   []revealSlide `json:"slides"`
}

type revealSlide struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
	Body    string   `json:"body"`
	Note    string   `json:"note"`
}

func (t WriteSlidesTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args slideArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Err("invalid arguments: %v", err)
	}
	if args.Filename == "" {
		return Err("filename is required")
	}
	if args.Title == "" {
		args.Title = strings.ReplaceAll(args.Filename, "-", " ")
	}
	if args.Theme == "" {
		args.Theme = "black"
	}

	ts := time.Now().UTC().Format("2006-01-02")
	html := buildRevealHTML(args.Title, args.Subtitle, ts, args.Theme, args.Slides)

	relPath := filepath.Join("slides", args.Filename+".html")
	abs, err := scopedPath(ictx.WorkspaceRoot, relPath)
	if err != nil {
		return Err("%v", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Err("create slides directory: %v", err)
	}
	if err := os.WriteFile(abs, []byte(html), 0o644); err != nil {
		return Err("write slides: %v", err)
	}

	return OKData(
		fmt.Sprintf("Reveal.js presentation written to %s (%d slides, %d bytes)", relPath, len(args.Slides)+1, len(html)),
		map[string]any{"path": relPath, "size": len(html)},
	)
}

func buildRevealHTML(title, subtitle, date, theme string, slides []revealSlide) string {
	var sections strings.Builder

	// Cover slide
	coverSubtitle := subtitle
	if coverSubtitle == "" {
		coverSubtitle = date
	}
	sections.WriteString(fmt.Sprintf(`
		<section>
			<h1>%s</h1>
			<p class="subtitle">%s</p>
		</section>`, escapeHTML(title), escapeHTML(coverSubtitle)))

	// Content slides
	for _, s := range slides {
		var inner strings.Builder

		inner.WriteString(fmt.Sprintf("<h2>%s</h2>\n", escapeHTML(s.Heading)))

		if s.Body != "" {
			inner.WriteString(fmt.Sprintf("<p>%s</p>\n", escapeHTML(s.Body)))
		}

		if len(s.Points) > 0 {
			inner.WriteString("<ul>\n")
			for _, p := range s.Points {
				inner.WriteString(fmt.Sprintf(`<li class="fragment">%s</li>`+"\n", escapeHTML(p)))
			}
			inner.WriteString("</ul>\n")
		}

		note := ""
		if s.Note != "" {
			note = fmt.Sprintf("\n\t\t\t<aside class=\"notes\">%s</aside>", escapeHTML(s.Note))
		}

		sections.WriteString(fmt.Sprintf(`
		<section>
			%s%s
		</section>`, inner.String(), note))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>%s</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reset.css">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reveal.css">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@5/dist/theme/%s.css">
  <style>
    .reveal h1 { font-size: 2.2em; }
    .reveal h2 { font-size: 1.6em; }
    .reveal .subtitle { opacity: 0.7; font-size: 1em; margin-top: 0.5em; }
    .reveal ul { list-style: none; padding: 0; }
    .reveal ul li { padding: 0.3em 0 0.3em 1.4em; position: relative; }
    .reveal ul li::before { content: "▸"; position: absolute; left: 0; opacity: 0.6; }
    .reveal p { font-size: 0.9em; opacity: 0.85; }
  </style>
</head>
<body>
  <div class="reveal">
    <div class="slides">%s
    </div>
  </div>
  <script type="module">
    import Reveal from 'https://cdn.jsdelivr.net/npm/reveal.js@5/dist/reveal.esm.js';
    const deck = new Reveal({
      hash: true,
      slideNumber: 'c/t',
      transition: 'slide',
      backgroundTransition: 'fade',
      controls: true,
      progress: true,
      center: true,
      plugins: [],
    });
    deck.initialize();
  </script>
</body>
</html>`, escapeHTML(title), theme, sections.String())
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}
