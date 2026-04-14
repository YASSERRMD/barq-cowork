package provider

import "context"

type concurrencyLimitedProvider struct {
	inner LLMProvider
	sem   chan struct{}
}

// WithDefaultConcurrencyLimit wraps a provider with a conservative semaphore
// based on the provider's own runtime profile.
func WithDefaultConcurrencyLimit(inner LLMProvider) LLMProvider {
	if inner == nil {
		return nil
	}
	limit := SuggestedMaxConcurrentRequests(ProviderConfig{ProviderName: inner.Name()})
	if limit <= 0 {
		return inner
	}
	return &concurrencyLimitedProvider{
		inner: inner,
		sem:   make(chan struct{}, limit),
	}
}

func (p *concurrencyLimitedProvider) Name() string {
	return p.inner.Name()
}

func (p *concurrencyLimitedProvider) ValidateConfig(cfg ProviderConfig) error {
	return p.inner.ValidateConfig(cfg)
}

func (p *concurrencyLimitedProvider) Chat(
	ctx context.Context,
	cfg ProviderConfig,
	req ChatCompletionRequest,
) (<-chan ChatCompletionChunk, error) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	ch, err := p.inner.Chat(ctx, cfg, req)
	if err != nil {
		<-p.sem
		return nil, err
	}

	out := make(chan ChatCompletionChunk, 32)
	go func() {
		defer close(out)
		defer func() { <-p.sem }()
		for chunk := range ch {
			out <- chunk
		}
	}()
	return out, nil
}
