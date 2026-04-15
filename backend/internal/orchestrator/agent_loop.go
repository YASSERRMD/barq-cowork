package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/tools"
	"github.com/google/uuid"
)

const agentSystemPrompt = `You are an autonomous AI agent for Barq Cowork, an intelligent document and file workspace assistant.

Complete the given task by calling tools one at a time.

OUTPUT FORMAT — pick the right tool:
- "presentation", "slides", "deck", "slideshow", "pptx", "powerpoint" → MUST use write_pptx
- "document", "report", "doc", "word", "brief", "proposal", "writeup", "paper" → MUST use write_docx
- "summary", "notes", "markdown" → use write_markdown_report (.md file)
- "data", "spreadsheet", "export" → use export_json (.json file)
- general file → use write_file

RULES:
1. Call ONE tool at a time.
2. ALWAYS produce at least one output file. Never finish without writing a file.
3. Stop after the file is written. Max 15 tool calls total.
4. Use ask_user when you need clarification, preference, or feedback mid-task. The user will respond in real-time via the UI. Keep questions short and specific.

INTERACTION: Only use ask_user when you genuinely need clarification — do NOT ask questions just to be polite. If the task is clear enough, start working immediately. After completing the task, tell the user what you did and that you are available if they want changes — do NOT end the conversation abruptly. The user can ask follow-up questions at any time.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WRITE_PPTX — 10 SLIDE TYPES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Required fields: "filename", "title", "slides".
Respect any explicit user slide count. If the user asks for 3 slides, do not silently expand to 6-10.
Each slide MUST have: "heading" (≤60 chars) and "type".

PLANNING ORDER — DO THIS BEFORE WRITING THE DECK:
1. Plan the full presentation first: subject, audience, narrative arc, theme, visual style, cover style, color story, motif, and the mix of slide layouts.
2. Only then plan each slide individually: choose the most suitable type, verify the slide has enough content, and check whether it needs icons, charts, diagrams, timelines, comparisons, or tables.
3. Do not use a fixed deck template or the same layout repeatedly. The structure must come from the user's subject and explicit instructions.
4. If the user explicitly requests a style, sections, sequence, visual direction, or specific slide elements, follow that exactly.
5. Before you call write_pptx, mentally audit every slide: proper heading, proper content density, proper layout choice, and proper icons/visuals for that slide.
6. The write_pptx tool validates slide fit before it renders. Do not send thin, repetitive, or underfilled slides.
7. Use a complete "deck" object on EVERY write_pptx tool call. It is required. Fill subject, audience, narrative, theme, visual_style, cover_style, color_story, motif, kicker, a full palette, and deck.design render directives.
7a. New PowerPoint generation is HTML-authored by default: provide deck.theme_css, deck.cover_html, and HTML markup for every content slide. Use the modern Bootstrap-compatible component kit and official Bootstrap Icons placeholders. Do not rely on structured fallback slides for client-facing decks.
8. Examples below show JSON shape only. Do not copy example palette values, repeated classroom amber tones, or the same cover layout across different decks unless the user explicitly asks for that direction.
9. Never put internal planning metadata on the slides. The final presentation should show user-facing content only.
10. The HTML preview and the downloaded PPTX must look like the same deck. Choose design directives that the native renderer can follow, and add slide-level design objects when a slide needs a specific composition.
11. Default to a refined contemporary presentation aesthetic. Avoid dated corporate blue-orange palettes, thick office-style borders, repetitive bullet slabs, and generic 2000s deck patterns unless the user explicitly asks for them.
12. For AI, digital-future, innovation, or forward-looking subjects, prefer airy split/gallery compositions, restrained surfaces, and a fresh palette rather than classroom amber or old business-deck blue.
13. Keep one coherent deck system across the full presentation: cover language, header treatment, spacing, card style, and table/chart chrome should feel like one designed document, not a different template on every slide.
14. If the request is a rollout, proposal, cost estimate, implementation plan, or delivery roadmap, treat it as a structured proposal/report system even if the subject is AI, healthcare, or technology.
15. Keep density disciplined in authored HTML: most slides should use 56-78px page padding, 12-24px gaps, 18-22px body copy, and 32-42px section titles unless the slide has a deliberate reason to break that range.
15a. On 3-5 slide decks and projector-first decks, prefer larger reading sizes and fewer larger panels: 20-24px body copy, 40-56px section titles, and 2-column or 2x2 compositions instead of 4-up micro-card layouts.
16. Avoid empty hero frames, giant icon boxes, oversized rounded cards, and slides that leave large parts of the canvas unused.
17. Do not bottom-anchor the full cover composition; the cover should occupy the canvas intentionally instead of leaving a large empty upper half.

TYPE REFERENCE — choose the right type per slide:

TYPE "bullets"  → detailed text list
  "points": ["Point 1", "Point 2", "Point 3"]   // 3-6 strings

TYPE "stats"  → KPI / metric cards
  "stats": [{"value":"$2.4M","label":"ARR","desc":"annual recurring revenue"},
            {"value":"92%","label":"Retention","desc":"30-day cohort"},
            {"value":"3.2x","label":"ROI","desc":"vs industry avg"}]  // 2-4 items

TYPE "steps"  → numbered process flow with arrows
  "steps": ["Define requirements","Design architecture","Build MVP","Test & iterate","Deploy"]

TYPE "cards"  → icon feature grid (4-6 cards)
  "cards": [{"icon":"automation","title":"Speed","desc":"Sub-50ms response time"},
            {"icon":"shield","title":"Security","desc":"SOC2 Type II certified"},
            {"icon":"integration","title":"Integrations","desc":"200+ native connectors"},
            {"icon":"chart","title":"Analytics","desc":"Real-time dashboards"}]

DECK OBJECT — REQUIRED on every write_pptx call:
  "deck": {
    "subject":"<subject framing>",
    "audience":"<who this deck is for>",
    "narrative":"<story arc>",
    "theme":"<domain theme>",
    "visual_style":"<chosen visual direction>",
    "cover_style":"<editorial|orbit|mosaic|poster|playful>",
    "color_story":"<chosen color mood>",
    "motif":"<semantic motif token>",
    "kicker":"<short cover line>",
    "design":{
      "composition":"<split|frame|stack|band|float|asym|gallery>",
      "density":"<airy|balanced|dense>",
      "shape_language":"<soft|mixed|crisp>",
      "accent_mode":"<rail|band|chip|ribbon|glow|block>",
      "hero_layout":"<motif|figures|data|people|product|abstract>"
    },
    "theme_css":"<optional CSS design system for authored HTML slides>",
    "cover_html":"<optional HTML body for the cover slide>",
    "palette":{"background":"<hex>","card":"<hex>","accent":"<hex>","accent2":"<hex>","text":"<hex>","muted":"<hex>","border":"<hex>"}
  }

OPTIONAL SLIDE DESIGN — use when a slide needs a deliberate composition:
  "design":{
    "layout_style":"<stack|split|grid|rail|stage|matrix|spotlight>",
    "panel_style":"<soft|solid|outline|glass|tint>",
    "accent_mode":"<rail|chip|ribbon|band|marker|glow>",
    "density":"<airy|balanced|dense>",
    "visual_focus":"<text|metric|icon|data|process|compare>"
  }

HTML SLIDE MODE — required for bespoke decks and the default for new decks:
  {
    "heading":"<user-facing slide title>",
    "type":"html",
    "html":"<self-contained slide body markup using Bootstrap-compatible deck classes and Bootstrap Icons placeholders when needed>",
    "speaker_notes":"<optional presenter notes>"
  }
- Use HTML slide mode for every new deck unless the user explicitly asks for a very simple internal draft.
- Keep markup semantic and self-contained.
- No scripts.
- No external assets.
- Do not author inline SVG for icons. Use Bootstrap Icons placeholders only, for example <i class="bi bi-graph-up-arrow" aria-hidden="true"></i>.
- Use diagrams made from Bootstrap-style cards, rows, list groups, timelines, tables, or chart data instead of hand-authored SVG drawings unless the user explicitly asks for an SVG diagram.
- Preview and downloaded PPTX are expected to come from the same HTML slide DOM.
- The write_pptx tool rejects incomplete HTML/CSS deck contracts.
- The renderer injects a slide shell wrapper automatically. Author the inner composition of the slide instead of building a giant outer page wrapper with excessive padding.
- Baseline modern Bootstrap-style class kit available in the HTML slide shell:
  - container-fluid
  - row, col, col-auto, col-1 through col-12
  - g-0 through g-5
  - d-flex, d-grid, flex-column, flex-wrap
  - align-items-start, align-items-center, align-items-stretch
  - justify-content-start, justify-content-center, justify-content-between, justify-content-end
  - h-100, w-100, gap-1 through gap-5, p-0, p-2 through p-5
  - display-1 through display-6, lead, small, fw-semibold, fw-bold, text-uppercase, text-muted
  - card, card-body, card-title, card-subtitle, card-text
  - badge, rounded-pill, rounded-4, border-0, border-top, border-start
  - list-group, list-group-item
  - icon-badge
  - Bootstrap Icons placeholders such as i.bi.bi-shield-check, i.bi.bi-graph-up-arrow, i.bi.bi-kanban, i.bi.bi-calendar3, i.bi.bi-people, i.bi.bi-check2-circle
- Legacy Barq helper classes remain available:
  - cover-shell, cover-grid, cover-stack, cover-aside
  - slide-shell, slide-head, slide-grid, slide-main, slide-side
  - eyebrow, display-title, section-title, lede, body-copy, muted-copy, rule
  - tag, panel, panel-light
  - grid-2, grid-3, grid-4
  - stat-card, stat-value, stat-label, stat-desc
  - bullet-list, bullet-item
  - steps-flow, step-item, step-num, step-title, step-desc
  - timeline-list, timeline-row, timeline-date
  - compare-grid, compare-col
- Prefer Bootstrap rows, columns, cards, list groups, badges, and icon-badges over ad-hoc browser layout wrappers so the exported PPTX keeps strong framing, denser content, and cleaner hierarchy.

TYPE "chart"  → full-slide native PowerPoint chart
  "chart_type": "column" | "bar" | "line" | "pie" | "doughnut" | "area"
  "chart_categories": ["Q1 2025","Q2 2025","Q3 2025","Q4 2025"]
  "chart_series": [{"name":"Revenue","values":[1.2,1.8,2.4,3.1]},
                   {"name":"Target", "values":[1.0,1.5,2.0,2.5]}]
  "y_label": "Revenue ($M)"   // optional

TYPE "timeline"  → horizontal milestone spine
  "timeline": [{"date":"Q1 2026","title":"Beta Launch","desc":"Closed beta with 50 teams"},
               {"date":"Q2 2026","title":"GA Release","desc":"Public availability"},
               {"date":"Q4 2026","title":"Enterprise Tier","desc":"SOC2 + SLA"},
               {"date":"Q2 2027","title":"Global Expansion","desc":"EMEA & APAC"}]

TYPE "compare"  → two-column side-by-side
  "left_column":  {"heading":"Before","points":["Manual processes","5-day cycle time","$80k/yr overhead"]}
  "right_column": {"heading":"After", "points":["Fully automated","4-hour cycle time","$12k/yr cost"]}

TYPE "table"  → styled data table
  "table": {"headers":["Feature","Starter","Pro","Enterprise"],
            "rows":[["Users","Up to 5","Up to 50","Unlimited"],
                    ["SLA","99.9%","99.95%","99.99%"],
                    ["Support","Email","Priority","Dedicated CSM"]]}

TYPE "blank"  → empty slide (use for section dividers)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
LAYOUT STRATEGY — MIX TYPES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
- The cover comes from deck.cover_html. Every slide in "slides" is a content slide and should carry its own authored HTML.
- Use "stats" for any slide with 2+ numeric metrics or KPIs.
- Use "chart" for any slide with time-series, comparison bars, or trend data.
- Use "steps" for process, pipeline, or how-it-works slides.
- Use "cards" for feature lists, benefit grids, team capabilities.
- Use "timeline" for roadmap, milestones, history.
- Use "compare" for before/after, legacy vs new, pros/cons.
- Use "table" for pricing, feature matrix, structured comparisons.
- Use icons on cards and capability slides.
- Use charts, diagrams, timelines, and comparison views when the content actually benefits from them.
- Avoid repeating the same type or visual structure unless the subject truly requires it.
- AIM for at least 4 different types in an 8+ slide deck.
- Add "speaker_notes" to any slide for presenter guidance.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WRITE_DOCX — WORD DOCUMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Required: "filename", "title", "sections" (array).
Each section: "heading", "level" (1 or 2), "content" (text; prefix lines with "• " for bullets),
optional "table": {"headers":[], "rows":[[]]}.
Include: executive summary, 4-6 body sections, conclusion.`

