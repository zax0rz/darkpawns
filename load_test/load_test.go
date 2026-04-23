package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// LoadTestConfig holds load test configuration
type LoadTestConfig struct {
	ServerURL      string
	NumClients     int
	Duration       time.Duration
	MessagesPerSec int
	MessageSize    int
	EnableMetrics  bool
}

// LoadTestResult holds load test results
type LoadTestResult struct {
	TotalClients      int
	TotalMessagesSent int64
	TotalMessagesRecv int64
	TotalErrors       int64
	TotalLatency      time.Duration
	StartTime         time.Time
	EndTime           time.Time
	Throughput        float64 // messages per second
	AvgLatency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
}

// LoadTestClient represents a single WebSocket client
type LoadTestClient struct {
	ID           int
	Conn         *websocket.Conn
	Config       *LoadTestConfig
	MessagesSent int64
	MessagesRecv int64
	Errors       int64
	TotalLatency time.Duration
	Latencies    []time.Duration
	StopChan     chan struct{}
	WG           *sync.WaitGroup
}

// NewLoadTestClient creates a new load test client
func NewLoadTestClient(id int, config *LoadTestConfig, wg *sync.WaitGroup) *LoadTestClient {
	return &LoadTestClient{
		ID:        id,
		Config:    config,
		StopChan:  make(chan struct{}),
		WG:        wg,
		Latencies: make([]time.Duration, 0, 1000),
	}
}

// Connect connects the client to the WebSocket server
func (c *LoadTestClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.Config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("client %d: %w", c.ID, err)
	}
	c.Conn = conn

	// Send login message
	loginMsg := map[string]interface{}{
		"type": "login",
		"data": map[string]interface{}{
			"player_name": fmt.Sprintf("loadtest-%d", c.ID),
			"mode":        "agent",
		},
	}

	if err := conn.WriteJSON(loginMsg); err != nil {
		conn.Close()
		return fmt.Errorf("client %d login: %w", c.ID, err)
	}

	// Wait for login response
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("client %d login response: %w", c.ID, err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		conn.Close()
		return fmt.Errorf("client %d parse response: %w", c.ID, err)
	}

	if response["type"] != "state" {
		conn.Close()
		return fmt.Errorf("client %d unexpected response: %v", c.ID, response)
	}

	return nil
}

// Start starts the client's message loop
func (c *LoadTestClient) Start() {
	defer c.WG.Done()

	// Calculate interval between messages
	interval := time.Second / time.Duration(c.Config.MessagesPerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Start reader goroutine
	readDone := make(chan struct{})
	go c.readLoop(readDone)

	// Send messages
	for {
		select {
		case <-ticker.C:
			c.sendMessage()
		case <-c.StopChan:
			close(readDone)
			c.Conn.Close()
			return
		}
	}
}

// sendMessage sends a test message
func (c *LoadTestClient) sendMessage() {
	start := time.Now()

	// Generate random command
	commands := []string{"look", "north", "south", "east", "west", "say hello", "stats"}
	command := commands[rand.Intn(len(commands))]

	msg := map[string]interface{}{
		"type": "command",
		"data": map[string]interface{}{
			"command": command,
		},
	}

	if err := c.Conn.WriteJSON(msg); err != nil {
		atomic.AddInt64(&c.Errors, 1)
		return
	}

	atomic.AddInt64(&c.MessagesSent, 1)

	// Store latency (will be updated when response is received)
	c.Latencies = append(c.Latencies, time.Since(start))
}

// readLoop reads messages from the WebSocket
func (c *LoadTestClient) readLoop(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			c.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, _, err := c.Conn.ReadMessage()
			if err != nil {
				atomic.AddInt64(&c.Errors, 1)
				continue
			}
			atomic.AddInt64(&c.MessagesRecv, 1)
		}
	}
}

// Stop stops the client
func (c *LoadTestClient) Stop() {
	close(c.StopChan)
}

// LoadTestRunner runs the load test
type LoadTestRunner struct {
	Config    *LoadTestConfig
	Clients   []*LoadTestClient
	Results   *LoadTestResult
	StartTime time.Time
}

// NewLoadTestRunner creates a new load test runner
func NewLoadTestRunner(config *LoadTestConfig) *LoadTestRunner {
	return &LoadTestRunner{
		Config:  config,
		Clients: make([]*LoadTestClient, 0, config.NumClients),
		Results: &LoadTestResult{},
	}
}

// Run runs the load test
func (r *LoadTestRunner) Run() (*LoadTestResult, error) {
	log.Printf("Starting load test with %d clients for %v", r.Config.NumClients, r.Config.Duration)

	r.StartTime = time.Now()

	// Connect all clients
	var wg sync.WaitGroup
	for i := 0; i < r.Config.NumClients; i++ {
		client := NewLoadTestClient(i, r.Config, &wg)
		if err := client.Connect(); err != nil {
			log.Printf("Warning: Client %d failed to connect: %v", i, err)
			continue
		}
		r.Clients = append(r.Clients, client)
		wg.Add(1)
		go client.Start()
	}

	log.Printf("Connected %d clients, running test for %v", len(r.Clients), r.Config.Duration)

	// Run for specified duration
	time.Sleep(r.Config.Duration)

	// Stop all clients
	for _, client := range r.Clients {
		client.Stop()
	}

	// Wait for clients to stop
	wg.Wait()

	// Collect results
	r.collectResults()

	return r.Results, nil
}

