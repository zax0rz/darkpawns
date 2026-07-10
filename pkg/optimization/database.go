package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// QueryOptimizer provides database query optimization.
type QueryOptimizer struct {
	mu                 sync.RWMutex
	queryStats         map[string]*QueryStat
	maxStats           int
	slowQueryThreshold time.Duration
}

// QueryStat tracks statistics for a single query.
type QueryStat struct {
	Query         string
	Count         int64
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
	LastExecuted  time.Time
	IndexUsed     bool
}

// NewQueryOptimizer creates a new query optimizer.
func NewQueryOptimizer(maxStats int, slowQueryThreshold time.Duration) *QueryOptimizer {
	return &QueryOptimizer{
		queryStats:         make(map[string]*QueryStat),
		maxStats:           maxStats,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// RecordQuery records query execution statistics.
func (qo *QueryOptimizer) RecordQuery(query string, duration time.Duration, indexUsed bool) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	stat, exists := qo.queryStats[query]
	if !exists {
		// Limit number of tracked queries
		if len(qo.queryStats) >= qo.maxStats {
			// Remove oldest query (simple implementation)
			var oldestKey string
			var oldestTime time.Time
			for key, s := range qo.queryStats {
				if oldestTime.IsZero() || s.LastExecuted.Before(oldestTime) {
					oldestTime = s.LastExecuted
					oldestKey = key
				}
			}
			delete(qo.queryStats, oldestKey)
		}

		stat = &QueryStat{
			Query:       query,
			MinDuration: duration,
		}
		qo.queryStats[query] = stat
	}

	stat.Count++
	stat.TotalDuration += duration
	stat.AvgDuration = stat.TotalDuration / time.Duration(stat.Count)

	if duration > stat.MaxDuration {
		stat.MaxDuration = duration
	}
	if duration < stat.MinDuration {
		stat.MinDuration = duration
	}

	stat.LastExecuted = time.Now()
	stat.IndexUsed = indexUsed
}

// GetSlowQueries returns queries that exceed the slow threshold.
// The returned QueryStat pointers are deep-copied snapshots so callers cannot
// mutate the optimizer's internal state or observe data races with RecordQuery.
func (qo *QueryOptimizer) GetSlowQueries() []*QueryStat {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	slowQueries := make([]*QueryStat, 0, len(qo.queryStats))
	for _, stat := range qo.queryStats {
		if stat.AvgDuration > qo.slowQueryThreshold {
			copy := *stat
			slowQueries = append(slowQueries, &copy)
		}
	}

	return slowQueries
}

// GetStats returns all query statistics.
// The returned QueryStat pointers are deep-copied snapshots so callers cannot
// mutate the optimizer's internal state or observe data races with RecordQuery.
func (qo *QueryOptimizer) GetStats() map[string]*QueryStat {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	stats := make(map[string]*QueryStat, len(qo.queryStats))
	for query, stat := range qo.queryStats {
		copy := *stat
		stats[query] = &copy
	}

	return stats
}

// IndexAnalyzer analyzes and suggests database indexes.
type IndexAnalyzer struct {
	db *sql.DB
}

// NewIndexAnalyzer creates a new index analyzer.
func NewIndexAnalyzer(db *sql.DB) *IndexAnalyzer {
	return &IndexAnalyzer{db: db}
}

// AnalyzeTable analyzes a table for missing indexes.
func (ia *IndexAnalyzer) AnalyzeTable(tableName string) ([]IndexRecommendation, error) {
	query := `
		SELECT 
			attname,
			most_common_vals,
			most_common_freqs,
			histogram_bounds,
			correlation
		FROM pg_stats 
		WHERE tablename = $1
	`

	rows, err := ia.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("query pg_stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var recommendations []IndexRecommendation

	for rows.Next() {
		var columnName string
		var mostCommonVals, mostCommonFreqs, histogramBounds sql.NullString
		var correlation sql.NullFloat64

		if err := rows.Scan(&columnName, &mostCommonVals, &mostCommonFreqs, &histogramBounds, &correlation); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Simple heuristic: recommend index for low-correlation columns
		if correlation.Valid && correlation.Float64 < 0.3 {
			recommendations = append(recommendations, IndexRecommendation{
				TableName:  tableName,
				ColumnName: columnName,
				Reason:     fmt.Sprintf("Low correlation (%.2f) suggests index would help", correlation.Float64),
				Priority:   "MEDIUM",
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating pg_stats rows: %w", err)
	}

	return recommendations, nil
}

// IndexRecommendation represents a suggested database index.
type IndexRecommendation struct {
	TableName  string
	ColumnName string
	Reason     string
	Priority   string // HIGH, MEDIUM, LOW
}

// BatchProcessor handles batch database operations.
// maxBacklogFactor bounds how many batches worth of operations may accumulate
// when flushes keep failing before the oldest are dropped to cap memory use.
const maxBacklogFactor = 100

type BatchProcessor struct {
	mu            sync.Mutex
	batchSize     int
	flushInterval time.Duration
	operations    []BatchOperation
	flushFunc     func([]BatchOperation) error
	closeCh       chan struct{}
	resetCh       chan struct{}
	wg            sync.WaitGroup
}

// BatchOperation represents a single batchable database operation.
type BatchOperation struct {
	Type      string // "insert", "update", "delete"
	Table     string
	Data      interface{}
	Timestamp time.Time
}

// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(batchSize int, flushInterval time.Duration, flushFunc func([]BatchOperation) error) *BatchProcessor {
	bp := &BatchProcessor{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		operations:    make([]BatchOperation, 0, batchSize),
		flushFunc:     flushFunc,
		closeCh:       make(chan struct{}),
		resetCh:       make(chan struct{}, 1),
	}

	bp.wg.Add(1)
	go bp.flushLoop()
	return bp
}

// Add adds an operation to the batch.
func (bp *BatchProcessor) Add(op BatchOperation) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.operations = append(bp.operations, op)

	// Flush if batch is full
	if len(bp.operations) >= bp.batchSize {
		return bp.flushLocked()
	}

	// Signal flushLoop to reset its timer
	select {
	case bp.resetCh <- struct{}{}:
	default:
	}

	return nil
}

// Flush immediately processes the current batch.
func (bp *BatchProcessor) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushLocked()
}

// flushLocked processes the current batch synchronously (caller must hold the
// lock). Errors are returned to the caller instead of being swallowed by a
// background goroutine.
func (bp *BatchProcessor) flushLocked() error {
	if len(bp.operations) == 0 {
		return nil
	}

	operations := bp.operations
	bp.operations = make([]BatchOperation, 0, bp.batchSize)

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 400ms, 900ms
			time.Sleep(time.Duration(attempt*attempt*100) * time.Millisecond)
		}
		if err := bp.flushFunc(operations); err != nil {
			lastErr = err
			slog.Warn("batch flush failed, retrying",
				"attempt", attempt+1,
				"batch_size", len(operations),
				"error", err)
			continue
		}
		return nil
	}

	// Requeue the failed batch so it is retried on the next flush instead of
	// being dropped. This gives at-least-once semantics to every caller,
	// including the background flushLoop which otherwise silently lost data.
	// We hold bp.mu, so bp.operations has not been touched since we detached
	// the batch above; restore it ahead of any (impossible here) new entries.
	bp.operations = append(operations, bp.operations...)
	// Bound the backlog: under a sustained flush outage, cap memory by dropping
	// the oldest operations rather than growing without limit.
	if limit := bp.batchSize * maxBacklogFactor; limit > 0 && len(bp.operations) > limit {
		dropped := len(bp.operations) - limit
		bp.operations = bp.operations[dropped:]
		slog.Error("batch backlog exceeded cap, dropping oldest operations",
			"dropped", dropped, "cap", limit)
	}
	slog.Error("batch flush failed after retries, requeued for retry",
		"max_retries", maxRetries,
		"batch_size", len(operations),
		"queued", len(bp.operations),
		"error", lastErr)
	return lastErr
}

