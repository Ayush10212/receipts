package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// openAIProvider speaks the OpenAI-compatible /v1/chat/completions API.
// Both Mistral and Grok use this shape with different base URLs and keys.
type openAIProvider struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func newOpenAIProvider(name, baseURL, apiKey, model string, httpClient *http.Client) Provider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &openAIProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  httpClient,
	}
}

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *openAIProvider) Complete(ctx context.Context, messages []Message, opts Options) (Response, error) {
	if p.apiKey == "" {
		return Response{}, ErrDisabled
	}
	if p.model == "" {
		return Response{}, fmt.Errorf("provider %s: no model configured", p.name)
	}

	body := chatRequest{Model: p.model, Messages: messages}
	if opts.MaxTokens > 0 {
		body.MaxTokens = opts.MaxTokens
	}

	data, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return Response{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return Response{}, fmt.Errorf("provider %s: HTTP %d: %s", p.name, resp.StatusCode, b)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Response{}, fmt.Errorf("decode: %w", err)
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
		return Response{}, fmt.Errorf("provider %s: empty response", p.name)
	}

	model := out.Model
	if model == "" {
		model = p.model
	}
	return Response{Text: out.Choices[0].Message.Content, Model: model, Provider: p.name}, nil
}
