package session

import (
	"net/http"
	"strings"
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

func TestCmdFlee_CanonicalCombatStateAndSilentXPLoss(t *testing.T) {
	m := makeFleeTestManager(t)
	room, ok := m.world.GetRoom(1001)
	if !ok {
		t.Fatal("source room missing")
	}
	room.Exits = make(map[string]parser.Exit, 6)
	for _, direction := range []string{"north", "east", "south", "west", "up", "down"} {
		room.Exits[direction] = parser.Exit{Direction: direction, ToRoom: 1002}
	}

	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetHealth(mob.GetMaxHP() - 50)
	s := makeFleeSession(t, m, "CanonicalFleer", 5)
	if err := m.combatEngine.StartCombat(s.player, mob); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	preExp := s.player.GetExp()
	if err := cmdFlee(s); err != nil {
		t.Fatalf("cmdFlee: %v", err)
	}
	if got, want := preExp-s.player.GetExp(), 50*mob.GetLevel(); got != want {
		t.Fatalf("XP loss = %d, want %d", got, want)
	}
	if s.player.GetRoom() != 1002 {
		t.Fatalf("room = %d, want 1002", s.player.GetRoom())
	}
	if m.combatEngine.IsFighting(s.player.Name) {
		t.Fatal("combat still active after successful flee")
	}
	if output := drainSendChannel(t, s); strings.Contains(output, "experience points for fleeing") {
		t.Fatalf("invented XP-loss message in output: %s", output)
	}
}

func TestCmdFlee_CanonicalHighLevelXPBonus(t *testing.T) {
	m := makeFleeTestManager(t)
	room, ok := m.world.GetRoom(1001)
	if !ok {
		t.Fatal("source room missing")
	}
	room.Exits = make(map[string]parser.Exit, 6)
	for _, direction := range []string{"north", "east", "south", "west", "up", "down"} {
		room.Exits[direction] = parser.Exit{Direction: direction, ToRoom: 1002}
	}
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetHealth(mob.GetMaxHP() - 50)
	s := makeFleeSession(t, m, "VeteranFleer", 11)
	if err := m.combatEngine.StartCombat(s.player, mob); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	preExp := s.player.GetExp()
	if err := cmdFlee(s); err != nil {
		t.Fatalf("cmdFlee: %v", err)
	}
	base := 50 * mob.GetLevel()
	bonus := int(500 * (float64(s.player.GetLevel()) / 2.6))
	if got, want := preExp-s.player.GetExp(), base+bonus; got != want {
		t.Fatalf("XP loss = %d, want base %d + bonus %d = %d", got, base, bonus, want)
	}
}

func TestCmdFlee_ThiefAutomaticCallHonorsWaitGate(t *testing.T) {
	m := makeFleeTestManager(t)
	s := makeFleeSession(t, m, "WaitingThief", 5)
	s.player.Class = game.ClassThief
	s.player.SetWaitState(1)

	if err := cmdFlee(s); err != nil {
		t.Fatalf("cmdFlee: %v", err)
	}
	if s.player.GetRoom() != 1001 {
		t.Fatalf("waiting thief moved to room %d", s.player.GetRoom())
	}
	if output := drainSendChannel(t, s); !strings.Contains(output, "You attempt to flee but cannot!") {
		t.Fatalf("missing C wait-gate message: %s", output)
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
