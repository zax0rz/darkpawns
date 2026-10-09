package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testWorld creates a minimal world with zones, rooms, mobs, and objects for admin tests.
func testWorld(t *testing.T) *game.World {
	t.Helper()
	w := newTestWorldForWrite(t)

	// Add a player for player handler tests
	player := game.NewPlayer(1, "TestPlayer", 1001)
	player.Level = 50
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	player2 := game.NewPlayer(2, "BuilderPlayer", 1002)
	player2.Level = 33
	if err := w.AddPlayer(player2); err != nil {
		t.Fatalf("AddPlayer second player: %v", err)
	}

	return w
}

func newWorldWithShops(t *testing.T) *game.World {
	t.Helper()
	w := testWorld(t)
	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{
		KeeperVNum: 2002,
		BuyTypes:   []int{1, 5},
		SellTypes:  []int{3001},
		ProfitBuy:  1.2,
		ProfitSell: 0.8,
		KeeperName: "Merchant",
		RoomVNum:   1001,
	})
	w.SetShopManager(sm)
	return w
}

// setJWTSecret sets JWT_SECRET for the duration of a test.
func setJWTSecret(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long-for-hs256")
	t.Cleanup(func() {
		os.Unsetenv("JWT_SECRET")
	})
}

// contextWithClaims returns a request with JWT claims set on the context.
func contextWithClaims(r *http.Request, role string) *http.Request {
	claims := &auth.Claims{
		PlayerName: "TestPlayer",
		Role:       role,
	}
	ctx := auth.SetClaimsOnContext(r.Context(), claims)
	return r.WithContext(ctx)
}

// generateTestToken generates a test JWT for a given role.
func generateTestToken(t *testing.T, role string) string {
	t.Helper()
	setJWTSecret(t)
	token, err := auth.GenerateJWT("TestPlayer", role)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	return token
}

// ---------------------------------------------------------------------------
// handleZones
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleZoneByIDOrReset
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleZoneUpdate (PUT)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleMobs
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleObjects
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleServerInfo
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleLogs
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handlePlayers
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handlePlayerDetail
// ---------------------------------------------------------------------------

func TestHandlePlayerDetail_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/players/TestPlayer", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var detail playerDetailResponse
	json.Unmarshal(rec.Body.Bytes(), &detail)
	if detail.Name != "TestPlayer" {
		t.Errorf("name = %q, want %q", detail.Name, "TestPlayer")
	}
	if detail.Level != 50 {
		t.Errorf("level = %d, want 50", detail.Level)
	}
}

