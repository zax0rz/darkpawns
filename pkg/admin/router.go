package admin

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/metrics"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

// RouterOption configures NewRouter.
type RouterOption func(*routerConfig)

type routerConfig struct {
	// sharedDoc, when set, is the apidoc document the router's Huma operations
	// are registered into. main passes it so the admin operations appear in
	// the server-wide spec served at /openapi.json. Tests leave it nil and get
	// a private document per router.
	sharedDoc *apidoc.Doc
}

// WithSharedSpec registers the router's Huma operations into doc, the
// server-wide OpenAPI document, instead of a private one.
func WithSharedSpec(doc *apidoc.Doc) RouterOption {
	return func(c *routerConfig) { c.sharedDoc = doc }
}

// routerInternal is the full router construction result. NewRouter returns
// only the handler; the drift gate and spec tests use newRouter to also see
// the registered route patterns and the Huma API holding the migrated
// operations.
type routerInternal struct {
	handler http.Handler
	routes  []string
	api     huma.API
}

// NewRouter creates an admin HTTP handler with role-protected endpoints.
// liveSessions is the session manager, or nil. The OLC endpoints use the
// provider interfaces it implements (OLCReadStateProvider and the rest).
// It returns an error if the agent store cannot be initialized (DP-1016).
func NewRouter(world *game.World, auditLogger *audit.AuditLogger, logBuffer *LogBuffer, database *db.DB, liveSessions any, opts ...RouterOption) (http.Handler, error) {
	ri, err := newRouter(world, auditLogger, logBuffer, database, liveSessions, opts...)
	if ri == nil {
		return nil, err
	}
	return ri.handler, err
}

