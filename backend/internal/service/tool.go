package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/tools"
	"github.com/google/uuid"
)

// ApprovalRepository is the minimal interface ToolService needs for approvals.
type ApprovalRepository interface {
	Create(ctx context.Context, a *domain.ApprovalRequest) error
	GetByID(ctx context.Context, id string) (*domain.ApprovalRequest, error)
	ListPending(ctx context.Context) ([]*domain.ApprovalRequest, error)
	Resolve(ctx context.Context, id, resolution string) error
}

// EventRepository is the minimal interface ToolService needs for events.
type EventRepository interface {
	Create(ctx context.Context, e *domain.Event) error
	ListByTask(ctx context.Context, taskID string) ([]*domain.Event, error)
}

// ToolService wraps the tool registry with approval and event emission logic.
type ToolService struct {
	registry  *tools.Registry
	approvals ApprovalRepository
	events    EventRepository
}

// NewToolService creates a ToolService.
func NewToolService(
	registry *tools.Registry,
	approvals ApprovalRepository,
	events EventRepository,
) *ToolService {
	return &ToolService{
		registry:  registry,
		approvals: approvals,
		events:    events,
	}
}

// ListTools returns metadata for all registered tools.
func (s *ToolService) ListTools() []ToolInfo {
	names := s.registry.List()
	out := make([]ToolInfo, 0, len(names))
	for _, name := range names {
		t, _ := s.registry.Get(name)
		out = append(out, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		})
	}
	return out
}

// ToolInfo is returned by ListTools.
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Invoke runs a registered tool, emits events, and gates destructive actions
// through the approval store.
func (s *ToolService) Invoke(
	ctx context.Context,
	taskID, workspaceRoot, toolName, argsJSON string,
	requireApproval bool,
) (tools.Result, error) {
	t, ok := s.registry.Get(toolName)
	if !ok {
		return tools.Err("tool not found: %s", toolName), nil
	}

	// Emit tool.called event
	s.emitEvent(ctx, taskID, domain.EventTypeToolCalled, map[string]any{
		"tool": toolName, "args": argsJSON,
	})

	ictx := tools.InvocationContext{
		WorkspaceRoot: workspaceRoot,
		TaskID:        taskID,
	}

	if requireApproval {
		ictx.RequireApproval = func(ctx context.Context, action, payload string) bool {
			ar := &domain.ApprovalRequest{
				ID:        uuid.NewString(),
				TaskID:    taskID,
				ToolName:  toolName,
				Action:    action,
				Payload:   payload,
				Status:    domain.ApprovalStatusPending,
				CreatedAt: time.Now().UTC(),
			}
			_ = s.approvals.Create(ctx, ar)
			// Synchronous mode: immediately check if there's an approved resolution.
			// In Phase 5 the executor will wait asynchronously for resolution.
			// For now, block the action until approved (returns false = deny immediately
			// so the UI can see it and the user resolves it in a follow-up request).
			return false
		}
	}

	result := t.Execute(ctx, ictx, argsJSON)

	// Emit tool.result event
	s.emitEvent(ctx, taskID, domain.EventTypeToolResult, map[string]any{
		"tool":   toolName,
		"status": result.Status,
	})

	return result, nil
}

// ListPendingApprovals returns all unresolved approval requests.
func (s *ToolService) ListPendingApprovals(ctx context.Context) ([]*domain.ApprovalRequest, error) {
	return s.approvals.ListPending(ctx)
}

// ResolveApproval approves or rejects an approval request.
func (s *ToolService) ResolveApproval(ctx context.Context, id, resolution string) error {
	if resolution != "approved" && resolution != "rejected" {
		return &domain.ValidationError{Field: "resolution", Message: "must be 'approved' or 'rejected'"}
	}
	return s.approvals.Resolve(ctx, id, resolution)
}

// ListEvents returns all events for a task.
func (s *ToolService) ListEvents(ctx context.Context, taskID string) ([]*domain.Event, error) {
	return s.events.ListByTask(ctx, taskID)
}

func (s *ToolService) emitEvent(ctx context.Context, taskID string, eventType domain.EventType, data map[string]any) {
	payload, _ := json.Marshal(data)
	e := &domain.Event{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Type:      eventType,
		Payload:   string(payload),
		CreatedAt: time.Now().UTC(),
	}
	_ = s.events.Create(ctx, e)
}

// BuildRegistry creates and registers all built-in tools.
// userInput and emitFn wire up the interactive ask_user tool;
// pass nil for both to skip registering ask_user.
func BuildRegistry(userInput *tools.UserInputStore, emitFn func(taskID, pendingID, question string)) *tools.Registry {
	r := tools.NewRegistry()
	r.Register(tools.ListFilesTool{})
	r.Register(tools.ReadFileTool{})
	r.Register(tools.WriteFileTool{})
	r.Register(tools.MoveFileTool{})
	r.Register(tools.CreateFolderTool{})
	r.Register(tools.SearchFilesTool{})
	r.Register(tools.NewHTTPFetchTool())
	r.Register(tools.MarkdownReportTool{})
	r.Register(tools.JSONExportTool{})
	r.Register(tools.WriteSlidesTool{})
	r.Register(tools.WritePPTXTool{})
	r.Register(tools.DocxTool{})
	r.Register(tools.WriteHTMLDocxTool{})
	r.Register(tools.WriteHTMLPDFTool{})
	if userInput != nil {
		r.Register(tools.AskUserTool{Store: userInput, Emitter: emitFn})
	}
	return r
}

// buildInvocationContextError returns an error message for missing workspace.
func buildInvocationContextError(workspaceRoot string) error {
	if workspaceRoot == "" {
		return fmt.Errorf("workspace root is required for file operations")
	}
	return nil
}

var _ = buildInvocationContextError // suppress unused warning
