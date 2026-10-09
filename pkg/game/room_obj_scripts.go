package game

// room_obj_scripts.go — port of C's room (LT_ROOM) and object (LT_OBJ) Lua
// script trigger sites. C's run_script(ch, me, obj, room, argument, fname,
// type) builds the script path strictly as scripts/<type>/<script name>
// (scripts.c:1775), so every run here is owner-typed: room triggers load only
// scripts/room/<name> and object triggers only scripts/obj/<name>. C passes
// me = ch for every room and object trigger except object onpulse, whose
// ch and me are both NULL (comm.c:789-793).

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// Room script trigger flags (src/structs.h:675-680).
const (
	rsEnter   = 1 << 1 // RS_ENTER
	rsOnPulse = 1 << 2 // RS_ONPULSE
	rsOnDrop  = 1 << 3 // RS_ONDROP
	rsOnGet   = 1 << 4 // RS_ONGET
	rsOnCmd   = 1 << 5 // RS_ONCMD
)

// Object script trigger flags (src/structs.h:683-685).
const (
	osOnCmd   = 1 << 1 // OS_ONCMD
	osOnPulse = 1 << 2 // OS_ONPULSE
)

// objHasScriptTrigger is C's GET_OBJ_RNUM(obj) != NOTHING &&
// GET_OBJ_SCRIPT(obj) && OBJ_SCRIPT_FLAGGED gate: the object has a prototype
// whose script carries the trigger bit. C never tests the name here, so a
// cleared name with the flag kept still passes (the owner's empty-name arm
// supplies the producer).
func objHasScriptTrigger(obj *ObjectInstance, bit int) bool {
	return obj != nil && obj.Prototype != nil && obj.Prototype.LuaFunctions&bit != 0
}

// objScriptActorPayload is GET_NAME(me) and GET_MOB_VNUM(me) for the object
// oncmd unassigned-script producer. A switched mobile's session (actorRef.NPC)
// acts as its mobile, whose short description and prototype vnum are the
// mobile's; a plain player actor yields its name and -1 (GET_MOB_VNUM is -1
// for a PC, src/utils.h:431-432). The room-contents arm is the only object
// oncmd site without an IS_NPC gate (interpreter.c:1470).
func (w *World) objScriptActorPayload(actorRef *scripting.CharRef, actorPlayer *Player) (string, int) {
	if actorRef != nil && actorRef.NPC {
		if m, ok := w.GetMobByID(actorRef.ID); ok && m != nil {
			return m.GetName(), m.GetVNum()
		}
	}
	if actorPlayer != nil {
		return actorPlayer.GetName(), -1
	}
	// Unreachable: every object-oncmd caller supplies an actor ref or player.
	return "", -1
}

// RunRoomOnCmdScript is the room oncmd arm of C's special()
// (interpreter.c:1419-1423): run_script(ch, ch, NULL, &world[ch->in_room],
// "CMD_NAME arg", "oncmd", LT_ROOM), gated on !IS_NPC(ch). A TRUE return
// consumes the command.
func (w *World) RunRoomOnCmdScript(ch *Player, argument string) bool {
	return w.runRoomScript(ch, nil, rsOnCmd, "oncmd", argument)
}

// RunRoomEnterScript is C's room enter trigger (act.movement.c:304-305):
// run_script(ch, ch, NULL, &world[ch->in_room], NULL, "enter", LT_ROOM) after
// a successful move. The return value is ignored in C.
func (w *World) RunRoomEnterScript(ch *Player) {
	w.runRoomScript(ch, nil, rsEnter, "enter", "")
}

// RunRoomPulseScript is C's room onpulse trigger in room_activity
// (comm.c:706-708): run_script(ch, ch, NULL, &world[ch->in_room], NULL,
// "onpulse", LT_ROOM), first in the per-character body, PC-only and gated on
// RS_ONPULSE. The return value is ignored in C. The script can draw number(),
// so its position in the pulse is draw-order-sensitive (R3).
func (w *World) RunRoomPulseScript(ch *Player) {
	w.runRoomScript(ch, nil, rsOnPulse, "onpulse", "")
}

// RunRoomObjTriggerScript is the room onget/ondrop family
// (act.item.c:211-212, 283-284, 505-506): run_script(ch, ch, obj,
// &world[ch->in_room], NULL, trigger, LT_ROOM). The return value is ignored
// in C.
func (w *World) RunRoomObjTriggerScript(ch *Player, obj *ObjectInstance, bit int, trigger string) {
	w.runRoomScript(ch, obj, bit, trigger, "")
}

