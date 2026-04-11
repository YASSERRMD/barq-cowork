package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
- "presentation", "slides", "deck", "slideshow", "pptx", "powerpoint" → MUST use write_pptx (creates a real .pptx file)
- "document", "report", "doc", "word", "brief", "proposal", "writeup", "paper" → MUST use write_docx (creates a real .docx Word document)
- "summary", "notes", "markdown" → use write_markdown_report (.md file)
- "data", "spreadsheet", "export" → use export_json (.json file)
- general file → use write_file

RULES:
1. Call ONE tool at a time.
2. ALWAYS produce at least one output file. Never finish without writing a file.

3. For write_pptx: provide "title", "subtitle", and "slides" (6-10 slides). Each slide MUST have a "heading" and one of:
   - "points": string array — for bullets/steps/cards layouts
   - "stats": array of {"value","label","desc"} — for stats layout
   - "categories" + "series": for chart layout (see below)

   LAYOUT SELECTION GUIDE — mix layouts across slides for visual variety:
   - "bullets" — detailed explanations, lists, insights (default)
   - "stats" — KPIs, metrics, percentages. Provide "stats": [{"value":"92%","label":"Accuracy","desc":"on test set"}]
   - "steps" — process/workflow, onboarding, pipelines, methodology. Provide ordered "points".
   - "cards" — features, benefits, categories (4-6 short items under 80 chars). Provide "points".
   - "chart" — full-slide chart. Provide "chart_type" ("column"|"bar"|"line"|"pie"|"doughnut"), "categories": ["A","B","C"], "series": [{"name":"S1","values":[1,2,3]}]. Optional "y_label".
   - "timeline" — milestone/roadmap slides. Provide "points" as milestone descriptions (dates or phases).
   - "compare" — side-by-side comparison. Provide "left_title", "right_title", "left_points", "right_points" (or plain "points" split by " vs ").
   - Aim for at least 3 different layouts in an 8-slide deck. Use "chart" for any slide with numerical data series. Use "timeline" for roadmaps/phases. Use "compare" for before/after or A-vs-B analysis.

4. For write_docx: provide "filename", "title", "author" (optional), and "sections" array. Each section has:
   - "heading": section title
   - "level": 1 (major) or 2 (sub-section), default 1
   - "content": paragraph text or bullet list (use "• item" prefix for bullets)
   - "table": optional {headers:[], rows:[[],[]]} for a table in this section
   Use multiple sections to create a well-structured professional document with executive summary, body sections, and conclusion.

5. Stop after the file is written.
6. Max 15 tool calls total.`

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
			Temperature: 0.3,
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

		// No tool calls → agent decided it is done
		if len(toolCalls) == 0 {
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
				a.maybeRecordAgentArtifact(ctx, step, task)
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

func (a *AgentLoop) maybeRecordAgentArtifact(ctx context.Context, step *domain.PlanStep, task *domain.Task) {
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

	artifact := &domain.Artifact{
		ID:          uuid.NewString(),
		TaskID:      task.ID,
		ProjectID:   task.ProjectID,
		Name:        output.Data.Path,
		Type:        artType,
		ContentPath: output.Data.Path,
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
