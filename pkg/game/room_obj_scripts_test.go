package game

// room_obj_scripts_test.go — R5h unit proofs for the room/object Lua trigger
// sites ported in DP-1360. The recorder captures the context shape
// (owner type and C-level refs), so each test fails when a site reverts to
// the pre-bridge context or stops firing.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// roomObjScriptRecorder records filename:trigger plus the parts of the
// ScriptContext DP-1360 added: the owner type and the ch/me/obj refs C
// passes to run_script.
type roomObjScriptRecorder struct {
	calls *[]string
}

func (roomObjScriptRecorder) ForgetFailures() {}

func (r roomObjScriptRecorder) RunScript(ctx *ScriptContext, filename, trigger string) (bool, error) {
	var b strings.Builder
	b.WriteString(filename)
	b.WriteString(":")
	b.WriteString(trigger)
	if ctx.OwnerType != "" {
		b.WriteString(":owner=")
		b.WriteString(ctx.OwnerType)
	}
	if ctx.ChRef != nil {
		b.WriteString(":chref")
	}
	if ctx.MeRef != nil {
		b.WriteString(":meref")
	}
	if ctx.ObjRef != nil {
		b.WriteString(":objref")
	}
	if ctx.RoomVNum != 0 {
		b.WriteString(":room=")
		b.WriteString(strconv.Itoa(ctx.RoomVNum))
	}
	*r.calls = append(*r.calls, b.String())
	return true, nil
}

func installRoomObjRecorder(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	prev := ScriptEngine
	ScriptEngine = roomObjScriptRecorder{calls: calls}
	t.Cleanup(func() { ScriptEngine = prev })
	return calls
}

// scriptedRoomWorld builds a one-room world whose room carries the given
// script flags, plus a player standing in it.
func scriptedRoomWorld(t *testing.T, scriptFuncs int) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Scripted", ScriptName: "room.lua", ScriptFunctions: scriptFuncs},
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
	return w, ch
}

// TestRoomEnterScriptFiresOnce proves the enter site fires exactly once per
// successful move into an RS_ENTER room, with C's context: me is the mover
// and the run is room-owner typed (act.movement.c:304-305).
func TestRoomEnterScriptFiresOnce(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Origin", Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		{VNum: 1002, Name: "Entry", ScriptName: "room.lua", ScriptFunctions: 1 << 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Mover", 1001)
	ch.SetMove(100)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	calls := installRoomObjRecorder(t)

	if !w.DoMove(ch, "north").Success {
		t.Fatal("DoMove failed")
	}
	want := "room.lua:enter:owner=room:chref:meref:room=1002"
	if got := *calls; len(got) != 1 || got[0] != want {
		t.Fatalf("enter calls = %v, want [%s]", got, want)
	}
	if ch.GetRoomVNum() != 1002 {
		t.Fatalf("mover room = %d, want 1002", ch.GetRoomVNum())
	}
}

// TestRoomOnDropFiresOnlyOnDrop proves ondrop runs on the drop path only —
// junk and donate vanish the object down their own arms and never reach C's
// SCMD_DROP trigger site (act.item.c:505-506).
func TestRoomOnDropFiresOnlyOnDrop(t *testing.T) {
	seat := func(t *testing.T) (*World, *Player) {
		w, ch := scriptedRoomWorld(t, 1<<3) // RS_ONDROP
		gem := newTransferItem(7100, "a gem", "gem", 1)
		gem.SetWeight(0)
		registerTransferObject(w, gem)
		if err := w.MoveObjectToPlayerInventory(gem, ch); err != nil {
			t.Fatalf("seat gem: %v", err)
		}
		return w, ch
	}

	t.Run("drop-fires", func(t *testing.T) {
		w, ch := seat(t)
		calls := installRoomObjRecorder(t)
		w.DoDrop(ch, "gem")
		if got := *calls; len(got) != 1 || !strings.HasPrefix(got[0], "room.lua:ondrop:owner=room:chref:meref:objref:") {
			t.Fatalf("ondrop calls = %v", got)
		}
	})
	t.Run("junk-silent", func(t *testing.T) {
		w, ch := seat(t)
		calls := installRoomObjRecorder(t)
		w.DoJunk(ch, "gem")
		if got := *calls; len(got) != 0 {
			t.Fatalf("junk fired room scripts: %v", got)
		}
	})
	t.Run("donate-silent", func(t *testing.T) {
		w, ch := seat(t)
		calls := installRoomObjRecorder(t)
		w.DoDonate(ch, "gem")
		if got := *calls; len(got) != 0 {
			t.Fatalf("donate fired room scripts: %v", got)
		}
	})
}

