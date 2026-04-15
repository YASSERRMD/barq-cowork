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

const segmentedPPTXCallTimeout = 90 * time.Second

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

	plan, err := a.generateSegmentedPPTXPlan(ctx, task, contentSlides, totalSlides, extraSystemPrompts, runtimeProfile)
	completed := time.Now().UTC()
	planStep.CompletedAt = &completed
	if err != nil {
		planStep.Status = domain.StepStatusFailed
		planStep.ToolOutput = marshalError(err)
		_ = a.plans.UpdateStep(ctx, planStep)
		a.emitAgentEvent(ctx, task.ID, domain.EventTypeAgentMessage, map[string]any{
			"text": "Presentation planning failed: " + err.Error(),
		})
		result.Failed++
		return result
	}

	normalizeSegmentedPPTXPlan(&plan, task, contentSlides)
	planStep.Status = domain.StepStatusCompleted
	planStep.ToolOutput = marshalJSON(map[string]any{"status": "ok", "slides": totalSlides, "title": plan.Title})
	_ = a.plans.UpdateStep(ctx, planStep)
	a.emitAgentEvent(ctx, task.ID, domain.EventTypeStepCompleted, map[string]any{
		"step_id": planStep.ID, "tool": "presentation_plan", "status": planStep.Status,
	})
	a.emitSegmentedPresentationDraft(ctx, task.ID, 1, totalSlides, "cover", plan.Title, plan.Deck.CoverHTML, plan.Deck)

	finalSlides := make([]presentationDraftSlide, 0, contentSlides)
	stepOrder := 1
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

		slide, slideErr := a.generateSegmentedPPTXSlide(ctx, task, plan, brief, i, totalSlides, runtimeProfile)
		slide = completeSegmentedSlide(slide, brief)
		if slideErr != nil {
			a.logger.Warn("segmented pptx: slide LLM failed, using planned brief", "task_id", task.ID, "slide", i+1, "error", slideErr)
		}
		finalSlides = append(finalSlides, slide)

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
		a.emitSegmentedPresentationDraft(ctx, task.ID, i+2, totalSlides, "slide", slide.Heading, presentationDraftSlideHTML(slide), plan.Deck)
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
  "stages":["opening stage","explanation stage","practice/safety stage","closing stage"]
}

Rules:
- Do not include per-slide content in this response. That will be generated one slide at a time.
- stages length must be 3 to 5.
- Keep the whole JSON compact. Do not pretty print. Do not add fields beyond the shape above.
- Keep deck.theme_css under 700 characters and deck.cover_html under 900 characters.
- No emoji anywhere.
- Use real Bootstrap Icon placeholders in HTML only: <i class="bi bi-stars" aria-hidden="true"></i>.
- Do not include visible slide counters.
- Keep copy readable for projector use: no tiny text, no empty decorative panels.
- Make the design decisions from the subject, not from a fixed template.`, task.Title, task.Description, contextText, totalSlides, contentSlides)

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 3000, []provider.ChatMessage{
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

	err := a.chatSegmentedJSON(ctx, runtimeProfile, 1800, []provider.ChatMessage{
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
	messages []provider.ChatMessage,
	out any,
) error {
	callCtx, cancel := context.WithTimeout(ctx, segmentedPPTXCallTimeout)
	defer cancel()
	retry := runtimeProfile.Retry
	retry.MaxAttempts = 1
	req := provider.ChatCompletionRequest{
		Model:          a.cfg.Model,
		Stream:         false,
		MaxTokens:      maxTokens,
		Temperature:    runtimeProfile.Temperature,
		Messages:       messages,
		ResponseFormat: map[string]any{"type": "json_object"},
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
	if strings.TrimSpace(plan.Deck.CoverHTML) == "" {
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

func completeSegmentedSlide(slide presentationDraftSlide, brief segmentedPPTXSlideBrief) presentationDraftSlide {
	slide.Heading = firstNonEmptyString(slide.Heading, brief.Heading, "Slide")
	if slideHTMLLikelyReady(slide.HTML) {
		slide.Type = "html"
		fillMissingSegmentedStructuredContent(&slide, brief)
		return slide
	}

	slide.HTML = ""
	slide.Type = firstNonEmptyString(slide.Type, brief.Type, "cards")
	fillMissingSegmentedStructuredContent(&slide, brief)
	if !segmentedSlideHasStructuredContent(slide) {
		slide.Type = "cards"
		slide.Cards = cardsFromSegmentedBrief(brief)
	}
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
	return `.row{display:flex;gap:18px}.card{border-radius:22px;border:1px solid var(--border);background:var(--card);box-shadow:0 18px 44px rgba(15,23,42,.10)}.card-body{padding:24px;display:grid;gap:12px}.badge{border-radius:999px;padding:8px 12px;background:color-mix(in srgb,var(--accent) 16%,transparent);border:1px solid color-mix(in srgb,var(--accent) 34%,transparent)}.icon-badge{width:58px;height:58px;border-radius:16px;background:color-mix(in srgb,var(--accent) 14%,white);color:var(--accent)}.list-group-item{border-left:5px solid var(--accent);border-radius:16px;padding:16px 18px;background:var(--card)}`
}

func defaultSegmentedCoverHTML(plan segmentedPPTXPlan) string {
	return `<div class="container-fluid h-100 d-grid gap-4" style="padding:68px 72px;align-content:center"><div class="badge">` +
		stripPresentationDraftEmoji(plan.Deck.Kicker) +
		`</div><h1 class="display-title">` + stripPresentationDraftEmoji(plan.Title) +
		`</h1><p class="lead">` + stripPresentationDraftEmoji(plan.Deck.Narrative) +
		`</p><div class="row"><div class="col-4"><div class="card"><div class="card-body">Audience: ` +
		stripPresentationDraftEmoji(plan.Deck.Audience) +
		`</div></div></div><div class="col-4"><div class="card"><div class="card-body">Theme: ` +
		stripPresentationDraftEmoji(plan.Deck.Theme) +
		`</div></div></div><div class="col-4"><div class="card"><div class="card-body">Motif: <i class="bi bi-stars" aria-hidden="true"></i></div></div></div></div></div>`
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
