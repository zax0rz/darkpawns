package session

// crash_load_lethal_test.go — DP-1419 follow-up (Zach's ruling, 2026-10-13).
// A zero-DEX trip that kills during Crash_load runs C's death pipeline while
// in_room is still NOWHERE. raw_kill returns at its first statement
// (src/fight.c:538-539), so death_cry, make_corpse and extract_char never run:
// the character stays POS_DEAD where it stands and then enters the game dead.
//
// Each test drives the real menu entry (MsgCharInput "1" -> handleMenuChoice ->
// enterReturningPlayer) on a real SQLite store.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// The full entry: the first trip is fatal, the later three are refused, and the
// character is admitted to the world dead — no corpse, no extraction.
func TestCrashLoadLethalEntryLeavesCharacterDead(t *testing.T) {
	m, _, s := crashBadStatsEnv(t, "Doomed", 10, game.ApplyDex, crashBadStatsWorn)
	objects := len(m.world.GetAllObjects())
	observer := mudlogObserver(t, m, "Deathwatch", game.LVL_GOD, game.PrfLog1, game.PrfLog2)

	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")

	// C's order: check_for_bad_stats sends the trip line, then damage() drops
	// HP by 40, update_pos makes POS_DEAD and the POS_DEAD arm sends its pair
	// (src/fight.c:1560-1588). The three later items still send their trip
	// line, but damage() now refuses at the corpse gate (:1319-1325), so there
	// is no second death message and no further HP loss. Everything precedes
	// the welcome message (src/interpreter.c:2184-2189).
	want := "You trip over your own feet and hit your head!\n\r" +
		"You are dead!  Sorry...\r\n" +
		strings.Repeat("You trip over your own feet and hit your head!\n\r", 3)
	if !strings.HasPrefix(got, want) {
		t.Fatalf("entry stream = %q, want prefix %q", got, want)
	}
	if rest := got[len(want):]; !strings.HasPrefix(rest, "\r\nWelcome to Dark Pawns!") {
		t.Fatalf("the welcome text did not follow the death bytes: %q", got)
	}

	// The character is in the world, dead, and C's HP floor applies (DP-1021
	// clamps at -11; C itself would keep 10-40 = -30 — recorded in the handoff).
	if p, ok := m.world.GetPlayer("Doomed"); !ok || p != s.player {
		t.Fatal("the character did not enter the world")
	}
	if s.player.GetHP() != -11 || s.player.GetPosition() != combat.PosDead {
		t.Fatalf("hp=%d pos=%d, want -11/POS_DEAD", s.player.GetHP(), s.player.GetPosition())
	}
	if s.player.Deaths != 1 || s.player.Kills != 1 {
		t.Fatalf("deaths=%d kills=%d, want 1/1", s.player.Deaths, s.player.Kills)
	}
	// raw_kill returned at NOWHERE: no corpse and no extraction.
	if now := len(m.world.GetAllObjects()); now != objects {
		t.Fatalf("objects=%d, want %d (raw_kill makes no corpse at NOWHERE)", now, objects)
	}
	if m.world.HasPendingExtractions() {
		t.Fatal("the character was queued for extraction; raw_kill returns at NOWHERE")
	}

	// The death mudlog is BRF / MAX(LVL_IMMORT, invis) / file TRUE. Its room
	// segment is the R1a decision: C reads world[victim->in_room].name with
	// in_room == NOWHERE, and the port prints the empty segment its room lookup
	// yields. The guarded act() SYSERR precedes it (src/comm.c:2519-2526).
	wantObserver := "[ Doomed retrieving crash-saved items and entering game. ]\r\n" +
		"[ SYSERR: no valid target to act()! ]\r\n" +
		"[ Doomed killed by Doomed at  ]\r\n"
	if line := strings.Join(drainSessionText(t, observer), ""); line != wantObserver {
		t.Fatalf("observer stream = %q, want %q", line, wantObserver)
	}
}

// Step 5: the character is dead in a room, so the next point_update does
// nothing for them (src/limits.c:493-513 handles POS_STUNNED and above, then
// POS_INCAP and POS_MORTALLYW; POS_DEAD takes no branch) and the command
// interpreter refuses every command (src/interpreter.c:925-927).
func TestCrashLoadLethalEntryFirstTickAndCommand(t *testing.T) {
	m, _, s := crashBadStatsEnv(t, "Doomed", 10, game.ApplyDex, crashBadStatsWorn)
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	drainSessionText(t, s)

	m.world.PointUpdate()
	if s.player.GetHP() != -11 || s.player.GetPosition() != combat.PosDead {
		t.Fatalf("after the tick hp=%d pos=%d, want -11/POS_DEAD", s.player.GetHP(), s.player.GetPosition())
	}
	// POS_DEAD takes no point_update branch, so the only bytes are the hunger
	// and thirst decay every character gets (src/limits.c:471-479) — the
	// fixture's record carries both at 0. Nothing death-related, and no HP or
	// position change.
	if got := strings.Join(drainSessionText(t, s), ""); got != "You are hungry.\r\nYou are thirsty.\r\n" {
		t.Fatalf("the tick sent %q, want only the condition decay", got)
	}

	if err := ExecuteCommand(s, "look", nil); err != nil {
		t.Fatalf("look: %v", err)
	}
	if got := strings.Join(drainSessionText(t, s), ""); got != "Lie still; you are DEAD!!! :-(\r\n" {
		t.Fatalf("command bytes = %q, want C's POS_DEAD refusal", got)
	}
}
