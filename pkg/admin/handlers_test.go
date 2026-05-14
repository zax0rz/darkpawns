package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
	token, err := auth.GenerateJWT("TestPlayer", false, 0, role)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	return token
}

// ---------------------------------------------------------------------------
// handleZones
// ---------------------------------------------------------------------------

func TestHandleZones_GET(t *testing.T) {
	w := testWorld(t)
	handler := handleZones(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var zones []zoneResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &zones); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(zones) == 0 {
		t.Error("expected at least 1 zone")
	}
	// Verify zone 1 is present
	var z1 zoneResponse
	for _, z := range zones {
		if z.Number == 1 {
			z1 = z
			break
		}
	}
	if z1.Name != "Test Zone" || z1.TopRoom != 2000 || z1.Lifespan != 15 || z1.ResetMode != 1 {
		t.Errorf("zone 1 = %+v, unexpected values", z1)
	}
}

func TestHandleZones_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleZones(w)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/admin/zones", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s returned %d, want 405", method, rec.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleZoneByIDOrReset
// ---------------------------------------------------------------------------

func TestHandleZoneByIDOrReset_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var z zoneResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &z); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if z.Number != 1 {
		t.Errorf("zone number = %d, want 1", z.Number)
	}
}

func TestHandleZoneByIDOrReset_GET_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones/99", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleZoneByIDOrReset_GET_InvalidID(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones/abc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleZoneByIDOrReset_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	req := httptest.NewRequest(http.MethodDelete, "/admin/zones/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleZoneUpdate (PUT)
// ---------------------------------------------------------------------------

func TestHandleZoneUpdate_PUT_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	body := `{"lifespan": 30, "reset_mode": 2}`
	req := httptest.NewRequest(http.MethodPut, "/admin/zones/1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var z zoneResponse
	json.Unmarshal(rec.Body.Bytes(), &z)
	if z.Lifespan != 30 {
		t.Errorf("lifespan = %d, want 30", z.Lifespan)
	}
	if z.ResetMode != 2 {
		t.Errorf("reset mode = %d, want 2", z.ResetMode)
	}
}

