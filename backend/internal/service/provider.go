package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/config"
	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/provider"
	"github.com/google/uuid"
)

// ProviderProfileRepository is the minimal store interface needed here.
type ProviderProfileRepository interface {
	Create(ctx context.Context, p *domain.ProviderProfile) error
	GetByID(ctx context.Context, id string) (*domain.ProviderProfile, error)
	List(ctx context.Context) ([]*domain.ProviderProfile, error)
	Update(ctx context.Context, p *domain.ProviderProfile) error
	Delete(ctx context.Context, id string) error
}

// ProviderService manages provider profiles and tests live connections.
type ProviderService struct {
	repo     ProviderProfileRepository
	registry *provider.Registry
	cfg      *config.Config
}

// NewProviderService creates a ProviderService.
func NewProviderService(
	repo ProviderProfileRepository,
	registry *provider.Registry,
	cfg *config.Config,
) *ProviderService {
	return &ProviderService{repo: repo, registry: registry, cfg: cfg}
}

// ─────────────────────────────────────────────
// Provider profile CRUD
// ─────────────────────────────────────────────

// Create saves a new provider profile. The API key must be referenced by env var name.
func (s *ProviderService) Create(ctx context.Context,
	name, providerName, baseURL, apiKeyEnv, model string,
	timeoutSec int, isDefault bool,
) (*domain.ProviderProfile, error) {
	if name == "" {
		return nil, &domain.ValidationError{Field: "name", Message: "required"}
	}
	if providerName == "" {
		return nil, &domain.ValidationError{Field: "provider_name", Message: "required"}
	}
	if _, ok := s.registry.Get(providerName); !ok {
		return nil, &domain.ValidationError{
			Field:   "provider_name",
			Message: fmt.Sprintf("unknown provider %q — registered: %s", providerName, strings.Join(s.registry.List(), ", ")),
		}
	}

	now := time.Now().UTC()
	p := &domain.ProviderProfile{
		ID:           uuid.NewString(),
		Name:         name,
		ProviderName: providerName,
		BaseURL:      baseURL,
		APIKeyEnv:    apiKeyEnv,
		Model:        model,
		TimeoutSec:   timeoutSec,
		IsDefault:    isDefault,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Get retrieves a provider profile by ID.
func (s *ProviderService) Get(ctx context.Context, id string) (*domain.ProviderProfile, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all saved provider profiles.
func (s *ProviderService) List(ctx context.Context) ([]*domain.ProviderProfile, error) {
	return s.repo.List(ctx)
}

// Update replaces the mutable fields of a provider profile.
func (s *ProviderService) Update(ctx context.Context,
	id, name, providerName, baseURL, apiKeyEnv, model string,
	timeoutSec int, isDefault bool,
) (*domain.ProviderProfile, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Name = name
	p.ProviderName = providerName
	p.BaseURL = baseURL
	p.APIKeyEnv = apiKeyEnv
	p.Model = model
	p.TimeoutSec = timeoutSec
	p.IsDefault = isDefault
	p.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete removes a provider profile.
func (s *ProviderService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ─────────────────────────────────────────────
// Available provider listing (from registry + config)
// ─────────────────────────────────────────────

// AvailableProvider describes a provider known to the registry.
type AvailableProvider struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	HasKey   bool   `json:"has_key"` // true if the env var is set (non-empty)
	KeyEnv   string `json:"key_env"` // env var name — never the value
}

// ListAvailable returns all registered providers with their config status.
func (s *ProviderService) ListAvailable() []AvailableProvider {
	names := s.registry.List()
	out := make([]AvailableProvider, 0, len(names))
	for _, name := range names {
		pc := s.cfg.LLM.Providers[name]
		hasKey := os.Getenv(pc.APIKeyEnv) != ""
		out = append(out, AvailableProvider{
			Name:    name,
			Enabled: pc.Enabled,
			BaseURL: pc.BaseURL,
			Model:   pc.Model,
			HasKey:  hasKey,
			KeyEnv:  pc.APIKeyEnv,
		})
	}
	return out
}

// ─────────────────────────────────────────────
// Test connection
// ─────────────────────────────────────────────

// TestResult is returned from TestConnection.
type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// TestConnection sends a tiny probe request to verify a provider config works.
// The API key is resolved here from the env var — it is NEVER returned or logged.
func (s *ProviderService) TestConnection(ctx context.Context, providerName, baseURL, apiKeyEnv, model string) TestResult {
	p, ok := s.registry.Get(providerName)
	if !ok {
		return TestResult{OK: false, Message: fmt.Sprintf("unknown provider: %s", providerName)}
	}

	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return TestResult{
			OK:      false,
			Message: fmt.Sprintf("env var %s is not set", apiKeyEnv),
		}
	}

	if baseURL == "" {
		if pc, exists := s.cfg.LLM.Providers[providerName]; exists {
			baseURL = pc.BaseURL
		}
	}
	if model == "" {
		if pc, exists := s.cfg.LLM.Providers[providerName]; exists {
			model = pc.Model
		}
	}

	cfg := provider.ProviderConfig{
		ProviderName: providerName,
		BaseURL:      baseURL,
		APIKey:       apiKey, // resolved here, never forwarded
		Model:        model,
		TimeoutSec:   15, // short timeout for probe
		ExtraHeaders: map[string]string{"Accept-Language": "en-US,en"},
	}

	if err := p.ValidateConfig(cfg); err != nil {
		return TestResult{OK: false, Message: err.Error()}
	}

	req := provider.ChatCompletionRequest{
		Model:     model,
		Stream:    false,
		MaxTokens: 5,
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Reply with the single word: ok"},
		},
	}

	ch, err := p.Chat(ctx, cfg, req)
	if err != nil {
		return TestResult{OK: false, Message: sanitizeError(err)}
	}

	// Drain the channel to avoid goroutine leak.
	var got string
	for chunk := range ch {
		if chunk.Done {
			break
		}
		got += chunk.ContentDelta
	}

	if got == "" {
		got = "(response received)"
	}
	return TestResult{OK: true, Message: fmt.Sprintf("Connected. Response: %s", strings.TrimSpace(got))}
}

// sanitizeError removes any accidental secret leakage from error strings.
// Practical safety net — keys should never appear in errors from our code,
// but third-party HTTP libraries may include request headers in errors.
func sanitizeError(err error) string {
	msg := err.Error()
	// Remove Bearer tokens if somehow present
	if idx := strings.Index(msg, "Bearer "); idx >= 0 {
		msg = msg[:idx] + "Bearer [REDACTED]"
	}
	return msg
}
