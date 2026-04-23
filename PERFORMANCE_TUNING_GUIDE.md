# Dark Pawns Performance Tuning Guide

## Overview

This guide provides step-by-step instructions for performance tuning Dark Pawns, from fixing current build issues to optimizing for 1000+ concurrent connections.

## Step 1: Fix Build Issues

### 1.1 Circular Import: pkg/game ↔ pkg/engine

**Problem:** `pkg/engine` imports `pkg/game` for the `Affectable` interface, and `pkg/game` imports `pkg/engine`.

**Solution:** Move interfaces to separate package.

Create `pkg/interfaces/affectable.go`:
```go
package interfaces

type Affectable interface {
    GetAffects() []interface{}  // Use interface{} or specific affect type
    SetAffects([]interface{})
    GetName() string
    GetID() int
    GetStrength() int
    SetStrength(int)
    // ... other stat methods
}
```

Update `pkg/engine/affect_manager.go`:
```go
import "github.com/zax0rz/darkpawns/pkg/interfaces"

type AffectManager struct {
    affects map[string][]*Affect
    entityMap map[string]interfaces.Affectable
}
```

Update `pkg/game/player.go`:
```go
import "github.com/zax0rz/darkpawns/pkg/interfaces"

// Player implements interfaces.Affectable
func (p *Player) GetAffects() []interface{} {
    // Implementation
}
```

### 1.2 Circular Import: pkg/command ↔ pkg/session

**Problem:** `pkg/command` imports `pkg/session` for `Manager`, and `pkg/session` imports `pkg/command` for command execution.

**Solution:** Use dependency injection.

Update `pkg/command/command.go`:
```go
type CommandExecutor interface {
    Execute(session interface{}, command string, args []string) error
}

var commandRegistry map[string]CommandExecutor
```

Update `pkg/session/commands.go`:
```go
import "github.com/zax0rz/darkpawns/pkg/command"

// Use command package without importing session types
func ExecuteCommand(s *Session, cmd string, args []string) error {
    executor := command.GetExecutor(cmd)
    return executor.Execute(s, cmd, args)
}
```

## Step 2: Implement Performance Monitoring

### 2.1 Add Prometheus Metrics

Create `pkg/metrics/server_metrics.go`:
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    websocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "darkpawns_websocket_connections_total",
        Help: "Current number of WebSocket connections",
    })
    
    messagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
        Name: "darkpawns_messages_processed_total",
        Help: "Total number of messages processed",
    })
    
    messageProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name: "darkpawns_message_processing_duration_seconds",
        Help: "Duration of message processing",
        Buckets: prometheus.DefBuckets,
    })
    
    databaseQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name: "darkpawns_database_query_duration_seconds",
        Help: "Duration of database queries",
        Buckets: prometheus.DefBuckets,
    })
    
    goroutineCount = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "darkpawns_goroutines_total",
        Help: "Current number of goroutines",
    })
    
    memoryAllocations = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "darkpawns_memory_allocations_bytes",
        Help: "Current memory allocations in bytes",
    })
)
```

### 2.2 Add Metrics to Key Operations

Update `pkg/session/manager.go`:
```go
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    metrics.WebsocketConnections.Inc()
    defer metrics.WebsocketConnections.Dec()
    
    start := time.Now()
    defer func() {
        metrics.MessageProcessingDuration.Observe(time.Since(start).Seconds())
    }()
    
    // Existing WebSocket handling
}
```

## Step 3: Database Optimization

### 3.1 Connection Pooling

Create `pkg/db/pool.go`:
```go
package db

import (
    "database/sql"
    "fmt"
    "sync"
    "time"
    
    _ "github.com/lib/pq"
)

type ConnectionPool struct {
    mu       sync.RWMutex
    pool     chan *sql.DB
    maxConns int
    inUse    int
}

func NewConnectionPool(maxConns int, dsn string) (*ConnectionPool, error) {
    pool := make(chan *sql.DB, maxConns)
    
    for i := 0; i < maxConns; i++ {
        db, err := sql.Open("postgres", dsn)
        if err != nil {
            return nil, err
        }
        
        // Configure connection
        db.SetMaxOpenConns(1)
        db.SetMaxIdleConns(1)
        db.SetConnMaxLifetime(5 * time.Minute)
        
        pool <- db
    }
    
    return &ConnectionPool{
        pool:     pool,
        maxConns: maxConns,
    }, nil
}

