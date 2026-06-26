package optimization

import (
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
