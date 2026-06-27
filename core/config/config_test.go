package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ayush10212/receipts/core/config"
)

func TestFileProvider_NoConfig(t *testing.T) {
	dir, err := os.MkdirTemp("", "receipts-cfg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cfg, err := config.FileProvider{}.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Defaults
	if cfg.LLM.Enabled {
		t.Error("expected LLM disabled by default")
	}
	if cfg.LLM.TimeoutMS != 8000 {
		t.Errorf("expected timeout 8000, got %d", cfg.LLM.TimeoutMS)
	}
}

func TestFileProvider_LoadsYAML(t *testing.T) {
	dir, err := os.MkdirTemp("", "receipts-cfg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	yaml := `
llm:
  enabled: true
  primary: grok
  mistral_model: mistral-large-latest
  timeout_ms: 5000
`
	if err := os.WriteFile(filepath.Join(dir, ".receipts.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.FileProvider{}.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.LLM.Enabled {
		t.Error("expected LLM enabled")
	}
	if cfg.LLM.Primary != "grok" {
		t.Errorf("primary: got %q want grok", cfg.LLM.Primary)
	}
	if cfg.LLM.MistralModel != "mistral-large-latest" {
		t.Errorf("mistral_model: got %q", cfg.LLM.MistralModel)
	}
	if cfg.LLM.TimeoutMS != 5000 {
		t.Errorf("timeout_ms: got %d want 5000", cfg.LLM.TimeoutMS)
	}
}
