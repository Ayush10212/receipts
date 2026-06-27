package llm

import (
	"context"
	"errors"
)

// ErrDisabled is returned when no API key is configured for the selected provider.
var ErrDisabled = errors.New("llm: disabled — no API key configured")

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options controls per-call behaviour.
type Options struct {
	MaxTokens int
}

// Response is the successful output from a provider call.
type Response struct {
	Text     string
	Model    string
	Provider string
}

// Provider is the single interface both Mistral and Grok satisfy.
type Provider interface {
	Complete(ctx context.Context, messages []Message, opts Options) (Response, error)
}
