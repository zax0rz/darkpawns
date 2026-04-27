# Dark Pawns Performance Tuning Guide

## Overview

This guide covers performance optimization techniques for the Dark Pawns game server, focusing on Go server optimization, Python AI processing, database performance, WebSocket communication, and load testing.

## Table of Contents

1. [Go Server Optimization](#go-server-optimization)
2. [Python AI Optimization](#python-ai-optimization)
3. [Database Optimization](#database-optimization)
4. [WebSocket Optimization](#websocket-optimization)
5. [Load Testing](#load-testing)
6. [Profiling](#profiling)
7. [Monitoring](#monitoring)
8. [Best Practices](#best-practices)

## Go Server Optimization

### Goroutine Pooling

Use the `WorkerPool` from `pkg/optimization/pool.go` to manage goroutines:

```go
import "github.com/zax0rz/darkpawns/pkg/optimization"

// Create a worker pool with 10 workers
pool := optimization.NewWorkerPool(10)
defer pool.Close()

// Submit tasks
pool.Submit(func() {
    // Process task
})
```

**Benefits:**
- Limits concurrent goroutines
- Prevents goroutine explosion
- Provides backpressure

### Connection Pooling

Use the `ConnectionPool` for database connections:

```go
pool := optimization.NewConnectionPool(
    10, // max connections
    30*time.Second, // idle timeout
    createFunc, // connection creation function
    closeFunc, // connection close function
)
defer pool.Close()

// Get connection
conn, err := pool.Get()
if err != nil {
    // handle error
}
defer pool.Put(conn)
```

### Memory Management

1. **Object Pooling:** Reuse objects instead of allocating new ones
2. **Buffer Pooling:** Use `sync.Pool` for frequently allocated buffers
3. **Slice Reuse:** Pre-allocate slices with capacity

```go
// Example: Buffer pool
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 1024))
    },
}

func getBuffer() *bytes.Buffer {
    return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
    buf.Reset()
    bufferPool.Put(buf)
}
```

## Python AI Optimization

### Async Processing

Use the `AsyncAIProcessor` for efficient AI request handling:

```python
from scripts.ai_optimizer import AsyncAIProcessor, AIRequest
import asyncio

processor = AsyncAIProcessor(cache_size=1000, batch_size=10)

async def process_request(prompt: str):
    request = AIRequest(
        request_id="unique-id",
        prompt=prompt,
        model="gpt-3.5-turbo"
    )
    response = await processor.process(request)
    return response.text
```

### Batch Processing

Batch AI requests to reduce API calls:

```python
# The AsyncAIProcessor automatically batches requests
# Configure batch size based on your needs:
processor = AsyncAIProcessor(batch_size=10)  # Batch 10 requests
```

### Caching

Cache AI responses to avoid duplicate processing:

```python
# Cache is built into AsyncAIProcessor
# Monitor cache hit rate:
stats = processor.stats()
print(f"Cache hit rate: {stats['hit_rate']:.2%}")
```

**Cache Tuning:**
- Increase cache size for frequently repeated prompts
- Adjust TTL based on response freshness requirements
- Monitor cache effectiveness with stats

## Database Optimization

### Query Optimization

Use the `QueryOptimizer` to identify slow queries:

```go
optimizer := optimization.NewQueryOptimizer(1000, 100*time.Millisecond)

// Record query execution
start := time.Now()
// Execute query
duration := time.Since(start)
optimizer.RecordQuery(query, duration, indexUsed)

// Get slow queries
slowQueries := optimizer.GetSlowQueries()
```

### Index Analysis

Use the `IndexAnalyzer` to identify missing indexes:

```go
analyzer := optimization.NewIndexAnalyzer(db)
recommendations, err := analyzer.AnalyzeTable("players")
```

### Batch Operations

Use the `BatchProcessor` for bulk database operations:

```go
processor := optimization.NewBatchProcessor(
    100, // batch size
    1*time.Second, // flush interval
    flushFunc, // batch processing function
)
defer processor.Close()

// Add operations
processor.Add(optimization.BatchOperation{
    Type: "insert",
    Table: "players",
    Data: playerData,
})
```

## WebSocket Optimization

### Compression

Enable WebSocket compression to reduce bandwidth:

```go
import "github.com/zax0rz/darkpawns/pkg/optimization"

// Create compressed WebSocket
cws := optimization.NewCompressedWebSocket(conn, gzip.DefaultCompression)
defer cws.Close()

// Use compressed read/write
cws.WriteMessage(websocket.TextMessage, data)
```

### Message Batching

Use the `BatchedSender` to batch WebSocket messages:

```go
sender := optimization.NewBatchedSender(
    50*time.Millisecond, // batch window
    100, // max batch size
    sendFunc, // batch sending function
)
defer sender.Close()

// Send messages (will be batched)
sender.Send(json.RawMessage(`{"type":"update"}`))
```

### Backpressure Management

Monitor WebSocket backpressure with `BackpressureMonitor`:

```go
monitor := optimization.NewBackpressureMonitor(256, 0.8) // 80% threshold

// Update buffer size
monitor.Update(sessionID, bufferSize)

// Check for backpressure
problematic := monitor.CheckBackpressure()
```

## Load Testing

### Running Load Tests

Use the load test scripts to simulate concurrent users:

```bash
# Run simple load test
cd load_test
go run load_test.go

# Run comprehensive test suite
go run load_test.go --clients 1000 --duration 5m
```

### Test Scenarios

1. **Connection Storm:** Rapid connection/disconnection
2. **Message Flood:** High message rate per client
3. **Broadcast Storm:** Many broadcast messages
4. **Mixed Workload:** Realistic mix of operations

### Interpreting Results

Key metrics to monitor:
- **Throughput:** Messages per second
- **Latency:** P95, P99 response times
- **Error Rate:** Percentage of failed operations
- **Memory Usage:** Heap allocation over time
- **Goroutine Count:** Concurrent goroutines

## Profiling

### CPU Profiling

```bash
# Start CPU profiling
./profiling/profiler monitor ./profiles 30s

# Analyze with pprof
go tool pprof ./profiles/cpu-*.prof
```

### Memory Profiling

```bash
# Take heap snapshot
./profiling/profiler stats

# Generate heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
```

### Goroutine Analysis

```bash
# Dump goroutines
./profiling/profiler goroutine

# Analyze goroutine leaks
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Monitoring

### Metrics Collection

Dark Pawns includes Prometheus metrics at `/metrics`:

```bash
# Query metrics
curl http://localhost:8080/metrics
```

Key metrics to monitor:
- `darkpawns_sessions_active`: Active WebSocket sessions
- `darkpawns_messages_total`: Total messages processed
- `darkpawns_commands_total`: Commands by type
- `darkpawns_combat_rounds_total`: Combat rounds processed
- `go_goroutines`: Number of goroutines
- `go_memstats_alloc_bytes`: Allocated memory

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: darkpawns
    rules:
      - alert: HighErrorRate
        expr: rate(darkpawns_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          
      - alert: MemoryLeak
        expr: increase(go_memstats_heap_alloc_bytes[1h]) > 100e6
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Possible memory leak detected"
```

## Best Practices

### General Guidelines

1. **Measure First:** Profile before optimizing
2. **Optimize Hot Paths:** Focus on frequently executed code
3. **Use Appropriate Data Structures:** Choose maps vs slices based on access patterns
4. **Avoid Premature Optimization:** Optimize only when needed

### Go-Specific Tips

1. **Use `sync.Pool`** for frequently allocated objects
2. **Pre-allocate slices** with known capacity
3. **Use buffered channels** to prevent blocking
4. **Limit goroutine creation** with worker pools
5. **Use `pprof`** for continuous profiling

### Python-Specific Tips

1. **Use async/await** for I/O-bound operations
2. **Batch API calls** to reduce overhead
3. **Implement caching** at multiple levels
4. **Use connection pooling** for database/API connections
5. **Monitor memory usage** with `tracemalloc`

### Database Tips

1. **Use indexes** on frequently queried columns
2. **Batch write operations** where possible
3. **Monitor slow queries** regularly
4. **Use connection pooling**
5. **Consider read replicas** for heavy read workloads

### WebSocket Tips

1. **Enable compression** for text-heavy applications
2. **Implement message batching**
3. **Monitor backpressure**
4. **Use heartbeats** to detect dead connections
5. **Limit message size** to prevent DoS

## Troubleshooting

### Common Issues

1. **High Memory Usage:**
   - Check for goroutine leaks
   - Review object allocation patterns
   - Enable memory profiling

2. **Slow Response Times:**
   - Profile CPU usage
   - Check database query performance
   - Review WebSocket message handling

3. **Connection Issues:**
   - Monitor WebSocket buffer sizes
   - Check network latency
   - Review rate limiting settings

4. **Database Bottlenecks:**
   - Analyze query execution plans
   - Check index usage
   - Monitor connection pool utilization

### Performance Checklist

- [ ] CPU profiling enabled
- [ ] Memory profiling enabled
- [ ] Database indexes optimized
- [ ] WebSocket compression enabled
- [ ] Connection pooling implemented
- [ ] Rate limiting configured
- [ ] Monitoring and alerting set up
- [ ] Load testing performed
- [ ] Performance baselines established

## Conclusion

Performance optimization is an ongoing process. Regularly profile your application, monitor key metrics, and test under load to ensure Dark Pawns can scale to meet demand.

For more information, refer to:
- [Go Performance Wiki](https://github.com/golang/go/wiki/Performance)
- [Python Performance Tips](https://wiki.python.org/moin/PythonSpeed/PerformanceTips)
- [PostgreSQL Performance Tips](https://wiki.postgresql.org/wiki/Performance_Optimization)