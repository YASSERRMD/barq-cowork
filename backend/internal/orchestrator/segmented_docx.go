package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/generator"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/google/uuid"
)

// Long-form document tasks (e.g. "10 page", "at least 8 pages") cannot be
// produced in a single tool call by GLM-5.1 / similar models — the model gives
// up and emits prose-only planning turns. This file implements a segmented
// HTML-document workflow that mirrors segmented_pptx.go: a plan call drafts a
// per-section outline, then one LLM call per section produces the HTML for
// that section, and finally the composed HTML is sent to write_html_docx.

const segmentedSectionInterCallDelay = 3 * time.Second

// requestedDocumentPagePattern matches phrases like "10 pages", "10-page",
// "at least 8 pages".
var requestedDocumentPagePattern = regexp.MustCompile(`(?i)\b(\d+)\s*[- ]?\s*pages?\b`)

// magazineModePattern catches requests that imply per-page visual variation.
var magazineModePattern = regexp.MustCompile(`(?i)\b(magazine|zine|editorial spread|photo essay|look ?book|mood ?board|visual essay|newsletter)\b`)

type segmentedDocPlan struct {
	Filename  string               `json:"filename"`
	Title     string               `json:"title"`
	Subtitle  string               `json:"subtitle"`
	Author    string               `json:"author"`
	CoverHTML string               `json:"cover_html"`
	Theme     *generator.DocxTheme `json:"theme"`
	Sections  []segmentedDocBrief  `json:"sections"`
}

type segmentedDocBrief struct {
	Heading    string `json:"heading"`
	Brief      string `json:"brief"`
	Depth      string `json:"depth"`       // "short" | "medium" | "long"
	LayoutKind string `json:"layout_kind"` // magazine mode: distinct per section
}

type segmentedDocSection struct {
	Heading string `json:"heading"`
	HTML    string `json:"html"`
}

// shouldUseSegmentedDocumentWorkflow reports whether the task should run
// through the segmented HTML-document pipeline.
func shouldUseSegmentedDocumentWorkflow(task *domain.Task) bool {
	_, ok := requestedDocumentSectionCount(task)
	return ok
}

// isMagazineModeTask returns true when the task text asks for a document with
// visibly distinct per-page layouts (magazine / zine / editorial spread / …).
func isMagazineModeTask(task *domain.Task) bool {
	if task == nil {
		return false
	}
	return magazineModePattern.MatchString(task.Title + " " + task.Description)
}

// requestedDocumentSectionCount returns the number of sections the LLM should
// produce and whether a page hint was found in the task text. 1 page → 1
// section; N pages → N sections (bounded to [3, 25]).
func requestedDocumentSectionCount(task *domain.Task) (int, bool) {
	if requiredOutputTool(task) != "write_html_docx" {
		return 0, false
	}
	text := strings.TrimSpace(task.Title + " " + task.Description)
	m := requestedDocumentPagePattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, false
	}
	if n > 25 {
		n = 25
	}
	if n < 3 {
		n = 3
	}
	return n, true
}

