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

const segmentedPPTXPlanTimeout = 75 * time.Second
const segmentedPPTXSlideTimeout = 8 * time.Second

var segmentedHTMLTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)
var segmentedHexColorPattern = regexp.MustCompile(`(?i)^[0-9a-f]{6}$`)

type segmentedPPTXArgs struct {
	Filename string                   `json:"filename"`
	Title    string                   `json:"title"`
	Subtitle string                   `json:"subtitle,omitempty"`
	Deck     segmentedPPTXDeck        `json:"deck"`
	Slides   []presentationDraftSlide `json:"slides"`
}

type segmentedPPTXDeck struct {
	Archetype   string            `json:"archetype,omitempty"`
	Subject     string            `json:"subject"`
	Audience    string            `json:"audience"`
	Narrative   string            `json:"narrative"`
	Theme       string            `json:"theme"`
	VisualStyle string            `json:"visual_style"`
	CoverStyle  string            `json:"cover_style"`
	ColorStory  string            `json:"color_story"`
	Motif       string            `json:"motif"`
	Kicker      string            `json:"kicker"`
	Design      map[string]string `json:"design,omitempty"`
	Palette     map[string]string `json:"palette"`
	ThemeCSS    string            `json:"theme_css"`
	CoverHTML   string            `json:"cover_html"`
}

type segmentedPPTXPlan struct {
	Filename string                    `json:"filename"`
	Title    string                    `json:"title"`
	Subtitle string                    `json:"subtitle"`
	Deck     segmentedPPTXDeck         `json:"deck"`
	Stages   []string                  `json:"stages"`
	Slides   []segmentedPPTXSlideBrief `json:"slides"`
}

type segmentedPPTXSlideBrief struct {
	Heading     string                          `json:"heading"`
	Type        string                          `json:"type"`
	Purpose     string                          `json:"purpose"`
	Visual      string                          `json:"visual"`
	Icon        string                          `json:"icon"`
	Points      []string                        `json:"points"`
	Stats       []presentationDraftStat         `json:"stats"`
	Steps       []string                        `json:"steps"`
	Cards       []presentationDraftCard         `json:"cards"`
	Timeline    []presentationDraftTimelineItem `json:"timeline"`
	LeftColumn  *presentationDraftCompareColumn `json:"left_column"`
	RightColumn *presentationDraftCompareColumn `json:"right_column"`
	Table       *presentationDraftTableData     `json:"table"`
}

func shouldUseSegmentedPresentationWorkflow(task *domain.Task) bool {
	_, _, ok := requestedPresentationSlideTargets(task)
	return ok
}

func requestedPresentationSlideTargets(task *domain.Task) (contentSlides int, totalSlides int, ok bool) {
	if requiredOutputTool(task) != "write_pptx" {
		return 0, 0, false
	}
	text := strings.TrimSpace(task.Title + " " + task.Description)
	matches := requestedSlideCountPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0, 0, false
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil || count < 1 || count > 30 {
		return 0, 0, false
	}
	isContentCount := strings.TrimSpace(matches[2]) != ""
	if isContentCount {
		return count, count + 1, true
	}
	if count <= 1 {
		return 0, 1, true
	}
	return count - 1, count, true
}

func (a *AgentLoop) runSegmentedPresentationWorkflow(
	ctx context.Context,
	task *domain.Task,
	workspaceRoot string,
	planID string,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) ExecuteResult {
	contentSlides, totalSlides, ok := requestedPresentationSlideTargets(task)
	if !ok {
		contentSlides, totalSlides = 6, 7
	}

	var result ExecuteResult
	now := time.Now().UTC()
	planStep := &domain.PlanStep{
		ID:          uuid.NewString(),
		PlanID:      planID,
		Order:       1,
		Title:       "Plan deck system",
		Description: fmt.Sprintf("Plan the full %d-slide presentation system before drafting slides.", totalSlides),
		Status:      domain.StepStatusRunning,
		StartedAt:   &now,
	}
	_ = a.plans.CreateStep(ctx, planStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
		"step_id": planStep.ID, "tool": "presentation_plan", "mode": "segmented",
	})

	provisionalPlan := fallbackSegmentedPPTXPlan(task, contentSlides, totalSlides)
	normalizeSegmentedPPTXPlan(&provisionalPlan, task, contentSlides)
	a.emitSegmentedPresentationDrafts(ctx, task.ID, provisionalPlan, totalSlides)

	plan, err := a.generateSegmentedPPTXPlan(ctx, task, contentSlides, totalSlides, extraSystemPrompts, runtimeProfile)
	completed := time.Now().UTC()
	planStep.CompletedAt = &completed
	if err != nil {
		a.logger.Warn("segmented pptx: LLM planning failed, using local presentation plan", "task_id", task.ID, "error", err)
		plan = provisionalPlan
		planStep.ToolOutput = marshalJSON(map[string]any{"status": "ok", "fallback": true, "warning": err.Error(), "slides": totalSlides, "title": plan.Title})
	} else {
		normalizeSegmentedPPTXPlan(&plan, task, contentSlides)
		planStep.ToolOutput = marshalJSON(map[string]any{"status": "ok", "fallback": false, "slides": totalSlides, "title": plan.Title})
	}

	planStep.Status = domain.StepStatusCompleted
	_ = a.plans.UpdateStep(ctx, planStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
		"step_id": planStep.ID, "tool": "presentation_plan", "status": planStep.Status,
	})
	a.emitSegmentedPresentationDrafts(ctx, task.ID, plan, totalSlides)

	finalSlides := plannedSegmentedSlides(plan)
	stepOrder := 1
	skipSlideLLM := false
	for i, brief := range plan.Slides {
		stepOrder++
		stepStarted := time.Now().UTC()
		step := &domain.PlanStep{
			ID:          uuid.NewString(),
			PlanID:      planID,
			Order:       stepOrder,
			Title:       fmt.Sprintf("Draft slide %d", i+1),
			Description: fmt.Sprintf(`Generate slide %d of %d: "%s".`, i+2, totalSlides, firstNonEmptyString(brief.Heading, "Untitled slide")),
			Status:      domain.StepStatusRunning,
			StartedAt:   &stepStarted,
		}
		_ = a.plans.CreateStep(ctx, step)
		a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepStarted, map[string]any{
			"step_id": step.ID, "tool": "presentation_slide", "slide": i + 2,
		})

		slide := finalSlides[i]
		var slideErr error
		if !skipSlideLLM {
			slide, slideErr = a.generateSegmentedPPTXSlide(ctx, task, plan, brief, i, totalSlides, runtimeProfile)
			slide = completeSegmentedSlide(slide, plan, brief, i, totalSlides)
			if slideErr != nil {
				skipSlideLLM = true
				slide = finalSlides[i]
				a.logger.Warn("segmented pptx: slide LLM failed, using planned briefs for remaining slides", "task_id", task.ID, "slide", i+1, "error", slideErr)
			} else {
				finalSlides[i] = slide
			}
		} else {
			slideErr = fmt.Errorf("skipped per-slide LLM after earlier slide timeout")
		}

		stepDone := time.Now().UTC()
		step.CompletedAt = &stepDone
		step.Status = domain.StepStatusCompleted
		if slideErr != nil {
			step.ToolOutput = marshalJSON(map[string]any{"status": "ok", "fallback": true, "warning": slideErr.Error()})
		} else {
			step.ToolOutput = marshalJSON(map[string]any{"status": "ok", "fallback": false})
		}
		_ = a.plans.UpdateStep(ctx, step)
		a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
			"step_id": step.ID, "tool": "presentation_slide", "status": step.Status, "slide": i + 2,
		})
		a.emitSegmentedPresentationDraft(ctx, task.ID, i+2, totalSlides, "slide", finalSlides[i].Heading, presentationDraftSlideHTML(finalSlides[i]), plan.Deck)
	}

	args := segmentedPPTXArgs{
		Filename: firstNonEmptyString(plan.Filename, slugifySegmentedFilename(task.Title)),
		Title:    firstNonEmptyString(plan.Title, task.Title, "Presentation"),
		Subtitle: plan.Subtitle,
		Deck:     plan.Deck,
		Slides:   finalSlides,
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		result.Failed++
		return result
	}

	tc := provider.ToolCall{ID: "segmented-write-pptx", Name: "write_pptx", Arguments: string(argsBytes)}
	renderStep := createToolPlanStep(planID, stepOrder, tc)
	renderStep.Title = "Render PowerPoint file"
	renderStep.Description = fmt.Sprintf("Export %d generated slides into the final .pptx.", totalSlides)
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

