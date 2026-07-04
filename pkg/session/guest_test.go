package session

import (
	"encoding/json"
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

	manager := NewManager(world, nil)

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
