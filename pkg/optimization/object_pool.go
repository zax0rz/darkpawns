package optimization

import (
	"sync"
	"time"
)

// ObjectPool provides reusable object pooling
type ObjectPool struct {
	mu       sync.Mutex
	pool     []interface{}
	create   func() interface{}
	reset    func(interface{})
	validate func(interface{}) bool
	maxSize  int
	created  int
	borrowed int
	stats    PoolStats
}

// PoolStats tracks object pool statistics
type PoolStats struct {
	TotalCreated   int64
	TotalBorrowed  int64
	TotalReturned  int64
	TotalDiscarded int64
	PeakSize       int
	AvgBorrowTime  time.Duration
	MaxBorrowTime  time.Duration
	LastReset      time.Time
}

// NewObjectPool creates a new object pool
func NewObjectPool(create func() interface{}, reset func(interface{}), validate func(interface{}) bool, maxSize int) *ObjectPool {
	return &ObjectPool{
		pool:     make([]interface{}, 0, maxSize),
		create:   create,
		reset:    reset,
		validate: validate,
		maxSize:  maxSize,
		stats: PoolStats{
			LastReset: time.Now(),
		},
	}
}

// Get borrows an object from the pool
func (op *ObjectPool) Get() interface{} {
	start := time.Now()
	op.mu.Lock()
	defer op.mu.Unlock()

	op.stats.TotalBorrowed++
	op.borrowed++

	// Return from pool if available
	if len(op.pool) > 0 {
		obj := op.pool[len(op.pool)-1]
		op.pool = op.pool[:len(op.pool)-1]

		// Validate object if validation function provided
		if op.validate != nil && !op.validate(obj) {
			// Object invalid, discard and create new one
			op.stats.TotalDiscarded++
			return op.createNewLocked()
		}

		// Reset object if reset function provided
		if op.reset != nil {
			op.reset(obj)
		}

		borrowTime := time.Since(start)
		op.updateBorrowTime(borrowTime)

		return obj
	}

	// Create new object if under max size
	if op.created < op.maxSize {
		obj := op.createNewLocked()
		borrowTime := time.Since(start)
		op.updateBorrowTime(borrowTime)
		return obj
	}

	// Pool exhausted, create temporary object
	obj := op.create()
	borrowTime := time.Since(start)
	op.updateBorrowTime(borrowTime)

	return obj
}

// createNewLocked creates a new object (caller must hold lock)
func (op *ObjectPool) createNewLocked() interface{} {
	obj := op.create()
	op.created++
	op.stats.TotalCreated++

	if op.created > op.stats.PeakSize {
		op.stats.PeakSize = op.created
	}

	return obj
}

// updateBorrowTime updates borrow time statistics
func (op *ObjectPool) updateBorrowTime(borrowTime time.Duration) {
	if borrowTime > op.stats.MaxBorrowTime {
		op.stats.MaxBorrowTime = borrowTime
	}

	// Update average borrow time
	totalBorrows := op.stats.TotalBorrowed
	if totalBorrows > 0 {
		op.stats.AvgBorrowTime = (op.stats.AvgBorrowTime*time.Duration(totalBorrows-1) + borrowTime) / time.Duration(totalBorrows)
	}
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.stats.TotalReturned++
	op.borrowed--

	// Validate object if validation function provided
	if op.validate != nil && !op.validate(obj) {
		op.stats.TotalDiscarded++
		return
	}

	// Reset object if reset function provided
	if op.reset != nil {
		op.reset(obj)
	}

	// Add to pool if not full
	if len(op.pool) < op.maxSize {
		op.pool = append(op.pool, obj)
	} else {
		// Pool is full, discard object
		op.stats.TotalDiscarded++
	}
}

// TryGet attempts to get an object without waiting
func (op *ObjectPool) TryGet() (interface{}, bool) {
	op.mu.Lock()
	defer op.mu.Unlock()

	if len(op.pool) == 0 && op.created >= op.maxSize {
		return nil, false
	}

	return op.Get(), true
}

// Clear removes all objects from the pool
func (op *ObjectPool) Clear() {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.pool = make([]interface{}, 0, op.maxSize)
	op.created = 0
	op.borrowed = 0
	op.stats.LastReset = time.Now()
}

// Stats returns pool statistics
func (op *ObjectPool) Stats() map[string]interface{} {
	op.mu.Lock()
	defer op.mu.Unlock()

	stats := make(map[string]interface{})
	stats["pool_size"] = len(op.pool)
	stats["created"] = op.created
	stats["borrowed"] = op.borrowed
	stats["max_size"] = op.maxSize
	stats["available"] = len(op.pool)
	stats["in_use"] = op.borrowed

	if op.created > 0 {
		stats["utilization"] = float64(op.borrowed) / float64(op.created)
		stats["efficiency"] = float64(op.stats.TotalReturned) / float64(op.stats.TotalBorrowed)
	}

	// Copy detailed stats
	stats["total_created"] = op.stats.TotalCreated
	stats["total_borrowed"] = op.stats.TotalBorrowed
	stats["total_returned"] = op.stats.TotalReturned
	stats["total_discarded"] = op.stats.TotalDiscarded
	stats["peak_size"] = op.stats.PeakSize
	stats["avg_borrow_time_ms"] = op.stats.AvgBorrowTime.Milliseconds()
	stats["max_borrow_time_ms"] = op.stats.MaxBorrowTime.Milliseconds()
	stats["last_reset"] = op.stats.LastReset.Format(time.RFC3339)

	return stats
}

// Prefill creates initial objects in the pool
func (op *ObjectPool) Prefill(count int) {
	op.mu.Lock()
	defer op.mu.Unlock()

	for i := 0; i < count && len(op.pool) < op.maxSize; i++ {
		obj := op.create()
		op.created++
		op.stats.TotalCreated++
		op.pool = append(op.pool, obj)
	}

	if op.created > op.stats.PeakSize {
		op.stats.PeakSize = op.created
	}
}

// StringPool provides string pooling to reduce allocations
type StringPool struct {
	mu   sync.RWMutex
	pool map[string]string
}

// NewStringPool creates a new string pool
func NewStringPool() *StringPool {
	return &StringPool{
		pool: make(map[string]string),
	}
}

// Get returns a pooled string or adds it to the pool
func (sp *StringPool) Get(s string) string {
	sp.mu.RLock()
	pooled, exists := sp.pool[s]
	sp.mu.RUnlock()

	if exists {
		return pooled
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Double-check after acquiring write lock
	if pooled, exists := sp.pool[s]; exists {
		return pooled
	}

	// Add to pool
	sp.pool[s] = s
	return s
}

// Size returns number of strings in pool
func (sp *StringPool) Size() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return len(sp.pool)
}

// Clear removes all strings from pool
func (sp *StringPool) Clear() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.pool = make(map[string]string)
}
