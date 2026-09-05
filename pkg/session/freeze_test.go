package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFreezeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("freeze")
	if !ok {
		t.Fatal("freeze command is not registered")
	}
	if entry.MinLevel != LVL_GRGOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("freeze gate = (%d, %d), want (%d, %d)",
			entry.MinLevel, entry.MinPosition, LVL_GRGOD, combat.PosDead)
	}
}

func TestCmdFreezeResolvesCCharacterTargets(t *testing.T) {
	t.Run("self alias", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Actor", LVL_GRGOD, 1001)
		registerInWorld(t, actor)

		if err := cmdFreeze(actor, []string{"self"}); err != nil {
			t.Fatal(err)
		}
		if got, want := readMsgText(t, actor), "Oh, yeah, THAT'S real smart...\r\n"; got != want {
			t.Fatalf("self response = %q, want %q", got, want)
		}
	})

	t.Run("leading fill word and trailing arguments", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Actor", LVL_GRGOD, 1001)
		victim := makeCommandTestSession(t, m, "Victim", 1, 1001)
		registerInWorld(t, actor)
		registerInWorld(t, victim)

		if err := cmdFreeze(actor, []string{"the", "Victim", "ignored"}); err != nil {
			t.Fatal(err)
		}
		if got, want := readMsgText(t, actor), "Frozen.\r\n"; got != want {
			t.Fatalf("actor response = %q, want %q", got, want)
		}
		if got, want := readMsgText(t, actor), "A sudden cold wind conjured from nowhere freezes Victim!\r\n"; got != want {
			t.Fatalf("room response = %q, want %q", got, want)
		}
		if got, want := readMsgText(t, victim), "A bitter wind suddenly rises and drains every ounce of heat from your body!\r\nYou feel frozen!\r\n"; got != want {
			t.Fatalf("victim response = %q, want %q", got, want)
		}
		if victim.player.GetFlags()&(1<<uint(game.PlrFrozen)) == 0 {
			t.Fatal("freeze did not set PLR_FROZEN")
		}
		if victim.player.FreezeLevel != actor.player.Level {
			t.Fatalf("freeze level = %d, want %d", victim.player.FreezeLevel, actor.player.Level)
		}
	})

	t.Run("visible mob", func(t *testing.T) {
		m := makeTestManagerWithMobs(t)
		actor := makeCommandTestSession(t, m, "Actor", LVL_GRGOD, 1001)
		registerInWorld(t, actor)
		registerMob(t, m, 2001, 1001)
		_ = readMsgText(t, actor) // mob-entry announcement

		if err := cmdFreeze(actor, []string{"guard"}); err != nil {
			t.Fatal(err)
		}
		if got, want := readMsgText(t, actor), "You can't do that to a mob!\r\n"; got != want {
			t.Fatalf("mob response = %q, want %q", got, want)
		}
	})
}
