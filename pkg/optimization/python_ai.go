package optimization

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// AIRequest represents a request to the AI service.
type AIRequest struct {
	ID          string
	Prompt      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timestamp   time.Time
}

// AIResponse represents a response from the AI service.
type AIResponse struct {
	ID       string
	Text     string
	Tokens   int
	Latency  time.Duration
	Model    string
	Metadata map[string]interface{}
}

// AIBatchProcessor batches AI requests for efficiency.
type AIBatchProcessor struct {
	mu          sync.Mutex
	wg          sync.WaitGroup // tracks in-flight Submit calls
	batchSize   int
	maxWaitTime time.Duration
	batch       []AIBatchItem
	processFunc func([]AIBatchItem) error
	timer       *time.Timer
	closed      bool
}

// AIBatchItem represents an item in an AI batch.
type AIBatchItem struct {
	Request  AIRequest
	Response chan AIResponse
	Error    chan error
}

// NewAIBatchProcessor creates a new AI batch processor.
func NewAIBatchProcessor(batchSize int, maxWaitTime time.Duration, processFunc func([]AIBatchItem) error) *AIBatchProcessor {
	bp := &AIBatchProcessor{
		batchSize:   batchSize,
		maxWaitTime: maxWaitTime,
		batch:       make([]AIBatchItem, 0, batchSize),
		processFunc: processFunc,
	}

	bp.timer = time.AfterFunc(maxWaitTime, bp.processBatch)
	return bp
}

// Submit submits an AI request for batch processing.
// The context governs how long the caller is willing to wait for a response.
func (bp *AIBatchProcessor) Submit(ctx context.Context, req AIRequest) (AIResponse, error) {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return AIResponse{}, ErrPoolClosed
	}

	// Counted from here (under the lock, before closed can flip) until
	// this call returns, so Wait/Close can't observe the counter hit zero
	// while a new request is still being admitted.
	bp.wg.Add(1)
	defer bp.wg.Done()

	item := AIBatchItem{
		Request:  req,
		Response: make(chan AIResponse, 1),
		Error:    make(chan error, 1),
	}

	bp.batch = append(bp.batch, item)

	shouldProcess := len(bp.batch) >= bp.batchSize

	if !shouldProcess {
		// Reset timer
		bp.timer.Stop()
		bp.timer.Reset(bp.maxWaitTime)
		bp.mu.Unlock()
	} else {
		batch := bp.batch
		bp.batch = make([]AIBatchItem, 0, bp.batchSize)
		bp.mu.Unlock()

		// Process batch
		if err := bp.processFunc(batch); err != nil {
			bp.sendBatchError(batch, err)
			return AIResponse{}, err
		}

		// Wait for response
		select {
		case resp := <-item.Response:
			return resp, nil
		case err := <-item.Error:
			return AIResponse{}, err
		case <-ctx.Done():
			return AIResponse{}, ctx.Err()
		}
	}

	// Wait for batch processing
	select {
	case resp := <-item.Response:
		return resp, nil
	case err := <-item.Error:
		return AIResponse{}, err
	case <-ctx.Done():
		return AIResponse{}, ctx.Err()
	}
}

// sendBatchError fans an error out to every item in a batch.
func (bp *AIBatchProcessor) sendBatchError(batch []AIBatchItem, err error) {
	for _, item := range batch {
		select {
		case item.Error <- err:
		default:
		}
	}
}

// processBatch processes the current batch.
func (bp *AIBatchProcessor) processBatch() {
	bp.mu.Lock()
	if bp.closed {
		// Close already flushed (or is about to flush) whatever was
		// pending; don't resurrect the timer or double-process the batch.
		bp.mu.Unlock()
		return
	}
	if len(bp.batch) == 0 {
		bp.timer.Reset(bp.maxWaitTime)
		bp.mu.Unlock()
		return
	}

	batch := bp.batch
	bp.batch = make([]AIBatchItem, 0, bp.batchSize)
	bp.mu.Unlock()

	// Process batch
	if err := bp.processFunc(batch); err != nil {
		bp.sendBatchError(batch, err)
	}
}

// Wait blocks until every request submitted before the call to Wait has
// either received a response or been cancelled via its context. It does
// not stop the processor from accepting new submissions.
func (bp *AIBatchProcessor) Wait() {
	bp.wg.Wait()
}

// Close gracefully shuts down the batch processor. It stops accepting new
// submissions, flushes any items still sitting in the pending batch, and
// waits for in-flight Submit calls to observe their response before
// returning. This guarantees no submitted request is silently dropped and
// that Close() is safe to call immediately before tearing down resources
// the processor depends on.
func (bp *AIBatchProcessor) Close() error {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return nil
	}
	bp.closed = true
	bp.timer.Stop()

	// Flush any remaining items synchronously so Close() is guaranteed
	// to complete before the caller shuts down resources the AI processor
	// depends on (a fire-and-forget goroutine would race with shutdown).
	var err error
	if len(bp.batch) > 0 {
		batch := bp.batch
		bp.batch = nil
		bp.mu.Unlock()

		if pErr := bp.processFunc(batch); pErr != nil {
			bp.sendBatchError(batch, pErr)
			err = pErr
		}
	} else {
		bp.mu.Unlock()
	}

	// Wait for any Submit calls that already pulled a full batch out and
	// handed it to processFunc themselves (see the shouldProcess branch in
	// Submit) to finish observing their response before we return.
	bp.wg.Wait()

	return err
}

