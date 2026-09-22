package admin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// unmigratedAdminRoutes is the allowlist of admin-mux routes that have not
// been migrated to Huma operations yet — tranche 2 leaves 21 entries. It
// must only SHRINK: landing a migration removes its entry, and the gate fails
// if an entry is stale, so nothing here rots.
var unmigratedAdminRoutes = []struct {
	path   string
	reason string
}{
	{"/admin", "SPA redirect to /admin/"},
	{"/admin/favicon.svg", "static console asset"},
	{"/admin/icons.svg", "static console asset"},
	{"/admin/assets/", "static console assets"},
	{"/admin/index.html", "SPA entry point"},
	{"/admin/prometheus", "prometheus exposition; text format, permanently not a typed JSON operation"},
	{"/admin/", "SPA fallback for client-side routes"},
}

// TestAdminRouteDriftGate enumerates every route registered on the admin mux
// and fails if any route is neither a Huma operation nor on the commented
// allowlist above. A route added to the plain mux in a future PR must either
// be a Huma operation or be allowlisted with a reason — no new route can
// bypass Huma unnoticed.
func TestAdminRouteDriftGate(t *testing.T) {
	setJWTSecret(t)

	// Force every conditional registration on, so the enumeration is the full
	// route table: the console-static routes need an ADMIN_UI_DIR containing
	// the files, and /admin/decisions + /admin/narrative need a database.
	uiDir := t.TempDir()
	for _, f := range []string{"favicon.svg", "icons.svg", "index.html"} {
		if err := os.WriteFile(filepath.Join(uiDir, f), []byte("x"), 0o600); err != nil {
			t.Fatalf("write console fixture %s: %v", f, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(uiDir, "assets"), 0o700); err != nil {
		t.Fatalf("write assets fixture: %v", err)
	}
	t.Setenv("ADMIN_UI_DIR", uiDir)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin_store.json"))

	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "drift-gate.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()

	ri, err := newRouter(testWorld(t), nil, NewLogBuffer(10), database, &fakeCaptureProvider{available: true})
	if err != nil {
		t.Fatalf("newRouter: %v", err)
	}

	// The paths documented by the generated spec — the migrated operations.
	humaPaths := map[string]bool{}
	for p := range ri.api.OpenAPI().Paths {
		humaPaths[p] = true
	}

	allow := map[string]string{}
	for _, a := range unmigratedAdminRoutes {
		allow[a.path] = a.reason
	}

	registered := map[string]bool{}
	for _, r := range ri.routes {
		registered[r] = true
		if humaPaths[r] {
			continue // a Huma operation — migrated, no allowlist entry expected
		}
		if _, ok := allow[r]; !ok {
			t.Errorf("route %q is neither a Huma operation nor on the unmigrated allowlist — migrate it to Huma or allowlist it with a comment", r)
		}
	}

	for path, reason := range allow {
		if humaPaths[path] {
			t.Errorf("allowlist entry %q (%s) is now a Huma operation — remove it from the allowlist; the list must only shrink", path, reason)
		}
		if !registered[path] {
			t.Errorf("allowlist entry %q (%s) is not registered on the admin mux — the entry is stale", path, reason)
		}
	}

	// Every Huma operation must actually be mounted on the admin mux; an
	// operation registered but not mounted would be missing from the running
	// server while the spec advertises it.
	for p := range humaPaths {
		if !registered[p] {
			t.Errorf("Huma operation %q is not registered on the admin mux — mount it", p)
		}
	}
}

// TestAdminRouteDriftGate_NilProviderPreservesGuard pins tranche-1 semantics:
// with a nil live-session provider the two migrated routes must not exist at
// all — same as the pre-Huma guard.
func TestAdminRouteDriftGate_NilProviderPreservesGuard(t *testing.T) {
	setJWTSecret(t)
	ri, err := newRouter(testWorld(t), nil, NewLogBuffer(10), nil, nil)
	if err != nil {
		t.Fatalf("newRouter: %v", err)
	}
	for _, r := range ri.routes {
		if r == "/admin/research/capture" || r == "/admin/sessions/agents" {
			t.Errorf("route %q registered with a nil live-session provider; the guard must keep it unregistered", r)
		}
	}
}
