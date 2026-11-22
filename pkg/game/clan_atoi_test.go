package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newClanTestWorld builds a world with one two-rank clan and a god-level
// actor, matching the DP-1523 clan-atoi-depth scenario's shape.
func newClanTestWorld(t *testing.T) (*World, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.Clans = NewClanManager()
	c := &Clan{ID: 1, Name: "Wolves", Ranks: 2}
	c.RankName[0] = "Member"
	c.RankName[1] = "Leader"
	w.Clans.AddClan(c)

	p := NewPlayer(1, "Clangod", 1)
	p.Level = 40
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var output strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == p.Name {
			output.Write(msg)
		}
	}
	return w, p, &output
}

// TestDoClanRenameOverflowRefusesLikeC pins the DP-1523 clan batch: C's
// do_clan_rename parses with atoi after the is_number gate (src/clan.c:134-141),
// so a 20-digit token wraps negative and answers "There is no clan with that
// number." instead of renaming clan 0. Fails if the site reverts to
// strconv.Atoi, whose error leaves 0 and proceeds (R5h).
func TestDoClanRenameOverflowRefusesLikeC(t *testing.T) {
	w, p, output := newClanTestWorld(t)

	w.doClanRename(p, "99999999999999999999 Ravens")

	if got, want := output.String(), "There is no clan with that number.\r\n"; got != want {
		t.Fatalf("rename(20-digit) output = %q, want %q", got, want)
	}
	if got := w.Clans.GetClanByIndex(0).Name; got != "Wolves" {
		t.Fatalf("clan 0 renamed to %q, want untouched (C refuses before any rename)", got)
	}
}

// TestDoClanBankOverflowWithdrawMatchesC pins do_clan_bank's overflow result
// (src/clan.c:940-965): C's atoi returns the wrapped negative amount, which is
// added to the player's gold and subtracted from clan treasure before the
// withdrawal acknowledgement. Atoi's previous error arm withdrew 0, while the
// previous treasury guard also dropped C's negative subtraction.
func TestDoClanBankOverflowWithdrawMatchesC(t *testing.T) {
	w, p, output := newClanTestWorld(t)

	w.doClanBank(p, "99999999999999999999 Wolves", CBWithdraw)

	if got, want := output.String(), "You withdraw from the clan's treasure.\r\n"; got != want {
		t.Fatalf("withdraw(20-digit) output = %q, want %q", got, want)
	}
	if got := p.GetGold(); got != -1 {
		t.Fatalf("withdraw(20-digit) player gold = %d, want -1", got)
	}
	if got := w.Clans.GetClanByIndex(0).TreasureBalance(); got != 1 {
		t.Fatalf("withdraw(20-digit) clan treasure = %d, want 1", got)
	}
}

// TestDoClanSetAppfeeOverflowRefusesLikeC pins the app-fee parse
// (src/clan.c:1029-1039): the wrapped negative amount takes "Please pick a
// number between 0 and 10,000 coins." rather than being stored as 0.
func TestDoClanSetAppfeeOverflowRefusesLikeC(t *testing.T) {
	w, p, output := newClanTestWorld(t)

	w.doClanMoney(p, "99999999999999999999 Wolves", 2)

	if got, want := output.String(), "Please pick a number between 0 and 10,000 coins.\r\n"; got != want {
		t.Fatalf("appfee(20-digit) output = %q, want %q", got, want)
	}
	if got := w.Clans.GetClanByIndex(0).AppFee; got != 0 {
		t.Fatalf("appfee stored %d, want 0 untouched (C refuses)", got)
	}
}