// AgentLoop implements a ReAct-style agent that iterates LLM → tool → LLM until done.
type AgentLoop struct {
	prov      provider.LLMProvider
	cfg       provider.ProviderConfig
	registry  *tools.Registry
	plans     PlanStore
	artifacts ArtifactStore
	events    EventEmitter
	logger    *slog.Logger
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(
	prov provider.LLMProvider,
	cfg provider.ProviderConfig,
	registry *tools.Registry,
	plans PlanStore,
	artifacts ArtifactStore,
	events EventEmitter,
	logger *slog.Logger,
) *AgentLoop {
	return &AgentLoop{
		prov:      prov,
		cfg:       cfg,
		registry:  registry,
		plans:     plans,
		artifacts: artifacts,
		events:    events,
		logger:    logger,
	}
}

// Run executes the task using a ReAct loop. It creates and updates plan steps
// so the frontend can poll them in real-time.
func (a *AgentLoop) Run(
	ctx context.Context,
	task *domain.Task,
	workspaceRoot string,
	extraSystemPrompts ...string,
) ExecuteResult {
	// Create plan record upfront
	planID := uuid.NewString()
	plan := &domain.Plan{
		ID:        planID,
		TaskID:    task.ID,
		CreatedAt: time.Now().UTC(),
	}
	if err := a.plans.CreatePlan(ctx, plan); err != nil {
		a.logger.Error("agent loop: failed to create plan", "error", err)
	}

	// Initial conversation
	userMsg := "Task: " + task.Title
	if task.Description != "" && task.Description != task.Title {
		userMsg += "\n\nDetails: " + task.Description
	}

	messages := []provider.ChatMessage{{Role: "system", Content: agentSystemPrompt}}
	if hint := requestedPresentationDeckHint(task); hint != "" {
		messages = append(messages, provider.ChatMessage{Role: "system", Content: hint})
	}
	for _, prompt := range extraSystemPrompts {
		if strings.TrimSpace(prompt) == "" {
			continue
		}
		messages = append(messages, provider.ChatMessage{Role: "system", Content: prompt})
	}
	runtimeProfile := buildAgentRuntimeProfile(a.cfg, task)
	if strings.TrimSpace(runtimeProfile.CompatibilityPrompt) != "" {
		messages = append(messages, provider.ChatMessage{Role: "system", Content: runtimeProfile.CompatibilityPrompt})
	}
	messages = append(messages, provider.ChatMessage{Role: "user", Content: userMsg})

	toolDefs := a.registry.Definitions()

	var result ExecuteResult
	stepOrder := 0
	forcedToolNudges := 0

	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
		"message": "agent loop started",
	})

	for iter := 0; iter < runtimeProfile.MaxIterations; iter++ {
		if ctx.Err() != nil {
			break
		}

		a.logger.Info("agent loop iteration", "task_id", task.ID, "iter", iter, "messages", len(messages))

		req := provider.ChatCompletionRequest{
			Model:       a.cfg.Model,
			Stream:      true,
			MaxTokens:   runtimeProfile.MaxTokens,
			Temperature: runtimeProfile.Temperature,
			Messages:    messages,
			Tools:       toolDefs,
		}

		ch, err := provider.ChatWithRetry(ctx, a.prov, a.cfg, req, runtimeProfile.Retry, a.logger)
		if err != nil {
			a.logger.Error("agent loop: LLM call failed", "error", err, "iter", iter)
			result.Failed++
			break
		}

		// Collect the full response
		var contentSB strings.Builder
		var toolCalls []provider.ToolCall
		for chunk := range ch {
			if chunk.Done {
				break
			}
			contentSB.WriteString(chunk.ContentDelta)
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}

		content := contentSB.String()
		a.logger.Info("agent loop: LLM responded",
			"task_id", task.ID,
			"iter", iter,
			"content_len", len(content),
			"tool_calls", len(toolCalls),
			"content_snippet", truncate(content, 200),
		)

		// Emit agent message event if there is meaningful text content
		if len(content) > 5 {
			a.emitAgentEvent(ctx, task.ID, domain.EventTypeAgentMessage, map[string]any{
				"text": content,
			})
		}

		requiredTool := requiredOutputTool(task)
		if len(toolCalls) == 0 && runtimeProfile.AllowRawJSONToolArgs && result.Completed == 0 {
			if recovered, ok := recoverToolCallFromContent(content, requiredTool); ok {
				toolCalls = []provider.ToolCall{recovered}
				a.logger.Info("agent loop: recovered direct tool args from assistant content", "task_id", task.ID, "iter", iter, "tool", requiredTool)
			}
		}

		// No tool calls → agent decided it is done
		if len(toolCalls) == 0 {
			if requiredTool != "" && result.Completed == 0 && forcedToolNudges < runtimeProfile.MaxForcedToolNudges {
				forcedToolNudges++
				nudge := fmt.Sprintf(
					"You have not called any tool yet. Your next response MUST be exactly one tool call to %s. "+
						"Do not answer with prose, planning text, or explanation. "+
						"Write the output file in this turn now.",
					requiredTool,
				)
				if requiredTool == "write_pptx" {
					nudge += " For presentation tasks, include the full deck object plus deck.theme_css, deck.cover_html, and HTML markup for every content slide. Compose slides with Bootstrap-style rows, columns, cards, list groups, badges, and Bootstrap Icons placeholders. Do not fall back to structured-only slides."
				}
				if strings.TrimSpace(runtimeProfile.CompatibilityPrompt) != "" {
					nudge += " If this model cannot emit native tool calls, respond with ONLY the JSON arguments object for the required tool."
				}
				messages = append(messages, provider.ChatMessage{
					Role:    "assistant",
					Content: content,
				})
				messages = append(messages, provider.ChatMessage{
					Role:    "system",
					Content: nudge,
				})
				a.logger.Warn("agent loop: no tool calls, nudging required output tool", "task_id", task.ID, "iter", iter, "tool", requiredTool, "nudges", forcedToolNudges)
				continue
			}
			if requiredTool != "" && result.Completed == 0 {
				a.logger.Error("agent loop: stopping without required output tool", "task_id", task.ID, "iter", iter, "tool", requiredTool)
				result.Failed++
				break
			}
			a.logger.Info("agent loop: no tool calls, stopping", "task_id", task.ID, "iter", iter)
			break
		}

		// Add assistant turn to conversation
		messages = append(messages, provider.ChatMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		})

		// Execute each tool call
		for _, tc := range toolCalls {
			step := createToolPlanStep(planID, stepOrder, tc)
			if seeded, ok := createPresentationPlanSteps(planID, stepOrder, tc); ok {
				for _, seededStep := range seeded {
					_ = a.plans.CreateStep(ctx, seededStep)
				}
				step = seeded[len(seeded)-1]
				stepOrder = step.Order
			} else {
				stepOrder = step.Order
				_ = a.plans.CreateStep(ctx, step)
			}

			a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
				"step_id": step.ID, "tool": tc.Name, "iter": iter,
			})

			// Execute
			toolOutput, stepErr := a.executeToolCall(ctx, tc, task, workspaceRoot)

			completed := time.Now().UTC()
			step.CompletedAt = &completed

			if stepErr != nil {
				step.Status = domain.StepStatusFailed
				step.ToolOutput = marshalError(stepErr)
				result.Failed++
			} else {
				step.Status = domain.StepStatusCompleted
				step.ToolOutput = toolOutput
				result.Completed++
				a.maybeRecordAgentArtifact(ctx, step, task, workspaceRoot)
			}
			_ = a.plans.UpdateStep(ctx, step)

			a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
				"step_id": step.ID, "tool": tc.Name, "status": step.Status,
			})

			// Feed tool result back to LLM
			messages = append(messages, provider.ChatMessage{
				Role:       "tool",
				Content:    toolOutput,
				ToolCallID: tc.ID,
			})
		}
	}

	return result
}

