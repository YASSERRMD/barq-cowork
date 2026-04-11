// Package tools defines the tool interface and registry used by the agent runtime.
// Each tool is a named, self-describing function that an LLM can call by name.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/provider"
)

// ─────────────────────────────────────────────
// Invocation context
// ─────────────────────────────────────────────

// InvocationContext carries per-call state that tools need for safety checks.
type InvocationContext struct {
	// WorkspaceRoot is the absolute path all file operations are scoped to.
	// Tools must reject any operation outside this directory.
	WorkspaceRoot string

	// TaskID is used for event emission and approval request correlation.
	TaskID string

	// RequireApproval is called by destructive tools before acting.
	// If it returns false the operation is aborted.
	RequireApproval func(ctx context.Context, action, payload string) bool
}

// ─────────────────────────────────────────────
// Result
// ─────────────────────────────────────────────

// ResultStatus classifies the outcome of a tool call.
type ResultStatus string

const (
	ResultOK      ResultStatus = "ok"
	ResultError   ResultStatus = "error"
	ResultDenied  ResultStatus = "denied"   // approval was rejected
	ResultPending ResultStatus = "pending"  // approval not yet resolved
)

// Result is returned by every tool call.
type Result struct {
	Status  ResultStatus `json:"status"`
	Content string       `json:"content"`          // human-readable text output
	Data    any          `json:"data,omitempty"`   // structured output (optional)
	Error   string       `json:"error,omitempty"`  // set when Status != ok
}

// OK returns a successful Result with a text content string.
func OK(content string) Result {
	return Result{Status: ResultOK, Content: content}
}

// OKData returns a successful Result with structured data.
func OKData(content string, data any) Result {
	return Result{Status: ResultOK, Content: content, Data: data}
}

// Err returns an error Result.
func Err(format string, args ...any) Result {
	msg := fmt.Sprintf(format, args...)
	return Result{Status: ResultError, Content: msg, Error: msg}
}

// Denied returns a denial Result.
func Denied(action string) Result {
	msg := fmt.Sprintf("action denied by user: %s", action)
	return Result{Status: ResultDenied, Content: msg, Error: msg}
}

// ToJSON serialises the result for LLM tool-result messages.
func (r Result) ToJSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// ─────────────────────────────────────────────
// Tool interface
// ─────────────────────────────────────────────

// Tool is the extension point for agent-callable actions.
// All implementations must be safe for concurrent use.
type Tool interface {
	// Name returns the unique tool identifier used in tool calls.
	Name() string

	// Description returns a plain-English description for the LLM.
	Description() string

	// InputSchema returns a JSON-schema object describing the tool's arguments.
	// This is passed directly to the provider in ToolDefinition.InputSchema.
	InputSchema() map[string]any

	// Execute runs the tool with the provided JSON-encoded arguments.
	Execute(ctx context.Context, ictx InvocationContext, argsJSON string) Result
}

// ToProviderDefinition converts a Tool to the provider.ToolDefinition format.
func ToProviderDefinition(t Tool) provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: t.InputSchema(),
	}
}

// ─────────────────────────────────────────────
// Registry
// ─────────────────────────────────────────────

// Registry stores registered tools and looks them up by name.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool. Panics on duplicate name (caught at startup).
func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; exists {
		panic("tool already registered: " + t.Name())
	}
	r.tools[t.Name()] = t
}

// Get returns the tool for name. ok is false if not registered.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	return names
}

// Definitions returns provider.ToolDefinition slices for all tools,
// ready to be passed to provider.ChatCompletionRequest.Tools.
func (r *Registry) Definitions() []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, ToProviderDefinition(t))
	}
	return defs
}