// runRoomScript runs one room-owner trigger with C's tables: me is the acting
// player, as C passes it for every live room trigger.
func (w *World) runRoomScript(ch *Player, obj *ObjectInstance, bit int, trigger, argument string) bool {
	if ScriptEngine == nil {
		return false
	}
	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || room.ScriptFunctions&bit == 0 {
		return false
	}
	if room.ScriptName == "" {
		// run_script's !*script_name arm for a room owner
		// (src/scripts.c:1752-1767). C passes me = ch for every room trigger
		// (interpreter.c:1421, act.item.c:212, :284, :506, act.movement.c:305,
		// comm.c:708), so the payload names the acting player and
		// GET_MOB_VNUM is -1.
		scriptUnassignedProducer(ch.GetName(), -1)
		return true
	}
	ctx := &ScriptContext{
		Ch:        ch,
		Obj:       obj,
		RoomVNum:  room.VNum,
		Argument:  argument,
		World:     NewWorldScriptableAdapter(w),
		ChRef:     &scripting.CharRef{ID: ch.ID},
		MeRef:     &scripting.CharRef{ID: ch.ID},
		OwnerType: "room",
	}
	if obj != nil {
		ctx.ObjRef = &scripting.ObjRef{ID: obj.ID}
	}
	handled, err := ScriptEngine.RunScript(ctx, room.ScriptName, trigger)
	if err != nil {
		slog.Warn("room script error", "trigger", trigger, "room_vnum", room.VNum, "script", room.ScriptName, "error", err)
	}
	return handled
}

// RunObjOnCmdScript is C's object oncmd arm (interpreter.c:1430-1436 worn,
// 1443-1448 carried, 1470-1476 room contents): run_script(ch, ch, obj,
// &world[ch->in_room], "CMD_NAME arg", "oncmd", LT_OBJ). actorPlayer is nil
// when the session acts through a switched mobile (only the room-contents
// site runs then — the other two gate on !IS_NPC). A TRUE return consumes
// the command.
func (w *World) RunObjOnCmdScript(actorRef *scripting.CharRef, actorPlayer *Player, obj *ObjectInstance, roomVNum int, argument string) bool {
	if ScriptEngine == nil || !objHasScriptTrigger(obj, osOnCmd) {
		return false
	}
	if obj.Prototype.ScriptName == "" {
		// run_script's !*script_name arm for an object owner
		// (src/scripts.c:1756-1767). C passes me = ch; the actor is a player
		// for the worn and carried arms (interpreter.c:1430, :1443, both
		// gated !IS_NPC) and may be a switched mobile at the room-contents
		// arm (interpreter.c:1470, no IS_NPC gate).
		name, vnum := w.objScriptActorPayload(actorRef, actorPlayer)
		scriptUnassignedProducer(name, vnum)
		return true
	}
	ctx := &ScriptContext{
		Ch:        actorPlayer,
		Obj:       obj,
		RoomVNum:  roomVNum,
		Argument:  argument,
		World:     NewWorldScriptableAdapter(w),
		ChRef:     actorRef,
		MeRef:     actorRef,
		ObjRef:    &scripting.ObjRef{ID: obj.ID},
		OwnerType: "obj",
	}
	handled, err := ScriptEngine.RunScript(ctx, obj.Prototype.ScriptName, "oncmd")
	if err != nil {
		slog.Warn("object oncmd script error", "obj_vnum", obj.VNum, "script", obj.Prototype.ScriptName, "error", err)
	}
	return handled
}

// RunObjPulseScript is C's object onpulse arm in object_activity
// (comm.c:789-793): run_script(NULL, NULL, obj, &world[room], NULL,
// "onpulse", LT_OBJ) — ch and me are NULL, and the room is the carrier's
// room or the room the object lies in. The return value is ignored in C.
func (w *World) RunObjPulseScript(obj *ObjectInstance) {
	if ScriptEngine == nil || !objHasScriptTrigger(obj, osOnPulse) {
		return
	}
	if obj.Prototype.ScriptName == "" {
		// DP-1416 approved divergence: C's object-onpulse caller passes both
		// ch and me as NULL (comm.c:789-793), so run_script's empty-name arm
		// formats GET_NAME(NULL) / GET_MOB_VNUM(NULL) and faults
		// (IS_NPC(NULL) / IS_MOB(NULL) dereference, src/utils.h:230-231) before
		// it emits a byte. Go cannot reproduce the fault, so it emits nothing;
		// C ignores this trigger's return value too, so there is no consume
		// difference to keep.
		return
	}
	// C indexes world[obj->carried_by->in_room] when carried and
	// world[obj->in_room] otherwise; a worn or contained object has
	// in_room NOWHERE and the read is undefined (R1a-adjacent; no live
	// object carries OS_ONPULSE). Those run without a room global.
	roomVNum := objPulseRoomVNum(w, obj)
	ctx := &ScriptContext{
		Obj:       obj,
		RoomVNum:  roomVNum,
		World:     NewWorldScriptableAdapter(w),
		ObjRef:    &scripting.ObjRef{ID: obj.ID},
		OwnerType: "obj",
	}
	if _, err := ScriptEngine.RunScript(ctx, obj.Prototype.ScriptName, "onpulse"); err != nil {
		slog.Warn("object onpulse script error", "obj_vnum", obj.VNum, "script", obj.Prototype.ScriptName, "error", err)
	}
}

