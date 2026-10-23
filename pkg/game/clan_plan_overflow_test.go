package game

import (
	"path/filepath"
	"strings"
	"testing"
)

// L2-005 as fidelity: C's string editor overflow (string_add, modify.c:131-150,
// the non-improved path do_clan_plan uses) — a first line over CLAN_PLAN_LENGTH
// truncates with C's reply and saves+exits; a later line that would cross it is
// skipped with C's reply and saves+exits; a boundary line is accepted and the
// editor continues.
func TestClanPlanEditorOverflowMatchesC(t *testing.T) {
	var sent []string
	player := NewPlayer(1, "Editor", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	world := &World{Clans: InitClans(filepath.Join(t.TempDir(), "clans.json"))}
	clan := &Clan{ID: 1, Name: "Testclan"}
	world.Clans.AddClan(clan)

	world.BeginClanPlanWrite(player, clan)
	if !world.HandleClanPlanInput(player, strings.Repeat("x", MaxClanPlanBytes)) {
		t.Fatal("editor did not consume the overflowing first line")
	}
	if !strings.Contains(strings.Join(sent, ""), "String too long - Truncated.\r\n") {
		t.Fatalf("missing C truncation reply: %q", sent)
	}
	if player.ClanPlanWriting {
		t.Fatal("editor still active after C's truncation exit")
	}
	if len(clan.Plan) != MaxClanPlanBytes-1 {
		t.Fatalf("truncated plan = %d bytes, want %d (cap-3 cut plus CRLF)", len(clan.Plan), MaxClanPlanBytes-1)
	}

	sent = nil
	world.BeginClanPlanWrite(player, clan)
	if !world.HandleClanPlanInput(player, strings.Repeat("y", MaxClanPlanBytes-8)) {
		t.Fatal("editor refused the fitting first line")
	}
	if !world.HandleClanPlanInput(player, "overflowing later line") {
		t.Fatal("editor did not consume the overflowing later line")
	}
	if !strings.Contains(strings.Join(sent, ""), "String too long.  Last line skipped.\r\n") {
		t.Fatalf("missing C skip reply: %q", sent)
	}
	if player.ClanPlanWriting {
		t.Fatal("editor still active after C's skip exit")
	}
	if strings.Contains(clan.Plan, "overflowing later line") {
		t.Fatal("skipped line was stored")
	}

	sent = nil
	world.BeginClanPlanWrite(player, clan)
	if !world.HandleClanPlanInput(player, strings.Repeat("z", MaxClanPlanBytes-4)) {
		t.Fatal("boundary line refused")
	}
	if !player.ClanPlanWriting {
		t.Fatal("editor exited on a boundary line")
	}
}