func (a *AgentLoop) generateSegmentedPPTXPlan(
	ctx context.Context,
	task *domain.Task,
	contentSlides int,
	totalSlides int,
	extraSystemPrompts []string,
	runtimeProfile agentRuntimeProfile,
) (segmentedPPTXPlan, error) {
	var out segmentedPPTXPlan
	contextText := compactSegmentedExtraContext(extraSystemPrompts)
	user := fmt.Sprintf(`Create the plan JSON for a PowerPoint deck.

Task title: %s
Task details: %s
Additional context: %s

Required count:
- Total slides: %d
- Content slides in slides array: %d
- The cover is deck.cover_html, not a slides[] entry.

Return ONLY compact valid JSON with this exact shape:
{
  "filename":"kebab-case-name",
  "title":"deck title",
  "subtitle":"short subtitle",
  "deck": {
    "archetype":"educational explainer | proposal | executive brief | ...",
    "subject":"specific subject",
    "audience":"specific audience",
    "narrative":"specific story arc",
    "theme":"domain theme",
    "visual_style":"subject-specific visual direction",
    "cover_style":"specific cover composition",
    "color_story":"specific color direction",
    "motif":"semantic motif",
    "kicker":"short cover kicker",
    "design":{"composition":"asym grid","density":"dense","shape_language":"crisp","accent_mode":"badge","hero_layout":"information"},
    "palette":{"background":"F6F8FB","card":"FFFFFF","accent":"0EA5E9","accent2":"14B8A6","text":"0F172A","muted":"475569","border":"CBD5E1"},
    "theme_css":".row{display:flex;gap:18px}.card{border:1px solid var(--border);border-radius:20px}.card-body{padding:22px}.badge{border-radius:999px}.list-group-item{border-left:5px solid var(--accent)}",
    "cover_html":"compact inner cover HTML using Bootstrap classes and one Bootstrap Icon placeholder"
  },
  "stages":["opening stage","explanation stage","practice/safety stage","closing stage"],
  "slides":[{"heading":"specific slide title","type":"cards|stats|steps|timeline|compare|table","purpose":"why this slide exists","visual":"visual composition","icon":"bootstrap-icon-name","points":["specific point","specific point","specific point"],"cards":[{"icon":"bootstrap-icon-name","title":"specific card title","desc":"specific card copy"}]}]
}

Rules:
- Include exactly %d concise slide briefs in slides[].
- Every slide brief must have heading, type, purpose, visual, icon, and at least 3 concrete points.
- Prefer structured fields where useful: cards, stats, steps, timeline, left_column/right_column, or table.
- Do not include full slide HTML in the plan; HTML enrichment is optional later.
- stages length must be 3 to 5.
- Keep the whole JSON compact. Do not pretty print. Do not add fields beyond the shape above.
- Keep deck.theme_css under 700 characters and deck.cover_html under 900 characters.
- No emoji anywhere.
- Use real Bootstrap Icon placeholders in HTML only: <i class="bi bi-stars" aria-hidden="true"></i>.
- Do not include visible slide counters.
- Keep copy readable for projector use: no tiny text, no empty decorative panels.
- Make the design decisions from the subject, not from a fixed template.`, task.Title, task.Description, contextText, totalSlides, contentSlides, contentSlides)

	err := a.chatSegmentedJSON(ctx, runtimeProfile, segmentedPlanTokenBudget(contentSlides), segmentedPPTXPlanTimeout, []provider.ChatMessage{
		{Role: "system", Content: segmentedPPTXSystemPrompt()},
		{Role: "user", Content: user},
	}, &out)
	return out, err
}

func (a *AgentLoop) generateSegmentedPPTXSlide(
	ctx context.Context,
	task *domain.Task,
	plan segmentedPPTXPlan,
	brief segmentedPPTXSlideBrief,
	index int,
	totalSlides int,
	runtimeProfile agentRuntimeProfile,
) (presentationDraftSlide, error) {
	var out presentationDraftSlide
	briefBytes, _ := json.Marshal(brief)
	deckBytes, _ := json.Marshal(map[string]any{
		"title":        plan.Title,
		"audience":     plan.Deck.Audience,
		"theme":        plan.Deck.Theme,
		"visual_style": plan.Deck.VisualStyle,
		"color_story":  plan.Deck.ColorStory,
		"stages":       plan.Stages,
		"palette":      plan.Deck.Palette,
		"theme_css":    plan.Deck.ThemeCSS,
	})
	user := fmt.Sprintf(`Generate one PowerPoint slide as JSON.

Deck summary JSON:
%s

Task title: %s
Slide position: %d of %d total slides
Slide role: %s
Slide brief JSON:
%s

Return ONLY valid JSON:
{
  "heading": "same or improved heading, max 60 chars",
  "type": "html",
  "html": "inner slide HTML",
  "points": ["specific point", "specific point", "specific point"]
}

HTML requirements:
- Use Bootstrap-style classes: container-fluid, row, col-*, card, card-body, badge, list-group, list-group-item, icon-badge.
- Use Bootstrap Icons placeholders only, for example <i class="bi bi-lightbulb" aria-hidden="true"></i>.
- No emoji, no scripts, no external assets, no generated SVG.
- Do not include a visible "Slide X of Y" counter.
- Build a dense, modern information layout from the role and brief; do not output a heading plus one paragraph.
- Body copy should be projector-readable, generally 20-24px through class/style choices.
- Keep the HTML compact enough for one slide.`, string(deckBytes), task.Title, index+2, totalSlides, segmentedSlideRole(index, len(plan.Slides), plan.Stages), string(briefBytes))

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 1800, segmentedPPTXSlideTimeout, []provider.ChatMessage{
		{Role: "system", Content: segmentedPPTXSystemPrompt()},
		{Role: "user", Content: user},
	}, &out)
	return out, err
}

func segmentedPPTXSystemPrompt() string {
	return "You are a senior presentation designer. Return only valid JSON, no markdown and no prose. " +
		"Design for the exported PowerPoint file. Use subject-specific layout decisions, Bootstrap-style components, official Bootstrap Icons placeholders, and readable projector-scale copy. Never use emoji."
}

