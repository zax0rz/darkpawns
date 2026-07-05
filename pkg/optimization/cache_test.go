package optimization

import (
	"sync"
	"testing"
	"time"
)

func TestCacheCloseIsIdempotent(t *testing.T) {
	cache := NewCache(time.Minute)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Cache.Close panicked on second call: %v", r)
		}
	}()

	cache.Close()
	cache.Close()
}

func TestRoomCacheCloseIsIdempotent(t *testing.T) {
	cache := NewRoomCache(time.Minute)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RoomCache.Close panicked on second call: %v", r)
		}
	}()

	cache.Close()
	cache.Close()
}

func TestCacheZeroTTLDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewCache(0) panicked: %v", r)
		}
	}()

	cache := NewCache(0)
	cache.Set("key", "value")
	if _, ok := cache.Get("key"); ok {
		t.Error("expected zero-TTL item to be treated as expired")
	}
	cache.Close()
}

func TestCacheSmallTTLDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewCache(1ns) panicked: %v", r)
		}
	}()

	cache := NewCache(time.Nanosecond)
	cache.Close()
}

func TestRoomCacheSmallTTLDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewRoomCache(1ns) panicked: %v", r)
		}
	}()

	cache := NewRoomCache(time.Nanosecond)
	cache.Close()
}

func TestRoomCacheZeroTTLDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewRoomCache(0) panicked: %v", r)
		}
	}()

	cache := NewRoomCache(0)
	_, _ = cache.GetRoom(1, func(vnum int) (*CachedRoom, error) {
		return &CachedRoom{VNum: vnum, CachedAt: time.Now()}, nil
	})
	cache.Close()
}

func TestCache_GetExpired(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)
	defer cache.Close()

	cache.Set("key", "value")
	if _, ok := cache.Get("key"); !ok {
		t.Fatal("expected key to exist before expiry")
	}

	time.Sleep(75 * time.Millisecond)

	if _, ok := cache.Get("key"); ok {
		t.Error("expected expired key to return false")
	}
	if cache.Size() != 0 {
		t.Errorf("expected cache size 0 after expiry, got %d", cache.Size())
	}
}

// TestCache_GetConcurrent exercises Get, Set, and Delete concurrently.
// Run with -race to detect data races.
func TestCache_GetConcurrent(t *testing.T) {
	cache := NewCache(time.Minute)
	defer cache.Close()

	for i := 0; i < 100; i++ {
		cache.Set(string(rune('a'+i%26)), i)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				key := string(rune('a' + (id+j)%26))
				switch j % 3 {
				case 0:
					cache.Get(key)
				case 1:
					cache.Set(key, j)
				case 2:
					cache.Delete(key)
				}
			}
		}(i)
	}
	wg.Wait()
}
