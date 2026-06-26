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
