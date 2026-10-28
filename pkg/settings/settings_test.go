package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withEnv(t *testing.T, env map[string]string) {
	t.Helper()
	original := lookupEnv
	lookupEnv = func(name string) (string, bool) {
		if value, ok := env[name]; ok {
			return value, true
		}
		return "", false
	}
	t.Cleanup(func() { lookupEnv = original })
}

func newManager(t *testing.T, env map[string]string) *Manager {
	t.Helper()
	withEnv(t, env)
	m, err := New(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestNewWithoutFileWritesDefaults(t *testing.T) {
	m := newManager(t, nil)
	if got := m.Revision(); got != 1 {
		t.Fatalf("fresh revision = %d, want 1", got)
	}
	live := m.Live()
	if live.LoginIdleTimeout != 120*time.Second || live.TelnetMaxConns != 200 || live.WebSocketMaxTotalConns != 500 {
		t.Fatalf("defaults not applied: %+v", live)
	}
	if _, err := os.Stat(m.Path()); err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
}

func TestMigrationImportsEnvironmentAndSealsSecrets(t *testing.T) {
	dir := t.TempDir()
	withEnv(t, map[string]string{
		"LOGIN_IDLE_TIMEOUT":      "90",
		"GRAPEVINE_CLIENT_SECRET": "relay-secret",
		"DP_MUDLOG_NTFY_TOKEN":    "ntfy-token",
	})
	path := filepath.Join(dir, "settings.json")
	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m.Live().LoginIdleTimeout != 90*time.Second {
		t.Fatalf("LOGIN_IDLE_TIMEOUT not imported: %v", m.Live().LoginIdleTimeout)
	}
	if value, ok := m.SecretValue(SecretGrapevineClient); !ok || value != "relay-secret" {
		t.Fatalf("grapevine secret not resolvable: %q %v", value, ok)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.Contains(string(raw), "relay-secret") || strings.Contains(string(raw), "ntfy-token") {
		t.Fatal("plaintext secret material written to the settings file")
	}
	view := m.SettingsView()
	if !view.Grapevine.ClientSecretSet || !view.StaffPush.TokenSet {
		t.Fatal("secret presence not reported")
	}
	viewJSON, _ := json.Marshal(view)
	if strings.Contains(string(viewJSON), "relay-secret") || strings.Contains(string(viewJSON), "ntfy-token") {
		t.Fatal("view leaks secret material")
	}
}

func TestEnvironmentOverridesFileAndIsBadged(t *testing.T) {
	m := newManager(t, nil)
	if _, err := m.SaveSessions(1, Sessions{LoginIdleTimeoutSeconds: 60, TelnetMaxConns: 200, TelnetMaxConnsPerIP: 8, WebSocketMaxConnsPerIP: 8, WebSocketMaxTotalConns: 500, AccountFailureWindowMinutes: 60}); err != nil {
		t.Fatalf("SaveSessions: %v", err)
	}
	withEnv(t, map[string]string{"LOGIN_IDLE_TIMEOUT": "45"})
	defer withEnv(t, nil)
	reloaded, err := New(m.Path())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Live().LoginIdleTimeout != 45*time.Second {
		t.Fatalf("exported variable must win on reload: %v", reloaded.Live().LoginIdleTimeout)
	}
	view := reloaded.SettingsView()
	if len(view.Meta[GroupSessions].EnvOverridden) != 1 || view.Meta[GroupSessions].EnvOverridden[0] != "login_idle_timeout_seconds" {
		t.Fatalf("override not badged: %+v", view.Meta[GroupSessions])
	}
	// The file value is unchanged; only the effective value reflects the env.
	if reloaded.Effective().Sessions.LoginIdleTimeoutSeconds != 45 {
		t.Fatal("effective config must carry the override")
	}
}

func TestSaveConflictOnStaleRevision(t *testing.T) {
	m := newManager(t, nil)
	if _, err := m.SaveSessions(1, Defaults().Sessions); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if _, err := m.SaveSessions(1, Defaults().Sessions); err == nil {
		t.Fatal("stale revision must conflict")
	}
}

func TestValidationRejections(t *testing.T) {
	m := newManager(t, nil)
	cases := []func() error{
		func() error {
			s := Defaults().Sessions
			s.TelnetMaxConns = 0
			_, err := m.SaveSessions(m.Revision(), s)
			return err
		},
		func() error {
			_, err := m.SaveAccess(m.Revision(), Access{TrustedProxies: []string{"not-a-cidr"}})
			return err
		},
		func() error {
			_, err := m.SaveAccess(m.Revision(), Access{CORSAllowedOrigins: []string{"https://example.com/path"}})
			return err
		},
		func() error {
			_, err := m.SaveAccess(m.Revision(), Access{AdminCORSOrigin: "ftp://example.com"})
			return err
		},
		func() error {
			_, err := m.SaveGrapevine(m.Revision(), Grapevine{Mode: "sometimes", URL: "wss://grapevine.haus/socket"}, nil)
			return err
		},
		func() error {
			_, err := m.SaveGrapevine(m.Revision(), Grapevine{Mode: "presence", URL: "wss://grapevine.haus/socket", ClientID: "id"}, nil)
			return err // secret missing
		},
		func() error {
			p := Defaults().StaffPush
			p.URL = "https://ntfy.example.com/topic"
			_, err := m.SaveStaffPush(m.Revision(), p, nil)
			return err // token missing
		},
		func() error {
			_, err := m.SaveStaffPush(m.Revision(), StaffPush{MinLevel: 31, MinType: 1, RatePerMinute: 0}, nil)
			return err
		},
		func() error {
			_, err := m.SaveContact(m.Revision(), Contact{SMTPPort: 70000}, nil)
			return err
		},
	}
	for i, attempt := range cases {
		if err := attempt(); err == nil {
			t.Fatalf("case %d accepted an invalid value", i)
		}
	}
}

func TestSecretUpdateSetAndClear(t *testing.T) {
	m := newManager(t, nil)
	token := "tok-123456"
	if _, err := m.SaveStaffPush(1, StaffPush{URL: "https://ntfy.example.com/staff", MinLevel: 31, MinType: 1, RatePerMinute: 30}, SecretUpdate{SecretStaffPushToken: &token}); err != nil {
		t.Fatalf("save with secret: %v", err)
	}
	if value, ok := m.SecretValue(SecretStaffPushToken); !ok || value != token {
		t.Fatalf("secret not stored: %q %v", value, ok)
	}
	clear := ""
	result, err := m.SaveStaffPush(2, StaffPush{MinLevel: 31, MinType: 1, RatePerMinute: 30}, SecretUpdate{SecretStaffPushToken: &clear})
	if err != nil {
		t.Fatalf("clear secret: %v", err)
	}
	if _, ok := m.SecretValue(SecretStaffPushToken); ok {
		t.Fatal("cleared secret still resolvable")
	}
	// Clearing the token invalidates the URL pair: url set without token.
	if len(result.RestartRequired) == 0 {
		t.Fatal("clearing a restart-marked secret must be reported as restart-required")
	}
	if m.SettingsView().StaffPush.TokenSet {
		t.Fatal("view still reports token after clear")
	}
}

func TestLiveApplierSeesSaves(t *testing.T) {
	m := newManager(t, nil)
	var observed *Live
	m.SetOnLive(func(l *Live) { observed = l })
	m.ApplyLive()
	s := Defaults().Sessions
	s.TelnetMaxConns = 111
	if _, err := m.SaveSessions(m.Revision(), s); err != nil {
		t.Fatalf("save: %v", err)
	}
	if observed == nil || observed.TelnetMaxConns != 111 {
		t.Fatalf("live applier not notified: %+v", observed)
	}
}

func TestPreviousGenerationRetained(t *testing.T) {
	m := newManager(t, nil)
	if _, err := m.SaveSessions(1, Defaults().Sessions); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(m.Path() + ".previous")
	if err != nil {
		t.Fatalf("previous generation missing: %v", err)
	}
	var stored Config
	if err := json.Unmarshal(data, &stored); err != nil || stored.Revision != 1 {
		t.Fatalf("previous generation is not the prior file: %v %v", err, stored.Revision)
	}
}

func TestExportExcludesSecretsAndRestoreRoundTrips(t *testing.T) {
	m := newManager(t, nil)
	token := "tok-123456"
	if _, err := m.SaveStaffPush(1, StaffPush{URL: "https://ntfy.example.com/staff", MinLevel: 31, MinType: 1, RatePerMinute: 30}, SecretUpdate{SecretStaffPushToken: &token}); err != nil {
		t.Fatalf("save: %v", err)
	}
	bundle := m.Export("v0.1.7-test", time.Unix(0, 0).UTC())
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	if strings.Contains(string(raw), "tok-123456") {
		t.Fatal("export contains secret material")
	}
	// Change a value, then restore the bundle: preview reports the drift.
	s := Defaults().Sessions
	s.TelnetMaxConns = 150
	if _, err := m.SaveSessions(2, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	preview, err := m.RestoreValidate(raw)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(preview.Diff) != 1 || preview.Diff[0].Field != "sessions.telnet_max_conns" {
		t.Fatalf("preview diff wrong: %+v", preview.Diff)
	}
	if len(preview.SecretsUntouched) != 1 || preview.SecretsUntouched[0] != SecretStaffPushToken {
		t.Fatalf("secrets must be reported untouched: %+v", preview.SecretsUntouched)
	}
	result, err := m.RestoreApply(3, raw)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if m.Effective().Sessions.TelnetMaxConns != 200 {
		t.Fatal("restore did not apply bundle value")
	}
	if value, ok := m.SecretValue(SecretStaffPushToken); !ok || value != token {
		t.Fatal("restore must keep existing secrets")
	}
	_ = result
}

func TestRestoreValidateReportsUnknownKeys(t *testing.T) {
	m := newManager(t, nil)
	bundle := m.Export("v", time.Unix(0, 0).UTC())
	raw, _ := json.Marshal(bundle)
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	document["future_field"] = "ignored"
	raw, _ = json.Marshal(document)
	preview, err := m.RestoreValidate(raw)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(preview.UnknownKeys) != 1 || preview.UnknownKeys[0] != "future_field" {
		t.Fatalf("unknown key not reported: %+v", preview.UnknownKeys)
	}
}

func TestContactIncompleteIsWarningNotRejection(t *testing.T) {
	m := newManager(t, nil)
	result, err := m.SaveContact(1, Contact{To: "admin@example.com", SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPUser: "admin"}, nil)
	if err != nil {
		t.Fatalf("partial contact must save: %v", err)
	}
	if result.Warning == "" {
		t.Fatal("incomplete contact must warn")
	}
	if m.SettingsView().Contact.Effective {
		t.Fatal("incomplete contact must stay disabled")
	}
}

func TestCorruptFileDegradesInsteadOfFailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"format": 1, "revision": 1, "sessions":`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := New(path)
	if err == nil {
		t.Fatal("a corrupt file must report its error")
	}
	if m == nil {
		t.Fatal("manager must still be returned so the server can run degraded")
	}
	if m.Degraded() == nil {
		t.Fatal("degraded state not reported")
	}
	s := Defaults().Sessions
	if _, saveErr := m.SaveSessions(m.Revision(), s); saveErr == nil {
		t.Fatal("saves must refuse while degraded")
	}
	if live := m.Live(); live.TelnetMaxConns != 200 {
		t.Fatalf("defaults not applied in degraded mode: %+v", live)
	}
	// The corrupt file must be untouched for manual repair.
	data, readErr := os.ReadFile(path)
	if readErr != nil || !strings.Contains(string(data), `"revision": 1`) {
		t.Fatal("corrupt file was overwritten or altered")
	}
}
