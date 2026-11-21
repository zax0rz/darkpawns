package game

import (
	"path/filepath"
	"strings"
	"testing"
)

// string_add's improved-editor overflow path (modify.c:131-155) truncates an
// overlong first line or skips an overlong later line while keeping the clan
// plan editor open.
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
	if !player.ClanPlanWriting {
		t.Fatal("editor closed after C's first-line truncation")
	}
	if got := len(player.ClanPlanBuffer); got != MaxClanPlanBytes-1 {
		t.Fatalf("truncated plan buffer = %d bytes, want %d (cap-3 cut plus CRLF)", got, MaxClanPlanBytes-1)
	}
	world.HandleClanPlanInput(player, "@")
	if player.ClanPlanWriting || len(clan.Plan) != MaxClanPlanBytes-1 {
		t.Fatalf("@ did not save the truncated plan: writing=%t plan-bytes=%d", player.ClanPlanWriting, len(clan.Plan))
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
	if !player.ClanPlanWriting {
		t.Fatal("editor closed after C skipped the later line")
	}
	if strings.Contains(player.ClanPlanBuffer, "overflowing later line") {
		t.Fatal("skipped line was stored")
	}
	world.HandleClanPlanInput(player, "@")
	if player.ClanPlanWriting {
		t.Fatal("@ did not close the still-active editor")
	}
	if strings.Contains(clan.Plan, "overflowing later line") {
		t.Fatal("skipped line was stored in the saved plan")
	}

	sent = nil
	world.BeginClanPlanWrite(player, clan)
	boundary := strings.Repeat("z", MaxClanPlanBytes-3)
	if !world.HandleClanPlanInput(player, boundary) {
		t.Fatal("boundary line refused")
	}
	if !player.ClanPlanWriting {
		t.Fatal("editor exited on a boundary line")
	}
	if got, want := player.ClanPlanBuffer, boundary+"\r\n"; got != want {
		t.Fatalf("boundary buffer = %d bytes, want exact accepted line at %d bytes", len(got), len(want))
	}
	if got := strings.Join(sent, ""); strings.Contains(got, "String too long") {
		t.Fatalf("C-fit boundary line received overflow notice: %q", got)
	}
}