// ObjectActivity ports C object_activity()'s per-object body (comm.c:758-797):
// every live object with a prototype runs its OS_ONPULSE script each
// PULSE_MOBILE, after mobile_activity and room_activity (comm.c:815-820).
// Drink containers, fountains and corpses are skipped before the script
// check exactly as in C. C's weight-mismatch SYSERR arm before the script is
// a defensive invariant that only fires on corrupt object weight and is not
// ported. C walks the object_list prepend order; the port iterates live
// instances by ascending ID for determinism (no live object uses OS_ONPULSE).
func (w *World) ObjectActivity() {
	objs := w.liveObjectsByID()
	for _, obj := range objs {
		if obj == nil || obj.Prototype == nil { // GET_OBJ_RNUM(obj) != NOTHING
			continue
		}
		// comm.c:770-771 — drink containers, fountains and corpses never
		// reach the weight check or the script block.
		if t := obj.GetTypeFlag(); t == ITEM_DRINKCON || t == ITEM_FOUNTAIN ||
			(t == ITEM_CONTAINER && obj.GetValue(3) == 1) {
			continue
		}
		w.repairObjectWeight(obj)
		w.RunObjPulseScript(obj)
	}
}

// repairObjectWeight ports C's weight-mismatch arm in object_activity
// (comm.c:773-787): an object whose instance weight differs from its
// prototype's is reported at BRF/LVL_IMMORT/file TRUE and then reset to the
// prototype's weight, but only while it holds nothing (comm.c:774). Drink
// containers, fountains and corpses never reach here: the caller's skip at
// comm.c:770-771 continues past both this block and the script block.
//
// The location wordings are C's, which are not the obvious mapping.
// C tests obj->carried_by for truth, then obj->in_room, then obj->worn_by, and
// falls back to "unknown" (comm.c:775-782). NOWHERE is -1 (src/structs.h:34),
// which is *truthy*, so only real room 0 takes the fallback:
//
//   - obj_to_char sets in_room = NOWHERE (src/handler.c:565) and equip_char
//     never resets it (src/handler.c:731), so a worn object reaches the
//     in_room test as -1 and prints "in room": C's "worn by" arm is dead code.
//   - an object inside a container, or one not yet placed, is also at -1 and
//     prints "in room".
//   - "unknown" prints only for an uncarried object whose in_room is real room
//     0, the first room loaded ("The Void", lib/world/wld/0.wld).
//   - a shop's stock is in the keeper's inventory in C, so it prints "carried".
//
// Held locks: none. liveObjectsByID snapshots under World.mu and releases it
// before the loop, and each field read is the same lock-free instance access
// the surrounding activity loop already makes.
func (w *World) repairObjectWeight(obj *ObjectInstance) {
	if obj == nil || obj.Prototype == nil {
		return
	}
	if obj.GetWeight() == obj.Prototype.Weight {
		return
	}
	if len(obj.Contains) != 0 {
		return
	}
	location := "in room"
	switch obj.Location.Kind {
	case ObjInInventory, ObjInShop:
		location = "carried"
	case ObjInRoom:
		if obj.Location.RoomVNum == 0 {
			location = "unknown"
		}
	}
	MudLog(fmt.Sprintf("SYSERR: Object '%s' weight incorrect, location '%s'",
		obj.GetShortDesc(), location), MudlogBrief, LVL_IMMORT, true)
	obj.SetWeight(obj.Prototype.Weight)
}

// liveObjectsByID snapshots every live object instance in ascending instance
// ID order.
func (w *World) liveObjectsByID() []*ObjectInstance {
	w.mu.RLock()
	objs := make([]*ObjectInstance, 0, len(w.objectInstances))
	for _, obj := range w.objectInstances {
		objs = append(objs, obj)
	}
	w.mu.RUnlock()
	sort.Slice(objs, func(i, j int) bool { return objs[i].ID < objs[j].ID })
	return objs
}

// objPulseRoomVNum resolves the room C passes for an object's onpulse: the
// carrier's room when carried, otherwise the room it lies in (0 = no room
// global).
func objPulseRoomVNum(w *World, obj *ObjectInstance) int {
	switch obj.Location.Kind {
	case ObjInInventory:
		if obj.Location.OwnerKind == OwnerPlayer {
			if p, ok := w.GetPlayer(obj.Location.PlayerName); ok && p != nil {
				return p.GetRoomVNum()
			}
		} else if m, ok := w.GetMobByID(obj.Location.MobID); ok && m != nil {
			return m.GetRoomVNum()
		}
	case ObjInRoom:
		return obj.Location.RoomVNum
	}
	return 0
}
