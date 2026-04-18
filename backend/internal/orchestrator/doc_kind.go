package orchestrator

import (
	"regexp"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// docKind labels a document task so the planner + renderer can pick an
// appropriate cover design, per-section layout strategy, graphical vocabulary,
// and running header/footer treatment.
type docKind string

const (
	docKindMagazine docKind = "magazine" // zine, editorial, photo essay, lookbook, newsletter
	docKindTextbook docKind = "textbook" // textbook, chapter, lesson, workbook, curriculum
	docKindReport   docKind = "report"   // report, whitepaper, brief, proposal, memo, analysis
	docKindArticle  docKind = "article"  // article, essay, op-ed, blog post
)

var (
	magazineKindPattern = regexp.MustCompile(`(?i)\b(magazine|zine|editorial spread|editorial feature|photo essay|look ?book|mood ?board|visual essay|newsletter|catalog|catalogue|brochure|pamphlet|flyer|leaflet|programme|program booklet)\b`)
	textbookKindPattern = regexp.MustCompile(`(?i)\b(textbook|chapter|lesson plan|lesson|workbook|study guide|curriculum|course material|coursebook|lecture notes|syllabus|worksheet|module \d+|handbook|guidebook|manual)\b`)
	reportKindPattern   = regexp.MustCompile(`(?i)\b(report|whitepaper|white paper|briefing|brief|proposal|memorandum|memo|analysis|case study|policy paper|rfc|research paper|datasheet|spec sheet|specification|technical specification|product spec|business plan|market analysis|executive summary|status report|annual report|impact report|sustainability report|charter|playbook|runbook)\b`)
	articleKindPattern  = regexp.MustCompile(`(?i)\b(article|feature article|blog post|blog|op-?ed|opinion piece|essay|column|think piece|long read|longform|cover letter|letter|statement|manifesto)\b`)
)

// categorizeDocTask picks the best docKind for a task based on keywords in the
// title + description. Falls back to docKindReport for generic "document" asks.
func categorizeDocTask(task *domain.Task) docKind {
	if task == nil {
		return docKindReport
	}
	text := task.Title + " " + task.Description
	switch {
	case magazineKindPattern.MatchString(text):
		return docKindMagazine
	case textbookKindPattern.MatchString(text):
		return docKindTextbook
	case articleKindPattern.MatchString(text):
		return docKindArticle
	case reportKindPattern.MatchString(text):
		return docKindReport
	default:
		return docKindReport
	}
}

// docKindCoverGuidance returns the prompt snippet describing how the first
// page (cover_html) should look for the given kind.
func docKindCoverGuidance(kind docKind) string {
	switch kind {
	case docKindMagazine:
		return `- cover_html: design a STRIKING editorial opener that doubles as the magazine's cover. Pick one of these moves (don't cycle through them, choose what fits):
    • a huge kicker line + massive title + one-sentence deck
    • a volume/issue masthead line + three-word title + a single quoted tagline
    • a <table> with two columns acting as a bold two-panel cover
    • a large <blockquote> used as the sole cover element
    • a typographic "contents" list of the section titles as a poster-style cover
  NEVER reuse a generic <div class="cover-page"><h1 class="cover-title">…</h1></div> template. Use rich typography — ALL-CAPS kickers, oversized headings, short decks. End with <hr class="pagebreak"/>.`
	case docKindTextbook:
		return `- cover_html: design a TEXTBOOK title page. Include, in this order: edition/volume line (e.g. "FIRST EDITION" or "VOL. I"), the book/module title as <h1>, a subtitle/deck as <p>, the author(s), and a short "FOR:" audience line (who this is for). Optionally add a <div class="callout-info"> with a one-paragraph "About this book" abstract. End with <hr class="pagebreak"/>. No decorative chrome — textbooks are restrained.`
	case docKindArticle:
		return `- cover_html: design an ARTICLE opener — not a separate cover page. Put the title as <h1>, a 1-2 sentence deck in <p>, then a byline line ("By …") and a date/publication line. Optionally lead with a <div class="pullquote"> if the piece has a strong central claim. End with <hr class="pagebreak"/>.`
	default: // docKindReport
		return `- cover_html: design a REPORT title page. Include: publication/org line at top ("BARQ COWORK" or similar), the report title as <h1>, a subtitle/scope line, the author(s), a date / version line, and a short classification/confidentiality line if relevant. Optionally add a <div class="callout-info"> with a 2-3 sentence executive abstract. End with <hr class="pagebreak"/>. Keep it calm and authoritative — not flashy.`
	}
}

// docKindLayoutGuidance returns the prompt snippet describing how sections'
// layout_kind values should vary (or stay uniform) across the document.
func docKindLayoutGuidance(kind docKind) string {
	switch kind {
	case docKindMagazine:
		return `- "layout_kind" MUST be a distinct editorial layout for EACH section — every section uses a different one. Pick from: "hero-spread", "pull-quote-banner", "two-column-feature", "stat-grid", "sidebar-note", "timeline", "photo-essay-block", "checklist-grid", "caption-gallery", "cover-story", "callout-stack", "fact-box", "interview-qa", "index-list". Layout should match the section's content, not just cycle through options.`
	case docKindTextbook:
		return `- "layout_kind" should follow the textbook rhythm: early sections use "chapter-opening" (learning objectives + opener), middle sections use "worked-example", "definition-list", "key-idea-callout", "concept-map", or "diagram-plus-prose"; closing section uses "chapter-summary" or "review-questions". Keep the chrome consistent — every chapter has headers and footers.`
	case docKindArticle:
		return `- "layout_kind" should stay focused: mostly "prose" with one or two sections using "pull-quote-banner" or "sidebar-note" for emphasis. Don't over-decorate — articles earn their look through typography, not chrome.`
	default: // docKindReport
		return `- "layout_kind" should follow the report rhythm: "executive-summary" (first body section), then "findings-prose", "stat-grid", "fact-box", "callout-warn" (risks), "callout-info" (recommendations), "methodology", "appendix-table". Reuse layouts when content repeats — consistency signals rigor.`
	}
}

// docKindGraphicsGuidance returns the prompt snippet listing the graphical
// components available to the LLM. The renderer maps these class names into
// OOXML shaded boxes, colored borders, tinted fills, pull-quote styling.
func docKindGraphicsGuidance(kind docKind) string {
	base := `Available styled components — use them generously to make pages look like a real publication, not plain prose:
  • <div class="pullquote">One striking quote — short.</div>
    Large italic quote with accent-color left bar. Use 1-2 per document at most.
  • <div class="callout">Bold statement or highlight.</div>
    Shaded box, accent-color left border.
  • <div class="callout-info"><strong>Note.</strong> supporting context…</div>
    Info-toned box (link-color border).
  • <div class="callout-tip"><strong>Tip.</strong> practical advice…</div>
    Tip-toned box (secondary-color border).
  • <div class="callout-warn"><strong>Caution.</strong> risk or limitation…</div>
    Warning-toned box (amber border).
  • <div class="keyidea"><strong>Key idea.</strong> one-sentence summary of the big point.</div>
    Strong-border highlight — the reader's eye should land on this.
  • <div class="definition"><strong>Term.</strong> Precise definition in 1-2 sentences.</div>
    Inline defined term, left border.
  • <div class="statbox"><span class="stat-value">72%</span><span class="stat-label">caption</span></div>
    Big-number stat card. Use <span class="stat-value">…</span> for the number.
  • <div class="sidebar"><strong>Sidebar label.</strong> optional supporting detail.</div>
    Muted-tone tinted box — for background context.
  • <div class="factbox"><strong>Facts.</strong> 3-5 compact label → value rows as a <table>.</div>
  • <hr class="divider-dots"/>    decorative centered-dots divider between subsections.
  • <hr class="pagebreak"/>       page break.
All classes are RECOGNIZED by the renderer — use them freely.

NEVER emit an empty styled component. Every <div class="pullquote">, <div class="callout…">, <div class="keyidea">, <div class="definition">, <div class="statbox">, <div class="sidebar">, <div class="factbox"> MUST contain real text content. An empty <div class="callout"></div> or a callout with only whitespace renders as a blank highlighted rectangle and looks like broken layout. If you don't have content for a styled box, don't emit the div at all.`
	switch kind {
	case docKindMagazine:
		return base + `

Magazine preference: favor <div class="pullquote">, <div class="statbox">, oversized headings, and bold typographic moves. EVERY section must feel visibly different from the others.`
	case docKindTextbook:
		return base + `

Textbook preference: use <div class="definition"> for every technical term on first use, <div class="keyidea"> at the end of each major subsection, <div class="callout-tip"> and <div class="callout-warn"> to flag gotchas, and <div class="factbox"> for compact reference tables. Keep the treatment CONSISTENT across chapters.`
	case docKindArticle:
		return base + `

Article preference: one <div class="pullquote"> near the middle for emphasis, occasional <div class="callout-info"> for asides. Don't over-decorate — prose carries the piece.`
	default:
		return base + `

Report preference: <div class="callout-info"> for key observations, <div class="callout-warn"> for risks/limitations, <div class="keyidea"> for primary findings, <div class="statbox"> for headline metrics, <div class="factbox"> for reference data. Apply them CONSISTENTLY across sections.`
	}
}

// docKindWantsChrome returns true when the document kind should render with
// a consistent page header + footer on every page after the cover.
func docKindWantsChrome(kind docKind) bool {
	return kind != docKindMagazine
}

// docKindMathGuidance returns the LaTeX math-notation instruction block. The
// renderer converts LaTeX inside $…$ (inline) and $$…$$ (display) into native
// OOXML math (OMML), so Word renders equations with proper typography instead
// of garbled Unicode runs. Applies to every kind — any section with a variable,
// fraction, root, sum, integral, matrix, or Greek letter must use this.
func docKindMathGuidance(kind docKind) string {
	base := `Mathematical notation — ALWAYS use LaTeX, never raw Unicode math glyphs or prose-like "x squared":
  • Inline math: wrap in single dollars — "the slope $m$ equals $\frac{dy}{dx}$".
  • Display math: wrap in double dollars on its own line — "$$ \int_{0}^{1} x^{2}\,dx = \tfrac{1}{3} $$".
  • Variables: every single variable reference gets its own $…$ (e.g. $v$, $\theta$, $x_i$) so it renders italic and correctly spaced.
  • Fractions: \frac{num}{den}.  Roots: \sqrt{x}, \sqrt[3]{x}.
  • Sub/superscripts: x_{i}, e^{i\pi}, x_{i}^{2}.
  • Sums/products/integrals: \sum_{i=1}^{n} a_i, \prod_{k=1}^{n} k, \int_{a}^{b} f(x)\,dx, \oint, \iint.
  • Greek letters: \alpha, \beta, \gamma, \theta, \lambda, \mu, \pi, \sigma, \phi, \omega, \Delta, \Omega, etc.
  • Operators/relations: \times, \cdot, \pm, \leq, \geq, \neq, \approx, \equiv, \in, \subset, \to, \Rightarrow, \infty, \partial, \nabla.
  • Functions (rendered upright): \sin, \cos, \tan, \log, \ln, \exp, \lim, \max, \min.
  • Matrices: \begin{pmatrix} a & b \\ c & d \end{pmatrix}; also bmatrix, vmatrix, Vmatrix, cases.
  • Grouped delimiters: \left( \frac{a}{b} \right), \left[ … \right], \left\{ … \right\}.
NEVER write math as "x^2" in plain text, or use Unicode "α", "∑", "∫", "√" directly — always wrap in LaTeX delimiters. Escape a literal dollar sign in prose as "\$".`
	switch kind {
	case docKindTextbook:
		return base + `

Textbook preference: every definition, formula, worked example, and review question that references a symbol MUST use LaTeX. Worked examples should interleave prose paragraphs with display-math $$…$$ blocks showing each derivation step.`
	case docKindReport:
		return base + `

Report preference: use LaTeX for any quantitative formula, statistical notation, or model equation the report cites.`
	case docKindArticle:
		return base + `

Article preference: only reach for math when the subject genuinely needs it — but when it does, use LaTeX, never ad-hoc notation.`
	default: // magazine
		return base + `

Magazine preference: math is rare here, but if a feature includes a formula (science/data magazine), wrap it in LaTeX.`
	}
}

// sectionLayoutGuidance returns the per-kind section instructions — how to
// structure the HTML for a given layout_kind label from the plan.
func sectionLayoutGuidance(kind docKind, layoutKind string) string {
	switch kind {
	case docKindMagazine:
		return magazineSectionLayoutGuidance(layoutKind)
	case docKindTextbook:
		return textbookSectionLayoutGuidance(layoutKind)
	case docKindArticle:
		return articleSectionLayoutGuidance(layoutKind)
	default:
		return reportSectionLayoutGuidance(layoutKind)
	}
}

func magazineSectionLayoutGuidance(layoutKind string) string {
	return "Layout kind for this section: \"" + layoutKind + "\".\n" +
		"Render it as a distinctive editorial spread — NOT the same structure as a typical prose section:\n" +
		"  - \"hero-spread\":       oversized <h1>, short bold deck, 2-3 big paragraphs, one pulled subheading.\n" +
		"  - \"pull-quote-banner\": <div class=\"pullquote\">…</div> near the top, then 2-3 supporting paragraphs.\n" +
		"  - \"two-column-feature\":<h1>, intro paragraph, then a <table> with two cells acting as left/right columns of prose.\n" +
		"  - \"stat-grid\":         <h1>, short lede, then a <table> of 3-6 cells each containing <div class=\"statbox\"><span class=\"stat-value\">N</span><span class=\"stat-label\">label</span></div>.\n" +
		"  - \"sidebar-note\":      <h1>, main prose, then a <div class=\"sidebar\"> with label + highlighted note.\n" +
		"  - \"timeline\":          <h1>, short lede, then an <ol> of milestones (date + headline + 1-line detail).\n" +
		"  - \"photo-essay-block\": <h1>, short stanza-like paragraph, <div class=\"pullquote\"> caption, another paragraph.\n" +
		"  - \"checklist-grid\":    <h1>, short intro, a <table> of action items (Action | Owner | Status).\n" +
		"  - \"caption-gallery\":   <h1>, 3-4 captioned blocks (each caption in <div class=\"pullquote\">, each body in <p>).\n" +
		"  - \"cover-story\":       <h1>, bold lede paragraph, then a <table> with two columns (summary | key facts).\n" +
		"  - \"callout-stack\":     <h1>, 3 stacked <div class=\"callout\"> blocks, each followed by one paragraph.\n" +
		"  - \"fact-box\":          <h1>, short intro, then a <div class=\"factbox\"> containing a <table> of label → value facts.\n" +
		"  - \"interview-qa\":      <h1>, 3-5 Q&A pairs (Q in <strong>, A in paragraph).\n" +
		"  - \"index-list\":        <h1>, short intro, then an <ol> that acts as an annotated index.\n" +
		"Pick exactly the structure for the given layout_kind. Do NOT fall back to a generic 3-paragraph pattern."
}

func textbookSectionLayoutGuidance(layoutKind string) string {
	return "Layout kind for this section: \"" + layoutKind + "\".\n" +
		"Render it with textbook conventions — consistent across chapters:\n" +
		"  - \"chapter-opening\":    <h1>, a <div class=\"callout-info\"><strong>Learning objectives.</strong> …</div>, then 2-3 paragraphs introducing the topic.\n" +
		"  - \"worked-example\":     <h1>, problem setup paragraph, numbered <ol> of solution steps, then a <div class=\"keyidea\"><strong>Takeaway.</strong> …</div>.\n" +
		"  - \"definition-list\":    <h1>, short intro, then a sequence of <div class=\"definition\"><strong>Term.</strong> definition.</div> blocks (3-6).\n" +
		"  - \"key-idea-callout\":   <h1>, 2-3 paragraphs, then a <div class=\"keyidea\"> summarizing the point.\n" +
		"  - \"concept-map\":        <h1>, short lede, then a <div class=\"factbox\"> containing a <table> relating concepts to definitions or relationships.\n" +
		"  - \"diagram-plus-prose\": <h1>, 2-3 paragraphs, then a <div class=\"sidebar\"> with a figure caption / diagram description.\n" +
		"  - \"chapter-summary\":    <h1>, 2-3 summary paragraphs, then a <div class=\"keyidea\"> restating the chapter's takeaways, then an <ol> of review points.\n" +
		"  - \"review-questions\":   <h1>, short intro, then an <ol> of 5-10 review/practice questions.\n" +
		"Include at least one <div class=\"definition\"> or <div class=\"keyidea\"> in every section."
}

func reportSectionLayoutGuidance(layoutKind string) string {
	return "Layout kind for this section: \"" + layoutKind + "\".\n" +
		"Render it with report/whitepaper conventions — consistent and authoritative:\n" +
		"  - \"executive-summary\": <h1>, a <div class=\"callout-info\"><strong>Summary.</strong> 3-5 sentences.</div>, then 2-3 paragraphs expanding the key findings.\n" +
		"  - \"findings-prose\":    <h1>, 3-5 paragraphs of analysis, with one <div class=\"keyidea\"> highlighting the main finding.\n" +
		"  - \"stat-grid\":         <h1>, short lede, then a <table> of 3-6 stat cards using <div class=\"statbox\"><span class=\"stat-value\">N</span><span class=\"stat-label\">label</span></div>.\n" +
		"  - \"fact-box\":          <h1>, short intro, then a <div class=\"factbox\"> with a <table> of label → value reference facts.\n" +
		"  - \"callout-info\":      <h1>, 2-3 paragraphs, then a <div class=\"callout-info\"><strong>Recommendation.</strong> …</div>.\n" +
		"  - \"callout-warn\":      <h1>, 2-3 paragraphs, then a <div class=\"callout-warn\"><strong>Risk / limitation.</strong> …</div>.\n" +
		"  - \"methodology\":       <h1>, a paragraph of scope, then an <ol> of methodology steps, then a <div class=\"sidebar\"> with data caveats.\n" +
		"  - \"appendix-table\":    <h1>, short preamble, then a <table> of reference data.\n" +
		"Include at least one styled box (callout / keyidea / statbox / factbox) per section."
}

func articleSectionLayoutGuidance(layoutKind string) string {
	return "Layout kind for this section: \"" + layoutKind + "\".\n" +
		"Articles are prose-first — use styled boxes sparingly:\n" +
		"  - \"prose\":             <h1>, 4-6 paragraphs of well-crafted prose.\n" +
		"  - \"pull-quote-banner\": <h1>, intro paragraph, a <div class=\"pullquote\"> with the piece's sharpest line, then 2-3 closing paragraphs.\n" +
		"  - \"sidebar-note\":      <h1>, main prose, then a <div class=\"sidebar\"> with supporting context.\n" +
		"Use at most one <div class=\"pullquote\"> per section. Prefer prose over chrome."
}
