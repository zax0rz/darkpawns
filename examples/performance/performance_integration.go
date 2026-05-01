package main

import (
	"fmt"
	"log"
	"time"

	"github.com/zax0rz/darkpawns/pkg/optimization"
)

func main() {
	fmt.Println("=== Dark Pawns Performance Optimization Demo ===")
	
	// 1. Demonstrate Cache
	fmt.Println("\n1. Cache Demo")
	cache := optimization.NewCache(1 * time.Minute)
	defer cache.Close()
	
	cache.Set("player:alice", map[string]interface{}{"name": "Alice", "level": 10})
	cache.Set("player:bob", map[string]interface{}{"name": "Bob", "level": 5})
	
	if val, hit := cache.Get("player:alice"); hit {
		fmt.Printf("Cache hit: %v\n", val)
	}
	
	stats := cache.Stats()
	fmt.Printf("Cache stats: %v\n", stats)
	
	// 2. Demonstrate Room Cache
	fmt.Println("\n2. Room Cache Demo")
	roomCache := optimization.NewRoomCache(2 * time.Minute)
	defer roomCache.Close()
	
	// Simulate room fetch function
	fetchRoom := func(vnum int) (*optimization.CachedRoom, error) {
		fmt.Printf("Fetching room %d from database...\n", vnum)
		time.Sleep(50 * time.Millisecond) // Simulate DB latency
		
		return &optimization.CachedRoom{
			VNum:        vnum,
			Name:        fmt.Sprintf("Room %d", vnum),
			Description: "A dark room",
			Players:     []string{"Alice", "Bob"},
			CachedAt:    time.Now(),
		}, nil
	}
	
	// First access - cache miss
	room1, err := roomCache.GetRoom(100, fetchRoom)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Room 100: %s\n", room1.Name)
	
	// Second access - cache hit
	room2, err := roomCache.GetRoom(100, fetchRoom)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Room 100 (cached): %s (access count: %d)\n", room2.Name, room2.AccessCount)
	
	// 3. Demonstrate Advanced Worker Pool
	fmt.Println("\n3. Advanced Worker Pool Demo")
	pool := optimization.NewAdvancedWorkerPool(4, 100)
	defer pool.Close()
	
	// Submit tasks
	for i := 0; i < 10; i++ {
		taskID := i
// #nosec G104
		_ = pool.Submit(func() {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Task %d completed\n", taskID)
		})
	}
	
	// Submit high priority task
// #nosec G104
	_ = pool.SubmitWithPriority(func() {
		fmt.Println("High priority task completed immediately")
	}, 2)
	
	time.Sleep(500 * time.Millisecond) // Wait for tasks to complete
	
	poolStats := pool.GetQueueStats()
	fmt.Printf("Pool stats: %v\n", poolStats)
	
	// 4. Demonstrate Object Pool
	fmt.Println("\n4. Object Pool Demo")
	
	// Create object pool for player objects
	playerPool := optimization.NewObjectPool(
		func() interface{} {
			return &Player{ID: 0, Name: "", Level: 0}
		},
		func(obj interface{}) {
			p := obj.(*Player)
			p.ID = 0
			p.Name = ""
			p.Level = 0
			p.Inventory = nil
		},
		func(obj interface{}) bool {
			// Always valid for demo
			return true
		},
		100,
	)
	
	// Prefill pool
	playerPool.Prefill(10)
	
	// Borrow and return objects
	players := make([]interface{}, 5)
	for i := 0; i < 5; i++ {
		player := playerPool.Get().(*Player)
		player.ID = i + 1
		player.Name = fmt.Sprintf("Player%d", i+1)
		player.Level = i * 5
		players[i] = player
		fmt.Printf("Borrowed player: %s (Level %d)\n", player.Name, player.Level)
	}
	
	// Return objects
	for _, player := range players {
		playerPool.Put(player)
	}
	
	poolStats2 := playerPool.Stats()
	fmt.Printf("Object pool stats: %v\n", poolStats2)
	
	// 5. Demonstrate String Pool
	fmt.Println("\n5. String Pool Demo")
	stringPool := optimization.NewStringPool()
	
	strings := []string{"hello", "world", "hello", "test", "world", "hello"}
	
	for _, s := range strings {
		pooled := stringPool.Get(s)
		fmt.Printf("Original: %p -> Pooled: %p (%s)\n", &s, &pooled, pooled)
	}
	
	fmt.Printf("String pool size: %d\n", stringPool.Size())
	
	// 6. Performance comparison
	fmt.Println("\n6. Performance Comparison")
	
	// Without pooling
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = &Player{ID: i, Name: fmt.Sprintf("Player%d", i), Level: i % 100}
	}
	withoutPool := time.Since(start)
	fmt.Printf("Without pooling: %v\n", withoutPool)
	
	// With pooling
	start = time.Now()
	for i := 0; i < 10000; i++ {
		player := playerPool.Get().(*Player)
		player.ID = i
		player.Name = fmt.Sprintf("Player%d", i)
		player.Level = i % 100
		playerPool.Put(player)
	}
	withPool := time.Since(start)
	fmt.Printf("With pooling: %v\n", withPool)
	fmt.Printf("Improvement: %.2fx faster\n", float64(withoutPool)/float64(withPool))
	
	fmt.Println("\n=== Demo Complete ===")
}

// Player represents a game player
type Player struct {
	ID       int
	Name     string
	Level    int
	Inventory []string
}