// flushLoop runs in a background goroutine, flushing on interval or when resetCh signals.
func (bp *BatchProcessor) flushLoop() {
	defer bp.wg.Done()
	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.mu.Lock()
			if len(bp.operations) > 0 {
				if err := bp.flushLocked(); err != nil {
					// flushLocked already logged and requeued the batch for the
					// next tick; nothing is dropped unless the backlog cap is hit.
					slog.Warn("background flush failed, batch requeued",
						"error", err)
				}
			}
			bp.mu.Unlock()

		case <-bp.resetCh:
			ticker.Reset(bp.flushInterval)

		case <-bp.closeCh:
			return
		}
	}
}

// Close gracefully shuts down the batch processor.
func (bp *BatchProcessor) Close() error {
	close(bp.closeCh)
	bp.wg.Wait()

	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Flush any remaining operations synchronously.
	return bp.flushLocked()
}

// ConnectionMonitor monitors database connection health.
type ConnectionMonitor struct {
	mu            sync.RWMutex
	db            *sql.DB
	stats         ConnectionStats
	checkInterval time.Duration
	stopChan      chan struct{}
	stopOnce      sync.Once
}

// ConnectionStats holds database connection statistics.
type ConnectionStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
	LastCheck       time.Time
	Healthy         bool
}

