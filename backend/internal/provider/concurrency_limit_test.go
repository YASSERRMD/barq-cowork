package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type blockingProvider struct {
	active int32
	peak   int32
}

func (p *blockingProvider) Name() string { return "zai" }

func (p *blockingProvider) ValidateConfig(ProviderConfig) error { return nil }

func (p *blockingProvider) Chat(_ context.Context, _ ProviderConfig, _ ChatCompletionRequest) (<-chan ChatCompletionChunk, error) {
	current := atomic.AddInt32(&p.active, 1)
	for {
		peak := atomic.LoadInt32(&p.peak)
		if current <= peak || atomic.CompareAndSwapInt32(&p.peak, peak, current) {
			break
		}
	}

	ch := make(chan ChatCompletionChunk, 1)
	go func() {
		defer close(ch)
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&p.active, -1)
		ch <- ChatCompletionChunk{Done: true}
	}()
	return ch, nil
}

func TestWithDefaultConcurrencyLimit_RespectsProviderCap(t *testing.T) {
	inner := &blockingProvider{}
	limited := WithDefaultConcurrencyLimit(inner)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, err := limited.Chat(context.Background(), ProviderConfig{ProviderName: "zai", Model: "GLM-4.7"}, ChatCompletionRequest{Stream: true})
			if err != nil {
				t.Errorf("Chat() error = %v", err)
				return
			}
			for range ch {
			}
		}()
	}
	wg.Wait()

	if peak := atomic.LoadInt32(&inner.peak); peak > 3 {
		t.Fatalf("expected provider concurrency to stay <= 3, got %d", peak)
	}
}