func (a *AgentLoop) chatSegmentedJSON(
	ctx context.Context,
	runtimeProfile agentRuntimeProfile,
	maxTokens int,
	timeout time.Duration,
	messages []provider.ChatMessage,
	out any,
) error {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	retry := runtimeProfile.Retry
	retry.MaxAttempts = 1
	req := provider.ChatCompletionRequest{
		Model:       a.cfg.Model,
		Stream:      false,
		MaxTokens:   maxTokens,
		Temperature: runtimeProfile.Temperature,
		Messages:    messages,
	}
	if segmentedProviderSupportsFastJSONMode(a.prov.Name()) {
		req.ResponseFormat = map[string]any{"type": "json_object"}
	}
	ch, err := provider.ChatWithRetry(callCtx, a.prov, a.cfg, req, retry, a.logger)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "response_format") && !strings.Contains(err.Error(), "400") {
			return err
		}
		req.ResponseFormat = nil
		ch, err = provider.ChatWithRetry(callCtx, a.prov, a.cfg, req, retry, a.logger)
		if err != nil {
			return err
		}
	}
	var content strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		content.WriteString(chunk.ContentDelta)
	}
	raw := strings.TrimSpace(content.String())
	candidate := extractJSONObject(raw)
	if candidate == "" {
		candidate = firstJSONObjectFromJSONArray(raw)
	}
	if candidate == "" {
		return fmt.Errorf("provider did not return valid JSON; response snippet: %s", truncate(raw, 300))
	}
	if err := json.Unmarshal([]byte(candidate), out); err != nil {
		return fmt.Errorf("parse provider JSON: %w", err)
	}
	return nil
}

func segmentedProviderSupportsFastJSONMode(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "zai", "ollama":
		return false
	default:
		return true
	}
}

func firstJSONObjectFromJSONArray(raw string) string {
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &items); err != nil || len(items) == 0 {
		return ""
	}
	first := strings.TrimSpace(string(items[0]))
	if !strings.HasPrefix(first, "{") || !json.Valid([]byte(first)) {
		return ""
	}
	return first
}

func segmentedPlanTokenBudget(contentSlides int) int {
	budget := 2400 + contentSlides*420
	if budget < 3000 {
		return 3000
	}
	if budget > 6200 {
		return 6200
	}
	return budget
}

func plannedSegmentedSlides(plan segmentedPPTXPlan) []presentationDraftSlide {
	slides := make([]presentationDraftSlide, 0, len(plan.Slides))
	totalSlides := len(plan.Slides) + 1
	for i, brief := range plan.Slides {
		slides = append(slides, completeSegmentedSlide(presentationDraftSlide{}, plan, brief, i, totalSlides))
	}
	return slides
}

func (a *AgentLoop) emitSegmentedPresentationDrafts(ctx context.Context, taskID string, plan segmentedPPTXPlan, totalSlides int) {
	a.emitSegmentedPresentationDraft(ctx, taskID, 1, totalSlides, "cover", plan.Title, plan.Deck.CoverHTML, plan.Deck)
	for i, slide := range plannedSegmentedSlides(plan) {
		a.emitSegmentedPresentationDraft(ctx, taskID, i+2, totalSlides, "slide", slide.Heading, presentationDraftSlideHTML(slide), plan.Deck)
	}
}

func fallbackSegmentedPPTXPlan(task *domain.Task, contentSlides, totalSlides int) segmentedPPTXPlan {
	subject := inferSegmentedSubject(task)
	filename := slugifySegmentedFilename(subject)
	if filename == "presentation" {
		filename = slugifySegmentedFilename(task.Title)
	}
	plan := segmentedPPTXPlan{
		Filename: filename,
		Title:    subject,
		Subtitle: "A concise presentation generated from the request",
		Deck: segmentedPPTXDeck{
			Archetype:   "structured briefing",
			Subject:     subject,
			Audience:    "the intended audience",
			Narrative:   "context -> key ideas -> practical next steps",
			Theme:       "modern briefing",
			VisualStyle: "editorial field notes with strong visual hierarchy",
			CoverStyle:  "split hero cover with agenda panels",
			ColorStory:  "soft slate background with high-contrast accent colors",
			Motif:       "connected insight cards",
			Kicker:      "Presentation",
			Design: map[string]string{
				"composition":    "asymmetric grid",
				"density":        "dense",
				"shape_language": "crisp cards",
				"accent_mode":    "rail and badge",
				"hero_layout":    "information-led",
			},
			Palette:  fallbackSegmentedPalette(subject),
			ThemeCSS: defaultSegmentedThemeCSS(),
		},
		Stages: []string{
			"frame the subject and audience need",
			"explain the most important ideas",
			"make the implications practical",
			"close with a clear action path",
		},
	}
	plan.Deck.CoverHTML = defaultSegmentedCoverHTML(plan)
	plan.Slides = fallbackSegmentedSlideBriefs(subject, contentSlides, totalSlides)
	return plan
}

func fallbackSegmentedPalette(subject string) map[string]string {
	palettes := []map[string]string{
		{"background": "FFF7ED", "card": "FFFBF5", "accent": "F97316", "accent2": "16A34A", "text": "111827", "muted": "57534E", "border": "FED7AA"},
		{"background": "F8FAFC", "card": "FFFFFF", "accent": "0F766E", "accent2": "EA580C", "text": "0F172A", "muted": "475569", "border": "CBD5E1"},
		{"background": "F5F3FF", "card": "FFFFFF", "accent": "6D28D9", "accent2": "0891B2", "text": "18181B", "muted": "52525B", "border": "DDD6FE"},
		{"background": "ECFEFF", "card": "F8FAFC", "accent": "0369A1", "accent2": "0D9488", "text": "082F49", "muted": "475569", "border": "BAE6FD"},
		{"background": "F7FEE7", "card": "FFFFFF", "accent": "4D7C0F", "accent2": "CA8A04", "text": "1A2E05", "muted": "4D5D2A", "border": "D9F99D"},
	}
	lower := strings.ToLower(subject)
	if strings.Contains(lower, "india") {
		return cloneSegmentedPalette(palettes[0])
	}
	sum := 0
	for _, r := range lower {
		sum += int(r)
	}
	return cloneSegmentedPalette(palettes[sum%len(palettes)])
}

