package session

// Tests for DP-1045 (partial): peaceful-room and low-level PC protection gates
// in cmdHit. Covers fight.c:1336-1357.
//
// Mob redirects (jail-guard, charm-retarget, high-level switcheroo) are OUT OF
// SCOPE — they need combat-engine retargeting, not a command-layer guard.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeGateTestManager builds a Manager with a room whose flags can be customized.
// If peaceful is true, the room gets the parsed numeric ROOM_PEACEFUL bit.
func makeGateTestManager(t *testing.T, peaceful bool) *Manager {
	t.Helper()
	flags := []string{"0", "0", "0", "0"}
	if peaceful {
		flags[0] = "16"
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
	m := newTestManager(t, w, nil)
	// damage()'s protections run inside the engine's performOneHit
	// (DP-1438), so the gate path needs the production callback wiring
	// cmd/server/main.go installs — without it the gate never refuses.
	m.WireCombatCallbacks()
	return m
}

func readSendText(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case message := <-s.send:
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message, &envelope); err != nil {
			t.Fatalf("unmarshal server message: %v", err)
		}
		var event EventData
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			t.Fatalf("unmarshal event data: %v", err)
		}
		return event.Text
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for session output")
		return ""
	}
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
	m.mu.Lock()
	m.sessions[name] = s
	m.mu.Unlock()
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

	if m.combatEngine.IsFighting(s.player) {
		t.Error("combat should not start in a peaceful room")
	}
	if got, want := readSendText(t, s), "This room just has such a peaceful, easy feeling...\r\n"; got != want {
		t.Errorf("peaceful gate message = %q, want %q", got, want)
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

	if m.combatEngine.IsFighting(attacker.player) {
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
	victim.SetFightingBody(attacker.player) // victim is already fighting the attacker
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"bully"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if !m.combatEngine.IsFighting(attacker.player) {
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

	if !m.combatEngine.IsFighting(attacker.player) {
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

	if m.combatEngine.IsFighting(attacker.player) {
		t.Error("level 5 attacker should be blocked from attacking a player")
	}
	if got, want := readSendText(t, attacker), "You are not experienced enough to attack Highlevel!\r\n"; got != want {
		t.Errorf("attacker newbie gate message = %q, want %q", got, want)
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

	if m.combatEngine.IsFighting(attacker.player) {
		t.Error("attacking a level 5 player should be blocked")
	}
	if got, want := readSendText(t, attacker), "Ancient forces protect Newbie from your wrath!\r\n"; got != want {
		t.Errorf("victim newbie gate message = %q, want %q", got, want)
	}
}

func TestCombatGateOrder_PeacefulBeforeNewbieProtection(t *testing.T) {
	m := makeGateTestManager(t, true)
	attacker := makeGateSession(t, m, 1, "Rookie", 5)
	victim := game.NewPlayer(2, "Newbie", 1001)
	victim.SetLevel(5)
	if err := m.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	if err := cmdHit(attacker, []string{"newbie"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}

	if got, want := readSendText(t, attacker), "This room just has such a peaceful, easy feeling...\r\n"; got != want {
		t.Errorf("first combat gate message = %q, want %q", got, want)
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

	if !m.combatEngine.IsFighting(attacker.player) {
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

	if !m.combatEngine.IsFighting(s.player) {
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

	if !m.combatEngine.IsFighting(s.player) {
		t.Error("level 20 attacking a mob in a normal room should start combat")
	}
}

// missSeed returns a stream seed whose first number(1,20) roll is a natural
// 1 — always a miss, so the swing consumes exactly hit()'s to-hit draw and no
// damage dice. DP-1438's draw-count proofs need the count pinned.
func missSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 4096; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 20) == 1 {
			return seed
		}
	}
	t.Fatal("no seed found whose first number(1,20) is 1")
	return 0
}

// TestCmdHitRefusedSwingStillDraws pins DP-1438's core finding for do_hit: C
// draws number(1,20) inside hit() (fight.c:1815) BEFORE damage()'s protection
// block (fight.c:1335-1342), so even a swing refused by the peaceful gate
// consumes the to-hit draw and enrols nobody (act.offensive.c:126-127). The
// old preflight returned before any engine call — zero draws — which is the
// stream-4106/4087/4081/4152 mismatch; this test fails on that shape (R5h).
func TestCmdHitRefusedSwingStillDraws(t *testing.T) {
	m := makeGateTestManager(t, true)
	if _, err := m.world.SpawnMob(5000, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)

	seed := missSeed(t)
	dprng.ResetStream(seed)
	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	gotNext := dprng.Number(0, 999)

	// Reference: exactly one draw (the missed to-hit roll) precedes the probe.
	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("refused swing changed the draw stream: next=%d, want %d (one number(1,20) consumed)", gotNext, wantNext)
	}
	if m.combatEngine.IsFighting(s.player) {
		t.Error("refused swing enrolled the attacker")
	}
}

// TestCmdAssistRefusedSwingStillDraws is the assist half (act.offensive.c:95):
// the helper's first swing draws hit()'s roll before damage()'s peaceful
// refusal, and the helper enrols nobody. Fails on the preflight shape (R5h).
func TestCmdAssistRefusedSwingStillDraws(t *testing.T) {
	m := makeGateTestManager(t, true)
	opponent, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	helpee := makeGateSession(t, m, 2, "Helpee", 20)
	helper := makeGateSession(t, m, 3, "Helper", 20)
	// The opponent is already fighting the helpee, so the helper's first
	// swing faces the peaceful gate (FIGHTING(victim) is the helpee).
	if err := m.combatEngine.StartCombat(opponent, helpee.player); err != nil {
		t.Fatalf("setup StartCombat failed: %v", err)
	}

	seed := missSeed(t)
	dprng.ResetStream(seed)
	if err := cmdAssist(helper, []string{"Helpee"}); err != nil {
		t.Fatalf("cmdAssist returned error: %v", err)
	}
	gotNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("refused assist changed the draw stream: next=%d, want %d (one number(1,20) consumed)", gotNext, wantNext)
	}
	if m.combatEngine.IsFighting(helper.player) {
		t.Error("refused assist enrolled the helper")
	}
}

// TestCmdHitAllowedSwingDrawsAndEnrols pins the allowed arm: the swing runs
// through the deferred entry — to-hit roll plus the bare-hands damage draw
// (fight.c:1865 number(0, level/3)) — and enrols both sides at the damage
// point. A broken hook that skips the swing or enrols eagerly fails here.
func TestCmdHitAllowedSwingDrawsAndEnrols(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Veteran", 20)

	seed := missSeed(t)
	dprng.ResetStream(seed)
	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	gotNext := dprng.Number(0, 999)

	// Miss in an open room: C still calls damage(ch, vict, 0, w_type), which
	// enrols both sides (fight.c:1408, 1445) — one to-hit draw, no dice.
	dprng.ResetStream(seed)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("allowed swing changed the draw stream: next=%d, want %d (one number(1,20) consumed)", gotNext, wantNext)
	}
	if !m.combatEngine.IsFighting(s.player) {
		t.Error("missed first swing failed to enrol the attacker")
	}
	if mob.GetFightingBody() == nil {
		t.Error("missed first swing failed to enrol the victim")
	}
}
