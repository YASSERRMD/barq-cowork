// Package config defines the application configuration types and constants.
package config

// Config is the root application configuration.
type Config struct {
	App      AppConfig      `yaml:"app"`
	LLM      LLMConfig      `yaml:"llm"`
	Storage  StorageConfig  `yaml:"storage"`
	Security SecurityConfig `yaml:"security"`
}

// AppConfig holds general application settings.
type AppConfig struct {
	Name    string `yaml:"name"`
	Env     string `yaml:"env"`
	DataDir string `yaml:"data_dir"`
}

// LLMConfig holds provider configuration.
type LLMConfig struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}

// ProviderConfig holds a single provider's settings.
// APIKey is never stored directly; use APIKeyEnv to reference an env var.
type ProviderConfig struct {
	Enabled      bool              `yaml:"enabled"`
	BaseURL      string            `yaml:"base_url"`
	APIKeyEnv    string            `yaml:"api_key_env"`
	Model        string            `yaml:"model"`
	TimeoutSec   int               `yaml:"timeout_sec"`
	ExtraHeaders map[string]string `yaml:"extra_headers"`
}

// StorageConfig holds persistence settings.
type StorageConfig struct {
	Driver     string `yaml:"driver"`      // "sqlite" | "postgres"
	SQLitePath string `yaml:"sqlite_path"` // only used when driver == "sqlite"
	DSN        string `yaml:"dsn"`         // only used when driver == "postgres"
}

// SecurityConfig holds safety-related settings.
type SecurityConfig struct {
	RequireApprovalForDestructiveActions bool     `yaml:"require_approval_for_destructive_actions"`
	AllowedWorkspaceRoots                []string `yaml:"allowed_workspace_roots"`
}
