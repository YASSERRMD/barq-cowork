// Package openai implements the OpenAI LLM provider.
package openai

import (
	"context"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/provider/openaicompat"
)

const (
	ProviderName   = "openai"
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-4.1"
)

// Provider implements provider.LLMProvider for OpenAI.
type Provider struct {
	client *openaicompat.Client
}

// New returns a new OpenAI Provider.
func New(timeoutSec int) *Provider {
	return &Provider{client: openaicompat.NewClient(timeoutSec)}
}

// Name implements provider.LLMProvider.
func (p *Provider) Name() string { return ProviderName }

// ValidateConfig checks required fields.
func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("openai: api key is empty — set OPENAI_API_KEY env var")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("openai: base_url is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("openai: model is required")
	}
	return nil
}

// Chat sends a chat completion request.
func (p *Provider) Chat(
	ctx context.Context,
	cfg provider.ProviderConfig,
	req provider.ChatCompletionRequest,
) (<-chan provider.ChatCompletionChunk, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if err := p.ValidateConfig(cfg); err != nil {
		return nil, err
	}
	return p.client.Chat(ctx, cfg, req)
}

// DefaultConfig returns a ProviderConfig pre-filled with OpenAI defaults.
func DefaultConfig() provider.ProviderConfig {
	return provider.ProviderConfig{
		ProviderName: ProviderName,
		BaseURL:      DefaultBaseURL,
		Model:        DefaultModel,
		TimeoutSec:   120,
		ExtraHeaders: map[string]string{},
	}
}
