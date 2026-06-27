package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type LLMConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Primary      string `yaml:"primary"`
	Fallback     string `yaml:"fallback"`
	MistralModel string `yaml:"mistral_model"`
	GrokModel    string `yaml:"grok_model"`
	TimeoutMS    int    `yaml:"timeout_ms"`
}

type Config struct {
	LLM LLMConfig `yaml:"llm"`
}

func defaultConfig() Config {
	return Config{
		LLM: LLMConfig{
			Enabled:   false,
			Primary:   "mistral",
			Fallback:  "grok",
			TimeoutMS: 8000,
		},
	}
}

type Provider interface {
	Load(workdir string) (Config, error)
}

type FileProvider struct{}

// Load walks upward from workdir merging .receipts.yaml files (child overrides parent).
func (FileProvider) Load(workdir string) (Config, error) {
	cfg := defaultConfig()

	// Collect candidate directories from workdir up to root.
	var dirs []string
	dir := workdir
	for {
		dirs = append(dirs, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Apply from root → workdir so child values override.
	for i := len(dirs) - 1; i >= 0; i-- {
		path := filepath.Join(dirs[i], ".receipts.yaml")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return cfg, err
		}
		var overlay Config
		if err := yaml.Unmarshal(data, &overlay); err != nil {
			return cfg, err
		}
		mergeConfig(&cfg, overlay)
	}
	return cfg, nil
}

func mergeConfig(base *Config, overlay Config) {
	if overlay.LLM.Enabled {
		base.LLM.Enabled = true
	}
	if overlay.LLM.Primary != "" {
		base.LLM.Primary = overlay.LLM.Primary
	}
	if overlay.LLM.Fallback != "" {
		base.LLM.Fallback = overlay.LLM.Fallback
	}
	if overlay.LLM.MistralModel != "" {
		base.LLM.MistralModel = overlay.LLM.MistralModel
	}
	if overlay.LLM.GrokModel != "" {
		base.LLM.GrokModel = overlay.LLM.GrokModel
	}
	if overlay.LLM.TimeoutMS != 0 {
		base.LLM.TimeoutMS = overlay.LLM.TimeoutMS
	}
}
