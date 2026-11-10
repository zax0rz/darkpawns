package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/settings"
)

// A first boot must not manufacture environment overrides from compiled-in
// defaults, and an inherited exported variable stays authoritative.
func TestApplySettingsEnvironmentInstallsOnlyFileChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	t.Setenv("DP_MUDLOG_NTFY_LEVEL", "")
	os.Unsetenv("DP_MUDLOG_NTFY_LEVEL")
	t.Setenv("DP_MUDLOG_NTFY_TYPE", "2")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if _, installed := os.LookupEnv("DP_MUDLOG_NTFY_LEVEL"); installed {
		t.Fatal("default ntfy level was installed as an environment override")
	}

	p := settings.Defaults().StaffPush
	p.MinLevel = 32
	if _, err := manager.SaveStaffPush(manager.Revision(), p, nil); err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if os.Getenv("DP_MUDLOG_NTFY_LEVEL") != "32" {
		t.Fatal("file change not installed into the environment")
	}
	if os.Getenv("DP_MUDLOG_NTFY_TYPE") != "2" {
		t.Fatal("inherited environment variable was overwritten")
	}
}

// A value boot installs into the environment is this process's own write, not
// an operator export: a console save of that field must take effect with no
// override badge for the life of the process.
func TestApplySettingsEnvironmentInstalledValuesDoNotMaskLaterSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	t.Setenv("DP_MUDLOG_NTFY_LEVEL", "")
	os.Unsetenv("DP_MUDLOG_NTFY_LEVEL")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	p := settings.Defaults().StaffPush
	p.MinLevel = 32
	if _, err := manager.SaveStaffPush(manager.Revision(), p, nil); err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if os.Getenv("DP_MUDLOG_NTFY_LEVEL") != "32" {
		t.Fatal("file change not installed into the environment")
	}

	p.MinLevel = 31
	result, err := manager.SaveStaffPush(manager.Revision(), p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.EnvOverridden) != 0 {
		t.Fatalf("installed value masked the save: %v", result.EnvOverridden)
	}
	view := manager.SettingsView()
	if view.StaffPush.MinLevel != 31 {
		t.Fatalf("view shows min level %d; the boot-installed copy masks the console", view.StaffPush.MinLevel)
	}
	if fields := view.Meta[settings.GroupStaffPush].EnvOverridden; len(fields) != 0 {
		t.Fatalf("installed value badged as an override: %v", fields)
	}
}

// A secret boot installs must not keep a cleared secret alive for the life of
// the process, and an inherited secret export is never overwritten.
func TestApplySettingsEnvironmentSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	t.Setenv("DP_MUDLOG_NTFY_TOKEN", "")
	os.Unsetenv("DP_MUDLOG_NTFY_TOKEN")
	manager, err := settings.New(path)
	if err != nil {
		t.Fatal(err)
	}
	secret := "ntfy-token"
	p := settings.Defaults().StaffPush
	if _, err := manager.SaveStaffPush(manager.Revision(), p, settings.SecretUpdate{settings.SecretStaffPushToken: &secret}); err != nil {
		t.Fatal(err)
	}
	applySettingsEnvironment(manager)
	if os.Getenv("DP_MUDLOG_NTFY_TOKEN") != secret {
		t.Fatal("sealed token not installed into the environment")
	}
	if !manager.SettingsView().StaffPush.TokenSet {
		t.Fatal("installed token not reported as set")
	}

	empty := ""
	if _, err := manager.SaveStaffPush(manager.Revision(), p, settings.SecretUpdate{settings.SecretStaffPushToken: &empty}); err != nil {
		t.Fatal(err)
	}
	if manager.SettingsView().StaffPush.TokenSet {
		t.Fatal("cleared token still reported as set")
	}
}
