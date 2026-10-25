package game

// mob_door_hunt_test.go — hunt_victim's door leg (src/graph.c:320-345) and
// the mob-side do_gen_door lane (src/act.movement.c:605-642): intelligent
// races open closed-unlocked doors and refuse locked ones, and the hunter
// moves through perform_move, which cannot cross a closed exit.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// doorHuntWorld is 1001 -north(door "gate")-> 1002 -north-> 1003 with a
// hunting mobile (RACE_NPCELF — intelligent) in 1001, the prey player in
// 1003, and listeners in 1001 and 1002.
func doorHuntWorld(t *testing.T, exitInfo int) (*World, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "South of Gate", Zone: 1, Exits: map[string]parser.Exit{
				"north": {
					Direction: "north", ToRoom: 1002, Keywords: "gate",
					DoorState: 1, ExitInfo: exitInfo, Key: 5001,
				},
			}},
			{VNum: 1002, Name: "North of Gate", Zone: 1, Exits: map[string]parser.Exit{
				"south": {
					Direction: "south", ToRoom: 1001, Keywords: "gate",
					DoorState: 1, ExitInfo: exitInfo, Key: 5001,
				},
				"north": {Direction: "north", ToRoom: 1003},
			}},
			{VNum: 1003, Name: "Prey Room", Zone: 1, Exits: map[string]parser.Exit{
				"south": {Direction: "south", ToRoom: 1002},
			}},
		},
		Mobs: []parser.Mob{
			{VNum: 9701, Keywords: "stalker", ShortDesc: "the dune stalker", Race: 1, Level: 20, Position: 8, DefaultPos: 8},
			{VNum: 9702, Keywords: "horse", ShortDesc: "a horse", Race: 12, Level: 5, Position: 8, DefaultPos: 8},
		},
		Objs: []parser.Obj{
			{VNum: 5001, Keywords: "brass key", ShortDesc: "a brass key"},
			{VNum: 5002, Keywords: "gate sign", ShortDesc: "a wooden sign"},
			{VNum: 5003, Keywords: "gate crate", ShortDesc: "a gate crate", TypeFlag: 15, // ITEM_CONTAINER
				Values: [4]int{0, contCloseable | contClosed, 0, 0}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }

	mob, err := w.SpawnMob(9701, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	prey := NewPlayer(1, "Prey", 1003)
	prey.SetPosition(8)
	if err := w.AddPlayer(prey); err != nil {
		t.Fatalf("AddPlayer prey: %v", err)
	}
	south := NewPlayer(2, "Southwatcher", 1001)
	south.SetPosition(8)
	if err := w.AddPlayer(south); err != nil {
		t.Fatalf("AddPlayer south: %v", err)
	}
	north := NewPlayer(3, "Northwatcher", 1002)
	north.SetPosition(8)
	if err := w.AddPlayer(north); err != nil {
		t.Fatalf("AddPlayer north: %v", err)
	}
	return w, mob, out
}

// scriptHuntDraws lives in mob_hunt_voice_depth_test.go.

func TestHuntVictimOpensClosedUnlockedDoor(t *testing.T) {
	w, mob, out := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	mob.SetHunting("Prey")
	// No evasion draw (prey has none); the trash-talk draw lands on the
	// silent default arm.
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1002 {
		t.Fatalf("hunter room = %d, want 1002 (through the opened gate)", got)
	}
	ext, ok := w.GetRoomInWorld(1001).Exits["north"]
	if !ok || ext.ExitInfo&parser.ExitClosed != 0 {
		t.Fatalf("gate not left open: %+v", ext)
	}
	back := w.GetRoomInWorld(1002).Exits["south"]
	if back.ExitInfo&parser.ExitClosed != 0 {
		t.Fatal("back side of the gate was not opened")
	}
	if want := "The dune stalker opens the gate.\r\n"; !strings.Contains(out["Southwatcher"], want) {
		t.Fatalf("south room bytes = %q, want %q among them", out["Southwatcher"], want)
	}
	if want := "The gate is opened from the other side.\r\n"; !strings.Contains(out["Northwatcher"], want) {
		t.Fatalf("north room bytes = %q, want %q among them", out["Northwatcher"], want)
	}
	// The mobile's own TO_CHAR lines vanish, as in C.
	if got := out[mob.GetShortDesc()]; strings.Contains(got, "Okay.") {
		t.Fatalf("mobile received its own door line: %q", got)
	}
}

func TestHuntVictimRefusesLockedDoor(t *testing.T) {
	w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed|parser.ExitLocked)
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("hunter crossed the locked gate to %d, want it stuck in 1001", got)
	}
	ext := w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&(parser.ExitClosed|parser.ExitLocked) != parser.ExitClosed|parser.ExitLocked {
		t.Fatalf("locked gate state changed: %#x", ext.ExitInfo)
	}
}

