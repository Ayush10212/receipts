package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/llm"
)

// chatResp builds a minimal OpenAI-compatible response payload.
func chatResp(model, content string) []byte {
	b, _ := json.Marshal(map[string]any{
		"model": model,
		"choices": []any{
			map[string]any{"message": map[string]any{"content": content}},
		},
	})
	return b
}

// mockServer returns a test server that responds with status and body on every request.
func mockServer(t *testing.T, status int, body []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}))
}

func testCfg(primary, fallback, mistralModel, grokModel string) config.LLMConfig {
	return config.LLMConfig{
		Enabled:      true,
		Primary:      primary,
		Fallback:     fallback,
		MistralModel: mistralModel,
		GrokModel:    grokModel,
		TimeoutMS:    5000,
	}
}

func TestRouter_PrimarySucceeds(t *testing.T) {
	primary := mockServer(t, 200, chatResp("mistral-tiny", "hello from mistral"))
	defer primary.Close()
	fallback := mockServer(t, 200, chatResp("grok-beta", "hello from grok"))
	defer fallback.Close()

	cfg := testCfg("mistral", "grok", "mistral-tiny", "grok-beta")
	r := llm.NewRouterWithURLsForTest(cfg, "key1", "key2", primary.URL, fallback.URL, primary.Client())

	resp, err := r.Complete(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "mistral" {
		t.Errorf("expected mistral provider, got %q", resp.Provider)
	}
	if resp.Text != "hello from mistral" {
		t.Errorf("unexpected text: %q", resp.Text)
	}
}

func TestRouter_PrimaryFails_FallbackSucceeds(t *testing.T) {
	primary := mockServer(t, 500, []byte(`{"error":"internal"}`))
	defer primary.Close()
	fallback := mockServer(t, 200, chatResp("grok-beta", "hello from grok"))
	defer fallback.Close()

	cfg := testCfg("mistral", "grok", "mistral-tiny", "grok-beta")
	r := llm.NewRouterWithURLsForTest(cfg, "key1", "key2", primary.URL, fallback.URL, primary.Client())

	resp, err := r.Complete(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.Options{})
	if err != nil {
		t.Fatalf("fallback should have succeeded: %v", err)
	}
	if resp.Provider != "grok" {
		t.Errorf("expected grok provider, got %q", resp.Provider)
	}
}

func TestRouter_MissingKeys_ReturnsErrDisabled(t *testing.T) {
	primary := mockServer(t, 200, chatResp("mistral-tiny", "hello"))
	defer primary.Close()
	fallback := mockServer(t, 200, chatResp("grok-beta", "hello"))
	defer fallback.Close()

	cfg := testCfg("mistral", "grok", "mistral-tiny", "grok-beta")
	// Empty keys → both providers return ErrDisabled.
	r := llm.NewRouterWithURLsForTest(cfg, "", "", primary.URL, fallback.URL, primary.Client())

	_, err := r.Complete(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.Options{})
	if !errors.Is(err, llm.ErrDisabled) {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestRouter_BothFail_ReturnsError(t *testing.T) {
	primary := mockServer(t, 500, []byte(`{"error":"down"}`))
	defer primary.Close()
	fallback := mockServer(t, 503, []byte(`{"error":"unavailable"}`))
	defer fallback.Close()

	cfg := testCfg("mistral", "grok", "mistral-tiny", "grok-beta")
	r := llm.NewRouterWithURLsForTest(cfg, "key1", "key2", primary.URL, fallback.URL, primary.Client())

	_, err := r.Complete(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.Options{})
	if err == nil {
		t.Fatal("expected error when both providers fail")
	}
	if errors.Is(err, llm.ErrDisabled) {
		t.Errorf("should NOT be ErrDisabled when keys are present but servers are down")
	}
}

func TestRouter_NilFallback_PrimaryFails_ReturnsError(t *testing.T) {
	primary := mockServer(t, 500, []byte(`{"error":"down"}`))
	defer primary.Close()

	// Same primary and fallback name → no fallback wired (primary==fallback guard).
	cfg := testCfg("mistral", "mistral", "mistral-tiny", "")
	r := llm.NewRouterWithURLsForTest(cfg, "key1", "key1", primary.URL, primary.URL, primary.Client())

	_, err := r.Complete(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.Options{})
	if err == nil {
		t.Fatal("expected error when primary fails with no fallback")
	}
}
