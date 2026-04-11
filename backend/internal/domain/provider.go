package domain

import "time"

// ProviderProfile is a saved, named provider configuration. It stores only
// metadata — the actual API key is always read from the referenced env var
// at request time. Raw keys are never persisted.
type ProviderProfile struct {
	ID           string
	Name         string // user-chosen label, e.g. "My Z.AI Coding"
	ProviderName string // "zai" | "openai" | future providers
	BaseURL      string
	APIKeyEnv    string // env var name, e.g. "ZAI_API_KEY"
	Model        string
	TimeoutSec   int
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