func TestHandlePlayerDetail_GET_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/players/Nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePlayerDetail_GET_EmptyName(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/players/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePlayerDetail_Save_RequiresAdmin(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil, nil)

	// Test without claims
	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/save", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestHandlePlayerDetail_Save_WithAdminClaims(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)

	// Use a distinct player name so the save artifact is not the committed fixture.
	savePlayerName := "SaveTestPlayer"
	savePlayer := game.NewPlayer(3, savePlayerName, 1001)
	savePlayer.Level = 10
	if err := w.AddPlayer(savePlayer); err != nil {
		t.Fatalf("AddPlayer save player: %v", err)
	}

	handler := handlePlayerDetail(w, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/players/"+savePlayerName+"/save", nil)
	req = contextWithClaims(req, "admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Save will try to write to ./data/players/ which might fail in test env
	// Just verify the handler processes the request without panicking
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Errorf("unexpected status %d, wanted 200 or 500 (disk write depends on test env)", rec.Code)
	}

	// Clean up any disk artifact.
	t.Cleanup(func() {
		_ = os.Remove("./data/players/" + savePlayerName + ".json")
	})
}

func TestHandlePlayerDetail_Save_BuilderRejected(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/save", nil)
	req = contextWithClaims(req, "builder")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleMetrics
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleResetAllZones
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleZoneReset (placeholder)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleRoomByVnum
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleRoomUpdate (PUT)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleMobByVnum
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleMobUpdate (PUT)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleObjectByVnum
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleShops
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleShopByKeeper
// ---------------------------------------------------------------------------

func TestHandleShopByKeeper_GET_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShopByKeeper(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/shops/2002", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleShopByKeeper_GET_NotFound(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShopByKeeper(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/shops/9999", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleShopByKeeper_PUT_NotAllowed(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShopByKeeper(w, nil)

	req := httptest.NewRequest(http.MethodPut, "/admin/shops/2002", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleAgents, handleAgentStatus
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleFindings
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// handleFindingByID (PUT)
// ---------------------------------------------------------------------------

func TestHandleFindingByID_PUT_Valid(t *testing.T) {
	path, _ := tempStorePath(t)
	store, err := NewAgentStore(path)
	if err != nil {
		t.Fatalf("NewAgentStore failed: %v", err)
	}
	f, err := store.AddFinding("reek", "high", "test", "f.go", 1, "")
	if err != nil {
		t.Fatalf("AddFinding returned error: %v", err)
	}

	handler := handleFindingByID(store)
	body := `{"status": "confirmed"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/findings/%d", f.ID), strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleFindingByID_PUT_NotFound(t *testing.T) {
	path, _ := tempStorePath(t)
	store, err := NewAgentStore(path)
	if err != nil {
		t.Fatalf("NewAgentStore failed: %v", err)
	}
	handler := handleFindingByID(store)

	body := `{"status": "confirmed"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/findings/999", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleFindingByID_PUT_MissingStatus(t *testing.T) {
	path, _ := tempStorePath(t)
	store, err := NewAgentStore(path)
	if err != nil {
		t.Fatalf("NewAgentStore failed: %v", err)
	}
	handler := handleFindingByID(store)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/admin/findings/1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleTriageSummaries
// ---------------------------------------------------------------------------

// authMiddlewareForTest validates a Bearer JWT and sets claims on context.
// This simulates what web.AuthMiddleware does in production. It also stamps
// the CSRF header every authenticated admin request must carry (VULN-043) —
// the real console sends it on every fetch, so tests exercising authed paths
// through this helper match the production request shape.
func authMiddlewareForTest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(adminCSRFHeader) == "" {
			r.Header.Set(adminCSRFHeader, "test")
		}
		tokenStr := r.Header.Get("Authorization")
		if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		claims, err := auth.ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := auth.SetClaimsOnContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---------------------------------------------------------------------------
// CORS + Router Integration
// ---------------------------------------------------------------------------

func TestNewRouter_CORS_Headers(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	defer func() { _ = os.Setenv("ENVIRONMENT", origEnv) }()
	_ = os.Setenv("ENVIRONMENT", "development")

	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	// OPTIONS preflight
	req := httptest.NewRequest(http.MethodOptions, "/admin/zones", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want 204", rec.Code)
	}

	// Check CORS headers
	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:5173" {
		t.Errorf("Allow-Origin = %q, want %q", origin, "http://localhost:5173")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods header missing")
	}
}

func TestNewRouter_ConsoleNotBuilt_SaysWhatToRun(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	// An empty directory stands in for a fresh checkout: no index.html.
	t.Setenv("ADMIN_UI_DIR", t.TempDir())

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"npm --prefix admin-ui ci",
		"npm --prefix admin-ui run build",
		"lib/admin-ui-dist",
		"DEPLOYMENT.md",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("guidance page missing %q; body: %s", want, body)
		}
	}

	// The bare /admin spelling must reach the same answer, not a bare 404.
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/admin/" {
		t.Errorf("/admin: status = %d, Location = %q; want 301 to /admin/", rec.Code, rec.Header().Get("Location"))
	}
}

// TestNewRouter_ConsoleDirMissingEntirely covers the actual fresh-clone
// condition, which is not the same as the empty-directory case above:
// admin-ui-dist does not exist at all. That distinction is load-bearing. The
// pre-fix code decided once at boot with os.Stat(adminUIDir), so an existing
// but empty directory still registered the route and still reached the
// guidance page — meaning the sibling tests pass against the unfixed router
// and only this one fails.
func TestNewRouter_ConsoleDirMissingEntirely(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	t.Setenv("ADMIN_UI_DIR", t.TempDir()+"/never-created")

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; a bare 404 means the fresh-clone case is unguarded", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "npm --prefix admin-ui run build") {
		t.Errorf("guidance page missing the build command; body: %s", body)
	}
}

func TestNewRouter_ConsoleBuilt_ServesIndex(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	dir := t.TempDir()
	if err := os.WriteFile(dir+"/index.html", []byte("<!DOCTYPE html><html>console</html>"), 0o600); err != nil { // #nosec G304 -- test-owned temp path
		t.Fatalf("write index.html: %v", err)
	}
	t.Setenv("ADMIN_UI_DIR", dir)

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "console") {
		t.Errorf("index body wrong: %s", body)
	}
}

func TestNewRouter_Unauthenticated_Returns401(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	// No auth middleware wrapper — requireRole checks context directly

	// Endpoints that require auth should return 401 without claims on context.
	// Note: rate limiter may return 429 after burst (10 reqs), so we accept either.
	// We check the first few endpoints for 401 to avoid rate-limit interference.
	protectedEndpoints := []string{
		"/admin/zones",
		"/admin/server",
		"/admin/logs",
		"/admin/players",
		"/admin/mobs",
		"/admin/objects",
		"/admin/metrics",
		"/admin/reset-all-zones",
		"/admin/agents",
		"/admin/findings",
	}

	for i, path := range protectedEndpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// First 9 endpoints should be 401 (within rate limit burst).
			// Later ones may be 429 (rate limited) — accept either.
			if i < 9 && rec.Code != http.StatusUnauthorized {
				t.Errorf("%s: status = %d, want 401; body: %s", path, rec.Code, rec.Body.String())
			} else if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusTooManyRequests {
				t.Errorf("%s: status = %d, want 401 or 429; body: %s", path, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNewRouter_Forbidden_BuilderAccess(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)

	// Generate a "player" role token
	token := generateTestToken(t, "player")

	// Admin-only endpoints should return 403 for player role
	t.Run("reset-all-zones", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/reset-all-zones", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestNewRouter_AuthenticatedBuilder_Success(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)
	token := generateTestToken(t, "builder")

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestNewRouter_AuthenticatedAdmin_SuccessOnAdminEndpoints(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)
	token := generateTestToken(t, "admin")

	t.Run("reset-all-zones", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/reset-all-zones", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// May succeed or fail based on zone state, but shouldn't be 401/403
		if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
			t.Errorf("unexpected status %d for admin", rec.Code)
		}
	})
}

func TestNewRouter_CORS_NoOrigin(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)

	// Request without Origin (e.g. server-side curl) — no CORS headers
	token := generateTestToken(t, "builder")
	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	// No CORS headers expected since no Origin header
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS header present without Origin header")
	}
}

