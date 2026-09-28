package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// The registries this file extends are filled during world init and then only
// read, so a test may add its own room/mob/object and put the map back
// afterwards. Unique vnums keep the probe rows apart from real assignments.
const (
	specProbeRoom   = 991001
	specProbeMobA   = 991002
	specProbeMobB   = 991003
	specProbeMobEx  = 991004
	specProbeBodyEq = 991007
	specProbeHeadEq = 991008
)

// installRoomSpec maps vnum to a fresh probe spec and restores the registry.
func installRoomSpec(t *testing.T, vnum int, name string, fn game.SpecFunc) {
	t.Helper()
	prev, had := game.RoomSpecAssign[vnum]
	game.RoomSpecAssign[vnum] = name
	game.RegisterSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.RoomSpecAssign[vnum] = prev
		} else {
			delete(game.RoomSpecAssign, vnum)
		}
		delete(game.SpecRegistry, name)
	})
}

// installMobSpec maps a mob vnum to a fresh probe spec and restores it.
func installMobSpec(t *testing.T, vnum int, name string, fn game.SpecFunc) {
	t.Helper()
	prev, had := game.MobSpecAssign[vnum]
	game.MobSpecAssign[vnum] = name
	game.RegisterSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.MobSpecAssign[vnum] = prev
		} else {
			delete(game.MobSpecAssign, vnum)
		}
		delete(game.SpecRegistry, name)
	})
}

// installObjSpec maps an object vnum to a fresh probe spec and restores it.
func installObjSpec(t *testing.T, vnum int, name string, fn game.ObjSpecFunc) {
	t.Helper()
	prev, had := game.ObjSpecAssign[vnum]
	game.ObjSpecAssign[vnum] = name
	game.RegisterObjSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.ObjSpecAssign[vnum] = prev
		} else {
			delete(game.ObjSpecAssign, vnum)
		}
		delete(game.ObjSpecRegistry, name)
	})
}

// specProbeSession builds a one-room world holding mobs, a manager over it and
// a session standing in that room.
func specProbeSession(t *testing.T, mobs []parser.Mob) (*Manager, *Session) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: specProbeRoom, Name: "Probe Room", Zone: 1}},
		Mobs:  mobs,
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Specprobe", 1, specProbeRoom)
	s.player.SetPosition(combat.PosStanding)
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return m, s
}

// TestRunSpecialsRoomPrecedesMob pins C's order: special() checks the room
// before world[room].people (src/interpreter.c:1414-1416 then 1452-1466), and
// the first special returning TRUE consumes the command. The pre-DP-1336 port
// ran every mobile before the room special.
func TestRunSpecialsRoomPrecedesMob(t *testing.T) {
	var calls []string
	installRoomSpec(t, specProbeRoom, "probe_room_first", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		calls = append(calls, "room:"+cmd)
		return true
	})
	installMobSpec(t, specProbeMobA, "probe_mob_second", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		calls = append(calls, "mob:"+cmd)
		return true
	})
	m, s := specProbeSession(t, []parser.Mob{{VNum: specProbeMobA, ShortDesc: "a probe mob", Keywords: "probe", Level: 1}})
	if _, err := m.world.SpawnMob(specProbeMobA, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	if err := ExecuteCommand(s, "look", nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if len(calls) != 1 || calls[0] != "room:look" {
		t.Fatalf("specials called = %v, want only the room special", calls)
	}
}

// TestRunSpecialsNewestMobWinsDeterministically pins the people-list order:
// char_to_room prepends, so the most recent arrival is first and its special
// wins (src/interpreter.c:1452-1466). The dispatch repeats 50 times because
// walking mobs in map order would pick a different winner on most runs.
func TestRunSpecialsNewestMobWinsDeterministically(t *testing.T) {
	winner := ""
	installMobSpec(t, specProbeMobA, "probe_mob_older", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		winner = "older"
		return true
	})
	installMobSpec(t, specProbeMobB, "probe_mob_newer", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		winner = "newer"
		return true
	})
	m, s := specProbeSession(t, []parser.Mob{
		{VNum: specProbeMobA, ShortDesc: "an older probe", Keywords: "older", Level: 1},
		{VNum: specProbeMobB, ShortDesc: "a newer probe", Keywords: "newer", Level: 1},
	})
	if _, err := m.world.SpawnMob(specProbeMobA, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob older: %v", err)
	}
	if _, err := m.world.SpawnMob(specProbeMobB, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob newer: %v", err)
	}

	for i := 0; i < 50; i++ {
		winner = ""
		if err := ExecuteCommand(s, "look", nil); err != nil {
			t.Fatalf("ExecuteCommand #%d: %v", i, err)
		}
		if winner != "newer" {
			t.Fatalf("dispatch #%d: %q special won, want the most recent arrival", i, winner)
		}
	}
}