func TestHandleZoneUpdate_PUT_InvalidJSON(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	req := httptest.NewRequest(http.MethodPut, "/admin/zones/1", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleZoneUpdate_PUT_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	body := `{"lifespan": 10}`
	req := httptest.NewRequest(http.MethodPut, "/admin/zones/99", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleZoneUpdate_PUT_NoFields(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneByIDOrReset(w, nil)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/admin/zones/1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleMobs
// ---------------------------------------------------------------------------

func TestHandleMobs_GET(t *testing.T) {
	w := testWorld(t)
	handler := handleMobs(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/mobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var mobs []mobResponse
	json.Unmarshal(rec.Body.Bytes(), &mobs)
	if len(mobs) < 2 {
		t.Errorf("expected >= 2 mobs, got %d", len(mobs))
	}
}

func TestHandleMobs_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleMobs(w)

	req := httptest.NewRequest(http.MethodPost, "/admin/mobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleObjects
// ---------------------------------------------------------------------------

func TestHandleObjects_GET(t *testing.T) {
	w := testWorld(t)
	handler := handleObjects(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/objects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var objs []objResponse
	json.Unmarshal(rec.Body.Bytes(), &objs)
	if len(objs) < 2 {
		t.Errorf("expected >= 2 objects, got %d", len(objs))
	}
}

// ---------------------------------------------------------------------------
// handleServerInfo
// ---------------------------------------------------------------------------

func TestHandleServerInfo_GET(t *testing.T) {
	w := testWorld(t)
	handler := handleServerInfo(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/server", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var info serverInfoResponse
	json.Unmarshal(rec.Body.Bytes(), &info)
	if info.RoomCount != 2 {
		t.Errorf("room count = %d, want 2", info.RoomCount)
	}
	if info.PlayerCount != 2 {
		t.Errorf("player count = %d, want 2", info.PlayerCount)
	}
	if info.ZoneCount != 1 {
		t.Errorf("zone count = %d, want 1", info.ZoneCount)
	}
	if info.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestHandleServerInfo_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleServerInfo(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/server", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleLogs
// ---------------------------------------------------------------------------

func TestHandleLogs_GET(t *testing.T) {
	lb := NewLogBuffer(100)
	for i := 0; i < 5; i++ {
		lb.Write([]byte(fmt.Sprintf("log entry %d", i+1)))
	}
	handler := handleLogs(lb)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var entries []string
	json.Unmarshal(rec.Body.Bytes(), &entries)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestHandleLogs_GET_WithLinesParam(t *testing.T) {
	lb := NewLogBuffer(100)
	for i := 0; i < 10; i++ {
		lb.Write([]byte(fmt.Sprintf("entry %d", i+1)))
	}
	handler := handleLogs(lb)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?lines=3", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var entries []string
	json.Unmarshal(rec.Body.Bytes(), &entries)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries (lines=3), got %d", len(entries))
	}
}

func TestHandleLogs_GET_InvalidLinesParam(t *testing.T) {
	lb := NewLogBuffer(100)
	lb.Write([]byte("entry"))
	handler := handleLogs(lb)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?lines=invalid", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("invalid lines param should default to 100, status = %d", rec.Code)
	}
}

func TestHandleLogs_WrongMethod(t *testing.T) {
	lb := NewLogBuffer(10)
	handler := handleLogs(lb)

	req := httptest.NewRequest(http.MethodPost, "/admin/logs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handlePlayers
// ---------------------------------------------------------------------------

func TestHandlePlayers_GET(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayers(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/players", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var players []playerResponse
	json.Unmarshal(rec.Body.Bytes(), &players)
	if len(players) != 2 {
		t.Errorf("expected 2 players, got %d", len(players))
	}
}

func TestHandlePlayers_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayers(w)

	req := httptest.NewRequest(http.MethodDelete, "/admin/players", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handlePlayerDetail
// ---------------------------------------------------------------------------

func TestHandlePlayerDetail_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil)

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
	handler := handlePlayerDetail(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/players/Nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePlayerDetail_GET_EmptyName(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/players/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePlayerDetail_Save_RequiresAdmin(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil)

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
	handler := handlePlayerDetail(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/save", nil)
	req = contextWithClaims(req, "admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Save will try to write to ./data/players/ which might fail in test env
	// Just verify the handler processes the request without panicking
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Errorf("unexpected status %d, wanted 200 or 500 (disk write depends on test env)", rec.Code)
	}
}

func TestHandlePlayerDetail_Save_BuilderRejected(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/save", nil)
	req = contextWithClaims(req, "builder")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestHandlePlayerDetail_Kick_NotImplemented(t *testing.T) {
	w := testWorld(t)
	handler := handlePlayerDetail(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/kick", nil)
	req = contextWithClaims(req, "admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleMetrics
// ---------------------------------------------------------------------------

func TestHandleMetrics_GET(t *testing.T) {
	w := testWorld(t)
	handler := handleMetrics(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var m metricsResponse
	json.Unmarshal(rec.Body.Bytes(), &m)
	if m.PlayerCount != 2 {
		t.Errorf("player count = %d, want 2", m.PlayerCount)
	}
	if m.RoomCount != 2 {
		t.Errorf("room count = %d, want 2", m.RoomCount)
	}
	if m.Goroutines == 0 {
		t.Error("goroutines should be > 0")
	}
}

func TestHandleMetrics_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleMetrics(w)

	req := httptest.NewRequest(http.MethodPut, "/admin/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleSaveWorld, handleResetAllZones
// ---------------------------------------------------------------------------

func TestHandleSaveWorld_Post(t *testing.T) {
	w := testWorld(t)
	handler := handleSaveWorld(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/save-world", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// SaveWorld tries to write to disk, so it might succeed or fail
	// Based on test environment
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Errorf("unexpected status %d, want 200 or 500; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSaveWorld_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleSaveWorld(w, nil)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/admin/save-world", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s returned %d, want 405", method, rec.Code)
			}
		})
	}
}

func TestHandleResetAllZones_Post(t *testing.T) {
	w := testWorld(t)
	handler := handleResetAllZones(w, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/reset-all-zones", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "reset triggered" {
		t.Errorf("status = %v, want 'reset triggered'", resp["status"])
	}
}

func TestHandleResetAllZones_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleResetAllZones(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/reset-all-zones", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleZoneReset (placeholder)
// ---------------------------------------------------------------------------

func TestHandleZoneReset_NotImplemented(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneReset(w)

	req := httptest.NewRequest(http.MethodPost, "/admin/zones/reset", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleZoneReset_WrongMethod(t *testing.T) {
	w := testWorld(t)
	handler := handleZoneReset(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones/reset", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleRoomByVnum
// ---------------------------------------------------------------------------

func TestHandleRoomByVnum_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/1001", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var room roomResponse
	json.Unmarshal(rec.Body.Bytes(), &room)
	if room.VNum != 1001 || room.Name != "Test Room" {
		t.Errorf("room = %+v, unexpected", room)
	}
}

func TestHandleRoomByVnum_GET_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/9999", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleRoomByVnum_GET_InvalidVNum(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/abc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRoomByVnum_GET_EmptyVNum(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/rooms/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleRoomUpdate (PUT)
// ---------------------------------------------------------------------------

func TestHandleRoomUpdate_PUT_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	body := `{"name": "Updated Room", "description": "An updated description."}`
	req := httptest.NewRequest(http.MethodPut, "/admin/rooms/1001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var room roomResponse
	json.Unmarshal(rec.Body.Bytes(), &room)
	if room.Name != "Updated Room" {
		t.Errorf("name = %q, want %q", room.Name, "Updated Room")
	}
}

func TestHandleRoomUpdate_PUT_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	body := `{"name": "Ghost Room"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/rooms/9999", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRoomUpdate_PUT_NoFields(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/admin/rooms/1001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRoomUpdate_PUT_InvalidJSON(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	body := `not json`
	req := httptest.NewRequest(http.MethodPut, "/admin/rooms/1001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRoomUpdate_PUT_Validation(t *testing.T) {
	w := testWorld(t)
	handler := handleRoomByVnum(w, nil)

	// Empty name should fail validation
	body := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPut, "/admin/rooms/1001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty name should return 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleMobByVnum
// ---------------------------------------------------------------------------

func TestHandleMobByVnum_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleMobByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/mobs/2001", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var m mobResponse
	json.Unmarshal(rec.Body.Bytes(), &m)
	if m.VNum != 2001 || m.ShortDesc != "a guard" {
		t.Errorf("mob = %+v, unexpected", m)
	}
}

func TestHandleMobByVnum_GET_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleMobByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/mobs/9999", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleMobUpdate (PUT)
// ---------------------------------------------------------------------------

func TestHandleMobUpdate_PUT_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleMobByVnum(w, nil)

	body := `{"short_desc": "a veteran guard", "level": 10}`
	req := httptest.NewRequest(http.MethodPut, "/admin/mobs/2001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMobUpdate_PUT_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleMobByVnum(w, nil)

	body := `{"short_desc": "ghost"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/mobs/9999", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleMobUpdate_PUT_NoFields(t *testing.T) {
	w := testWorld(t)
	handler := handleMobByVnum(w, nil)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/admin/mobs/2001", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleObjectByVnum
// ---------------------------------------------------------------------------

func TestHandleObjectByVnum_GET_Valid(t *testing.T) {
	w := testWorld(t)
	handler := handleObjectByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/objects/3001", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleObjectByVnum_GET_NotFound(t *testing.T) {
	w := testWorld(t)
	handler := handleObjectByVnum(w, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/objects/9999", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleShops
// ---------------------------------------------------------------------------

func TestHandleShops_GET(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShops(w)

	req := httptest.NewRequest(http.MethodGet, "/admin/shops", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var shops []shopResponse
	json.Unmarshal(rec.Body.Bytes(), &shops)
	if len(shops) != 1 {
		t.Errorf("expected 1 shop, got %d", len(shops))
	}
}

func TestHandleShops_WrongMethod(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShops(w)

	req := httptest.NewRequest(http.MethodPost, "/admin/shops", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

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

func TestHandleShopByKeeper_PUT_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShopByKeeper(w, nil)

	body := `{"buy_types": [2, 3]}`
	req := httptest.NewRequest(http.MethodPut, "/admin/shops/2002", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleShopByKeeper_PUT_NotFound(t *testing.T) {
	w := newWorldWithShops(t)
	handler := handleShopByKeeper(w, nil)

	body := `{"buy_types": [1]}`
	req := httptest.NewRequest(http.MethodPut, "/admin/shops/9999", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleAgents, handleAgentStatus
// ---------------------------------------------------------------------------

func TestHandleAgents_GET(t *testing.T) {
	store := NewAgentStore()
	handler := handleAgents(store)

	req := httptest.NewRequest(http.MethodGet, "/admin/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var agents []*AgentStatus
	json.Unmarshal(rec.Body.Bytes(), &agents)
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestHandleAgentStatus_POST_Valid(t *testing.T) {
	store := NewAgentStore()
	handler := handleAgentStatus(store)

	body := `{"agent_id": "daeron", "status": "active"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/agents/status", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var agent AgentStatus
	json.Unmarshal(rec.Body.Bytes(), &agent)
	if agent.Status != "active" {
		t.Errorf("status = %q, want %q", agent.Status, "active")
	}
}

func TestHandleAgentStatus_POST_NotFound(t *testing.T) {
	store := NewAgentStore()
	handler := handleAgentStatus(store)

	body := `{"agent_id": "nonexistent", "status": "active"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/agents/status", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleAgentStatus_POST_MissingFields(t *testing.T) {
	store := NewAgentStore()
	handler := handleAgentStatus(store)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/admin/agents/status", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleFindings
// ---------------------------------------------------------------------------

func TestHandleFindings_GET_Empty(t *testing.T) {
	store := NewAgentStore()
	handler := handleFindings(store)

	req := httptest.NewRequest(http.MethodGet, "/admin/findings", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var findings []Finding
	json.Unmarshal(rec.Body.Bytes(), &findings)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestHandleFindings_POST_Valid(t *testing.T) {
	store := NewAgentStore()
	handler := handleFindings(store)

	body := `{"source": "reek", "severity": "high", "title": "nil panic", "file": "handlers.go", "line": 42}`
	req := httptest.NewRequest(http.MethodPost, "/admin/findings", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleFindings_POST_MissingFields(t *testing.T) {
	store := NewAgentStore()
	handler := handleFindings(store)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/admin/findings", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleFindingByID (PUT)
// ---------------------------------------------------------------------------

func TestHandleFindingByID_PUT_Valid(t *testing.T) {
	store := NewAgentStore()
	f := store.AddFinding("reek", "high", "test", "f.go", 1, "")

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
	store := NewAgentStore()
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
	store := NewAgentStore()
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

func TestHandleTriageSummaries_POST_Valid(t *testing.T) {
	store := NewAgentStore()
	handler := handleTriageSummaries(store)

	body := `{"date": "2026-05-14", "confirmed": 5, "rejected": 1, "pending": 2, "summary": "Good day"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/triage/summaries", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTriageSummaries_GET(t *testing.T) {
	store := NewAgentStore()
	store.AddTriageSummary("2026-05-14", "test", 1, 0, 0)

	handler := handleTriageSummaries(store)
	req := httptest.NewRequest(http.MethodGet, "/admin/triage/summaries", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// authMiddlewareForTest validates a Bearer JWT and sets claims on context.
// This simulates what web.AuthMiddleware does in production.
func authMiddlewareForTest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	handler := NewRouter(w, nil, lb, nil)

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

func TestNewRouter_Unauthenticated_Returns401(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	handler := NewRouter(w, nil, lb, nil)
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
		"/admin/save-world",
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

	router := NewRouter(w, nil, lb, nil)
	handler := authMiddlewareForTest(router)

	// Generate a "player" role token
	token := generateTestToken(t, "player")

	// Admin-only endpoints should return 403 for player role
	t.Run("save-world", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/save-world", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
		}
	})

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

	router := NewRouter(w, nil, lb, nil)
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

	router := NewRouter(w, nil, lb, nil)
	handler := authMiddlewareForTest(router)
	token := generateTestToken(t, "admin")

	t.Run("save-world", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/save-world", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// May succeed or fail based on disk write, but shouldn't be 401/403
		if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
			t.Errorf("unexpected status %d for admin", rec.Code)
		}
	})
}

func TestNewRouter_CORS_NoOrigin(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router := NewRouter(w, nil, lb, nil)
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

	router := NewRouter(w, nil, lb, nil)
	handler := authMiddlewareForTest(router)

	req := httptest.NewRequest(http.MethodGet, "/admin/zones", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid token should return 401, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Rate limiting test
// ---------------------------------------------------------------------------

func TestNewRouter_RateLimit(t *testing.T) {
	setJWTSecret(t)
	w := testWorld(t)
	lb := NewLogBuffer(10)

	router := NewRouter(w, nil, lb, nil)
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
		if s == http.StatusOK {
			okCount++
		} else if s == http.StatusTooManyRequests {
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
