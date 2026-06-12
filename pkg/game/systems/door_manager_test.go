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