func newRouter(world *game.World, auditLogger *audit.AuditLogger, logBuffer *LogBuffer, database *db.DB, liveSessions any, opts ...RouterOption) (*routerInternal, error) {
	var cfg routerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	mux := http.NewServeMux()
	ri := &routerInternal{handler: mux}

	// track registers a pattern on the mux and records it for the route drift
	// gate: every pattern on this mux must be either a Huma operation or on
	// the commented allowlist in drift_gate_test.go.
	track := func(pattern string, hf http.HandlerFunc) {
		ri.routes = append(ri.routes, pattern)
		mux.HandleFunc(pattern, hf)
	}

	// Rate limiter for admin endpoints
	rateLimiter := auth.NewIPRateLimiter()
	wrap := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := auth.GetIPFromRequest(r)
			if !rateLimiter.GetLimiter(ip).Allow() {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}

	// Separate lockout tracker for admin login — independent from the session (telnet) tracker.
	// An IP locked out on telnet can still attempt admin login and vice versa.
	loginAttempts := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
		Threshold: 10,
		Lockout:   15 * time.Minute,
	})

	// Static admin UI files (no auth required — SPA needs to load before login)
	adminUIDir := os.Getenv("ADMIN_UI_DIR")
	if adminUIDir == "" {
		adminUIDir = "admin-ui-dist"
	}
	track("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})
	if _, err := os.Stat(adminUIDir); err == nil { // #nosec G703 -- adminUIDir is operator-set (ADMIN_UI_DIR env), not request-derived
		track("/admin/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/favicon.svg") // #nosec G703 -- constant filename under operator-set adminUIDir, not request-derived
		})
		track("/admin/icons.svg", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/icons.svg") // #nosec G703 -- constant filename under operator-set adminUIDir, not request-derived
		})
		track("/admin/assets/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.StripPrefix("/admin/", http.FileServer(http.Dir(adminUIDir))).ServeHTTP(w, r)
		})
		// SPA fallback — serve index.html for any /admin/* that doesn't match an API route
		track("/admin/index.html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/index.html") // #nosec G703 -- constant filename under operator-set adminUIDir, not request-derived
		})
	}

	// Authenticated routes below — require valid JWT + role
	// Health is public, everything else requires auth

	// Huma operations live on a private mux mounted per-path behind the same
	// rate-limit/CORS/role chain the hand-rolled handlers sat behind —
	// wrap(corsMiddleware(requireRole(...))) — so the middleware composition is
	// unchanged. Tranche 1 registered this API inside the liveSessions guard;
	// tranche 2's world reads are unconditional, so the mux and the API are
	// created here and the guard below only adds the session-dependent
	// operations.
	humaMux := http.NewServeMux()
	doc := cfg.sharedDoc
	if doc == nil {
		doc = apidoc.New()
	}
	ri.api = doc.NewInternalAPI(humaMux)
	var loginDatabase loginPlayerDB
	if database != nil {
		loginDatabase = database
	}
	registerLoginOperation(ri.api, loginDatabase, loginAttempts)
	// Public route: rate-limited exactly like the legacy login, but deliberately
	// outside the JWT/role middleware used by every authenticated operation.
	track("/admin/login", wrap(withClientIP(humaMux.ServeHTTP)))
	registerZones(ri.api, world)
	registerServerInfo(ri.api, world, auditLogger)
	registerLogs(ri.api, logBuffer)
	registerPlayers(ri.api, world)
	registerMobs(ri.api, world)
	registerObjects(ri.api, world)
	registerShops(ri.api, world)
	registerRooms(ri.api, world)
	registerWorldSearch(ri.api, world)
	registerMetrics(ri.api, world)
	registerWorldCompletion(ri.api, world, auditLogger)
	var olcState OLCReadStateProvider
	if provider, ok := liveSessions.(OLCReadStateProvider); ok {
		olcState = provider
	}
	var olcWrites OLCWriteStateProvider
	if provider, ok := liveSessions.(OLCWriteStateProvider); ok {
		olcWrites = provider
	}
	var olcPresence OLCPresenceProvider
	if provider, ok := liveSessions.(OLCPresenceProvider); ok {
		olcPresence = provider
	}
	registerOLC(ri.api, world, database, olcState, olcWrites, olcPresence, auditLogger, olc.NewDraftStore())
	registerFileEdit(ri.api, world, database, auditLogger)

	// Zones — read/write, requires builder role
	track("/admin/zones", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/zones/reset", wrap(corsMiddleware(requireRole("admin", requireMethod(http.MethodPost, withClientIP(humaMux.ServeHTTP))))))
	track("/admin/zones/{number}", wrap(corsMiddleware(requireRole("builder", requireMethod(http.MethodGet, withClientIP(humaMux.ServeHTTP))))))
	track("/admin/zones/{number}/reset", wrap(corsMiddleware(requireRole("admin", requireMethod(http.MethodPost, withClientIP(humaMux.ServeHTTP))))))

	// Server info — requires builder role. withClientIP stashes the client IP
	// for the operation's audit log, which the old handler read from the
	// request directly.
	track("/admin/server", wrap(corsMiddleware(requireRole("builder", withClientIP(humaMux.ServeHTTP)))))

	// Server logs — requires builder role
	track("/admin/logs", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// Online players — requires builder role
	track("/admin/players", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	// Player detail — requires builder role for GET, admin for POST
	track("/admin/players/{name}", wrap(corsMiddleware(requireRole("builder", withClientIP(humaMux.ServeHTTP)))))
	track("/admin/players/{name}/save", wrap(corsMiddleware(requireRole("admin", withClientIP(humaMux.ServeHTTP)))))
	track("/admin/players/{name}/kick", wrap(corsMiddleware(requireRole("admin", withClientIP(humaMux.ServeHTTP)))))

	// Mobs — read/write, requires builder role
	track("/admin/mobs", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/mobs/{vnum}", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// Objects — read/write, requires builder role
	track("/admin/objects", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/objects/{vnum}", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// Shops — read-only, requires builder role
	track("/admin/shops", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/shops/{keeper}", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// Rooms — read/write, requires builder role
	track("/admin/rooms/{vnum}", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/search", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// Server metrics — requires builder role
	track("/admin/metrics", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// OLC read surface — authorization is operation middleware so it can use
	// Huma's parsed {kind}/{vnum} parameters. These exact mounts keep the
	// route drift gate at zero new allowlist entries.
	track("/admin/olc/schema/{kind}", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/{kind}/{vnum}/preview", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/held", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/pending", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/zones/{zone}/vnums", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/lookup/{kind}/{vnum}/name", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/{kind}/{vnum}", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/{kind}/{vnum}/draft", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/{kind}/{vnum}/draft/commit", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/zones/{zone}", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/zones/{zone}/save", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/room/{vnum}", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/room/{vnum}/draft", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/olc/room/{vnum}/draft/commit", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/files/{root}/listing", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/files/{root}/content", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/files/lua/usage", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))
	track("/admin/files/lua/resolve", wrap(corsMiddleware(withClientIP(humaMux.ServeHTTP))))

	// The Prometheus endpoint, moved here from an unauthenticated /metrics on
	// the root mux. It was public on darkpawns.org and nobody noticed, because
	// every gauge read zero until the collectors were wired: publishing real
	// player counts, command volume and command_duration_seconds to anyone who
	// asks should be a decision, not a side effect of fixing them.
	//
	// "builder" matches the sibling /admin/metrics above and the rest of the
	// read-only console: any immortal who can sign in can read it. A Prometheus
	// scraper cannot present a JWT, so if one is ever wanted the usual answer is
	// a second listener bound to localhost, not loosening this.
	track("/admin/prometheus", wrap(corsMiddleware(requireRole("builder", metrics.Handler().ServeHTTP))))

	// Save world — requires admin role
	track("/admin/save-world", wrap(corsMiddleware(requireRole("admin", withClientIP(humaMux.ServeHTTP)))))

	// Reset all zones — requires admin role
	track("/admin/reset-all-zones", wrap(corsMiddleware(requireRole("admin", withClientIP(humaMux.ServeHTTP)))))

	// Agent status, findings, and triage — requires builder role
	storePath := os.Getenv("ADMIN_STORE_PATH")
	if storePath == "" {
		storePath = "data/admin_store.json"
	}
	agentStore, err := NewAgentStore(storePath)
	if err != nil {
		return nil, fmt.Errorf("init agent store: %w", err)
	}
	registerAgentStoreCompletion(ri.api, agentStore)
	track("/admin/agents", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/agents/status", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/findings", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/findings/{id}", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))
	track("/admin/triage/summaries", wrap(corsMiddleware(requireRole("builder", humaMux.ServeHTTP))))

	// The handlers these routes replace answered wrong methods with a JSON
	// 405; the Huma mux answers plain-text 405 on its own. Keep the old body
	// for methods the operations do not define — the bare pattern loses to the
	// method-qualified Huma pattern on Go's ServeMux, so defined methods still
	// reach the operation.
	for _, p := range []string{
		"/admin/zones",
		"/admin/server",
		"/admin/logs",
		"/admin/players",
		"/admin/mobs",
		"/admin/mobs/{vnum}",
		"/admin/objects",
		"/admin/objects/{vnum}",
		"/admin/shops",
		"/admin/rooms/{vnum}",
		"/admin/metrics",
		"/admin/login",
		"/admin/save-world",
		"/admin/reset-all-zones",
		"/admin/agents",
		"/admin/agents/status",
		"/admin/findings",
		"/admin/triage/summaries",
	} {
		humaMux.HandleFunc(p, methodNotAllowedJSON)
	}
	// Parameterized routes keep their legacy fallback parsers for undefined
	// methods. Besides preserving the JSON 405, this retains the historical
	// path parsing and validation order. Method-qualified Huma patterns still win for the
	// operations documented in OpenAPI.
	humaMux.HandleFunc("/admin/players/{name}", handlePlayerDetail(world, auditLogger))
	humaMux.HandleFunc("/admin/players/{name}/save", handlePlayerDetail(world, auditLogger))
	humaMux.HandleFunc("/admin/players/{name}/kick", handlePlayerDetail(world, auditLogger))
	humaMux.HandleFunc("/admin/shops/{keeper}", handleShopByKeeper(world, auditLogger))
	humaMux.HandleFunc("/admin/findings/{id}", handleFindingByID(agentStore))

	// SPA fallback — this MUST be registered last, after all API routes.
	// Catches any /admin/* path that didn't match an API route above.
	// Registered whether or not the console is built: when the build is absent
	// the fallback answers with the build commands instead of Go's bare 404,
	// so a fresh checkout says what to run rather than looking broken.
	track("/admin/", func(w http.ResponseWriter, r *http.Request) {
		index := adminUIDir + "/index.html"
		if _, err := os.Stat(index); err != nil { // #nosec G703 -- constant filename under operator-set adminUIDir, not request-derived
			serveConsoleNotBuilt(w, index)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, index) // #nosec G703 -- constant filename under operator-set adminUIDir, not request-derived
	})

	return ri, nil
}

// methodNotAllowedJSON answers 405 with the JSON body the pre-Huma handlers
// returned for undefined methods. Registered on the Huma mux without a
// method, so Go's ServeMux precedence sends defined methods to the Huma
// operation and everything else here.
func methodNotAllowedJSON(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
}

func requireMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			methodNotAllowedJSON(w, r)
			return
		}
		next(w, r)
	}
}

// serveConsoleNotBuilt answers /admin/ when the console build is absent. A
// fresh checkout has no admin-ui-dist — the console is a separate npm build,
// not something `go build` produces — and Go's default 404 says nothing about
// the step that creates it. The commands mirror DEPLOYMENT.md ("Admin console").
func serveConsoleNotBuilt(w http.ResponseWriter, index string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, consoleNotBuiltPage, html.EscapeString(index))
}

const consoleNotBuiltPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Dark Pawns admin console — not built</title>
</head>
<body>
<h1>Admin console not built</h1>
<p>This server has no admin console at <code>%s</code>.
The console is a separate frontend build, not part of <code>go build</code>.</p>
<p>From the repository root:</p>
<pre>npm --prefix admin-ui ci
npm --prefix admin-ui run build
rm -rf lib/admin-ui-dist
mkdir -p lib/admin-ui-dist
cp -R admin-ui/dist/. lib/admin-ui-dist/</pre>
<p>Then restart the server and reload this page.
See DEPLOYMENT.md, &ldquo;Admin console&rdquo;, for details &mdash;
including <code>ADMIN_UI_DIR</code> if the console lives somewhere else.</p>
</body>
</html>
`

// requireRole wraps a handler, rejecting requests that lack the required role.
// It first ensures a valid JWT is present (parsing the Authorization header if
// claims aren't already on the context), then enforces the role. This makes
// the router self-protecting: even if an outer wrapper forgets to install
// web.AuthMiddleware, protected routes still require a valid bearer token
// (DP-855). The double validation is harmless — already-set claims skip the
// parse — and matches what pkg/admin/handlers_test.go's authMiddlewareForTest
// has always simulated.
func requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaimsFromContext(r.Context())
		if !ok {
			// No outer middleware injected claims — parse the bearer token
			// ourselves. This is the production path today; outer wrapping
			// is still welcomed as defense-in-depth.
			parsed, err := claimsFromAuthorization(r)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims = parsed
			// Re-store on context so downstream handlers and requireRole's
			// outer-wrapped siblings see the same claims.
			r = r.WithContext(auth.SetClaimsOnContext(r.Context(), claims))
		}
		if !claims.HasRole(role) {
			http.Error(w, `{"error":"forbidden","required":"`+role+`"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// claimsFromAuthorization extracts and validates a Bearer JWT from the
// Authorization header. Returns an error on any failure (missing header,
// wrong scheme, invalid/expired/wrong-issuer token).
func claimsFromAuthorization(r *http.Request) (*auth.Claims, error) {
	return claimsFromBearerHeader(r.Header.Get("Authorization"))
}

func claimsFromBearerHeader(authHeader string) (*auth.Claims, error) {
	if authHeader == "" {
		return nil, errNoBearerToken
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errNoBearerToken
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	return auth.ValidateJWT(token)
}

// errNoBearerToken is returned by claimsFromAuthorization when the request
// has no usable Bearer token. Callers map this to a 401.
var errNoBearerToken = errors.New("missing or malformed Authorization header")

// corsMiddleware adds CORS headers for allowed origins.
// Production: set ADMIN_CORS_ORIGIN env var to the SPA origin.
// Development: localhost:5173 and localhost:4350 are allowed only when
// ENVIRONMENT=development, matching the web CORS behavior.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next(w, r)
			return
		}

		allowed := false
		if envOrigin := os.Getenv("ADMIN_CORS_ORIGIN"); envOrigin != "" && origin == envOrigin {
			allowed = true
		}
		if !allowed && os.Getenv("ENVIRONMENT") == "development" {
			allowed = origin == "http://localhost:5173" || origin == "http://localhost:4350" ||
				origin == "https://localhost:5173" || origin == "https://localhost:4350"
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// serveAdminUI serves the React SPA from the admin-ui-dist directory.
// For any path that doesn't match an API route or a static file, it serves
// index.html (SPA fallback for client-side routing).
