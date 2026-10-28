package portable

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func configFixture(t *testing.T, mutate func(*Config)) string {
	t.Helper()
	instance := t.TempDir()
	cfg := Config{
		Format:     Format,
		HTTPPort:   4350,
		JWTSecret:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		TelnetBind: loopbackBind,
		TelnetPort: 7777,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, "instance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return instance
}

func TestSaveNetworkConfigValidatesAndBumpsRevision(t *testing.T) {
	instance := configFixture(t, nil)
	proposal := NetworkProposal{
		Telnet:    ListenerProposal{Enabled: true, Port: 42069},
		TelnetTLS: ListenerProposal{Enabled: false},
		Whod:      ListenerProposal{Enabled: true, Port: 42070},
	}
	result, err := SaveNetworkConfig(instance, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Revision != 1 || !result.RestartRequired {
		t.Fatalf("revision %d, restart %v", result.Config.Revision, result.RestartRequired)
	}
	if len(result.Diff) != 2 {
		t.Fatalf("expected telnet port and whod enable diffs, got %v", result.Diff)
	}
	cfg, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TelnetPort != 42069 || cfg.TelnetBind != loopbackBind || cfg.WhodPort != 42070 {
		t.Fatalf("network fields not saved: %+v", cfg)
	}
	// The secret and every non-network field survive untouched.
	if cfg.JWTSecret != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatal("signing secret was not preserved")
	}
}

func TestSaveNetworkConfigStaleRevisionRefused(t *testing.T) {
	instance := configFixture(t, func(c *Config) { c.ConfigRevision = 4 })
	if _, err := SaveNetworkConfig(instance, NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 7777}}); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("stale revision accepted: %v", err)
	}
}

func TestSaveNetworkConfigRejectsRangeAndCollisions(t *testing.T) {
	instance := configFixture(t, nil)
	cases := []struct {
		name      string
		proposal  NetworkProposal
		wantError string
	}{
		{"port out of range", NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 70000}}, "between 1 and 65535"},
		{"port zero", NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 0}}, "between 1 and 65535"},
		{"telnet collides with telnet tls", NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 7777}, TelnetTLS: ListenerProposal{Enabled: true, Port: 7777}}, "already used"},
		{"whod collides with http", NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 7777}, Whod: ListenerProposal{Enabled: true, Port: 4350}}, "already used by http"},
		{"tls without certificate", NetworkProposal{TelnetTLS: ListenerProposal{Enabled: true, Port: 7778}}, "certificate"},
	}
	for _, tc := range cases {
		_, err := SaveNetworkConfig(instance, tc.proposal)
		if err == nil || !contains(err.Error(), tc.wantError) {
			t.Fatalf("%s: got %v, want it to mention %q", tc.name, err, tc.wantError)
		}
	}
	// A refused proposal must not bump the revision or write anything.
	if _, err := ReadNetworkConfig(instance); err != nil || revisionOf(t, instance) != 0 {
		t.Fatalf("refused proposal changed state: %+v", err)
	}
}

