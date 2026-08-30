package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

type elementsMinionTestWorld struct {
	w        *World
	actor    *Player
	peer     *Player
	mob      *MobInstance
	messages map[string]string
}

func newElementsMinionTestWorld(t *testing.T, objects []parser.Obj) elementsMinionTestWorld {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8105, Name: "Minion Test Room"}},
		Mobs: []parser.Mob{{
			VNum:      1313,
			Keywords:  "elemental minion servant",
			ShortDesc: "an elemental minion",
			LongDesc:  "An elemental minion is here.",
			Level:     10,
			Str:       10,
			Dex:       10,
		}},
		Objs: objects,
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	mob, err := w.SpawnMob(1313, 8105)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	actor := NewPlayer(1, "MinionActor", 8105)
	peer := NewPlayer(2, "MinionPeer", 8105)
	for _, player := range []*Player{actor, peer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.GetName(), err)
		}
	}

	messages := map[string]string{actor.GetName(): "", peer.GetName(): ""}
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	return elementsMinionTestWorld{w: w, actor: actor, peer: peer, mob: mob, messages: messages}
}

func placeElementsMinionObject(t *testing.T, fixture elementsMinionTestWorld, vnum int) *ObjectInstance {
	t.Helper()
	obj, err := fixture.w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	if err := fixture.w.MoveObjectToMobInventory(obj, fixture.mob); err != nil {
		t.Fatalf("MoveObjectToMobInventory(%d): %v", vnum, err)
	}
	return obj
}

func TestSpecElementsMinion_UsesOrderedVisibleKeywordPasses(t *testing.T) {
	keywords := []string{"air", "fire", "water", "earth", "element", "talisman"}
	objects := make([]parser.Obj, 0, len(keywords))
	for i, keyword := range keywords {
		objects = append(objects, parser.Obj{
			VNum:      7000 + i,
			Keywords:  keyword,
			ShortDesc: "an elemental " + keyword,
		})
	}
	fixture := newElementsMinionTestWorld(t, objects)
	for i := range keywords {
		placeElementsMinionObject(t, fixture, 7000+i)
	}

	if specElementsMinion(fixture.w, fixture.actor, fixture.mob, "say", "hello") {
		t.Fatal("elements_minion should fall through after its inventory scan")
	}

	want := ""
	for _, keyword := range []string{"talisman", "element", "earth", "fire", "water", "air"} {
		want += "An elemental minion utters the words 'eradico paratus' and an elemental " + keyword + " disintegrates.\r\n"
	}
	for _, player := range []*Player{fixture.actor, fixture.peer} {
		if got := fixture.messages[player.GetName()]; got != want {
			t.Errorf("%s received %q, want %q", player.GetName(), got, want)
		}
	}
	if len(fixture.mob.Inventory) != 0 {
		t.Fatalf("mob inventory after extraction = %#v, want empty", fixture.mob.Inventory)
	}
	if got := len(fixture.w.GetAllObjects()); got != 0 {
		t.Fatalf("active object registry after extraction = %d, want 0", got)
	}
}

func TestSpecElementsMinion_UsesKeywordsNotVnumsAndSkipsInvisibleObjects(t *testing.T) {
	fixture := newElementsMinionTestWorld(t, []parser.Obj{
		{VNum: 8000, Keywords: "unrelated", ShortDesc: "a plain stone"},
		{VNum: 8001, Keywords: "air", ShortDesc: "a strange air token"},
		{VNum: 8002, Keywords: "water", ShortDesc: "a hidden water token"},
	})
	plain := placeElementsMinionObject(t, fixture, 8000)
	air := placeElementsMinionObject(t, fixture, 8001)
	hidden := placeElementsMinionObject(t, fixture, 8002)
	hidden.SetExtraFlag(0, extraFlagInvisible)

	if specElementsMinion(fixture.w, nil, fixture.mob, "", "") {
		t.Fatal("commandless elements_minion should fall through")
	}
	if len(fixture.messages[fixture.actor.GetName()]) == 0 || len(fixture.messages[fixture.peer.GetName()]) == 0 {
		t.Fatal("visible keyword match did not emit the room Act")
	}
	if objectIsActive(fixture.w, air) {
		t.Fatal("keyword-matched non-elemental vnum remained in the object registry")
	}
	if !objectIsActive(fixture.w, plain) {
		t.Fatal("unrelated object was extracted")
	}
	if !objectIsActive(fixture.w, hidden) {
		t.Fatal("invisible object was extracted without mob detect-invis")
	}
	if got := strings.Count(fixture.messages[fixture.actor.GetName()], "disintegrates."); got != 1 {
		t.Fatalf("actor received %d destruction Acts, want 1", got)
	}
}

func objectIsActive(w *World, want *ObjectInstance) bool {
	for _, obj := range w.GetAllObjects() {
		if obj == want {
			return true
		}
	}
	return false
}

func TestFindMobInRoomUsesAuthoredKeywords(t *testing.T) {
	fixture := newElementsMinionTestWorld(t, nil)
	if got := fixture.w.FindMobInRoom(8105, "elemental"); got != fixture.mob {
		t.Fatal("elemental alias did not resolve to the minion")
	}
	if got := fixture.w.FindMobInRoom(8105, "servant"); got != fixture.mob {
		t.Fatal("servant alias did not resolve to the minion")
	}
}
