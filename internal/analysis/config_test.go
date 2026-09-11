package analysis

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigReadsOpenAICompatibleModel(t *testing.T) {
	t.Setenv("TELESKOPE_TEST_LLM_KEY", "secret")
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`llm:
  provider: openai-compatible
  base_url: http://127.0.0.1:11434/v1
  api_key_env: TELESKOPE_TEST_LLM_KEY
  model: local-model
  timeout: 2s
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.LLM.Provider != "openai-compatible" || cfg.LLM.BaseURL != "http://127.0.0.1:11434/v1" || cfg.LLM.Model != "local-model" {
		t.Fatalf("unexpected config: %+v", cfg.LLM)
	}
	if cfg.LLM.APIKey != "secret" {
		t.Fatalf("api key = %q, want env value", cfg.LLM.APIKey)
	}
	if cfg.LLM.Timeout != 2*time.Second {
		t.Fatalf("timeout = %s", cfg.LLM.Timeout)
	}
}

func TestLoadConfigRequiresModelAndKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`llm:
  base_url: http://example.test/v1
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig succeeded without model and api key")
	}
}