func createToolPlanStep(planID string, startOrder int, tc provider.ToolCall) *domain.PlanStep {
	now := time.Now().UTC()
	return &domain.PlanStep{
		ID:          uuid.NewString(),
		PlanID:      planID,
		Order:       startOrder + 1,
		Title:       tc.Name,
		Description: "Tool call: " + tc.Name,
		ToolName:    tc.Name,
		ToolInput:   tc.Arguments,
		Status:      domain.StepStatusRunning,
		StartedAt:   &now,
	}
}

func createPresentationPlanSteps(planID string, startOrder int, tc provider.ToolCall) ([]*domain.PlanStep, bool) {
	if tc.Name != "write_pptx" {
		return nil, false
	}

	var payload struct {
		Title    string `json:"title"`
		Filename string `json:"filename"`
		Deck     struct {
			Subject  string `json:"subject"`
			Audience string `json:"audience"`
		} `json:"deck"`
		Slides []struct {
			Heading string `json:"heading"`
		} `json:"slides"`
	}
	if err := json.Unmarshal([]byte(tc.Arguments), &payload); err != nil {
		return nil, false
	}

	deckTitle := firstNonEmptyString(strings.TrimSpace(payload.Title), strings.TrimSpace(payload.Deck.Subject), strings.TrimSpace(payload.Filename), "presentation")
	audience := firstNonEmptyString(strings.TrimSpace(payload.Deck.Audience), "the intended audience")
	timestamp := time.Now().UTC()
	completedAt := timestamp

	completedStep := func(order int, title, description string) *domain.PlanStep {
		return &domain.PlanStep{
			ID:          uuid.NewString(),
			PlanID:      planID,
			Order:       order,
			Title:       title,
			Description: description,
			Status:      domain.StepStatusCompleted,
			StartedAt:   &timestamp,
			CompletedAt: &completedAt,
		}
	}

	var steps []*domain.PlanStep
	order := startOrder + 1
	steps = append(steps,
		completedStep(order, "Plan deck system", fmt.Sprintf(`Define the narrative, audience, and design language for "%s" for %s.`, deckTitle, audience)),
	)
	order++
	steps = append(steps,
		completedStep(order, "Draft cover slide", "Author the cover HTML, theme CSS, and deck-level visual system."),
	)
	for idx, slide := range payload.Slides {
		order++
		heading := firstNonEmptyString(strings.TrimSpace(slide.Heading), fmt.Sprintf("Slide %d", idx+1))
		steps = append(steps,
			completedStep(order, fmt.Sprintf("Draft slide %d", idx+1), fmt.Sprintf(`Compose "%s" with the slide-specific HTML, layout, and visuals.`, heading)),
		)
	}

	order++
	renderStarted := time.Now().UTC()
	steps = append(steps, &domain.PlanStep{
		ID:          uuid.NewString(),
		PlanID:      planID,
		Order:       order,
		Title:       "Render PowerPoint file",
		Description: fmt.Sprintf("Export %d total slides into the final .pptx and publish the finished artifact.", len(payload.Slides)+1),
		ToolName:    tc.Name,
		ToolInput:   tc.Arguments,
		Status:      domain.StepStatusRunning,
		StartedAt:   &renderStarted,
	})

	return steps, true
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (a *AgentLoop) executeToolCall(
	ctx context.Context,
	tc provider.ToolCall,
	task *domain.Task,
	workspaceRoot string,
) (string, error) {
	t, ok := a.registry.Get(tc.Name)
	if !ok {
		return tools.Err("tool not found: %s", tc.Name).ToJSON(), fmt.Errorf("tool not found: %s", tc.Name)
	}

	ictx := tools.InvocationContext{
		WorkspaceRoot: workspaceRoot,
		TaskID:        task.ID,
	}

	result := t.Execute(ctx, ictx, tc.Arguments)
	if result.Status != tools.ResultOK {
		return result.ToJSON(), fmt.Errorf("%s", result.Error)
	}
	return result.ToJSON(), nil
}

func resolveArtifactContentPath(workspaceRoot, outputPath string) string {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return ""
	}
	if filepath.IsAbs(outputPath) {
		return filepath.Clean(outputPath)
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		return filepath.Clean(filepath.FromSlash(outputPath))
	}
	return filepath.Clean(filepath.Join(workspaceRoot, filepath.FromSlash(outputPath)))
}

