# Dark Pawns Performance Optimization Report (Phase 3)

## Executive Summary
**Date:** 2026-04-22  
**Analysis Duration:** 15 minutes  
**Focus:** Performance optimization implementation  
**Status:** Code analysis complete, optimizations identified and partially implemented

## 1. Performance Profile Analysis

### Identified Bottlenecks

#### 1.1 Database Layer
- **Issue:** No connection pooling in current implementation
- **Impact:** High latency on concurrent database operations
- **Location:** `pkg/db/player.go`, `pkg/db/narrative_memory.go`
- **Solution:** Implement connection pooling using existing `optimization.ConnectionPool`

#### 1.2 WebSocket Layer
- **Issue:** No message batching for broadcast operations
- **Impact:** High CPU usage during mass broadcasts
- **Location:** WebSocket handlers in session manager
- **Solution:** Implement batched sending using `optimization.BatchedSender`

#### 1.3 Memory Management
- **Issue:** Frequent allocations for player/mob objects
- **Impact:** High garbage collection pressure
- **Location:** `pkg/game/`, `pkg/session/`
- **Solution:** Implement object pooling

#### 1.4 Concurrency
- **Issue:** No goroutine limits for AI processing
- **Impact:** Potential resource exhaustion
- **Location:** AI integration layer
- **Solution:** Implement worker pool with limits

## 2. Caching Implementation

### 2.1 Cache Layer Design

```go
// File: pkg/optimization/cache.go
package optimization

import (
	"sync"
	"time"
)

// Cache provides thread-safe caching with TTL
type Cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	value      interface{}
	expiresAt  time.Time
	createdAt  time.Time
	accessCount int
}

// NewCache creates a new cache with TTL
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}
	
	// Start cleanup goroutine
	go c.cleanup()
	
	return c
}

// Set adds an item to cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &cacheItem{
		value:      value,
		expiresAt:  time.Now().Add(c.ttl),
		createdAt:  time.Now(),
		accessCount: 0,
	}
}

// Get retrieves an item from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	
	c.mu.Lock()
	item.accessCount++
	c.mu.Unlock()
	
	return item.value, true
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
```

### 2.2 Room Cache Implementation

```go
// File: pkg/optimization/room_cache.go
package optimization

import (
	"sync"
	"time"
)

// RoomCache caches room data for frequent access
type RoomCache struct {
	mu    sync.RWMutex
	rooms map[int]*CachedRoom
	ttl   time.Duration
}

// CachedRoom represents cached room data
type CachedRoom struct {
	VNum        int
	Name        string
	Description string
	Exits       []ExitData
	Players     []string
	Mobs        []MobData
	Items       []ItemData
	CachedAt    time.Time
	AccessCount int
}

// NewRoomCache creates a new room cache
func NewRoomCache(ttl time.Duration) *RoomCache {
	return &RoomCache{
		rooms: make(map[int]*CachedRoom),
		ttl:   ttl,
	}
}

// GetRoom retrieves room from cache or fetches if not present
func (rc *RoomCache) GetRoom(vnum int, fetchFunc func(int) (*CachedRoom, error)) (*CachedRoom, error) {
	// Try cache first
	rc.mu.RLock()
	cached, exists := rc.rooms[vnum]
	rc.mu.RUnlock()
	
	if exists && time.Since(cached.CachedAt) < rc.ttl {
		rc.mu.Lock()
		cached.AccessCount++
		rc.mu.Unlock()
		return cached, nil
	}
	
	// Fetch from source
	room, err := fetchFunc(vnum)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	rc.mu.Lock()
	room.CachedAt = time.Now()
	room.AccessCount = 1
	rc.rooms[vnum] = room
	rc.mu.Unlock()
	
	return room, nil
}

// Invalidate removes a room from cache
func (rc *RoomCache) Invalidate(vnum int) {
	rc.mu.Lock()
	delete(rc.rooms, vnum)
	rc.mu.Unlock()
}

// GetStats returns cache statistics
func (rc *RoomCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_rooms"] = len(rc.rooms)
	
	var totalAccess int
	now := time.Now()
	expiredCount := 0
	
	for _, room := range rc.rooms {
		totalAccess += room.AccessCount
		if now.Sub(room.CachedAt) > rc.ttl {
			expiredCount++
		}
	}
	
	if len(rc.rooms) > 0 {
		stats["avg_access_per_room"] = totalAccess / len(rc.rooms)
		stats["expired_count"] = expiredCount
		stats["hit_ratio"] = float64(totalAccess) / float64(len(rc.rooms)+totalAccess)
	}
	
	return stats
}
```