func TestHuntVictimNonIntelligentRaceDoesNotOpen(t *testing.T) {
	w, _, out := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	mob, err := w.SpawnMob(9702, 1001) // RACE_HORSE (12): not intelligent
	if err != nil {
		t.Fatalf("SpawnMob horse: %v", err)
	}
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("horse-race hunter crossed the closed gate to %d", got)
	}
	if strings.Contains(out["Southwatcher"], "opens the gate") {
		t.Fatal("non-intelligent hunter attempted to open the gate")
	}
}

func TestMobGenDoorUnlockNeedsKey(t *testing.T) {
	w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed|parser.ExitLocked)

	// Without the key, the unlock gate refuses (do_gen_door's has_key arm).
	w.mobGenDoor(mob, "gate", scmdUnlock)
	ext := w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&parser.ExitLocked == 0 {
		t.Fatal("keyless mobile unlocked the gate")
	}

	// With the key held, unlock then open — the mayor's 'O' cycle
	// (src/spec_procs.c 'O').
	key, err := w.SpawnObject(5001, -1)
	if err != nil {
		t.Fatalf("SpawnObject key: %v", err)
	}
	mob.AddToInventory(key)
	w.mobGenDoor(mob, "gate", scmdUnlock)
	w.mobGenDoor(mob, "gate", scmdOpen)
	ext = w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&parser.ExitLocked != 0 || ext.ExitInfo&parser.ExitClosed != 0 {
		t.Fatalf("keyed unlock+open left the gate shut: %#x", ext.ExitInfo)
	}
}

func TestMobIsIntelligentMatchesCRaceTable(t *testing.T) {
	for _, tc := range []struct {
		race int
		want bool
		why  string
	}{
		{0, true, "RACE_NPCHUMAN"},
		{1, true, "RACE_NPCELF"},
		{10, true, "RACE_DRAGON"},
		{11, true, "RACE_DEMON"},
		{12, false, "RACE_HORSE"},
		{13, false, "RACE_REPTILE"},
		{18, true, "RACE_GIANT"},
		{29, true, "RACE_NPCSSAUR"},
		{30, false, "RACE_NPCMINOTAUR — the 19th entry sits past NUM_INTEL_RACES=18 (src/utils.c:675-682)"},
	} {
		m := NewMob(&parser.Mob{VNum: 9800, Keywords: "x", ShortDesc: "x", Race: tc.race}, 1001)
		if got := mobIsIntelligent(m); got != tc.want {
			t.Errorf("race %d (%s): mobIsIntelligent = %v, want %v", tc.race, tc.why, got, tc.want)
		}
	}
}

// TestHuntVictimMultiWordKeywordDoesNotOpen pins C's argument quirk: the
// hunt passes "<keyword list> <direction>", and two_arguments takes the
// SECOND word of a multi-word list ("gate iron") as the direction token —
// which is not a direction, so do_gen_door dies at find_door and the hunter
// is stopped by the closed door. 233 world doors carry multi-word lists.
func TestHuntVictimMultiWordKeywordDoesNotOpen(t *testing.T) {
	w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	// "gate iron" instead of "gate".
	w.mu.Lock()
	north := w.rooms[1001].Exits["north"]
	north.Keywords = "gate iron"
	w.rooms[1001].Exits["north"] = north
	south := w.rooms[1002].Exits["south"]
	south.Keywords = "gate iron"
	w.rooms[1002].Exits["south"] = south
	w.mu.Unlock()
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("hunter passed the multi-word-keyword gate to %d, want it stopped in 1001", got)
	}
	ext := w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&parser.ExitClosed == 0 {
		t.Fatal("multi-word-keyword gate was opened")
	}
}

// TestHuntVictimSameKeywordOpensOnlyTravelExit pins find_door's direction
// branch: when the argument carries a direction word, only that exit is
// considered, so a second keyword-sharing exit in another direction is never
// touched.
func TestHuntVictimSameKeywordOpensOnlyTravelExit(t *testing.T) {
	w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	// A second closed, unlocked "gate" door to the east.
	w.mu.Lock()
	w.rooms[1001].Exits["east"] = parser.Exit{
		Direction: "east", ToRoom: 1002, Keywords: "gate",
		DoorState: 1, ExitInfo: parser.ExitIsDoor | parser.ExitClosed,
	}
	w.mu.Unlock()
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1002 {
		t.Fatalf("hunter room = %d, want 1002 through the north gate", got)
	}
	east := w.GetRoomInWorld(1001).Exits["east"]
	if east.ExitInfo&parser.ExitClosed == 0 {
		t.Fatal("the east gate sharing the keyword was opened")
	}
}

