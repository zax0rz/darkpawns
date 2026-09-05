package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type castleGuardUpCombatEngine struct {
	starts [][2]string
}

func (e *castleGuardUpCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *castleGuardUpCombatEngine) IsFighting(string) bool { return false }

func (e *castleGuardUpCombatEngine) GetCombatTarget(string) (combat.Combatant, bool) {
	return nil, false
}

func TestSpecCastleGuardUp_EntryGatesAndBlockAudience(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*World, *Player, *MobInstance)
		cmd   string
		want  bool
	}{
		{name: "commandless", cmd: "", want: false},
		{name: "nonmatching command", cmd: "look", want: false},
		{
			name: "sleeping guard",
			cmd:  "up",
			setup: func(_ *World, _ *Player, guard *MobInstance) {
				guard.SetPosition(combat.PosSleeping)
			},
			want: false,
		},
		{
			name: "immortal target",
			cmd:  "up",
			setup: func(_ *World, player *Player, _ *MobInstance) {
				player.SetLevel(LVL_IMMORT)
			},
			want: false,
		},
		{
			name: "house owner at room plus one",
			cmd:  "up",
			setup: func(w *World, player *Player, _ *MobInstance) {
				w.HouseControl = []HouseControl{{VNum: player.GetRoomVNum() + 1, Owner: int64(player.GetID())}}
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
			lastMsg() // discard the mob-arrival act from SpawnMob
			if tc.setup != nil {
				tc.setup(w, player, guard)
			}
			if got := specCastleGuardUp(w, player, guard, tc.cmd, ""); got != tc.want {
				t.Fatalf("handled = %v, want %v", got, tc.want)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("gated invocation emitted %q", got)
			}
		})
	}

	t.Run("blocked uses C audience split and no attack", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		lastMsg()
		peer := NewPlayer(2, "Peer", player.GetRoomVNum())
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
		messages := make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

		if got := specCastleGuardUp(w, player, guard, "up", ""); !got {
			t.Fatal("blocked up should be handled")
		}
		if got := messages[player.GetName()]; !strings.Contains(got, "blocks your way.") ||
			!strings.Contains(got, "Thou shalt not pass.") {
			t.Errorf("actor messages = %q", got)
		}
		if got := messages[peer.GetName()]; !strings.Contains(got, "blocks "+player.GetName()+"'s path.") ||
			!strings.Contains(got, "Thou shalt not pass.") {
			t.Errorf("peer messages = %q", got)
		}
		if got := player.GetRoomVNum(); got != guard.GetRoomVNum() {
			t.Fatalf("blocked actor moved to room %d, want %d", got, guard.GetRoomVNum())
		}
	})

	t.Run("grouped owner uses current-room house coordinate", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		lastMsg()
		owner := NewPlayer(2, "Owner", player.GetRoomVNum())
		if err := w.AddPlayer(owner); err != nil {
			t.Fatalf("AddPlayer owner: %v", err)
		}
		player.SetFollowing(owner.GetName())
		// The C procedure intentionally checks room+1 for the actor but the
		// current room for the grouped owner branch.
		w.HouseControl = []HouseControl{{VNum: player.GetRoomVNum(), Owner: int64(owner.GetID())}}
		messages := make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

		if got := specCastleGuardUp(w, player, guard, "up", ""); got {
			t.Fatal("grouped owner should pass through")
		}
		if got := messages[player.GetName()]; !strings.Contains(got, "allows you to pass.") {
			t.Errorf("actor messages = %q", got)
		}
		if got := messages[owner.GetName()]; !strings.Contains(got, "allows "+player.GetName()+" to pass.") {
			t.Errorf("owner messages = %q", got)
		}
	})
}

func TestSpecCastleGuardUp_AutonomousSecondGuardTarget(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	other := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	guard.VNum = 19650
	other.VNum = 19650
	other.SetFighting(player.GetName())
	lastMsg()

	engine := &castleGuardUpCombatEngine{}
	w.SetCombatEngine(engine)
	if got := specCastleGuardUp(w, nil, guard, "", ""); !got {
		t.Fatal("awake idle guard should intercept a fighting peer guard")
	}
	if want := [2]string{guard.GetName(), player.GetName()}; len(engine.starts) != 1 || engine.starts[0] != want {
		t.Fatalf("combat starts = %#v, want %#v", engine.starts, want)
	}
}

func TestSpecCastleGuardUp_AutonomousSecondGuardTargetsMob(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	other := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	target := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	guard.VNum = 19650
	other.VNum = 19650
	guard.Prototype.ShortDesc = "a castle-up guard"
	other.Prototype.ShortDesc = "another castle-up guard"
	target.Prototype.ShortDesc = "a castle-up target mob"
	other.SetFighting(target.GetName())
	lastMsg()

	engine := &castleGuardUpCombatEngine{}
	w.SetCombatEngine(engine)
	if got := specCastleGuardUp(w, nil, guard, "", ""); !got {
		t.Fatal("awake idle guard should intercept a fighting peer guard")
	}
	if want := [2]string{guard.GetName(), target.GetName()}; len(engine.starts) != 1 || engine.starts[0] != want {
		t.Fatalf("combat starts = %#v, want %#v", engine.starts, want)
	}
}

func TestSpecCastleGuardUp_DoesNotTreatUnregisteredMobAsPeer(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	other := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	other.SetFighting(player.GetName())
	lastMsg()

	engine := &castleGuardUpCombatEngine{}
	w.SetCombatEngine(engine)
	if got := specCastleGuardUp(w, nil, guard, "", ""); got {
		t.Fatal("unregistered peer mob should not trigger castle guard handoff")
	}
	if len(engine.starts) != 0 {
		t.Fatalf("combat starts = %#v, want none", engine.starts)
	}
}
