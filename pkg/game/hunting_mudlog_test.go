package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
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

func TestLuaSetHuntPlayerHunterUsesCPlayerPreyLog(t *testing.T) {
	w, mobile, _, output := huntingMudlogWorld(t)
	hunter := NewPlayer(3, "Playerhunter", 1001)
	if err := w.AddPlayer(hunter); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hunt.lua"), []byte(
		"function oncmd()\nset_hunt(ch, ch)\nreturn TRUE\nend\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewWorldScriptableAdapter(w)
	e := scripting.NewEngine(dir, a)
	t.Cleanup(e.Close)
	chRef := scripting.CharRef{ID: hunter.GetID()}
	meRef := scripting.CharRef{NPC: true, ID: mobile.GetID()}
	ctx := &scripting.ScriptContext{World: a, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001}
	if handled, err := e.RunScript(ctx, "hunt.lua", "oncmd"); err != nil || !handled {
		t.Fatalf("script contract: handled=%t err=%v", handled, err)
	}
	want := "[ Playerhunter started hunting Playerhunter ]\r\n"
	if got := output.String(); got != want {
		t.Fatalf("player hunter log = %q, want %q", got, want)
	}

	output.Reset()
	if err := os.WriteFile(filepath.Join(dir, "hunt.lua"), []byte(
		"function oncmd()\nset_hunt(ch, me)\nreturn TRUE\nend\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if handled, err := e.RunScript(ctx, "hunt.lua", "oncmd"); err != nil || !handled {
		t.Fatalf("mobile-prey script contract: handled=%t err=%v", handled, err)
	}
	if got := output.String(); got != "" {
		t.Fatalf("mobile prey arm logged %q, want no C log", got)
	}
}
