package session

import (
	"net/http"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeFleeTestManager builds a Manager with two connected rooms and a target mob.
func makeFleeTestManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Flee Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
		},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			LongDesc:  "A test target stands here.",
			Level:     5,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
			Alignment: 0,
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	m := NewManager(w, nil)
	m.combatEngine = combat.NewCombatEngine()
	return m
}

func makeFleeSession(t *testing.T, m *Manager, name string, level int) *Session {
	t.Helper()
	s := &Session{
		conn:           nil,
		request:        &http.Request{},
		manager:        m,
		send:           make(chan []byte, 256),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
	}
	p := game.NewPlayer(1, name, 1001)
	p.SetLevel(level)
	p.SetExp(10000)
	p.SetMove(100)
	s.player = p
	s.playerName = name
	s.authenticated = true
	if err := m.world.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return s
}

func TestCmdFleeMovement_XPLossAtLowLevel(t *testing.T) {
	m := makeFleeTestManager(t)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	s := makeFleeSession(t, m, "Fleer", 5)
	mob.SetHealth(50) // opponent missing 50 HP
	m.combatEngine.StartCombat(s.player, mob)

	preExp := s.player.GetExp()
	if err := cmdFleeMovement(s); err != nil {
		t.Fatalf("cmdFleeMovement returned error: %v", err)
	}

	if s.player.GetExp() >= preExp {
		t.Errorf("expected XP loss at level 5, exp unchanged at %d", s.player.GetExp())
	}
}

func TestCmdFleeMovement_XPLossCapped(t *testing.T) {
	m := makeFleeTestManager(t)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	s := makeFleeSession(t, m, "Fleer", 50)
	// Give player huge exp so the cap is the limiting factor.
	s.player.SetExp(1000000)
	mob.SetHealth(1) // opponent missing 99 HP
	m.combatEngine.StartCombat(s.player, mob)

	preExp := s.player.GetExp()
	if err := cmdFleeMovement(s); err != nil {
		t.Fatalf("cmdFleeMovement returned error: %v", err)
	}

	actualLoss := preExp - s.player.GetExp()
	if actualLoss <= 0 {
		t.Fatalf("expected XP loss, got none")
	}
	if actualLoss > 500000 {
		t.Errorf("XP loss %d exceeds max_exp_loss cap of 500000", actualLoss)
	}
}
