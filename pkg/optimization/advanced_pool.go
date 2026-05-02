package optimization

import (
	"sync"
	"sync/atomic"
	"time"
)

// AdvancedWorkerPool provides enhanced worker pool with metrics
type AdvancedWorkerPool struct {
	workers       int
	taskQueue     chan func()
	priorityQueue chan priorityTask
	wg            sync.WaitGroup
	mu            sync.RWMutex
	closed        bool
	metrics       PoolMetrics
	stop          chan struct{}
}

// PoolMetrics tracks pool performance
type PoolMetrics struct {
	TasksSubmitted    int64
	TasksCompleted    int64
	TasksFailed       int64
	QueueLength       int64
	PriorityLength    int64
	AvgWaitTime       time.Duration
	MaxWaitTime       time.Duration
	WorkerUtilization float64
	TotalWaitTime     time.Duration
}

type priorityTask struct {
	task      func()
	priority  int // 0=low, 1=normal, 2=high
	submitted time.Time
}

// NewAdvancedWorkerPool creates an enhanced worker pool
func NewAdvancedWorkerPool(workers, queueSize int) *AdvancedWorkerPool {
	pool := &AdvancedWorkerPool{
		workers:       workers,
		taskQueue:     make(chan func(), queueSize),
		priorityQueue: make(chan priorityTask, queueSize/2),
		stop:          make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.advancedWorker(i)
	}

	// Start priority dispatcher
	go pool.priorityDispatcher()

	return pool
}

// priorityDispatcher handles priority task scheduling
func (p *AdvancedWorkerPool) priorityDispatcher() {
	for {
		select {
		case pt := <-p.priorityQueue:
			atomic.AddInt64(&p.metrics.PriorityLength, -1)

			// Try to send immediately
			select {
			case p.taskQueue <- pt.task:
				waitTime := time.Since(pt.submitted)
				p.updateWaitTime(waitTime)
			default:
				// Queue full, try again with backoff
				go func(pt priorityTask) {
					time.Sleep(time.Millisecond * time.Duration(pt.priority*10))
					select {
					case p.taskQueue <- pt.task:
						waitTime := time.Since(pt.submitted)
						p.updateWaitTime(waitTime)
					case <-p.stop:
						// Pool closed, drop task
						atomic.AddInt64(&p.metrics.TasksFailed, 1)
					}
				}(pt)
			}
		case <-p.stop:
			return
		}
	}
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

		// Update metrics
		processTime := time.Since(start)
		p.mu.Lock()
		p.metrics.TotalWaitTime += processTime
		if processTime > p.metrics.MaxWaitTime {
			p.metrics.MaxWaitTime = processTime
		}
		p.mu.Unlock()
	}
}

// updateWaitTime updates wait time metrics
func (p *AdvancedWorkerPool) updateWaitTime(waitTime time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalWaitTime += waitTime
	if p.metrics.TasksCompleted > 0 {
		p.metrics.AvgWaitTime = p.metrics.TotalWaitTime / time.Duration(p.metrics.TasksCompleted)
	}
	if waitTime > p.metrics.MaxWaitTime {
		p.metrics.MaxWaitTime = waitTime
	}
}

// Submit submits a task with normal priority
func (p *AdvancedWorkerPool) Submit(task func()) error {
	return p.SubmitWithPriority(task, 1)
}

// SubmitWithPriority submits task with priority handling
func (p *AdvancedWorkerPool) SubmitWithPriority(task func(), priority int) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPoolClosed
	}

	atomic.AddInt64(&p.metrics.TasksSubmitted, 1)

	pt := priorityTask{
		task:      task,
		priority:  priority,
		submitted: time.Now(),
	}

	// High priority tasks go to priority queue
	if priority >= 2 {
		select {
		case p.priorityQueue <- pt:
			atomic.AddInt64(&p.metrics.PriorityLength, 1)
			return nil
		default:
			atomic.AddInt64(&p.metrics.TasksFailed, 1)
			return ErrPoolFull
		}
	}

	// Normal and low priority tasks
	select {
	case p.taskQueue <- task:
		atomic.AddInt64(&p.metrics.QueueLength, 1)
		return nil
	default:
		// Queue full, try priority queue as fallback
		select {
		case p.priorityQueue <- pt:
			atomic.AddInt64(&p.metrics.PriorityLength, 1)
			return nil
		default:
			atomic.AddInt64(&p.metrics.TasksFailed, 1)
			return ErrPoolFull
		}
	}
}

// SubmitBatch submits multiple tasks at once
func (p *AdvancedWorkerPool) SubmitBatch(tasks []func()) error {
	for _, task := range tasks {
		if err := p.Submit(task); err != nil {
			return err
		}
	}
	return nil
}

// GetMetrics returns current pool metrics
func (p *AdvancedWorkerPool) GetMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics := p.metrics
	metrics.QueueLength = atomic.LoadInt64(&p.metrics.QueueLength)
	metrics.PriorityLength = atomic.LoadInt64(&p.metrics.PriorityLength)

	if metrics.TasksSubmitted > 0 {
		metrics.WorkerUtilization = float64(metrics.TasksCompleted) / float64(metrics.TasksSubmitted)
	}

	return metrics
}

// GetQueueStats returns detailed queue statistics
func (p *AdvancedWorkerPool) GetQueueStats() map[string]interface{} {
	metrics := p.GetMetrics()

	stats := make(map[string]interface{})
	stats["total_tasks"] = metrics.TasksSubmitted
	stats["completed_tasks"] = metrics.TasksCompleted
	stats["failed_tasks"] = metrics.TasksFailed
	stats["queue_length"] = metrics.QueueLength
	stats["priority_queue_length"] = metrics.PriorityLength
	stats["avg_wait_time_ms"] = metrics.AvgWaitTime.Milliseconds()
	stats["max_wait_time_ms"] = metrics.MaxWaitTime.Milliseconds()
	stats["worker_utilization"] = metrics.WorkerUtilization
	stats["success_rate"] = float64(metrics.TasksCompleted) / float64(metrics.TasksSubmitted)

	return stats
}

// Resize changes the number of workers
func (p *AdvancedWorkerPool) Resize(newWorkers int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	if newWorkers < 1 {
		return ErrInvalidPoolSize
	}

	// Add more workers
	if newWorkers > p.workers {
		for i := p.workers; i < newWorkers; i++ {
			p.wg.Add(1)
			go p.advancedWorker(i)
		}
	} else if newWorkers < p.workers {
		// Cannot easily reduce workers, they'll exit when pool closes
		// For now, just update the count
	}

	p.workers = newWorkers
	return nil
}

// Close gracefully shuts down the pool
func (p *AdvancedWorkerPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.stop)
	close(p.taskQueue)
	close(p.priorityQueue)
	p.wg.Wait()
}
