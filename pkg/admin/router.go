package admin

import (
	"net/http"
	"os"
	"time"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// NewRouter creates an admin HTTP handler with role-protected endpoints.
// liveSessions is the session manager (or nil to disable live session endpoints).
func NewRouter(world *game.World, auditLogger *audit.AuditLogger, logBuffer *LogBuffer, database *db.DB, liveSessions LiveSessionProvider) http.Handler {
	mux := http.NewServeMux()

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

	// Public routes (no auth required)
	mux.HandleFunc("/admin/login", wrap(handleLogin(database, loginAttempts)))

	// Static admin UI files (no auth required — SPA needs to load before login)
	adminUIDir := os.Getenv("ADMIN_UI_DIR")
	if adminUIDir == "" {
		adminUIDir = "admin-ui-dist"
	}
	if _, err := os.Stat(adminUIDir); err == nil {
		mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
		})
		mux.HandleFunc("/admin/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/favicon.svg")
		})
		mux.HandleFunc("/admin/icons.svg", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/icons.svg")
		})
		mux.HandleFunc("/admin/assets/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.StripPrefix("/admin/", http.FileServer(http.Dir(adminUIDir))).ServeHTTP(w, r)
		})
		// SPA fallback — serve index.html for any /admin/* that doesn't match an API route
		mux.HandleFunc("/admin/index.html", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, adminUIDir+"/index.html")
		})
	}

	// Authenticated routes below — require valid JWT + role
	// Health is public, everything else requires auth

	// Zones — read/write, requires builder role
	mux.HandleFunc("/admin/zones", wrap(corsMiddleware(requireRole("builder", handleZones(world)))))
	mux.HandleFunc("/admin/zones/reset", wrap(corsMiddleware(requireRole("admin", handleZoneReset(world)))))
	mux.HandleFunc("/admin/zones/", wrap(corsMiddleware(requireRole("builder", handleZoneByIDOrReset(world, auditLogger)))))

	// Server info — requires builder role
	mux.HandleFunc("/admin/server", wrap(corsMiddleware(requireRole("builder", handleServerInfo(world, auditLogger)))))

	// Server logs — requires builder role
	mux.HandleFunc("/admin/logs", wrap(corsMiddleware(requireRole("builder", handleLogs(logBuffer)))))

	// Online players — requires builder role
	mux.HandleFunc("/admin/players", wrap(corsMiddleware(requireRole("builder", handlePlayers(world)))))
	// Player detail — requires builder role for GET, admin for POST
	mux.HandleFunc("/admin/players/", wrap(corsMiddleware(requireRole("builder", handlePlayerDetail(world, auditLogger)))))

	// Mobs — read/write, requires builder role
	mux.HandleFunc("/admin/mobs", wrap(corsMiddleware(requireRole("builder", handleMobs(world)))))
	mux.HandleFunc("/admin/mobs/", wrap(corsMiddleware(requireRole("builder", handleMobByVnum(world, auditLogger)))))

	// Objects — read/write, requires builder role
	mux.HandleFunc("/admin/objects", wrap(corsMiddleware(requireRole("builder", handleObjects(world)))))
	mux.HandleFunc("/admin/objects/", wrap(corsMiddleware(requireRole("builder", handleObjectByVnum(world, auditLogger)))))

	// Shops — read/write, requires builder role
	mux.HandleFunc("/admin/shops", wrap(corsMiddleware(requireRole("builder", handleShops(world)))))
	mux.HandleFunc("/admin/shops/", wrap(corsMiddleware(requireRole("builder", handleShopByKeeper(world, auditLogger)))))

	// Rooms — read/write, requires builder role
	mux.HandleFunc("/admin/rooms/", wrap(corsMiddleware(requireRole("builder", handleRoomByVnum(world, auditLogger)))))

	// Server metrics — requires builder role
	mux.HandleFunc("/admin/metrics", wrap(corsMiddleware(requireRole("builder", handleMetrics(world)))))

	// Save world — requires admin role
	mux.HandleFunc("/admin/save-world", wrap(corsMiddleware(requireRole("admin", handleSaveWorld(world, auditLogger)))))

	// Reset all zones — requires admin role
	mux.HandleFunc("/admin/reset-all-zones", wrap(corsMiddleware(requireRole("admin", handleResetAllZones(world, auditLogger)))))

	// Agent status, findings, and triage — requires builder role
	storePath := os.Getenv("ADMIN_STORE_PATH")
	if storePath == "" {
		storePath = "data/admin_store.json"
	}
	agentStore := NewAgentStore(storePath)
	mux.HandleFunc("/admin/agents", wrap(corsMiddleware(requireRole("builder", handleAgents(agentStore)))))
	mux.HandleFunc("/admin/agents/status", wrap(corsMiddleware(requireRole("builder", handleAgentStatus(agentStore)))))
	mux.HandleFunc("/admin/findings", wrap(corsMiddleware(requireRole("builder", handleFindings(agentStore)))))
	mux.HandleFunc("/admin/findings/", wrap(corsMiddleware(requireRole("builder", handleFindingByID(agentStore)))))
	mux.HandleFunc("/admin/triage/summaries", wrap(corsMiddleware(requireRole("builder", handleTriageSummaries(agentStore)))))

	// Live agent sessions — requires builder role, shows connected game agents
	if liveSessions != nil {
		mux.HandleFunc("/admin/sessions/agents", wrap(corsMiddleware(requireRole("builder", handleLiveAgentSessions(liveSessions)))))
	}

	// Decision log — requires builder role
	if database != nil {
		mux.HandleFunc("/admin/decisions", wrap(corsMiddleware(requireRole("builder", handleDecisionLog(database)))))
	}

	// Narrative feed — requires builder role
	if database != nil {
		mux.HandleFunc("/admin/narrative", wrap(corsMiddleware(requireRole("builder", handleNarrativeFeed(database)))))
	}

	// SPA fallback — this MUST be registered last, after all API routes.
	// Catches any /admin/* path that didn't match an API route above.
	if _, err := os.Stat(adminUIDir); err == nil {
		mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, adminUIDir+"/index.html")
		})
	}

	return mux
}

// requireRole wraps a handler, rejecting requests that lack the required role.
// Claims must already be on the context (set by web.AuthMiddleware).
func requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if !claims.HasRole(role) {
			http.Error(w, `{"error":"forbidden","required":"`+role+`"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// corsMiddleware adds CORS headers for allowed origins.
// Production: set ADMIN_CORS_ORIGIN env var to the SPA origin.
// Development: localhost:5173 and localhost:4350 are allowed.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next(w, r)
			return
		}

		allowed := origin == "http://localhost:5173" || origin == "http://localhost:4350" ||
			origin == "https://localhost:5173" || origin == "https://localhost:4350"
		if envOrigin := os.Getenv("ADMIN_CORS_ORIGIN"); envOrigin != "" && origin == envOrigin {
			allowed = true
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
