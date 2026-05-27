package optimization

import (
	"sync"
	"time"
)

// Cache provides thread-safe caching with TTL
type Cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	ttl   time.Duration
	stop  chan struct{}
}

type cacheItem struct {
	value       interface{}
	expiresAt   time.Time
	createdAt   time.Time
	accessCount int
}

// NewCache creates a new cache with TTL
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Set adds an item to cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:       value,
		expiresAt:   time.Now().Add(c.ttl),
		createdAt:   time.Now(),
		accessCount: 0,
	}
}

// Get retrieves an item from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	item.accessCount++
	c.mu.Unlock()

	return item.value, true
}

// Delete removes an item from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Size returns number of items in cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["size"] = len(c.items)

	var totalAccess int
	now := time.Now()
	expiredCount := 0

	for _, item := range c.items {
		totalAccess += item.accessCount
		if now.After(item.expiresAt) {
			expiredCount++
		}
	}

	if len(c.items) > 0 {
		stats["avg_access_per_item"] = totalAccess / len(c.items)
		stats["expired_count"] = expiredCount
		stats["hit_ratio"] = float64(totalAccess) / float64(len(c.items)+totalAccess)
	}

	return stats
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, item := range c.items {
				if now.After(item.expiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}

// Close stops the cleanup goroutine
func (c *Cache) Close() {
	close(c.stop)
}
