package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCmdReciteHoldPriority pins the SCMD_RECITE half of C do_use's
// WEAR_HOLD-first resolution (act.other.c:897-916): `recite scroll` with a
// held keyword-matching scroll and a carried one dissolves the HELD copy —
// the recite line names the held scroll, the hold slot empties, the carried
// copy survives — and mag_objectmagic's WAIT_STATE(PULSE_VIOLENCE) lands
// (spell_parser.c:683). Sentinel -1 spell slots keep the cast loop inert.
func TestCmdReciteHoldPriority(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Recitehold", 1001, true)
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("add player to world: %v", err)
	}
	m.world.MessageSink = func(name string, msg []byte) {
		if name == s.player.Name && len(s.send) < cap(s.send) {
			s.send <- msg
		}
	}

	held := game.NewObjectInstance(&parser.Obj{
		VNum:      9101,
		Keywords:  "scroll held",
		ShortDesc: "a held scroll",
		TypeFlag:  2, // ITEM_SCROLL
		WearFlags: [4]int{16385, 0, 0, 0},
		Values:    [4]int{10, -1, -1, -1},
	}, 1001)
	carried := game.NewObjectInstance(&parser.Obj{
		VNum:      9102,
		Keywords:  "scroll carried",
		ShortDesc: "a carried scroll",
		TypeFlag:  2,
		Values:    [4]int{10, -1, -1, -1},
	}, 1001)

	if err := s.player.Inventory.AddItem(held); err != nil {
		t.Fatalf("add held scroll: %v", err)
	}
	if err := s.player.Inventory.AddItem(carried); err != nil {
		t.Fatalf("add carried scroll: %v", err)
	}
	if err := s.player.Equipment.Equip(held, s.player.Inventory); err != nil {
		t.Fatalf("hold scroll: %v", err)
	}

	if err := cmdRecite(s, []string{"scroll"}); err != nil {
		t.Fatalf("cmdRecite: %v", err)
	}

	out := drainSendChannel(t, s)
	if !strings.Contains(out, "You recite a held scroll which dissolves.") {
		t.Fatalf("recite line did not name the HELD scroll; output:\n%s", out)
	}
	if strings.Contains(out, "a carried scroll") {
		t.Fatalf("carried scroll was consumed or named; output:\n%s", out)
	}

	if _, stillHeld := s.player.Equipment.GetItemInSlot(game.SlotHold); stillHeld {
		t.Fatal("held scroll survived the recite; C extract_obj removes it from WEAR_HOLD")
	}
	if item := s.player.Inventory.FindItems("carried"); len(item) == 0 {
		t.Fatal("carried scroll vanished; only the held copy dissolves")
	}
	if got, want := s.player.GetWaitState(), engine.PULSE_VIOLENCE; got != want {
		t.Fatalf("wait state = %d pulses, want %d (WAIT_STATE(PULSE_VIOLENCE))", got, want)
	}
}
