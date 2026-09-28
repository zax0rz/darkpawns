package session

// room_obj_script_gates_test.go — R5h unit proof for special()'s !IS_NPC
// gates (interpreter.c:1419-1476): a session acting through a switched
// mobile runs none of the room, worn or carried oncmd scripts, but still
// runs the room-object oncmd, which has no IS_NPC gate in C — with the
// mobile as ch and me.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// gateRecorder records every oncmd run with its owner type and whether ch/me
// name the switched mobile.
type gateRecorder struct {
	calls *[]string
}

func (gateRecorder) ForgetFailures() {}

func (r gateRecorder) RunScript(ctx *game.ScriptContext, filename, trigger string) (bool, error) {
	var b strings.Builder
	b.WriteString(filename)
	b.WriteString(":")
	b.WriteString(trigger)
	b.WriteString(":owner=")
	b.WriteString(ctx.OwnerType)
	if ctx.MeRef != nil && ctx.MeRef.NPC {
		b.WriteString(":me=npc")
	} else if ctx.MeRef != nil {
		b.WriteString(":me=pc")
	}
	*r.calls = append(*r.calls, b.String())
	// Not handled: every oncmd site must be visited, not just the first.
	return false, nil
}

// newOnCmdGatesWorld builds a manager whose room 1001 carries RS_ONCMD and
// holds one OS_ONCMD object on the floor; the session's player wears one
// OS_ONCMD object and carries another.
func newOnCmdGatesWorld(t *testing.T) (*Manager, *Session) {
	t.Helper()
	scriptedObj := func(vnum int) parser.Obj {
		return parser.Obj{
			VNum: vnum, Keywords: "token", ShortDesc: "a token",
			ScriptName: "gate-obj.lua", LuaFunctions: 1 << 1,
			WearFlags: [4]int{(1 << 0) | (1 << 13)}, // TAKE + WEAR_HOLD, as the transfer tests equip
		}
	}
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Scripted", Zone: 1, ScriptName: "gate-room.lua", ScriptFunctions: 1 << 5},
			{VNum: 1002, Name: "Plain", Zone: 1},
		},
		Objs: []parser.Obj{scriptedObj(7001), scriptedObj(7002), scriptedObj(7003)},
		Mobs: []parser.Mob{{VNum: 2001, Keywords: "switchable", ShortDesc: "a switchable mob"}},
	}
	m := newTestManager(t, mustWorld(t, parsed), nil)

	s := makeTestSession(t, m, "Gatekeeper", 1001, true)
	w := m.world
	if err := w.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	spawn := func(vnum int) *game.ObjectInstance {
		t.Helper()
		obj, err := w.SpawnObject(vnum, -1)
		if err != nil {
			t.Fatalf("SpawnObject(%d): %v", vnum, err)
		}
		return obj
	}

	// Worn (equipment) object, equipped straight from the void as objsave's
	// load path does (it never passes through the inventory list).
	worn := spawn(7001)
	if err := s.player.Equipment.Equip(worn, s.player.Inventory); err != nil {
		t.Fatalf("equip worn object: %v", err)
	}
	// Carried object.
	carried := spawn(7002)
	if err := w.MoveObjectToPlayerInventory(carried, s.player); err != nil {
		t.Fatalf("seat carried object: %v", err)
	}
	// Room floor object.
	floor := spawn(7003)
	if err := w.MoveObjectToRoom(floor, 1001); err != nil {
		t.Fatalf("seat floor object: %v", err)
	}

	calls := &[]string{}
	prev := game.ScriptEngine
	game.ScriptEngine = gateRecorder{calls: calls}
	t.Cleanup(func() { game.ScriptEngine = prev })

	return m, s
}

func mustWorld(t *testing.T, parsed *parser.World) *game.World {
	t.Helper()
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	return w
}

// TestRunSpecialsRoomObjScriptNPCGates proves the !IS_NPC gates of C's
// special(): a PC session reaches the room, worn, carried and room-object
// oncmd arms in order; a switched session reaches only the room-object arm,
// with the mobile as run_script's ch and me.
func TestRunSpecialsRoomObjScriptNPCGates(t *testing.T) {
	t.Run("pc-runs-all-four-arms", func(t *testing.T) {
		_, s := newOnCmdGatesWorld(t)
		if runSpecials(s, "look", nil, "") {
			t.Fatal("runSpecials consumed the command though no script handled it")
		}
		want := []string{
			"gate-room.lua:oncmd:owner=room:me=pc",
			"gate-obj.lua:oncmd:owner=obj:me=pc", // worn
			"gate-obj.lua:oncmd:owner=obj:me=pc", // carried
			"gate-obj.lua:oncmd:owner=obj:me=pc", // room floor
		}
		got := *scriptGateCalls(t)
		if len(got) != len(want) {
			t.Fatalf("oncmd calls = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("oncmd call %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("switched-runs-only-room-object", func(t *testing.T) {
		m, s := newOnCmdGatesWorld(t)
		mob, err := m.world.SpawnMob(2001, 1001)
		if err != nil {
			t.Fatalf("SpawnMob: %v", err)
		}
		s.isSwitched = true
		s.switchedMob = mob

		runSpecials(s, "look", nil, "")
		got := *scriptGateCalls(t)
		want := []string{"gate-obj.lua:oncmd:owner=obj:me=npc"}
		if len(got) != 1 || got[0] != want[0] {
			t.Fatalf("switched oncmd calls = %v, want %v", got, want)
		}
	})
}

// scriptGateCalls reads the recorder's calls slice installed by
// newOnCmdGatesWorld. The recorder is package state (game.ScriptEngine), so
// the test grabs it through the same pointer the recorder writes.
func scriptGateCalls(t *testing.T) *[]string {
	rec, ok := game.ScriptEngine.(gateRecorder)
	if !ok {
		t.Fatal("gate recorder not installed")
	}
	return rec.calls
}
