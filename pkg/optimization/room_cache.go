package optimization

import (
	"sync"
	"time"
)

// RoomCache caches room data for frequent access
type RoomCache struct {
	mu    sync.RWMutex
	rooms map[int]*CachedRoom
	ttl   time.Duration
	stop  chan struct{}
}

// CachedRoom represents cached room data
type CachedRoom struct {
	VNum        int
	Name        string
	Description string
	Exits       []ExitData
	Players     []string
	Mobs        []MobData
	Items       []ItemData
	CachedAt    time.Time
	AccessCount int
	LastUpdated time.Time
}

// ExitData represents room exit information
type ExitData struct {
	Direction string
	ToRoom    int
	Flags     []string
}

// MobData represents mob information in room
type MobData struct {
	ID    int
	Name  string
	Level int
}

// ItemData represents item information in room
type ItemData struct {
	ID   int
	Name string
	Type string
}

// NewRoomCache creates a new room cache
func NewRoomCache(ttl time.Duration) *RoomCache {
	rc := &RoomCache{
		rooms: make(map[int]*CachedRoom),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rc.cleanup()

	return rc
}

// GetRoom retrieves room from cache or fetches if not present
func (rc *RoomCache) GetRoom(vnum int, fetchFunc func(int) (*CachedRoom, error)) (*CachedRoom, error) {
	// Try cache first
	rc.mu.RLock()
	cached, exists := rc.rooms[vnum]
	rc.mu.RUnlock()

	if exists && time.Since(cached.CachedAt) < rc.ttl {
		rc.mu.Lock()
		cached.AccessCount++
		rc.mu.Unlock()
		return cached, nil
	}

	// Fetch from source
	room, err := fetchFunc(vnum)
	if err != nil {
		return nil, err
	}

	// Update cache
	rc.mu.Lock()
	room.CachedAt = time.Now()
	room.AccessCount = 1
	room.LastUpdated = time.Now()
	rc.rooms[vnum] = room
	rc.mu.Unlock()

	return room, nil
}

// UpdateRoom updates room in cache
func (rc *RoomCache) UpdateRoom(room *CachedRoom) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	room.LastUpdated = time.Now()
	room.CachedAt = time.Now()
	rc.rooms[room.VNum] = room
}

// UpdateRoomPartial updates specific fields of a room
func (rc *RoomCache) UpdateRoomPartial(vnum int, updates map[string]interface{}) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	room, exists := rc.rooms[vnum]
	if !exists {
		return false
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "players":
			if players, ok := value.([]string); ok {
				room.Players = players
			}
		case "mobs":
			if mobs, ok := value.([]MobData); ok {
				room.Mobs = mobs
			}
		case "items":
			if items, ok := value.([]ItemData); ok {
				room.Items = items
			}
		case "name":
			if name, ok := value.(string); ok {
				room.Name = name
			}
		case "description":
			if desc, ok := value.(string); ok {
				room.Description = desc
			}
		}
	}

	room.LastUpdated = time.Now()
	return true
}

// Invalidate removes a room from cache
func (rc *RoomCache) Invalidate(vnum int) {
	rc.mu.Lock()
	delete(rc.rooms, vnum)
	rc.mu.Unlock()
}

// InvalidateAll removes all rooms from cache
func (rc *RoomCache) InvalidateAll() {
	rc.mu.Lock()
	rc.rooms = make(map[int]*CachedRoom)
	rc.mu.Unlock()
}

// GetStats returns cache statistics
func (rc *RoomCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_rooms"] = len(rc.rooms)

	var totalAccess int
	now := time.Now()
	expiredCount := 0
	staleCount := 0

	for _, room := range rc.rooms {
		totalAccess += room.AccessCount

		if now.Sub(room.CachedAt) > rc.ttl {
			expiredCount++
		}

		// Consider room stale if not updated in 2x TTL
		if now.Sub(room.LastUpdated) > rc.ttl*2 {
			staleCount++
		}
	}

	if len(rc.rooms) > 0 {
		stats["avg_access_per_room"] = totalAccess / len(rc.rooms)
		stats["expired_count"] = expiredCount
		stats["stale_count"] = staleCount
		stats["hit_ratio"] = float64(totalAccess) / float64(len(rc.rooms)+totalAccess)

		// Find most accessed room
		var maxAccess int
		var maxAccessVNum int
		for vnum, room := range rc.rooms {
			if room.AccessCount > maxAccess {
				maxAccess = room.AccessCount
				maxAccessVNum = vnum
			}
		}
		stats["most_accessed_room"] = maxAccessVNum
		stats["most_accessed_count"] = maxAccess
	}

	return stats
}

// GetHotRooms returns rooms accessed above threshold
func (rc *RoomCache) GetHotRooms(threshold int) []int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var hotRooms []int
	for vnum, room := range rc.rooms {
		if room.AccessCount >= threshold {
			hotRooms = append(hotRooms, vnum)
		}
	}

	return hotRooms
}

// cleanup periodically removes expired rooms
func (rc *RoomCache) cleanup() {
	ticker := time.NewTicker(rc.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.mu.Lock()
			now := time.Now()
			for vnum, room := range rc.rooms {
				if now.Sub(room.CachedAt) > rc.ttl {
					delete(rc.rooms, vnum)
				}
			}
			rc.mu.Unlock()
		case <-rc.stop:
			return
		}
	}
}

// Close stops the cleanup goroutine
func (rc *RoomCache) Close() {
	close(rc.stop)
}