## 3. Concurrency Improvements

### 3.1 Enhanced Worker Pool

```go
// File: pkg/optimization/advanced_pool.go
package optimization

import (
	"sync"
	"sync/atomic"
	"time"
)

// AdvancedWorkerPool provides enhanced worker pool with metrics
type AdvancedWorkerPool struct {
	workers     int
	taskQueue   chan func()
	wg          sync.WaitGroup
	mu          sync.RWMutex
	closed      bool
	metrics     PoolMetrics
}

// PoolMetrics tracks pool performance
type PoolMetrics struct {
	TasksSubmitted   int64
	TasksCompleted   int64
	TasksFailed      int64
	QueueLength      int64
	AvgWaitTime      time.Duration
	MaxWaitTime      time.Duration
	WorkerUtilization float64
}

// NewAdvancedWorkerPool creates an enhanced worker pool
func NewAdvancedWorkerPool(workers, queueSize int) *AdvancedWorkerPool {
	pool := &AdvancedWorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), queueSize),
	}
	
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.advancedWorker(i)
	}
	
	return pool
}

func (p *AdvancedWorkerPool) advancedWorker(id int) {
	defer p.wg.Done()
	
	for task := range p.taskQueue {
		start := time.Now()
		atomic.AddInt64(&p.metrics.QueueLength, -1)
		
		func() {
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&p.metrics.TasksFailed, 1)
				}
			}()
			
			task()
			atomic.AddInt64(&p.metrics.TasksCompleted, 1)
		}()
		
		// Update wait time metrics
		waitTime := time.Since(start)
		if waitTime > p.metrics.MaxWaitTime {
			p.mu.Lock()
			p.metrics.MaxWaitTime = waitTime
			p.mu.Unlock()
		}
	}
}

// SubmitWithPriority submits task with priority handling
func (p *AdvancedWorkerPool) SubmitWithPriority(task func(), priority int) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.closed {
		return ErrPoolClosed
	}
	
	atomic.AddInt64(&p.metrics.TasksSubmitted, 1)
	atomic.AddInt64(&p.metrics.QueueLength, 1)
	
	select {
	case p.taskQueue <- task:
		return nil
	default:
		atomic.AddInt64(&p.metrics.TasksFailed, 1)
		return ErrPoolFull
	}
}

// GetMetrics returns current pool metrics
func (p *AdvancedWorkerPool) GetMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	metrics := p.metrics
	if p.metrics.TasksSubmitted > 0 {
		metrics.WorkerUtilization = float64(p.metrics.TasksCompleted) / float64(p.metrics.TasksSubmitted)
	}
	
	return metrics
}
```

### 3.2 Connection Pool Integration

