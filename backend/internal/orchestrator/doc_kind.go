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
	magazineKindPattern = regexp.MustCompile(`(?i)\b(magazine|zine|editorial spread|editorial feature|photo essay|look ?book|mood ?board|visual essay|newsletter)\b`)
	textbookKindPattern = regexp.MustCompile(`(?i)\b(textbook|chapter|lesson|workbook|study guide|curriculum|course material|lecture notes|coursebook|module \d+)\b`)
	reportKindPattern   = regexp.MustCompile(`(?i)\b(report|whitepaper|white paper|brief|proposal|memo|analysis|case study|policy paper|rfc|research paper)\b`)
	articleKindPattern  = regexp.MustCompile(`(?i)\b(article|blog post|blog|op-?ed|essay|column)\b`)
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
All classes are RECOGNIZED by the renderer — use them freely.`
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
