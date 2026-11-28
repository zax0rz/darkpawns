package session

// switched_mobile_look_test.go — do_switch moves the wizard's descriptor to
// the mobile body (src/act.wizard.c:1198-1199), so what that body looks at is
// the wizard's own output: C's send_to_char bytes, then the prompt. The port
// framed those writes for a mobile body, which made do_description's trailing
// space (src/act.informative.c:2784) a line of its own and capitalized the
// room's name (src/act.informative.c:808). World.MobileLiteralSink gives the
// body the player's literal path; routing bare and verbatim writes back to
// MobileMessageSink fails this.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestLiteralSeamSwitchedMobileLook(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "stable room, do not delete", Description: "A stable.\n",
				ExtraDescs: []parser.ExtraDesc{{Keywords: "straw", Description: "Straw."}},
			},
			{VNum: 1002, Name: "Yard"},
		},
		Mobs: []parser.Mob{{VNum: 9001, Keywords: "pony", ShortDesc: "a pony", LongDesc: "a pony stands here.\n"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	wizard := makeTestSession(t, m, "Wizard", 1002, true)
	wizard.transportDone = make(chan struct{})
	registerTestSession(t, m, wizard, "Wizard")
	pony, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	if err := cmdSwitch(wizard, []string{"pony"}); err != nil {
		t.Fatalf("cmdSwitch: %v", err)
	}
	if got := renderAllOutput(wizard); got != "Okay.\r\n" {
		t.Fatalf("switch reply = %q", got)
	}

	// look_at_room writes the name, the description, and the space that
	// starts whatever comes next (src/act.informative.c:765-811, :2784). A
	// mobile has no PRF_COMPACT (src/comm.c:1631 reads the descriptor's
	// character, which do_switch made the mobile), so the prompt's "\r\n"
	// frame follows it.
	w.RenderObservationMessages(w.DoLookRoom(pony, true))
	assertBareWire(t, wizard, "stable room, do not delete\r\nA stable.\r\n ")
}