// collectResults collects results from all clients
func (r *LoadTestRunner) collectResults() {
	r.Results.StartTime = r.StartTime
	r.Results.EndTime = time.Now()
	r.Results.TotalClients = len(r.Clients)

	// Aggregate statistics
	var allLatencies []time.Duration
	for _, client := range r.Clients {
		r.Results.TotalMessagesSent += client.MessagesSent
		r.Results.TotalMessagesRecv += client.MessagesRecv
		r.Results.TotalErrors += client.Errors
		r.Results.TotalLatency += client.TotalLatency
		allLatencies = append(allLatencies, client.Latencies...)
	}

	// Calculate throughput
	duration := r.Results.EndTime.Sub(r.Results.StartTime)
	r.Results.Throughput = float64(r.Results.TotalMessagesSent) / duration.Seconds()

	// Calculate latency percentiles
	if len(allLatencies) > 0 {
		// Sort latencies (simplified - in production use proper sorting)
		total := time.Duration(0)
		for _, lat := range allLatencies {
			total += lat
		}
		r.Results.AvgLatency = total / time.Duration(len(allLatencies))

		// Simple percentile calculation (for demo)
		if len(allLatencies) >= 100 {
			// Get 95th and 99th percentiles (approximate)
			idx95 := len(allLatencies) * 95 / 100
			idx99 := len(allLatencies) * 99 / 100
			r.Results.P95Latency = allLatencies[idx95]
			r.Results.P99Latency = allLatencies[idx99]
		}
	}
}

// PrintResults prints the load test results
func (r *LoadTestRunner) PrintResults() {
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration:           %v\n", r.Results.EndTime.Sub(r.Results.StartTime))
	fmt.Printf("Total Clients:      %d\n", r.Results.TotalClients)
	fmt.Printf("Messages Sent:      %d\n", r.Results.TotalMessagesSent)
	fmt.Printf("Messages Received:  %d\n", r.Results.TotalMessagesRecv)
	fmt.Printf("Errors:             %d\n", r.Results.TotalErrors)
	fmt.Printf("Throughput:         %.2f msg/sec\n", r.Results.Throughput)
	fmt.Printf("Avg Latency:        %v\n", r.Results.AvgLatency)
	if r.Results.P95Latency > 0 {
		fmt.Printf("P95 Latency:        %v\n", r.Results.P95Latency)
	}
	if r.Results.P99Latency > 0 {
		fmt.Printf("P99 Latency:        %v\n", r.Results.P99Latency)
	}
	fmt.Printf("Success Rate:       %.2f%%\n",
		float64(r.Results.TotalMessagesRecv)/float64(r.Results.TotalMessagesSent)*100)
}

// RunConcurrentLoadTest runs multiple concurrent load tests
func RunConcurrentLoadTest() {
	configs := []LoadTestConfig{
		{
			ServerURL:      "ws://localhost:8080/ws",
			NumClients:     100,
			Duration:       30 * time.Second,
			MessagesPerSec: 2,
			EnableMetrics:  true,
		},
		{
			ServerURL:      "ws://localhost:8080/ws",
			NumClients:     500,
			Duration:       60 * time.Second,
			MessagesPerSec: 1,
			EnableMetrics:  true,
		},
		{
			ServerURL:      "ws://localhost:8080/ws",
			NumClients:     1000,
			Duration:       120 * time.Second,
			MessagesPerSec: 1, // 1 message per second
			EnableMetrics:  true,
		},
	}

	for _, config := range configs {
		runner := NewLoadTestRunner(&config)
		result, err := runner.Run()
		if err != nil {
			log.Printf("Load test failed: %v", err)
			continue
		}

		_ = result // Use result if needed
		runner.PrintResults()

		// Wait between tests
		time.Sleep(10 * time.Second)
	}
}

// SimpleLoadTest runs a simple load test for quick verification
func SimpleLoadTest(serverURL string, numClients int, duration time.Duration) {
	config := LoadTestConfig{
		ServerURL:      serverURL,
		NumClients:     numClients,
		Duration:       duration,
		MessagesPerSec: 1,
		EnableMetrics:  true,
	}

	runner := NewLoadTestRunner(&config)
	result, err := runner.Run()
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	_ = result // Use result if needed
	runner.PrintResults()
}

func main() {
	// Parse command line arguments
	// For now, run a simple test
	fmt.Println("Dark Pawns Load Test")
	fmt.Println("====================")

	// Check if server is running
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		log.Fatalf("Server not running: %v", err)
	}
	resp.Body.Close()

	fmt.Println("Server is running. Starting load test...")

	// Run simple load test
	SimpleLoadTest("ws://localhost:8080/ws", 50, 10*time.Second)

	fmt.Println("\nLoad test completed!")
}
