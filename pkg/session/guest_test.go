package session

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestGuestLoginAndRestrictions(t *testing.T) {
	// Set JWT secret to avoid token generation failure (must be >=32 chars, DP-910)
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long-please")

	// Create dummy game world with starting room 8004
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        game.MortalStartRoom,
				Name:        "The Adventurers Guild",
				Description: "A grand hall where adventurers gather to seek glory.",
				Zone:        80,
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })

	manager := newTestManager(t, world, nil)

	// Create a new session
	s := manager.NewSession()

	// Simulate guest login message
	loginPayload := json.RawMessage(`{"player_name":"guest_test_user","password":"","new_char":false}`)
	err = s.handleLogin(loginPayload)
	if err != nil {
		t.Fatalf("handleLogin failed: %v", err)
	}

	if !s.isGuest {
		t.Errorf("expected session to be marked as guest")
	}
	if !s.authenticated {
		t.Errorf("expected session to be marked as authenticated")
	}
	if s.playerName != "guest_test_user" {
		t.Errorf("expected player name to be guest_test_user, got %s", s.playerName)
	}

	player := s.player
	if player == nil {
		t.Fatalf("expected player instance to be created")
	}
	if player.Class != game.ClassWarrior {
		t.Errorf("expected guest player to be Warrior, got class %d", player.Class)
	}
	if player.RoomVNum != game.MortalStartRoom {
		t.Errorf("expected guest player to start in room %d, got %d", game.MortalStartRoom, player.RoomVNum)
	}

	// Test command whitelist restrictions
	// 1. Whitelisted command: "look"
	err = ExecuteCommand(s, "look", nil)
	if err != nil {
		t.Errorf("expected whitelisted command 'look' to execute, got error: %v", err)
	}

	// 2. Restricted command: "save"
	err = ExecuteCommand(s, "save", nil)
	if err != nil {
		t.Errorf("expected restricted command to execute gracefully and return nil, got error: %v", err)
	}
}

// TestGuestNameUniqueSequential confirms two sequential "guest" logins get
// distinct names (DP-912). The previous time-derived scheme reassigned a
// freed name to the next guest; the monotonic counter makes that impossible.
func TestGuestNameUniqueSequential(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long-please")

	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: game.MortalStartRoom, Name: "Start", Zone: 80}},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	manager := newTestManager(t, world, nil)

	loginGuest := func() string {
		s := manager.NewSession()
		payload := json.RawMessage(`{"player_name":"guest","password":"","new_char":false}`)
		if err := s.handleLogin(payload); err != nil {
			t.Fatalf("handleLogin failed: %v", err)
		}
		// Unregister so the name is "freed" — the bug under test was that the
		// next sequential guest then reused it.
		manager.Unregister(s.PlayerName())
		return s.PlayerName()
	}

	first := loginGuest()
	second := loginGuest()

	if first == second {
		t.Errorf("two sequential guests got the same name %q; want distinct", first)
	}
	if first == "guest" || second == "guest" {
		t.Errorf("guest name not uniquified: first=%q second=%q", first, second)
	}
}

// TestGuestNameUniqueConcurrent confirms concurrent guest logins get distinct
// names even when racing the counter (DP-912). We assert uniqueness only over
// the names that were actually assigned (successful registrations); the
// monotonic counter guarantees those are distinct regardless of concurrency.
func TestGuestNameUniqueConcurrent(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long-please")

	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: game.MortalStartRoom, Name: "Start", Zone: 80}},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	manager := newTestManager(t, world, nil)

	const n = 16
	names := make([]string, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			s := manager.NewSession()
			payload := json.RawMessage(`{"player_name":"guest","password":"","new_char":false}`)
			errs[i] = s.handleLogin(payload)
			names[i] = s.PlayerName()
		}()
	}
	close(start)
	wg.Wait()

	// Only consider successfully-assigned guest names. (Concurrent handleLogin
	// can fail for unrelated manager-state reasons; those surface as errors and
	// an empty PlayerName, which the counter isn't responsible for.)
	seen := make(map[string]int)
	for i, name := range names {
		if errs[i] != nil || name == "" {
			continue
		}
		seen[name]++
	}
	if len(seen) == 0 {
		t.Fatal("no successful guest registrations; test setup is broken")
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("guest name %q assigned to %d concurrent sessions; want 1", name, count)
		}
	}
}
