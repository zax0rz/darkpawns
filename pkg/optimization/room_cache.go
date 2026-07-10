package optimization

import (
	"sync"
	"time"
)

// RoomCache caches room data for frequent access
type RoomCache struct {
	mu       sync.RWMutex
	rooms    map[int]*CachedRoom
	ttl      time.Duration
	stop     chan struct{}
	stopOnce sync.Once
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

// Clone returns a deep copy of the cached room, including all slice fields.
func (r *CachedRoom) Clone() *CachedRoom {
	if r == nil {
		return nil
	}

	clone := &CachedRoom{
		VNum:        r.VNum,
		Name:        r.Name,
		Description: r.Description,
		CachedAt:    r.CachedAt,
		AccessCount: r.AccessCount,
		LastUpdated: r.LastUpdated,
	}

	if len(r.Exits) > 0 {
		clone.Exits = make([]ExitData, len(r.Exits))
		copy(clone.Exits, r.Exits)
		for i := range clone.Exits {
			if len(r.Exits[i].Flags) > 0 {
				clone.Exits[i].Flags = make([]string, len(r.Exits[i].Flags))
				copy(clone.Exits[i].Flags, r.Exits[i].Flags)
			}
		}
	}

	if len(r.Players) > 0 {
		clone.Players = make([]string, len(r.Players))
		copy(clone.Players, r.Players)
	}

	if len(r.Mobs) > 0 {
		clone.Mobs = make([]MobData, len(r.Mobs))
		copy(clone.Mobs, r.Mobs)
	}

	if len(r.Items) > 0 {
		clone.Items = make([]ItemData, len(r.Items))
		copy(clone.Items, r.Items)
	}

	return clone
}

// NewRoomCache creates a new room cache. A non-positive TTL disables
// background cleanup (rooms expire lazily on access) and avoids a panic from
// an invalid ticker interval.
func NewRoomCache(ttl time.Duration) *RoomCache {
	rc := &RoomCache{
		rooms: make(map[int]*CachedRoom),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}

	if ttl > 0 {
		go rc.cleanup()
	}

	return rc
}

// GetRoom retrieves room from cache or fetches if not present. The returned
// CachedRoom is a deep copy owned by the caller; mutating it will not affect
// the cache's internal state.
func (rc *RoomCache) GetRoom(vnum int, fetchFunc func(int) (*CachedRoom, error)) (*CachedRoom, error) {
	// Try cache first. Hold the write lock across the freshness check, the
	// access-count increment, and the Clone so the entry cannot be replaced
	// or deleted by a concurrent writer between the check and the mutation
	// (a TOCTOU that would increment/return a stale, orphaned entry).
	rc.mu.Lock()
	if cached, exists := rc.rooms[vnum]; exists && time.Since(cached.CachedAt) < rc.ttl {
		cached.AccessCount++
		result := cached.Clone()
		rc.mu.Unlock()
		return result, nil
	}
	rc.mu.Unlock()

	// Fetch from source
	room, err := fetchFunc(vnum)
	if err != nil {
		return nil, err
	}

	// Update cache with a clone so the caller cannot mutate our internal copy.
	// Clone the return value while still holding the lock — once stored is in
	// the map, a concurrent GetRoom hit can increment stored.AccessCount, so
	// reading it after Unlock would race.
	rc.mu.Lock()
	stored := room.Clone()
	stored.CachedAt = time.Now()
	stored.AccessCount = 1
	stored.LastUpdated = time.Now()
	rc.rooms[vnum] = stored
	result := stored.Clone()
	rc.mu.Unlock()

	return result, nil
}

// UpdateRoom updates room in cache. The cache stores a deep copy of the
// supplied room so caller mutations do not affect cached state.
func (rc *RoomCache) UpdateRoom(room *CachedRoom) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	stored := room.Clone()
	stored.LastUpdated = time.Now()
	stored.CachedAt = time.Now()
	rc.rooms[room.VNum] = stored
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
	interval := rc.ttl / 2
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
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

// Close stops the cleanup goroutine. It is safe to call more than once.
func (rc *RoomCache) Close() {
	rc.stopOnce.Do(func() { close(rc.stop) })
}
