package agentcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStateFileGetReturnsDeepCopy(t *testing.T) {
	tmp := t.TempDir()
	sf := &StateFile{
		path:  filepath.Join(tmp, "state.json"),
		state: &GameState{},
	}

	if err := sf.Update(func(s *GameState) {
		s.Player.Health = 100
		s.Room.Mobs = []Mob{{Name: "rat"}}
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	copy := sf.Get()
	copy.Player.Health = 1
	copy.Room.Mobs[0].Name = "dragon"

	fresh := sf.Get()
	if fresh.Player.Health != 100 {
		t.Errorf("Get copy mutated internal Player.Health: got %d", fresh.Player.Health)
	}
	if len(fresh.Room.Mobs) != 1 || fresh.Room.Mobs[0].Name != "rat" {
		t.Errorf("Get copy mutated internal Room.Mobs: got %+v", fresh.Room.Mobs)
	}
}

func TestStateFileConcurrentGetAndUpdate(t *testing.T) {
	tmp := t.TempDir()
	sf := &StateFile{
		path:  filepath.Join(tmp, "state.json"),
		state: &GameState{},
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if err := sf.Update(func(s *GameState) {
					s.Player.Health = i*1000 + j
					s.Room.Name = "room"
				}); err != nil {
					t.Errorf("update: %v", err)
				}
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s := sf.Get()
				if _, err := json.Marshal(s); err != nil {
					t.Errorf("marshal: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	if _, err := os.Stat(sf.path); err != nil {
		t.Fatalf("state file not saved: %v", err)
	}
}
