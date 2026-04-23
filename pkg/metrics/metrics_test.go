package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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