// AICache provides caching for AI responses.
type AICache struct {
	mu      sync.RWMutex
	cache   map[string]CacheEntry
	maxSize int
	ttl     time.Duration
}

// CacheEntry represents a cached AI response.
type CacheEntry struct {
	Response  AIResponse
	Timestamp time.Time
	Hits      int
}

// NewAICache creates a new AI cache.
func NewAICache(maxSize int, ttl time.Duration) *AICache {
	return &AICache{
		cache:   make(map[string]CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// GenerateCacheKey generates a cache key for an AI request.
func (ac *AICache) GenerateCacheKey(req AIRequest) string {
	// Simple key generation - in production, use a proper hash
	keyData := map[string]interface{}{
		"prompt":     req.Prompt,
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"temp":       req.Temperature,
	}

	keyBytes, err := json.Marshal(keyData)
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return ""
	}
	return string(keyBytes)
}

// Get retrieves a cached response. The write lock is held for the entire
// read-modify-write window so a concurrent Set/Delete cannot replace or
// remove the entry between the lookup and the hit-count write-back (a TOCTOU
// that would let a stale Get overwrite a newer entry).
func (ac *AICache) Get(key string) (AIResponse, bool) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	entry, exists := ac.cache[key]
	if !exists {
		return AIResponse{}, false
	}

	// Check TTL
	if time.Since(entry.Timestamp) > ac.ttl {
		delete(ac.cache, key)
		return AIResponse{}, false
	}

	// Update hit count
	entry.Hits++
	ac.cache[key] = entry

	return entry.Response, true
}

// Set stores a response in the cache.
func (ac *AICache) Set(key string, resp AIResponse) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Evict if cache is full
	if len(ac.cache) >= ac.maxSize {
		ac.evictOne()
	}

	ac.cache[key] = CacheEntry{
		Response:  resp,
		Timestamp: time.Now(),
		Hits:      0,
	}
}

// evictOne evicts one entry from the cache (LRU approximation).
func (ac *AICache) evictOne() {
	var oldestKey string
	var oldestTime time.Time
	minHits := int(^uint(0) >> 1) // Max int

	// Simple eviction: find entry with fewest hits
	for key, entry := range ac.cache {
		if entry.Hits < minHits {
			minHits = entry.Hits
			oldestKey = key
			oldestTime = entry.Timestamp
		} else if entry.Hits == minHits && entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(ac.cache, oldestKey)
	}
}

// Stats returns cache statistics.
func (ac *AICache) Stats() map[string]interface{} {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["size"] = len(ac.cache)
	stats["max_size"] = ac.maxSize
	stats["ttl"] = ac.ttl

	var totalHits int
	now := time.Now()
	expiredCount := 0

	for _, entry := range ac.cache {
		totalHits += entry.Hits
		if now.Sub(entry.Timestamp) > ac.ttl {
			expiredCount++
		}
	}

	if len(ac.cache) > 0 {
		stats["avg_hits"] = float64(totalHits) / float64(len(ac.cache))
	}
	stats["expired_count"] = expiredCount

	return stats
}

// Clear clears the cache.
func (ac *AICache) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cache = make(map[string]CacheEntry)
}

// AsyncProcessor provides asynchronous AI request processing.
type AsyncProcessor struct {
	mu             sync.Mutex
	workerPool     *WorkerPool
	cache          *AICache
	batchProcessor *AIBatchProcessor
	timeout        time.Duration
}

// NewAsyncProcessor creates a new async AI processor.
func NewAsyncProcessor(workers int, cacheSize int, cacheTTL time.Duration, timeout time.Duration) *AsyncProcessor {
	ap := &AsyncProcessor{
		workerPool: NewWorkerPool(workers),
		cache:      NewAICache(cacheSize, cacheTTL),
		timeout:    timeout,
	}

	// Initialize batch processor
	ap.batchProcessor = NewAIBatchProcessor(10, 100*time.Millisecond, ap.processBatch)

	return ap
}

// Process processes an AI request asynchronously.
func (ap *AsyncProcessor) Process(req AIRequest, callback func(AIResponse, error)) error {
	// Check cache first
	cacheKey := ap.cache.GenerateCacheKey(req)
	if resp, hit := ap.cache.Get(cacheKey); hit {
		callback(resp, nil)
		return nil
	}

	// Submit to worker pool for async processing
	return ap.workerPool.Submit(func() {
		ctx, cancel := context.WithTimeout(context.Background(), ap.timeout)
		defer cancel()

		// Process through batch processor
		resp, err := ap.batchProcessor.Submit(ctx, req)
		if err != nil {
			callback(AIResponse{}, err)
			return
		}

		// Cache the response
		ap.cache.Set(cacheKey, resp)

		callback(resp, nil)
	})
}

// processBatch processes a batch of AI requests.
func (ap *AsyncProcessor) processBatch(batch []AIBatchItem) error {
	// In a real implementation, this would call the AI API
	// For now, simulate processing
	for _, item := range batch {
		resp := AIResponse{
			ID:      item.Request.ID,
			Text:    "Simulated response for: " + item.Request.Prompt[:min(20, len(item.Request.Prompt))] + "...",
			Tokens:  50,
			Latency: 100 * time.Millisecond,
			Model:   item.Request.Model,
		}

		select {
		case item.Response <- resp:
		default:
		}
	}

	return nil
}

// Close gracefully shuts down the async processor.
func (ap *AsyncProcessor) Close() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	_ = ap.batchProcessor.Close()
	ap.workerPool.Close()

	return nil
}
