package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AskUserTool pauses the agent loop and asks the user a question.
// The agent blocks until the user responds via the frontend input widget
// or until a 5-minute timeout.
type AskUserTool struct {
	Store   *UserInputStore
	Emitter func(taskID, pendingID, question string) // called to emit input.needed event
}

func (AskUserTool) Name() string { return "ask_user" }
func (AskUserTool) Description() string {
	return "Pause execution and ask the user a question or request feedback. " +
		"Use this when you need clarification, a preference, approval for a major decision, " +
		"or want to show a draft and get user feedback before continuing. " +
		"The agent will wait up to 5 minutes for a response."
}

func (AskUserTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question or request to show the user. Be specific and concise.",
			},
		},
		"required": []string{"question"},
	}
}

func (t AskUserTool) Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result {
	var args struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Question == "" {
		return Err("ask_user: question is required")
	}

	pendingID := uuid.NewString()
	ch := t.Store.Register(pendingID, ictx.TaskID, args.Question)

	// Notify frontend via event emitter
	if t.Emitter != nil {
		t.Emitter(ictx.TaskID, pendingID, args.Question)
	}

	// Block until user replies or timeout (5 minutes)
	answer, ok := t.Store.Wait(ctx, ch, 5*time.Minute)
	if !ok {
		// Timed out — clean up and let the agent continue without input
		t.Store.Answer(pendingID, "") // drain
		return OKData("User did not respond in time. Continue without user input.", map[string]any{
			"answered": false,
			"answer":   "",
		})
	}

	return OKData("User responded: "+answer, map[string]any{
		"answered": true,
		"answer":   answer,
	})
}
