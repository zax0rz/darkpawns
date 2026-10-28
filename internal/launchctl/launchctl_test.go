package launchctl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withEnv(t *testing.T, env map[string]string) {
	t.Helper()
	original := getenv
	getenv = func(name string) string { return env[name] }
	t.Cleanup(func() { getenv = original })
}

func TestFromEnvironmentRejectsNonLoopbackOrMalformedCapability(t *testing.T) {
	token := strings.Repeat("1", 64)
	for _, env := range []map[string]string{
		{},
		{"DP_CONTROL_ADDR": "127.0.0.1:8080"},
		{"DP_CONTROL_TOKEN": token},
		{"DP_CONTROL_ADDR": "example.com:8080", "DP_CONTROL_TOKEN": token},
		{"DP_CONTROL_ADDR": "127.0.0.1:0", "DP_CONTROL_TOKEN": token},
		{"DP_CONTROL_ADDR": "127.0.0.1:70000", "DP_CONTROL_TOKEN": token},
		{"DP_CONTROL_ADDR": "127.0.0.1:8080", "DP_CONTROL_TOKEN": "short"},
		{"DP_CONTROL_ADDR": "127.0.0.1:8080", "DP_CONTROL_TOKEN": strings.Repeat("z", 64)},
	} {
		withEnv(t, env)
		if client := FromEnvironment(); client != nil {
			t.Fatalf("accepted malformed capability: %v", env)
		}
	}
	withEnv(t, map[string]string{"DP_CONTROL_ADDR": "127.0.0.1:8080", "DP_CONTROL_TOKEN": token})
	if client := FromEnvironment(); client == nil {
		t.Fatal("rejected a well-formed loopback capability")
	}
}

func TestClientProxiesTransactionsAndSurfacesConflicts(t *testing.T) {
	token := strings.Repeat("1", 64)
	var sawAuthorization string
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"state":"running","run_id":"abc","bundle":"/releases/v1"}`)) // #nosec G104 -- test fixture response
	})
	mux.HandleFunc("/update-status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"state":"committed","runtime_state":"running","current_version":"v1"}`)) // #nosec G104 -- test fixture response
	})
	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		sawAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"update requested"}`)) // #nosec G104 -- test fixture response
	})
	mux.HandleFunc("/restart", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"update abc needs recovery; use darkpawns update recover"}`)) // #nosec G104 -- test fixture response
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := New(server.Listener.Addr().String(), token)
	if client == nil {
		t.Fatal("stub address rejected; the client only accepts loopback and the test listener is local")
	}

	status, err := client.UpdateStatus(context.Background())
	if err != nil || status.State != "committed" {
		t.Fatalf("status: %+v %v", status, err)
	}
	if err := client.TriggerUpdate(context.Background()); err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if sawAuthorization != "Bearer "+token {
		t.Fatalf("bearer capability not sent: %q", sawAuthorization)
	}
	err = client.Restart(context.Background())
	if err == nil || !strings.Contains(err.Error(), "update recover") {
		t.Fatalf("conflict text lost: %v", err)
	}
	if status, err := client.Status(context.Background()); err != nil || !status.Managed || status.State != "running" {
		t.Fatalf("status read: %+v %v", status, err)
	}
}

func TestNilClientReportsUnavailable(t *testing.T) {
	var client *Client
	// Status doubles as the managed probe: a nil client answers "not
	// managed" without an error, which is how /admin/updates/status reports
	// an unmanaged deployment instead of failing.
	if status, err := client.Status(context.Background()); err != nil || status.Managed {
		t.Fatalf("nil client status: %+v %v", status, err)
	}
	if _, err := client.CheckUpdate(context.Background()); err != ErrUnavailable {
		t.Fatal("nil client check must be ErrUnavailable")
	}
	if err := client.TriggerUpdate(context.Background()); err != ErrUnavailable {
		t.Fatal("nil client trigger must be ErrUnavailable")
	}
	if err := client.Restart(context.Background()); err != ErrUnavailable {
		t.Fatal("nil client restart must be ErrUnavailable")
	}
}