func revisionOf(t *testing.T, instance string) int64 {
	t.Helper()
	cfg, err := Load(instance)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.ConfigRevision
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestSaveNetworkConfigWarnings(t *testing.T) {
	instance := configFixture(t, nil)
	result, err := SaveNetworkConfig(instance, NetworkProposal{
		Telnet:    ListenerProposal{Enabled: true, Port: 23, Public: true},
		TelnetTLS: ListenerProposal{Enabled: false},
		Whod:      ListenerProposal{Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Disabling every listener adds the web-client-only warning.
	second, err := SaveNetworkConfig(instance, NetworkProposal{
		Revision:  result.Config.Revision,
		Telnet:    ListenerProposal{Enabled: false},
		TelnetTLS: ListenerProposal{Enabled: false},
		Whod:      ListenerProposal{Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := func(warnings []string) string {
		out := ""
		for _, w := range warnings {
			out += w + "\n"
		}
		return out
	}
	for _, want := range []string{"below 1024", "plain text"} {
		if !contains(joined(result.Warnings), want) {
			t.Fatalf("first proposal warnings missing %q: %v", want, result.Warnings)
		}
	}
	if !contains(joined(second.Warnings), "web client") {
		t.Fatalf("all-off proposal warnings missing the web client note: %v", second.Warnings)
	}
}

func TestSaveNetworkConfigSnapshotsPreviousForRollback(t *testing.T) {
	instance := configFixture(t, nil)
	if _, err := SaveNetworkConfig(instance, NetworkProposal{Telnet: ListenerProposal{Enabled: true, Port: 42069}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.ReadFile(filepath.Join(instance, configRollbackFile))
	if err != nil {
		t.Fatal(err)
	}
	var before Config
	if err := json.Unmarshal(snapshot, &before); err != nil || before.TelnetPort != 7777 {
		t.Fatalf("rollback snapshot is not the previous configuration: %v %d", err, before.TelnetPort)
	}
	// The apply transaction owns its rollback target from the moment it
	// starts: a second save rewrites the save-time snapshot, but the copy
	// the transaction took still restores the configuration that was live
	// before the reviewed proposal (DP-V017-02).
	if err := takeApplySnapshot(instance); err != nil {
		t.Fatal(err)
	}
	second, err := SaveNetworkConfig(instance, NetworkProposal{
		Revision: 1,
		Telnet:   ListenerProposal{Enabled: true, Port: 42069},
		Whod:     ListenerProposal{Enabled: true, Port: 42070},
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Config.Revision != 2 {
		t.Fatalf("second save revision = %d, want 2", second.Config.Revision)
	}
	restored, err := restoreApplyRollback(instance)
	if err != nil || !restored {
		t.Fatalf("restore failed: %v %v", restored, err)
	}
	cfg, err := Load(instance)
	if err != nil || cfg.TelnetPort != 7777 || cfg.WhodPort != 0 || cfg.ConfigRevision != 0 {
		t.Fatalf("rollback did not restore the configuration live before the apply: %+v %v", cfg, err)
	}
	// Restoring without a transaction snapshot reports nothing to restore.
	discardApplySnapshot(instance)
	restored, err = restoreApplyRollback(instance)
	if err != nil || restored {
		t.Fatalf("restore without the transaction snapshot: %v %v", restored, err)
	}
}

func TestListenerTargetsCoverPublicAndLoopback(t *testing.T) {
	cfg := Config{HTTPPort: 4350, TelnetPort: 7777, TelnetBind: "", TelnetTLSPort: 7778, TelnetTLSBind: loopbackBind, WhodPort: 0}
	targets := listenerTargets(cfg)
	if len(targets) != 2 {
		t.Fatalf("expected telnet and tls targets, got %v", targets)
	}
	if targets[0].Address != "127.0.0.1:7777" || targets[1].Address != "127.0.0.1:7778" {
		t.Fatalf("public bind must be verified on loopback: %v", targets)
	}
}

func TestVerifyListenerAcceptsABoundSocket(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	if err := verifyListener(t.Context(), "127.0.0.1:"+port); err != nil {
		t.Fatalf("bound listener failed verification: %v", err)
	}
}

func TestCurrentNetworkMapsScopes(t *testing.T) {
	cfg := Config{TelnetPort: 7777, TelnetBind: "192.0.2.10", TelnetTLSPort: 0, WhodPort: 7779, WhodBind: ""}
	view := currentNetwork(cfg)
	if view.Telnet.Scope != "address" || view.Telnet.Address != "192.0.2.10" {
		t.Fatalf("explicit bind not surfaced read-only: %+v", view.Telnet)
	}
	if view.TelnetTLS.Enabled || view.Whod.Scope != "public" {
		t.Fatalf("scope mapping wrong: %+v", view)
	}
}

// The apply transaction is bound to the revision the operator reviewed: a
// configuration that changed underneath the request is refused before the
// world is touched, and the receipt says so (DP-V017-02).
func TestApplyNetworkConfigStaleRevisionRefused(t *testing.T) {
	instance := configFixture(t, func(c *Config) { c.ConfigRevision = 4 })
	if _, err := SaveNetworkConfig(instance, NetworkProposal{
		Revision: 4,
		Telnet:   ListenerProposal{Enabled: true, Port: 42069},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyNetworkConfig(t.Context(), instance, 4); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	receipt, err := ReadConfigApplyStatus(instance)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != "refused" || receipt.Error == "" {
		t.Fatalf("refusal not recorded in the receipt: %+v", receipt)
	}
	// A second operator's newer save must survive an older apply attempt.
	if got := revisionOf(t, instance); got != 5 {
		t.Fatalf("refused apply changed the configuration: revision %d, want 5", got)
	}
	if _, err := os.Stat(filepath.Join(instance, applyRollbackFile)); !os.IsNotExist(err) {
		t.Fatalf("refused apply left a transaction snapshot behind: %v", err)
	}
}
