package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Connection metrics
	connectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "darkpawns_connections_active",
		Help: "Number of active WebSocket connections",
	})

	connectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_connections_total",
		Help: "Total number of WebSocket connections established",
	})

	connectionsErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_connection_errors_total",
		Help: "Total number of WebSocket connection errors",
	})

	// Command metrics
	commandsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "darkpawns_commands_processed_total",
		Help: "Total number of commands processed",
	}, []string{"type"})

	commandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "darkpawns_command_duration_seconds",
		Help:    "Duration of command processing in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	// Game state metrics
	playersOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "darkpawns_players_online",
		Help: "Number of players currently online",
	})

	roomsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "darkpawns_rooms_active",
		Help: "Number of active rooms with players",
	})

	mobsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "darkpawns_mobs_active",
		Help: "Number of active mobs in the world",
	})

	// Combat metrics
	combatRounds = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_combat_rounds_total",
		Help: "Total number of combat rounds processed",
	})

	damageDealt = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "darkpawns_damage_dealt_total",
		Help: "Total damage dealt in combat",
	}, []string{"source_type"})

	deathsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_deaths_total",
		Help: "Total number of player/mob deaths",
	})

	// Error metrics
	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "darkpawns_errors_total",
		Help: "Total number of errors by type",
	}, []string{"type"})

	// Database metrics
	dbQueriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_db_queries_total",
		Help: "Total number of database queries",
	})

	dbQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "darkpawns_db_query_duration_seconds",
		Help:    "Duration of database queries in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Memory metrics
	memoryWrites = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_memory_writes_total",
		Help: "Total number of narrative memory writes",
	})

	memoryReads = promauto.NewCounter(prometheus.CounterOpts{
		Name: "darkpawns_memory_reads_total",
		Help: "Total number of narrative memory reads",
	})
)

// Connection tracking
func ConnectionOpened() {
	connectionsActive.Inc()
	connectionsTotal.Inc()
}

func ConnectionClosed() {
	connectionsActive.Dec()
}

func ConnectionError() {
	connectionsErrors.Inc()
}

// Command tracking
func CommandProcessed(cmdType string, duration time.Duration) {
	commandsProcessed.WithLabelValues(cmdType).Inc()
	commandDuration.WithLabelValues(cmdType).Observe(duration.Seconds())
}

// Game state tracking
func SetPlayersOnline(count int) {
	playersOnline.Set(float64(count))
}

func SetRoomsActive(count int) {
	roomsActive.Set(float64(count))
}

func SetMobsActive(count int) {
	mobsActive.Set(float64(count))
}

// Combat tracking
func CombatRound() {
	combatRounds.Inc()
}

func DamageDealt(sourceType string, amount int) {
	damageDealt.WithLabelValues(sourceType).Add(float64(amount))
}

func Death() {
	deathsTotal.Inc()
}

// Error tracking
func ErrorOccurred(errorType string) {
	errorsTotal.WithLabelValues(errorType).Inc()
}

// Database tracking
func DBQuery(duration time.Duration) {
	dbQueriesTotal.Inc()
	dbQueryDuration.Observe(duration.Seconds())
}

// Memory tracking
func MemoryWrite() {
	memoryWrites.Inc()
}

func MemoryRead() {
	memoryReads.Inc()
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RegisterMetrics registers all metrics (called automatically on import)
func init() {
	// Metrics are registered automatically via promauto
}