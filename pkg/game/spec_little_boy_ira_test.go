package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// seedForNumber010 finds a seed whose first number(0,10) is want.
func seedForNumber010(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(0, 10) == want {
			return seed
		}
	}
	t.Fatalf("no seed draws %d first", want)
	return 0
}

func littleBoyWorld(t *testing.T) (*World, map[string]string, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Origin", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 99010, Keywords: "boy", ShortDesc: "a little boy", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 10}, Alignment: 0, Race: 1, Int: 11, Wis: 11, Sex: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	watcher := NewPlayer(1, "Watcher", 1001)
	watcher.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	boy, err := w.SpawnMobQuiet(99010, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	boy.SetLevel(1)
	boy.SetPosition(combat.PosStanding)
	boy.CurrentHP = 30
	boy.MaxHP = 30
	boy.SetHealth(30)
	out["Watcher"] = ""
	return w, out, boy
}

// TestSpecLittleBoy_Exclaim pins SPECIAL(little_boy) (src/spec_procs2.c:1391-1403):
// on number(0,10)==0 it exclaims C's exact line (note the missing closing
// quote, verbatim) to TO_ROOM and returns TRUE; otherwise it is silent.
func TestSpecLittleBoy_Exclaim(t *testing.T) {
	// Firing roll.
	w, out, boy := littleBoyWorld(t)
	dprng.ResetStream(seedForNumber010(t, 0))
	if !specLittleBoy(w, nil, boy, "", "") {
		t.Fatal("firing roll should return true")
	}
	want := "A little boy exclaims, 'Don't let the bad man hurt me, mommy!\r\n"
	if got := out["Watcher"]; got != want {
		t.Fatalf("watcher = %q, want %q", got, want)
	}
	// Silent roll.
	w2, out2, boy2 := littleBoyWorld(t)
	dprng.ResetStream(seedForNumber010(t, 5))
	if specLittleBoy(w2, nil, boy2, "", "") {
		t.Fatal("silent roll should return false")
	}
	if got := out2["Watcher"]; got != "" {
		t.Fatalf("silent emitted %q", got)
	}
	// Gates: sleeping, command, player-trigger, fighting, negative HP.
	for _, gate := range []string{"sleeping", "command", "player", "fighting", "negative"} {
		t.Run(gate, func(t *testing.T) {
			w, out, boy := littleBoyWorld(t)
			var ch *Player
			cmd := ""
			switch gate {
			case "sleeping":
				boy.SetPosition(combat.PosSleeping)
			case "command":
				cmd = "look"
			case "player":
				ch = NewPlayer(9, "Trig", 1001)
			case "fighting":
				boy.SetFightingBody(boy)
			case "negative":
				boy.CurrentHP = -1
			}
			dprng.ResetStream(seedForNumber010(t, 0))
			if specLittleBoy(w, ch, boy, cmd, "") {
				t.Fatal("gate should return false")
			}
			if got := out["Watcher"]; got != "" {
				t.Fatalf("gate emitted %q", got)
			}
		})
	}
}

// TestSpecIra_ExclaimAndHit pins SPECIAL(ira) (src/spec_procs2.c:1405-1425):
// the first visible non-NOHASSLE player is exclaimed at and hit via the
// mob-hit seam, with no RNG and no invented warning; NOHASSLE and empty
// rooms are silent.
func TestSpecIra_ExclaimAndHit(t *testing.T) {
	t.Run("hits first player", func(t *testing.T) {
		w, out, mob := iraWorld(t)
		engine := &mickeyTestCombatEngine{}
		w.SetCombatEngine(engine)
		if !specIra(w, nil, mob, "", "") {
			t.Fatal("ira should handle the first player")
		}
		if got := strings.SplitN(out["Watcher"], "\r\n", 2)[0] + "\r\n"; got != "Ira exclaims, 'Dirty Orange Scum!'\r\n" {
			t.Fatalf("watcher first line = %q, want exact C exclaim line", got)
		}
		if len(engine.initialAttacks) != 1 {
			t.Fatalf("initial attacks = %d, want 1", len(engine.initialAttacks))
		}
	})
	t.Run("nohassle is skipped", func(t *testing.T) {
		w, out, mob := iraWorld(t)
		for _, p := range w.GetPlayersInRoom(mob.GetRoomVNum()) {
			p.SetPlrFlag(PrfNohassle, true)
		}
		if specIra(w, nil, mob, "", "") {
			t.Fatal("nohassle-only room should return false")
		}
		if got := out["Watcher"]; got != "" {
			t.Fatalf("nohassle emitted %q", got)
		}
	})
	t.Run("gates", func(t *testing.T) {
		w, _, mob := iraWorld(t)
		mob.SetPosition(combat.PosSleeping)
		if specIra(w, nil, mob, "", "") {
			t.Fatal("sleeping ira should return false")
		}
		mob.SetPosition(combat.PosStanding)
		mob.SetFightingBody(mob)
		if specIra(w, nil, mob, "", "") {
			t.Fatal("fighting ira should return false")
		}
	})
}

func iraWorld(t *testing.T) (*World, map[string]string, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Origin", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 99011, Keywords: "ira", ShortDesc: "Ira", Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 50}, Alignment: 0, Race: 1, Int: 11, Wis: 11, Sex: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	watcher := NewPlayer(1, "Watcher", 1001)
	watcher.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	ira, err := w.SpawnMobQuiet(99011, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	ira.SetLevel(10)
	ira.SetPosition(combat.PosStanding)
	ira.CurrentHP = 100
	ira.MaxHP = 100
	ira.SetHealth(100)
	out["Watcher"] = ""
	return w, out, ira
}
