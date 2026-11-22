package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

type mobScriptCall struct {
	filename string
	trigger  string
	actor    string
	chRefID  int
	chRefNPC bool
	meRefID  int
	meRefNPC bool
	roomVNum int
	worldSet bool
}

type mobScriptRecorder struct {
	calls *[]mobScriptCall
}

func (r mobScriptRecorder) RunScript(ctx *ScriptContext, filename, trigger string) (bool, error) {
	actor := ""
	if ctx.Ch != nil {
		actor = ctx.Ch.GetName()
	}
	call := mobScriptCall{
		filename: filename,
		trigger:  trigger,
		actor:    actor,
		roomVNum: ctx.RoomVNum,
		worldSet: ctx.World != nil,
	}
	if ctx.ChRef != nil {
		call.chRefID = ctx.ChRef.ID
		call.chRefNPC = ctx.ChRef.NPC
	}
	if ctx.MeRef != nil {
		call.meRefID = ctx.MeRef.ID
		call.meRefNPC = ctx.MeRef.NPC
	}
	*r.calls = append(*r.calls, call)
	return true, nil
}

func TestFireMobScriptsShareDispatchContext(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Script Room"}},
		Mobs: []parser.Mob{{
			VNum:         9001,
			ShortDesc:    "a script mob",
			ScriptName:   "mob.lua",
			LuaFunctions: 32 | 256, // MS_DEATH | MS_FIGHTING
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	actor := NewPlayer(1, "Target", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	calls := make([]mobScriptCall, 0, 2)
	previousEngine := ScriptEngine
	ScriptEngine = mobScriptRecorder{calls: &calls}
	t.Cleanup(func() { ScriptEngine = previousEngine })

	w.FireMobFightScript(mob, actor, 1001)
	w.FireMobDeathScript(mob, actor, 1001)

	want := []mobScriptCall{
		{filename: "mob.lua", trigger: "fight", actor: "Target", chRefID: actor.ID, meRefID: mob.GetID(), meRefNPC: true, roomVNum: 1001, worldSet: true},
		{filename: "mob.lua", trigger: "death", actor: "Target", chRefID: actor.ID, meRefID: mob.GetID(), meRefNPC: true, roomVNum: 1001, worldSet: true},
	}
	if len(calls) != len(want) {
		t.Fatalf("script calls = %+v, want %+v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("script call %d = %+v, want %+v", i, calls[i], want[i])
		}
	}
}