func (a *AgentLoop) runSegmentedDocumentWorkflow(
	ctx context.Context,
	task *domain.Task,
	workspaceRoot string,
	planID string,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) ExecuteResult {
	sectionCount, ok := requestedDocumentSectionCount(task)
	if !ok {
		sectionCount = 6
	}
	magazineMode := isMagazineModeTask(task)

	var result ExecuteResult
	taskStart := time.Now()
	now := time.Now().UTC()

	// ── PLAN PHASE ──────────────────────────────────────────────────────────
	planStep := &domain.PlanStep{
		ID:          uuid.NewString(),
		PlanID:      planID,
		Order:       1,
		Title:       "Plan document outline",
		Description: fmt.Sprintf("Plan the full %d-section document before drafting.", sectionCount),
		Status:      domain.StepStatusRunning,
		StartedAt:   &now,
	}
	_ = a.plans.CreateStep(ctx, planStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
		"step_id": planStep.ID, "tool": "document_plan", "mode": "segmented",
	})

	var plan segmentedDocPlan
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			return result
		}
		planCallStart := time.Now()
		a.logger.Info("segmented docx: calling plan LLM",
			"task_id", task.ID, "attempt", attempt+1,
			"sections", sectionCount, "provider", a.prov.Name(), "model", a.cfg.Model)

		var planErr error
		plan, planErr = a.generateSegmentedDocPlan(ctx, task, sectionCount, magazineMode, extraSystemPrompts, runtimeProfile)
		elapsed := time.Since(planCallStart).Round(time.Millisecond)
		if planErr == nil && len(plan.Sections) > 0 {
			normalizeSegmentedDocPlan(&plan, task, sectionCount)
			a.logger.Info("segmented docx: plan LLM succeeded",
				"task_id", task.ID, "attempt", attempt+1, "elapsed", elapsed,
				"title", plan.Title, "sections_planned", len(plan.Sections))
			break
		}
		if planErr == nil {
			planErr = fmt.Errorf("plan returned zero sections")
		}
		a.logger.Warn("segmented docx: plan LLM failed — retrying",
			"task_id", task.ID, "attempt", attempt+1, "elapsed", elapsed,
			"retry_delay", segmentedSlideRetryDelay, "error", planErr)
		select {
		case <-time.After(segmentedSlideRetryDelay):
		case <-ctx.Done():
			return result
		}
	}

	planDone := time.Now().UTC()
	planStep.CompletedAt = &planDone
	planStep.Status = domain.StepStatusCompleted
	planStep.ToolOutput = marshalJSON(map[string]any{"status": "ok", "sections": len(plan.Sections), "title": plan.Title})
	_ = a.plans.UpdateStep(ctx, planStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
		"step_id": planStep.ID, "tool": "document_plan", "status": planStep.Status,
	})

	// ── SECTION PHASE ───────────────────────────────────────────────────────
	sections := make([]segmentedDocSection, len(plan.Sections))
	stepOrder := 1
	for i, brief := range plan.Sections {
		stepOrder++
		stepStarted := time.Now().UTC()
		step := &domain.PlanStep{
			ID:          uuid.NewString(),
			PlanID:      planID,
			Order:       stepOrder,
			Title:       fmt.Sprintf("Draft section %d", i+1),
			Description: fmt.Sprintf(`Generate section %d of %d: "%s".`, i+1, len(plan.Sections), firstNonEmptyString(brief.Heading, "Untitled section")),
			Status:      domain.StepStatusRunning,
			StartedAt:   &stepStarted,
		}
		_ = a.plans.CreateStep(ctx, step)
		a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
			"step_id": step.ID, "tool": "document_section", "section": i + 1,
		})

		if i > 0 {
			select {
			case <-time.After(segmentedSectionInterCallDelay):
			case <-ctx.Done():
			}
		}

		var section segmentedDocSection
		for attempt := 0; ; attempt++ {
			if ctx.Err() != nil {
				return result
			}
			callStart := time.Now()
			a.logger.Info("segmented docx: calling section LLM",
				"task_id", task.ID, "section", i+1, "of", len(plan.Sections),
				"heading", brief.Heading, "attempt", attempt+1,
				"elapsed_total", time.Since(taskStart).Round(time.Second))

			raw, sErr := a.generateSegmentedDocSection(ctx, task, plan, brief, i, magazineMode, runtimeProfile)
			sElapsed := time.Since(callStart).Round(time.Millisecond)

			if sErr == nil && strings.TrimSpace(raw.HTML) == "" {
				sErr = fmt.Errorf("section LLM returned empty HTML")
			}
			if sErr == nil {
				raw.Heading = firstNonEmptyString(raw.Heading, brief.Heading)
				section = raw
				a.logger.Info("segmented docx: section LLM succeeded",
					"task_id", task.ID, "section", i+1, "attempt", attempt+1,
					"elapsed", sElapsed, "html_bytes", len(section.HTML))
				break
			}
			a.logger.Warn("segmented docx: section LLM failed — retrying",
				"task_id", task.ID, "section", i+1, "attempt", attempt+1,
				"elapsed", sElapsed, "retry_delay", segmentedSlideRetryDelay, "error", sErr)
			select {
			case <-time.After(segmentedSlideRetryDelay):
			case <-ctx.Done():
				return result
			}
		}
		sections[i] = section

		stepDone := time.Now().UTC()
		step.CompletedAt = &stepDone
		step.Status = domain.StepStatusCompleted
		step.ToolOutput = marshalJSON(map[string]any{"status": "ok", "heading": section.Heading, "html_bytes": len(section.HTML)})
		_ = a.plans.UpdateStep(ctx, step)
		a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
			"step_id": step.ID, "tool": "document_section", "status": step.Status, "section": i + 1,
		})
	}

	// ── COMPOSE + RENDER ────────────────────────────────────────────────────
	composed := composeSegmentedDocHTML(plan, sections)
	renderArgs := map[string]any{
		"filename": firstNonEmptyString(plan.Filename, slugifySegmentedFilename(task.Title)),
		"title":    firstNonEmptyString(plan.Title, task.Title, "Document"),
		"author":   plan.Author,
		"html":     composed,
	}
	if plan.Theme != nil {
		renderArgs["theme"] = plan.Theme
	}
	argsBytes, _ := json.Marshal(renderArgs)
	tc := provider.ToolCall{ID: "segmented-write-html-docx", Name: "write_html_docx", Arguments: string(argsBytes)}
	renderStep := createToolPlanStep(planID, stepOrder+1, tc)
	renderStep.Title = "Render Word document"
	renderStep.Description = fmt.Sprintf("Compose %d sections into the final .docx.", len(sections))
	_ = a.plans.CreateStep(ctx, renderStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
		"step_id": renderStep.ID, "tool": tc.Name, "mode": "segmented",
	})

	toolOutput, stepErr := a.executeToolCall(ctx, tc, task, workspaceRoot)
	renderDone := time.Now().UTC()
	renderStep.CompletedAt = &renderDone
	if stepErr != nil {
		renderStep.Status = domain.StepStatusFailed
		renderStep.ToolOutput = marshalError(stepErr)
		result.Failed++
	} else {
		renderStep.Status = domain.StepStatusCompleted
		renderStep.ToolOutput = toolOutput
		result.Completed++
		a.maybeRecordAgentArtifact(ctx, renderStep, task, workspaceRoot)
	}
	_ = a.plans.UpdateStep(ctx, renderStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
		"step_id": renderStep.ID, "tool": tc.Name, "status": renderStep.Status,
	})
	return result
}

