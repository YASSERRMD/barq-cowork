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

// WriteSlidesTool creates a self-contained HTML presentation file.
type WriteSlidesTool struct{}

func (WriteSlidesTool) Name() string        { return "write_html_slides" }
func (WriteSlidesTool) Description() string {
	return "Create a beautiful self-contained HTML slide presentation. " +
		"Each slide is separated by '---'. Supports markdown-style headings and bullet points. " +
		"Saves to slides/<filename>.html and can be opened directly in any browser."
}
func (WriteSlidesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{"type": "string", "description": "Output filename without extension, e.g. 'ai-trends-slides'"},
			"title":    map[string]any{"type": "string", "description": "Presentation title shown on the first slide"},
			"slides":   map[string]any{
				"type":        "array",
				"description": "Array of slide objects",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"heading": map[string]any{"type": "string", "description": "Slide heading/title"},
						"points":  map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Bullet points or paragraphs for this slide",
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
	Filename string  `json:"filename"`
	Title    string  `json:"title"`
	Slides   []slide `json:"slides"`
}

type slide struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
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

	ts := time.Now().UTC().Format("2006-01-02")
	html := buildSlidesHTML(args.Title, ts, args.Slides)

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
		fmt.Sprintf("Slides written to %s (%d slides, %d bytes)", relPath, len(args.Slides), len(html)),
		map[string]any{"path": relPath, "size": len(html)},
	)
}

func buildSlidesHTML(title, date string, slides []slide) string {
	var sb strings.Builder

	// Build slide HTML blocks
	var slideBlocks strings.Builder
	// Title slide
	slideBlocks.WriteString(fmt.Sprintf(`
		<div class="slide active" id="slide-0">
			<div class="slide-content title-slide">
				<h1>%s</h1>
				<p class="subtitle">%s</p>
			</div>
		</div>`, escapeHTML(title), escapeHTML(date)))

	for i, s := range slides {
		var pointsHTML strings.Builder
		if len(s.Points) > 0 {
			pointsHTML.WriteString("<ul>")
			for _, p := range s.Points {
				pointsHTML.WriteString(fmt.Sprintf("<li>%s</li>", escapeHTML(p)))
			}
			pointsHTML.WriteString("</ul>")
		}
		slideBlocks.WriteString(fmt.Sprintf(`
		<div class="slide" id="slide-%d">
			<div class="slide-content">
				<h2>%s</h2>
				%s
			</div>
		</div>`, i+1, escapeHTML(s.Heading), pointsHTML.String()))
	}

	total := len(slides) + 1 // +1 for title slide

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
         background: #0f1117; color: #e2e8f0; height: 100vh; overflow: hidden; }
  .slideshow { width: 100vw; height: 100vh; position: relative; }
  .slide { position: absolute; inset: 0; display: none; align-items: center;
           justify-content: center; padding: 60px; }
  .slide.active { display: flex; }
  .slide-content { max-width: 900px; width: 100%%; }
  .title-slide { text-align: center; }
  .title-slide h1 { font-size: clamp(2rem, 5vw, 3.5rem); font-weight: 700;
                     background: linear-gradient(135deg, #818cf8, #a78bfa);
                     -webkit-background-clip: text; -webkit-text-fill-color: transparent;
                     line-height: 1.15; margin-bottom: 16px; }
  .title-slide .subtitle { color: #64748b; font-size: 1.1rem; }
  h2 { font-size: clamp(1.5rem, 3vw, 2.2rem); font-weight: 600; color: #a5b4fc;
       margin-bottom: 28px; padding-bottom: 12px;
       border-bottom: 2px solid rgba(165,180,252,0.2); }
  ul { list-style: none; display: flex; flex-direction: column; gap: 14px; }
  li { display: flex; align-items: flex-start; gap: 12px; font-size: clamp(1rem, 2vw, 1.2rem);
       color: #cbd5e1; line-height: 1.5; }
  li::before { content: "▸"; color: #818cf8; font-size: 0.9em; margin-top: 3px; flex-shrink: 0; }
  .controls { position: fixed; bottom: 24px; left: 50%%; transform: translateX(-50%%);
              display: flex; align-items: center; gap: 16px; z-index: 10; }
  .btn { background: rgba(129,140,248,0.15); border: 1px solid rgba(129,140,248,0.3);
         color: #a5b4fc; padding: 8px 20px; border-radius: 8px; font-size: 14px;
         cursor: pointer; transition: all 150ms; }
  .btn:hover { background: rgba(129,140,248,0.25); }
  .btn:disabled { opacity: 0.3; cursor: default; }
  .counter { font-size: 13px; color: #475569; min-width: 60px; text-align: center; }
  .progress { position: fixed; bottom: 0; left: 0; height: 3px;
              background: linear-gradient(90deg, #818cf8, #a78bfa);
              transition: width 300ms ease; }
</style>
</head>
<body>
<div class="slideshow">
%s
</div>
<div class="controls">
  <button class="btn" id="prev" onclick="go(-1)">← Prev</button>
  <span class="counter" id="counter">1 / %d</span>
  <button class="btn" id="next" onclick="go(1)">Next →</button>
</div>
<div class="progress" id="progress"></div>
<script>
  var cur = 0, total = %d;
  function go(dir) {
    document.getElementById('slide-'+cur).classList.remove('active');
    cur = Math.max(0, Math.min(total-1, cur+dir));
    document.getElementById('slide-'+cur).classList.add('active');
    document.getElementById('counter').textContent = (cur+1)+' / '+total;
    document.getElementById('prev').disabled = cur === 0;
    document.getElementById('next').disabled = cur === total-1;
    document.getElementById('progress').style.width = ((cur+1)/total*100)+'%%';
  }
  document.addEventListener('keydown', function(e) {
    if (e.key==='ArrowRight'||e.key===' ') go(1);
    if (e.key==='ArrowLeft') go(-1);
  });
  go(0);
</script>
</body>
</html>`, escapeHTML(title), slideBlocks.String(), total, total))

	return sb.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}
