package optimization

import (
	"context"
	"database/sql"
	"fmt"
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
func (qo *QueryOptimizer) GetSlowQueries() []*QueryStat {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	var slowQueries []*QueryStat
	for _, stat := range qo.queryStats {
		if stat.AvgDuration > qo.slowQueryThreshold {
			slowQueries = append(slowQueries, stat)
		}
	}

	return slowQueries
}

// GetStats returns all query statistics.
func (qo *QueryOptimizer) GetStats() map[string]*QueryStat {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	stats := make(map[string]*QueryStat)
	for query, stat := range qo.queryStats {
		stats[query] = stat
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
	defer rows.Close()

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
type BatchProcessor struct {
	mu            sync.Mutex
	batchSize     int
	flushInterval time.Duration
	operations    []BatchOperation
	flushFunc     func([]BatchOperation) error
	timer         *time.Timer
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
	}

	bp.timer = time.AfterFunc(flushInterval, bp.flushTimer)
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

	// Reset timer
	bp.timer.Stop()
	bp.timer.Reset(bp.flushInterval)

	return nil
}

// Flush immediately processes the current batch.
func (bp *BatchProcessor) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushLocked()
}

// flushLocked processes the current batch (caller must hold the lock).
func (bp *BatchProcessor) flushLocked() error {
	if len(bp.operations) == 0 {
		return nil
	}

	operations := bp.operations
	bp.operations = make([]BatchOperation, 0, bp.batchSize)

	// Process batch asynchronously
	go func() {
		if err := bp.flushFunc(operations); err != nil {
			// TODO: Add proper error handling and retry logic
			fmt.Printf("Batch processing error: %v\n", err)
		}
	}()

	return nil
}

// flushTimer handles timer-based flushing.
func (bp *BatchProcessor) flushTimer() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if len(bp.operations) > 0 {
		_ = bp.flushLocked()
	}
}

// Close gracefully shuts down the batch processor.
func (bp *BatchProcessor) Close() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.timer.Stop()
	return bp.flushLocked()
}

// ConnectionMonitor monitors database connection health.
type ConnectionMonitor struct {
	mu            sync.RWMutex
	db            *sql.DB
	stats         ConnectionStats
	checkInterval time.Duration
	stopChan      chan struct{}
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
	close(cm.stopChan)
}
