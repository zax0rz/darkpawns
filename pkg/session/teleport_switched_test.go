package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1516 item 8: do_switch attaches the descriptor to the switched body and
// clears the original's desc (src/act.wizard.c:414-421). do_teleport against
// such a victim (src/act.wizard.c:366-388) therefore emits neither its TO_VICT
// notice nor look_at_room(victim) (src/act.informative.c:1103-1104).
func TestTeleportSwitchedVictimSkipsRecipientOutput(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room A"}, {VNum: 1002, Name: "Room B"}},
		Mobs:  []parser.Mob{{VNum: 9009, Keywords: "horse", ShortDesc: "a horse", Level: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)

	obs := makeTestSession(t, m, "Helpermage", 1001, true)
	obs.player.SetLevel(game.LVL_IMPL)
	registerTestSession(t, m, obs, "Helpermage")

	victim := makeTestSession(t, m, "Ridergod", 1001, true)
	victim.player.SetLevel(10)
	registerTestSession(t, m, victim, "Ridergod")
	mob, err := w.SpawnMob(9009, 1001)
	if err != nil {
		t.Fatal(err)
	}
	victim.isSwitched = true
	victim.switchedMob = mob
	victim.switchedOriginal = victim.player

	if err := cmdTeleport(obs, []string{"Ridergod", "1002"}); err != nil {
		t.Fatalf("cmdTeleport: %v", err)
	}
	if got := victim.player.GetRoomVNum(); got != 1002 {
		t.Fatalf("teleport did not move the victim body: room %d, want 1002", got)
	}

	if got := drainFrames(victim); strings.Contains(got, "has teleported you!") || strings.Contains(got, "Room B") {
		t.Fatalf("switched victim received recipient/destination output: %q", got)
	}
}
