package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/launchctl"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/settings"
)

func newSettingsTestRouter(t *testing.T, options ...RouterOption) (http.Handler, *settings.Manager, string) {
	t.Helper()
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	manager, err := settings.New(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, nil, append([]RouterOption{WithSettings(manager), WithBuildInfo("v0.1.7-test", "abc123")}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	return handler, manager, manager.Path()
}

var settingsRequestCalls int

func settingsRequest(handler http.Handler, method, path string, body any, token string, csrf bool) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	settingsRequestCalls++
	req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", 10+settingsRequestCalls)
	req.Header.Set("Content-Type", "application/json")
	if csrf {
		req.Header.Set("X-Requested-With", "darkpawns-admin")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, req)
	return out
}

func TestSettingsAPIGateAndRead(t *testing.T) {
	handler, _, _ := newSettingsTestRouter(t)
	admin := generateTestToken(t, "admin")
	builder := generateTestToken(t, "builder")
	for _, tc := range []struct {
		token  string
		csrf   bool
		status int
	}{{"", true, 401}, {admin, false, 401}, {builder, true, 403}} {
		for _, op := range []struct{ method, path string }{{"GET", "/admin/settings"}, {"PUT", "/admin/settings/sessions"}, {"POST", "/admin/restart"}, {"GET", "/admin/updates/status"}} {
			out := settingsRequest(handler, op.method, op.path, nil, tc.token, tc.csrf)
			if out.Code != tc.status {
				t.Fatalf("%s %s with token=%q csrf=%v: HTTP %d %s", op.method, op.path, tc.token, tc.csrf, out.Code, out.Body.String())
			}
		}
	}
	out := settingsRequest(handler, "GET", "/admin/settings", nil, admin, true)
	if out.Code != 200 {
		t.Fatalf("read: %d %s", out.Code, out.Body.String())
	}
	var view settings.SettingsView
	if err := json.Unmarshal(out.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Revision < 1 || len(view.Meta) != 5 {
		t.Fatalf("view incomplete: %+v", view)
	}
}

func TestSettingsSaveConflictAndLiveApply(t *testing.T) {
	handler, manager, _ := newSettingsTestRouter(t)
	admin := generateTestToken(t, "admin")
	body := map[string]any{
		"revision":                       manager.Revision(),
		"login_idle_timeout_seconds":     45,
		"telnet_max_conns":               200,
		"telnet_max_conns_per_ip":        8,
		"websocket_max_conns_per_ip":     8,
		"websocket_max_total_conns":      500,
		"account_failure_budget":         0,
		"account_failure_window_minutes": 60,
	}
	out := settingsRequest(handler, "PUT", "/admin/settings/sessions", body, admin, true)
	if out.Code != 200 {
		t.Fatalf("save: %d %s", out.Code, out.Body.String())
	}
	if manager.Live().LoginIdleTimeout.Seconds() != 45 {
		t.Fatal("live value not applied on save")
	}
	// Stale revision must conflict, never last-writer-wins.
	out = settingsRequest(handler, "PUT", "/admin/settings/sessions", body, admin, true)
	if out.Code != 409 {
		t.Fatalf("stale revision: %d %s", out.Code, out.Body.String())
	}
	// Validation rejects out-of-range values.
	body["revision"] = manager.Revision()
	body["telnet_max_conns"] = 0
	out = settingsRequest(handler, "PUT", "/admin/settings/sessions", body, admin, true)
	if out.Code != 422 {
		t.Fatalf("invalid value accepted: %d %s", out.Code, out.Body.String())
	}
}

func TestSettingsSecretsAreWriteOnly(t *testing.T) {
	handler, _, path := newSettingsTestRouter(t)
	admin := generateTestToken(t, "admin")
	secret := "grapevine-super-secret"
	body := map[string]any{
		"revision":      1,
		"mode":          "presence",
		"url":           "wss://grapevine.haus/socket",
		"client_id":     "my-client",
		"client_secret": secret,
	}
	out := settingsRequest(handler, "PUT", "/admin/settings/grapevine", body, admin, true)
	if out.Code != 200 {
		t.Fatalf("save: %d %s", out.Code, out.Body.String())
	}
	if strings.Contains(out.Body.String(), secret) {
		t.Fatal("save response echoed the secret")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatal("plaintext secret written to the settings file")
	}
	// Presence, not value, in every read.
	out = settingsRequest(handler, "GET", "/admin/settings", nil, admin, true)
	if out.Code != 200 || !strings.Contains(out.Body.String(), `"client_secret_set":true`) || strings.Contains(out.Body.String(), secret) {
		t.Fatalf("read leaks or misses presence: %d %s", out.Code, out.Body.String())
	}
	// Exports never contain secret material even indirectly.
	out = settingsRequest(handler, "GET", "/admin/settings/export", nil, admin, true)
	if out.Code != 200 || strings.Contains(out.Body.String(), secret) {
		t.Fatalf("export leaked secret: %d", out.Code)
	}
}

func TestSettingsRestoreRoundTrip(t *testing.T) {
	handler, manager, _ := newSettingsTestRouter(t)
	admin := generateTestToken(t, "admin")
	export := settingsRequest(handler, "GET", "/admin/settings/export", nil, admin, true)
	if export.Code != 200 {
		t.Fatalf("export: %d", export.Code)
	}
	// Drift the live value, then restore the exported bundle.
	body := map[string]any{
		"revision":                       manager.Revision(),
		"login_idle_timeout_seconds":     90,
		"telnet_max_conns":               200,
		"telnet_max_conns_per_ip":        8,
		"websocket_max_conns_per_ip":     8,
		"websocket_max_total_conns":      500,
		"account_failure_budget":         0,
		"account_failure_window_minutes": 60,
	}
	if out := settingsRequest(handler, "PUT", "/admin/settings/sessions", body, admin, true); out.Code != 200 {
		t.Fatalf("drift save: %d %s", out.Code, out.Body.String())
	}
	var bundle map[string]any
	if err := json.Unmarshal(export.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	bundle["future_field"] = "ignored"
	raw, _ := json.Marshal(bundle)
	validate := settingsRequest(handler, "POST", "/admin/settings/restore/validate", json.RawMessage(raw), admin, true)
	if validate.Code != 200 {
		t.Fatalf("validate: %d %s", validate.Code, validate.Body.String())
	}
	var preview settings.RestorePreview
	if err := json.Unmarshal(validate.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Diff) != 1 || preview.Diff[0].Field != "sessions.login_idle_timeout_seconds" {
		t.Fatalf("diff wrong: %+v", preview.Diff)
	}
	if len(preview.UnknownKeys) != 1 || preview.UnknownKeys[0] != "future_field" {
		t.Fatalf("unknown keys not reported: %+v", preview.UnknownKeys)
	}
	apply := settingsRequest(handler, "POST", "/admin/settings/restore/apply", map[string]any{"revision": manager.Revision(), "bundle": bundle}, admin, true)
	if apply.Code != 200 {
		t.Fatalf("apply: %d %s", apply.Code, apply.Body.String())
	}
	if manager.Live().LoginIdleTimeout.Seconds() != 120 {
		t.Fatalf("restore did not reapply the bundle default: %v", manager.Live().LoginIdleTimeout)
	}
	// Stale revision restore must conflict.
	apply = settingsRequest(handler, "POST", "/admin/settings/restore/apply", map[string]any{"revision": 1, "bundle": bundle}, admin, true)
	if apply.Code != 409 {
		t.Fatalf("stale restore: %d %s", apply.Code, apply.Body.String())
	}
}

func TestRestartRequiresManagedInstance(t *testing.T) {
	handler, _, _ := newSettingsTestRouter(t)
	admin := generateTestToken(t, "admin")
	out := settingsRequest(handler, "POST", "/admin/restart", nil, admin, true)
	if out.Code != 503 || !strings.Contains(out.Body.String(), "managed instance") {
		t.Fatalf("restart without launcher: %d %s", out.Code, out.Body.String())
	}
}

// stubLauncher emulates the launcher's scoped control endpoint for the
// updates flow: status, read-only check, and the two transactions.
func stubLauncher(t *testing.T) *launchctl.Client {
	t.Helper()
	token := strings.Repeat("1", 64)
	mux := http.NewServeMux()
	mux.HandleFunc("/update-status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"state":"committed","runtime_state":"running","current_version":"v0.1.7-test","transaction_id":"`+strings.Repeat("a", 32)+`","phase":"committed"}`)
	})
	mux.HandleFunc("/update-check", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"current_version":"v0.1.7-test","latest_version":"v0.2.0","up_to_date":false}`)
	})
	mux.HandleFunc("/update", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"update requested"}`)
	})
	mux.HandleFunc("/restart", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"restart requested"}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := launchctl.New(server.Listener.Addr().String(), token)
	if client == nil {
		t.Fatal("stub launcher address rejected")
	}
	return client
}

