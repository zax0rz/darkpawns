package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/optimization"
)

// BenchmarkWebSocketConnection tests WebSocket connection performance
func BenchmarkWebSocketConnection(b *testing.B) {
	// Create test server
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
		}
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	url := "ws" + server.URL[4:] + "/ws"
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				b.Fatal(err)
			}
			conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`))
			conn.ReadMessage()
			conn.Close()
		}
	})
}

// BenchmarkWebSocketBroadcast tests broadcast performance
func BenchmarkWebSocketBroadcast(b *testing.B) {
	const numClients = 100
	var clients []*websocket.Conn
	var server *httptest.Server
	
	// Create test server with broadcast capability
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	
	clientsMutex := sync.RWMutex{}
	broadcastChan := make(chan []byte, 100)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		
		clientsMutex.Lock()
		clients = append(clients, conn)
		clientsMutex.Unlock()
		
		go func() {
			defer func() {
				clientsMutex.Lock()
				for i, c := range clients {
					if c == conn {
						clients = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				clientsMutex.Unlock()
				conn.Close()
			}()
			
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()
	})
	
	server = httptest.NewServer(handler)
	defer server.Close()
	
	// Connect clients
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[4:]+"/ws", nil)
		if err != nil {
			b.Fatal(err)
		}
		clients = append(clients, conn)
	}
	
	// Start broadcast goroutine
	go func() {
		for msg := range broadcastChan {
			clientsMutex.RLock()
			for _, conn := range clients {
				conn.WriteMessage(websocket.TextMessage, msg)
			}
			clientsMutex.RUnlock()
		}
	}()
	
	message := []byte(`{"type":"broadcast","data":"test message"}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastChan <- message
	}
	
	close(broadcastChan)
	
	// Cleanup
	for _, conn := range clients {
		conn.Close()
	}
}

// BenchmarkWorkerPool tests worker pool performance
func BenchmarkWorkerPool(b *testing.B) {
	pool := optimization.NewWorkerPool(10)
	defer pool.Close()
	
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(func() {
				atomic.AddInt64(&counter, 1)
				time.Sleep(time.Microsecond) // Simulate work
			})
		}
	})
	
	b.StopTimer()
	log.Printf("Worker pool processed %d tasks", atomic.LoadInt64(&counter))
}

// BenchmarkConnectionPool tests connection pool performance
func BenchmarkConnectionPool(b *testing.B) {
	createFunc := func() (interface{}, error) {
		// Simulate connection creation
		time.Sleep(5 * time.Millisecond)
		return "connection", nil
	}
	
	closeFunc := func(conn interface{}) error {
		return nil
	}
	
	pool := optimization.NewConnectionPool(10, 30*time.Second, createFunc, closeFunc)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.Get()
			if err != nil {
				b.Fatal(err)
			}
			time.Sleep(100 * time.Microsecond) // Simulate work
			pool.Put(conn)
		}
	})
}

// BenchmarkAIBatchProcessor tests AI batch processing performance
func BenchmarkAIBatchProcessor(b *testing.B) {
	processFunc := func(batch []optimization.AIBatchItem) error {
		// Simulate batch processing
		time.Sleep(10 * time.Millisecond)
		
		for _, item := range batch {
			resp := optimization.AIResponse{
				ID:      item.Request.ID,
				Text:    "response",
				Tokens:  50,
				Latency: 10 * time.Millisecond,
			}
			item.Response <- resp
		}
		
		return nil
	}
	
	processor := optimization.NewAIBatchProcessor(10, 50*time.Millisecond, processFunc)
	defer processor.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		reqID := 0
		for pb.Next() {
			reqID++
			req := optimization.AIRequest{
				ID:        fmt.Sprintf("req-%d", reqID),
				Prompt:    "Test prompt",
				Model:     "test-model",
				MaxTokens: 100,
			}
			
			_, err := processor.Submit(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkAICache tests AI cache performance
func BenchmarkAICache(b *testing.B) {
	cache := optimization.NewAICache(1000, time.Minute)
	
	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		resp := optimization.AIResponse{
			ID:   key,
			Text: fmt.Sprintf("response-%d", i),
		}
		cache.Set(key, resp)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			cache.Get(key)
			i++
		}
	})
}

// BenchmarkJSONMarshal tests JSON marshaling performance
func BenchmarkJSONMarshal(b *testing.B) {
	type TestMessage struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
		Timestamp time.Time         `json:"timestamp"`
	}
	
	message := TestMessage{
		Type: "state",
		Data: map[string]interface{}{
			"room": map[string]interface{}{
				"name": "Test Room",
				"exits": []string{"north", "south", "east", "west"},
				"mobs": []map[string]interface{}{
					{"name": "goblin", "health": 10},
					{"name": "orc", "health": 20},
				},
			},
			"player": map[string]interface{}{
				"name":   "test",
				"health": 100,
				"level":  5,
			},
		},
		Timestamp: time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(message)
	}
}

// BenchmarkSessionManager tests session manager performance
func BenchmarkSessionManager(b *testing.B) {
	// This would require mocking the game world and database
	// For now, just test the data structures
	sessions := make(map[string]chan []byte)
	mutex := sync.RWMutex{}
	
	// Create test sessions
	for i := 0; i < 1000; i++ {
		sessions[fmt.Sprintf("player-%d", i)] = make(chan []byte, 256)
	}
	
	message := []byte(`{"type":"broadcast","data":"test"}`)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mutex.RLock()
			for _, ch := range sessions {
				select {
				case ch <- message:
				default:
					// Drop if full
				}
			}
			mutex.RUnlock()
		}
	})
	
	// Cleanup
	for _, ch := range sessions {
		close(ch)
	}
}

func main() {
	// Run benchmarks
	fmt.Println("Running WebSocket benchmarks...")
	
	// Note: In real usage, run with: go test -bench=. -benchtime=10s ./benchmarks
	fmt.Println("Benchmarks defined. Run with: go test -bench=. ./benchmarks")
}