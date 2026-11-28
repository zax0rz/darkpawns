package game

// spec_trailing_space_test.go — C strips leading whitespace only
// (skip_spaces, src/interpreter.c:1187), so a trailing space fails every
// special's strcmp/strcasecmp/isname gate ("couch ", "horse ", "door ",
// "itoh "). Go's former strings.TrimSpace accepted them. The interpreter
// tokenizes command lines with strings.Fields, so no trailing space can
// reach these gates in play today; the direct calls below bypass that
// tokenizer to pin the code-level contract, and each fails on origin/main.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestIsCNameTrailingSpaceFailsLikeC pins C isname's no-trim rule
// (src/handler.c:81): a trailing space in the tested name fails the match.
func TestIsCNameTrailingSpaceFailsLikeC(t *testing.T) {
	if isCName("foot ", "foot sword") {
		t.Fatal(`isCName("foot ", ...) matched; C isname fails a trailing space`)
	}
	if isCName(" foot", "foot sword") {
		t.Fatal(`isCName(" foot", ...) matched; C isname trims nothing`)
	}
	if !isCName("foot", "foot sword") {
		t.Fatal(`isCName("foot", ...) did not match`)
	}
}

// TestSkipSpacesGatesRejectTrailingSpace covers the four gates named in the
// cross-cutting audit, plus dracula's abbreviation gate.
func TestSkipSpacesGatesRejectTrailingSpace(t *testing.T) {
	// itoh: say "itoh" teleports to 19875 (src/spec_procs3.c:583-606).
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Origin", Zone: 1},
		{VNum: 19875, Name: "Destination", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	actor := NewPlayer(1, "Itohactor", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if specItoh(w, actor, nil, "say", "itoh ") {
		t.Fatal(`"itoh " teleported; C strcasecmp("itoh ") fails the gate`)
	}
	if got := actor.GetRoomVNum(); got != 1001 {
		t.Fatalf("actor room = %d after trailing-space say, want 1001", got)
	}
	if !specItoh(w, actor, nil, "say", "  itoh") {
		t.Fatal(`"  itoh" did not teleport; C skip_spaces strips leading spaces`)
	}
	if got := actor.GetRoomVNum(); got != 19875 {
		t.Fatalf("actor room = %d after leading-space say, want 19875", got)
	}
}

// TestSpecCouchRejectsTrailingSpace pins the couch strcmp gate
// (src/spec_procs2.c:289-291).
func TestSpecCouchRejectsTrailingSpace(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Pillow Room", Zone: 1},
			{VNum: mimicRoomVnum, Name: "Mimic Home", Zone: 1},
		},
		Objs: []parser.Obj{{VNum: 3002, Keywords: "couch", ShortDesc: "a worn couch"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	actor := NewPlayer(1, "Couchlook", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	couch, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Fatalf("SpawnObject couch: %v", err)
	}
	w.AddItemToRoom(couch, 1001)
	proc := newSpecProcTestMob(t, w, 1001, 10)
	newSpecProcTestMob(t, w, mimicRoomVnum, 10)
	if specCouch(w, actor, proc, "look", "couch ") {
		t.Fatal(`"couch " triggered the mimic; C strcmp("couch ") fails`)
	}
	if got := len(w.GetItemsInRoom(1001)); got != 1 {
		t.Fatalf("room items = %d after trailing-space look, want 1", got)
	}
	if !specCouch(w, actor, proc, "look", "  couch") {
		t.Fatal(`"  couch" did not trigger the mimic; C skip_spaces strips leading`)
	}
	if got := len(w.GetItemsInRoom(1001)); got != 0 {
		t.Fatalf("room items = %d after leading-space look, want 0", got)
	}
}

// TestSpecAlienElevatorRejectsTrailingSpace pins the door strcasecmp gate
// (src/spec_procs3.c:406-407).
func TestSpecAlienElevatorRejectsTrailingSpace(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 19551, Name: "Lower Room", Zone: 1, Exits: map[string]parser.Exit{"east": {Direction: "east", ToRoom: 19599, ExitInfo: parser.ExitIsDoor}}},
		{VNum: 19599, Name: "Upper Room", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	caller := NewPlayer(1, "Caller", 19551)
	if err := w.AddPlayer(caller); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if specAlienElevator(w, caller, nil, "close", "door ") {
		t.Fatal(`"door " moved the room; C strcasecmp("door ") fails`)
	}
	if got := caller.GetRoomVNum(); got != 19551 {
		t.Fatalf("caller room = %d after trailing-space close, want 19551", got)
	}
	if !specAlienElevator(w, caller, nil, "close", "door") {
		t.Fatal(`"door" did not move the room`)
	}
	if got := caller.GetRoomVNum(); got != 19599 {
		t.Fatalf("caller room = %d after exact close, want 19599", got)
	}
}

// TestSpecStableboyRejectsTrailingSpace pins the horse strcmp gate
// (src/spec_procs2.c:323-325): "horse " takes the "Buy what" arm.
func TestSpecStableboyRejectsTrailingSpace(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Stables", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 8022, Keywords: "stableboy", ShortDesc: "the stableboy"},
			{VNum: horseVnum, Keywords: "horse", ShortDesc: "a horse", ActionFlags: []string{"mountable"}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	ch.Stats.Cha = 10
	ch.CopyBaseAttributes()
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	proto, _ := w.GetMobPrototype(8022)
	boy := NewMob(proto, 1001)

	specStableboy(w, ch, boy, "buy", "horse ")
	if msg := out.String(); !strings.Contains(msg, "Buy what") {
		t.Fatalf("buy with trailing-space horse: got %q, want the Buy what arm", msg)
	}
	out.Reset()

	specStableboy(w, ch, boy, "buy", "horse")
	if msg := out.String(); !strings.Contains(msg, "can't afford") {
		t.Fatalf("buy with exact horse: got %q, want the affordability arm", msg)
	}
}

// TestSpecDraculaRejectsTrailingSpace pins dracula's abbreviation gate: C
// is_abbrev fails a trailing space (src/spec_procs.c:1811-1812).
func TestSpecDraculaRejectsTrailingSpace(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	p := *mob.Proto()
	p.Keywords = "Lothar Vampire Lord"
	p.ShortDesc = "Lothar the Vampire Lord"
	mob.SetProto(&p)
	player.Stats.Int = 10
	player.Stats.Wis = 10
	player.CopyBaseAttributes()
	lastMsg()

	if specDracula(w, player, mob, "look", "lothar ") {
		t.Fatal(`"lothar " triggered dracula; C is_abbrev fails a trailing space`)
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("trailing-space output = %q, want empty", got)
	}
	if !specDracula(w, player, mob, "look", "  lothar") {
		t.Fatal(`"  lothar" did not trigger dracula; C skip_spaces strips leading`)
	}
}

// TestBytesEnterCircleEnterWhatTerminator pins "Enter what?\n\r" (LF CR)
// against the former CRLF send (R1b: the oracle normalizer folds terminators,
// so a census cannot see this; src/spec_procs.c:1933).
func TestBytesEnterCircleEnterWhatTerminator(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	if !specEnterCircle(w, player, nil, "enter", "foo") {
		t.Fatal("enter foo was not consumed by enter_circle")
	}
	if got, want := lastMsg(), "Enter what?\n\r"; got != want {
		t.Fatalf("enter what bytes = %q, want %q", got, want)
	}
	// "circle " fails the strcmp gate too and takes the same arm.
	if !specEnterCircle(w, player, nil, "enter", "circle ") {
		t.Fatal("enter with trailing-space circle was not consumed")
	}
	if got, want := lastMsg(), "Enter what?\n\r"; got != want {
		t.Fatalf("trailing-space enter bytes = %q, want %q", got, want)
	}
}

// TestRoomCleanupExtractsAndDropsResetCount pins dump's extract_obj port:
// C frees the object and decrements obj_index[].number (src/spec_procs.c:265,
// handler.c:1024-1025), so a later zone reset reloads it. Parking the object
// in nowhere kept Go's count and would stop the reload.
func TestRoomCleanupExtractsAndDropsResetCount(t *testing.T) {
	w, _, _ := newSpecProcTestWorld(t)
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.AddItemToRoom(obj, 1001)
	if got := w.countObjectInstances(3001); got != 1 {
		t.Fatalf("object count before cleanup = %d, want 1", got)
	}
	w.roomCleanup(1001)
	if got := len(w.GetItemsInRoom(1001)); got != 0 {
		t.Fatalf("room items after cleanup = %d, want 0", got)
	}
	if got := w.countObjectInstances(3001); got != 0 {
		t.Fatalf("object count after cleanup = %d, want 0 (reset reload)", got)
	}
}

// TestFidoExtractsCorpseAndDropsResetCount pins fido's extract_obj port
// (src/spec_procs.c:742).
func TestFidoExtractsCorpseAndDropsResetCount(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 4001, Keywords: "corpse", ShortDesc: "a corpse", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	corpse, err := w.SpawnObject(4001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject corpse: %v", err)
	}
	w.AddItemToRoom(corpse, 1001)
	me := newSpecProcTestMob(t, w, 1001, 10)
	if !specFido(w, nil, me, "", "") {
		t.Fatal("specFido did not devour the corpse")
	}
	if got := w.countObjectInstances(4001); got != 0 {
		t.Fatalf("object count after fido = %d, want 0 (reset reload)", got)
	}
}

// TestRoachExtractsEatenItemAndDropsResetCount pins roach's extract_obj port
// (src/spec_procs3.c:765).
func TestRoachExtractsEatenItemAndDropsResetCount(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 3001, Keywords: "sword steel", ShortDesc: "a steel sword", WearFlags: [4]int{1, 0, 0, 0}}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.AddItemToRoom(obj, 1001)
	me := newSpecProcTestMob(t, w, 1001, 10)
	if !w.specRoachFood(me, 1001) {
		t.Fatal("specRoachFood did not eat the item")
	}
	if got := w.countObjectInstances(3001); got != 0 {
		t.Fatalf("object count after roach = %d, want 0 (reset reload)", got)
	}
}
