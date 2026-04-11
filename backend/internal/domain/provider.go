package domain

import "time"

// ProviderProfile is a saved, named provider configuration.
// APIKey is stored in the local SQLite database (user-owned).
// It is never returned to the frontend in plain text — responses show only whether a key is set.
type ProviderProfile struct {
	ID           string
	Name         string
	ProviderName string
	BaseURL      string
	APIKeyEnv    string // legacy env-var reference; only used if APIKey is empty
	APIKey       string // direct key value; takes precedence over APIKeyEnv
	Model        string
	TimeoutSec   int
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
