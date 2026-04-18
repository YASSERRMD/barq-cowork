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

type segmentedDocPlan struct {
	Filename      string              `json:"filename"`
	Title         string              `json:"title"`
	Subtitle      string              `json:"subtitle"`
	Author        string              `json:"author"`
	CoverHTML     string              `json:"cover_html"`
	CoverSubtitle string              `json:"cover_subtitle"`
	Sections      []segmentedDocBrief `json:"sections"`
}

type segmentedDocBrief struct {
	Heading string `json:"heading"`
	Brief   string `json:"brief"`
	Depth   string `json:"depth"` // "short" | "medium" | "long"
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
		plan, planErr = a.generateSegmentedDocPlan(ctx, task, sectionCount, extraSystemPrompts, runtimeProfile)
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

			raw, sErr := a.generateSegmentedDocSection(ctx, task, plan, brief, i, runtimeProfile)
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
	argsBytes, _ := json.Marshal(map[string]any{
		"filename": firstNonEmptyString(plan.Filename, slugifySegmentedFilename(task.Title)),
		"title":    firstNonEmptyString(plan.Title, task.Title, "Document"),
		"author":   plan.Author,
		"html":     composed,
	})
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
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) (segmentedDocPlan, error) {
	var out segmentedDocPlan
	contextText := compactSegmentedExtraContext(extraSystemPrompts)
	user := fmt.Sprintf(`Plan a long-form Word document that will be rendered through write_html_docx.

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
  "cover_subtitle":"one-sentence positioning line",
  "cover_html":"<div class=\"cover-page\">…cover content…</div>",
  "sections":[
    {"heading":"Executive Summary","brief":"specific scope of this section — what will be covered","depth":"short"},
    {"heading":"Introduction","brief":"specific scope — include concrete topics to touch","depth":"medium"},
    {"heading":"…","brief":"…","depth":"long"}
  ]
}

Rules:
- sections[] must contain exactly %d entries. Do NOT include a conclusion + executive summary on top of that — count them inside the %d.
- Each brief is 1-2 sentences describing what that section will cover. Be specific to the subject.
- depth: "short" ≈ 1/2 page, "medium" ≈ 1 page, "long" ≈ 1.5-2 pages. Mix them.
- cover_html should be a <div class="cover-page"> containing <h1 class="cover-title"> and a short meta line.
- No emoji, no markdown, no fenced code. Return compact JSON only.`, task.Title, task.Description, contextText, sectionCount, sectionCount, sectionCount)

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 3200, []provider.ChatMessage{
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
	runtimeProfile agentRuntimeProfile,
) (segmentedDocSection, error) {
	var out segmentedDocSection
	briefBytes, _ := json.Marshal(brief)
	user := fmt.Sprintf(`Write one section of a long Word document.

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
- No emoji, no <style>, no <script>, no <html>/<body>/<head> tags.`, firstNonEmptyString(plan.Title, task.Title), plan.Subtitle, index+1, len(plan.Sections), string(briefBytes), firstNonEmptyString(brief.Heading, "Section"))

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
	for _, s := range sections {
		b.WriteString(strings.TrimSpace(s.HTML))
		b.WriteString("\n")
	}
	return b.String()
}

func defaultSegmentedDocCoverHTML(plan segmentedDocPlan) string {
	subtitle := firstNonEmptyString(plan.CoverSubtitle, plan.Subtitle)
	var meta string
	if plan.Author != "" {
		meta = fmt.Sprintf(`<div class="cover-meta"><span><strong>Prepared by:</strong> %s</span></div>`, escapeHTMLText(plan.Author))
	}
	subtitleHTML := ""
	if subtitle != "" {
		subtitleHTML = fmt.Sprintf(`<p class="cover-subtitle">%s</p>`, escapeHTMLText(subtitle))
	}
	return fmt.Sprintf(`<div class="cover-page"><div class="cover-accent"></div><h1 class="cover-title">%s</h1>%s%s</div>`,
		escapeHTMLText(firstNonEmptyString(plan.Title, "Document")), subtitleHTML, meta)
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
	}
}