// ─── LLM calls ──────────────────────────────────────────────────────────────

func (a *AgentLoop) generateSegmentedDocPlan(
	ctx context.Context,
	task *domain.Task,
	sectionCount int,
	magazineMode bool,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) (segmentedDocPlan, error) {
	var out segmentedDocPlan
	contextText := compactSegmentedExtraContext(extraSystemPrompts)

	layoutGuidance := `- "layout_kind" is a short kebab-case label that describes the HTML layout of that section (e.g. "prose", "two-column", "stat-grid", "timeline", "checklist-table", "callout-heavy"). Keep consistent when depth is uniform.`
	coverGuidance := `- cover_html: design the opening page around the subject. You pick the HTML — it can be a bold headline + deck, a hero statement with a masthead line, a fact strip, a stacked poster-style intro, a manifesto paragraph, a typographic cover, etc. Do NOT default to <div class="cover-page"> with a title and a subtitle every time. Compose the first page the way a designer would, using <h1>, <h2>, <p>, <blockquote>, <table>, <hr> as needed. End the cover with <hr class="pagebreak"/> so it occupies its own page.`
	if magazineMode {
		layoutGuidance = `- "layout_kind" MUST be a distinct editorial layout for each section — EVERY section uses a different one. Pick from: "hero-spread", "pull-quote-banner", "two-column-feature", "stat-grid", "sidebar-note", "timeline", "photo-essay-block", "checklist-grid", "caption-gallery", "cover-story", "callout-stack", "fact-box", "interview-qa", "index-list". The layout should match the section's content, not just cycle through options.`
		coverGuidance = `- cover_html: design a STRIKING editorial opener that doubles as the magazine's cover. Examples (pick whichever fits — don't just cycle): a giant kicker line + massive title + one-sentence deck; a volume/issue masthead line + three-word title + quoted tagline; a <table> with two columns acting as a bold two-panel cover; a large <blockquote> as the only visible text; a typographic "index" of the section titles as a contents page. NEVER reuse the generic <div class="cover-page"><h1 class="cover-title">…</h1></div> template. End with <hr class="pagebreak"/> so the cover is its own page.`
	}

	user := fmt.Sprintf(`Plan a Word document that will be rendered through write_html_docx.

Task title: %s
Task details: %s
Additional context: %s

Required: exactly %d body sections in sections[].

Return ONLY compact valid JSON with this exact shape:
{
  "filename":"kebab-case-name",
  "title":"document title",
  "subtitle":"short deck-line subtitle",
  "author":"Barq Cowork",
  "cover_html":"<HTML you design for the first page — free-form, subject-driven>",
  "theme":{
    "name":"short theme name",
    "heading_font":"Georgia | Inter | Playfair Display | Source Serif Pro | Merriweather | ... pick one that fits the subject",
    "body_font":"Inter | Source Sans Pro | Lora | Georgia | ... readable body font",
    "mono_font":"Consolas | JetBrains Mono | Fira Code",
    "body_color":"0F172A",
    "heading1_color":"HEX",
    "heading2_color":"HEX",
    "heading3_color":"HEX",
    "heading4_color":"HEX",
    "accent_color":"HEX",
    "secondary_color":"HEX",
    "link_color":"HEX",
    "quote_color":"HEX",
    "muted_color":"HEX",
    "code_bg_color":"HEX",
    "title_color":"HEX"
  },
  "sections":[
    {"heading":"…","brief":"specific scope","depth":"short","layout_kind":"…"},
    {"heading":"…","brief":"…","depth":"medium","layout_kind":"…"},
    {"heading":"…","brief":"…","depth":"long","layout_kind":"…"}
  ]
}

Rules:
- sections[] must contain exactly %d entries.
- Each brief is 1-2 sentences describing what that section will cover. Be specific to the subject.
- depth: "short" ≈ 1/2 page, "medium" ≈ 1 page, "long" ≈ 1.5-2 pages. Mix them.
%s
- theme: design fonts and colors that SUIT THE SUBJECT. A legal briefing, a wedding lookbook, a cybersecurity report, and a food magazine should look visibly different. Hex values are 6-digit, no '#'. Do not reuse the same palette across unrelated subjects.
%s
- No emoji, no markdown, no fenced code. Return compact JSON only.`, task.Title, task.Description, contextText, sectionCount, sectionCount, coverGuidance, layoutGuidance)

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 3600, []provider.ChatMessage{
		{Role: "system", Content: segmentedDocSystemPrompt()},
		{Role: "user", Content: user},
	}, &out)
	return out, err
}

