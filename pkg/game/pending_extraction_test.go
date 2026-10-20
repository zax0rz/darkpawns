package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPendingBodyStaysAddressableUntilExtractionPass pins C's delayed
// extraction. extract_char on a mobile only sets MOB_EXTRACT and counts the
// pending extraction (src/handler.c:1194-1209); the body leaves character_list
// and world[room].people in extract_char_final, which extract_pending_chars runs
// at the start of the next heartbeat (src/comm.c:803-805,
// src/handler.c:1221-1248). Until then the body is still in the room: room
// listings hide it, but name resolution finds it, so a command that follows a
// kill in the same pulse still reaches the body — which is what makes C's
// `hit <just-killed mob>` swing (and lose carve-depth's attack roll when the
// port removed it eagerly).
func TestPendingBodyStaysAddressableUntilExtractionPass(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	killer := NewPlayer(1, "Killer", 1001)
	killer.SetLevel(5)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	victim := newSpecProcTestMob(t, w, 1001, 3)

	w.RawKillCombatant(victim, combat.TYPE_SLASH)

	if victim.IsAlive() {
		t.Fatal("killed mob is still alive")
	}
	if !victim.HasMobFlag(MobFlagExtract) {
		t.Fatal("killed mob is not flagged MOB_EXTRACT")
	}
	if got := w.GetMobsInRoom(1001); len(got) != 1 || got[0] != victim {
		t.Fatalf("room occupants = %d, want the pending-extract body", len(got))
	}
	// get_char_room_vis still finds the body (src/handler.c:1276-1300 has no
	// MOB_EXTRACT filter), which is what C's `hit` after `kill` relies on.
	if _, ok := w.resolveCharInRoomAt(killer, 1001, "testmob"); !ok {
		t.Fatal("pending body is no longer addressable by name")
	}

	w.ExtractPendingPlayers()

	if got := w.GetMobsInRoom(1001); len(got) != 0 {
		t.Fatalf("extraction pass left %d bodies in the room", len(got))
	}
	if victim.GetRoomVNum() >= 0 {
		t.Fatalf("extracted body still has room %d", victim.GetRoomVNum())
	}
}

// TestRoomSpecialLoopsIncludePendingBodies pins the other half of C's delayed
// extraction. Room-special bodies walk world[ch->in_room].people with no
// MOB_EXTRACT test — start_room (src/spec_procs.c:2209) and the Elemental
// Temple loops (src/spec_procs3.c:948, 1012, 1164, 1195, 1254) — so a body
// flagged earlier in the pulse is still an occupant and is moved like any
// other. Only special()'s room-mobile loop skips flagged bodies
// (src/interpreter.c:1453).
func TestRoomSpecialLoopsIncludePendingBodies(t *testing.T) {
	w, _ := newActorRoomTestWorld(t,
		parser.Room{VNum: 1326, Name: "Earth Platform"},
		parser.Room{VNum: 1314, Name: "Column Base"},
	)
	pet := newSpecProcTestMob(t, w, 1326, 1)
	w.RawKillCombatant(pet, combat.TYPE_SLASH)
	if !pet.HasMobFlag(MobFlagExtract) {
		t.Fatal("killed mob is not flagged MOB_EXTRACT")
	}

	people := cRoomPeople(w, 1326)
	if len(people) != 1 || people[0] != Actor(pet) {
		t.Fatalf("cRoomPeople = %d occupants, want the pending body", len(people))
	}

	specElementsPlatformsMove(w, 1326)

	if got := pet.GetRoomVNum(); got != 1314 {
		t.Fatalf("pending body room = %d, want 1314: C's room-special loops do not skip it", got)
	}
}
