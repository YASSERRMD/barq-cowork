package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultConfig returns a config populated with safe, usable defaults.
func defaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".barq-cowork")

	return &Config{
		App: AppConfig{
			Name:    "Barq Cowork",
			Env:     "development",
			DataDir: dataDir,
		},
		LLM: LLMConfig{
			DefaultProvider: "zai",
			Providers: map[string]ProviderConfig{
				"zai": {
					Enabled:    true,
					BaseURL:    "https://api.z.ai/api/coding/paas/v4",
					APIKeyEnv:  "ZAI_API_KEY",
					Model:      "GLM-4.7",
					TimeoutSec: 120,
					ExtraHeaders: map[string]string{
						"Accept-Language": "en-US,en",
					},
				},
				"openai": {
					Enabled:      true,
					BaseURL:      "https://api.openai.com/v1",
					APIKeyEnv:    "OPENAI_API_KEY",
					Model:        "gpt-4.1",
					TimeoutSec:   120,
					ExtraHeaders: map[string]string{},
				},
			},
		},
		Storage: StorageConfig{
			Driver:     "sqlite",
			SQLitePath: filepath.Join(dataDir, "barq-cowork.db"),
		},
		Security: SecurityConfig{
			RequireApprovalForDestructiveActions: true,
			AllowedWorkspaceRoots:                []string{},
		},
	}
}

// Load reads config with the following precedence (highest wins):
//  1. Environment variables
//  2. Local config file (BARQ_CONFIG_FILE or ./configs/local.yaml or ./configs/default.yaml)
//  3. Built-in defaults
func Load() (*Config, error) {
	cfg := defaultConfig()

	// Layer 2: file
	if err := loadFile(cfg); err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	// Layer 1: env overrides (highest precedence)
	applyEnv(cfg)

	return cfg, nil
}

func loadFile(cfg *Config) error {
	candidates := []string{
		os.Getenv("BARQ_CONFIG_FILE"),
		"configs/local.yaml",
		"configs/default.yaml",
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		return nil
	}
	return nil // no file found — defaults apply
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("BARQ_APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("BARQ_DATA_DIR"); v != "" {
		cfg.App.DataDir = v
	}
	if v := os.Getenv("BARQ_LLM_PROVIDER"); v != "" {
		cfg.LLM.DefaultProvider = v
	}

	// Z.AI overrides
	if cfg.LLM.Providers == nil {
		cfg.LLM.Providers = map[string]ProviderConfig{}
	}
	applyProviderEnv(cfg, "zai", "ZAI_API_KEY", "ZAI_BASE_URL", "ZAI_MODEL")
	applyProviderEnv(cfg, "openai", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL")

	// Storage overrides
	if v := os.Getenv("BARQ_SQLITE_PATH"); v != "" {
		cfg.Storage.SQLitePath = v
	}
	if v := os.Getenv("BARQ_STORAGE_DRIVER"); v != "" {
		cfg.Storage.Driver = v
	}

	// Security overrides
	if v := os.Getenv("BARQ_REQUIRE_APPROVAL"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.Security.RequireApprovalForDestructiveActions = b
		}
	}
	if v := os.Getenv("BARQ_WORKSPACE_ROOTS"); v != "" {
		cfg.Security.AllowedWorkspaceRoots = strings.Split(v, ":")
	}
}

func applyProviderEnv(cfg *Config, name, keyEnv, baseURLEnv, modelEnv string) {
	p := cfg.LLM.Providers[name]
	if v := os.Getenv(keyEnv); v != "" {
		p.APIKeyEnv = keyEnv // keep the env var name, never store the raw key
	}
	if v := os.Getenv(baseURLEnv); v != "" {
		p.BaseURL = v
	}
	if v := os.Getenv(modelEnv); v != "" {
		p.Model = v
	}
	cfg.LLM.Providers[name] = p
}

// ResolveAPIKey reads the actual key from the env var referenced by cfg.APIKeyEnv.
// This must only be called in backend service code, never forwarded to the frontend.
func ResolveAPIKey(p ProviderConfig) string {
	if p.APIKeyEnv == "" {
		return ""
	}
	return os.Getenv(p.APIKeyEnv)
}