func (p *ConnectionPool) Get() (*sql.DB, error) {
    select {
    case db := <-p.pool:
        p.mu.Lock()
        p.inUse++
        p.mu.Unlock()
        return db, nil
    case <-time.After(5 * time.Second):
        return nil, fmt.Errorf("connection pool timeout")
    }
}

func (p *ConnectionPool) Put(db *sql.DB) {
    p.mu.Lock()
    p.inUse--
    p.mu.Unlock()
    p.pool <- db
}

func (p *ConnectionPool) Stats() (inUse, available int) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.inUse, len(p.pool)
}
```

### 3.2 Query Optimization

Create `pkg/db/query_optimizer.go`:
```go
package db

import (
    "time"
    "sync"
)

type QueryStats struct {
    Query      string
    Count      int64
    TotalTime  time.Duration
    AvgTime    time.Duration
    MaxTime    time.Duration
    LastCalled time.Time
}

type QueryOptimizer struct {
    mu    sync.RWMutex
    stats map[string]*QueryStats
    slowThreshold time.Duration
}

func NewQueryOptimizer(slowThreshold time.Duration) *QueryOptimizer {
    return &QueryOptimizer{
        stats: make(map[string]*QueryStats),
        slowThreshold: slowThreshold,
    }
}

func (qo *QueryOptimizer) Track(query string, duration time.Duration) {
    qo.mu.Lock()
    defer qo.mu.Unlock()
    
    stats, exists := qo.stats[query]
    if !exists {
        stats = &QueryStats{Query: query}
        qo.stats[query] = stats
    }
    
    stats.Count++
    stats.TotalTime += duration
    stats.AvgTime = stats.TotalTime / time.Duration(stats.Count)
    if duration > stats.MaxTime {
        stats.MaxTime = duration
    }
    stats.LastCalled = time.Now()
    
    // Log slow queries
    if duration > qo.slowThreshold {
        // Log or alert about slow query
    }
}

func (qo *QueryOptimizer) GetSlowQueries() []*QueryStats {
    qo.mu.RLock()
    defer qo.mu.RUnlock()
    
    var slow []*QueryStats
    for _, stats := range qo.stats {
        if stats.AvgTime > qo.slowThreshold {
            slow = append(slow, stats)
        }
    }
    return slow
}
```

## Step 4: WebSocket Optimization

### 4.1 Connection Pooling

Update `pkg/optimization/websocket.go`:
```go
type WebSocketPool struct {
    mu      sync.RWMutex
    pool    map[string]*websocket.Conn
    maxSize int
}

func NewWebSocketPool(maxSize int) *WebSocketPool {
    return &WebSocketPool{
        pool:    make(map[string]*websocket.Conn),
        maxSize: maxSize,
    }
}

func (p *WebSocketPool) Get(key string) (*websocket.Conn, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    conn, exists := p.pool[key]
    return conn, exists
}

func (p *WebSocketPool) Put(key string, conn *websocket.Conn) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if len(p.pool) >= p.maxSize {
        // Evict oldest connection
        var oldestKey string
        var oldestTime time.Time
        for k := range p.pool {
            // Implementation depends on tracking connection time
        }
        if oldestKey != "" {
            delete(p.pool, oldestKey)
        }
    }
    
    p.pool[key] = conn
    return nil
}
```

### 4.2 Message Batching

Update `pkg/optimization/websocket.go`:
```go
type BatchedSender struct {
    mu          sync.Mutex
    messages    []interface{}
    batchSize   int
    flushInterval time.Duration
    flushChan   chan struct{}
}

func NewBatchedSender(batchSize int, flushInterval time.Duration) *BatchedSender {
    bs := &BatchedSender{
        messages:    make([]interface{}, 0, batchSize),
        batchSize:   batchSize,
        flushInterval: flushInterval,
        flushChan:   make(chan struct{}, 1),
    }
    
    go bs.flushLoop()
    return bs
}

func (bs *BatchedSender) Send(message interface{}) {
    bs.mu.Lock()
    bs.messages = append(bs.messages, message)
    
    if len(bs.messages) >= bs.batchSize {
        bs.flush()
    }
    bs.mu.Unlock()
}

func (bs *BatchedSender) flush() {
    if len(bs.messages) == 0 {
        return
    }
    
    // Batch and send messages
    messages := bs.messages
    bs.messages = make([]interface{}, 0, bs.batchSize)
    
    // Send batched messages (implementation depends on WebSocket library)
}

func (bs *BatchedSender) flushLoop() {
    ticker := time.NewTicker(bs.flushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            bs.mu.Lock()
            bs.flush()
            bs.mu.Unlock()
        case <-bs.flushChan:
            return
        }
    }
}
```

## Step 5: Memory Optimization

### 5.1 Object Pooling

Create `pkg/optimization/object_pool.go`:
```go
package optimization

