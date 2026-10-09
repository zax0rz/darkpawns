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

	m := newTestManager(t, w, nil)
	m.combatEngine.Stop()
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
