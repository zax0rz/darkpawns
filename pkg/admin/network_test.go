package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zax0rz/darkpawns/internal/launchctl"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// stubNetworkLauncher emulates the launcher's config control surface.
type stubNetworkLauncher struct {
	revision   atomic.Int64
	saved      atomic.Int64
	applied    atomic.Int64
	rejectNext atomic.Bool
	config     atomic.Value // launchctl.NetworkConfig
}

func newStubNetworkLauncher() *stubNetworkLauncher {
	s := &stubNetworkLauncher{}
	s.revision.Store(2)
	s.config.Store(launchctl.NetworkConfig{
		Revision:  2,
		HTTPPort:  4350,
		Telnet:    launchctl.Listener{Enabled: true, Port: 7777, Scope: "loopback"},
		TelnetTLS: launchctl.Listener{Enabled: false},
		Whod:      launchctl.Listener{Enabled: false},
	})
	return s
}

func (s *stubNetworkLauncher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/config":
		switch r.Method {
		case http.MethodGet:
			cfg := s.config.Load().(launchctl.NetworkConfig)
			cfg.Revision = s.revision.Load()
			_ = json.NewEncoder(w).Encode(cfg)
		case http.MethodPost:
			if s.rejectNext.CompareAndSwap(true, false) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "telnet port 7777 is already used by telnet_tls"})
				return
			}
			var proposal launchctl.NetworkProposal
			if err := json.NewDecoder(r.Body).Decode(&proposal); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if proposal.Revision != s.revision.Load() {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "configuration changed; reload before saving"})
				return
			}
			s.revision.Add(1)
			s.saved.Add(1)
			_ = json.NewEncoder(w).Encode(launchctl.SaveNetworkResult{
				Config:          launchctl.NetworkConfig{Revision: s.revision.Load()},
				RestartRequired: true,
				Diff:            []launchctl.NetworkDiff{{Field: "telnet", From: "port 7777, this computer", To: "port 42069, this computer"}},
				Warnings:        []string{"Public telnet without TLS sends passwords in plain text."},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case "/config-apply":
		// Mirrors the launcher: the reviewed revision must travel with the
		// request and must still be current.
		var request struct {
			Revision *int64 `json:"revision"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Revision == nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "the revision of the configuration you reviewed is required"})
			return
		}
		if *request.Revision != s.revision.Load() {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "configuration changed; reload before saving"})
			return
		}
		s.applied.Add(1)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"apply requested"}`))
	case "/config-status":
		_ = json.NewEncoder(w).Encode(launchctl.ConfigApplyStatus{State: "committed"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newNetworkTestRouter(t *testing.T, control *launchctl.Client) http.Handler {
	t.Helper()
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, nil,
		WithLaunchctl(control),
		WithNetworkObserved(func() NetworkObserved {
			return NetworkObserved{HTTPPort: 4350, HTTPBind: "127.0.0.1", Telnet: "127.0.0.1:7777"}
		}))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func networkRequest(handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = "192.0.2.40:1234"
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Requested-With", "darkpawns-admin")
	}
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, req)
	return out
}

func TestNetworkGateAndRead(t *testing.T) {
	server := httptest.NewServer(newStubNetworkLauncher())
	defer server.Close()
	control := launchctl.New(server.Listener.Addr().String(), strings.Repeat("1", 64))
	if control == nil {
		t.Fatal("stub address rejected")
	}
	handler := newNetworkTestRouter(t, control)
	adminToken := generateTestToken(t, "admin")
	builderToken := generateTestToken(t, "builder")

	// Gate: builder cannot touch the surface at all.
	for _, op := range []struct{ method, path string }{{"GET", "/admin/network"}, {"PUT", "/admin/network"}, {"POST", "/admin/network/apply"}} {
		out := networkRequest(handler, op.method, op.path, nil, builderToken)
		if out.Code != 403 {
			t.Fatalf("%s %s builder: HTTP %d", op.method, op.path, out.Code)
		}
	}

	out := networkRequest(handler, "GET", "/admin/network", nil, adminToken)
	if out.Code != 200 {
		t.Fatalf("read: %d %s", out.Code, out.Body.String())
	}
	var view networkView
	if err := json.Unmarshal(out.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if !view.Managed || view.Revision != 2 || !view.HTTP.ReadOnly || view.Telnet.Observed != "127.0.0.1:7777" {
		t.Fatalf("view wrong: %+v", view)
	}
	if !strings.Contains(view.HTTP.HowToChange, "instance.json") {
		t.Fatalf("HTTP read-only explanation missing: %q", view.HTTP.HowToChange)
	}
}

func TestNetworkSaveDraftFlow(t *testing.T) {
	server := httptest.NewServer(newStubNetworkLauncher())
	defer server.Close()
	control := launchctl.New(server.Listener.Addr().String(), strings.Repeat("1", 64))
	handler := newNetworkTestRouter(t, control)
	adminToken := generateTestToken(t, "admin")

	proposal := map[string]any{
		"revision":   2,
		"telnet":     map[string]any{"enabled": true, "port": 42069, "public": false},
		"telnet_tls": map[string]any{"enabled": false, "port": 0, "public": false},
		"whod":       map[string]any{"enabled": false, "port": 0, "public": false},
	}
	out := networkRequest(handler, "PUT", "/admin/network", proposal, adminToken)
	if out.Code != 200 {
		t.Fatalf("save: %d %s", out.Code, out.Body.String())
	}
	var saved networkSaveResponse
	if err := json.Unmarshal(out.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.RestartRequired || len(saved.Diff) != 1 || saved.Diff[0].To != "port 42069, this computer" {
		t.Fatalf("save response wrong: %+v", saved)
	}
	if len(saved.Warnings) == 0 {
		t.Fatal("warnings not surfaced")
	}

	// Stale revision conflicts.
	out = networkRequest(handler, "PUT", "/admin/network", proposal, adminToken)
	if out.Code != 409 {
		t.Fatalf("stale revision: %d %s", out.Code, out.Body.String())
	}

	// Launcher-side validation errors map to 422 with the message.
	stale := newStubNetworkLauncher()
	stale.rejectNext.Store(true)
	staleServer := httptest.NewServer(stale)
	defer staleServer.Close()
	staleControl := launchctl.New(staleServer.Listener.Addr().String(), strings.Repeat("1", 64))
	staleHandler := newNetworkTestRouter(t, staleControl)
	out = networkRequest(staleHandler, "PUT", "/admin/network", map[string]any{
		"revision":   2,
		"telnet":     map[string]any{"enabled": true, "port": 7777, "public": false},
		"telnet_tls": map[string]any{"enabled": true, "port": 7777, "public": false},
		"whod":       map[string]any{"enabled": false, "port": 0, "public": false},
	}, adminToken)
	if out.Code != 422 || !strings.Contains(out.Body.String(), "already used") {
		t.Fatalf("validation refusal: %d %s", out.Code, out.Body.String())
	}
}

func TestNetworkApplyAndStatus(t *testing.T) {
	stub := newStubNetworkLauncher()
	server := httptest.NewServer(stub)
	defer server.Close()
	control := launchctl.New(server.Listener.Addr().String(), strings.Repeat("1", 64))
	handler := newNetworkTestRouter(t, control)
	adminToken := generateTestToken(t, "admin")

	out := networkRequest(handler, "POST", "/admin/network/apply", map[string]any{"revision": 2}, adminToken)
	if out.Code != 202 {
		t.Fatalf("apply: %d %s", out.Code, out.Body.String())
	}
	if stub.applied.Load() != 1 {
		t.Fatal("apply did not reach the launcher")
	}
	// An apply bound to a stale revision conflicts and restarts nothing.
	out = networkRequest(handler, "POST", "/admin/network/apply", map[string]any{"revision": 1}, adminToken)
	if out.Code != 409 || !strings.Contains(out.Body.String(), "reload before applying") {
		t.Fatalf("stale apply: %d %s", out.Code, out.Body.String())
	}
	if stub.applied.Load() != 1 {
		t.Fatal("a refused apply reached the launcher")
	}
	out = networkRequest(handler, "GET", "/admin/network/status", nil, adminToken)
	if out.Code != 200 || !strings.Contains(out.Body.String(), `"state":"committed"`) {
		t.Fatalf("status: %d %s", out.Code, out.Body.String())
	}
}

func TestNetworkNonManagedIsReadOnly(t *testing.T) {
	handler := newNetworkTestRouter(t, nil)
	adminToken := generateTestToken(t, "admin")

	out := networkRequest(handler, "PUT", "/admin/network", map[string]any{
		"revision":   0,
		"telnet":     map[string]any{"enabled": false, "port": 0, "public": false},
		"telnet_tls": map[string]any{"enabled": false, "port": 0, "public": false},
		"whod":       map[string]any{"enabled": false, "port": 0, "public": false},
	}, adminToken)
	if out.Code != 503 || !strings.Contains(out.Body.String(), "managed instance") {
		t.Fatalf("non-managed save: %d %s", out.Code, out.Body.String())
	}
	out = networkRequest(handler, "POST", "/admin/network/apply", map[string]any{"revision": 0}, adminToken)
	if out.Code != 503 {
		t.Fatalf("non-managed apply: %d", out.Code)
	}
	// The read works and explains itself.
	out = networkRequest(handler, "GET", "/admin/network", nil, adminToken)
	if out.Code != 200 || strings.Contains(out.Body.String(), `"managed":true`) {
		t.Fatalf("non-managed read: %d %s", out.Code, out.Body.String())
	}
	var view networkView
	if err := json.Unmarshal(out.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Managed || view.Telnet.Observed != "127.0.0.1:7777" {
		t.Fatalf("non-managed view should still show observed listeners: %+v", view)
	}
}

func TestNetworkAuditTrailRecordsTheExposureDiff(t *testing.T) {
	dir := t.TempDir()
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(dir, "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	server := httptest.NewServer(newStubNetworkLauncher())
	defer server.Close()
	control := launchctl.New(server.Listener.Addr().String(), strings.Repeat("1", 64))
	logger, err := audit.NewAuditLogger(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), logger, NewLogBuffer(10), database, nil,
		WithLaunchctl(control),
		WithNetworkObserved(func() NetworkObserved { return NetworkObserved{HTTPPort: 4350} }))
	if err != nil {
		t.Fatal(err)
	}
	adminToken := generateTestToken(t, "admin")
	out := networkRequest(handler, "PUT", "/admin/network", map[string]any{
		"revision":   2,
		"telnet":     map[string]any{"enabled": true, "port": 42069, "public": false},
		"telnet_tls": map[string]any{"enabled": false, "port": 0, "public": false},
		"whod":       map[string]any{"enabled": false, "port": 0, "public": false},
	}, adminToken)
	if out.Code != 200 {
		t.Fatalf("save: %d %s", out.Code, out.Body.String())
	}
	entries, err := settingsAuditHistory(logger)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Action == "network.save" && strings.Contains(entry.Details, "telnet") && strings.Contains(entry.Details, "42069") {
			found = true
		}
	}
	if !found {
		t.Fatal("network.save audit entry with the exposure diff is missing")
	}
}