import (
    "sync"
)

type ObjectPool struct {
    mu     sync.Mutex
    pool   chan interface{}
    newFunc func() interface{}
    resetFunc func(interface{})
}

func NewObjectPool(maxSize int, newFunc func() interface{}, resetFunc func(interface{})) *ObjectPool {
    return &ObjectPool{
        pool:     make(chan interface{}, maxSize),
        newFunc:  newFunc,
        resetFunc: resetFunc,
    }
}

func (p *ObjectPool) Get() interface{} {
    select {
    case obj := <-p.pool:
        if p.resetFunc != nil {
            p.resetFunc(obj)
        }
        return obj
    default:
        return p.newFunc()
    }
}

func (p *ObjectPool) Put(obj interface{}) {
    select {
    case p.pool <- obj:
        // Object returned to pool
    default:
        // Pool full, object discarded
    }
}

// Example usage for Player objects
var playerPool = NewObjectPool(1000,
    func() interface{} { return &game.Player{} },
    func(obj interface{}) {
        player := obj.(*game.Player)
        // Reset player state
        player.ID = 0
        player.Name = ""
        player.Health = 0
        // ... reset other fields
    },
)
```

### 5.2 Buffer Pooling

Create `pkg/optimization/buffer_pool.go`:
```go
package optimization

import (
    "bytes"
    "sync"
)

var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func GetBuffer() *bytes.Buffer {
    return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
    buf.Reset()
    bufferPool.Put(buf)
}
```

## Step 6: Concurrency Optimization

### 6.1 Goroutine Pool

Update `pkg/optimization/pool.go`:
```go
type WorkerPool struct {
    mu       sync.Mutex
    workers  chan struct{}
    maxWorkers int
    queue    chan func()
    wg       sync.WaitGroup
}

func NewWorkerPool(maxWorkers, queueSize int) *WorkerPool {
    wp := &WorkerPool{
        workers:    make(chan struct{}, maxWorkers),
        maxWorkers: maxWorkers,
        queue:      make(chan func(), queueSize),
    }
    
    // Start dispatcher
    go wp.dispatcher()
    
    return wp
}

func (wp *WorkerPool) Submit(task func()) error {
    select {
    case wp.queue <- task:
        return nil
    default:
        return fmt.Errorf("worker pool queue full")
    }
}

func (wp *WorkerPool) dispatcher() {
    for task := range wp.queue {
        wp.wg.Add(1)
        wp.workers <- struct{}{} // Acquire worker slot
        
        go func(t func()) {
            defer func() {
                <-wp.workers // Release worker slot
                wp.wg.Done()
            }()
            
            t()
        }(task)
    }
}

func (wp *WorkerPool) Wait() {
    wp.wg.Wait()
}

func (wp *WorkerPool) Stats() (active, queued int) {
    wp.mu.Lock()
    defer wp.mu.Unlock()
    return len(wp.workers), len(wp.queue)
}
```

## Step 7: Load Testing Configuration

### 7.1 Load Test Scenarios

Create `load_test/scenarios.yaml`:
```yaml
scenarios:
  - name: "baseline-100"
    clients: 100
    duration: "30s"
    messages_per_sec: 2
    commands:
      - "look"
      - "north"
      - "south"
      - "east"
      - "west"
      - "say hello"
      - "stats"
  
  - name: "medium-500"
    clients: 500
    duration: "60s"
    messages_per_sec: 1
    commands: ["look", "north", "say test"]
  
  - name: "stress-1000"
    clients: 1000
    duration: "120s"
    messages_per_sec: 0.5
    commands: ["look", "say stress"]
  
  - name: "mixed-workload"
    clients: 300
    duration: "180s"
    messages_per_sec: 3
    command_mix:
      movement: 40%
      chat: 30%
      combat: 20%
      inventory: 10%
```

### 7.2 Automated Load Test Runner

Create `scripts/run_load_test.sh`:
```bash
#!/bin/bash

# Dark Pawns Load Test Runner
set -e

SERVER_URL="ws://localhost:8080/ws"
RESULTS_DIR="load-test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$RESULTS_DIR"

echo "Starting Dark Pawns load tests..."
echo "Timestamp: $TIMESTAMP"
echo ""

