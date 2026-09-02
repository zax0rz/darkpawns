package session

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRerollRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["reroll"]
	if !ok {
		t.Fatal("reroll command has no C gate")
	}
	if entry.MinLevel != LVL_GRGOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("reroll gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GRGOD, combat.PosDead)
	}
}

func TestCmdRerollNoArgumentUsesCResponse(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Rerollgod", 1001, true)
	actor.player.SetLevel(LVL_GRGOD)

	if err := cmdReroll(actor, nil); err != nil {
		t.Fatalf("cmdReroll: %v", err)
	}
	if got, want := readSessionText(t, actor), "Yes, but for whom?!?\r\n"; got != want {
		t.Fatalf("no-argument response = %q, want %q", got, want)
	}
}

func TestCmdRerollMatchesCStatRollAndState(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Rerollgod", 1001, true)
	actor.player.SetLevel(LVL_GRGOD)
	target := makeTestSession(t, m, "Rerolltarget", 1001, true)
	target.player.Class = game.ClassWarrior
	target.player.Race = game.RaceHuman
	m.mu.Lock()
	m.sessions["rerollgod"] = actor
	m.sessions["rerolltarget"] = target
	m.mu.Unlock()

	dprng.ResetStream(31)
	wantStats := game.RollRealAbils(target.player.Class, target.player.Race)
	dprng.ResetStream(31)
	if err := cmdReroll(actor, []string{"the", "Rerolltarget", "trailing", "words"}); err != nil {
		t.Fatalf("cmdReroll: %v", err)
	}

	wantOutput := "Rerolled...\r\n" + fmt.Sprintf(
		"New stats: Str %d/%d, Int %d, Wis %d, Dex %d, Con %d, Cha %d\r\n",
		wantStats.Str, wantStats.StrAdd, wantStats.Int, wantStats.Wis, wantStats.Dex, wantStats.Con, wantStats.Cha,
	)
	if got := readSessionText(t, actor) + readSessionText(t, actor); got != wantOutput {
		t.Fatalf("reroll output = %q, want %q", got, wantOutput)
	}
	if target.player.Stats != wantStats {
		t.Fatalf("target stats = %+v, want %+v", target.player.Stats, wantStats)
	}
	if target.player.OrigCon != wantStats.Con {
		t.Fatalf("target original constitution = %d, want %d", target.player.OrigCon, wantStats.Con)
	}
	if target.player.Strength != wantStats.Str {
		t.Fatalf("target inventory strength = %d, want %d", target.player.Strength, wantStats.Str)
	}
}

func TestCmdRerollRejectsHigherImmortal(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Rerollgod", 1001, true)
	actor.player.SetLevel(LVL_GRGOD)
	target := makeTestSession(t, m, "Rerollhigher", 1001, true)
	target.player.SetLevel(LVL_GRGOD + 1)
	before := target.player.Stats
	m.mu.Lock()
	m.sessions["rerollgod"] = actor
	m.sessions["rerollhigher"] = target
	m.mu.Unlock()

	if err := cmdReroll(actor, []string{"Rerollhigher"}); err != nil {
		t.Fatalf("cmdReroll: %v", err)
	}
	if got, want := readSessionText(t, actor), "Hmmm...you'd better not.\r\n"; got != want {
		t.Fatalf("higher-immortal response = %q, want %q", got, want)
	}
	if target.player.Stats != before {
		t.Fatalf("higher target stats changed from %+v to %+v", before, target.player.Stats)
	}
}
