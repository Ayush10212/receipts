package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Ayush10212/receipts/core/config"
)

const (
	MistralBaseURL = "https://api.mistral.ai/v1"
	GrokBaseURL    = "https://api.x.ai/v1"
)

// Router tries primary, then fallback on any error.
type Router struct {
	primary  Provider
	fallback Provider
	timeout  time.Duration
}

// NewRouter builds a Router with explicit API keys and an optional custom HTTP client.
// Pass nil httpClient to use http.DefaultClient.
func NewRouter(cfg config.LLMConfig, mistralKey, grokKey string, httpClient *http.Client) *Router {
	return newRouterWithURLs(cfg, mistralKey, grokKey, MistralBaseURL, GrokBaseURL, httpClient)
}

// NewRouterFromEnv reads MISTRAL_API_KEY and XAI_API_KEY from environment.
func NewRouterFromEnv(cfg config.LLMConfig, httpClient *http.Client) *Router {
	return NewRouter(cfg, os.Getenv("MISTRAL_API_KEY"), os.Getenv("XAI_API_KEY"), httpClient)
}

// newRouterWithURLs is used by tests to inject custom base URLs.
func newRouterWithURLs(cfg config.LLMConfig, mistralKey, grokKey, mistralURL, grokURL string, httpClient *http.Client) *Router {
	providers := map[string]Provider{
		"mistral": newOpenAIProvider("mistral", mistralURL, mistralKey, cfg.MistralModel, httpClient),
		"grok":    newOpenAIProvider("grok", grokURL, grokKey, cfg.GrokModel, httpClient),
	}

	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 8 * time.Second
	}

	r := &Router{timeout: timeout}
	if p, ok := providers[cfg.Primary]; ok {
		r.primary = p
	}
	if p, ok := providers[cfg.Fallback]; ok && cfg.Fallback != cfg.Primary {
		r.fallback = p
	}
	return r
}

// Disabled reports whether neither primary nor fallback can make calls (no API keys).
func (r *Router) Disabled() bool {
	if r.primary == nil {
		return true
	}
	// Probe by inspecting the type — if apiKey is empty, Complete returns ErrDisabled.
	// We use a context that's already cancelled so no network call is made.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled → any real HTTP call will fail fast
	_, err := r.primary.Complete(ctx, []Message{{Role: "user", Content: "ping"}}, Options{MaxTokens: 1})
	if !errors.Is(err, ErrDisabled) {
		return false // primary has a key (even if the cancelled ctx caused another error)
	}
	if r.fallback == nil {
		return true
	}
	_, err2 := r.fallback.Complete(ctx, []Message{{Role: "user", Content: "ping"}}, Options{MaxTokens: 1})
	return errors.Is(err2, ErrDisabled)
}

// Complete calls primary; on error tries fallback once.
// If both are disabled (no API key), returns ErrDisabled.
func (r *Router) Complete(ctx context.Context, messages []Message, opts Options) (Response, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if r.primary == nil {
		return Response{}, ErrDisabled
	}

	resp, primaryErr := r.primary.Complete(callCtx, messages, opts)
	if primaryErr == nil {
		return resp, nil
	}

	// Primary failed. Try fallback.
	if r.fallback == nil {
		return Response{}, primaryErr
	}

	resp2, fallbackErr := r.fallback.Complete(callCtx, messages, opts)
	if fallbackErr == nil {
		return resp2, nil
	}

	// Both failed. If both are disabled, surface ErrDisabled cleanly.
	if errors.Is(primaryErr, ErrDisabled) && errors.Is(fallbackErr, ErrDisabled) {
		return Response{}, ErrDisabled
	}
	return Response{}, fmt.Errorf("primary: %w; fallback: %v", primaryErr, fallbackErr)
}