func (a *AgentLoop) maybeRecordAgentArtifact(ctx context.Context, step *domain.PlanStep, task *domain.Task, workspaceRoot string) {
	artType, ok := artifactTools[step.ToolName]
	if !ok {
		return
	}

	var output struct {
		Data struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(step.ToolOutput), &output); err != nil || output.Data.Path == "" {
		return
	}

	contentPath := resolveArtifactContentPath(workspaceRoot, output.Data.Path)
	if contentPath == "" {
		return
	}

	artifact := &domain.Artifact{
		ID:          uuid.NewString(),
		TaskID:      task.ID,
		ProjectID:   task.ProjectID,
		Name:        output.Data.Path,
		Type:        artType,
		ContentPath: contentPath,
		Size:        output.Data.Size,
		CreatedAt:   time.Now().UTC(),
	}
	if err := a.artifacts.Create(ctx, artifact); err != nil {
		a.logger.Warn("agent loop: failed to record artifact", "error", err)
		return
	}

	a.emitAgentEvent(ctx, task.ID, domain.EventTypeArtifactReady, map[string]any{
		"artifact_id": artifact.ID,
		"name":        artifact.Name,
		"type":        artType,
	})
}

func (a *AgentLoop) emitAgentEvent(ctx context.Context, taskID string, t domain.EventType, data map[string]any) {
	payload, _ := json.Marshal(data)
	ev := &domain.Event{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Type:      t,
		Payload:   string(payload),
		CreatedAt: time.Now().UTC(),
	}
	if err := a.events.Create(ctx, ev); err != nil {
		a.logger.Warn("agent loop: event emit failed", "error", err)
	}
}

