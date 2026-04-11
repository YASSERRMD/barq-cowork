package provider

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// RetryConfig controls the exponential-backoff retry behaviour applied to
// transient LLM provider errors.
type RetryConfig struct {
	// MaxAttempts is the total number of tries (initial + retries).
	// Default: 3.
	MaxAttempts int

	// BaseDelay is the wait before the second attempt.
	// Each subsequent attempt doubles this value (plus jitter).
	// Default: 1s.
	BaseDelay time.Duration

	// MaxDelay caps the inter-attempt delay.
	// Default: 30s.
	MaxDelay time.Duration
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	}
}

// ChatWithRetry calls provider.Chat, retrying on transient errors with
// exponential back-off + jitter. It returns the first successful channel,
// or the last error if all attempts are exhausted.
func ChatWithRetry(
	ctx context.Context,
	p LLMProvider,
	cfg ProviderConfig,
	req ChatCompletionRequest,
	rc RetryConfig,
	logger *slog.Logger,
) (<-chan ChatCompletionChunk, error) {
	if rc.MaxAttempts <= 0 {
		rc.MaxAttempts = 3
	}
	if rc.BaseDelay <= 0 {
		rc.BaseDelay = time.Second
	}
	if rc.MaxDelay <= 0 {
		rc.MaxDelay = 30 * time.Second
	}

	var lastErr error
	delay := rc.BaseDelay

	for attempt := 1; attempt <= rc.MaxAttempts; attempt++ {
		ch, err := p.Chat(ctx, cfg, req)
		if err == nil {
			return ch, nil
		}
		lastErr = err

		if !isTransient(err) {
			// Non-transient: authentication error, bad request, etc.
			// Retrying won't help; surface immediately.
			return nil, err
		}

		if attempt == rc.MaxAttempts {
			break
		}

		// Exponential back-off with ±25 % jitter.
		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		wait := delay + jitter
		if wait > rc.MaxDelay {
			wait = rc.MaxDelay
		}

		logger.Warn("provider call failed, retrying",
			"provider", p.Name(),
			"attempt", attempt,
			"max_attempts", rc.MaxAttempts,
			"wait", wait.Round(time.Millisecond),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}

		delay *= 2
	}

	return nil, lastErr
}

// isTransient returns true for errors that are worth retrying:
// network errors, timeouts, and HTTP 429/5xx-class issues.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	// Context cancellation / deadline — do not retry.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network-level transients.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	msg := strings.ToLower(err.Error())

	// HTTP status codes embedded in error strings (common pattern across providers).
	for _, keyword := range []string{
		"429",                   // rate limit
		"500", "502", "503", "504", // server-side errors
		"timeout", "connection reset", "eof",
		"temporary failure", "rate limit", "overloaded",
	} {
		if strings.Contains(msg, keyword) {
			return true
		}
	}

	// Unwrap any wrapped *http.Response if providers surface them.
	var httpErr interface{ StatusCode() int }
	if errors.As(err, &httpErr) {
		code := httpErr.StatusCode()
		return code == http.StatusTooManyRequests ||
			(code >= 500 && code < 600)
	}

	return false
}
