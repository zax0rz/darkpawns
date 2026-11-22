package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// seedForNumber06 finds a seed whose first number(0,6) is want.
func seedForNumber06(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(0, 6) == want {
			return seed
		}
	}
	t.Fatalf("no seed draws %d first", want)
	return 0
}

// TestSpecPortalRoom_Teleport pins SPECIAL(portal_room)
// (src/spec_procs2.c:1648-1677): say kallinistra triggers do_say, the
// teleport stc, origin/destination acts, relocation to 21264 and do_look;
// anything else is silent.
func TestSpecPortalRoom_Teleport(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Origin", Zone: 1},
		{VNum: 21264, Name: "Destiny", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	actor := NewPlayer(1, "Actor", 1001)
	actor.SetPosition(combat.PosStanding)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	actor.CopyBaseAttributes()
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	watcher.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	destWatcher := NewPlayer(3, "Destwatch", 21264)
	destWatcher.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(destWatcher); err != nil {
		t.Fatalf("AddPlayer destwatch: %v", err)
	}
	out["Actor"] = ""
	out["Watcher"] = ""

	for _, command := range []string{"say", "'"} {
		for name := range out {
			out[name] = ""
		}
		actor.SetRoom(1001)
		if !specPortalRoom(w, actor, nil, command, "kallinistra") {
			t.Fatalf("%s kallinistra should be handled", command)
		}
		if got := actor.GetRoomVNum(); got != 21264 {
			t.Fatalf("%s actor room = %d, want 21264", command, got)
		}
		if got, want := out["Actor"], "You say 'kallinistra'\r\n\r\n"+
			"With a blinding flash of light and a crack of thunder, you are teleported...\r\n"+
			"Destiny\r\n[ Exits: None! ]\r\nDestwatch  is standing here.\r\n"; got != want {
			t.Errorf("%s actor transcript = %q, want %q", command, got, want)
		}
		if got, want := out["Watcher"], "Actor says, 'kallinistra'\r\n\r\n"+
			"With a blinding flash of light and a crack of thunder, Actor disappears!\r\n\r\n\r\n"; got != want {
			t.Errorf("%s origin transcript = %q, want %q", command, got, want)
		}
		if got, want := out["Destwatch"], "\r\n"+
			"With a blinding flash of light and a crack of thunder, Actor appears!\r\n\r\n\r\n"; got != want {
			t.Errorf("%s destination transcript = %q, want %q", command, got, want)
		}
	}
	// Gates: wrong command, wrong phrase, and exact C argument comparison.
	for _, tc := range []struct{ cmd, arg string }{{"look", "kallinistra"}, {"say", "other"}, {"'", "other"}} {
		for name := range out {
			out[name] = ""
		}
		actor.SetRoom(1001)
		if specPortalRoom(w, actor, nil, tc.cmd, tc.arg) {
			t.Fatalf("%s %q should not be handled", tc.cmd, tc.arg)
		}
		if got := actor.GetRoomVNum(); got != 1001 {
			t.Fatalf("gate moved to %d", got)
		}
		if got := out["Actor"] + out["Watcher"] + out["Destwatch"]; got != "" {
			t.Fatalf("gate %s %q emitted %q", tc.cmd, tc.arg, got)
		}
	}
}

// TestSpecBatRoom_WakesSleeping pins SPECIAL(bat_room)
// (src/spec_procs2.c:1756-1768): each sleeping bat named "bat" wakes on
// number(0,6)==0 with TO_ROOM bytes and standing; always returns FALSE.
func TestSpecBatRoom_WakesSleeping(t *testing.T) {
	w, out, entrant, bat := batRoomWorld(t)
	dprng.ResetStream(seedForNumber06(t, 0))
	if specBatRoom(w, entrant, nil, "north", "") {
		t.Fatal("bat_room always returns false (C returns FALSE)")
	}
	want := "Your movements wake up A bat!\r\n"
	if got := out["Watcher"]; got != want {
		t.Fatalf("watcher = %q, want %q", got, want)
	}
	if got := bat.GetPosition(); got != combat.PosStanding {
		t.Fatalf("bat pos = %d, want standing", got)
	}
	// Silent roll leaves the bat asleep.
	w2, out2, entrant2, bat2 := batRoomWorld(t)
	dprng.ResetStream(seedForNumber06(t, 3))
	if specBatRoom(w2, entrant2, nil, "north", "") {
		t.Fatal("silent roll still returns false")
	}
	if got := out2["Watcher"]; got != "" {
		t.Fatalf("silent emitted %q", got)
	}
	if got := bat2.GetPosition(); got != combat.PosSleeping {
		t.Fatalf("bat pos = %d, want sleeping", got)
	}
}

func batRoomWorld(t *testing.T) (*World, map[string]string, *Player, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Cave", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 99012, Keywords: "bat", ShortDesc: "A bat", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 10}, Alignment: 0, Race: 1, Int: 11, Wis: 11}},
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
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	entrant := NewPlayer(2, "Entrant", 1001)
	entrant.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(entrant); err != nil {
		t.Fatalf("AddPlayer entrant: %v", err)
	}
	bat, err := w.SpawnMobQuiet(99012, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	bat.SetLevel(1)
	bat.SetPosition(combat.PosSleeping)
	bat.CurrentHP = 20
	bat.MaxHP = 20
	bat.SetHealth(20)
	out["Watcher"] = ""
	return w, out, entrant, bat
}

// TestSpecBat_Arms pins SPECIAL(bat) (src/spec_procs2.c:1770-1821): the
// 0-5 arms emit TO_ROOM lines and return FALSE; the 6 arm sleeps the bat
// when alone (TRUE) or wakes a sleeping bat (TRUE).
func TestSpecBat_Arms(t *testing.T) {
	want := map[int]string{
		0: "A bat swoops down, narrowly missing your face.\r\n",
		1: "A bat swoops down, narrowly missing your face.\r\n",
		2: "A bat lets out a shrill, piercing scream.\r\n",
		3: "A bat lets out a shrill, piercing scream.\r\n",
		4: "Circling around your head, A bat lets out a high-pitched shriek.\r\n",
		5: "Circling around your head, A bat lets out a high-pitched shriek.\r\n",
	}
	for pick, line := range want {
		t.Run(string(rune('0'+pick)), func(t *testing.T) {
			w, out, bat := batWorld(t, false)
			dprng.ResetStream(seedForNumber06(t, pick))
			if specBat(w, nil, bat, "", "") {
				t.Fatal("arms 0-5 must return false")
			}
			if got := out["Watcher"]; got != line {
				t.Fatalf("watcher = %q, want %q", got, line)
			}
		})
	}
	// 6 alone: goes to sleep, returns TRUE. No player in the room, so
	// C's waking check is false (the Watcher helper would count).
	w, _, bat := batWorld(t, false)
	if watcher, ok := w.GetPlayer("Watcher"); ok {
		watcher.SetRoom(9999)
	}
	dprng.ResetStream(seedForNumber06(t, 6))
	if !specBat(w, nil, bat, "", "") {
		t.Fatal("sleep arm should return true")
	}
	if got := bat.GetPosition(); got != combat.PosSleeping {
		t.Fatalf("bat pos = %d, want sleeping", got)
	}
	// 6 with sleeper: wakes it, returns TRUE.
	w2, out2, bat2 := batWorld(t, true)
	dprng.ResetStream(seedForNumber06(t, 6))
	if !specBat(w2, nil, bat2, "", "") {
		t.Fatal("wake arm should return true")
	}
	if got := out2["Watcher"]; !strings.Contains(got, "wakes") {
		t.Fatalf("wake bytes = %q, want wakes line", got)
	}
}

func batWorld(t *testing.T, withSleeper bool) (*World, map[string]string, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Cave", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 99013, Keywords: "bat", ShortDesc: "A bat", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 10}, Alignment: 0, Race: 1, Int: 11, Wis: 11}},
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
	bat, err := w.SpawnMobQuiet(99013, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	bat.SetLevel(1)
	bat.SetPosition(combat.PosStanding)
	bat.CurrentHP = 20
	bat.MaxHP = 20
	bat.SetHealth(20)
	if withSleeper {
		sleeper, err := w.SpawnMobQuiet(99013, 1001)
		if err != nil {
			t.Fatalf("SpawnMobQuiet sleeper: %v", err)
		}
		sleeper.SetPosition(combat.PosSleeping)
		sleeper.CurrentHP = 20
		sleeper.MaxHP = 20
		sleeper.SetHealth(20)
	}
	out["Watcher"] = ""
	return w, out, bat
}
