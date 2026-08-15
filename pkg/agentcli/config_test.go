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

// TestLoadConfigFrom_ExplicitPath verifies LoadConfigFrom reads the given path
// (the --config override) rather than the default ConfigPath.
func TestLoadConfigFrom_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/custom.json"
	if err := os.WriteFile(path, []byte(`{"player_name":"Zork","game_port":9999}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}
	if cfg.PlayerName != "Zork" {
		t.Errorf("PlayerName = %q, want Zork", cfg.PlayerName)
	}
	if cfg.GamePort != 9999 {
		t.Errorf("GamePort = %d, want 9999", cfg.GamePort)
	}
}

// TestLoadConfigFrom_MissingPathUsesDefaults verifies a non-existent explicit
// path returns defaults without error (mirrors the missing-default-file case).
func TestLoadConfigFrom_MissingPathUsesDefaults(t *testing.T) {
	cfg, err := LoadConfigFrom(t.TempDir() + "/nope.json")
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}
	if cfg.GamePort != DefaultPort {
		t.Errorf("GamePort = %d, want default %d", cfg.GamePort, DefaultPort)
	}
}

func TestLoadConfigFrom_SecureFlagParsed(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/secure.json"
	if err := os.WriteFile(path, []byte(`{"player_name":"Zork","game_secure":true}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}
	if !cfg.Secure {
		t.Errorf("Secure = false, want true from game_secure config")
	}
}

func TestWsScheme(t *testing.T) {
	if got := wsScheme(&AgentConfig{Secure: true}); got != "wss" {
		t.Errorf("wsScheme(secure) = %q, want wss", got)
	}
	if got := wsScheme(&AgentConfig{}); got != "ws" {
		t.Errorf("wsScheme(insecure) = %q, want ws", got)
	}
}
