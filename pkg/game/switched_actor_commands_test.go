package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

type actorRoomScriptCall struct {
	trigger  string
	owner    string
	roomVNum int
	chNPC    bool
	chID     int
	meNPC    bool
	meID     int
	objID    int
}

type actorRoomScriptRecorder struct {
	calls *[]actorRoomScriptCall
}

func (actorRoomScriptRecorder) ForgetFailures() {}

func (r actorRoomScriptRecorder) RunScript(ctx *ScriptContext, _ string, trigger string) (bool, error) {
	call := actorRoomScriptCall{trigger: trigger, owner: ctx.OwnerType, roomVNum: ctx.RoomVNum}
	if ctx.ChRef != nil {
		call.chNPC, call.chID = ctx.ChRef.NPC, ctx.ChRef.ID
	}
	if ctx.MeRef != nil {
		call.meNPC, call.meID = ctx.MeRef.NPC, ctx.MeRef.ID
	}
	if ctx.ObjRef != nil {
		call.objID = ctx.ObjRef.ID
	}
	*r.calls = append(*r.calls, call)
	return true, nil
}

func TestSwitchedGetDropRoomScriptsReceiveNPCBody(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Scripted", ScriptName: "room.lua", ScriptFunctions: rsOnGet | rsOnDrop}},
		Mobs:  []parser.Mob{{VNum: 9001, Keywords: "watcher", ShortDesc: "a watcher", Level: 10}},
		Objs: []parser.Obj{{
			VNum: 7100, ShortDesc: "a gem", Keywords: "gem", TypeFlag: ITEM_OTHER,
			Weight: 0, WearFlags: [4]int{1},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	mob, err := w.SpawnMobQuiet(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	obj, err := w.SpawnObject(7100, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToRoom(obj, 1001); err != nil {
		t.Fatalf("MoveObjectToRoom: %v", err)
	}
	calls := &[]actorRoomScriptCall{}
	previousEngine := ScriptEngine
	ScriptEngine = actorRoomScriptRecorder{calls: calls}
	t.Cleanup(func() { ScriptEngine = previousEngine })

	var output strings.Builder
	ctx := CharacterCommandContext{Actor: mob, World: w, RawArgs: "gem", PrivateOutput: func(line string) { output.WriteString(line) }}
	if got := CharacterGet(ctx); got != CharacterCommandHandled {
		t.Fatalf("CharacterGet result = %v", got)
	}
	if obj.Location.Kind != ObjInInventory || obj.Location.OwnerKind != OwnerMob || obj.Location.MobID != mob.GetID() {
		t.Fatalf("get did not move item to mob: output=%q location=%+v room-items=%d", output.String(), obj.Location, len(w.GetItemsInRoom(1001)))
	}
	if got := CharacterDrop(ctx); got != CharacterCommandHandled {
		t.Fatalf("CharacterDrop result = %v", got)
	}
	want := []actorRoomScriptCall{
		{trigger: "onget", owner: "room", roomVNum: 1001, chNPC: true, chID: mob.GetID(), meNPC: true, meID: mob.GetID(), objID: obj.ID},
		{trigger: "ondrop", owner: "room", roomVNum: 1001, chNPC: true, chID: mob.GetID(), meNPC: true, meID: mob.GetID(), objID: obj.ID},
	}
	if len(*calls) != len(want) {
		t.Fatalf("room script calls = %+v, want %+v", *calls, want)
	}
	for i := range want {
		if (*calls)[i] != want[i] {
			t.Errorf("room script call %d = %+v, want %+v", i, (*calls)[i], want[i])
		}
	}
}

func TestMobDummyConditionsUseSharedSentinelSlots(t *testing.T) {
	npcDummyConditions.Lock()
	previous := npcDummyConditions.values
	npcDummyConditions.values = [3]int{-1, -1, -1}
	npcDummyConditions.Unlock()
	t.Cleanup(func() {
		npcDummyConditions.Lock()
		npcDummyConditions.values = previous
		npcDummyConditions.Unlock()
	})

	first := NewMob(&parser.Mob{VNum: 9001, Keywords: "first", ShortDesc: "the first"}, 1001)
	second := NewMob(&parser.Mob{VNum: 9002, Keywords: "second", ShortDesc: "the second"}, 1001)
	for condition := range npcDummyConditions.values {
		if got := first.GetCondition(condition); got != -1 {
			t.Fatalf("initial condition %d = %d, want shared no-change sentinel -1", condition, got)
		}
	}
	first.SetCondition(CondFull, 17)
	if got := second.GetCondition(CondFull); got != 17 {
		t.Fatalf("second mob fullness = %d, want shared value 17", got)
	}
	first.SetCondition(CondFull, -1)
	ctx := CharacterCommandContext{PrivateOutput: func(string) {}}
	gainConditionMobile(ctx, first, CondFull, 12)
	if got := second.GetCondition(CondFull); got != -1 {
		t.Fatalf("gainCondition changed shared -1 sentinel to %d", got)
	}
}

func TestMobileConsumablePoisonUsesCSpellAffect(t *testing.T) {
	mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "watcher", ShortDesc: "a watcher"}, 1001)
	applyPoisonMobile(mob, 6, "poisoned drink")
	if !mob.IsAffected(affPoison) {
		t.Fatal("poisoned mobile lacks AFF_POISON")
	}
	affects := mob.OrderedAffects()
	if len(affects) != 1 {
		t.Fatalf("mobile affects = %d, want 1", len(affects))
	}
	if got := affects[0].SpellID; got != spells.SpellPoison {
		t.Fatalf("poison affect spell ID = %d, want C SPELL_POISON %d", got, spells.SpellPoison)
	}
	if got := affects[0].Duration; got != 6 {
		t.Fatalf("poison affect duration = %d, want 6", got)
	}
}

func TestCharacterRetreatChecksBodyPositionBeforeNpcSkill(t *testing.T) {
	mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "watcher", ShortDesc: "a watcher"}, 1001)
	var output strings.Builder
	ctx := CharacterCommandContext{Actor: mob, PrivateOutput: func(line string) { output.WriteString(line) }}

	mob.SetPosition(combat.PosSleeping)
	if got := CharacterRetreat(ctx); got != CharacterCommandHandled {
		t.Fatalf("sleeping retreat result = %v", got)
	}
	if got, want := output.String(), "Get on your feet first!\r\n"; got != want {
		t.Fatalf("sleeping retreat output = %q, want %q", got, want)
	}

	output.Reset()
	mob.SetPosition(combat.PosStanding)
	if got := CharacterRetreat(ctx); got != CharacterCommandHandled {
		t.Fatalf("standing retreat result = %v", got)
	}
	if got, want := output.String(), "Huh?\r\n"; got != want {
		t.Fatalf("standing retreat output = %q, want %q", got, want)
	}
}