func (a *AgentLoop) generateSegmentedDocSection(
	ctx context.Context,
	task *domain.Task,
	plan segmentedDocPlan,
	brief segmentedDocBrief,
	index int,
	magazineMode bool,
	runtimeProfile agentRuntimeProfile,
) (segmentedDocSection, error) {
	var out segmentedDocSection
	briefBytes, _ := json.Marshal(brief)

	layoutKind := strings.TrimSpace(brief.LayoutKind)
	if layoutKind == "" {
		layoutKind = "prose"
	}
	layoutGuidance := fmt.Sprintf(`Layout kind for this section: "%s". Choose HTML structure that reflects that layout.`, layoutKind)
	if magazineMode {
		layoutGuidance = fmt.Sprintf(`Layout kind for this section: "%s".
Render it as a distinctive editorial spread — NOT the same structure as a typical prose section:
  - "hero-spread":         oversized <h1>, short bold deck, 2-3 big paragraphs, one pulled subheading.
  - "pull-quote-banner":   a <blockquote> near the top in a large voice, then 2-3 supporting paragraphs.
  - "two-column-feature":  <h1>, then an intro paragraph, then a <table> with two cells acting as left/right columns of prose.
  - "stat-grid":           <h1>, short lede, then a <table> of 3-6 stat cards (label + big number + 1-line description).
  - "sidebar-note":        <h1>, main prose, then a <table> acting as a sidebar box with a label and a highlighted note.
  - "timeline":            <h1>, short lede, then an <ol> of dated / numbered milestones, each with date + headline + 1-line detail.
  - "photo-essay-block":   <h1>, a short stanza-like paragraph, a <blockquote> caption, another short paragraph.
  - "checklist-grid":      <h1>, short intro, a <table> of action items with columns like "Action | Owner | Status".
  - "caption-gallery":     <h1>, 3-4 short captioned blocks (each caption in <blockquote>, each body in <p>).
  - "cover-story":         <h1>, a bold lede paragraph, then a <table> with two columns (summary | key facts).
  - "callout-stack":       <h1>, 3 stacked callout <blockquote> blocks, each followed by one supporting paragraph.
  - "fact-box":            <h1>, short intro, then a <table> of label → value pairs of concrete facts.
  - "interview-qa":        <h1>, 3-5 Q&A pairs where Q is a <strong>-wrapped paragraph and A is a regular paragraph.
  - "index-list":          <h1>, short intro, then an <ol> that acts as an annotated index (term → 1-sentence gloss).
Pick exactly the structure for the given layout_kind. Do NOT fall back to a generic 3-paragraph pattern.`, layoutKind)
	}

	user := fmt.Sprintf(`Write one section of a Word document.

Document title: %s
Document subtitle: %s
Section position: %d of %d
Section brief JSON:
%s

Return ONLY compact valid JSON:
{
  "heading":"same heading or a slightly improved one",
  "html":"<h1>…</h1><p>…</p>…"
}

HTML requirements:
- Open with <h1>%s</h1> (use the section heading verbatim) and then body content.
- Use <h2> for sub-sections if helpful, <p> for prose, <ul>/<ol> for lists,
  <table><thead>…</thead><tbody>…</tbody></table> for tabular data, <blockquote> for pull quotes.
- For "short" depth: 2-3 paragraphs. For "medium": 4-6 paragraphs, possibly a list or small table.
  For "long": 6-10 paragraphs with at least one <ul>/<ol> or <table>.
- Write specific, domain-accurate content. No filler like "this section will discuss…". No TODO markers.
- No emoji, no <style>, no <script>, no <html>/<body>/<head> tags.
%s`, firstNonEmptyString(plan.Title, task.Title), plan.Subtitle, index+1, len(plan.Sections), string(briefBytes), firstNonEmptyString(brief.Heading, "Section"), layoutGuidance)

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 3200, []provider.ChatMessage{
		{Role: "system", Content: segmentedDocSystemPrompt()},
		{Role: "user", Content: user},
	}, &out)
	return out, err
}