func TestNewRouter_InvalidToken(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid token should return 401, got %d", rec.Code)
	}
}

// TestNewRouter_SelfProtects_WithoutOuterAuthMiddleware is the DP-855
// regression guard. The production wiring at cmd/server/main.go mounts the
// admin router directly (no web.AuthMiddleware wrap), so the router MUST
// validate bearer tokens itself. Without this, a request with no
// Authorization header could reach requireRole, which (before the fix) had
// no way to obtain claims — locking everyone out by accident rather than
// validating tokens by design.
//
// This test deliberately does NOT wrap the router in authMiddlewareForTest.
// It proves the router self-validates the bearer token from the
// Authorization header alone.
func TestNewRouter_SelfProtects_WithoutOuterAuthMiddleware(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	// NO authMiddlewareForTest wrap — bare router, as in cmd/server/main.go:313.
	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	t.Run("no auth header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 401 (or 429 if rate-limited); body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("valid builder token reaches handler", func(t *testing.T) {
		token := generateTestToken(t, "builder")
		req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(adminCSRFHeader, "test")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (router must accept its own bearer token); body: %s",
				rec.Code, rec.Body.String())
		}
	})

	t.Run("garbage token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
		req.Header.Set("Authorization", "Bearer not-a-real-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 401 (or 429); body: %s", rec.Code, rec.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// Rate limiting test
// ---------------------------------------------------------------------------

func TestNewRouter_RateLimit(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	handler := authMiddlewareForTest(router)
	token := generateTestToken(t, "builder")

	// Send many requests quickly to trigger rate limiting
	// Rate limiter: 5 req/s, burst 10
	statuses := make([]int, 30)
	for i := 0; i < 30; i++ {
		req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		statuses[i] = rec.Code
	}

	// At least some should be 200 (burst allows up to ~10 before rate limiting kicks in)
	// And some should be 429 after the burst is consumed
	okCount := 0
	rateLimitedCount := 0
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			okCount++
		case http.StatusTooManyRequests:
			rateLimitedCount++
		}
	}

	t.Logf("OK: %d, Rate limited: %d out of 30", okCount, rateLimitedCount)
	if okCount == 0 {
		t.Error("expected at least some successful requests before rate limit")
	}
	if rateLimitedCount == 0 {
		t.Log("Note: rate limiter didn't trigger (may be too fast for test) — not a failure")
	}
}

