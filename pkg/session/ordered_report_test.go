package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestOrderedMobileReportUsesReportersStats pins the reported DP-1453 failure:
// `order <horse> rep` reached command_interpreter's abbreviation resolution but
// then hit an unported actor lane and was dropped with only a server log. C
// do_order hands the raw message to command_interpreter (src/act.offensive.c:
// 332), so `rep` resolves to `report` (src/interpreter.c:634) and do_report
// prints the *reporter's* H/M/V to every character in the room through that
// recipient's color wrappers (src/act.other.c:799-821).
//
// The horse's numbers differ from the orderer's, so a port that prints the
// orderer's stats (or drops the verb) cannot pass. One recipient has color off
// and one has C_CMP on, so a port that drops or hard-codes the CC* wrappers
// cannot pass either (R1/R2/R5e/R5h).
func TestOrderedMobileReportUsesReportersStats(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Stable", Exits: map[string]parser.Exit{}}},
		Mobs: []parser.Mob{{
			VNum: 99100, Keywords: "horse", ShortDesc: "a horse",
			LongDesc: "A horse is here, grazing.\n",
			Level:    5, Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	// Pin the interpreter's leading number(0,3) draw so this test cannot shift
	// the process-wide RNG stream for later package tests.
	previousDraw := characterCommandNumber
	characterCommandNumber = func(int, int) int { return 1 }
	t.Cleanup(func() { characterCommandNumber = previousDraw })
	m := newTestManager(t, w, nil)
	m.SetCommandExecFunc()

	plain := game.NewPlayer(2, "Plain", 1001)
	plain.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(plain); err != nil {
		t.Fatal(err)
	}
	colored := game.NewPlayer(3, "Colored", 1001)
	colored.SetPosition(combat.PosStanding)
	colored.SetPlrFlag(game.PrfColor1, true)
	colored.SetPlrFlag(game.PrfColor2, true)
	if err := w.AddPlayer(colored); err != nil {
		t.Fatal(err)
	}
	// C's act(...TO_CHAR) is gated by SENDOK(tch, false): asleep occupants
	// receive neither the room header nor the reporter line.
	sleeper := game.NewPlayer(4, "Sleeper", 1001)
	sleeper.SetPosition(combat.PosSleeping)
	if err := w.AddPlayer(sleeper); err != nil {
		t.Fatal(err)
	}
	output := map[string]*strings.Builder{plain.GetName(): {}, colored.GetName(): {}, sleeper.GetName(): {}}
	w.MessageSink = func(name string, message []byte) {
		if buf, ok := output[name]; ok {
			buf.Write(message)
		}
	}

	orderer := makeCommandTestSession(t, m, "Egon", 12, 1001)
	orderer.player.SetPosition(combat.PosStanding)
	orderer.player.SetHealth(100)
	orderer.player.SetMaxHP(100)
	orderer.player.SetMana(100)
	orderer.player.SetMaxMana(100)
	orderer.player.SetMove(100)
	orderer.player.MaxMove = 100
	if err := w.AddPlayer(orderer.player); err != nil {
		t.Fatal(err)
	}
	output[orderer.player.GetName()] = &strings.Builder{}

	horse, err := w.SpawnMob(99100, 1001)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	horse.SetHealth(7)
	horse.SetMaxHP(40)
	horse.SetMana(6)
	horse.SetMaxMana(10)
	horse.SetMovePool(230, 230)
	horse.SetAffected(game.AffCharm)
	horse.SetFollowing(orderer.player.GetName())

	// "rep" is the exact abbreviation the production log carried.
	if err := cmdOrder(orderer, []string{"horse", "rep"}); err != nil {
		t.Fatalf("order returned error: %v", err)
	}

	const orderAnnouncement = "Egon gives a horse an order.\r\n"
	const reportHeader = "A horse reports:\r\n"
	const plainLine = "    [7/40]H [6/10]M [230/230]V [0]Kills [0]PKs [0]Deaths\r\n"
	if got := output[plain.GetName()].String(); got != orderAnnouncement+reportHeader+plainLine {
		t.Fatalf("uncoloured recipient bytes =\n%q\nwant\n%q", got, orderAnnouncement+reportHeader+plainLine)
	}
	const coloredLine = "    [\x1b[32m7\x1b[0m/\x1b[32m40\x1b[0m]H" +
		" [\x1b[32m6\x1b[0m/\x1b[32m10\x1b[0m]M" +
		" [\x1b[32m230\x1b[0m/\x1b[32m230\x1b[0m]V" +
		" [\x1b[31m0\x1b[0m]Kills [\x1b[31m0\x1b[0m]PKs [\x1b[31m0\x1b[0m]Deaths\r\n"
	if got := output[colored.GetName()].String(); got != orderAnnouncement+reportHeader+coloredLine {
		t.Fatalf("coloured recipient bytes =\n%q\nwant\n%q", got, orderAnnouncement+reportHeader+coloredLine)
	}
	// The orderer is in the room: do_report prints its line to the orderer's
	// descriptor, then do_order sends OK after command_interpreter returns.
	if got := output[orderer.player.GetName()].String(); got != reportHeader+plainLine {
		t.Fatalf("orderer report line = %q, want the reporter line", got)
	}
	if got := readMsgText(t, orderer); got != "Okay.\r\n" {
		t.Fatalf("orderer acknowledgement = %q, want Okay.", got)
	}
	if got := output[sleeper.GetName()].String(); got != "" {
		t.Fatalf("sleeping occupant received %q, want nothing (SENDOK)", got)
	}
}
