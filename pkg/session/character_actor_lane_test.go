package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestRegisteredNPCActorLanesRunCBody covers the ordered-NPC verbs added beside
// report in DP-1453. Each lane is the same do_* body the player lane uses, and
// each assertion can fail by losing the actor registration (result becomes
// Unsupported and no room line appears).
func TestRegisteredNPCActorLanesRunCBody(t *testing.T) {
	t.Run("rest", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosStanding)
		var private strings.Builder
		if got := callRegisteredActorCommand(t, f, "rest", &private); got != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", got)
		}
		if got, want := f.actor.GetPosition(), combat.PosResting; got != want {
			t.Errorf("position = %d, want %d", got, want)
		}
		if got, want := f.roomOut.String(), "A guard trainee sits down and rests.\r\n"; got != want {
			t.Errorf("room bytes = %q, want %q", got, want)
		}
	})

	t.Run("sleep", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosResting)
		var private strings.Builder
		if got := callRegisteredActorCommand(t, f, "sleep", &private); got != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", got)
		}
		if got, want := f.actor.GetPosition(), combat.PosSleeping; got != want {
			t.Errorf("position = %d, want %d", got, want)
		}
		if got, want := f.roomOut.String(), "A guard trainee lies down and falls asleep.\r\n"; got != want {
			t.Errorf("room bytes = %q, want %q", got, want)
		}
	})

	t.Run("wake", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosSleeping)
		var private strings.Builder
		if got := callRegisteredActorCommand(t, f, "wake", &private); got != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", got)
		}
		if got, want := f.actor.GetPosition(), combat.PosSitting; got != want {
			t.Errorf("position = %d, want %d", got, want)
		}
		if got, want := f.roomOut.String(), "A guard trainee awakens.\r\n"; got != want {
			t.Errorf("room bytes = %q, want %q", got, want)
		}
	})

	t.Run("follow", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosStanding)
		entry, ok := cmdRegistry.Lookup("follow")
		if !ok || entry.ActorHandler == nil {
			t.Fatal("follow has no actor lane")
		}
		// C's raw remainder begins at the delimiter (one_argument lowercases
		// and drops fill words; extractArg mirrors it).
		result := entry.ActorHandler(game.CharacterCommandContext{
			Actor: f.actor, World: f.world, RawArgs: " the Observer",
		}, "follow", nil)
		if result != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", result)
		}
		if got, want := f.actor.GetFollowing(), "Observer"; got != want {
			t.Errorf("following = %q, want %q", got, want)
		}
		// act(TO_NOTVICT) excludes the leader, so the leader sees only the
		// TO_VICT line; another occupant would see "$n starts to follow $N."
		want := "A guard trainee starts following you.\r\n"
		if got := f.roomOut.String(); got != want {
			t.Errorf("room bytes = %q, want %q", got, want)
		}
	})

	t.Run("dismount refuses when not mounted", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosStanding)
		var private strings.Builder
		if got := callRegisteredActorCommand(t, f, "dismount", &private); got != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", got)
		}
		if got := private.String(); got != "" {
			t.Errorf("mobile private bytes = %q, want none", got)
		}
		if got := f.roomOut.String(); got != "" {
			t.Errorf("room bytes = %q, want none", got)
		}
	})

	t.Run("mount is a C no-op for a mobile", func(t *testing.T) {
		f := newActorPositionFixture(t, combat.PosStanding)
		var private strings.Builder
		if got := callRegisteredActorCommand(t, f, "mount", &private); got != game.CharacterCommandHandled {
			t.Fatalf("result = %v, want handled", got)
		}
		if f.actor.IsAffected(game.AffMount) {
			t.Error("a mobile rider acquired AFF_MOUNT; C CAN_MOUNT is false for IS_MOB")
		}
		if got := f.roomOut.String(); got != "" {
			t.Errorf("room bytes = %q, want none", got)
		}
	})
}
