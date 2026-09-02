package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRestoreRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["restore"]
	if !ok {
		t.Fatal("restore command has no C gate")
	}
	if gate.MinLevel != LVL_GOD-1 || gate.MinPosition != combat.PosDead {
		t.Fatalf("restore gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GOD-1, combat.PosDead)
	}
	entry, ok := cmdRegistry.Lookup("restore")
	if !ok {
		t.Fatal("restore command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("restore registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestCmdRestoreMatchesCEntryAndTargetMessages(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Restoregod", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Restoredtarget", 1, 1001)
	observer := makeCommandTestSession(t, m, "Restoreobserver", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)
	registerInWorld(t, observer)

	target.player.SetHP(1)
	target.player.SetMana(2)
	target.player.SetMove(3)
	target.player.SetPosition(combat.PosStunned)

	if err := cmdRestore(actor, []string{"the", "Restoredtarget", "trailing", "words"}); err != nil {
		t.Fatalf("cmdRestore: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("actor response = %q, want %q", got, want)
	}
	if got, want := readMsgText(t, target), "The hand of Restoregod touches you, healing your wounds and leaving you refreshed!\r\n"; got != want {
		t.Fatalf("target response = %q, want %q", got, want)
	}
	assertRestoreNoMessage(t, observer)

	if target.player.GetHP() != target.player.GetMaxHP() || target.player.GetMana() != target.player.GetMaxMana() || target.player.GetMove() != target.player.GetMaxMove() {
		t.Fatalf("target resources = (%d, %d, %d), want max (%d, %d, %d)", target.player.GetHP(), target.player.GetMana(), target.player.GetMove(), target.player.GetMaxHP(), target.player.GetMaxMana(), target.player.GetMaxMove())
	}
	if got, want := target.player.GetPosition(), combat.PosStanding; got != want {
		t.Fatalf("target position = %d, want %d after positive-HP update_pos", got, want)
	}
}

func TestCmdRestoreUsesCNotFoundText(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Restoregod", LVL_IMPL, 1001)
	registerInWorld(t, actor)

	if err := cmdRestore(actor, []string{"Nobody"}); err != nil {
		t.Fatalf("cmdRestore: %v", err)
	}
	if got, want := readMsgText(t, actor), "No-one by that name here.\r\n"; got != want {
		t.Fatalf("missing-target response = %q, want %q", got, want)
	}
}

func TestCmdRestoreRestoresVisibleMob(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	actor := makeCommandTestSession(t, m, "Restoregod", LVL_IMPL, 1001)
	registerInWorld(t, actor)
	mob := registerMob(t, m, 2001, 1001)
	_ = readMsgText(t, actor) // discard C/Go mob-arrival setup output
	mob.SetHealth(1)
	mob.SetMana(2)
	mob.SetMove(3)
	mob.SetPosition(combat.PosStunned)

	if err := cmdRestore(actor, []string{"guard"}); err != nil {
		t.Fatalf("cmdRestore: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("mob actor response = %q, want %q", got, want)
	}
	if mob.GetHP() != mob.GetMaxHP() || mob.GetMana() != mob.GetMaxMana() || mob.GetMove() != mob.GetMaxMove() {
		t.Fatalf("mob resources = (%d, %d, %d), want max (%d, %d, %d)", mob.GetHP(), mob.GetMana(), mob.GetMove(), mob.GetMaxHP(), mob.GetMaxMana(), mob.GetMaxMove())
	}
	if got, want := mob.GetPosition(), combat.PosStanding; got != want {
		t.Fatalf("mob position = %d, want %d after positive-HP update_pos", got, want)
	}
}

func TestCmdRestoreMatchesCHighImmortalStateBranch(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Restoregrgod", LVL_GRGOD, 1001)
	target := makeCommandTestSession(t, m, "Restoreimmortal", LVL_GRGOD, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	target.player.Stats = game.CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
	target.player.Strength = 10
	target.player.SetSkill("kick", 7)
	target.player.SetHP(1)
	target.player.SetMana(2)
	target.player.SetMove(3)
	target.player.SetPosition(combat.PosDead)

	if err := cmdRestore(actor, []string{"Restoreimmortal"}); err != nil {
		t.Fatalf("cmdRestore: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("high-immortal actor response = %q, want %q", got, want)
	}
	if got, want := readMsgText(t, target), "The hand of Restoregrgod touches you, healing your wounds and leaving you refreshed!\r\n"; got != want {
		t.Fatalf("high-immortal target response = %q, want %q", got, want)
	}
	if got := target.player.GetSkill("kick"); got != 100 {
		t.Fatalf("restored skill = %d, want 100", got)
	}
	wantStats := game.CharStats{Str: 25, StrAdd: 100, Int: 25, Wis: 25, Dex: 25, Con: 25, Cha: 25}
	if target.player.Stats != wantStats {
		t.Fatalf("restored stats = %+v, want %+v", target.player.Stats, wantStats)
	}
	if target.player.Strength != 25 {
		t.Fatalf("restored inventory strength = %d, want 25", target.player.Strength)
	}
	if target.player.GetPosition() != combat.PosStanding {
		t.Fatalf("restored high immortal position = %d, want standing", target.player.GetPosition())
	}
}

func assertRestoreNoMessage(t *testing.T, s *Session) {
	t.Helper()
	select {
	case msg := <-s.send:
		t.Fatalf("unexpected restore audience message: %s", msg)
	case <-time.After(50 * time.Millisecond):
	}
}
