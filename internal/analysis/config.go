package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	LLM LLMConfig `yaml:"llm"`
}

type LLMConfig struct {
	Provider  string        `yaml:"provider"`
	BaseURL   string        `yaml:"base_url"`
	APIKey    string        `yaml:"api_key"`
	APIKeyEnv string        `yaml:"api_key_env"`
	Model     string        `yaml:"model"`
	Timeout   time.Duration `yaml:"timeout"`
	JSONMode  *bool         `yaml:"json_mode"`
}

func (c LLMConfig) UseJSONMode() bool {
	return c.JSONMode == nil || *c.JSONMode
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".teleskope", "config.yaml"), nil
}

func LoadConfig(path string) (Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return Config{}, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.LLM.applyDefaults()
	if err := cfg.LLM.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

func (c *LLMConfig) applyDefaults() {
	if c.Provider == "" {
		c.Provider = "openai-compatible"
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com/v1"
	}
	if c.APIKey == "" && c.APIKeyEnv != "" {
		c.APIKey = os.Getenv(c.APIKeyEnv)
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}
}

func (c LLMConfig) validate() error {
	if c.Provider != "openai-compatible" {
		return fmt.Errorf("llm.provider %q is unsupported; expected openai-compatible", c.Provider)
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("llm.base_url is required")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("llm.model is required")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		if c.APIKeyEnv != "" {
			return fmt.Errorf("llm.api_key_env %q is not set", c.APIKeyEnv)
		}
		return fmt.Errorf("llm.api_key or llm.api_key_env is required")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("llm.timeout must not be negative")
	}
	return nil
}
