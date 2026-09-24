package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// rootAllowlist is the set of root-mux patterns that are not Huma operations
// after tranche 1. Method-qualified patterns (e.g. "GET /health") are never
// allowlisted: they must come from the generated spec, so a hand-rolled
// method route cannot slip past the gate. Entries shrink only as later
// tranches migrate them; anything new here needs a comment.
var rootAllowlist = []struct {
	path   string
	reason string
}{
	{"/ws", "WebSocket upgrade endpoint; not a typed REST operation — whether and how to model it is a later decision"},
	{"/darkpawns-map.xml", "Mudlet world map (XML file Mudlet imports, fetched via GMCP Client.Map); a file download, not a REST operation"},
	{"/", "front door: static site, browser client, or plain-text index"},
	{"/api/contact", "contact form handler (env-gated)"},
	{"/api/", "JWT-protected API catch-all; 404s with the consult-openapi advice"},
	{"/api/openapi.json", "spec compatibility alias, served by apidoc.Handler from the generated document"},
	{"/admin/health", "unauthenticated admin liveness JSON"},
	{"/admin/", "admin console mount: admin mux plus SPA fallback"},
	{"/assets/", "fingerprinted admin UI static assets"},
}

// TestAPIRouteDriftGate boots the server in a subprocess (the same helper
// pattern as main_test.go) with the DP_DUMP_API_ROUTES test hook set, reads
// the route table main registered on the root mux plus the Huma operations
// from the generated spec, and fails if any root pattern is neither a Huma
// operation nor on the commented allowlist above.
func TestAPIRouteDriftGate(t *testing.T) {
	worldDir := parseableWorld(t)
	// The fixture root holds lib/world; boot from there so the server anchors
	// its runtime state (data/, logs/) in the temp dir, not the checkout.
	root := filepath.Dir(filepath.Dir(worldDir))
	t.Chdir(root)

	code, out := bootServerContext(t,
		[]string{"-world", worldDir},
		60*time.Second,
		"ENVIRONMENT=development",
		"JWT_SECRET=test-secret-that-is-at-least-32-chars-long-for-hs256",
		"DATABASE_URL=",
		"DP_DUMP_API_ROUTES=1",
	)
	if code != 0 {
		t.Fatalf("boot exit code = %d, want 0 (route dump)\n%s", code, out)
	}

	var dump struct {
		Root []string `json:"root"`
		Ops  []string `json:"ops"`
	}
	found := false
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, dumpRoutesPrefix) {
			continue
		}
		if found {
			t.Fatalf("multiple route dumps in output\n%s", out)
		}
		found = true
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, dumpRoutesPrefix)), &dump); err != nil {
			t.Fatalf("unmarshal route dump: %v\nline: %s", err, line)
		}
	}
	if !found {
		t.Fatalf("no route dump line in boot output\n%s", out)
	}

	opSet := map[string]bool{}
	for _, o := range dump.Ops {
		opSet[o] = true
	}

	allow := map[string]string{}
	for _, a := range rootAllowlist {
		allow[a.path] = a.reason
	}

	for _, pattern := range dump.Root {
		method, path, methodOK := splitMethodPattern(pattern)
		if methodOK {
			if !opSet[pattern] {
				t.Errorf("root route %q is a hand-rolled %s route, not a Huma operation — migrate it or allowlist it with a comment", pattern, method)
			}
			_ = path
			continue
		}
		if _, ok := allow[pattern]; !ok {
			t.Errorf("root route %q is neither a Huma operation nor on the root allowlist", pattern)
		}
	}

	// The migrated operations must be in the generated spec: if one vanished,
	// the spec no longer describes the running server.
	for _, want := range []string{
		"GET /health",
		"GET /admin/zones",
		"GET /admin/server",
		"GET /admin/logs",
		"GET /admin/players",
		"GET /admin/mobs",
		"GET /admin/mobs/{vnum}",
		"GET /admin/objects",
		"GET /admin/objects/{vnum}",
		"GET /admin/shops",
		"GET /admin/rooms/{vnum}",
		"GET /admin/metrics",
	} {
		if !opSet[want] {
			t.Errorf("generated spec missing operation %s; ops: %v", want, dump.Ops)
		}
	}

	// Allowlist entries must still be registered; a stale entry means the
	// route moved to Huma (shrink the list) or was deleted (delete the entry).
	registered := map[string]bool{}
	for _, pattern := range dump.Root {
		registered[pattern] = true
	}
	for path, reason := range allow {
		bare := path
		if !registered[bare] {
			t.Errorf("root allowlist entry %q (%s) is not registered on the root mux — the entry is stale", path, reason)
		}
	}
}

// splitMethodPattern splits a Go 1.22 method pattern ("GET /health") into its
// method and path. Bare patterns ("/ws") return methodOK=false.
func splitMethodPattern(pattern string) (method, path string, methodOK bool) {
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		return "", pattern, false
	}
	for _, r := range method {
		if r < 'A' || r > 'Z' {
			return "", pattern, false
		}
	}
	return method, path, true
}
