package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics(t *testing.T) {
	// Test connection metrics
	ConnectionOpened()
	ConnectionClosed()
	ConnectionError()

	// Test command metrics
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	CommandProcessed("look", time.Since(start))

	// Test game state metrics
	SetPlayersOnline(5)
	SetRoomsActive(10)
	SetMobsActive(3)

	// Test combat metrics
	CombatRound()
	DamageDealt("player", 25)
	Death()

	// Test error metrics
	ErrorOccurred("database")

	// Test database metrics
	DBQuery(50 * time.Millisecond)

	// Test memory metrics
	MemoryWrite()
	MemoryRead()

	// Verify metric values, not just names (DP-578).
	if got := testutil.ToFloat64(connectionsActive); got != 0 {
		t.Errorf("connections_active = %v, want 0", got)
	}
	if got := testutil.ToFloat64(connectionsTotal); got != 1 {
		t.Errorf("connections_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(connectionsErrors); got != 1 {
		t.Errorf("connection_errors_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(commandsProcessed.WithLabelValues("look")); got != 1 {
		t.Errorf("commands_processed_total{look} = %v, want 1", got)
	}
	if got := testutil.ToFloat64(playersOnline); got != 5 {
		t.Errorf("players_online = %v, want 5", got)
	}
	if got := testutil.ToFloat64(roomsActive); got != 10 {
		t.Errorf("rooms_active = %v, want 10", got)
	}
	if got := testutil.ToFloat64(mobsActive); got != 3 {
		t.Errorf("mobs_active = %v, want 3", got)
	}
	if got := testutil.ToFloat64(combatRounds); got != 1 {
		t.Errorf("combat_rounds_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(damageDealt.WithLabelValues("player")); got != 25 {
		t.Errorf("damage_dealt_total{player} = %v, want 25", got)
	}
	if got := testutil.ToFloat64(deathsTotal); got != 1 {
		t.Errorf("deaths_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(errorsTotal.WithLabelValues("database")); got != 1 {
		t.Errorf("errors_total{database} = %v, want 1", got)
	}
	if got := testutil.ToFloat64(dbQueriesTotal); got != 1 {
		t.Errorf("db_queries_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(memoryWrites); got != 1 {
		t.Errorf("memory_writes_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(memoryReads); got != 1 {
		t.Errorf("memory_reads_total = %v, want 1", got)
	}

	// Test HTTP handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler := Handler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that metrics are present in response
	body := rr.Body.String()
	expectedMetrics := []string{
		"darkpawns_connections_active",
		"darkpawns_connections_total",
		"darkpawns_connection_errors_total",
		"darkpawns_commands_processed_total",
		"darkpawns_players_online",
		"darkpawns_rooms_active",
		"darkpawns_mobs_active",
		"darkpawns_combat_rounds_total",
		"darkpawns_damage_dealt_total",
		"darkpawns_deaths_total",
		"darkpawns_errors_total",
		"darkpawns_db_queries_total",
		"darkpawns_memory_writes_total",
		"darkpawns_memory_reads_total",
	}

	for _, metric := range expectedMetrics {
		if !contains(body, metric) {
			t.Errorf("expected metric %s not found in response", metric)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

func TestConnectionClosed_NegativeFloor(t *testing.T) {
	// Reset shared state so this test is independent of execution order.
	activeConnections.Store(0)
	connectionsActive.Set(0)

	before := testutil.ToFloat64(connectionsUnderflow)

	// Closing without opening should not drive the gauge negative.
	ConnectionClosed()
	ConnectionClosed()

	if got := testutil.ToFloat64(connectionsActive); got != 0 {
		t.Errorf("connections_active = %v, want 0", got)
	}
	if got := testutil.ToFloat64(connectionsUnderflow) - before; got != 2 {
		t.Errorf("connections_underflow_total increased by %v, want 2", got)
	}
}

func TestInit_TwiceDoesNotPanic(t *testing.T) {
	fresh := prometheus.NewRegistry()
	Init(fresh)
	Init(fresh) // second call must not panic
}

func TestDamageDealt_Negative(t *testing.T) {
	// Add positive damage
	DamageDealt("player_test", 10)

	// Verify it increased
	req2 := httptest.NewRequest("GET", "/metrics", nil)
	rr2 := httptest.NewRecorder()
	Handler().ServeHTTP(rr2, req2)
	body2 := rr2.Body.String()

	if !strings.Contains(body2, `darkpawns_damage_dealt_total{source_type="player_test"} 10`) {
		t.Errorf("expected metric value to be 10, got body: %s", body2)
	}

	// Try to add negative damage
	DamageDealt("player_test", -5)

	// Verify it did NOT change (still 10)
	req3 := httptest.NewRequest("GET", "/metrics", nil)
	rr3 := httptest.NewRecorder()
	Handler().ServeHTTP(rr3, req3)
	body3 := rr3.Body.String()

	if !strings.Contains(body3, `darkpawns_damage_dealt_total{source_type="player_test"} 10`) {
		t.Errorf("expected metric value to remain 10, got body: %s", body3)
	}
}