// TestHuntVictimKeywordObjectShadowsDoor pins generic_find running before
// find_door (src/act.movement.c:612): a visible object matching the keyword
// diverts the command away from the door — the door stays shut and the
// hunter is stopped by it.
func TestHuntVictimKeywordObjectShadowsDoor(t *testing.T) {
	w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	sign, err := w.SpawnObject(5002, 1001) // keywords "gate sign"
	if err != nil {
		t.Fatalf("SpawnObject sign: %v", err)
	}
	w.AddItemToRoom(sign, 1001) // SpawnObject does not register room items
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("hunter passed the shadowed gate to %d, want it stopped in 1001", got)
	}
	ext := w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&parser.ExitClosed == 0 {
		t.Fatal("the shadowed gate was opened")
	}
}

// setDoorKeywords rewrites both sides of the fixture gate's keyword list.
func setDoorKeywords(t *testing.T, w *World, keywords string) {
	t.Helper()
	w.mu.Lock()
	north := w.rooms[1001].Exits["north"]
	north.Keywords = keywords
	w.rooms[1001].Exits["north"] = north
	south := w.rooms[1002].Exits["south"]
	south.Keywords = keywords
	w.rooms[1002].Exits["south"] = south
	w.mu.Unlock()
}

// TestHuntVictimSecretCaseKeywordIsFindable pins C's case-sensitive
// strstr(keyword, "secret") (src/act.movement.c:382, 417): a capitalized
// "Secret door" keyword list does NOT hide the exit, so the hunter opens it;
// only the lowercase form is hidden.
func TestHuntVictimSecretCaseKeywordIsFindable(t *testing.T) {
	t.Run("capital Secret stays findable", func(t *testing.T) {
		w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
		// Single-word: a two-word list stops the hunter through the
		// direction-token rule regardless of case (see the multi-word test).
		setDoorKeywords(t, w, "Secret")
		mob.SetHunting("Prey")
		scriptHuntDraws(t, 150)

		w.huntVictim(mob)

		if got := mob.GetRoom(); got != 1002 {
			t.Fatalf("hunter stopped by the 'Secret' gate at %d, want it through to 1002", got)
		}
	})

	t.Run("lowercase secret stays hidden", func(t *testing.T) {
		w, mob, _ := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
		setDoorKeywords(t, w, "secret")
		mob.SetHunting("Prey")
		scriptHuntDraws(t, 150)

		w.huntVictim(mob)

		if got := mob.GetRoom(); got != 1001 {
			t.Fatalf("hunter passed the hidden 'secret' gate to %d", got)
		}
	})
}

// TestFindDoorSecretCaseSensitive pins the player lane's copy of the same
// strstr (src/act.movement.c:382 direction branch, :417 keyword scan).
func TestFindDoorSecretCaseSensitive(t *testing.T) {
	for _, tc := range []struct {
		keywords string
		want     int
	}{
		{"Secret door", 0},
		{"secret door", -1},
	} {
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{
			{VNum: 1001, Name: "R", Zone: 1, Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 1002, Keywords: tc.keywords},
			}},
			{VNum: 1002, Name: "R2", Zone: 1},
		}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		ch := NewPlayer(1, "Opener", 1001)
		if err := w.AddPlayer(ch); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		if got := findDoor(w, ch, "door", "north", "open"); got != tc.want {
			t.Errorf("keywords %q direction branch: findDoor = %d, want %d", tc.keywords, got, tc.want)
		}
		if got := findDoor(w, ch, "door", "", "open"); got != tc.want {
			t.Errorf("keywords %q keyword scan: findDoor = %d, want %d", tc.keywords, got, tc.want)
		}
	}
}

// TestHuntVictimOpensMatchingContainer pins generic_find's object arm going
// all the way through C's ladder into do_doorcmd (src/act.movement.c:
// 612-635, 484-511): a closed unlocked container matching the door keyword
// is opened — not the door — and the room sees do_doorcmd's object act.
func TestHuntVictimOpensMatchingContainer(t *testing.T) {
	w, mob, out := doorHuntWorld(t, parser.ExitIsDoor|parser.ExitClosed)
	crate, err := w.SpawnObject(5003, 1001)
	if err != nil {
		t.Fatalf("SpawnObject crate: %v", err)
	}
	w.AddItemToRoom(crate, 1001)
	mob.SetHunting("Prey")
	scriptHuntDraws(t, 150)

	w.huntVictim(mob)

	if flags := crate.GetValue(contFlags); flags&contClosed != 0 {
		t.Fatalf("container still closed: flags=%#x", flags)
	}
	if flags := crate.GetValue(contFlags); flags&contLocked != 0 {
		t.Fatalf("container locked by the open: flags=%#x", flags)
	}
	ext := w.GetRoomInWorld(1001).Exits["north"]
	if ext.ExitInfo&parser.ExitClosed == 0 {
		t.Fatal("the door was opened instead of the container")
	}
	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("hunter passed the still-closed gate to %d", got)
	}
	if want := "The dune stalker opens a gate crate.\r\n"; !strings.Contains(out["Southwatcher"], want) {
		t.Fatalf("south room bytes = %q, want %q among them", out["Southwatcher"], want)
	}
}