func segmentedDocSystemPrompt() string {
	return "You are a senior technical writer. Return only valid JSON, no markdown and no prose. " +
		"Write substantive, subject-specific content — no placeholder phrasing. Use semantic HTML fragments (no <html>, <head>, <body>, <style>, <script>)."
}

// ─── Composition + normalization ────────────────────────────────────────────

func composeSegmentedDocHTML(plan segmentedDocPlan, sections []segmentedDocSection) string {
	var b strings.Builder
	cover := strings.TrimSpace(plan.CoverHTML)
	if cover == "" {
		cover = defaultSegmentedDocCoverHTML(plan)
	}
	b.WriteString(cover)
	b.WriteString("\n")
	for i, s := range sections {
		if i > 0 {
			b.WriteString(`<hr class="pagebreak"/>`)
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimSpace(s.HTML))
		b.WriteString("\n")
	}
	return b.String()
}

// defaultSegmentedDocCoverHTML is the bare-minimum fallback when the LLM
// returns no cover_html. It intentionally emits nothing beyond the title +
// subtitle + page break so there is no "templated" look to inherit — a real
// cover is expected to come from the plan call.
func defaultSegmentedDocCoverHTML(plan segmentedDocPlan) string {
	title := escapeHTMLText(firstNonEmptyString(plan.Title, "Document"))
	var sub string
	if s := strings.TrimSpace(plan.Subtitle); s != "" {
		sub = fmt.Sprintf(`<p>%s</p>`, escapeHTMLText(s))
	}
	return fmt.Sprintf(`<h1>%s</h1>%s<hr class="pagebreak"/>`, title, sub)
}

func escapeHTMLText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func normalizeSegmentedDocPlan(plan *segmentedDocPlan, task *domain.Task, target int) {
	plan.Filename = firstNonEmptyString(plan.Filename, slugifySegmentedFilename(task.Title))
	plan.Title = firstNonEmptyString(plan.Title, task.Title, "Document")
	if plan.Author == "" {
		plan.Author = "Barq Cowork"
	}
	if len(plan.Sections) > target {
		plan.Sections = plan.Sections[:target]
	}
	for len(plan.Sections) < target {
		plan.Sections = append(plan.Sections, segmentedDocBrief{
			Heading: fmt.Sprintf("Section %d", len(plan.Sections)+1),
			Brief:   "Expand the document with relevant, specific content for this section.",
			Depth:   "medium",
		})
	}
	for i := range plan.Sections {
		plan.Sections[i].Heading = firstNonEmptyString(plan.Sections[i].Heading, fmt.Sprintf("Section %d", i+1))
		if plan.Sections[i].Depth == "" {
			plan.Sections[i].Depth = "medium"
		}
		plan.Sections[i].LayoutKind = strings.TrimSpace(plan.Sections[i].LayoutKind)
	}
}
