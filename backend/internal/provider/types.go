// Package provider defines the LLM provider abstraction used throughout the app.
// All concrete provider implementations live in sub-packages; nothing in here
// imports them, keeping the domain clean.
package provider

import "context"

// ─────────────────────────────────────────────
// Wire types
// ─────────────────────────────────────────────

// ChatMessage is a single turn in a conversation.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant" | "tool"
	Content string
	// ToolCallID is set when Role == "tool" (returning a tool result).
	ToolCallID string
}

// ToolDefinition describes a callable tool the LLM may invoke.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any // JSON-schema object describing the arguments
}

// ToolCall is a request by the LLM to invoke a tool.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON string
}

// ChatCompletionRequest is the input to LLMProvider.Chat.
type ChatCompletionRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
	Stream      bool
	Tools       []ToolDefinition
}

// ChatCompletionChunk is one streaming delta delivered via the channel.
// When Done == true all other fields are zero.
type ChatCompletionChunk struct {
	ContentDelta string
	ToolCalls    []ToolCall
	FinishReason string // "stop" | "tool_calls" | "length" | ""
	Done         bool
}

// ─────────────────────────────────────────────
// Provider config
// ─────────────────────────────────────────────

// ProviderConfig carries runtime settings for a single provider instance.
// APIKey is the resolved secret — it must never be logged or returned to the
// frontend. It is populated from the referenced env var by the service layer.
type ProviderConfig struct {
	ProviderName string
	BaseURL      string
	APIKey       string            // resolved at call-time, never persisted
	Model        string
	TimeoutSec   int
	ExtraHeaders map[string]string
}

// ─────────────────────────────────────────────
// Provider interface
// ─────────────────────────────────────────────

// LLMProvider is the extension point for adding new AI backends.
// Implementations must be safe for concurrent use.
type LLMProvider interface {
	// Name returns the unique provider identifier, e.g. "zai" or "openai".
	Name() string

	// ValidateConfig checks that cfg has all required fields before making
	// any network calls.
	ValidateConfig(cfg ProviderConfig) error

	// Chat sends a completion request and streams chunks back through the
	// returned channel. The channel is closed after the final chunk (Done==true)
	// or on error. The caller must drain the channel even on early cancel.
	Chat(ctx context.Context, cfg ProviderConfig, req ChatCompletionRequest) (<-chan ChatCompletionChunk, error)
}

// ─────────────────────────────────────────────
// Registry
// ─────────────────────────────────────────────

// Registry stores registered LLM providers and resolves them by name.
type Registry struct {
	providers map[string]LLMProvider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]LLMProvider)}
}

// Register adds a provider. Panics if a provider with the same name is
// registered twice (caught at startup, not at request time).
func (r *Registry) Register(p LLMProvider) {
	if _, exists := r.providers[p.Name()]; exists {
		panic("provider already registered: " + p.Name())
	}
	r.providers[p.Name()] = p
}

// Get returns the provider for name. ok is false if not registered.
func (r *Registry) Get(name string) (LLMProvider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns the names of all registered providers, sorted.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	return names
}