# Test 1: Baseline (100 clients)
echo "=== Test 1: Baseline (100 clients) ==="
go run load_test/load_test.go \
  -url "$SERVER_URL" \
  -clients 100 \
  -duration 30s \
  -rate 2 \
  > "$RESULTS_DIR/baseline_${TIMESTAMP}.log"

# Test 2: Medium load (500 clients)
echo "=== Test 2: Medium load (500 clients) ==="
go run load_test/load_test.go \
  -url "$SERVER_URL" \
  -clients 500 \
  -duration 60s \
  -rate 1 \
  > "$RESULTS_DIR/medium_${TIMESTAMP}.log"

# Test 3: Stress test (1000 clients)
echo "=== Test 3: Stress test (1000 clients) ==="
go run load_test/load_test.go \
  -url "$SERVER_URL" \
  -clients 1000 \
  -duration 120s \
  -rate 0.5 \
  > "$RESULTS_DIR/stress_${TIMESTAMP}.log"

echo ""
echo "Load tests completed!"
echo "Results saved to: $RESULTS_DIR/"
echo ""
echo "=== Summary ==="
grep -E "(Total Clients|Throughput|Avg Latency|P95|P99|Success Rate)" "$RESULTS_DIR"/*.log
```

## Step 8: Monitoring Setup

### 8.1 Docker Compose Monitoring

Update `docker-compose.monitoring.yml`:
```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    depends_on:
      - prometheus

  node-exporter:
    image: prom/node-exporter:latest
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'

volumes:
  prometheus_data:
  grafana_data:
```

### 8.2 Prometheus Configuration

Create `prometheus/prometheus.yml`:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'darkpawns'
    static_configs:
      - targets: ['server:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:9121']
```

## Step 9: Performance Validation

### 9.1 Success Criteria

Define performance targets in `tests/performance/benchmarks.go`:
```go
package performance

var PerformanceTargets = struct {
    WebSocketConnections int
    MessageThroughput    int // messages/sec
    P95Latency          time.Duration
    P99Latency          time.Duration
    MemoryPerConnection int // MB
    CPUUtilization      float64 // percentage
}{
    WebSocketConnections: 1000,
    MessageThroughput:    500,
    P95Latency:          100 * time.Millisecond,
    P99Latency:          250 * time.Millisecond,
    MemoryPerConnection:  5, // MB
    CPUUtilization:      70.0,
}

func ValidatePerformance(results LoadTestResults) []string {
    var failures []string
    
    if results.TotalClients < PerformanceTargets.WebSocketConnections {
        failures = append(failures, 
            fmt.Sprintf("Failed to reach target connections: %d < %d",
                results.TotalClients, PerformanceTargets.WebSocketConnections))
    }
    
    if results.Throughput < float64(PerformanceTargets.MessageThroughput) {
        failures = append(failures,
            fmt.Sprintf("Throughput below target: %.2f < %d",
                results.Throughput, PerformanceTargets.MessageThroughput))
    }
    
    if results.P95Latency > PerformanceTargets.P95Latency {
        failures = append(failures,
            fmt.Sprintf("P95 latency above target: %v > %v",
                results.P95Latency, PerformanceTargets.P95Latency))
    }
    
    return failures
}
```

### 9.2 Continuous Performance Testing

Create `.github/workflows/performance.yml`:
```yaml
name: Performance Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM

jobs:
  performance:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: darkpawns
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
      
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.25'
    
    - name: Build
      run: make build
    
    - name: Start server
      run: |
        ./darkpawns -world ../darkpawns/lib -port 8080 &
        sleep 10
    
    - name: Run load tests
      run: |
        go run load_test/load_test.go -url "ws://localhost:8080/ws" -clients 100 -duration 30s
        go run load_test/load_test.go -url "ws://localhost:8080/ws" -clients 500 -duration 60s
    
    - name: Validate performance
      run: |
        go test ./tests/performance -v
```

## Conclusion

This performance tuning guide provides a comprehensive approach to optimizing Dark Pawns for 1000+ concurrent connections. The key steps are:

1. **Fix build issues** - Resolve circular imports
2. **Implement monitoring** - Add Prometheus metrics
3. **Optimize database** - Connection pooling, query optimization
4. **Optimize WebSocket** - Connection pooling, message batching
5. **Optimize memory** - Object pooling, buffer pooling
6. **Optimize concurrency** - Goroutine pools, worker pools
7. **Implement load testing** - Automated test scenarios
8. **Set up monitoring** - Prometheus + Grafana dashboard
9. **Validate performance** - Automated performance validation

By following these steps, Dark Pawns can achieve the target performance of 1000+ concurrent connections with acceptable latency and resource utilization.