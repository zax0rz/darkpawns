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
	// applySettingsEnvironment writes the saved client id and secret into the
	// real process environment; pin them so cleanup undoes those writes and no
	// later test mistakes the leftovers for inherited exports.
	t.Setenv("GRAPEVINE_CLIENT_ID", "")
	os.Unsetenv("GRAPEVINE_CLIENT_ID")
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "")
	os.Unsetenv("GRAPEVINE_CLIENT_SECRET")
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

// A value boot installs into the environment is this process's own write, not
// an operator export: a console save of that field must take effect with no
// override badge for the life of the process (DP-V017-01).
func TestApplySettingsEnvironmentInstalledValuesDoNotMaskLaterSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAPEVINE_MODE", "")
	os.Unsetenv("GRAPEVINE_MODE")
	t.Setenv("GRAPEVINE_CLIENT_ID", "")
	os.Unsetenv("GRAPEVINE_CLIENT_ID")
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "")
	os.Unsetenv("GRAPEVINE_CLIENT_SECRET")

	g := settings.Defaults().Grapevine
	g.Mode = "presence"
	g.ClientID = "test-client"
	secret := "relay-secret"
	if _, err := manager.SaveGrapevine(manager.Revision(), g, settings.SecretUpdate{settings.SecretGrapevineClient: &secret}); err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if os.Getenv("GRAPEVINE_MODE") != "presence" {
		t.Fatal("file change not installed into the environment")
	}

	// The operator reverts the field in the console; the installed copy must
	// neither mask the save nor badge it as an environment override.
	g.Mode = "off"
	result, err := manager.SaveGrapevine(manager.Revision(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.EnvOverridden) != 0 {
		t.Fatalf("installed value masked the save: %v", result.EnvOverridden)
	}
	view := manager.SettingsView()
	if view.Grapevine.Mode != "off" {
		t.Fatalf("view shows %q; the boot-installed copy masks the console", view.Grapevine.Mode)
	}
	if fields := view.Meta[settings.GroupGrapevine].EnvOverridden; len(fields) != 0 {
		t.Fatalf("installed value badged as an override: %v", fields)
	}
}

// A secret boot installs must not keep a cleared secret alive for the life of
// the process, and an inherited secret export is never overwritten.
func TestApplySettingsEnvironmentSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "")
	os.Unsetenv("GRAPEVINE_CLIENT_SECRET")

	secret := "relay-secret"
	g := settings.Defaults().Grapevine
	if _, err := manager.SaveGrapevine(manager.Revision(), g, settings.SecretUpdate{settings.SecretGrapevineClient: &secret}); err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if os.Getenv("GRAPEVINE_CLIENT_SECRET") != "relay-secret" {
		t.Fatal("sealed secret not installed into the environment")
	}
	if !manager.SettingsView().Grapevine.ClientSecretSet {
		t.Fatal("installed secret not reported as set")
	}

	// Clearing the secret in the console must clear it everywhere.
	empty := ""
	if _, err := manager.SaveGrapevine(manager.Revision(), g, settings.SecretUpdate{settings.SecretGrapevineClient: &empty}); err != nil {
		t.Fatal(err)
	}
	if manager.SettingsView().Grapevine.ClientSecretSet {
		t.Fatal("cleared secret still reported set; the boot-installed copy masks the console")
	}

	// An inherited secret export is never overwritten by the installer.
	t.Setenv("DP_MUDLOG_NTFY_TOKEN", "inherited-token")
	applySettingsEnvironment(manager)
	if os.Getenv("DP_MUDLOG_NTFY_TOKEN") != "inherited-token" {
		t.Fatal("inherited secret export was overwritten")
	}
}
