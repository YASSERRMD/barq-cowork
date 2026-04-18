package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
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

const agentSystemPrompt = `You are an autonomous AI agent for Barq Cowork, an intelligent document and file workspace.

Complete the user's task by calling tools. Pick the right tool, fill it out completely, and ship the file.

TOOL ROUTING
- presentation, slides, deck, slideshow, pptx, powerpoint → write_pptx
- document, report, doc, word, brief, proposal, writeup, paper → write_html_docx
- summary, notes, markdown → write_markdown_report
- data, spreadsheet, export → export_json
- everything else → write_file

LOOP RULES
1. One tool per turn. At most 15 tool calls for the whole task.
2. Always produce at least one output file — the task isn't done until a file is saved.
3. Plan silently inside the tool call. No prose-only planning turns before writing.
4. Ask clarification only when you genuinely need it — not to be polite. If the ask is clear, start working.
5. After the file is written, briefly tell the user what you did and offer to adjust. Don't end abruptly.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WRITE_PPTX — HOW TO BUILD A DECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Required: filename, title, slides, deck.

The renderer is PARAMETRIC, not template-driven. Preview and final .pptx are rendered from the same structured fields — you describe the design, the renderer composes it. You do NOT author raw HTML or CSS. You do NOT pick from a fixed layout enum.

PLANNING CHECKLIST (do this silently before the call):
1. Archetype — what kind of deck is this? (proposal, executive brief, product narrative, strategy roadmap, educational explainer, cultural/heritage, etc.)
2. Structure — opening → evidence → comparison/roadmap → close. Use the slide count the user asked for; don't silently expand.
3. Per-slide type — bullets / stats / cards / steps / timeline / compare / table / chart / blank.
4. Per-slide composition — describe it in natural language in design.layout_style. No two adjacent slides may share the same composition.
5. Palette — pick colors that fit the topic mood. Avoid generic blue-on-white defaults.
6. Audit — every slide has a strong heading, dense real content (no thin filler), concrete numbers where relevant, no planning-language leakage.

SLIDE TYPES (pick the one that best fits each slide's content):
- bullets:   4-6 points, each = claim + proof OR recommendation + rationale.
- stats:     2-4 real metrics with value, label, desc. Values are actual numbers.
- cards:     3-6 cards, each with semantic icon (shield, chart, people, automation, spark, leaf, gear, growth, integration, strategy, learning, health), title, desc.
- steps:     3-6 "title: description" steps.
- timeline:  3-6 milestones with date/phase, title, desc.
- compare:   both columns substantive — heading + 3-5 points each.
- table:     headers + 3+ rows.
- chart:     chart_type (column/bar/line/pie/doughnut/area), chart_categories, chart_series, optional y_label.
- blank:     only for section dividers.

Aim for at least 4 different types in an 8+ slide deck. Don't use emoji in visible slide content — icons are referenced by name (e.g. "shield"), the renderer draws them.

DESIGN FIELDS (these drive the parametric renderer)

design.layout_style — freeform phrase describing slide composition. The renderer interprets any of these words and you can combine them or add your own descriptive words:
  split / side / panel / aside / left / right    → left/right panel split
  wide / broad / major / 60 / 65                  → wider split (~52%)
  narrow / minor / 30 / 35                        → narrower split (~32%)
  3-col / three-col / triple                      → 3 columns
  2-col / two-col / dual / double / grid          → 2 columns
  hero / spotlight / focus / stage / lead / featured / banner / above / top → lead block first
  rail / horizontal / row / h-flow                → content flows horizontally

  Examples of good layout_style values (be creative, vary every slide):
    "wide split with hero spotlight above, dense two-col points"
    "narrow left aside, three-col content grid"
    "horizontal rail of milestones"
    "single column stack, airy spacing"
    "featured banner stat, supporting stats in row below"
    "asymmetric 65/35 split, left panel solid"

design.panel_style — how the lead panel looks:
  solid / filled / block / dark / bold → solid filled
  outline / border / wire / ghost      → outline only
  (anything else)                      → semi-transparent glass

design.accent_mode — where the accent color sits:
  rail (default)             → vertical bar on left edge
  band / top / bar / stripe  → horizontal bar across top
  chip / badge / dot / pill  → small badge top-right
  glow / ambient / soft      → ambient glow

design.density — whitespace and text size:
  airy / sparse / open  → 0.75× (lots of whitespace)
  balanced (default)    → 1.0×
  dense / compact / tight → 1.28× (fuller slide)

VARIATION IS MANDATORY — no two consecutive slides may share the same layout_style phrase. Vary accent_mode and density across the deck so the rhythm shifts.

DECK OBJECT (required on every write_pptx call):
  deck: {
    subject, audience, narrative, theme, visual_style,
    cover_style: "editorial" | "orbit" | "mosaic" | "poster" | "playful",
    color_story, motif, kicker,
    palette: { background, card, accent, accent2, text, muted, border }  // all hex
  }

PALETTE GUIDANCE — match the topic mood:
  Education / warm topics: amber/orange accent, warm off-white bg
  Tech / data:             teal/indigo accent, cool neutral bg
  Health / environment:    green accent, soft paper bg
  Finance / strategy:      navy/slate accent, clean white
  Creative / marketing:    vibrant purple/pink accent, bold bg
  Culture / heritage:      burgundy or ochre accent, cream bg

QUALITY BAR
- Every slide has a strong, specific heading (≤60 chars).
- No empty hero frames, no oversized icon boxes, no slides that leave half the canvas blank.
- Default to a refined contemporary aesthetic. Avoid dated blue-orange corporate palettes unless the user asks for them.
- One coherent deck system: cover language, header treatment, card style, table/chart chrome should feel like one designed document.
- For 3-5 slide decks: larger reading sizes, fewer larger panels (density airy or balanced).
- For 8+ slide decks: each slide concise but complete — one strong heading, one lead sentence, 2-4 dense items.
- If the user asks for a proposal, rollout, cost estimate, implementation plan, or roadmap, treat it as a structured proposal system even if the subject is tech or healthcare.

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

DECK OBJECT SHAPE — matches DECK OBJECT above. Keys:
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
    "palette":{"background":"<hex>","card":"<hex>","accent":"<hex>","accent2":"<hex>","text":"<hex>","muted":"<hex>","border":"<hex>"}
  }

SLIDE DESIGN — set per slide, all freeform phrases the parametric renderer interprets:
  "design":{
    "layout_style":"<freeform phrase, see DESIGN FIELDS above>",
    "panel_style":"<solid|outline|glass|etc>",
    "accent_mode":"<rail|band|chip|glow>",
    "density":"<airy|balanced|dense>",
    "visual_focus":"<text|metric|icon|data|process|compare>"
  }

Do NOT author raw HTML or CSS. The renderer composes slides from the structured fields (points, stats, cards, steps, timeline, table, etc.) using the design phrases you supply. Preview and the .pptx file are produced from the same source — they will match only if you fill the structured fields well.

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
PICKING THE RIGHT SLIDE TYPE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
- Cover is generated from title, subtitle, deck.kicker, deck.color_story — no HTML needed.
- stats     → any slide with 2+ numeric metrics or KPIs.
- chart     → time-series, comparison bars, trend data.
- steps     → process, pipeline, how-it-works.
- cards     → feature list, benefit grid, capability catalog.
- timeline  → roadmap, milestones, history.
- compare   → before/after, legacy vs new, pros/cons.
- table     → pricing, feature matrix, structured comparison.
- bullets   → everything else that's a list of claims or recommendations.
- At least 4 different types in any 8+ slide deck. Vary composition on every slide.
- Add "speaker_notes" when it helps the presenter.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WRITE_HTML_DOCX — WORD DOCUMENT (HTML-driven)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Required: "filename", "title", "html".
Optional: "author", "css", "theme".

THEME (strongly encouraged — design it for THIS subject, don't reuse a template):
- Pass a "theme" object alongside "html". All fields optional, missing values fall back to a neutral palette.
- Fields: name, heading_font, body_font, mono_font, body_color, heading1_color, heading2_color,
  heading3_color, heading4_color, accent_color, secondary_color, link_color, quote_color, muted_color,
  code_bg_color, title_color.
- Colors are 6-digit hex WITHOUT the '#'. Fonts are family names (e.g. "Inter", "Playfair Display",
  "Source Serif Pro", "Georgia", "Merriweather", "Source Sans Pro", "Lora").
- A legal brief, a wedding lookbook, a cybersecurity report, and a food magazine should look visibly different.

AUTHORING RULES:
- Ship a complete HTML body fragment — <h1> for major sections, <h2> for sub-sections, <p> for prose,
  <ul>/<ol> for lists, <table><thead>…</thead><tbody>…</tbody></table> for tables, <blockquote> for pull quotes.
- Wrap the opening page in <div class="cover-page">…</div> — it renders as a full cover and forces a page break.
- If the request mentions magazine, zine, editorial spread, or lookbook: make every section visibly different —
  e.g. hero spread, pull-quote banner, two-column feature, stat grid, sidebar note, timeline, photo-essay block.
- Do NOT include <html>, <head>, <body>, or <style> tags — just the body fragment.`

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

	if shouldUseSegmentedPresentationWorkflow(task) {
		a.logger.Info("agent loop: using segmented presentation workflow", "task_id", task.ID)
		return a.runSegmentedPresentationWorkflow(ctx, task, workspaceRoot, planID, extraSystemPrompts, runtimeProfile)
	}

	if shouldUseSegmentedDocumentWorkflow(task) {
		a.logger.Info("agent loop: using segmented document workflow", "task_id", task.ID)
		return a.runSegmentedDocumentWorkflow(ctx, task, workspaceRoot, planID, extraSystemPrompts, runtimeProfile)
	}

	for iter := 0; iter < runtimeProfile.MaxIterations; iter++ {
		if ctx.Err() != nil {
			break
		}

		a.logger.Info("agent loop iteration", "task_id", task.ID, "iter", iter, "messages", len(messages))
		requiredTool := requiredOutputTool(task)
		activeToolDefs := toolDefs
		forceToolName := ""
		if requiredTool != "" && result.Completed == 0 {
			if requiredDefs := filterToolDefinitions(toolDefs, requiredTool); len(requiredDefs) > 0 {
				activeToolDefs = requiredDefs
				forceToolName = requiredTool
			}
		}

		req := provider.ChatCompletionRequest{
			Model:         a.cfg.Model,
			Stream:        true,
			MaxTokens:     runtimeProfile.MaxTokens,
			Temperature:   runtimeProfile.Temperature,
			Messages:      messages,
			Tools:         activeToolDefs,
			ForceToolName: forceToolName,
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

		if len(toolCalls) == 0 && runtimeProfile.AllowRawJSONToolArgs && result.Completed == 0 {
			if recovered, ok := recoverToolCallFromContent(content, requiredTool); ok {
				toolCalls = []provider.ToolCall{recovered}
				a.logger.Info("agent loop: recovered direct tool args from assistant content", "task_id", task.ID, "iter", iter, "tool", requiredTool)
			}
		}

		// Emit meaningful assistant text only when it is not blocking a required output tool.
		if len(content) > 5 && !(requiredTool != "" && result.Completed == 0 && len(toolCalls) == 0) {
			a.emitAgentEvent(ctx, task.ID, domain.EventTypeAgentMessage, map[string]any{
				"text": content,
			})
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
					nudge += " For presentation tasks, include the full deck object plus deck.theme_css, deck.cover_html, and HTML markup or structured content for every content slide. Compose slides with Bootstrap-style rows, columns, cards, list groups, badges, and Bootstrap Icons placeholders. For decks with 8+ total slides, keep each slide concise and complete rather than writing long prose. Do not fall back to prose planning."
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
				a.emitPresentationSlideDrafts(ctx, task.ID, tc)
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

func (a *AgentLoop) emitPresentationSlideDrafts(ctx context.Context, taskID string, tc provider.ToolCall) {
	if tc.Name != "write_pptx" {
		return
	}
	var payload presentationDraftToolPayload
	if err := json.Unmarshal([]byte(tc.Arguments), &payload); err != nil {
		return
	}
	total := len(payload.Slides) + 1
	if strings.TrimSpace(payload.Deck.CoverHTML) != "" {
		a.emitAgentEvent(ctx, taskID, domain.EventTypePresentationSlide, map[string]any{
			"index":     1,
			"total":     total,
			"kind":      "cover",
			"heading":   firstNonEmptyString(strings.TrimSpace(payload.Title), "Cover"),
			"html":      stripPresentationDraftEmoji(payload.Deck.CoverHTML),
			"theme_css": payload.Deck.ThemeCSS,
			"palette":   payload.Deck.Palette,
		})
	}
	for i, slide := range payload.Slides {
		slideHTML := presentationDraftSlideHTML(slide)
		if strings.TrimSpace(slideHTML) == "" && strings.TrimSpace(slide.Heading) == "" {
			continue
		}
		a.emitAgentEvent(ctx, taskID, domain.EventTypePresentationSlide, map[string]any{
			"index":     i + 2,
			"total":     total,
			"kind":      "slide",
			"heading":   firstNonEmptyString(strings.TrimSpace(slide.Heading), fmt.Sprintf("Slide %d", i+2)),
			"html":      slideHTML,
			"theme_css": payload.Deck.ThemeCSS,
			"palette":   payload.Deck.Palette,
		})
	}
}

type presentationDraftToolPayload struct {
	Title string `json:"title"`
	Deck  struct {
		ThemeCSS  string            `json:"theme_css"`
		CoverHTML string            `json:"cover_html"`
		Palette   map[string]string `json:"palette"`
	} `json:"deck"`
	Slides []presentationDraftSlide `json:"slides"`
}

type presentationDraftSlide struct {
	Heading     string                          `json:"heading"`
	Type        string                          `json:"type"`
	Layout      string                          `json:"layout"`
	HTML        string                          `json:"html"`
	Points      []string                        `json:"points"`
	Stats       []presentationDraftStat         `json:"stats"`
	Steps       []string                        `json:"steps"`
	Cards       []presentationDraftCard         `json:"cards"`
	Timeline    []presentationDraftTimelineItem `json:"timeline"`
	LeftColumn  *presentationDraftCompareColumn `json:"left_column"`
	RightColumn *presentationDraftCompareColumn `json:"right_column"`
	Table       *presentationDraftTableData     `json:"table"`
}

type presentationDraftStat struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

type presentationDraftCard struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

type presentationDraftTimelineItem struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

type presentationDraftCompareColumn struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
}

type presentationDraftTableData struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

func presentationDraftSlideHTML(slide presentationDraftSlide) string {
	if raw := strings.TrimSpace(stripPresentationDraftEmoji(slide.HTML)); raw != "" {
		return raw
	}

	heading := escapePresentationDraftText(firstNonEmptyString(slide.Heading, "Slide"))
	layout := strings.ToLower(firstNonEmptyString(slide.Type, slide.Layout))
	var body string
	switch {
	case len(slide.Stats) > 0:
		body = presentationDraftStatsHTML(slide.Stats)
	case len(slide.Cards) > 0:
		body = presentationDraftCardsHTML(slide.Cards)
	case len(slide.Steps) > 0:
		body = presentationDraftListHTML(slide.Steps, "list-group", "list-group-item")
	case len(slide.Timeline) > 0:
		body = presentationDraftTimelineHTML(slide.Timeline)
	case slide.LeftColumn != nil && slide.RightColumn != nil:
		body = presentationDraftCompareHTML(slide.LeftColumn, slide.RightColumn)
	case slide.Table != nil && len(slide.Table.Headers) > 0 && len(slide.Table.Rows) > 0:
		body = presentationDraftTableHTML(slide.Table)
	case len(slide.Points) > 0:
		body = presentationDraftListHTML(slide.Points, "list-group", "list-group-item")
	default:
		body = `<p class="lead">Slide content is being rendered into the final deck.</p>`
	}
	if layout == "" {
		layout = "content"
	}
	return `<div class="slide-shell content-shell" data-draft-layout="` + html.EscapeString(layout) + `"><div class="eyebrow">Slide draft</div><h2 class="display-4">` + heading + `</h2>` + body + `</div>`
}

func presentationDraftStatsHTML(stats []presentationDraftStat) string {
	var b strings.Builder
	b.WriteString(`<div class="row">`)
	for _, stat := range stats {
		value := escapePresentationDraftText(stat.Value)
		label := escapePresentationDraftText(stat.Label)
		desc := escapePresentationDraftText(stat.Desc)
		if value == "" && label == "" && desc == "" {
			continue
		}
		b.WriteString(`<div class="col-6"><div class="card"><div class="card-body"><span class="badge">Metric</span><h3 class="display-5">` + value + `</h3><p class="card-title">` + label + `</p><p class="card-text">` + desc + `</p></div></div></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func presentationDraftCardsHTML(cards []presentationDraftCard) string {
	var b strings.Builder
	b.WriteString(`<div class="row">`)
	for _, card := range cards {
		title := escapePresentationDraftText(card.Title)
		desc := escapePresentationDraftText(card.Desc)
		if title == "" && desc == "" {
			continue
		}
		icon := presentationDraftIconClass(card.Icon, "stars")
		b.WriteString(`<div class="col-4"><div class="card"><div class="card-body"><span class="icon-badge"><i class="bi ` + icon + `" aria-hidden="true"></i></span><h3 class="card-title">` + title + `</h3><p class="card-text">` + desc + `</p></div></div></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func presentationDraftListHTML(items []string, listClass, itemClass string) string {
	var b strings.Builder
	b.WriteString(`<ul class="` + listClass + `">`)
	for _, item := range items {
		text := escapePresentationDraftText(item)
		if text == "" {
			continue
		}
		b.WriteString(`<li class="` + itemClass + `">` + text + `</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

func presentationDraftTimelineHTML(items []presentationDraftTimelineItem) string {
	var b strings.Builder
	b.WriteString(`<div class="list-group">`)
	for _, item := range items {
		date := escapePresentationDraftText(item.Date)
		title := escapePresentationDraftText(item.Title)
		desc := escapePresentationDraftText(item.Desc)
		if date == "" && title == "" && desc == "" {
			continue
		}
		b.WriteString(`<div class="list-group-item"><span class="badge">` + date + `</span><h3 class="card-title">` + title + `</h3><p class="card-text">` + desc + `</p></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func presentationDraftCompareHTML(left, right *presentationDraftCompareColumn) string {
	return `<div class="row"><div class="col-6"><div class="card"><div class="card-body"><h3 class="card-title">` + escapePresentationDraftText(left.Heading) + `</h3>` + presentationDraftListHTML(left.Points, "bullet-list", "bullet-item") + `</div></div></div><div class="col-6"><div class="card"><div class="card-body"><h3 class="card-title">` + escapePresentationDraftText(right.Heading) + `</h3>` + presentationDraftListHTML(right.Points, "bullet-list", "bullet-item") + `</div></div></div></div>`
}

func presentationDraftTableHTML(table *presentationDraftTableData) string {
	var b strings.Builder
	b.WriteString(`<table class="table w-100"><thead><tr>`)
	for _, header := range table.Headers {
		b.WriteString(`<th>` + escapePresentationDraftText(header) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, row := range table.Rows {
		b.WriteString(`<tr>`)
		for _, cell := range row {
			b.WriteString(`<td>` + escapePresentationDraftText(cell) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func presentationDraftIconClass(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	var cleaned strings.Builder
	lastHyphen := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cleaned.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			cleaned.WriteByte('-')
			lastHyphen = true
		}
	}
	value = strings.Trim(cleaned.String(), "-")
	if value == "" {
		value = fallback
	}
	if strings.HasPrefix(value, "bi-") {
		return value
	}
	return "bi-" + value
}

func escapePresentationDraftText(value string) string {
	return html.EscapeString(strings.TrimSpace(stripPresentationDraftEmoji(value)))
}

func stripPresentationDraftEmoji(raw string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF:
			return -1
		case r >= 0x2600 && r <= 0x27BF:
			return -1
		case r == 0xFE0F || r == 0x200D:
			return -1
		default:
			return r
		}
	}, raw)
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
			completedStep(order, fmt.Sprintf("Draft slide %d", idx+1), fmt.Sprintf(`Compose "%s" with slide-specific content, layout, and visuals.`, heading)),
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
	case containsTaskKeyword(text,
		"presentation", "slide", "slides", "deck", "slideshow", "pptx", "powerpoint",
		"keynote", "pitch deck", "investor deck", "sales deck", "webinar deck",
		"conference talk", "talk deck", "seminar deck"):
		return "write_pptx"
	case containsTaskKeyword(text,
		// generic document asks
		"document", "doc", "word document", "word file", "docx",
		// reports & professional deliverables
		"report", "whitepaper", "white paper", "brief", "briefing", "proposal",
		"memo", "memorandum", "analysis", "case study", "policy paper",
		"research paper", "rfc", "request for comment", "charter", "playbook",
		"runbook", "handbook", "guidebook", "guide", "manual", "datasheet",
		"product spec", "spec sheet", "specification", "technical specification",
		"business plan", "market analysis", "executive summary", "status report",
		"annual report", "impact report", "sustainability report",
		// editorial / consumer
		"magazine", "zine", "editorial", "newsletter", "lookbook", "mood board",
		"moodboard", "photo essay", "visual essay", "catalog", "catalogue",
		"brochure", "pamphlet", "flyer", "leaflet", "program", "programme",
		// academic / pedagogical
		"textbook", "chapter", "lesson", "lesson plan", "workbook",
		"curriculum", "coursebook", "course material", "syllabus",
		"study guide", "lecture notes", "worksheet", "module",
		// journalism / commentary
		"article", "feature article", "essay", "op-ed", "opinion piece",
		"column", "blog post", "blog", "think piece", "long read", "longform",
		// letters & creative
		"cover letter", "letter", "statement", "manifesto", "white book",
		"writeup", "write-up", "paper"):
		return "write_html_docx"
	case containsTaskKeyword(text,
		"summary", "notes", "markdown", "readme", "changelog", "release notes",
		"meeting notes", "post-mortem", "postmortem", "retrospective", "retro",
		"standup notes", "tl;dr", "tldr", "cheat sheet", "cheatsheet"):
		return "write_markdown_report"
	case containsTaskKeyword(text,
		"data", "spreadsheet", "export", "json", "csv", "tsv", "dataset",
		"structured data", "table data", "record list", "records export"):
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

func filterToolDefinitions(defs []provider.ToolDefinition, requiredTool string) []provider.ToolDefinition {
	requiredTool = strings.TrimSpace(requiredTool)
	if requiredTool == "" {
		return defs
	}
	for _, def := range defs {
		if def.Name == requiredTool {
			return []provider.ToolDefinition{def}
		}
	}
	return defs
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
