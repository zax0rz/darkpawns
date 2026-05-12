package agentcli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfig values.
const (
	DefaultHost        = "192.168.1.106"
	DefaultPort        = 4350
	DefaultLiteLLM     = "http://192.168.1.106:4000"
	DefaultModelFast   = "zai/glm-5-turbo"
	DefaultModelFall   = "anthropic/claude-sonnet-4-6"
	DefaultTier        = "medium"
)

// AgentConfig holds all agent configuration.
type AgentConfig struct {
	Key           string  `json:"key"`
	Tier          string  `json:"tier"` // small / medium / large / unlimited
	ModelFast     string  `json:"model_fast"`
	ModelFallback string  `json:"model_fallback"`
	LiteLLM       string  `json:"litellm_endpoint"`
	GameHost      string  `json:"game_host"`
	GamePort      int     `json:"game_port"`
	Temperature   float64 `json:"temperature"`          // LLM temperature (0 = deterministic, default)
	Valence       bool    `json:"valence"`              // enable emotional valence in memory (default: true)
	LogDir        string  `json:"log_dir,omitempty"`   // local log directory (optional)
	LogLevel      string  `json:"log_level"`            // debug / info / warn / error
}

// ConfigPath returns the default config file path.
func ConfigPath() string {
	if v := os.Getenv("DP_CONFIG"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dp-agent.json")
}

// LoadConfig reads config from disk, with defaults for any missing fields.
func LoadConfig() (*AgentConfig, error) {
	cfg := &AgentConfig{
		Tier:          DefaultTier,
		ModelFast:     DefaultModelFast,
		ModelFallback: DefaultModelFall,
		LiteLLM:       DefaultLiteLLM,
		GameHost:      DefaultHost,
		GamePort:      DefaultPort,
		Temperature:   0.0,
		Valence:       true,
		LogLevel:      "info",
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // no config file, defaults are fine
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes config to disk.
func SaveConfig(cfg *AgentConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Validate checks required fields are present.
func (c *AgentConfig) Validate() error {
	if c.Key == "" && os.Getenv("DP_KEY") == "" {
		return fmt.Errorf("agent key required: set DP_KEY or run `dp-agent config --key dp_...`")
	}
	if c.Tier != "small" && c.Tier != "medium" && c.Tier != "large" && c.Tier != "unlimited" {
		return fmt.Errorf("invalid tier %q: must be small, medium, large, or unlimited", c.Tier)
	}
	return nil
}

// EffectiveKey returns the key from config or env var.
func (c *AgentConfig) EffectiveKey() string {
	if os.Getenv("DP_KEY") != "" {
		return os.Getenv("DP_KEY")
	}
	return c.Key
}
