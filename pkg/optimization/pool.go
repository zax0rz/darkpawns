// Package optimization provides performance optimizations for Dark Pawns.
package optimization

import (
	"sync"
	"time"
)

// WorkerPool manages a pool of goroutines for concurrent task processing.
type WorkerPool struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup // tracks the worker goroutines themselves
	taskWg    sync.WaitGroup // tracks submitted-but-not-yet-completed tasks
	mu        sync.RWMutex
	closed    bool
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
func NewWorkerPool(workers int) *WorkerPool {
	pool := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), workers*10), // Buffer size 10x workers
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker processes tasks from the queue.
func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for task := range p.taskQueue {
		task()
		p.taskWg.Done()
	}
}

// Submit adds a task to the pool. The task is counted by Wait until it has
// finished executing.
func (p *WorkerPool) Submit(task func()) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPoolClosed
	}

	p.taskWg.Add(1)
	select {
	case p.taskQueue <- task:
		return nil
	default:
		p.taskWg.Done()
		return ErrPoolFull
	}
}

// Wait blocks until every task submitted before the call has finished
// running. Unlike Close, it does not stop the pool from accepting further
// work, so callers can use it to synchronize on a batch of submitted tasks
// without tearing the pool down.
func (p *WorkerPool) Wait() {
	p.taskWg.Wait()
}

// Close stops the pool from accepting new work, then blocks until every
// task already queued or in flight has completed before returning. It is
// always safe to call Close directly (without calling Wait first) since it
// drains the queue itself.
func (p *WorkerPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.taskQueue)
	p.wg.Wait()
}

// ConnectionPool manages database connection pooling.
type ConnectionPool struct {
	mu          sync.RWMutex
	connections []interface{}
	maxSize     int
	idleTimeout time.Duration
	createFunc  func() (interface{}, error)
	closeFunc   func(interface{}) error
	stats       ConnectionPoolStats
}

// ConnectionPoolStats holds connection pool statistics.
type ConnectionPoolStats struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	WaitCount         int64
	WaitDuration      time.Duration
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(
	maxSize int,
	idleTimeout time.Duration,
	createFunc func() (interface{}, error),
	closeFunc func(interface{}) error,
) *ConnectionPool {
	return &ConnectionPool{
		connections: make([]interface{}, 0, maxSize),
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		createFunc:  createFunc,
		closeFunc:   closeFunc,
	}
}

// Get acquires a connection from the pool.
func (p *ConnectionPool) Get() (interface{}, error) {
	start := time.Now()

	p.mu.Lock()
	p.stats.WaitCount++

	// Try to get an idle connection
	if len(p.connections) > 0 {
		conn := p.connections[len(p.connections)-1]
		p.connections = p.connections[:len(p.connections)-1]
		p.stats.ActiveConnections++
		p.stats.IdleConnections = len(p.connections)
		p.stats.WaitDuration += time.Since(start)
		p.mu.Unlock()
		return conn, nil
	}

	// Create new connection if under max size
	// Release lock during creation to avoid blocking other Get/Put calls
	if p.stats.TotalConnections < p.maxSize {
		p.stats.TotalConnections++
		p.stats.ActiveConnections++
		p.mu.Unlock()
		conn, err := p.createFunc()
		if err != nil {
			p.mu.Lock()
			p.stats.TotalConnections--
			p.stats.ActiveConnections--
			p.stats.WaitDuration += time.Since(start)
			p.mu.Unlock()
			return nil, err
		}
		p.mu.Lock()
		p.stats.WaitDuration += time.Since(start)
		p.mu.Unlock()
		return conn, nil
	}

	p.stats.WaitDuration += time.Since(start)
	p.mu.Unlock()
	return nil, ErrPoolExhausted
}

// Put returns a connection to the pool.
func (p *ConnectionPool) Put(conn interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.ActiveConnections--

	// If pool is full or connection is bad, close it
	if len(p.connections) >= p.maxSize {
		return p.closeFunc(conn)
	}

	p.connections = append(p.connections, conn)
	p.stats.IdleConnections = len(p.connections)
	return nil
}

// Stats returns current pool statistics.
func (p *ConnectionPool) Stats() ConnectionPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	for _, conn := range p.connections {
		if closeErr := p.closeFunc(conn); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	p.connections = nil
	p.stats = ConnectionPoolStats{}

	return err
}

// WebSocketPool manages WebSocket connection pooling for broadcast operations.
type WebSocketPool struct {
	mu         sync.RWMutex
	sessions   map[string]chan []byte
	bufferSize int
}

// NewWebSocketPool creates a new WebSocket pool.
func NewWebSocketPool(bufferSize int) *WebSocketPool {
	return &WebSocketPool{
		sessions:   make(map[string]chan []byte),
		bufferSize: bufferSize,
	}
}

// Register adds a session to the pool.
func (p *WebSocketPool) Register(sessionID string, sendChan chan []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessions[sessionID] = sendChan
}

// Unregister removes a session from the pool.
func (p *WebSocketPool) Unregister(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, sessionID)
}

// Broadcast sends a message to all sessions.
func (p *WebSocketPool) Broadcast(message []byte) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, sendChan := range p.sessions {
		select {
		case sendChan <- message:
		default:
			// Channel full, drop message to prevent blocking
		}
	}
}

// BroadcastToRoom sends a message to sessions in a specific room.
func (p *WebSocketPool) BroadcastToRoom(roomVNum int, message []byte, excludeSession string, getRoomFunc func(string) int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for sessionID, sendChan := range p.sessions {
		if sessionID == excludeSession {
			continue
		}
		if getRoomFunc(sessionID) == roomVNum {
			select {
			case sendChan <- message:
			default:
				// Channel full, drop message
			}
		}
	}
}

// Errors are defined in errors.go

func init() {
	// Initialize errors package
	_ = ErrPoolClosed
	_ = ErrPoolFull
	_ = ErrPoolExhausted
}
