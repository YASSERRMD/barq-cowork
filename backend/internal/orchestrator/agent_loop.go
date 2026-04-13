package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
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

Required fields: "filename", "title", "slides" (6-10 slides).
Each slide MUST have: "heading" (≤60 chars) and "type".

PLANNING ORDER — DO THIS BEFORE WRITING THE DECK:
1. Plan the full presentation first: subject, audience, narrative arc, theme, visual style, cover style, color story, motif, and the mix of slide layouts.
2. Only then plan each slide individually: choose the most suitable type, verify the slide has enough content, and check whether it needs icons, charts, diagrams, timelines, comparisons, or tables.
3. Do not use a fixed deck template or the same layout repeatedly. The structure must come from the user's subject and explicit instructions.
4. If the user explicitly requests a style, sections, sequence, visual direction, or specific slide elements, follow that exactly.
5. Before you call write_pptx, mentally audit every slide: proper heading, proper content density, proper layout choice, and proper icons/visuals for that slide.
6. The write_pptx tool validates slide fit before it renders. Do not send thin, repetitive, or underfilled slides.
7. Use a complete "deck" object on EVERY write_pptx tool call. It is required. Fill subject, audience, narrative, theme, visual_style, cover_style, color_story, motif, kicker, a full palette, and deck.design render directives.
8. Examples below show JSON shape only. Do not copy example palette values, repeated classroom amber tones, or the same cover layout across different decks unless the user explicitly asks for that direction.
9. Never put internal planning metadata on the slides. The final presentation should show user-facing content only.
10. The HTML preview and the downloaded PPTX must look like the same deck. Choose design directives that the native renderer can follow, and add slide-level design objects when a slide needs a specific composition.
11. Default to a refined contemporary presentation aesthetic. Avoid dated corporate blue-orange palettes, thick office-style borders, repetitive bullet slabs, and generic 2000s deck patterns unless the user explicitly asks for them.
12. For AI, digital-future, innovation, or forward-looking subjects, prefer airy split/gallery compositions, restrained surfaces, and a fresh palette rather than classroom amber or old business-deck blue.

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
- Slide 1: first slide auto-becomes the cover. Set "type":"bullets" or omit type for slide 1 and let it auto-convert.
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
) ExecuteResult {
	const maxIter = 15

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

	messages := []provider.ChatMessage{
		{Role: "system", Content: agentSystemPrompt},
		{Role: "user", Content: userMsg},
	}

	toolDefs := a.registry.Definitions()

	var result ExecuteResult
	stepOrder := 0
	forcedToolNudges := 0

	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
		"message": "agent loop started",
	})

	for iter := 0; iter < maxIter; iter++ {
		if ctx.Err() != nil {
			break
		}

		a.logger.Info("agent loop iteration", "task_id", task.ID, "iter", iter, "messages", len(messages))

		req := provider.ChatCompletionRequest{
			Model:       a.cfg.Model,
			Stream:      true,
			MaxTokens:   2048,
			Temperature: taskTemperature(task),
			Messages:    messages,
			Tools:       toolDefs,
		}

		ch, err := provider.ChatWithRetry(ctx, a.prov, a.cfg, req, provider.DefaultRetryConfig(), a.logger)
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

		// No tool calls → agent decided it is done
		if len(toolCalls) == 0 {
			requiredTool := requiredOutputTool(task)
			if requiredTool != "" && result.Completed == 0 && forcedToolNudges < 2 {
				forcedToolNudges++
				messages = append(messages, provider.ChatMessage{
					Role:    "assistant",
					Content: content,
				})
				messages = append(messages, provider.ChatMessage{
					Role: "system",
					Content: fmt.Sprintf(
						"You have not called any tool yet. Stop planning and call %s now. "+
							"Produce the output file in this turn and do not answer with prose only.",
						requiredTool,
					),
				})
				a.logger.Warn("agent loop: no tool calls, nudging required output tool", "task_id", task.ID, "iter", iter, "tool", requiredTool, "nudges", forcedToolNudges)
				continue
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
			stepOrder++

			// Create step record
			now := time.Now().UTC()
			step := &domain.PlanStep{
				ID:          uuid.NewString(),
				PlanID:      planID,
				Order:       stepOrder,
				Title:       tc.Name,
				Description: "Tool call: " + tc.Name,
				ToolName:    tc.Name,
				ToolInput:   tc.Arguments,
				Status:      domain.StepStatusRunning,
				StartedAt:   &now,
			}
			_ = a.plans.CreateStep(ctx, step)

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

func taskTemperature(task *domain.Task) float64 {
	if requiredOutputTool(task) == "write_pptx" {
		return 0.55
	}
	return 0.3
}

func containsTaskKeyword(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
