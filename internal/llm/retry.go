// Package llm holds the two real model clients behind domain.LLM.
// Gemini streams the AI student. Anthropic returns JSON for the checker.
package llm

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

// ErrUnsupported is returned by the method a client does not serve.
var ErrUnsupported = errors.New("llm: method not supported by this client")

// Retry tunes attempts and per-attempt time. Zero values use the defaults.
type Retry struct {
	Attempts int           // default 3
	Backoff  time.Duration // first wait, doubles each time. default 300ms
	Timeout  time.Duration // per attempt. default 30s
}

func (r Retry) withDefaults() Retry {
	if r.Attempts <= 0 {
		r.Attempts = 3
	}
	if r.Backoff <= 0 {
		r.Backoff = 300 * time.Millisecond
	}
	if r.Timeout <= 0 {
		r.Timeout = 30 * time.Second
	}
	return r
}

// do runs fn until it succeeds, fails with a non-retryable error, or runs out
// of attempts. Each attempt gets its own timeout context.
func (r Retry) do(ctx context.Context, fn func(ctx context.Context) error) error {
	r = r.withDefaults()
	var err error
	wait := r.Backoff
	for i := 0; i < r.Attempts; i++ {
		actx, cancel := context.WithTimeout(ctx, r.Timeout)
		err = fn(actx)
		cancel()
		if err == nil || ctx.Err() != nil || !retryable(err) {
			return err
		}
		if i == r.Attempts-1 {
			break
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
		wait *= 2
	}
	return err
}

// retryable is true for timeouts, network errors and 429/5xx. The SDK errors
// carry the status in their message, so this matches on text. ponytail: string
// match, swap for typed errors if an SDK exposes them stably.
func retryable(err error) bool {
	var nr errNoRetry
	if errors.As(err, &nr) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	s := err.Error()
	for _, code := range []string{"429", "500", "502", "503", "504", "UNAVAILABLE", "RESOURCE_EXHAUSTED", "overloaded"} {
		if strings.Contains(s, code) {
			return true
		}
	}
	return false
}
