package game

// unassigned_script_mudlog_test.go — R5h proofs for run_script's empty-name
// producer (src/scripts.c:1763-1767, DP-1416).
//
// Builder-reachable state: the owner's script record carries a trigger flag but
// no name. C creates every owner's script record with name = NULL at load (mobs
// src/db.c:1278-1279, rooms :803-804, objects :1374-1375), so a builder who sets
// a trigger flag in the OLC script menu without ever setting a name leaves the
// record with name == NULL and run_script takes its !*script_name arm.
//
// R5e note on the OLC routes: clearing an existing name is possible for rooms
// (src/redit.c:1032-1037) and objects (src/oedit.c:1447-1451), whose parsers
// have no numeric gate, but NOT for mobiles — medit_parse rejects empty input
// for every mode above MEDIT_NUMERICAL_RESPONSE (src/medit.c:718-722;
// MEDIT_SCRIPT_NAME is 28 > 10), so the mobile route is flag-without-name only.
// Go reproduces both (pkg/session/medit.go:650, :739). The resulting state, and
// therefore the producer, is the same.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// MS_ONPULSE_ALL (src/structs.h mscript bits).
const unassignedMobOnPulse = 64

// unassignedScriptWorld is a room (RS_ONCMD, no name), a mobile (MS_ONPULSE_ALL,
// no name) and an actor standing in the room.
func unassignedScriptWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Unassigned", Zone: 1, ScriptFunctions: rsOnCmd}},
		Mobs: []parser.Mob{{
			VNum: 2001, Keywords: "goblin", ShortDesc: "a goblin guard",
			Level: 20, LuaFunctions: unassignedMobOnPulse,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Actor", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, ch
}

// unassignedScriptObj registers one object instance whose prototype carries the
// given trigger bit and no name.
func unassignedScriptObj(w *World, bit int) *ObjectInstance {
	obj := NewObjectInstance(&parser.Obj{
		VNum: 3001, Keywords: "orb", ShortDesc: "a dull orb", LuaFunctions: bit,
	}, -1)
	obj.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[obj.ID] = obj
	return obj
}

// src/scripts.c:1763-1767: the empty-name arm logs at BRF / LVL_IMMORT /
// file TRUE, names me, and returns TRUE (the trigger is consumed). me is the
// mobile, so the payload is its short description and prototype vnum.
func TestUnassignedScriptMobProducer(t *testing.T) {
	w, _ := unassignedScriptWorld(t)
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	s.probe = func() {
		if !w.mu.TryLock() {
			t.Fatal("world lock held at the mob producer")
		}
		w.mu.Unlock()
	}
	mob, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	// C's gate is record-and-flag, never the name (mobact.c:161).
	if !mob.HasScript("onpulse_all") {
		t.Fatal("HasScript must pass on the flag alone")
	}
	handled, err := mob.RunScript("onpulse_all", mob.CreateSelfScriptContext())
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("C returns TRUE here; the trigger must be consumed")
	}
	if len(*calls) != 0 {
		t.Fatalf("the engine ran on an unassigned script: %v", *calls)
	}
	requireDiagnostic(t, s, file,
		"SYSERR: Attempting to call unassigned script for a goblin guard (#2001).", true)
}

// The room owner passes me = ch (a player), so GET_MOB_VNUM is -1
// (src/utils.h:431-432) and a TRUE return consumes the command
// (interpreter.c:1419-1423).
func TestUnassignedScriptRoomOnCmdConsumes(t *testing.T) {
	w, ch := unassignedScriptWorld(t)
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	s.probe = func() {
		if !w.mu.TryLock() {
			t.Fatal("world lock held at the room producer")
		}
		w.mu.Unlock()
	}
	if !w.RunRoomOnCmdScript(ch, "look") {
		t.Fatal("C returns TRUE here; the command must be consumed")
	}
	if len(*calls) != 0 {
		t.Fatalf("the engine ran on an unassigned script: %v", *calls)
	}
	requireDiagnostic(t, s, file,
		"SYSERR: Attempting to call unassigned script for Actor (#-1).", true)
}

