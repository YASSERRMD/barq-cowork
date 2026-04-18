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

// backgroundRequestPattern matches explicit asks for a tinted / decorative
// page background. When present, the orchestrator passes a soft tint to the
// renderer so every page carries a subtle full-page fill.
var backgroundRequestPattern = regexp.MustCompile(`(?i)\b(background (graphics|graphic|pattern|color|colour|tint|fill)|page background|decorative background|tinted page|tinted background|watermark background|colored page|coloured page)\b`)

// wantsBackgroundGraphics reports whether the task explicitly asks for a
// tinted / decorative page background.
func wantsBackgroundGraphics(task *domain.Task) bool {
	if task == nil {
		return false
	}
	return backgroundRequestPattern.MatchString(task.Title + " " + task.Description)
}

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
	_, _, ok := requestedDocumentPageBudget(task)
	return ok
}

// requestedDocumentPageBudget parses the user's "N pages" ask and returns the
// total page count (cover + body) and the number of body sections to plan.
// The invariant is: total pages = 1 cover + sectionCount, so a "3 page" ask
// produces 1 cover + 2 body sections. Page count is clamped to [3, 25];
// sectionCount therefore lands in [2, 24]. Each body section is expected to
// fit on exactly one A4 page (the plan/section prompts enforce this).
func requestedDocumentPageBudget(task *domain.Task) (pages int, sectionCount int, ok bool) {
	if requiredOutputTool(task) != "write_html_docx" {
		return 0, 0, false
	}
	text := strings.TrimSpace(task.Title + " " + task.Description)
	m := requestedDocumentPagePattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, 0, false
	}
	if n < 3 {
		n = 3
	}
	if n > 25 {
		n = 25
	}
	return n, n - 1, true
}

