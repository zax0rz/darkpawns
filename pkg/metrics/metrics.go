package metrics

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Connection metrics
	connectionsActive    prometheus.Gauge
	connectionsTotal     prometheus.Counter
	connectionsErrors    prometheus.Counter
	connectionsUnderflow prometheus.Counter

	// Command metrics
	commandsProcessed *prometheus.CounterVec
	commandDuration   *prometheus.HistogramVec

	// Game state metrics
	playersOnline prometheus.Gauge
	roomsActive   prometheus.Gauge
	mobsActive    prometheus.Gauge

	// Combat metrics
	combatRounds prometheus.Counter
	damageDealt  *prometheus.CounterVec
	deathsTotal  prometheus.Counter

	// Error metrics
	errorsTotal *prometheus.CounterVec

	// Database metrics
	dbQueriesTotal  prometheus.Counter
	dbQueryDuration prometheus.Histogram

	// Memory metrics
	memoryWrites prometheus.Counter
	memoryReads  prometheus.Counter

	// registerer is the registry metrics are collected into.
	registerer prometheus.Registerer = prometheus.DefaultRegisterer
	initOnce   sync.Once

	// activeConnections mirrors connectionsActive so we can guard against
	// unmatched close events without reading the Prometheus gauge value.
	activeConnections atomic.Int64
)

// Init creates and registers all metrics with the provided registerer.
// It is safe to call multiple times; only the first call has effect.
func Init(r prometheus.Registerer) {
	initOnce.Do(func() {
		registerer = r

		connectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "darkpawns_connections_active",
			Help: "Number of active WebSocket connections",
		})
		connectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_connections_total",
			Help: "Total number of WebSocket connections established",
		})
		connectionsErrors = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_connection_errors_total",
			Help: "Total number of WebSocket connection errors",
		})
		connectionsUnderflow = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_connections_underflow_total",
			Help: "Total number of connection close events that would have driven the active gauge negative",
		})

		commandsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "darkpawns_commands_processed_total",
			Help: "Total number of commands processed",
		}, []string{"type"})
		commandDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "darkpawns_command_duration_seconds",
			Help:    "Duration of command processing in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"type"})

		playersOnline = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "darkpawns_players_online",
			Help: "Number of players currently online",
		})
		roomsActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "darkpawns_rooms_active",
			Help: "Number of active rooms with players",
		})
		mobsActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "darkpawns_mobs_active",
			Help: "Number of active mobs in the world",
		})

		combatRounds = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_combat_rounds_total",
			Help: "Total number of combat rounds processed",
		})
		damageDealt = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "darkpawns_damage_dealt_total",
			Help: "Total damage dealt in combat",
		}, []string{"source_type"})
		deathsTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_deaths_total",
			Help: "Total number of player/mob deaths",
		})

		errorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "darkpawns_errors_total",
			Help: "Total number of errors by type",
		}, []string{"type"})

		dbQueriesTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_db_queries_total",
			Help: "Total number of database queries",
		})
		dbQueryDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "darkpawns_db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: prometheus.DefBuckets,
		})

		memoryWrites = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_memory_writes_total",
			Help: "Total number of narrative memory writes",
		})
		memoryReads = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "darkpawns_memory_reads_total",
			Help: "Total number of narrative memory reads",
		})

		registerer.MustRegister(
			connectionsActive,
			connectionsTotal,
			connectionsErrors,
			connectionsUnderflow,
			commandsProcessed,
			commandDuration,
			playersOnline,
			roomsActive,
			mobsActive,
			combatRounds,
			damageDealt,
			deathsTotal,
			errorsTotal,
			dbQueriesTotal,
			dbQueryDuration,
			memoryWrites,
			memoryReads,
		)
	})
}

// Connection tracking
func ConnectionOpened() {
	activeConnections.Add(1)
	connectionsActive.Inc()
	connectionsTotal.Inc()
}

func ConnectionClosed() {
	for {
		v := activeConnections.Load()
		if v > 0 {
			if activeConnections.CompareAndSwap(v, v-1) {
				connectionsActive.Dec()
				return
			}
			continue
		}
		connectionsUnderflow.Inc()
		return
	}
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
	if amount < 0 {
		return
	}
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

// Handler returns the Prometheus metrics HTTP handler using the configured
// registry. Init must have been called before Handler is used.
func Handler() http.Handler {
	return promhttp.HandlerFor(registerer.(prometheus.Gatherer), promhttp.HandlerOpts{})
}

// RegisterMetrics registers all metrics (called automatically on import)
func init() {
	Init(prometheus.DefaultRegisterer)
}