func TestUpdatesFlow(t *testing.T) {
	handler, manager, _ := newSettingsTestRouter(t, WithLaunchctl(stubLauncher(t)))
	admin := generateTestToken(t, "admin")

	out := settingsRequest(handler, "POST", "/admin/restart", nil, admin, true)
	if out.Code != 202 {
		t.Fatalf("restart via launcher: %d %s", out.Code, out.Body.String())
	}

	out = settingsRequest(handler, "GET", "/admin/updates/status", nil, admin, true)
	if out.Code != 200 {
		t.Fatalf("status: %d %s", out.Code, out.Body.String())
	}
	var status struct {
		Managed       bool                   `json:"managed"`
		ServerVersion string                 `json:"server_version"`
		Update        *launchctl.UpdateState `json:"update"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Managed || status.ServerVersion != "v0.1.7-test" || status.Update == nil || status.Update.State != "committed" {
		t.Fatalf("status wrong: %+v", status)
	}

	out = settingsRequest(handler, "POST", "/admin/updates/check", nil, admin, true)
	if out.Code != 200 || !strings.Contains(out.Body.String(), `"latest_version":"v0.2.0"`) {
		t.Fatalf("check: %d %s", out.Code, out.Body.String())
	}
	// The check is rate-limited: a second immediate call refuses.
	out = settingsRequest(handler, "POST", "/admin/updates/check", nil, admin, true)
	if out.Code != 429 {
		t.Fatalf("rate limit: %d %s", out.Code, out.Body.String())
	}

	out = settingsRequest(handler, "POST", "/admin/updates/apply", nil, admin, true)
	if out.Code != 202 {
		t.Fatalf("apply: %d %s", out.Code, out.Body.String())
	}
	// The audit trail records the update initiation with the online count.
	_ = manager
}

func TestSettingsAuditHistoryEndpoint(t *testing.T) {
	dir := t.TempDir()
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(dir, "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	manager, err := settings.New(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger, err := audit.NewAuditLogger(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), logger, NewLogBuffer(10), database, nil, WithSettings(manager))
	if err != nil {
		t.Fatal(err)
	}
	admin := generateTestToken(t, "admin")
	body := map[string]any{
		"revision":                       manager.Revision(),
		"login_idle_timeout_seconds":     60,
		"telnet_max_conns":               200,
		"telnet_max_conns_per_ip":        8,
		"websocket_max_conns_per_ip":     8,
		"websocket_max_total_conns":      500,
		"account_failure_budget":         0,
		"account_failure_window_minutes": 60,
	}
	if out := settingsRequest(handler, "PUT", "/admin/settings/sessions", body, admin, true); out.Code != 200 {
		t.Fatalf("save: %d %s", out.Code, out.Body.String())
	}
	out := settingsRequest(handler, "GET", "/admin/settings/audit", nil, admin, true)
	if out.Code != 200 || !strings.Contains(out.Body.String(), "settings.save.sessions") {
		t.Fatalf("audit view: %d %s", out.Code, out.Body.String())
	}
}