func (a *AgentLoop) runSegmentedDocumentWorkflow(
	ctx context.Context,
	task *domain.Task,
	workspaceRoot string,
	planID string,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) ExecuteResult {
	pages, sectionCount, ok := requestedDocumentPageBudget(task)
	if !ok {
		pages = 7
		sectionCount = 6
	}
	kind := categorizeDocTask(task)
	a.logger.Info("segmented docx: categorized",
		"task_id", task.ID, "kind", string(kind),
		"pages", pages, "body_sections", sectionCount)

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
		plan, planErr = a.generateSegmentedDocPlan(ctx, task, pages, sectionCount, kind, extraSystemPrompts, runtimeProfile)
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

			raw, sErr := a.generateSegmentedDocSection(ctx, task, plan, brief, i, pages, kind, runtimeProfile)
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
	if docKindWantsChrome(kind) {
		renderArgs["chrome"] = map[string]any{
			"header_text":   firstNonEmptyString(plan.Title, task.Title, "Document"),
			"footer_text":   firstNonEmptyString(plan.Author, "Barq Cowork"),
			"show_page_num": true,
		}
	}
	if wantsBackgroundGraphics(task) {
		renderArgs["background"] = map[string]any{
			"color": backgroundTintForPlan(plan),
		}
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
	pages int,
	sectionCount int,
	kind docKind,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) (segmentedDocPlan, error) {
	var out segmentedDocPlan
	contextText := compactSegmentedExtraContext(extraSystemPrompts)

	coverGuidance := docKindCoverGuidance(kind)
	layoutGuidance := docKindLayoutGuidance(kind)
	graphicsGuidance := docKindGraphicsGuidance(kind)
	mathGuidance := docKindMathGuidance(kind)

	user := fmt.Sprintf(`Plan a Word document that will be rendered through write_html_docx.

Task title: %s
Task details: %s
Additional context: %s

Document kind (auto-classified from the task): %s — tune cover, layout, and graphical vocabulary to this kind.

PAGE BUDGET: exactly %d pages total (1 cover page + %d body sections). Each body section MUST fit on EXACTLY ONE A4 page — no more, no less. Plan section scope so it fills one page with dense, substantive content and does not overflow. Do NOT pad with filler to hit the page count; do NOT leave pages half-empty.

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
- cover_html MUST fill the ENTIRE first A4 page — use oversized typography (huge <h1>), a short <h2> or <p> deck, a small meta line (date/author/issue), and at LEAST one extra element (a <blockquote>, a <div class="callout">, a two-cell <table>, or a <div class="statbox">) so the page does not look empty. End with <hr class="pagebreak"/>. Do not produce a minimal "<h1>title</h1><p>subtitle</p><hr/>" cover.
- sections[] must contain exactly %d entries.
- Each brief is 1-2 sentences describing what that section will cover. Be specific to the subject.
- depth: ALWAYS "medium" — every body section targets EXACTLY ONE A4 page of dense, substantive content. Do not use "short" (leaves whitespace) or "long" (overflows). The renderer enforces one-page-per-section sizing.
%s
- theme: design fonts and colors that SUIT THE SUBJECT. A legal briefing, a wedding lookbook, a cybersecurity report, and a food magazine should look visibly different. Hex values are 6-digit, no '#'. Do not reuse the same palette across unrelated subjects.
%s

%s

%s

- No emoji, no markdown, no fenced code. Return compact JSON only.`, task.Title, task.Description, contextText, string(kind), pages, sectionCount, sectionCount, sectionCount, coverGuidance, layoutGuidance, graphicsGuidance, mathGuidance)

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
	pages int,
	kind docKind,
	runtimeProfile agentRuntimeProfile,
) (segmentedDocSection, error) {
	var out segmentedDocSection
	briefBytes, _ := json.Marshal(brief)

	layoutKind := strings.TrimSpace(brief.LayoutKind)
	if layoutKind == "" {
		layoutKind = "prose"
	}
	layoutGuidance := sectionLayoutGuidance(kind, layoutKind)
	graphicsGuidance := docKindGraphicsGuidance(kind)
	mathGuidance := docKindMathGuidance(kind)

	user := fmt.Sprintf(`Write one section of a Word document.

Document kind: %s
Document title: %s
Document subtitle: %s
Document page budget: %d pages total (1 cover + %d body sections).
Section position: %d of %d
Section brief JSON:
%s

Return ONLY compact valid JSON:
{
  "heading":"same heading or a slightly improved one",
  "html":"<h1>…</h1><p>…</p>…"
}

ONE-PAGE RULE: This section MUST fill EXACTLY ONE A4 page — no more, no less.
- Target: ~380–480 words of body prose (roughly 4–6 paragraphs of 3–5 sentences).
- Always include ONE styled component from the graphics catalog (pullquote / callout / keyidea / definition / statbox / factbox / sidebar) near the middle or end — it absorbs vertical space and balances the page visually.
- If the content is light, add a compact <table> (3-5 rows) or a short <ul>/<ol>. Do NOT leave the page half-empty.
- If the content is heavy, trim paragraphs — do NOT overflow to a second page. Tight, dense writing over verbose.
- Do NOT emit <hr class="pagebreak"/>; the composer inserts page breaks between sections.

HTML requirements:
- Open with <h1>%s</h1> (use the section heading verbatim) and then body content.
- Use <h2> for sub-sections if helpful, <p> for prose, <ul>/<ol> for lists,
  <table><thead>…</thead><tbody>…</tbody></table> for tabular data, <blockquote> for pull quotes.
- Write specific, domain-accurate content. No filler like "this section will discuss…". No TODO markers.
- No emoji, no <style>, no <script>, no <html>/<body>/<head> tags.
%s

%s

%s`, string(kind), firstNonEmptyString(plan.Title, task.Title), plan.Subtitle, pages, len(plan.Sections), index+1, len(plan.Sections), string(briefBytes), firstNonEmptyString(brief.Heading, "Section"), layoutGuidance, graphicsGuidance, mathGuidance)

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

// backgroundTintForPlan returns a very-light hex tint suitable for a full-page
// background fill. Prefers the theme's muted_color / code_bg_color (already
// tuned to be light), otherwise falls back to a neutral cream so body text
// remains readable.
func backgroundTintForPlan(plan segmentedDocPlan) string {
	if plan.Theme != nil {
		for _, c := range []string{plan.Theme.CodeBgColor, plan.Theme.MutedColor} {
			c = strings.TrimSpace(strings.TrimPrefix(c, "#"))
			if len(c) == 6 {
				return strings.ToUpper(c)
			}
		}
	}
	return "F7F5EF"
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
