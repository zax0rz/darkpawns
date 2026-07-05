package systems

import (
	"sync"
	"testing"
)

func TestConcurrentOpenClose(t *testing.T) {
	dm := NewDoorManager()

	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    true,
		Locked:    false,
		Hp:        100,
		MaxHp:     100,
	}
	dm.AddDoor(door)

	var wg sync.WaitGroup
	numGoroutines := 100
	numIterations := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				if id%2 == 0 {
					_, _ = dm.OpenDoor(100, "north")
				} else {
					_, _ = dm.CloseDoor(100, "north")
				}
			}
		}(i)
	}

	wg.Wait()

	// Ensure we can safely fetch the status and it's consistent
	status, exists := dm.GetDoorStatus(100, "north")
	if !exists {
		t.Fatal("door should exist")
	}
	if status != "open" && status != "closed" {
		t.Errorf("unexpected final status: %s", status)
	}
}

// TestConcurrentCanPassAndToggle runs CanPass/GetDoorStatus readers concurrently
// with OpenDoor/CloseDoor mutators to catch data races on Door state.
func TestConcurrentCanPassAndToggle(t *testing.T) {
	dm := NewDoorManager()

	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    true,
		Locked:    false,
		Hp:        100,
		MaxHp:     100,
	}
	dm.AddDoor(door)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = dm.CanPass(100, "north")
				_, _ = dm.GetDoorStatus(100, "north")
				_ = dm.GetVisibleDoorsInRoom(100)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = dm.OpenDoor(100, "north")
				_, _ = dm.CloseDoor(100, "north")
			}
		}()
	}
	wg.Wait()
}

// TestDoorManager_GetDoorReturnsCopy verifies that GetDoor returns an
// independent snapshot: mutating the returned value must not affect the
// door held by the DoorManager (DP-698).
func TestDoorManager_GetDoorReturnsCopy(t *testing.T) {
	dm := NewDoorManager()
	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    true,
		Locked:    true,
		Hp:        100,
	}
	dm.AddDoor(door)

	snapshot, ok := dm.GetDoor(100, "north")
	if !ok {
		t.Fatal("GetDoor() should find the door")
	}

	snapshot.Closed = false
	snapshot.Locked = false
	snapshot.Hp = 0

	if snapshot.Closed || snapshot.Locked || snapshot.Hp != 0 {
		t.Fatal("mutation of the snapshot's own fields should have taken effect")
	}
	if !door.Closed || !door.Locked || door.Hp != 100 {
		t.Error("mutating the returned snapshot should not affect the underlying door")
	}
}

// TestDoorManager_ConcurrentAccess runs GetDoor readers concurrently with
// OpenDoor writers under -race to confirm no data race occurs now that
// GetDoor returns a value snapshot instead of a shared *Door (DP-698).
func TestDoorManager_ConcurrentAccess(t *testing.T) {
	dm := NewDoorManager()
	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    true,
	}
	dm.AddDoor(door)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				snapshot, ok := dm.GetDoor(100, "north")
				if ok {
					_ = snapshot.Closed
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = dm.OpenDoor(100, "north")
				_, _ = dm.CloseDoor(100, "north")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentBashAndLock(t *testing.T) {
	dm := NewDoorManager()

	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    true,
		Locked:    false,
		Bashable:  true,
		Hp:        100,
		MaxHp:     100,
		KeyVNum:   500,
	}
	dm.AddDoor(door)

	var wg sync.WaitGroup
	numBASH := 50
	numLOCK := 50

	for i := 0; i < numBASH; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = dm.BashDoor(100, "north", 50)
			}
		}()
	}

	for i := 0; i < numLOCK; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = dm.LockDoor(100, "north", 500)
				_, _ = dm.UnlockDoor(100, "north", 500)
			}
		}()
	}

	wg.Wait()

	// Verify no panics/races occurred and the door HP is at least 0
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if door.Hp < 0 {
		t.Errorf("expected door Hp >= 0, got %d", door.Hp)
	}
}
