package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/errlog"
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

// AnalyzeTable reports the indexes a table already has, so a caller can see
// whether a column it queries by is covered.
//
// It used to read PostgreSQL's pg_stats for correlation estimates. That query is
// gone with the PostgreSQL runtime: the store is SQLite, which has no pg_stats,
// and guessing at a recommendation from a catalog that does not exist would be
// inventing analysis rather than performing it. What SQLite can answer exactly is
// which indexes exist and what they cover.
func (ia *IndexAnalyzer) AnalyzeTable(tableName string) ([]IndexRecommendation, error) {
	rows, err := ia.db.Query(`
		SELECT il.name, ii.name
		FROM pragma_index_list(?) AS il
		JOIN pragma_index_info(il.name) AS ii
		ORDER BY il.name, ii.seqno
	`, tableName)
	if err != nil {
		return nil, fmt.Errorf("read sqlite indexes for %s: %w", tableName, err)
	}
	defer errlog.Close(rows, "close query-analysis rows")

	var recommendations []IndexRecommendation
	seen := make(map[string]bool)
	for rows.Next() {
		var indexName, columnName string
		if err := rows.Scan(&indexName, &columnName); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if seen[columnName] {
			continue
		}
		seen[columnName] = true
		recommendations = append(recommendations, IndexRecommendation{
			TableName:  tableName,
			ColumnName: columnName,
			Reason:     fmt.Sprintf("already covered by index %s", indexName),
			Priority:   "COVERED",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sqlite index rows: %w", err)
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

	// Every number here comes from the database/sql pool, which is what a SQLite
	// store has: the former pg_stat_activity/pg_stat_database query went with the
	// PostgreSQL runtime, and there is no SQLite equivalent to substitute for it.
	dbStats := cm.db.Stats()
	cm.stats.OpenConnections = dbStats.OpenConnections
	cm.stats.InUse = dbStats.InUse
	cm.stats.Idle = dbStats.Idle
	cm.stats.WaitCount = dbStats.WaitCount
	cm.stats.WaitDuration = dbStats.WaitDuration
	cm.stats.LastCheck = time.Now()

	// Health is whether the file is still answerable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cm.stats.Healthy = cm.db.PingContext(ctx) == nil
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
