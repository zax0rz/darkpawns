package examples

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/optimization"
)

func OptimizationIntegration() {
	fmt.Println("Dark Pawns Optimization Integration Example")
	fmt.Println("===========================================")

	// Example 1: Worker Pool
	fmt.Println("\n1. Worker Pool Example:")
	WorkerPoolExample()

	// Example 2: Connection Pool
	fmt.Println("\n2. Connection Pool Example:")
	ConnectionPoolExample()

	// Example 3: AI Batch Processing
	fmt.Println("\n3. AI Batch Processing Example:")
	AiBatchProcessingExample()

	// Example 4: WebSocket Optimization
	fmt.Println("\n4. WebSocket Optimization Example:")
	WebsocketOptimizationExample()

	// Example 5: Query Optimization
	fmt.Println("\n5. Query Optimization Example:")
	QueryOptimizationExample()

	fmt.Println("\nAll examples completed!")
}

func WorkerPoolExample() {
	// Create worker pool with 5 workers
	pool := optimization.NewWorkerPool(5)
	defer pool.Close()

	// Submit tasks
	for i := 0; i < 20; i++ {
		taskID := i
		if err := pool.Submit(func() {
			time.Sleep(10 * time.Millisecond)
			fmt.Printf("  Task %d completed\n", taskID)
		}); err != nil {
			log.Printf("  Error submitting task %d: %v", taskID, err)
		}
	}

	// Wait for all submitted tasks to finish before Close (deferred above)
	// tears the pool down.
	pool.Wait()
}

func ConnectionPoolExample() {
	// Simulate connection creation
	createFunc := func() (interface{}, error) {
		fmt.Println("  Creating new connection...")
		time.Sleep(5 * time.Millisecond)
		return fmt.Sprintf("connection-%d", time.Now().UnixNano()), nil
	}

	closeFunc := func(conn interface{}) error {
		fmt.Printf("  Closing connection: %v\n", conn)
		return nil
	}

	// Create connection pool
	pool := optimization.NewConnectionPool(3, 30*time.Second, createFunc, closeFunc)
	defer func() { _ = pool.Close() }() //nolint:errcheck

	// Get and use connections
	for i := 0; i < 5; i++ {
		conn, err := pool.Get()
		if err != nil {
			log.Printf("  Error getting connection: %v", err)
			continue
		}

		fmt.Printf("  Using connection: %v\n", conn)
		time.Sleep(2 * time.Millisecond)

		if err := pool.Put(conn); err != nil {
			log.Printf("  Error returning connection to pool: %v", err)
		}
	}

	// Print pool stats
	stats := pool.Stats()
	fmt.Printf("  Pool stats: Total=%d, Active=%d, Idle=%d\n",
		stats.TotalConnections, stats.ActiveConnections, stats.IdleConnections)
}

func AiBatchProcessingExample() {
	// Create batch processor
	processor := optimization.NewAIBatchProcessor(5, 100*time.Millisecond, func(batch []optimization.AIBatchItem) error {
		fmt.Printf("  Processing batch of %d requests\n", len(batch))

		// Simulate AI processing
		time.Sleep(20 * time.Millisecond)

		// Send responses
		for _, item := range batch {
			resp := optimization.AIResponse{
				ID:      item.Request.ID,
				Text:    "Processed: " + item.Request.Prompt,
				Tokens:  50,
				Latency: 20 * time.Millisecond,
				Model:   item.Request.Model,
			}
			item.Response <- resp
		}

		return nil
	})
	defer func() { _ = processor.Close() }()

	// Submit requests. reqWg tracks the request goroutines themselves (Add
	// happens before each goroutine starts) so we can deterministically
	// wait for every request to be submitted and answered before Close
	// runs, instead of guessing at a fixed sleep duration.
	var reqWg sync.WaitGroup
	for i := 0; i < 15; i++ {
		reqWg.Add(1)
		go func(id int) {
			defer reqWg.Done()

			req := optimization.AIRequest{
				ID:     fmt.Sprintf("req-%d", id),
				Prompt: fmt.Sprintf("Question %d", id),
				Model:  "test-model",
			}

			resp, err := processor.Submit(context.Background(), req)
			if err != nil {
				log.Printf("  Error processing request %d: %v", id, err)
				return
			}

			fmt.Printf("  Response for request %d: %s\n", id, resp.Text[:20])
		}(i)
	}

	// Wait for every submitted request to receive its response (or error),
	// then wait on the processor itself before the deferred Close above
	// drains and tears it down.
	reqWg.Wait()
	processor.Wait()
}

func WebsocketOptimizationExample() {
	// Create WebSocket pool
	pool := optimization.NewWebSocketPool(256)

	// Simulate sessions
	session1 := make(chan []byte, 256)
	session2 := make(chan []byte, 256)

	pool.Register("session-1", session1)
	pool.Register("session-2", session2)

	// Start message consumers
	go func() {
		for msg := range session1 {
			fmt.Printf("  Session 1 received: %s\n", string(msg[:min(20, len(msg))]))
		}
	}()

	go func() {
		for msg := range session2 {
			fmt.Printf("  Session 2 received: %s\n", string(msg[:min(20, len(msg))]))
		}
	}()

	// Broadcast message
	message := []byte(`{"type":"broadcast","data":"Hello from server"}`)
	pool.Broadcast(message)

	// Wait for messages to be processed
	time.Sleep(50 * time.Millisecond)

	// Cleanup: unregister from the pool before closing the channels so the
	// pool cannot send on a channel that has just been closed.
	pool.Unregister("session-1")
	pool.Unregister("session-2")
	close(session1)
	close(session2)
}

func QueryOptimizationExample() {
	// Create query optimizer
	optimizer := optimization.NewQueryOptimizer(100, 50*time.Millisecond)

	// Simulate queries
	queries := []string{
		"SELECT * FROM players WHERE name = $1",
		"UPDATE players SET health = $1 WHERE id = $2",
		"INSERT INTO narrative_memory (player_id, event) VALUES ($1, $2)",
	}

	for i := 0; i < 50; i++ {
		query := queries[i%len(queries)]
		duration := time.Duration(10+(i%40)) * time.Millisecond
		indexUsed := i%2 == 0

		optimizer.RecordQuery(query, duration, indexUsed)
	}

	// Get slow queries
	slowQueries := optimizer.GetSlowQueries()
	fmt.Printf("  Found %d slow queries:\n", len(slowQueries))

	for _, stat := range slowQueries {
		fmt.Printf("  - %s: avg=%v, count=%d\n",
			stat.Query[:30], stat.AvgDuration, stat.Count)
	}

	// Get all stats
	allStats := optimizer.GetStats()
	fmt.Printf("  Total queries tracked: %d\n", len(allStats))
}