// NewConnectionMonitor creates a new connection monitor.
func NewConnectionMonitor(db *sql.DB, checkInterval time.Duration) *ConnectionMonitor {
	cm := &ConnectionMonitor{
		db:            db,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}

	go cm.monitor()
	return cm
}

// monitor periodically checks connection health.
func (cm *ConnectionMonitor) monitor() {
	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.checkHealth()
		case <-cm.stopChan:
			return
		}
	}
}

// checkHealth checks database connection health.
func (cm *ConnectionMonitor) checkHealth() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get connection stats from PostgreSQL
	var stats struct {
		NumBackends  int
		XactCommit   int64
		XactRollback int64
		BlksRead     int64
		BlksHit      int64
	}

	err := cm.db.QueryRow(`
		SELECT 
			(SELECT count(*) FROM pg_stat_activity) as num_backends,
			(SELECT sum(xact_commit) FROM pg_stat_database) as xact_commit,
			(SELECT sum(xact_rollback) FROM pg_stat_database) as xact_rollback,
			(SELECT sum(blks_read) FROM pg_stat_database) as blks_read,
			(SELECT sum(blks_hit) FROM pg_stat_database) as blks_hit
	`).Scan(&stats.NumBackends, &stats.XactCommit, &stats.XactRollback, &stats.BlksRead, &stats.BlksHit)
	if err != nil {
		cm.stats.Healthy = false
		return
	}

	cm.stats.OpenConnections = stats.NumBackends
	cm.stats.LastCheck = time.Now()
	cm.stats.Healthy = true

	// Populate Go sql.DB stats (InUse/Idle/WaitCount/WaitDuration) from the
	// connection pool itself.
	dbStats := cm.db.Stats()
	cm.stats.InUse = dbStats.InUse
	cm.stats.Idle = dbStats.Idle
	cm.stats.WaitCount = dbStats.WaitCount
	cm.stats.WaitDuration = dbStats.WaitDuration

	// Simple health check: ping the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cm.db.PingContext(ctx); err != nil {
		cm.stats.Healthy = false
	}
}

// GetStats returns current connection statistics.
func (cm *ConnectionMonitor) GetStats() ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}

// Stop stops the connection monitor.
func (cm *ConnectionMonitor) Stop() {
	cm.stopOnce.Do(func() {
		close(cm.stopChan)
	})
}
