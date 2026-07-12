package session

// Tests for DP-1045 (partial): peaceful-room and low-level PC protection gates
// in cmdHit. Covers fight.c:1336-1357.
//
// Mob redirects (jail-guard, charm-retarget, high-level switcheroo) are OUT OF
// SCOPE — they need combat-engine retargeting, not a command-layer guard.

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeGateTestManager builds a Manager with a room whose flags can be customized.
// If peaceful is true, the room gets the "peaceful" flag.
func makeGateTestManager(t *testing.T, peaceful bool) *Manager {
	t.Helper()
	flags := []string{}
	if peaceful {
		flags = append(flags, "peaceful")
	}
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Gate Test Room", Zone: 1, Flags: flags},
		},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			LongDesc:  "A test target stands here.",
			Level:     15,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
			Race:      1,
		}},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return NewManager(w, nil)
}

// makeGateSession creates a session for a player at the given level in room 1001.
func makeGateSession(t *testing.T, m *Manager, id int, name string, level int) *Session {
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
	p := game.NewPlayer(id, name, 1001)
	p.SetLevel(level)
	s.player = p
	s.playerName = name
	s.authenticated = true
	if err := m.world.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return s
}

// drainSend reads all pending messages from s.send and returns true if any
// contains substr.
func drainSendContains(s *Session, substr string) bool {
	found := false
	for {
		select {
		case msg := <-s.send:
			if strings.Contains(strings.ToLower(string(msg)), strings.ToLower(substr)) {
				found = true
			}
		case <-time.After(200 * time.Millisecond):
			return found
		}
	}
}

// ---------------------------------------------------------------------------
// Peaceful room gates
// ---------------------------------------------------------------------------

func TestPeacefulRoom_BlocksAttackOnMob(t *testing.T) {
	// Case 1: peaceful room blocks attacking a mob → no combat, peaceful msg.
	m := makeGateTestManager(t, true)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	_ = mob

	s := makeGateSession(t, m, 1, "Hero", 20)
	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if m.combatEngine.IsFighting("Hero") {
		t.Error("combat should not start in a peaceful room")
	}
	if !drainSendContains(s, "peaceful") {
		t.Error("expected 'peaceful, easy feeling' message")
	}
}

func TestPeacefulRoom_BlocksAttackOnPlayer(t *testing.T) {
	// Case 2: peaceful room blocks attacking a player.
	m := makeGateTestManager(t, true)
	attacker := makeGateSession(t, m, 1, "Attacker", 20)
	victim := game.NewPlayer(2, "Victim", 1001)
	victim.SetLevel(20)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"victim"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if m.combatEngine.IsFighting("Attacker") {
		t.Error("combat should not start in a peaceful room (PC target)")
	}
	if !drainSendContains(attacker, "peaceful") {
		t.Error("expected 'peaceful, easy feeling' message")
	}
}

func TestPeacefulRoom_AllowsRetaliation(t *testing.T) {
	// Case 3: peaceful room does NOT block when victim is already fighting the
	// attacker (FIGHTING(victim) == ch → retaliation allowed).
	m := makeGateTestManager(t, true)
	attacker := makeGateSession(t, m, 1, "Hero", 20)
	victim := game.NewPlayer(2, "Bully", 1001)
	victim.SetLevel(20)
	victim.SetFighting("Hero") // victim is already fighting the attacker
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"bully"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting("Hero") {
		t.Error("retaliation should be allowed even in a peaceful room")
	}
}

func TestPeacefulRoom_AllowsOutlawTarget(t *testing.T) {
	// Case 4: peaceful room does NOT block when the victim is an outlaw.
	m := makeGateTestManager(t, true)
	attacker := makeGateSession(t, m, 1, "Hero", 20)
	victim := game.NewPlayer(2, "Bandit", 1001)
	victim.SetLevel(20)
	victim.SetPlrFlag(game.PlrOutlaw, true) // outlaw — no protection
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"bandit"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting("Hero") {
		t.Error("attacking an outlaw should be allowed even in a peaceful room")
	}
}

// ---------------------------------------------------------------------------
// Low-level PC protection gates (PC vs PC only)
// ---------------------------------------------------------------------------

func TestLowLevelGate_AttackerTooLow(t *testing.T) {
	// Case 5: attacker level ≤ 10 vs a player → "not experienced enough".
	m := makeGateTestManager(t, false)
	attacker := makeGateSession(t, m, 1, "Rookie", 5)
	victim := game.NewPlayer(2, "Highlevel", 1001)
	victim.SetLevel(20)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"highlevel"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if m.combatEngine.IsFighting("Rookie") {
		t.Error("level 5 attacker should be blocked from attacking a player")
	}
	if !drainSendContains(attacker, "not experienced") {
		t.Error("expected 'not experienced enough' message")
	}
}

func TestLowLevelGate_VictimTooLow(t *testing.T) {
	// Case 6: victim player level ≤ 10 (non-outlaw) → "Ancient forces protect".
	m := makeGateTestManager(t, false)
	attacker := makeGateSession(t, m, 1, "Bully", 20)
	victim := game.NewPlayer(2, "Newbie", 1001)
	victim.SetLevel(5)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"newbie"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if m.combatEngine.IsFighting("Bully") {
		t.Error("attacking a level 5 player should be blocked")
	}
	if !drainSendContains(attacker, "ancient forces protect") {
		t.Error("expected 'Ancient forces protect' message")
	}
}

func TestLowLevelGate_VictimLowButOutlaw(t *testing.T) {
	// Case 7: victim level ≤ 10 AND outlaw → protection waived, combat proceeds.
	m := makeGateTestManager(t, false)
	attacker := makeGateSession(t, m, 1, "Hero", 20)
	victim := game.NewPlayer(2, "Outlaw", 1001)
	victim.SetLevel(5)
	victim.SetPlrFlag(game.PlrOutlaw, true)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"outlaw"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting("Hero") {
		t.Error("attacking a level 5 outlaw should be allowed (protection waived)")
	}
}

func TestLowLevelGate_DoesNotApplyToMobs(t *testing.T) {
	// Case 8: low-level gate is PC-vs-PC only — a level 5 player CAN attack a mob.
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	_ = mob

	s := makeGateSession(t, m, 1, "Rookie", 5)
	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting("Rookie") {
		t.Error("level 5 player should be able to attack a mob (low-level gate is PC-vs-PC only)")
	}
}

func TestPositiveControl_NormalAttackProceeds(t *testing.T) {
	// Case 9: level-20 player attacks a mob in a normal room → combat starts.
	// Regression guard that the gates don't over-block.
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	_ = mob

	s := makeGateSession(t, m, 1, "Veteran", 20)
	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting("Veteran") {
		t.Error("level 20 attacking a mob in a normal room should start combat")
	}
}