// The object oncmd arms pass me = ch; the worn and carried sites are PC-only
// (interpreter.c:1430, :1443), so the payload is the player's name and -1.
func TestUnassignedScriptObjOnCmdConsumes(t *testing.T) {
	w, ch := unassignedScriptWorld(t)
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	obj := unassignedScriptObj(w, osOnCmd)
	if !w.RunObjOnCmdScript(&scripting.CharRef{ID: ch.ID}, ch, obj, 1001, "look") {
		t.Fatal("C returns TRUE here; the command must be consumed")
	}
	if len(*calls) != 0 {
		t.Fatalf("the engine ran on an unassigned script: %v", *calls)
	}
	requireDiagnostic(t, s, file,
		"SYSERR: Attempting to call unassigned script for Actor (#-1).", true)
}

// The room-contents oncmd site has no IS_NPC gate (interpreter.c:1470-1476), so
// a switched mobile's actor is named by its short description and vnum.
func TestUnassignedScriptObjOnCmdSwitchedMobActor(t *testing.T) {
	w, _ := unassignedScriptWorld(t)
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	mob, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	obj := unassignedScriptObj(w, osOnCmd)
	if !w.RunObjOnCmdScript(&scripting.CharRef{NPC: true, ID: mob.GetID()}, nil, obj, 1001, "look") {
		t.Fatal("C returns TRUE here; the command must be consumed")
	}
	if len(*calls) != 0 {
		t.Fatalf("the engine ran on an unassigned script: %v", *calls)
	}
	requireDiagnostic(t, s, file,
		"SYSERR: Attempting to call unassigned script for a goblin guard (#2001).", true)
}

// DP-1416 approved divergence: C's object-onpulse caller passes me = NULL
// (comm.c:791-793), so the !*script_name arm formats GET_NAME(NULL) and faults
// before emitting. Go emits nothing and does not panic; C ignores this
// trigger's return value, so there is no consume difference.
func TestUnassignedScriptObjOnPulseIsSilentDivergence(t *testing.T) {
	w, _ := unassignedScriptWorld(t)
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	obj := unassignedScriptObj(w, osOnPulse)
	w.RunObjPulseScript(obj)
	if len(s.messages) != 0 || file.Len() != 0 {
		t.Fatalf("object-onpulse divergence emitted %v (file %q)", s.messages, file.String())
	}
	if len(*calls) != 0 {
		t.Fatalf("the engine ran on an unassigned script: %v", *calls)
	}
}

// A below-LVL_IMMORT observer and a writer both see nothing (utils.c:258-270).
func TestUnassignedScriptAudienceFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*Player)
	}{
		{"below level", func(p *Player) { p.SetLevel(LVL_IMMORT - 1) }},
		{"writing", func(p *Player) { p.SetPlrFlag(PlrWriting, true) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := unassignedScriptWorld(t)
			installRoomObjRecorder(t)
			s, _ := diagnosticCapture(t, MudlogBrief)
			tc.prepare(s.observer)
			if !w.RunRoomOnCmdScript(ch, "look") {
				t.Fatal("the trigger is still consumed")
			}
			if len(s.messages) != 0 {
				t.Fatalf("%s observer saw %v", tc.name, s.messages)
			}
		})
	}
}

// Control (R5h): a named script still reaches the engine and its TRUE return
// still consumes the command. This fails if the empty-name branch swallows the
// named case.
func TestUnassignedScriptNamedControl(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Scripted", Zone: 1, ScriptName: "room.lua", ScriptFunctions: rsOnCmd},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Actor", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	calls := installRoomObjRecorder(t)
	s, file := diagnosticCapture(t, MudlogBrief)
	if !w.RunRoomOnCmdScript(ch, "look") {
		t.Fatal("a named script's TRUE return consumes the command")
	}
	if len(*calls) != 1 {
		t.Fatalf("named room script did not run: %v", *calls)
	}
	if len(s.messages) != 0 || file.Len() != 0 {
		t.Fatalf("a named script emitted the unassigned producer: %v (file %q)", s.messages, file.String())
	}
}
