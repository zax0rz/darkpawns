package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCmdQuaffHoldPriority pins C do_use's WEAR_HOLD-first resolution
// (act.other.c:897-910): `quaff potion` with a held keyword-matching potion
// and a carried one consumes the HELD item — the quaff line names the held
// potion's short desc, the hold slot empties, the carried copy survives, and
// mag_objectmagic's WAIT_STATE(PULSE_VIOLENCE) lands (spell_parser.c:710).
func TestCmdQuaffHoldPriority(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Quaffhold", 1001, true)

	held := game.NewObjectInstance(&parser.Obj{
		VNum:      9001,
		Keywords:  "potion yellow seeinvis",
		ShortDesc: "a yellow potion of see invisible",
		TypeFlag:  10, // ITEM_POTION
		WearFlags: [4]int{16385, 0, 0, 0},
		Values:    [4]int{12, -1, -1, -1},
	}, 1001)
	carried := game.NewObjectInstance(&parser.Obj{
		VNum:      9002,
		Keywords:  "sanctuary potion",
		ShortDesc: "Testing Re-Sanc",
		TypeFlag:  10,
		Values:    [4]int{30, -1, -1, -1},
	}, 1001)

	if err := s.player.Inventory.AddItem(held); err != nil {
		t.Fatalf("add held potion: %v", err)
	}
	if err := s.player.Inventory.AddItem(carried); err != nil {
		t.Fatalf("add carried potion: %v", err)
	}
	if err := s.player.Equipment.Equip(held, s.player.Inventory); err != nil {
		t.Fatalf("hold potion: %v", err)
	}

	if err := cmdQuaff(s, []string{"potion"}); err != nil {
		t.Fatalf("cmdQuaff: %v", err)
	}

	out := drainSendChannel(t, s)
	if !strings.Contains(out, "You quaff a yellow potion of see invisible.") {
		t.Fatalf("quaff line did not name the HELD potion; output:\n%s", out)
	}
	if strings.Contains(out, "Testing Re-Sanc") {
		t.Fatalf("carried potion was consumed or named; output:\n%s", out)
	}

	if _, stillHeld := s.player.Equipment.GetItemInSlot(game.SlotHold); stillHeld {
		t.Fatal("held potion survived the quaff; C extract_obj removes it from WEAR_HOLD")
	}
	if item := s.player.Inventory.FindItems("sanctuary"); len(item) == 0 {
		t.Fatal("carried potion vanished; only the held copy is consumed")
	}
	if got, want := s.player.GetWaitState(), engine.PULSE_VIOLENCE; got != want {
		t.Fatalf("wait state = %d pulses, want %d (WAIT_STATE(PULSE_VIOLENCE))", got, want)
	}
}
