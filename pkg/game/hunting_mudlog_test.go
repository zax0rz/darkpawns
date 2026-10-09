package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// huntingMudlogWorld is a room, a mob prototype and a complete-syslog immortal
// at LVL_IMMORT whose delivered bytes the test can read.
func huntingMudlogWorld(t *testing.T) (*World, *MobInstance, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Lab", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 2001, Keywords: "goblin", ShortDesc: "A goblin guard", Level: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	output := &strings.Builder{}
	w.MessageSink = func(name string, message []byte) {
		if name == "Hunterwatch" {
			output.Write(message)
		}
	}
	watcher := NewPlayer(1, "Hunterwatch", 1001)
	watcher.Level = LVL_IMMORT
	watcher.SetPlrFlag(PrfLog1, true)
	watcher.SetPlrFlag(PrfLog2, true)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatal(err)
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(&testSessions{watcher})
	t.Cleanup(func() { SetImmortalSessionProvider(old) })

	hunter, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	prey := NewPlayer(2, "Prey", 1001)
	if err := w.AddPlayer(prey); err != nil {
		t.Fatal(err)
	}
	return w, hunter, prey, output
}

// src/utils.c:715-724: the producer names the hunter and the prey at
// CMP / LVL_IMMORT / file FALSE, and only for a prey that is not a mobile and
// has an id number (GET_IDNUM(vict) > 0).
func TestLogHuntingStartGates(t *testing.T) {
	t.Run("player prey with an id", func(t *testing.T) {
		_, hunter, prey, output := huntingMudlogWorld(t)
		LogHuntingStart(hunter, prey)
		want := "[ " + hunter.GetName() + " started hunting Prey ]\r\n"
		if got := output.String(); got != want {
			t.Fatalf("observer = %q, want %q", got, want)
		}
	})

	t.Run("id-less prey is silent", func(t *testing.T) {
		// C's GET_IDNUM(vict) > 0 arm; a guest body has no id in the port.
		_, hunter, prey, output := huntingMudlogWorld(t)
		prey.ID = 0
		LogHuntingStart(hunter, prey)
		if got := output.String(); got != "" {
			t.Fatalf("an id-less prey logged %q", got)
		}
	})

	t.Run("missing actors are silent", func(t *testing.T) {
		_, hunter, prey, output := huntingMudlogWorld(t)
		LogHuntingStart(nil, prey)
		LogHuntingStart(hunter, nil)
		if got := output.String(); got != "" {
			t.Fatalf("a nil actor logged %q", got)
		}
	})

	// damage()'s hunter arm (src/fight.c:1453-1455) reaches the producer only
	// for a player victim; a mobile victim is C's IS_MOB arm, which stores the
	// target silently.
	t.Run("combat arm: player logs, mobile is silent", func(t *testing.T) {
		w, hunter, prey, output := huntingMudlogWorld(t)
		hunter.SetMobFlag(MobFlagHunter)
		other, err := w.SpawnMobQuiet(2001, 1001)
		if err != nil {
			t.Fatal(err)
		}
		cb := w.WireCombatCallbacks()

		cb.RangedHunt(hunter, other)
		if got := output.String(); got != "" {
			t.Fatalf("a mobile victim logged %q", got)
		}

		cb.RangedHunt(hunter, prey)
		want := "[ " + hunter.GetName() + " started hunting Prey ]\r\n"
		if got := output.String(); got != want {
			t.Fatalf("observer = %q, want %q", got, want)
		}
	})
}