// TestRunSpecialsSkipsExtractFlaggedMob pins MOB_EXTRACT: C skips an extracted
// mobile entirely (src/interpreter.c:1453).
func TestRunSpecialsSkipsExtractFlaggedMob(t *testing.T) {
	called := false
	installMobSpec(t, specProbeMobEx, "probe_mob_extract", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		called = true
		return true
	})
	m, s := specProbeSession(t, []parser.Mob{{VNum: specProbeMobEx, ShortDesc: "a departing probe", Keywords: "departing", Level: 1}})
	mob, err := m.world.SpawnMob(specProbeMobEx, specProbeRoom)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetMobFlag(game.MobFlagExtract)

	if err := ExecuteCommand(s, "look", nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if called {
		t.Fatal("a MOB_EXTRACT mobile's special ran")
	}
}

// TestRunSpecialsSeesResolvedCommandName pins CMD_NAME: C resolves the command
// first and hands special() cmd_info[cmd].command (src/interpreter.c:905-949,
// src/interpreter.h:35), so a typed "s" reaches a special as "south".
func TestRunSpecialsSeesResolvedCommandName(t *testing.T) {
	seen := ""
	installMobSpec(t, specProbeMobA, "probe_mob_name", func(w *game.World, ch *game.Player, me *game.MobInstance, cmd, arg string) bool {
		seen = cmd
		return true
	})
	m, s := specProbeSession(t, []parser.Mob{{VNum: specProbeMobA, ShortDesc: "a naming probe", Keywords: "naming", Level: 1}})
	if _, err := m.world.SpawnMob(specProbeMobA, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	if err := ExecuteCommand(s, "s", nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if seen != "south" {
		t.Fatalf("special received cmd %q, want the resolved name south", seen)
	}
}

// TestRunSpecialsWornOrderFollowsCWearIndex pins the worn scan to C's WEAR_
// numbering (src/structs.h:390-411), not to Go's EquipmentSlot values: C walks
// WEAR_BODY (5) before WEAR_HEAD (6), while Go numbers Head 0 and Body 1. The
// pre-DP-1336 port iterated the equipment map, so either item could win; the
// dispatch repeats 50 times to catch that.
func TestRunSpecialsWornOrderFollowsCWearIndex(t *testing.T) {
	winner := ""
	installObjSpec(t, specProbeBodyEq, "probe_obj_body", func(w *game.World, ch *game.Player, obj *game.ObjectInstance, cmd, arg string) bool {
		winner = "body"
		return true
	})
	installObjSpec(t, specProbeHeadEq, "probe_obj_head", func(w *game.World, ch *game.Player, obj *game.ObjectInstance, cmd, arg string) bool {
		winner = "head"
		return true
	})
	_, s := specProbeSession(t, nil)

	body := game.NewObjectInstance(&parser.Obj{
		VNum: specProbeBodyEq, Keywords: "vest", ShortDesc: "a leather vest",
		TypeFlag: 9, WearFlags: [4]int{1 << 3}, // ITEM_WEAR_BODY
	}, -1)
	head := game.NewObjectInstance(&parser.Obj{
		VNum: specProbeHeadEq, Keywords: "helm", ShortDesc: "a steel helm",
		TypeFlag: 9, WearFlags: [4]int{1 << 4}, // ITEM_WEAR_HEAD
	}, -1)
	for _, item := range []*game.ObjectInstance{body, head} {
		if err := s.player.Inventory.AddItem(item); err != nil {
			t.Fatalf("AddItem %s: %v", item.GetShortDesc(), err)
		}
		if err := s.player.Equipment.Equip(item, s.player.Inventory); err != nil {
			t.Fatalf("Equip %s: %v", item.GetShortDesc(), err)
		}
	}

	for i := 0; i < 50; i++ {
		winner = ""
		if err := ExecuteCommand(s, "look", nil); err != nil {
			t.Fatalf("ExecuteCommand #%d: %v", i, err)
		}
		if winner != "body" {
			t.Fatalf("dispatch #%d: %q won, want the lower C WEAR_ index (WEAR_BODY 5)", i, winner)
		}
	}
}
