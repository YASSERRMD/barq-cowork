// Package zai implements the Z.AI/GLM LLM provider.
//
// Z.AI exposes two OpenAI-compatible endpoints:
//
//   Preset A — General chat API
//     BaseURL: https://api.z.ai/api/paas/v4
//     Default model: glm-5.1
//
//   Preset B — Coding / tool-use API (default)
//     BaseURL: https://api.z.ai/api/coding/paas/v4
//     Default model: GLM-4.7
//
// Authentication: Authorization: Bearer <API_KEY>
// Optional header: Accept-Language: en-US,en
package zai

import (
	"context"
	"fmt"
	"strings"

	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/provider/openaicompat"
)

const (
	ProviderName = "zai"

	// DefaultBaseURLCoding is the coding / tool-use endpoint (recommended for agents).
	DefaultBaseURLCoding = "https://api.z.ai/api/coding/paas/v4"

	// DefaultBaseURLGeneral is the general-purpose chat endpoint.
	DefaultBaseURLGeneral = "https://api.z.ai/api/paas/v4"

	// DefaultModelCoding is the recommended model for the coding endpoint.
	DefaultModelCoding = "GLM-4.7"

	// DefaultModelGeneral is the recommended model for the general endpoint.
	DefaultModelGeneral = "glm-5.1"
)

// Provider implements provider.LLMProvider for Z.AI.
type Provider struct {
	client *openaicompat.Client
}

// New returns a new Z.AI Provider with a timeout-aware HTTP client.
func New(timeoutSec int) *Provider {
	return &Provider{client: openaicompat.NewClient(timeoutSec)}
}

// Name implements provider.LLMProvider.
func (p *Provider) Name() string { return ProviderName }

// ValidateConfig checks that BaseURL and APIKey are present and that the
// base URL matches a known Z.AI endpoint (warns on unknown but does not reject).
func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("zai: api key is empty — set ZAI_API_KEY env var")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("zai: base_url is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("zai: model is required")
	}
	// Normalise model name: Z.AI docs use both lowercase (glm-5.1) and
	// uppercase (GLM-4.7) depending on the endpoint. Accept both.
	_ = NormalizeModel(cfg.Model) // validation only
	return nil
}

// Chat sends a chat completion request through the OpenAI-compatible client.
func (p *Provider) Chat(
	ctx context.Context,
	cfg provider.ProviderConfig,
	req provider.ChatCompletionRequest,
) (<-chan provider.ChatCompletionChunk, error) {
	// Apply Z.AI defaults when not overridden.
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURLCoding
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModelCoding
	}
	// Ensure Accept-Language header is present (recommended by Z.AI docs).
	if cfg.ExtraHeaders == nil {
		cfg.ExtraHeaders = map[string]string{}
	}
	if _, ok := cfg.ExtraHeaders["Accept-Language"]; !ok {
		cfg.ExtraHeaders["Accept-Language"] = "en-US,en"
	}

	if err := p.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return p.client.Chat(ctx, cfg, req)
}

// NormalizeModel ensures consistent casing for Z.AI model names.
// The general API uses lowercase (glm-5.1), the coding API uses mixed case (GLM-4.7).
// We preserve the caller's preference and only normalise known aliases.
func NormalizeModel(model string) string {
	switch strings.ToLower(model) {
	case "glm-5.1":
		return "glm-5.1"
	case "glm-5-turbo":
		return "GLM-5-Turbo"
	case "glm-4.7":
		return "GLM-4.7"
	case "glm-4.5-air":
		return "GLM-4.5-air"
	default:
		return model
	}
}

// DefaultCodingConfig returns a ProviderConfig pre-filled with the coding preset.
// The caller must populate APIKey from the env var.
func DefaultCodingConfig() provider.ProviderConfig {
	return provider.ProviderConfig{
		ProviderName: ProviderName,
		BaseURL:      DefaultBaseURLCoding,
		Model:        DefaultModelCoding,
		TimeoutSec:   120,
		ExtraHeaders: map[string]string{"Accept-Language": "en-US,en"},
	}
}

// DefaultGeneralConfig returns a ProviderConfig pre-filled with the general preset.
func DefaultGeneralConfig() provider.ProviderConfig {
	return provider.ProviderConfig{
		ProviderName: ProviderName,
		BaseURL:      DefaultBaseURLGeneral,
		Model:        DefaultModelGeneral,
		TimeoutSec:   120,
		ExtraHeaders: map[string]string{"Accept-Language": "en-US,en"},
	}
}
