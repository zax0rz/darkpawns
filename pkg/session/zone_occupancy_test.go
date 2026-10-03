package session

import (
	"reflect"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneOccupancyDescriptorBodies(t *testing.T) {
	m, s, _ := switchGateFixture(t)
	check := func(want []int) {
		t.Helper()
		if got := m.world.OccupiedZoneRooms(); !reflect.DeepEqual(got, want) {
			t.Fatalf("occupied descriptor rooms = %v, want %v", got, want)
		}
	}
	check([]int{1001})
	m.mu.Lock()
	s.authenticated = false
	m.mu.Unlock()
	check([]int{})
	m.mu.Lock()
	s.authenticated = true
	m.mu.Unlock()
	s.superseded.Store(true)
	check([]int{})
	s.superseded.Store(false)
	check([]int{1001}) // retained linkdead body in 1002 is not a descriptor
	if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
		t.Fatal(err)
	}
	check([]int{1002}) // original body's room is not occupied by this descriptor
	m.mu.Lock()
	s.menuActive = true
	m.mu.Unlock()
	check([]int{})
	m.mu.Lock()
	s.menuActive = false
	s.charCreating = true
	m.mu.Unlock()
	check([]int{})
	m.mu.Lock()
	s.charCreating = false
	s.switchedMob = game.NewMob(&parser.Mob{VNum: 7}, 3001)
	m.mu.Unlock()
	check([]int{3001}) // NPC switch occupies the mobile's room
	s.DetachTransport()
	check([]int{})
}

func TestZoneOccupancyDoesNotAcquireLifecycle(t *testing.T) {
	m, _, _ := switchGateFixture(t)
	m.playerLifecycleMu.Lock()
	defer m.playerLifecycleMu.Unlock()
	if got := m.world.OccupiedZoneRooms(); !reflect.DeepEqual(got, []int{1001}) {
		t.Fatalf("ordinary descriptor: %v", got)
	}
}

func TestZoneOccupancyConcurrentMenuAndSwitch(t *testing.T) {
	m, s, _ := switchGateFixture(t)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			s.showMainMenu()
			s.clearMenuState()
		}
	})
	wg.Go(func() {
		for range 100 {
			_ = m.world.OccupiedZoneRooms()
		}
	})
	wg.Wait()
	wg.Go(func() {
		for range 100 {
			if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
				t.Error(err)
			}
			if err := cmdReturn(s, nil); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Go(func() {
		for range 100 {
			_ = m.world.OccupiedZoneRooms()
		}
	})
	wg.Wait()
}
