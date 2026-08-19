package agentcli

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestGameURLSchemeFollowsGameSecure(t *testing.T) {
	cfg := &AgentConfig{GameHost: "mud.example", GamePort: 4350}
	a := NewAgentClient(cfg)
	if got, want := a.gameURL(), "ws://mud.example:4350/ws"; got != want {
		t.Fatalf("insecure: got %q, want %q", got, want)
	}
	cfg.GameSecure = true
	if got, want := a.gameURL(), "wss://mud.example:4350/ws"; got != want {
		t.Fatalf("secure: got %q, want %q", got, want)
	}
}

func TestLoadConfig_GameSecureRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	in := &AgentConfig{GameHost: "h", GamePort: 1, GameSecure: true}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.GameSecure {
		t.Fatal("game_secure lost on round trip")
	}
}
