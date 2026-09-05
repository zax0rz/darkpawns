package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSpecChosenGuard_EntryGatesAndAudience(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Player, *MobInstance)
	}{
		{
			name:  "commandless",
			setup: func(_ *Player, _ *MobInstance) {},
		},
		{
			name:  "non-movement command",
			setup: func(_ *Player, _ *MobInstance) {},
		},
		{
			name:  "sleeping guard",
			setup: func(_ *Player, mob *MobInstance) { mob.SetPosition(combat.PosSleeping) },
		},
		{
			name:  "immortal target",
			setup: func(player *Player, _ *MobInstance) { player.SetLevel(LVL_IMMORT) },
		},
		{
			name:  "chosen target",
			setup: func(player *Player, _ *MobInstance) { player.SetPLRFlag(PlrChosen) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
			lastMsg() // discard the mob-arrival act from SpawnMob
			tc.setup(player, mob)
			cmd := "south"
			switch tc.name {
			case "commandless":
				cmd = ""
			case "non-movement command":
				cmd = "look"
			}
			if got := specChosenGuard(w, player, mob, cmd, ""); got {
				t.Fatal("chosen_guard handled a gated invocation")
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("gated invocation emitted output: %q", got)
			}
		})
	}

	t.Run("south blocks with C audience split", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		lastMsg() // discard the mob-arrival act from SpawnMob
		peer := NewPlayer(2, "Peer", player.GetRoomVNum())
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
		messages := make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

		if got := specChosenGuard(w, player, mob, "south", ""); !got {
			t.Fatal("chosen_guard should handle south")
		}
		if !strings.Contains(messages[player.GetName()], "blocks your way.") ||
			!strings.Contains(messages[player.GetName()], "Thou shalt not pass.") {
			t.Errorf("actor messages = %q", messages[player.GetName()])
		}
		if !strings.Contains(messages[peer.GetName()], "blocks "+player.GetName()+"'s way.") ||
			!strings.Contains(messages[peer.GetName()], "Thou shalt not pass.") {
			t.Errorf("peer messages = %q", messages[peer.GetName()])
		}
		if got := player.GetRoomVNum(); got != mob.GetRoomVNum() {
			t.Fatalf("blocked actor moved to room %d, want %d", got, mob.GetRoomVNum())
		}
	})
}