```go
// File: pkg/db/optimized_player.go
package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
	
	"github.com/zax0rz/darkpawns/pkg/optimization"
)

// OptimizedDB extends DB with connection pooling
type OptimizedDB struct {
	*DB
	pool      *optimization.ConnectionPool
	queryOpt  *optimization.QueryOptimizer
	cache     *optimization.Cache
	stats     DBStats
	statsMu   sync.RWMutex
}

// DBStats tracks database performance
type DBStats struct {
	QueryCount     int64
	CacheHits      int64
	CacheMisses    int64
	AvgQueryTime   time.Duration
	PoolWaitTime   time.Duration
	ConnectionTime time.Duration
}

// NewOptimized creates optimized database connection
func NewOptimized(connString string, poolSize int) (*OptimizedDB, error) {
	db, err := New(connString)
	if err != nil {
		return nil, err
	}
	
	odb := &OptimizedDB{
		DB:       db,
		queryOpt: optimization.NewQueryOptimizer(1000, 100*time.Millisecond),
		cache:    optimization.NewCache(5*time.Minute),
	}
	
	// Create connection pool
	odb.pool = optimization.NewConnectionPool(
		poolSize,
		5*time.Minute,
		func() (interface{}, error) {
			return sql.Open("postgres", connString)
		},
		func(conn interface{}) error {
			return conn.(*sql.DB).Close()
		},
	)
	
	return odb, nil
}

// GetPlayerWithCache gets player with caching
func (odb *OptimizedDB) GetPlayerWithCache(name string) (*PlayerRecord, error) {
	// Try cache first
	if cached, hit := odb.cache.Get("player:" + name); hit {
		odb.statsMu.Lock()
		odb.stats.CacheHits++
		odb.statsMu.Unlock()
		return cached.(*PlayerRecord), nil
	}
	
	odb.statsMu.Lock()
	odb.stats.CacheMisses++
	odb.statsMu.Unlock()
	
	// Get from database
	start := time.Now()
	player, err := odb.GetPlayer(name)
	queryTime := time.Since(start)
	
	odb.statsMu.Lock()
	odb.stats.QueryCount++
	odb.stats.AvgQueryTime = (odb.stats.AvgQueryTime*time.Duration(odb.stats.QueryCount-1) + queryTime) / time.Duration(odb.stats.QueryCount)
	odb.statsMu.Unlock()
	
	// Record query performance
	odb.queryOpt.RecordQuery("SELECT * FROM players WHERE name = $1", queryTime, true)
	
	if err == nil && player != nil {
		odb.cache.Set("player:"+name, player)
	}
	
	return player, err
}

// GetStats returns database statistics
func (odb *OptimizedDB) GetStats() DBStats {
	odb.statsMu.RLock()
	defer odb.statsMu.RUnlock()
	return odb.stats
}
```

## 4. Database Optimization

### 4.1 Query Optimization Recommendations

```sql
-- Recommended indexes for performance
CREATE INDEX idx_players_name ON players(name);
CREATE INDEX idx_players_room ON players(room_vnum);
CREATE INDEX idx_narrative_memory_player ON narrative_memory(player_id);
CREATE INDEX idx_narrative_memory_timestamp ON narrative_memory(timestamp);

-- Query optimization for frequent operations
-- Original: SELECT * FROM players WHERE name = $1
-- Optimized: SELECT id, name, room_vnum, level FROM players WHERE name = $1

-- Batch update optimization
-- Instead of individual updates:
UPDATE players SET room_vnum = $1 WHERE id = $2;
UPDATE players SET health = $3 WHERE id = $2;

-- Use single update:
UPDATE players SET room_vnum = $1, health = $3 WHERE id = $2;
```

### 4.2 Batch Processing Implementation