func requiredOutputTool(task *domain.Task) string {
	text := strings.ToLower(strings.TrimSpace(task.Title + " " + task.Description))
	switch {
	case containsTaskKeyword(text, "presentation", "slides", "deck", "slideshow", "pptx", "powerpoint"):
		return "write_pptx"
	case containsTaskKeyword(text, "document", "report", "doc", "word", "brief", "proposal", "writeup", "paper"):
		return "write_docx"
	case containsTaskKeyword(text, "summary", "notes", "markdown"):
		return "write_markdown_report"
	case containsTaskKeyword(text, "data", "spreadsheet", "export", "json"):
		return "export_json"
	case text != "":
		return "write_file"
	default:
		return ""
	}
}

func containsTaskKeyword(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

var requestedSlideCountPattern = regexp.MustCompile(`(?i)\b(\d+)\s*[- ]?\s*(content\s+)?slides?\b`)

func requestedPresentationDeckHint(task *domain.Task) string {
	if requiredOutputTool(task) != "write_pptx" {
		return ""
	}
	text := strings.TrimSpace(task.Title + " " + task.Description)
	matches := requestedSlideCountPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil || count < 1 || count > 30 {
		return ""
	}
	isContentCount := strings.TrimSpace(matches[2]) != ""
	if isContentCount {
		return fmt.Sprintf(
			"User explicitly requested %d content slides. Honor that exactly. Produce exactly %d entries in the slides array, plus the separate cover from deck.cover_html. Do not silently expand the deck.",
			count,
			count,
		)
	}
	if count <= 1 {
		return "User explicitly requested 1 slide. Keep the deck minimal and do not expand it."
	}
	return fmt.Sprintf(
		"User explicitly requested %d total slides. Honor that exactly. Because write_pptx uses deck.cover_html for the cover separately, produce exactly %d content slides in the slides array so the final deck totals %d slides. Do not silently expand the deck.",
		count,
		count-1,
		count,
	)
}
