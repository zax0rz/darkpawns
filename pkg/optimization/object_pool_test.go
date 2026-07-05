package optimization

import (
	"testing"
)

func TestObjectPool_ClearWithBorrowedObjects(t *testing.T) {
	create := func() interface{} { return "object" }
	reset := func(interface{}) {}
	validate := func(interface{}) bool { return true }

	pool := NewObjectPool(create, reset, validate, 2)

	obj := pool.Get()
	if obj == nil {
		t.Fatal("expected Get to return an object")
	}

	// Clear while an object is still borrowed should not reset counters.
	pool.Clear()

	// Returning the borrowed object must not make borrowed negative.
	pool.Put(obj)

	stats := pool.Stats()
	if borrowed, ok := stats["borrowed"].(int); !ok || borrowed != 0 {
		t.Errorf("expected borrowed=0 after Put, got %v", stats["borrowed"])
	}
	if created, ok := stats["created"].(int); !ok || created != 1 {
		t.Errorf("expected created=1 after Put, got %v", stats["created"])
	}
}

func TestObjectPool_TryGet_NoDeadlock(t *testing.T) {
	createdCount := 0
	create := func() interface{} {
		createdCount++
		return "test_object"
	}
	reset := func(interface{}) {}
	validate := func(interface{}) bool { return true }

	pool := NewObjectPool(create, reset, validate, 2)

	// Test TryGet on empty pool under max size (should create one)
	obj, ok := pool.TryGet()
	if !ok {
		t.Fatal("expected TryGet to return true")
	}
	if obj != "test_object" {
		t.Errorf("expected 'test_object', got %v", obj)
	}
	if createdCount != 1 {
		t.Errorf("expected createdCount to be 1, got %d", createdCount)
	}

	// Test TryGet again (should create another one)
	obj2, ok := pool.TryGet()
	if !ok {
		t.Fatal("expected TryGet to return true")
	}
	if obj2 != "test_object" {
		t.Errorf("expected 'test_object', got %v", obj2)
	}
	if createdCount != 2 {
		t.Errorf("expected createdCount to be 2, got %d", createdCount)
	}

	// Test TryGet when pool size is exhausted (should return false, because size = 2 and 2 are active/created)
	_, ok = pool.TryGet()
	if ok {
		t.Error("expected TryGet to return false when pool is exhausted")
	}

	// Put one back
	pool.Put(obj)

	// TryGet should now succeed using the returned object without creating a new one
	obj3, ok := pool.TryGet()
	if !ok {
		t.Fatal("expected TryGet to succeed after Put")
	}
	if obj3 != "test_object" {
		t.Errorf("expected 'test_object', got %v", obj3)
	}
	if createdCount != 2 {
		t.Errorf("expected createdCount to remain 2, got %d", createdCount)
	}
}