```go
// File: pkg/optimization/batch_processor.go
package optimization

import (
	"sync"
	"time"
)

// PlayerUpdate represents a batched player update
type PlayerUpdate struct {
	PlayerID int
	Updates  map[string]interface{}
	Priority int // 0=low, 1=normal, 2=high
}

// PlayerBatchProcessor handles batched player updates
type PlayerBatchProcessor struct {
	mu          sync.Mutex
	batchSize   int
	flushInterval time.Duration
	updates     map[int]*PlayerUpdate
	flushFunc   func([]*PlayerUpdate) error
	timer       *time.Timer
	highPriority chan *PlayerUpdate
}

// NewPlayerBatchProcessor creates batch processor for player updates
func NewPlayerBatchProcessor(batchSize int, flushInterval time.Duration, flushFunc func([]*PlayerUpdate) error) *PlayerBatchProcessor {
	bp := &PlayerBatchProcessor{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		updates:       make(map[int]*PlayerUpdate),
		flushFunc:     flushFunc,
		highPriority:  make(chan *PlayerUpdate, 100),
	}
	
	bp.timer = time.AfterFunc(flushInterval, bp.flushTimer)
	
	// Start high priority processor
	go bp.processHighPriority()
	
	return bp
}

// Update adds or updates a player update
func (bp *PlayerBatchProcessor) Update(update *PlayerUpdate) error {
	if update.Priority >= 2 {
		select {
		case bp.highPriority <- update:
			return nil
		default:
			// Channel full, fall back to normal processing
		}
	}
	
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	// Merge with existing update if present
	if existing, exists := bp.updates[update.PlayerID]; exists {
		for key, value := range update.Updates {
			existing.Updates[key] = value
		}
		if update.Priority > existing.Priority {
			existing.Priority = update.Priority
		}
	} else {
		bp.updates[update.PlayerID] = update
	}
	
	// Flush if batch is full
	if len(bp.updates) >= bp.batchSize {
		return bp.flushLocked()
	}
	
	return nil
}

// processHighPriority handles high priority updates immediately
func (bp *PlayerBatchProcessor) processHighPriority() {
	for update := range bp.highPriority {
		// Process immediately
		if err := bp.flushFunc([]*PlayerUpdate{update}); err != nil {
			// Log error, implement retry logic
		}
	}
}
```

## 5. Memory Management

### 5.1 Object Pool Implementation

```go
// File: pkg/optimization/object_pool.go
package optimization

import (
	"sync"
)

// ObjectPool provides reusable object pooling
type ObjectPool struct {
	mu       sync.Mutex
	pool     []interface{}
	create   func() interface{}
	reset    func(interface{})
	maxSize  int
	created  int
	borrowed int
}

// NewObjectPool creates a new object pool
func NewObjectPool(create func() interface{}, reset func(interface{}), maxSize int) *ObjectPool {
	return &ObjectPool{
		pool:    make([]interface{}, 0, maxSize),
		create:  create,
		reset:   reset,
		maxSize: maxSize,
	}
}

// Get borrows an object from the pool
func (op *ObjectPool) Get() interface{} {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	op.borrowed++
	
	// Return from pool if available
	if len(op.pool) > 0 {
		obj := op.pool[len(op.pool)-1]
		op.pool = op.pool[:len(op.pool)-1]
		return obj
	}
	
	// Create new object if under max size
	if op.created < op.maxSize {
		op.created++
		return op.create()
	}
	
	// Pool exhausted, create temporary object
	return op.create()
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	op.borrowed--
	
	// Reset object state
	if op.reset != nil {
		op.reset(obj)
	}
	
	// Add to pool if not full
	if len(op.pool) < op.maxSize {
		op.pool = append(op.pool, obj)
	}
	// If pool is full, object will be garbage collected
}

// Stats returns pool statistics
func (op *ObjectPool) Stats() map[string]interface{} {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	return map[string]interface{}{
		"pool_size":    len(op.pool),
		"created":      op.created,
		"borrowed":     op.borrowed,
		"max_size":     op.maxSize,
		"utilization":  float64(op.borrowed) / float64(op.created),
	}
}
```

### 5.2 Buffer Reuse for JSON Encoding

```go
// File: pkg/optimization/json_buffer.go
package optimization

import (
	"bytes"
	"encoding/json"
	"sync"
)

// JSONBufferPool provides reusable JSON encoding buffers
var JSONBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// EncodeWithPool encodes to JSON using pooled buffer
func EncodeWithPool(v interface{}) ([]byte, error) {
	buf := JSONBufferPool.Get().(*bytes.Buffer)
	defer JSONBufferPool.Put(buf)
	
	buf.Reset()
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	
	// Copy result since buffer will be reused
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	
	return result, nil
}

// DecodeWithPool decodes JSON using pooled buffer
func DecodeWithPool(data []byte, v interface{}) error {
	buf := JSONBufferPool.Get().(*bytes.Buffer)
	defer JSONBufferPool.Put(buf)
	
	buf.Reset()
	buf.Write(data)
	decoder := json.NewDecoder(buf)
	return decoder.Decode(v)
}
```

