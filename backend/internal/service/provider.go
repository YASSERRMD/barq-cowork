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

func NewProviderService(
	repo ProviderProfileRepository,
	registry *provider.Registry,
	cfg *config.Config,
) *ProviderService {
	return &ProviderService{repo: repo, registry: registry, cfg: cfg}
}

// resolveKey returns the API key for a profile. Direct key takes precedence over env var.
func resolveKey(p *domain.ProviderProfile) string {
	if p.APIKey != "" {
		return p.APIKey
	}
	if p.APIKeyEnv != "" {
		return os.Getenv(p.APIKeyEnv)
	}
	return ""
}

// Create saves a new provider profile.
func (s *ProviderService) Create(ctx context.Context,
	name, providerName, baseURL, apiKeyEnv, apiKey, model string,
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
		APIKey:       apiKey,
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

func (s *ProviderService) Get(ctx context.Context, id string) (*domain.ProviderProfile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProviderService) List(ctx context.Context) ([]*domain.ProviderProfile, error) {
	return s.repo.List(ctx)
}

func (s *ProviderService) Update(ctx context.Context,
	id, name, providerName, baseURL, apiKeyEnv, apiKey, model string,
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
	// Only overwrite key if a new one was provided (empty string = keep existing).
	if apiKey != "" {
		p.APIKey = apiKey
	}
	p.Model = model
	p.TimeoutSec = timeoutSec
	p.IsDefault = isDefault
	p.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProviderService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// AvailableProvider describes a provider known to the registry.
type AvailableProvider struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	HasKey  bool   `json:"has_key"`
	KeyEnv  string `json:"key_env"`
}

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

// TestResult is returned from TestConnection.
type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// TestConnection verifies a provider config using a direct API key or env var.
func (s *ProviderService) TestConnection(ctx context.Context, providerName, baseURL, apiKey, model string) TestResult {
	p, ok := s.registry.Get(providerName)
	if !ok {
		return TestResult{OK: false, Message: fmt.Sprintf("unknown provider: %s", providerName)}
	}

	if apiKey == "" {
		return TestResult{OK: false, Message: "API key is required to test connection"}
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
		APIKey:       apiKey,
		Model:        model,
		TimeoutSec:   15,
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

// TestProfile tests the connection for a saved profile using its stored key.
func (s *ProviderService) TestProfile(ctx context.Context, profileID string) TestResult {
	p, err := s.repo.GetByID(ctx, profileID)
	if err != nil {
		return TestResult{OK: false, Message: "profile not found"}
	}
	key := resolveKey(p)
	return s.TestConnection(ctx, p.ProviderName, p.BaseURL, key, p.Model)
}

func sanitizeError(err error) string {
	msg := err.Error()
	if idx := strings.Index(msg, "Bearer "); idx >= 0 {
		msg = msg[:idx] + "Bearer [REDACTED]"
	}
	return msg
}
