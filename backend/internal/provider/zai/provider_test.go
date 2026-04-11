package zai_test

import (
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/barq-cowork/barq-cowork/internal/provider/zai"
)

func TestZAIProvider_Name(t *testing.T) {
	p := zai.New(30)
	if p.Name() != "zai" {
		t.Errorf("expected 'zai', got %q", p.Name())
	}
}

func TestZAIProvider_ValidateConfig(t *testing.T) {
	p := zai.New(30)

	tests := []struct {
		name    string
		cfg     provider.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  provider.ProviderConfig{APIKey: "key", BaseURL: "https://api.z.ai/api/coding/paas/v4", Model: "GLM-4.7"},
		},
		{
			name:    "missing api key",
			cfg:     provider.ProviderConfig{BaseURL: "https://api.z.ai/api/coding/paas/v4", Model: "GLM-4.7"},
			wantErr: true,
		},
		{
			name:    "missing base url",
			cfg:     provider.ProviderConfig{APIKey: "key", Model: "GLM-4.7"},
			wantErr: true,
		},
		{
			name:    "missing model",
			cfg:     provider.ProviderConfig{APIKey: "key", BaseURL: "https://api.z.ai/api/coding/paas/v4"},
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

func TestNormalizeModel(t *testing.T) {
	cases := map[string]string{
		"glm-5.1":    "glm-5.1",
		"GLM-5.1":    "glm-5.1",
		"glm-4.7":    "GLM-4.7",
		"GLM-4.7":    "GLM-4.7",
		"glm-5-turbo": "GLM-5-Turbo",
		"unknown-model": "unknown-model",
	}
	for input, want := range cases {
		got := zai.NormalizeModel(input)
		if got != want {
			t.Errorf("NormalizeModel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDefaultCodingConfig(t *testing.T) {
	cfg := zai.DefaultCodingConfig()
	if cfg.BaseURL != zai.DefaultBaseURLCoding {
		t.Errorf("unexpected BaseURL: %s", cfg.BaseURL)
	}
	if cfg.Model != zai.DefaultModelCoding {
		t.Errorf("unexpected Model: %s", cfg.Model)
	}
	if cfg.ExtraHeaders["Accept-Language"] == "" {
		t.Error("Accept-Language header should be set")
	}
}

func TestDefaultGeneralConfig(t *testing.T) {
	cfg := zai.DefaultGeneralConfig()
	if cfg.BaseURL != zai.DefaultBaseURLGeneral {
		t.Errorf("unexpected BaseURL: %s", cfg.BaseURL)
	}
	if cfg.Model != zai.DefaultModelGeneral {
		t.Errorf("unexpected Model: %s", cfg.Model)
	}
}
