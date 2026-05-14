package admin

import (
	"net/http"
	"os"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// NewRouter creates an admin HTTP handler with role-protected endpoints.
func NewRouter(world *game.World, auditLogger *audit.AuditLogger, logBuffer *LogBuffer, database *db.DB) http.Handler {
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

	// Login (unauthenticated — no auth required)
	mux.HandleFunc("/admin/login", wrap(handleLogin(database)))

	// Health is registered OUTSIDE AuthMiddleware in main.go (DP-82 fix)

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
	agentStore := NewAgentStore()
	mux.HandleFunc("/admin/agents", wrap(corsMiddleware(requireRole("builder", handleAgents(agentStore)))))
	mux.HandleFunc("/admin/agents/status", wrap(corsMiddleware(requireRole("builder", handleAgentStatus(agentStore)))))
	mux.HandleFunc("/admin/findings", wrap(corsMiddleware(requireRole("builder", handleFindings(agentStore)))))
	mux.HandleFunc("/admin/findings/", wrap(corsMiddleware(requireRole("builder", handleFindingByID(agentStore)))))
	mux.HandleFunc("/admin/triage/summaries", wrap(corsMiddleware(requireRole("builder", handleTriageSummaries(agentStore)))))

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

		allowed := false
		if origin == "http://localhost:5173" || origin == "http://localhost:4350" ||
			origin == "https://localhost:5173" || origin == "https://localhost:4350" {
			allowed = true
		}
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
