package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newMorticianTestWorld(t *testing.T) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()

	w, actor, _ := newSpecProcTestWorld(t)
	peer := NewPlayer(2, "Peer", actor.GetRoomVNum())
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	mob := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 10)
	mob.Prototype.ShortDesc = "the Mortician"
	mob.Prototype.Keywords = "mortician undertaker"
	clearMorticianMessages(messages)
	return w, actor, peer, mob, messages
}

func clearMorticianMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func newMorticianObject(t *testing.T, w *World, vnum, room, typeFlag int, keywords string, values [4]int) *ObjectInstance {
	t.Helper()

	proto := &parser.Obj{
		VNum:      vnum,
		Keywords:  keywords,
		ShortDesc: "a test object",
		LongDesc:  "A test object is here.",
		TypeFlag:  typeFlag,
		Values:    values,
	}
	w.mu.Lock()
	w.objs[vnum] = proto
	w.mu.Unlock()
	obj := w.newObjectInstance(proto, room)
	w.AddItemToRoom(obj, room)
	return obj
}

func TestSpecMortician_CommandSurfaceAndTellBranches(t *testing.T) {
	w, actor, peer, mob, messages := newMorticianTestWorld(t)
	cost := actor.GetLevel() * 116

	if got := specMortician(w, actor, mob, "", ""); got {
		t.Fatal("commandless mortician invocation was handled")
	}
	if got := specMortician(w, actor, mob, "look", ""); got {
		t.Fatal("unknown mortician command was handled")
	}
	if got := messages[actor.Name]; got != "" {
		t.Fatalf("entry/fallthrough output = %q, want empty", got)
	}

	if got := specMortician(w, actor, mob, "list", "ignored"); !got {
		t.Fatal("list was not handled")
	}
	if got, want := messages[actor.Name], "The Mortician tells you, 'It will cost 580 coins to retrieve your corpse.'\r\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("list leaked to peer: %q", got)
	}
	clearMorticianMessages(messages)

	actor.SetGold(cost - 1)
	if got := specMortician(w, actor, mob, "retrieve", "ignored"); !got {
		t.Fatal("unaffordable retrieve was not handled")
	}
	if got, want := messages[actor.Name], "The Mortician tells you, 'I'm sorry, you can't afford the cost.'\r\n"; got != want {
		t.Fatalf("unaffordable output = %q, want %q", got, want)
	}
	if got := actor.GetGold(); got != cost-1 {
		t.Fatalf("unaffordable gold = %d, want %d", got, cost-1)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("unaffordable retrieve leaked to peer: %q", got)
	}
}

func TestSpecMortician_RetrievePredicateGlobalOrderAndState(t *testing.T) {
	w, actor, peer, mob, messages := newMorticianTestWorld(t)
	cost := actor.GetLevel() * 116
	actor.SetGold(cost * 2)

	// C scans the global object_list, not only the mortician's room. The
	// newest qualifying object is first because object creation prepends.
	invalidType := newMorticianObject(t, w, 4001, 1002, ITEM_OTHER, "morticianactor corpse", [4]int{0, 0, 0, 1})
	invalidValue := newMorticianObject(t, w, 4002, 1002, ITEM_CONTAINER, "morticianactor box", [4]int{0, 0, 0, 0})
	oldest := newMorticianObject(t, w, 4003, 1002, ITEM_CONTAINER, "tester corpse", [4]int{0, 0, 0, 1})
	newest := newMorticianObject(t, w, 4004, 1002, ITEM_CONTAINER, "tester corpse", [4]int{0, 0, 0, 1})
	nearby := newMorticianObject(t, w, 4005, actor.GetRoomVNum(), ITEM_OTHER, "nearby object", [4]int{})

	if got := specMortician(w, actor, mob, "retrieve", "not-used"); !got {
		t.Fatal("retrieve was not handled")
	}
	if got, want := messages[actor.Name], "The Mortician dumps your corpse on the ground.\r\n"; got != want {
		t.Fatalf("actor retrieve output = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], "The Mortician dumps Tester's corpse on the ground.\r\n"; got != want {
		t.Fatalf("peer retrieve output = %q, want %q", got, want)
	}
	if got := actor.GetGold(); got != cost {
		t.Fatalf("gold after retrieve = %d, want %d", got, cost)
	}
	if got := newest.GetRoomVNum(); got != actor.GetRoomVNum() {
		t.Fatalf("newest corpse room = %d, want %d", got, actor.GetRoomVNum())
	}
	if got := oldest.GetRoomVNum(); got != 1002 {
		t.Fatalf("older corpse room = %d, want 1002", got)
	}
	if got := invalidType.GetRoomVNum(); got != 1002 {
		t.Fatalf("non-container room = %d, want 1002", got)
	}
	if got := invalidValue.GetRoomVNum(); got != 1002 {
		t.Fatalf("val[3]-clear room = %d, want 1002", got)
	}
	items := w.GetItemsInRoom(actor.GetRoomVNum())
	if len(items) < 2 || items[0] != newest || items[1] != nearby {
		t.Fatalf("destination room order = %#v, want newest corpse prepended before nearby object", items)
	}

	// The C procedure does not consume or alter a retrieved corpse. Clear the
	// predicate explicitly so the next call exercises its no-match branch.
	oldest.SetValue(3, 0)
	newest.SetValue(3, 0)
	clearMorticianMessages(messages)

	// Restore enough gold so the second call reaches the no-match tell instead
	// of the affordability gate.
	actor.SetGold(cost)
	if got := specMortician(w, actor, mob, "retrieve", ""); !got {
		t.Fatal("missing-corpse retrieve was not handled")
	}
	if got, want := messages[actor.Name], "The Mortician tells you, 'I'm sorry, I can't find your corpse anywhere!'\r\n"; got != want {
		t.Fatalf("missing-corpse output = %q, want %q", got, want)
	}
	if got := actor.GetGold(); got != cost {
		t.Fatalf("gold after missing retrieve = %d, want %d", got, cost)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("missing-corpse retrieve leaked to peer: %q", got)
	}
}

func TestSpecMortician_SleepingMobStillHandlesCommand(t *testing.T) {
	w, actor, _, mob, messages := newMorticianTestWorld(t)
	mob.SetPosition(combat.PosSleeping)

	if got := specMortician(w, actor, mob, "list", ""); !got {
		t.Fatal("C command path should not gate on mortician position")
	}
	if got := messages[actor.Name]; got == "" {
		t.Fatal("sleeping mortician did not send list tell")
	}
}