func cloneSegmentedPalette(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func inferSegmentedSubject(task *domain.Task) string {
	text := strings.TrimSpace(task.Title)
	if text == "" {
		text = strings.TrimSpace(task.Description)
	}
	if text == "" {
		return "Presentation"
	}
	cleaned := requestedSlideCountPattern.ReplaceAllString(text, "")
	replacements := []string{
		"presentation about", "presentation on", "presentation for",
		"powerpoint about", "powerpoint on", "powerpoint for",
		"pptx about", "pptx on", "pptx for",
		"generate", "create", "make", "build", "deck about", "deck on", "slide about", "slide on",
	}
	cleaned = strings.ToLower(cleaned)
	for _, token := range replacements {
		cleaned = strings.ReplaceAll(cleaned, token, "")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "Presentation"
	}
	words := strings.Fields(cleaned)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		if len(word) <= 3 && word == strings.ToUpper(word) {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func fallbackSegmentedSlideBriefs(subject string, contentSlides, totalSlides int) []segmentedPPTXSlideBrief {
	roles := []struct {
		heading string
		kind    string
		purpose string
		visual  string
		icon    string
	}{
		{"Why It Matters", "cards", "frame the topic and explain why the audience should care", "three context cards", "bullseye"},
		{"Landscape and Context", "cards", "map the background forces shaping the topic", "context map cards", "compass"},
		{"People and Stakeholders", "cards", "show who is affected and what each group needs", "stakeholder cards", "people"},
		{"Core Ideas", "cards", "break the subject into the few ideas the audience must understand", "dense concept cards", "diagram-3"},
		{"Risks and Guardrails", "compare", "separate helpful practice from avoidable risk", "two-column comparison", "shield-check"},
		{"Signals to Track", "stats", "give the audience concrete indicators to monitor", "metric cards", "bar-chart-line"},
		{"Decision Framework", "table", "turn the topic into practical choices", "compact decision table", "grid-3x3-gap"},
		{"Operating Playbook", "steps", "show how to apply the subject in a practical sequence", "numbered action flow", "list-check"},
		{"Opportunities and Leverage", "cards", "highlight where the audience has the most room to act", "opportunity mosaic", "lightning-charge"},
		{"Global or External Role", "timeline", "connect the topic to external milestones, markets, or influence", "milestone strip", "globe2"},
		{"Action Plan", "steps", "close with immediate next steps", "roadmap steps", "check2-circle"},
	}
	slides := make([]segmentedPPTXSlideBrief, 0, contentSlides)
	for i := 0; i < contentSlides; i++ {
		role := roles[i%len(roles)]
		if i == contentSlides-1 && contentSlides > 1 {
			role = roles[len(roles)-1]
		}
		heading := role.heading
		if contentSlides <= 3 {
			heading = fmt.Sprintf("%s: %s", subject, role.heading)
		}
		brief := segmentedPPTXSlideBrief{
			Heading: heading,
			Type:    role.kind,
			Purpose: role.purpose,
			Visual:  role.visual,
			Icon:    role.icon,
			Points: []string{
				fmt.Sprintf("Define the specific audience problem around %s.", subject),
				fmt.Sprintf("Show what changes when %s is handled well.", subject),
				"Convert the idea into a concrete next action for the audience.",
			},
			Cards: []presentationDraftCard{
				{Icon: role.icon, Title: "Audience need", Desc: fmt.Sprintf("The slide connects %s to a real decision the audience must make.", subject)},
				{Icon: "lightbulb", Title: "Key insight", Desc: "A focused idea keeps the slide useful instead of decorative."},
				{Icon: "check2-circle", Title: "Practical move", Desc: "End with an action the audience can apply after the presentation."},
			},
		}
		if role.kind == "steps" {
			brief.Steps = []string{
				fmt.Sprintf("Map the current situation around %s.", subject),
				"Choose the highest-impact intervention.",
				"Assign ownership, timeline, and success measure.",
			}
		}
		if role.kind == "stats" {
			brief.Stats = []presentationDraftStat{
				{Value: "01", Label: "Priority", Desc: "Primary decision the audience should make first."},
				{Value: "03", Label: "Signals", Desc: "Metrics or observations worth tracking."},
				{Value: "30d", Label: "Action window", Desc: "Near-term period to validate progress."},
			}
		}
		if role.kind == "compare" {
			brief.LeftColumn = &presentationDraftCompareColumn{Heading: "Helpful pattern", Points: []string{"Specific guidance", "Visible accountability", "Measured improvement"}}
			brief.RightColumn = &presentationDraftCompareColumn{Heading: "Risk pattern", Points: []string{"Vague ownership", "Unverified assumptions", "No follow-through"}}
		}
		if role.kind == "timeline" {
			brief.Timeline = []presentationDraftTimelineItem{
				{Date: "Now", Title: "Current position", Desc: fmt.Sprintf("Where %s stands today.", subject)},
				{Date: "Next", Title: "Emerging shift", Desc: "The next visible change the audience should watch."},
				{Date: "Then", Title: "Strategic role", Desc: "How the topic can influence wider decisions or outcomes."},
			}
		}
		if role.kind == "table" {
			brief.Table = &presentationDraftTableData{
				Headers: []string{"Choice", "Use when", "Watch for"},
				Rows: [][]string{
					{"Educate", "The audience needs shared understanding", "Too much theory"},
					{"Pilot", "The team needs proof before scale", "Weak measurement"},
					{"Scale", "The model is already validated", "Operational gaps"},
				},
			}
		}
		slides = append(slides, brief)
	}
	return slides
}

func normalizeSegmentedPPTXPlan(plan *segmentedPPTXPlan, task *domain.Task, contentSlides int) {
	plan.Filename = firstNonEmptyString(plan.Filename, slugifySegmentedFilename(task.Title))
	plan.Title = firstNonEmptyString(plan.Title, task.Title, "Presentation")
	plan.Deck.Subject = firstNonEmptyString(plan.Deck.Subject, plan.Title)
	plan.Deck.Audience = firstNonEmptyString(plan.Deck.Audience, "the intended audience")
	plan.Deck.Narrative = firstNonEmptyString(plan.Deck.Narrative, "context -> insight -> action")
	plan.Deck.Theme = firstNonEmptyString(plan.Deck.Theme, "presentation")
	plan.Deck.VisualStyle = firstNonEmptyString(plan.Deck.VisualStyle, "modern editorial briefing")
	plan.Deck.CoverStyle = firstNonEmptyString(plan.Deck.CoverStyle, "editorial")
	plan.Deck.ColorStory = firstNonEmptyString(plan.Deck.ColorStory, "clear high-contrast palette")
	plan.Deck.Motif = firstNonEmptyString(plan.Deck.Motif, "stars")
	plan.Deck.Kicker = firstNonEmptyString(plan.Deck.Kicker, "Briefing")
	if len(plan.Deck.Design) == 0 {
		plan.Deck.Design = map[string]string{
			"composition":    "asymmetric grid",
			"density":        "dense",
			"shape_language": "crisp",
			"accent_mode":    "badge and rail",
			"hero_layout":    "information-led",
		}
	}
	if len(plan.Deck.Palette) == 0 {
		plan.Deck.Palette = map[string]string{
			"background": "F6F8FB",
			"card":       "FFFFFF",
			"accent":     "0EA5E9",
			"accent2":    "14B8A6",
			"text":       "0F172A",
			"muted":      "475569",
			"border":     "CBD5E1",
		}
	}
	ensurePaletteKey(plan.Deck.Palette, "background", "F6F8FB")
	ensurePaletteKey(plan.Deck.Palette, "card", "FFFFFF")
	ensurePaletteKey(plan.Deck.Palette, "accent", "0EA5E9")
	ensurePaletteKey(plan.Deck.Palette, "accent2", "14B8A6")
	ensurePaletteKey(plan.Deck.Palette, "text", "0F172A")
	ensurePaletteKey(plan.Deck.Palette, "muted", "475569")
	ensurePaletteKey(plan.Deck.Palette, "border", "CBD5E1")
	if strings.Count(plan.Deck.ThemeCSS, "{") < 4 {
		plan.Deck.ThemeCSS = defaultSegmentedThemeCSS()
	}
	if !strings.Contains(plan.Deck.ThemeCSS, ".seg-slide") {
		plan.Deck.ThemeCSS = strings.TrimSpace(plan.Deck.ThemeCSS) + segmentedHTMLThemeCSS()
	}
	if !strings.Contains(plan.Deck.CoverHTML, "seg-cover") {
		plan.Deck.CoverHTML = defaultSegmentedCoverHTML(*plan)
	}
	if len(plan.Stages) == 0 {
		plan.Stages = []string{"open the topic", "explain the important ideas", "show practical application", "close with takeaways"}
	}
	if len(plan.Slides) > contentSlides {
		plan.Slides = plan.Slides[:contentSlides]
	}
	for len(plan.Slides) < contentSlides {
		next := len(plan.Slides) + 1
		role := segmentedSlideRole(next-1, contentSlides, plan.Stages)
		plan.Slides = append(plan.Slides, segmentedPPTXSlideBrief{
			Heading: fmt.Sprintf("Part %d: %s", next, role),
			Type:    "cards",
			Purpose: role,
			Icon:    "stars",
			Points: []string{
				"Make the idea concrete for the audience.",
				"Show a useful example or decision point.",
				"End with a practical takeaway.",
			},
			Cards: []presentationDraftCard{
				{Icon: "stars", Title: "Key idea", Desc: "A clear point that supports the deck narrative."},
				{Icon: "diagram-3", Title: "How it works", Desc: "A concise explanation of the mechanism or flow."},
				{Icon: "check2-circle", Title: "Takeaway", Desc: "A practical takeaway for the audience."},
			},
		})
	}
	for i := range plan.Slides {
		plan.Slides[i].Heading = firstNonEmptyString(plan.Slides[i].Heading, fmt.Sprintf("Slide %d", i+2))
		plan.Slides[i].Type = firstNonEmptyString(plan.Slides[i].Type, "cards")
		plan.Slides[i].Icon = firstNonEmptyString(plan.Slides[i].Icon, "stars")
	}
}

func completeSegmentedSlide(slide presentationDraftSlide, plan segmentedPPTXPlan, brief segmentedPPTXSlideBrief, index, totalSlides int) presentationDraftSlide {
	slide.Heading = firstNonEmptyString(slide.Heading, brief.Heading, "Slide")
	if slideHTMLLikelyReady(slide.HTML) {
		slide.Type = "html"
		fillMissingSegmentedStructuredContent(&slide, brief)
		return slide
	}

	slide.Type = firstNonEmptyString(slide.Type, brief.Type, "cards")
	fillMissingSegmentedStructuredContent(&slide, brief)
	if !segmentedSlideHasStructuredContent(slide) {
		slide.Type = "cards"
		slide.Cards = cardsFromSegmentedBrief(brief)
	}
	ensureSegmentedSlideRenderable(&slide, brief)
	slide.Type = "html"
	slide.Layout = "html"
	slide.HTML = segmentedSlideHTML(plan, slide, brief, index, totalSlides)
	return slide
}

func fillMissingSegmentedStructuredContent(slide *presentationDraftSlide, brief segmentedPPTXSlideBrief) {
	if len(slide.Points) == 0 {
		slide.Points = brief.Points
	}
	if len(slide.Cards) == 0 {
		slide.Cards = brief.Cards
	}
	if len(slide.Stats) == 0 {
		slide.Stats = brief.Stats
	}
	if len(slide.Steps) == 0 {
		slide.Steps = brief.Steps
	}
	if len(slide.Timeline) == 0 {
		slide.Timeline = brief.Timeline
	}
	if slide.LeftColumn == nil {
		slide.LeftColumn = brief.LeftColumn
	}
	if slide.RightColumn == nil {
		slide.RightColumn = brief.RightColumn
	}
	if slide.Table == nil {
		slide.Table = brief.Table
	}
}

func segmentedSlideRole(index, contentSlides int, stages []string) string {
	if contentSlides <= 1 {
		return firstNonEmptyString(firstStage(stages), "deliver the core takeaway")
	}
	if index == 0 {
		return firstNonEmptyString(firstStage(stages), "open the topic with the main idea")
	}
	if index == contentSlides-1 {
		return firstNonEmptyString(lastStage(stages), "close with practical takeaways")
	}
	if len(stages) == 0 {
		return "develop the middle of the narrative with concrete examples"
	}
	stageIndex := 1 + ((index - 1) * maxInt(1, len(stages)-2) / maxInt(1, contentSlides-2))
	if stageIndex >= len(stages) {
		stageIndex = len(stages) - 1
	}
	return stages[stageIndex]
}

func firstStage(stages []string) string {
	if len(stages) == 0 {
		return ""
	}
	return stages[0]
}

func lastStage(stages []string) string {
	if len(stages) == 0 {
		return ""
	}
	return stages[len(stages)-1]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func segmentedSlideHasStructuredContent(slide presentationDraftSlide) bool {
	return len(slide.Points) >= 2 ||
		len(slide.Stats) >= 2 ||
		len(slide.Steps) >= 2 ||
		len(slide.Cards) >= 2 ||
		len(slide.Timeline) >= 2 ||
		(slide.LeftColumn != nil && slide.RightColumn != nil && len(slide.LeftColumn.Points) > 0 && len(slide.RightColumn.Points) > 0) ||
		(slide.Table != nil && len(slide.Table.Headers) > 0 && len(slide.Table.Rows) > 0)
}

func ensureSegmentedSlideRenderable(slide *presentationDraftSlide, brief segmentedPPTXSlideBrief) {
	layout := strings.ToLower(strings.TrimSpace(firstNonEmptyString(slide.Type, slide.Layout)))
	switch layout {
	case "html":
		if !slideHTMLLikelyReady(slide.HTML) {
			promoteSegmentedSlideToCards(slide, brief)
		}
	case "stats":
		if len(slide.Stats) < 2 {
			promoteSegmentedSlideToCards(slide, brief)
		}
	case "steps":
		if len(slide.Steps) < 3 {
			if len(slide.Points) >= 3 {
				slide.Steps = slide.Points
			} else {
				promoteSegmentedSlideToCards(slide, brief)
			}
		}
	case "cards":
		if len(slide.Cards) < 3 {
			slide.Cards = cardsFromSegmentedBrief(brief)
		}
	case "timeline":
		if len(slide.Timeline) < 3 {
			promoteSegmentedSlideToCards(slide, brief)
		}
	case "compare":
		if slide.LeftColumn == nil || slide.RightColumn == nil {
			promoteSegmentedSlideToCards(slide, brief)
		}
	case "table":
		if slide.Table == nil || len(slide.Table.Headers) < 2 || len(slide.Table.Rows) == 0 {
			promoteSegmentedSlideToCards(slide, brief)
		}
	default:
		if len(slide.Points) < 2 {
			promoteSegmentedSlideToCards(slide, brief)
		}
	}
}

func segmentedSlideHTML(plan segmentedPPTXPlan, slide presentationDraftSlide, brief segmentedPPTXSlideBrief, index, totalSlides int) string {
	layout := strings.ToLower(strings.TrimSpace(firstNonEmptyString(slide.Type, slide.Layout, brief.Type, "cards")))
	eyebrow := escapePresentationDraftText(firstNonEmptyString(brief.Purpose, brief.Visual, plan.Deck.Kicker, "Briefing"))
	heading := escapePresentationDraftText(firstNonEmptyString(slide.Heading, brief.Heading, plan.Title))
	lead := escapePresentationDraftText(segmentedSlideLead(slide, brief, plan))
	icon := segmentedIconHTML(firstNonEmptyString(brief.Icon, plan.Deck.Motif, "stars"))

	var body string
	switch layout {
	case "stats":
		body = segmentedStatsHTML(slide)
	case "steps":
		body = segmentedStepsHTML(slide)
	case "timeline":
		body = segmentedTimelineHTML(slide)
	case "compare":
		body = segmentedCompareHTML(slide)
	case "table":
		body = segmentedTableHTML(slide)
	default:
		body = segmentedCardsHTML(slide, brief, index)
	}
	if strings.TrimSpace(body) == "" {
		body = segmentedCardsHTML(slide, brief, index)
	}

	switch segmentedCompositionVariant(layout, index) {
	case "band":
		return `<div class="seg-slide seg-slide-band"><div class="seg-band-copy"><div class="seg-eyebrow">` + eyebrow + `</div><h2 class="seg-title">` + heading + `</h2><p class="seg-lead">` + lead + `</p><div class="seg-signal"><span class="seg-icon">` + icon + `</span><span>` + escapePresentationDraftText(firstNonEmptyString(brief.Visual, "Purpose-built visual system")) + `</span></div></div><div class="seg-band-body">` + body + `</div></div>`
	case "stack":
		return `<div class="seg-slide seg-slide-stack"><div class="seg-top"><span class="badge">` + escapePresentationDraftText(plan.Deck.Kicker) + `</span><span class="seg-mark">` + escapePresentationDraftText(plan.Deck.Motif) + `</span></div><div class="seg-stack-head"><span class="seg-icon">` + icon + `</span><div><div class="seg-eyebrow">` + eyebrow + `</div><h2 class="seg-title">` + heading + `</h2><p class="seg-lead">` + lead + `</p></div></div><div class="seg-stack-body">` + body + `</div></div>`
	case "canvas":
		return `<div class="seg-slide seg-slide-canvas"><div class="seg-canvas-head"><div><div class="seg-eyebrow">` + eyebrow + `</div><h2 class="seg-title">` + heading + `</h2></div><span class="seg-icon">` + icon + `</span></div><p class="seg-lead">` + lead + `</p><div class="seg-canvas-body">` + body + `</div></div>`
	case "rail":
		return `<div class="seg-slide seg-slide-rail"><div class="seg-rail"><span class="seg-icon">` + icon + `</span><span class="seg-mark">` + escapePresentationDraftText(plan.Deck.Motif) + `</span></div><div class="seg-rail-content"><div class="seg-eyebrow">` + eyebrow + `</div><h2 class="seg-title">` + heading + `</h2><p class="seg-lead">` + lead + `</p><div class="seg-rail-body">` + body + `</div></div></div>`
	case "poster":
		return `<div class="seg-slide seg-slide-poster"><div class="seg-poster-title"><span class="badge">` + escapePresentationDraftText(plan.Deck.Kicker) + `</span><h2 class="seg-title">` + heading + `</h2><p class="seg-lead">` + lead + `</p></div><div class="seg-poster-body">` + body + `</div></div>`
	default:
		return `<div class="seg-slide seg-slide-split"><div class="seg-top"><span class="badge">` + escapePresentationDraftText(plan.Deck.Kicker) + `</span><span class="seg-mark">` + escapePresentationDraftText(plan.Deck.Motif) + `</span></div><div class="seg-split-grid"><div class="seg-hero"><span class="seg-icon">` + icon + `</span><div class="seg-eyebrow">` + eyebrow + `</div><h2 class="seg-title">` + heading + `</h2><p class="seg-lead">` + lead + `</p></div><div class="seg-main-panel">` + body + `</div></div></div>`
	}
}

func segmentedCompositionVariant(layout string, index int) string {
	switch layout {
	case "stats":
		if index%2 == 0 {
			return "canvas"
		}
		return "poster"
	case "steps", "timeline":
		if index%2 == 0 {
			return "rail"
		}
		return "band"
	case "compare", "table":
		if index%2 == 0 {
			return "stack"
		}
		return "canvas"
	}
	variants := []string{"split", "band", "stack", "canvas", "rail", "poster"}
	return variants[index%len(variants)]
}

func segmentedSlideLead(slide presentationDraftSlide, brief segmentedPPTXSlideBrief, plan segmentedPPTXPlan) string {
	if strings.TrimSpace(brief.Visual) != "" && strings.TrimSpace(brief.Purpose) != "" {
		return brief.Purpose + " through " + brief.Visual + "."
	}
	if len(slide.Points) > 0 {
		return slide.Points[0]
	}
	if strings.TrimSpace(brief.Purpose) != "" {
		return brief.Purpose
	}
	return firstNonEmptyString(plan.Deck.Narrative, plan.Subtitle, "A focused slide built from the requested subject.")
}

func segmentedCardsHTML(slide presentationDraftSlide, brief segmentedPPTXSlideBrief, index int) string {
	cards := slide.Cards
	if len(cards) < 3 {
		cards = cardsFromSegmentedBrief(brief)
	}
	var out strings.Builder
	if index%2 == 0 {
		out.WriteString(`<div class="seg-card-mosaic">`)
	} else {
		out.WriteString(`<div class="seg-card-ladder">`)
	}
	for i, card := range cards {
		if i >= 4 {
			break
		}
		out.WriteString(`<div class="seg-card"><div class="seg-card-icon">`)
		out.WriteString(segmentedIconHTML(firstNonEmptyString(card.Icon, brief.Icon, "stars")))
		out.WriteString(`</div><h3>`)
		out.WriteString(escapePresentationDraftText(firstNonEmptyString(card.Title, fmt.Sprintf("Point %d", i+1))))
		out.WriteString(`</h3><p>`)
		out.WriteString(escapePresentationDraftText(firstNonEmptyString(card.Desc, segmentedPointAt(slide.Points, i))))
		out.WriteString(`</p></div>`)
	}
	out.WriteString(`</div>`)
	return out.String()
}

func segmentedStatsHTML(slide presentationDraftSlide) string {
	var out strings.Builder
	out.WriteString(`<div class="seg-stat-grid">`)
	for i, stat := range slide.Stats {
		if i >= 4 {
			break
		}
		out.WriteString(`<div class="seg-stat"><strong>`)
		out.WriteString(escapePresentationDraftText(stat.Value))
		out.WriteString(`</strong><span>`)
		out.WriteString(escapePresentationDraftText(stat.Label))
		out.WriteString(`</span><p>`)
		out.WriteString(escapePresentationDraftText(stat.Desc))
		out.WriteString(`</p></div>`)
	}
	out.WriteString(`</div>`)
	return out.String()
}

func segmentedStepsHTML(slide presentationDraftSlide) string {
	items := slide.Steps
	if len(items) == 0 {
		items = slide.Points
	}
	var out strings.Builder
	out.WriteString(`<div class="seg-step-list">`)
	for i, item := range items {
		if i >= 5 {
			break
		}
		out.WriteString(`<div class="seg-step"><span>`)
		out.WriteString(fmt.Sprintf("%02d", i+1))
		out.WriteString(`</span><p>`)
		out.WriteString(escapePresentationDraftText(item))
		out.WriteString(`</p></div>`)
	}
	out.WriteString(`</div>`)
	return out.String()
}

func segmentedTimelineHTML(slide presentationDraftSlide) string {
	var out strings.Builder
	out.WriteString(`<div class="seg-timeline">`)
	for i, item := range slide.Timeline {
		if i >= 5 {
			break
		}
		out.WriteString(`<div class="seg-time"><span>`)
		out.WriteString(escapePresentationDraftText(firstNonEmptyString(item.Date, fmt.Sprintf("T%d", i+1))))
		out.WriteString(`</span><h3>`)
		out.WriteString(escapePresentationDraftText(item.Title))
		out.WriteString(`</h3><p>`)
		out.WriteString(escapePresentationDraftText(item.Desc))
		out.WriteString(`</p></div>`)
	}
	out.WriteString(`</div>`)
	return out.String()
}

func segmentedCompareHTML(slide presentationDraftSlide) string {
	if slide.LeftColumn == nil || slide.RightColumn == nil {
		return ""
	}
	return `<div class="seg-compare"><div class="seg-compare-col"><h3>` + escapePresentationDraftText(slide.LeftColumn.Heading) + `</h3>` + segmentedPointListHTML(slide.LeftColumn.Points) + `</div><div class="seg-compare-col seg-compare-alt"><h3>` + escapePresentationDraftText(slide.RightColumn.Heading) + `</h3>` + segmentedPointListHTML(slide.RightColumn.Points) + `</div></div>`
}

func segmentedTableHTML(slide presentationDraftSlide) string {
	if slide.Table == nil || len(slide.Table.Headers) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(`<table class="seg-table"><thead><tr>`)
	for _, header := range slide.Table.Headers {
		out.WriteString(`<th>`)
		out.WriteString(escapePresentationDraftText(header))
		out.WriteString(`</th>`)
	}
	out.WriteString(`</tr></thead><tbody>`)
	for i, row := range slide.Table.Rows {
		if i >= 4 {
			break
		}
		out.WriteString(`<tr>`)
		for _, cell := range row {
			out.WriteString(`<td>`)
			out.WriteString(escapePresentationDraftText(cell))
			out.WriteString(`</td>`)
		}
		out.WriteString(`</tr>`)
	}
	out.WriteString(`</tbody></table>`)
	return out.String()
}

func segmentedPointListHTML(points []string) string {
	var out strings.Builder
	out.WriteString(`<ul class="seg-points">`)
	for i, point := range points {
		if i >= 5 {
			break
		}
		out.WriteString(`<li>`)
		out.WriteString(escapePresentationDraftText(point))
		out.WriteString(`</li>`)
	}
	out.WriteString(`</ul>`)
	return out.String()
}

func segmentedPointAt(points []string, index int) string {
	if index >= 0 && index < len(points) {
		return points[index]
	}
	return "A concrete supporting point for this slide."
}

func segmentedIconHTML(value string) string {
	icon := segmentedBootstrapIcon(value)
	return `<i class="bi bi-` + icon + `" aria-hidden="true"></i>`
}

func segmentedBootstrapIcon(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "automation", "ai", "chip", "technology":
		return "cpu"
	case "spark", "star", "stars", "magic":
		return "stars"
	case "chart", "data", "metric", "metrics":
		return "bar-chart-line"
	case "people", "audience", "users":
		return "people"
	case "learning", "idea", "lightbulb":
		return "lightbulb"
	case "risk", "guardrail", "safety":
		return "shield-check"
	case "action", "takeaway", "check":
		return "check2-circle"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "stars"
	}
	return cleaned
}

func promoteSegmentedSlideToCards(slide *presentationDraftSlide, brief segmentedPPTXSlideBrief) {
	slide.Type = "cards"
	slide.Layout = "cards"
	if len(slide.Cards) < 3 {
		slide.Cards = cardsFromSegmentedBrief(brief)
	}
}

func cardsFromSegmentedBrief(brief segmentedPPTXSlideBrief) []presentationDraftCard {
	points := brief.Points
	if len(points) == 0 && strings.TrimSpace(brief.Purpose) != "" {
		points = append(points, brief.Purpose)
	}
	if len(points) == 0 && strings.TrimSpace(brief.Visual) != "" {
		points = append(points, brief.Visual)
	}
	for len(points) < 3 {
		points = append(points, "A concrete supporting point for "+firstNonEmptyString(brief.Heading, "this slide")+".")
	}
	cards := make([]presentationDraftCard, 0, 3)
	for i := 0; i < 3; i++ {
		cards = append(cards, presentationDraftCard{
			Icon:  firstNonEmptyString(brief.Icon, "stars"),
			Title: fmt.Sprintf("Point %d", i+1),
			Desc:  points[i],
		})
	}
	return cards
}

func slideHTMLLikelyReady(raw string) bool {
	raw = strings.TrimSpace(stripPresentationDraftEmoji(raw))
	if !strings.Contains(raw, "<") || !strings.Contains(raw, ">") {
		return false
	}
	text := segmentedHTMLTagPattern.ReplaceAllString(raw, " ")
	text = strings.Join(strings.Fields(text), " ")
	return len(text) >= 56
}

func (a *AgentLoop) emitSegmentedPresentationDraft(ctx context.Context, taskID string, index, total int, kind, heading, htmlBody string, deck segmentedPPTXDeck) {
	a.emitAgentEvent(ctx, taskID, domain.EventTypePresentationSlide, map[string]any{
		"index":     index,
		"total":     total,
		"kind":      kind,
		"heading":   firstNonEmptyString(heading, fmt.Sprintf("Slide %d", index)),
		"html":      stripPresentationDraftEmoji(htmlBody),
		"theme_css": deck.ThemeCSS,
		"palette":   deck.Palette,
	})
}

func compactSegmentedExtraContext(extraSystemPrompts []string) string {
	var parts []string
	for _, prompt := range extraSystemPrompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" || strings.HasPrefix(prompt, "Active skill prompt:") {
			continue
		}
		if len(prompt) > 2000 {
			prompt = prompt[:2000]
		}
		parts = append(parts, prompt)
	}
	return strings.Join(parts, "\n\n")
}

func ensurePaletteKey(palette map[string]string, key, fallback string) {
	value := strings.TrimPrefix(strings.TrimSpace(palette[key]), "#")
	if !segmentedHexColorPattern.MatchString(value) {
		palette[key] = fallback
		return
	}
	palette[key] = value
}

func defaultSegmentedThemeCSS() string {
	return `.row{display:flex;gap:18px}.card{border-radius:22px;border:1px solid var(--border);background:var(--card);box-shadow:0 18px 44px rgba(15,23,42,.10)}.card-body{padding:24px;display:grid;gap:12px}.badge{border-radius:999px;padding:8px 12px;background:color-mix(in srgb,var(--accent) 16%,transparent);border:1px solid color-mix(in srgb,var(--accent) 34%,transparent)}.icon-badge{width:58px;height:58px;border-radius:16px;background:color-mix(in srgb,var(--accent) 14%,white);color:var(--accent)}.list-group-item{border-left:5px solid var(--accent);border-radius:16px;padding:16px 18px;background:var(--card)}` + segmentedHTMLThemeCSS()
}

func segmentedHTMLThemeCSS() string {
	return `.seg-cover,.seg-slide{height:100%;padding:46px 54px;background:radial-gradient(circle at 88% 10%,color-mix(in srgb,var(--accent) 16%,transparent),transparent 30%),var(--bg);color:var(--text)}.seg-cover{display:grid;grid-template-columns:1.05fr .95fr;gap:34px;align-items:stretch}.seg-cover-left{display:flex;flex-direction:column;justify-content:center;gap:22px}.seg-cover-title{font-size:68px;line-height:.96;font-weight:900;letter-spacing:-.06em;margin:0}.seg-cover-lead{font-size:25px;line-height:1.32;color:var(--muted);margin:0;max-width:760px}.seg-cover-chips{display:flex;flex-wrap:wrap}.seg-cover-chips span{padding:12px 14px;border-radius:999px;background:var(--card);border:1px solid var(--border);font-size:17px;margin:0 12px 12px 0}.seg-cover-right{display:grid;grid-template-rows:1fr .78fr;gap:18px}.seg-cover-panel{border:1px solid var(--border);border-radius:28px;background:color-mix(in srgb,var(--card) 92%,transparent);box-shadow:0 20px 54px rgba(15,23,42,.12);padding:24px;display:flex;flex-direction:column;justify-content:space-between}.seg-cover-panel-main h2{font-size:32px;line-height:1.08;margin:18px 0 10px}.seg-cover-panel p{font-size:19px;line-height:1.34;color:var(--muted);margin:0}.seg-cover-row{display:grid;grid-template-columns:1fr 1fr;gap:18px}.seg-cover-panel strong{font-size:42px;color:var(--accent);line-height:1}.seg-cover-panel span{font-size:13px;font-weight:850;text-transform:uppercase;letter-spacing:.11em;color:var(--muted)}.seg-slide{display:flex;flex-direction:column;gap:22px}.seg-top{display:flex;justify-content:space-between;align-items:center}.seg-mark,.seg-eyebrow{font-size:13px;font-weight:850;letter-spacing:.11em;text-transform:uppercase;color:var(--muted)}.seg-title{font-size:45px;line-height:1.02;font-weight:880;letter-spacing:-.045em;margin:0}.seg-lead{font-size:22px;line-height:1.34;color:var(--muted);margin:0}.seg-icon,.seg-card-icon{display:inline-flex;align-items:center;justify-content:center;width:60px;height:60px;border-radius:18px;background:color-mix(in srgb,var(--accent) 15%,white);color:var(--accent);border:1px solid color-mix(in srgb,var(--accent) 32%,white)}.seg-slide-split .seg-split-grid{display:grid;grid-template-columns:.92fr 1.35fr;gap:28px;min-height:0;flex:1}.seg-hero,.seg-main-panel,.seg-band-copy,.seg-band-body{border:1px solid var(--border);border-radius:28px;background:color-mix(in srgb,var(--card) 92%,transparent);box-shadow:0 20px 54px rgba(15,23,42,.12)}.seg-hero{padding:30px;display:flex;flex-direction:column;justify-content:space-between}.seg-main-panel,.seg-band-body{padding:24px}.seg-slide-band{display:grid;grid-template-columns:.78fr 1.35fr;gap:28px}.seg-band-copy{padding:32px;display:flex;flex-direction:column;gap:18px;justify-content:center}.seg-band-body{display:flex;align-items:stretch}.seg-slide-stack{padding-top:42px}.seg-stack-head{display:grid;grid-template-columns:74px 1fr;gap:20px;align-items:center}.seg-stack-body{flex:1;min-height:0}.seg-slide-canvas{padding-top:42px}.seg-canvas-head{display:flex;justify-content:space-between;align-items:flex-start}.seg-canvas-body{flex:1;min-height:0;border:1px solid var(--border);border-radius:28px;background:color-mix(in srgb,var(--card) 92%,transparent);box-shadow:0 20px 54px rgba(15,23,42,.12);padding:24px}.seg-slide-rail{display:grid;grid-template-columns:112px 1fr;gap:28px}.seg-rail{border-radius:30px;background:linear-gradient(180deg,var(--accent),var(--accent2));color:white;display:flex;flex-direction:column;align-items:center;justify-content:space-between;padding:24px 14px}.seg-rail .seg-icon{background:rgba(255,255,255,.18);border-color:rgba(255,255,255,.28);color:white}.seg-rail .seg-mark{writing-mode:vertical-rl;color:white;opacity:.88}.seg-rail-content{display:flex;flex-direction:column;gap:18px;min-width:0}.seg-rail-body{flex:1;border:1px solid var(--border);border-radius:28px;background:color-mix(in srgb,var(--card) 92%,transparent);padding:24px}.seg-slide-poster{display:grid;grid-template-columns:.9fr 1.1fr;gap:28px}.seg-poster-title{border-radius:32px;background:linear-gradient(135deg,color-mix(in srgb,var(--accent) 92%,black),color-mix(in srgb,var(--accent2) 86%,black));color:white;padding:34px;display:flex;flex-direction:column;justify-content:space-between}.seg-poster-title .seg-lead,.seg-poster-title .badge{color:white}.seg-poster-body{border:1px solid var(--border);border-radius:28px;background:var(--card);padding:24px;display:flex;align-items:stretch}.seg-card-mosaic{display:grid;grid-template-columns:1fr 1fr;gap:16px;width:100%}.seg-card-ladder{display:grid;grid-template-columns:1fr;gap:14px;width:100%}.seg-card{padding:18px;border-radius:22px;background:var(--card);border:1px solid var(--border);display:grid;gap:10px}.seg-card h3,.seg-compare h3,.seg-time h3{font-size:23px;line-height:1.12;margin:0;font-weight:820}.seg-card p,.seg-step p,.seg-time p,.seg-stat p,.seg-points li,.seg-table td{font-size:19px;line-height:1.34;color:var(--muted);margin:0}.seg-stat-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;width:100%}.seg-stat{padding:22px;border-radius:24px;background:var(--card);border:1px solid var(--border)}.seg-stat strong{display:block;font-size:42px;line-height:1;color:var(--accent);letter-spacing:-.04em}.seg-stat span{display:block;font-size:14px;font-weight:850;text-transform:uppercase;letter-spacing:.1em;margin:12px 0 8px}.seg-step-list{display:grid;gap:14px;width:100%}.seg-step{display:grid;grid-template-columns:58px 1fr;gap:14px;align-items:center;padding:16px;border-radius:20px;background:var(--card);border:1px solid var(--border)}.seg-step span{font-size:22px;font-weight:850;color:var(--accent)}.seg-compare{display:grid;grid-template-columns:1fr 1fr;gap:18px;width:100%}.seg-compare-col{padding:22px;border-radius:24px;background:var(--card);border:1px solid var(--border)}.seg-compare-alt{background:color-mix(in srgb,var(--accent2) 9%,var(--card))}.seg-points{display:grid;gap:10px;margin:14px 0 0;padding:0;list-style:none}.seg-points li{padding:10px 12px;border-left:4px solid var(--accent);background:color-mix(in srgb,var(--accent) 7%,transparent);border-radius:12px}.seg-table{width:100%;border-collapse:separate;border-spacing:0 10px}.seg-table th{font-size:13px;text-align:left;text-transform:uppercase;letter-spacing:.1em;color:var(--muted)}.seg-table td{padding:14px 16px;background:var(--card);border-top:1px solid var(--border);border-bottom:1px solid var(--border)}.seg-timeline{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}.seg-time{padding:18px;border-radius:22px;background:var(--card);border:1px solid var(--border)}.seg-time span{font-size:13px;font-weight:850;color:var(--accent);text-transform:uppercase}`
}

func defaultSegmentedCoverHTML(plan segmentedPPTXPlan) string {
	title := escapePresentationDraftText(plan.Title)
	kicker := escapePresentationDraftText(plan.Deck.Kicker)
	narrative := escapePresentationDraftText(plan.Deck.Narrative)
	audience := escapePresentationDraftText(plan.Deck.Audience)
	theme := escapePresentationDraftText(plan.Deck.Theme)
	style := escapePresentationDraftText(plan.Deck.VisualStyle)
	firstHeading, firstDesc := segmentedCoverAgenda(plan, 0, "Context", firstNonEmptyString(firstStage(plan.Stages), "Frame the subject"))
	secondHeading, secondDesc := segmentedCoverAgenda(plan, 1, "Signal", firstNonEmptyString(lastStage(plan.Stages), "Close with action"))
	return `<div class="seg-cover"><div class="seg-cover-left"><span class="badge">` + kicker + `</span><h1 class="seg-cover-title">` + title + `</h1><p class="seg-cover-lead">` + narrative + `</p><div class="seg-cover-chips"><span>Audience: ` + audience + `</span><span>Theme: ` + theme + `</span></div></div><div class="seg-cover-right"><div class="seg-cover-panel seg-cover-panel-main"><div class="seg-card-icon">` + segmentedIconHTML(plan.Deck.Motif) + `</div><h2>` + style + `</h2><p>` + escapePresentationDraftText(plan.Deck.CoverStyle) + `</p></div><div class="seg-cover-row"><div class="seg-cover-panel"><strong>01</strong><span>` + firstHeading + `</span><p>` + firstDesc + `</p></div><div class="seg-cover-panel"><strong>02</strong><span>` + secondHeading + `</span><p>` + secondDesc + `</p></div></div></div></div>`
}

func segmentedCoverAgenda(plan segmentedPPTXPlan, index int, fallbackHeading, fallbackDesc string) (string, string) {
	if index >= 0 && index < len(plan.Slides) {
		brief := plan.Slides[index]
		heading := escapePresentationDraftText(firstNonEmptyString(brief.Heading, fallbackHeading))
		desc := escapePresentationDraftText(firstNonEmptyString(brief.Purpose, segmentedPointAt(brief.Points, 0), fallbackDesc))
		return heading, desc
	}
	return escapePresentationDraftText(fallbackHeading), escapePresentationDraftText(fallbackDesc)
}

func slugifySegmentedFilename(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastHyphen := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "presentation"
	}
	return out
}

func marshalJSON(value any) string {
	b, _ := json.Marshal(value)
	return string(b)
}
