package openai_test

import (
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/provider/openai"
)

func TestOpenAIProvider_Name(t *testing.T) {
	p := openai.New(30)
	if p.Name() != "openai" {
		t.Errorf("expected 'openai', got %q", p.Name())
	}
}

func TestOpenAIProvider_ValidateConfig(t *testing.T) {
	p := openai.New(30)

	tests := []struct {
		name    string
		cfg     provider.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  provider.ProviderConfig{APIKey: "sk-key", BaseURL: "https://api.openai.com/v1", Model: "gpt-4.1"},
		},
		{
			name:    "missing api key",
			cfg:     provider.ProviderConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4.1"},
			wantErr: true,
		},
		{
			name:    "missing base url",
			cfg:     provider.ProviderConfig{APIKey: "sk-key", Model: "gpt-4.1"},
			wantErr: true,
		},
		{
			name:    "missing model",
			cfg:     provider.ProviderConfig{APIKey: "sk-key", BaseURL: "https://api.openai.com/v1"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := p.ValidateConfig(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := openai.DefaultConfig()
	if cfg.BaseURL != openai.DefaultBaseURL {
		t.Errorf("unexpected BaseURL: %s", cfg.BaseURL)
	}
	if cfg.Model != openai.DefaultModel {
		t.Errorf("unexpected Model: %s", cfg.Model)
	}
	if cfg.TimeoutSec != 120 {
		t.Errorf("unexpected TimeoutSec: %d", cfg.TimeoutSec)
	}
}