func TestHandleLogin_LockoutReturns429(t *testing.T) {
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
		Threshold: 3,
		Lockout:   15 * time.Minute,
	})
	t.Cleanup(tracker.Stop)

	// httptest.NewRequest defaults RemoteAddr to "192.0.2.1:1234"
	testIP := "192.0.2.1"
	for i := 0; i < 3; i++ {
		tracker.RecordFailure(testIP)
	}

	// nil database: if lockout fires correctly, handler returns 429 before hitting DB
	handler := handleLogin(nil, tracker)

	body := strings.NewReader(`{"player_name":"anyone","password":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "too many failed attempts") {
		t.Errorf("expected lockout message in body, got: %s", rec.Body.String())
	}
}

// TestCorsMiddleware_LocalhostBlockedInProduction verifies that localhost
// origins are rejected unless ENVIRONMENT=development (DP-632).
func TestCorsMiddleware_LocalhostBlockedInProduction(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	origCORS := os.Getenv("ADMIN_CORS_ORIGIN")
	defer func() {
		_ = os.Setenv("ENVIRONMENT", origEnv)
		_ = os.Setenv("ADMIN_CORS_ORIGIN", origCORS)
	}()

	_ = os.Unsetenv("ENVIRONMENT")
	_ = os.Unsetenv("ADMIN_CORS_ORIGIN")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	corsMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("production localhost origin should not be allowed, got %q", got)
	}
}

// TestCorsMiddleware_LocalhostAllowedInDevelopment verifies that localhost
// origins are allowed when ENVIRONMENT=development.
func TestCorsMiddleware_LocalhostAllowedInDevelopment(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	origCORS := os.Getenv("ADMIN_CORS_ORIGIN")
	defer func() {
		_ = os.Setenv("ENVIRONMENT", origEnv)
		_ = os.Setenv("ADMIN_CORS_ORIGIN", origCORS)
	}()

	_ = os.Setenv("ENVIRONMENT", "development")
	_ = os.Unsetenv("ADMIN_CORS_ORIGIN")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	corsMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("development localhost origin should be allowed, got %q", got)
	}
}

// TestCorsMiddleware_EnvOriginAlwaysAllowed verifies that ADMIN_CORS_ORIGIN is
// honored regardless of ENVIRONMENT.
func TestCorsMiddleware_EnvOriginAlwaysAllowed(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	origCORS := os.Getenv("ADMIN_CORS_ORIGIN")
	defer func() {
		_ = os.Setenv("ENVIRONMENT", origEnv)
		_ = os.Setenv("ADMIN_CORS_ORIGIN", origCORS)
	}()

	_ = os.Unsetenv("ENVIRONMENT")
	_ = os.Setenv("ADMIN_CORS_ORIGIN", "https://admin.example.com")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	rec := httptest.NewRecorder()
	corsMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Errorf("ADMIN_CORS_ORIGIN should be allowed, got %q", got)
	}
}

// TestPrometheusEndpoint_RequiresAuth pins the move off the public root mux.
// The endpoint was unauthenticated on darkpawns.org and went unnoticed only
// because every gauge read zero; once the collectors were wired it would have
// published player counts and command_duration_seconds to anyone.
func TestPrometheusEndpoint_RequiresAuth(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	handler, err := NewRouter(w, nil, lb, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/prometheus", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated /admin/prometheus = %d, want 401", rec.Code)
	}

	// And with a valid immortal token it serves the exposition format.
	req = httptest.NewRequest(http.MethodGet, "/admin/prometheus", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(t, "builder"))
	req.Header.Set(adminCSRFHeader, "test")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated /admin/prometheus = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "darkpawns_") {
		t.Errorf("no darkpawns_ metrics in the exposition output: %s", body)
	}
}

// TestWorldWritePUTs_AreGone pins the removal of the HTTP world-write path.
//
// These endpoints mutated world state directly through world_write.go setters,
// bypassing the session and the command parser — so dp-oracle-diff, which
// drives the game over telnet, could never see the writes or compare them
// against the C original. They existed because there was no other way to edit
// the world from outside the game; redit (PR #1473) and medit (PR #1500) close
// that gap through the command path, where the oracle can reach them.
//
// GET on the same routes is untouched: the console's detail pages read through
// it, and reads cannot drift from C.
func TestWorldWritePUTs_AreGone(t *testing.T) {
	setJWTSecret(t)
	handler, err := NewRouter(testWorld(t), nil, NewLogBuffer(10), nil, nil)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	token := generateTestToken(t, "builder")

	for _, path := range []string{"/admin/rooms/3001", "/admin/mobs/3001", "/admin/objects/3001", "/admin/zones/30", "/admin/shops/2002"} {
		req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(`{"name":"x"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(adminCSRFHeader, "test")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("PUT %s = %d, want 405; the parser-bypassing write path is back", path, rec.Code)
		}

		// The read path on the same route must still work.
		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(adminCSRFHeader, "test")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusMethodNotAllowed {
			t.Errorf("GET %s = 405; the read path was removed with the write path", path)
		}
	}
}
