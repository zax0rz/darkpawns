package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
)

// TestGeneratedOpenAPISpec pins the contract for the generated document: it
// parses, is OpenAPI 3.1, is served at both public URLs with identical
// bytes, and documents exactly the migrated operations — nothing missing,
// nothing extra. Tranche 1 migrated /health, /admin/research/capture and
// /admin/sessions/agents; tranche 2 adds the eleven builder world reads.
// /ws and /onboarding are deliberately absent (a WebSocket upgrade and a
// browser entry page are not typed REST operations; modeling them is a later
// decision), as are the routes still on the plain mux per the drift gate's
// allowlist. The webOLC read and lifecycle operations below are typed REST.
func TestGeneratedOpenAPISpec(t *testing.T) {
	setJWTSecret(t)

	doc := apidoc.New()
	root := http.NewServeMux()
	api := doc.NewAPI(root)
	apidoc.RegisterHealth(api)
	root.HandleFunc("/api/openapi.json", doc.Handler())

	ri, err := newRouter(testWorld(t), nil, NewLogBuffer(10), nil, &fakeCaptureProvider{available: true}, WithSharedSpec(doc))
	if err != nil {
		t.Fatalf("newRouter: %v", err)
	}
	root.Handle("/admin/", ri.handler)

	wantPaths := map[string][]string{
		"/health":                               {"get"},
		"/admin/login":                          {"post"},
		"/admin/decisions":                      {"get"},
		"/admin/narrative":                      {"get"},
		"/admin/research/capture":               {"get", "post"},
		"/admin/sessions/agents":                {"get"},
		"/admin/zones":                          {"get"},
		"/admin/server":                         {"get"},
		"/admin/logs":                           {"get"},
		"/admin/players":                        {"get"},
		"/admin/mobs":                           {"get"},
		"/admin/mobs/{vnum}":                    {"get"},
		"/admin/objects":                        {"get"},
		"/admin/objects/{vnum}":                 {"get"},
		"/admin/shops":                          {"get"},
		"/admin/rooms/{vnum}":                   {"get"},
		"/admin/metrics":                        {"get"},
		"/admin/zones/{number}":                 {"get"},
		"/admin/zones/reset":                    {"post"},
		"/admin/zones/{number}/reset":           {"post"},
		"/admin/players/{name}":                 {"get"},
		"/admin/players/{name}/save":            {"post"},
		"/admin/players/{name}/kick":            {"post"},
		"/admin/shops/{keeper}":                 {"get"},
		"/admin/save-world":                     {"post"},
		"/admin/reset-all-zones":                {"post"},
		"/admin/agents":                         {"get"},
		"/admin/agents/status":                  {"post"},
		"/admin/findings":                       {"get", "post"},
		"/admin/findings/{id}":                  {"put"},
		"/admin/triage/summaries":               {"get", "post"},
		"/admin/olc/schema/{kind}":              {"get"},
		"/admin/olc/{kind}/{vnum}/preview":      {"get"},
		"/admin/olc/held":                       {"get"},
		"/admin/olc/pending":                    {"get"},
		"/admin/olc/zones/{zone}/vnums":         {"get"},
		"/admin/olc/lookup/{kind}/{vnum}/name":  {"get"},
		"/admin/olc/{kind}/{vnum}":              {"post"},
		"/admin/olc/{kind}/{vnum}/draft":        {"get", "patch", "delete"},
		"/admin/olc/{kind}/{vnum}/draft/commit": {"post"},
		"/admin/olc/zones/{zone}/save":          {"post"},
		"/admin/olc/zones/{zone}":               {"post"},
		"/admin/olc/room/{vnum}":                {"post"},
		"/admin/olc/room/{vnum}/draft":          {"get", "patch", "delete"},
		"/admin/olc/room/{vnum}/draft/commit":   {"post"},
	}

	var first []byte
	for _, url := range []string{"/openapi.json", "/api/openapi.json"} {
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200; body: %s", url, rec.Code, rec.Body.String())
		}
		body := rec.Body.Bytes()
		if first == nil {
			first = body
		} else if !bytes.Equal(first, body) {
			t.Errorf("GET %s body differs from /openapi.json", url)
		}

		var spec struct {
			OpenAPI string                                `json:"openapi"`
			Info    struct{ Title, Version string }       `json:"info"`
			Paths   map[string]map[string]json.RawMessage `json:"paths"`
		}
		if err := json.Unmarshal(body, &spec); err != nil {
			t.Fatalf("GET %s does not parse as JSON: %v", url, err)
		}
		if spec.OpenAPI != "3.1.0" {
			t.Errorf("openapi version = %q, want 3.1.0", spec.OpenAPI)
		}
		if spec.Info.Title == "" || spec.Info.Version == "" {
			t.Errorf("spec info missing title or version: %+v", spec.Info)
		}

		if len(spec.Paths) != len(wantPaths) {
			got := make([]string, 0, len(spec.Paths))
			for p := range spec.Paths {
				got = append(got, p)
			}
			t.Errorf("spec paths = %v, want exactly the migrated operations %v", got, wantPaths)
		}
		for p, methods := range wantPaths {
			item, ok := spec.Paths[p]
			if !ok {
				t.Errorf("spec missing path %q", p)
				continue
			}
			seen := map[string]bool{}
			for m := range item {
				seen[m] = true
			}
			for _, m := range methods {
				if !seen[m] {
					t.Errorf("spec %s missing %s operation", p, m)
				}
				delete(seen, m)
			}
			for extra := range seen {
				t.Errorf("spec %s has unexpected %s operation", p, extra)
			}
		}
	}
}