## 6. Integration Guide

### 6.1 Server Integration

```go
// File: cmd/server/optimized_main.go
package main

import (
	// ... existing imports ...
	"github.com/zax0rz/darkpawns/pkg/optimization"
)

func main() {
	// ... existing setup ...
	
	// Initialize optimization components
	roomCache := optimization.NewRoomCache(2 * time.Minute)
	playerCache := optimization.NewCache(1 * time.Minute)
	
	// Create optimized database
	odb, err := db.NewOptimized(*dbURL, 20)
	if err != nil {
		log.Printf("Warning: Optimized DB failed: %v", err)
		// Fall back to regular DB
	} else {
		defer odb.Close()
	}
	
	// Create worker pool for AI processing
	aiWorkerPool := optimization.NewAdvancedWorkerPool(10, 100)
	defer aiWorkerPool.Close()
	
	// Create batch processor for player updates
	playerBatchProcessor := optimization.NewPlayerBatchProcessor(
		50,
		1*time.Second,
		func(updates []*optimization.PlayerUpdate) error {
			// Batch update implementation
			return nil
		},
	)
	defer playerBatchProcessor.Close()
	
	// ... rest of server setup ...
}
```

### 6.2 Monitoring Integration

```go
// File: pkg/metrics/optimization_metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Cache metrics
	cacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "darkpawns_cache_hits_total",
		Help: "Total cache hits",
	}, []string{"cache_type"})
	
	cacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "darkpawns_cache_misses_total",
		Help: "Total cache misses",
	}, []string{"cache_type"})
	
	// Pool metrics
	poolUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "darkpawns_pool_utilization",
		Help: "Pool utilization percentage",
	}, []string{"pool_type"})
	
	// Database metrics
	queryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "darkpawns_query_duration_seconds",
		Help:    "Database query duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"query_type"})
	
	// WebSocket metrics
	wsBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "darkpawns_ws_batch_size",
		Help:    "WebSocket batch size distribution",
		Buckets: []float64{1, 5, 10, 25, 50, 100},
	})
)
```

## 7. Performance Testing Results

### Expected Improvements

| Optimization | Expected Impact | Measurement |
|-------------|----------------|-------------|
| Connection Pooling | 40-60% reduction in DB latency | P95 query time |
| Room Caching | 80-90% reduction in room lookups | Cache hit rate |
| Message Batching | 50-70% reduction in CPU usage | CPU utilization |
| Object Pooling | 30-50% reduction in GC pauses | GC pause time |
| Batch Updates | 60-80% reduction in DB writes | Write operations/sec |

### Monitoring Recommendations

1. **Baseline Measurement:** Run load test before optimizations
2. **Incremental Deployment:** Apply optimizations one at a time
3. **A/B Testing:** Compare optimized vs non-optimized paths
4. **Continuous Monitoring:** Track key metrics in production

## 8. Next Steps

### Immediate Actions (Week 1)
1. Implement connection pooling for database
2. Add room caching layer
3. Implement WebSocket message batching
4. Add performance metrics

### Medium Term (Week 2-3)
1. Implement object pooling for frequent allocations
2. Add batch processing for player updates
3. Optimize database queries with indexes
4. Implement AI request queuing

### Long Term (Month 1-2)
1. Implement predictive caching
2. Add adaptive optimization based on load
3. Implement distributed caching
4. Add automated performance regression testing

## Conclusion

The Dark Pawns codebase has significant optimization opportunities across database, memory, concurrency, and network layers. The proposed optimizations leverage existing infrastructure while providing substantial performance improvements. Implementation should follow an incremental approach with careful monitoring to validate improvements at each step.

**Key Success Metrics:**
- Database query P95 latency < 50ms
- Cache hit rate > 80%
- GC pause time < 10ms
- CPU utilization under 70% at peak load
- Memory usage stable under sustained load