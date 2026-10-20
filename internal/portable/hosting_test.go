package portable

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHostingConfigurationFailsClosed(t *testing.T) {
	for _, cfg := range []Config{
		{HTTPPort: 4350, TelnetPort: 4350},
		{HTTPPort: 4350, TelnetPort: 7777, WhodPort: 7777},
		{HTTPPort: 4350, WhodPort: -1},
		{HTTPPort: 4350, TelnetTLSPort: 7778},
		{HTTPPort: 4350, ServerEnv: map[string]string{"DP_ALLOW_NO_DB": "1"}},
		{HTTPPort: 4350, ServerEnv: map[string]string{"JWT_SECRET": "foreign"}},
		{HTTPPort: 4350, ServerEnv: map[string]string{"DP_MUDLET_MAP_URL": "http://example.invalid/map"}},
		{HTTPPort: 4350, ServerEnv: map[string]string{"DP_MUDLET_MAP_URL": "https://user:password@example.invalid/map"}},
		{HTTPPort: 4350, ServerEnv: map[string]string{"TELNET_TLS_KEY_FILE": "relative.key"}},
		{HTTPPort: 4350, MudletDirectory: "relative"},
	} {
		if err := cfg.validateHosting(); err == nil {
			t.Fatalf("unsafe hosting configuration accepted: %+v", cfg)
		}
	}
	cfg := Config{HTTPPort: 4350, TelnetPort: 7777, TelnetTLSPort: 7778, WhodPort: 7779, ServerEnv: map[string]string{
		"TELNET_TLS_CERT_FILE": filepath.Join(t.TempDir(), "cert"), "TELNET_TLS_KEY_FILE": filepath.Join(t.TempDir(), "key"), "DP_MUDLET_PACKAGE_URL": "https://example.invalid/darkpawns.xml",
	}}
	if err := cfg.validateHosting(); err != nil {
		t.Fatal(err)
	}
	env := strings.Join(instanceEnv([]string{"DP_MUDLET_PACKAGE_URL=https://foreign.invalid/pkg", "TELNET_TLS_KEY_FILE=foreign"}, "bundle", "instance", cfg), "\n")
	if strings.Contains(env, "foreign") || !strings.Contains(env, "DP_MUDLET_PACKAGE_URL=https://example.invalid/darkpawns.xml") {
		t.Fatal("hosting settings leaked from another instance or explicit configuration was lost")
	}
}

func TestPublishMudletPreservesOtherFilesAndRefusesMissingPayload(t *testing.T) {
	bundle, public := t.TempDir(), t.TempDir()
	source := filepath.Join(bundle, "release-assets", "mudlet")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	packageFile := filepath.Join(source, "darkpawns.xml")
	if err := os.WriteFile(packageFile, []byte("new package"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(public, "other.xml")
	if err := os.WriteFile(other, []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{MudletDirectory: public}
	for range 2 {
		if err := publishMudlet(bundle, cfg); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(filepath.Join(public, "darkpawns.xml"))
	if err != nil || string(got) != "new package" {
		t.Fatal(string(got), err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(public, "darkpawns.xml"))
		if err != nil || info.Mode().Perm() != 0o644 {
			t.Fatal("public package must be readable by the reverse proxy", err)
		}
	}
	if err := os.Remove(packageFile); err != nil {
		t.Fatal(err)
	}
	if err := publishMudlet(bundle, cfg); err == nil {
		t.Fatal("missing package accepted")
	}
	got, err = os.ReadFile(other)
	if err != nil || string(got) != "unrelated" {
		t.Fatal("unrelated public file changed", err)
	}
	got, err = os.ReadFile(filepath.Join(public, "darkpawns.xml"))
	if err != nil || string(got) != "new package" {
		t.Fatal("failed publication changed the active package", err)
	}
}

func TestSupervisionRespectsStopAndPendingRecovery(t *testing.T) {
	bundle := fixture(t)
	instance := filepath.Join(t.TempDir(), "world")
	if err := Init(bundle, instance, 4350); err != nil {
		t.Fatal(err)
	}
	if err := updateJSON(filepath.Join(instance, ".supervision.json"), supervisionState{Wanted: true}); err != nil {
		t.Fatal(err)
	}
	if err := Stop(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if err := ensureSupervised(context.Background(), "missing bundle must not launch", instance); err != nil {
		t.Fatal("an intentional stop tried to launch a game", err)
	}
	if err := setSupervisionIntent(instance, true); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(instance, ".updates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, ".updates", "transaction.json"), []byte("broken journal"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureSupervised(context.Background(), "missing bundle must not launch", instance); err != nil {
		t.Fatal("pending recovery was not left to the operator", err)
	}
	var state supervisionState
	if err := readJSON(filepath.Join(instance, ".supervision.json"), &state); err != nil || !state.Wanted {
		t.Fatal("update wait changed desired state", err)
	}
}

func TestHostedListenerPreflightRefusesOccupiedPort(t *testing.T) {
	// Occupy the same wildcard listener used by the game and its preflight.
	// On macOS an IPv4-only loopback listener can coexist with the wildcard
	// IPv6 listener selected by net.Listen("tcp", ":port").
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := preflightHosting(Config{TelnetPort: port}); err == nil {
		t.Fatal("occupied hosted listener was accepted")
	}
}

func TestNewLocalConfigAndExistingHostedBindings(t *testing.T) {
	bundle := fixture(t)
	instance := filepath.Join(t.TempDir(), "new-local")
	if err := Init(bundle, instance, 4350); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TelnetPort != 7777 || cfg.TelnetBind != "127.0.0.1" {
		t.Fatalf("local defaults: %+v", cfg)
	}
	old := Config{Format: Format, HTTPPort: 4350, TelnetPort: 7777, JWTSecret: cfg.JWTSecret}
	path := filepath.Join(instance, "instance.json")
	if err := updateJSON(path, old); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelnetBind != "" {
		t.Fatal("existing host was rebound")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("load rewrote existing configuration")
	}
	for _, bind := range []string{"localhost", "127.0.0.1:7777", "bad\nbind"} {
		if err := (Config{HTTPPort: 4350, TelnetBind: bind}).validateHosting(); err == nil {
			t.Fatal(bind)
		}
	}
}
