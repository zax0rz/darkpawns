package agentcli

import (
	"os"
	"testing"
)

func TestValidate_PlayerNamePathTraversalRejected(t *testing.T) {
	cases := []string{
		"../../etc/passwd",
		"foo/../bar",
		"foo/bar/baz",
		"foo\\bar",
		"..",
	}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := &AgentConfig{
				Key:        "dp_test_key",
				PlayerName: name,
				Tier:       DefaultTier,
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected validation error for player_name %q, got nil", name)
			}
		})
	}
}

func TestValidate_PlayerNameNormalPasses(t *testing.T) {
	cfg := &AgentConfig{
		Key:        "dp_test_key",
		PlayerName: "Hero",
		Tier:       DefaultTier,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected validation to pass, got: %v", err)
	}
}

func TestValidate_LogDirCleaned(t *testing.T) {
	cfg := &AgentConfig{
		Key:        "dp_test_key",
		PlayerName: "Hero",
		Tier:       DefaultTier,
		LogDir:     "/tmp/foo/../bar",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got: %v", err)
	}
	if cfg.LogDir != "/tmp/bar" {
		t.Errorf("expected LogDir cleaned to %q, got %q", "/tmp/bar", cfg.LogDir)
	}
}

func TestValidate_KeyFromEnv(t *testing.T) {
	os.Setenv("DP_KEY", "dp_env_key")
	defer os.Unsetenv("DP_KEY")

	cfg := &AgentConfig{
		PlayerName: "Hero",
		Tier:       DefaultTier,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected validation to pass with DP_KEY set, got: %v", err)
	}
}
