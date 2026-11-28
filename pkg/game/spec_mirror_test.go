package game

// spec_mirror_test.go — mirror port cover (src/spec_procs3.c:612-668). C
// declares the special without assigning it (src/spec_assign.c), so the
// registered object handler is exercised directly; the registration lookup
// fails closed on origin/main.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mirrorWorld builds the mirror's room (1001), the destination (14496),
// the mirror object, one looker, and one person waiting in 14496. Transcript
// keeps per-recipient sinks: the disappears line is TO_ROOM, which excludes
// the actor, so only an observer in 1001 receives it (src/spec_procs3.c:646).
func mirrorWorld(t *testing.T) (*World, *Player, *Player, *ObjectInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Mirror Room", Zone: 1},
			{VNum: 14496, Name: "Behind Room", Zone: 1},
		},
		Objs: []parser.Obj{
			{VNum: 501, Keywords: "mirror", ShortDesc: "a tall mirror"},
			{VNum: 14503, Keywords: "piece", ShortDesc: "a mirror piece"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	looker := NewPlayer(1, "Mirrorlooker", 1001)
	swapper := NewPlayer(2, "Mirrorswapper", 14496)
	observer := NewPlayer(3, "Mirrorobserver", 1001)
	for _, p := range []*Player{looker, swapper, observer} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}
	mirror, err := w.SpawnObject(501, 1001)
	if err != nil {
		t.Fatalf("SpawnObject mirror: %v", err)
	}
	w.AddItemToRoom(mirror, 1001)
	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }
	return w, looker, swapper, mirror, transcript
}

func TestSpecMirrorLookSwapsRooms(t *testing.T) {
	w, looker, swapper, mirror, transcript := mirrorWorld(t)
	fn := ObjSpecRegistry["mirror"]
	if fn == nil {
		t.Fatal("mirror is not registered as an object special")
	}
	if !fn(w, looker, mirror, "look", "mirror") {
		t.Fatal("look at mirror did not handle")
	}
	// src/spec_procs3.c:656-657: the looker ends in 14496; the other person
	// ends in the mirror's room.
	if got := looker.GetRoomVNum(); got != 14496 {
		t.Fatalf("looker room = %d, want 14496", got)
	}
	if got := swapper.GetRoomVNum(); got != 1001 {
		t.Fatalf("swapper room = %d, want 1001", got)
	}
	// src/spec_procs3.c:645-646: the looker's pulled line (TO_CHAR) and the
	// disappears line (TO_ROOM, so the observer hears it, not the actor).
	if got := transcript[looker.Name]; !strings.Contains(got, "pulled in a hundred different directions") {
		t.Fatalf("looker output = %q, want the pulled line", got)
	}
	if got := transcript["Mirrorobserver"]; !strings.Contains(got, "disappears in a brilliant flash") {
		t.Fatalf("observer output = %q, want the disappears line", got)
	}
}

func TestSpecMirrorHitShattersAndSwaps(t *testing.T) {
	w, looker, swapper, mirror, _ := mirrorWorld(t)
	fn := ObjSpecRegistry["mirror"]
	if fn == nil {
		t.Fatal("mirror is not registered as an object special")
	}
	if !fn(w, looker, mirror, "hit", "mirror") {
		t.Fatal("hit mirror did not handle")
	}
	// src/spec_procs3.c:639: the mirror is extracted; a piece lands in
	// its room (src/spec_procs3.c:638).
	if got := w.countObjectInstances(501); got != 0 {
		t.Fatalf("mirror count = %d, want 0", got)
	}
	if got := w.countObjectInstances(14503); got != 1 {
		t.Fatalf("piece count = %d, want 1", got)
	}
	if got := len(w.GetItemsInRoom(1001)); got != 1 {
		t.Fatalf("room items = %d, want the piece", got)
	}
	// src/spec_procs3.c:630-635: the other person is pulled behind you.
	if got := swapper.GetRoomVNum(); got != 1001 {
		t.Fatalf("swapper room = %d, want 1001", got)
	}
}

func TestSpecMirrorGatesDecline(t *testing.T) {
	w, looker, _, mirror, _ := mirrorWorld(t)
	fn := ObjSpecRegistry["mirror"]
	if fn == nil {
		t.Fatal("mirror is not registered as an object special")
	}
	// src/spec_procs3.c:622: isname(argument, obj->name) gates both arms.
	if fn(w, looker, mirror, "look", "window") {
		t.Fatal("look at window should decline")
	}
	// Only hit/kill and look arms exist (src/spec_procs3.c:625 vs :644).
	if fn(w, looker, mirror, "say", "mirror") {
		t.Fatal("say mirror should decline")
	}
	// A floating mirror (in_room < 0) declines (src/spec_procs3.c:617).
	if err := w.MoveObjectToNowhere(mirror); err != nil {
		t.Fatalf("MoveObjectToNowhere: %v", err)
	}
	if fn(w, looker, mirror, "look", "mirror") {
		t.Fatal("floating mirror should decline")
	}
}
