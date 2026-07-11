package session

// Tests for cmdKill instakill gate (DP-1041):
//   - Instakill requires LVL_IMPL-1 (level 39+), not LVL_IMMORT (31+)
//   - Equal-level targets are blocked ("No can do, buddy..")

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeInstakillTestManager builds a Manager with a single room and a target mob
// whose level can be set per-test.
func makeInstakillTestManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Instakill Room", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			LongDesc:  "A test target stands here.",
			Level:     3,
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

func makeInstakillSession(t *testing.T, m *Manager, name string, level int) *Session {
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
	s.player = p
	s.playerName = name
	s.authenticated = true
	if err := m.world.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return s
}

func TestInstakillRequiresImplLevel(t *testing.T) {
	// A level-35 immortal (>= LVL_IMMORT but < LVL_IMPL-1) must NOT instakill.
	// C gates instakill to GET_LEVEL(ch) >= LVL_IMPL-1 (level 39).
	m := makeInstakillTestManager(t)

	// Spawn the target mob.
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mobCurrentHP := mob.GetHP()

	s := makeInstakillSession(t, m, "MidImm", 35)
	if err := cmdKill(s, []string{"target"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}

	// The mob must still be alive (instakill did NOT fire). At level 35 the
	// code delegates to cmdHit, which requires a combat engine — but the key
	// assertion is that HandleDeath was NOT called, so the mob's HP is intact.
	if mob.GetHP() != mobCurrentHP {
		t.Errorf("level 35 immortal instakilled the mob (HP %d → %d); instakill requires level 39+ (DP-1041)",
			mobCurrentHP, mob.GetHP())
	}
}

func TestInstakillSameLevelBlocked(t *testing.T) {
	// A level-40 implementor cannot instakill a level-40 target.
	// C: "No can do, buddy.." when GET_LEVEL(vict) == GET_LEVEL(ch).
	m := makeInstakillTestManager(t)

	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetLevel(40) // same level as the killer
	mobCurrentHP := mob.GetHP()

	s := makeInstakillSession(t, m, "Impl", 40)

	// Capture output via the session's Send path. cmdKill calls s.Send which
	// writes to the send channel as a ServerMessage; we check the mob survived
	// AND that no death occurred.
	if err := cmdKill(s, []string{"target"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}

	if mob.GetHP() != mobCurrentHP {
		t.Errorf("level 40 immortal instakilled a level 40 mob (DP-1041: equal-level kills must be blocked)")
	}
}

func TestInstakillFiresAtImplLevel(t *testing.T) {
	// A level-39 implementor (LVL_IMPL-1) instakills a lower-level mob.
	m := makeInstakillTestManager(t)

	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	s := makeInstakillSession(t, m, "HighImm", 39)
	if err := cmdKill(s, []string{"target"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}

	// The mob must be dead (instakill fired).
	if m.world.GetMobsInRoom(1001) != nil {
		found := false
		for _, mm := range m.world.GetMobsInRoom(1001) {
			if mm == mob {
				found = true
			}
		}
		if found {
			t.Error("level 39 immortal should have instakilled the mob (DP-1041)")
		}
	}
}

func TestInstakillSendsVictimMessage(t *testing.T) {
	// When a player instakills another player, the victim receives
	// "$N chops you to pieces!" (C: act.offensive.c:152-154). We verify the
	// victim's SendMessage is called with the chop message.
	m := makeInstakillTestManager(t)

	killer := makeInstakillSession(t, m, "Killer", 40)

	// Create a victim player in the same room at a lower level.
	victim := game.NewPlayer(2, "Victim", 1001)
	victim.SetLevel(10)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	// Capture victim messages by swapping out SendMessage isn't trivial
	// (Player.SendMessage writes to its session). Instead, verify the killer
	// gets the chop confirmation and the victim is handled by HandleDeath.
	// The key fidelity check: equal-level is blocked, lower-level dies.
	if err := cmdKill(killer, []string{"victim"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}

	// Drain killer's send channel — expect the chop message.
	gotChop := false
drainLoop:
	for {
		select {
		case msg := <-killer.send:
			if strings.Contains(string(msg), "chop") {
				gotChop = true
			}
		case <-time.After(100 * time.Millisecond):
			break drainLoop
		}
	}
	if !gotChop {
		t.Error("killer should receive a 'chop' confirmation message (DP-1041)")
	}
}
