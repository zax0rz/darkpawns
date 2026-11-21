package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// buildTurnUndeadWorld constructs the two-room 19875/19876 world the special
// gates on (src/spec_procs3.c:546-547 real_room), with an actor in 19875, a
// room peer who receives the act line, and an off-room player who must not.
func buildTurnUndeadWorld(t *testing.T) (*World, *Player, *Player, *ObjectInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 19875, Name: "North Chapel", Zone: 1},
		{VNum: 19876, Name: "South Chapel", Zone: 1},
		{VNum: 1204, Name: "Elsewhere", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] += string(msg)
	}

	actor := NewPlayer(1, "Turner", 19875)
	roomPeer := NewPlayer(2, "Witness", 19875)
	offRoomPeer := NewPlayer(3, "Away", 1204)
	for _, player := range []*Player{actor, roomPeer, offRoomPeer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}

	wand := NewObjectInstance(&parser.Obj{
		VNum:      19877,
		Keywords:  "turn undead wand",
		ShortDesc: "a strange wand",
	}, -1)

	return w, actor, roomPeer, wand, messages
}

// TestSpecTurnUndeadObjectGatesAndAct pins SPECIAL(turn_undead)
// (src/spec_procs3.c:543-581) through its object-aware handler: the isname
// keyword gate ( :553), the act line with $p = OBJS(obj) — the object's short
// description ( :555-556), and the north/south exits it creates ( :558-563).
// Before the fix the port hardcoded "the object" and skipped the name gate
// entirely on the nil-mob adapter; each case below fails against that body.
func TestSpecTurnUndeadObjectGatesAndAct(t *testing.T) {
	w, actor, roomPeer, wand, messages := buildTurnUndeadWorld(t)

	// Wrong keyword: isname("torch", "turn undead wand") fails; C returns
	// FALSE with no bytes and no exits (src/spec_procs3.c:553-565 fall-through).
	if got := specTurnUndeadObject(w, actor, wand, "use", "torch"); got {
		t.Fatal("non-matching keyword was handled")
	}
	if messages[roomPeer.Name] != "" {
		t.Errorf("room output on keyword miss = %q, want none", messages[roomPeer.Name])
	}
	if room := w.GetRoomInWorld(19875); room != nil {
		if _, ok := room.Exits["north"]; ok {
			t.Error("north exit created on keyword miss")
		}
	}

	// Right keyword in the right room: act line names the SHORT DESCRIPTION
	// ($p -> OBJS, src/spec_procs3.c:555), and both exits are created.
	if got := specTurnUndeadObject(w, actor, wand, "use", "wand"); !got {
		t.Fatal("matching use was not handled")
	}
	if got, want := messages[roomPeer.Name], "A ray of flame bursts out of a strange wand, consuming the undead!\r\n"; got != want {
		t.Errorf("room act line = %q, want %q", got, want)
	}
	if got := messages[actor.Name]; got != "" {
		t.Errorf("actor received the TO_ROOM line: %q", got)
	}
	roomA := w.GetRoomInWorld(19875)
	roomB := w.GetRoomInWorld(19876)
	if roomA == nil || roomB == nil {
		t.Fatal("19875/19876 missing from world")
	}
	if exit, ok := roomA.Exits["north"]; !ok || exit.ToRoom != 19876 {
		t.Errorf("north exit = %+v (present %v), want ToRoom 19876", exit, ok)
	}
	if exit, ok := roomB.Exits["south"]; !ok || exit.ToRoom != 19875 {
		t.Errorf("south exit = %+v (present %v), want ToRoom 19875", exit, ok)
	}
}

// TestSpecTurnUndeadObjectRoomGate pins that only rooms 19875/19876 arm the
// special (src/spec_procs3.c:549-550): the same object and keyword elsewhere
// returns FALSE and prints nothing.
func TestSpecTurnUndeadObjectRoomGate(t *testing.T) {
	w, _, roomPeer, wand, messages := buildTurnUndeadWorld(t)

	away := NewPlayer(4, "Faraway", 1204)
	if err := w.AddPlayer(away); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if got := specTurnUndeadObject(w, away, wand, "use", "wand"); got {
		t.Fatal("use outside 19875/19876 was handled")
	}
	if messages[roomPeer.Name] != "" {
		t.Errorf("room output from a gated room = %q, want none", messages[roomPeer.Name])
	}
}

// TestSpecTurnUndeadObjectPulseReset pins the !cmd arm: both exits are freed
// and the special declines so other pulse work continues
// (src/spec_procs3.c:567-580).
func TestSpecTurnUndeadObjectPulseReset(t *testing.T) {
	w, actor, _, wand, _ := buildTurnUndeadWorld(t)

	if got := specTurnUndeadObject(w, actor, wand, "use", "wand"); !got {
		t.Fatal("matching use was not handled")
	}
	if got := specTurnUndeadObject(w, actor, wand, "", ""); got {
		t.Fatal("pulse arm must return FALSE (src/spec_procs3.c:580)")
	}
	roomA := w.GetRoomInWorld(19875)
	roomB := w.GetRoomInWorld(19876)
	if _, ok := roomA.Exits["north"]; ok {
		t.Error("pulse arm left the north exit in place")
	}
	if _, ok := roomB.Exits["south"]; ok {
		t.Error("pulse arm left the south exit in place")
	}

	// C's FREE/NULL arms are idempotent (src/spec_procs3.c:569-578): a second
	// pulse with no exits must not fail or redraw anything.
	if got := specTurnUndeadObject(w, actor, wand, "", ""); got {
		t.Fatal("second pulse arm call returned TRUE")
	}
}

// TestSpecTurnUndeadObjectRegistered pins the registration contract: the
// object-aware handler is registered in ObjSpecRegistry under the spec name
// (the moon_gate pattern), so any future ObjSpecAssign entry resolves it
// through GetObjSpecForObject, and the mob lane stays inert.
func TestSpecTurnUndeadObjectRegistered(t *testing.T) {
	if _, ok := ObjSpecRegistry["turn_undead"]; !ok {
		t.Fatal("turn_undead missing from ObjSpecRegistry")
	}
	if fn := GetMobSpec(19877); fn != nil {
		t.Fatal("turn_undead must not resolve a mob-lane spec")
	}
	if name, ok := GetObjSpecName(19877); ok && name == "turn_undead" {
		t.Fatal("turn_undead must stay unassigned: no ObjSpecAssign entry (the brief forbids assigning it)")
	}
}
