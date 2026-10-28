package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/settings"
)

// A first boot must not manufacture environment overrides from compiled-in
// defaults (the rehearsal caught GRAPEVINE_MODE=off pinning mode off forever),
// and an inherited exported variable stays authoritative.
func TestApplySettingsEnvironmentInstallsOnlyFileChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAPEVINE_MODE", "")
	os.Unsetenv("GRAPEVINE_MODE")
	applySettingsEnvironment(manager)
	if _, installed := os.LookupEnv("GRAPEVINE_MODE"); installed {
		t.Fatal("default grapevine mode was installed as an environment override")
	}
	if _, installed := os.LookupEnv("DP_MUDLOG_NTFY_LEVEL"); installed {
		t.Fatal("default ntfy level was installed as an environment override")
	}

	// A real file change installs; an inherited variable is never overwritten.
	g := settings.Defaults().Grapevine
	g.Mode = "presence"
	g.ClientID = "test-client"
	secret := "test-secret"
	if _, err := manager.SaveGrapevine(manager.Revision(), g, settings.SecretUpdate{settings.SecretGrapevineClient: &secret}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAPEVINE_URL", "wss://inherited.example/socket")
	applySettingsEnvironment(manager)
	if os.Getenv("GRAPEVINE_MODE") != "presence" {
		t.Fatal("file change not installed into the environment")
	}
	if os.Getenv("GRAPEVINE_URL") != "wss://inherited.example/socket" {
		t.Fatal("inherited environment variable was overwritten")
	}
}
