// Package orchestrator contains the task planning and execution engine.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/google/uuid"
)

// plannerSystemPrompt instructs the LLM to return a JSON execution plan.
const plannerSystemPrompt = `You are an expert task planner for an AI agent workspace called Barq Cowork.

Given a task title, description, and optional project instructions, decompose the work into a sequence of concrete, executable steps.

Available tools:
- list_files   : list files/dirs in workspace path
- read_file    : read text content of a file
- write_file   : write text content to a file (creates dirs)
- create_folder: create a directory
- search_files : search file contents for a pattern
- http_fetch   : fetch a URL (GET free; POST requires approval)
- write_markdown_report: write a formatted .md report to reports/
- write_xlsx   : write a structured Excel workbook as .xlsx to spreadsheets/
- export_json  : write structured data as .json to exports/

Rules:
1. Keep steps minimal and focused — one action per step.
2. For pure reasoning or summarisation steps, set tool_name to "" and tool_input to {}.
3. tool_input must be a valid JSON object matching the tool's schema.
4. Do NOT include any text outside the JSON object.

Return ONLY this JSON structure (no markdown fences, no explanation):
{
  "steps": [
    {
      "title": "Short action title",
      "description": "What this step does and why",
      "tool_name": "tool_name_or_empty",
      "tool_input": {}
    }
  ]
}`

// rawStep is the JSON shape the LLM returns for each step.
type rawStep struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	ToolName    string         `json:"tool_name"`
	ToolInput   map[string]any `json:"tool_input"`
}

type rawPlan struct {
	Steps []rawStep `json:"steps"`
}

// LLMProviderGetter is the minimal interface Planner needs from the registry.
type LLMProviderGetter interface {
	Get(name string) (provider.LLMProvider, bool)
}

// ProjectMemory is the narrow interface the Planner needs for context retrieval.
// Passing nil disables context injection without error.
type ProjectMemory interface {
	Recall(projectID, query string) ([]domain.MemoryEntry, error)
}

// Planner generates a structured execution plan from a task using an LLM.
type Planner struct {
	registry LLMProviderGetter
	memory   ProjectMemory // may be nil
	logger   *slog.Logger
}

// NewPlanner creates a Planner. Pass nil for memory to disable context injection.
func NewPlanner(registry LLMProviderGetter, memory ProjectMemory, logger *slog.Logger) *Planner {
	return &Planner{registry: registry, memory: memory, logger: logger}
}

// Plan decomposes a task into a domain.Plan using the provided LLM config.
// On any LLM or parse error it falls back to a single "execute" step so
// the executor always has something to run.
func (p *Planner) Plan(
	ctx context.Context,
	task *domain.Task,
	project *domain.Project,
	cfg provider.ProviderConfig,
) (*domain.Plan, error) {
	prov, ok := p.registry.Get(cfg.ProviderName)
	if !ok {
		return p.fallbackPlan(task, fmt.Sprintf("provider %q not registered", cfg.ProviderName)), nil
	}

	// Retrieve project context files and inject them into the prompt.
	var memEntries []domain.MemoryEntry
	if p.memory != nil {
		if entries, err := p.memory.Recall(task.ProjectID, task.Title); err == nil {
			memEntries = entries
		} else {
			p.logger.Warn("memory recall failed, continuing without context", "error", err)
		}
	}

	userMsg := buildPlanningPrompt(task, project, memEntries)

	req := provider.ChatCompletionRequest{
		Model:       cfg.Model,
		Stream:      false,
		MaxTokens:   1024,
		Temperature: 0.2,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: plannerSystemPrompt},
			{Role: "user", Content: userMsg},
		},
	}

	ch, err := provider.ChatWithRetry(ctx, prov, cfg, req, provider.DefaultRetryConfig(), p.logger)
	if err != nil {
		p.logger.Warn("planner LLM call failed after retries, using fallback", "error", err)
		return p.fallbackPlan(task, err.Error()), nil
	}

	// Collect full response
	var sb strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		sb.WriteString(chunk.ContentDelta)
	}
	raw := sb.String()
	p.logger.Info("planner raw response", "task_id", task.ID, "length", len(raw), "snippet", truncate(raw, 300))

	parsed, err := parsePlan(raw)
	if err != nil {
		p.logger.Warn("planner parse failed, using fallback", "error", err, "raw_snippet", truncate(raw, 200))
		return p.fallbackPlan(task, "could not parse LLM plan: "+err.Error()), nil
	}
	if len(parsed.Steps) == 0 {
		return p.fallbackPlan(task, "LLM returned empty plan"), nil
	}

	return buildDomainPlan(task.ID, parsed), nil
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func buildPlanningPrompt(task *domain.Task, project *domain.Project, memory []domain.MemoryEntry) string {
	var sb strings.Builder
	sb.WriteString("Task Title: " + task.Title + "\n")
	if task.Description != "" {
		sb.WriteString("Task Description:\n" + task.Description + "\n")
	}
	if project != nil && project.Instructions != "" {
		sb.WriteString("\nProject Instructions:\n" + project.Instructions + "\n")
	}
	// Inject context files (project memory) so the LLM has full project context.
	if len(memory) > 0 {
		sb.WriteString("\n── Project Context Files ──\n")
		for _, e := range memory {
			sb.WriteString(fmt.Sprintf("\n### %s\n%s\n", e.Label, e.Content))
		}
	}
	return sb.String()
}

var jsonBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*({.*?})\\s*```")

// parsePlan tries to extract a rawPlan from an LLM response string.
func parsePlan(raw string) (*rawPlan, error) {
	raw = strings.TrimSpace(raw)

	// Try direct JSON parse first
	var plan rawPlan
	if err := json.Unmarshal([]byte(raw), &plan); err == nil {
		return &plan, nil
	}

	// Try to extract JSON from a markdown code block
	if m := jsonBlockRe.FindStringSubmatch(raw); len(m) > 1 {
		if err := json.Unmarshal([]byte(m[1]), &plan); err == nil {
			return &plan, nil
		}
	}

	// Try to find the first { ... } blob in the response
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		blob := raw[start : end+1]
		if err := json.Unmarshal([]byte(blob), &plan); err == nil {
			return &plan, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON plan found in response")
}

func buildDomainPlan(taskID string, raw *rawPlan) *domain.Plan {
	planID := uuid.NewString()
	now := time.Now().UTC()

	steps := make([]*domain.PlanStep, len(raw.Steps))
	for i, rs := range raw.Steps {
		toolInput := "{}"
		if rs.ToolInput != nil {
			if b, err := json.Marshal(rs.ToolInput); err == nil {
				toolInput = string(b)
			}
		}
		steps[i] = &domain.PlanStep{
			ID:          uuid.NewString(),
			PlanID:      planID,
			Order:       i + 1,
			Title:       rs.Title,
			Description: rs.Description,
			ToolName:    rs.ToolName,
			ToolInput:   toolInput,
			ToolOutput:  "",
			Status:      domain.StepStatusPending,
		}
	}

	return &domain.Plan{
		ID:        planID,
		TaskID:    taskID,
		Steps:     steps,
		CreatedAt: now,
	}
}

// fallbackPlan returns a single placeholder step when planning fails.
func (p *Planner) fallbackPlan(task *domain.Task, reason string) *domain.Plan {
	p.logger.Warn("using fallback plan", "task_id", task.ID, "reason", reason)
	planID := uuid.NewString()
	return &domain.Plan{
		ID:     planID,
		TaskID: task.ID,
		Steps: []*domain.PlanStep{
			{
				ID:          uuid.NewString(),
				PlanID:      planID,
				Order:       1,
				Title:       "Execute task",
				Description: task.Description,
				ToolName:    "",
				ToolInput:   "{}",
				Status:      domain.StepStatusPending,
			},
		},
		CreatedAt: time.Now().UTC(),
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