// TestRoomOnGetFiresFromRoomAndContainer proves onget fires once per gotten
// object, both from the room floor and from a container
// (act.item.c:211-212 and 283-284).
func TestRoomOnGetFiresFromRoomAndContainer(t *testing.T) {
	t.Run("from-room", func(t *testing.T) {
		w, ch := scriptedRoomWorld(t, 1<<4) // RS_ONGET
		gem := newTransferItem(7100, "a gem", "gem", 1)
		gem.SetWeight(0)
		registerTransferObject(w, gem)
		if err := w.MoveObjectToRoom(gem, 1001); err != nil {
			t.Fatalf("floor gem: %v", err)
		}
		calls := installRoomObjRecorder(t)
		w.DoGet(ch, "gem")
		if got := *calls; len(got) != 1 || !strings.HasPrefix(got[0], "room.lua:onget:owner=room:chref:meref:objref:") {
			t.Fatalf("onget calls = %v", got)
		}
	})
	t.Run("from-container", func(t *testing.T) {
		w, ch := scriptedRoomWorld(t, 1<<4) // RS_ONGET
		cont := NewObjectInstance(&parser.Obj{
			VNum: 6400, ShortDesc: "a leather sack", Keywords: "sack",
			TypeFlag: ITEM_CONTAINER, Values: [4]int{100, 0, -1, 0}, WearFlags: [4]int{1},
		}, -1)
		registerTransferObject(w, cont)
		if err := w.MoveObjectToPlayerInventory(cont, ch); err != nil {
			t.Fatalf("seat sack: %v", err)
		}
		gem := newTransferItem(7101, "a gem", "gem", 1)
		gem.SetWeight(0)
		registerTransferObject(w, gem)
		if err := w.MoveObjectToContainer(gem, cont); err != nil {
			t.Fatalf("contain gem: %v", err)
		}
		calls := installRoomObjRecorder(t)
		w.DoGet(ch, "gem sack")
		if got := *calls; len(got) != 1 || !strings.HasPrefix(got[0], "room.lua:onget:owner=room:chref:meref:objref:") {
			t.Fatalf("onget calls = %v", got)
		}
	})
}

// pulseOrderRecorder is roomObjScriptRecorder plus the player HP snapshot
// that proves the onpulse-before-damage ordering.
type pulseOrderRecorder struct {
	calls      *[]string
	hpAtScript *int
	player     *Player
}

func (pulseOrderRecorder) ForgetFailures() {}

func (r pulseOrderRecorder) RunScript(ctx *ScriptContext, filename, trigger string) (bool, error) {
	*r.calls = append(*r.calls, filename+":"+trigger+":owner="+ctx.OwnerType)
	*r.hpAtScript = r.player.GetHP()
	return true, nil
}

// TestRoomOnPulseRunsBeforeFlamingDamage proves the RS_ONPULSE script is the
// first arm in room_activity's per-character body: when it runs, the player's
// HP has not yet been touched by the AFF_FLAMING damage that follows
// (comm.c:706-715).
func TestRoomOnPulseRunsBeforeFlamingDamage(t *testing.T) {
	w, ch := scriptedRoomWorld(t, 1<<2) // RS_ONPULSE
	ch.AddAffect(&engine.Affect{SpellID: 96, Duration: -1, Flags: engine.AFFFlaming})
	if !ch.IsAffected(affFlaming) {
		t.Fatal("player lacks AFF_FLAMING")
	}

	calls := &[]string{}
	hpAtScript := -1
	prev := ScriptEngine
	ScriptEngine = pulseOrderRecorder{calls: calls, hpAtScript: &hpAtScript, player: ch}
	t.Cleanup(func() { ScriptEngine = prev })

	w.RoomActivity()

	if hpAtScript != ch.MaxHealth {
		t.Fatalf("onpulse saw HP %d, want %d (script must run before the flaming damage)", hpAtScript, ch.MaxHealth)
	}
	if got := ch.GetHP(); got >= ch.MaxHealth {
		t.Fatalf("flaming damage did not land after the script: HP = %d", got)
	}
	if got := *calls; len(got) != 1 || got[0] != "room.lua:onpulse:owner=room" {
		t.Fatalf("onpulse calls = %v", got)
	}
}

// TestObjectActivityRunsOSOnPulseScripts proves object_activity's script arm:
// every live object with a prototype carrying OS_ONPULSE runs onpulse each
// pass, drink containers/fountains/corpses are skipped, the flag gate holds,
// and the room is the carrier's room or the object's own room
// (comm.c:789-793).
func TestObjectActivityRunsOSOnPulseScripts(t *testing.T) {
	w, ch := scriptedRoomWorld(t, 0)

	mkObj := func(vnum int, scriptFuncs int, typ int) *ObjectInstance {
		proto := &parser.Obj{VNum: vnum, Keywords: "thing", ShortDesc: "a thing", TypeFlag: typ, ScriptName: "obj.lua", LuaFunctions: scriptFuncs}
		obj := NewObjectInstance(proto, -1)
		registerTransferObject(w, obj)
		return obj
	}

	floor := mkObj(3001, 1<<2, ITEM_OTHER)       // in a room
	carried := mkObj(3002, 1<<2, ITEM_OTHER)     // carried by the player
	drinkcon := mkObj(3003, 1<<2, ITEM_DRINKCON) // skipped: drink container
	corpse := mkObj(3004, 1<<2, ITEM_CONTAINER)  // skipped: corpse (val3=1)
	corpse.Prototype.Values[3] = 1
	unflagged := mkObj(3005, 0, ITEM_OTHER) // no OS_ONPULSE bit

	for _, obj := range []*ObjectInstance{floor, drinkcon, corpse, unflagged} {
		if err := w.MoveObjectToRoom(obj, 1001); err != nil {
			t.Fatalf("floor %d: %v", obj.VNum, err)
		}
	}
	if err := w.MoveObjectToPlayerInventory(carried, ch); err != nil {
		t.Fatalf("carry obj: %v", err)
	}

	calls := installRoomObjRecorder(t)
	w.ObjectActivity()

	want := []string{
		"obj.lua:onpulse:owner=obj:objref:room=1001", // floor object, its own room
		"obj.lua:onpulse:owner=obj:objref:room=1001", // carried object, carrier's room
	}
	if len(*calls) != len(want) {
		t.Fatalf("onpulse calls = %v, want %v", *calls, want)
	}
	for i := range want {
		if (*calls)[i] != want[i] {
			t.Fatalf("onpulse call %d = %q, want %q", i, (*calls)[i], want[i])
		}
	}
}
