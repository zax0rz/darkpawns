package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

var galeruColumnTalismanRooms = []int{1360, 1364, 1380, 1384}

func newGaleruColumnTestWorld(t *testing.T) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()

	rooms := []parser.Room{
		{VNum: 1360, Name: "Earth Corner"},
		{VNum: 1364, Name: "Air Corner"},
		{VNum: 1380, Name: "Fire Corner"},
		{VNum: 1384, Name: "Water Corner"},
		{VNum: 1372, Name: "Galeru Column", Description: "The four corners surround the column."},
		// 1389 carries an extra description in the shipped world (lib/world/wld/13.wld),
		// which is what makes do_description append its trailing space — the byte
		// that ends up in front of the peer's arrival line.
		{
			VNum: 1389, Name: "Temple of Elements", Description: "The altar of elements stands here.",
			ExtraDescs: []parser.ExtraDesc{{Keywords: "altar", Description: "A great altar to the elements."}},
		},
		{VNum: 1395, Name: "Tropical Island", Description: "A tropical island surrounds you."},
	}
	objs := make([]parser.Obj, 0, 4)
	for vnum := 1300; vnum <= 1303; vnum++ {
		objs = append(objs, parser.Obj{
			VNum:      vnum,
			Keywords:  "talisman",
			ShortDesc: "an elemental talisman",
			LongDesc:  "An elemental talisman lies here.",
		})
	}
	w, err := NewWorld(&parser.World{Rooms: rooms, Objs: objs})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	npc := newSpecProcTestMob(t, w, 1372, 10)
	actor := NewPlayer(1, "GaleruCarrier", 1372)
	peer := NewPlayer(2, "GaleruPeer", 1372)
	actor.SetAutoExit(false)
	peer.SetAutoExit(false)
	// C's world[room].people is front-inserted by char_to_room, so the last
	// body added to the room is the head of the list. Add the peer first so the
	// command actor is the newest entrant, matching the oracle vehicle
	// (scenarios/spec-proc-elements-*.txt: peer teleported, actor walks in).
	for _, player := range []*Player{peer, actor} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.GetName(), err)
		}
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, peer, npc, messages
}

func placeGaleruTalisman(t *testing.T, w *World, vnum, room int) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, room)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	if err := w.MoveObjectToRoom(obj, room); err != nil {
		t.Fatalf("MoveObjectToRoom(%d): %v", vnum, err)
	}
	return obj
}

func placeAllGaleruTalismans(t *testing.T, w *World) {
	t.Helper()
	for i, room := range galeruColumnTalismanRooms {
		placeGaleruTalisman(t, w, 1300+i, room)
	}
}

func TestSpecElementsGaleruColumn_RequiresAllExactTalismans(t *testing.T) {
	w, actor, peer, npc, messages := newGaleruColumnTestWorld(t)
	for i, room := range galeruColumnTalismanRooms[:3] {
		placeGaleruTalisman(t, w, 1300+i, room)
	}

	if specElementsGaleruColumn(w, actor, nil, "say", "hello") {
		t.Fatal("incomplete talisman set should fall through")
	}
	for _, player := range []*Player{actor, peer} {
		if got := player.GetRoomVNum(); got != 1372 {
			t.Errorf("%s room = %d, want 1372", player.GetName(), got)
		}
	}
	if got := npc.GetRoom(); got != 1372 {
		t.Errorf("NPC room = %d, want 1372", got)
	}
	for name, msg := range messages {
		if msg != "" {
			t.Fatalf("incomplete invocation emitted output for %s: %q", name, msg)
		}
	}
}

func TestSpecElementsGaleruColumn_AudienceLookAndRelocation(t *testing.T) {
	w, actor, peer, npc, messages := newGaleruColumnTestWorld(t)
	placeAllGaleruTalismans(t, w)
	// The look's bare bracket writes (which carry do_description's trailing
	// space) go to a separate sink from act() output; capture both.
	literal := make(map[string]string)
	w.LiteralMessageSink = func(name string, msg []byte) { literal[name] += string(msg) }

	if !specElementsGaleruColumn(w, actor, nil, "say", "hello") {
		t.Fatal("complete talisman set should be consumed")
	}
	for _, player := range []*Player{actor, peer} {
		if got := player.GetRoomVNum(); got != 1389 {
			t.Errorf("%s room = %d, want 1389", player.GetName(), got)
		}
		got := messages[player.GetName()]
		if !strings.Contains(got, "Four beams of colored light from the corners of the chamber converge around you.\r\n\n") {
			t.Errorf("%s missing exact direct beam message: %q", player.GetName(), got)
		}
		if !strings.Contains(got, "Temple of Elements") {
			t.Errorf("%s missing destination look: %q", player.GetName(), got)
		}
	}
	if got := messages[actor.GetName()]; strings.Contains(got, "GaleruCarrier is struck") {
		t.Errorf("actor received own departure: %q", got)
	}
	if got := messages[actor.GetName()]; !strings.Contains(got, "GaleruPeer materialises from nowhere in a swirl of colors.") {
		t.Errorf("actor missing peer arrival: %q", got)
	}
	// The arrival act is C's exact format, with no leading space
	// (src/spec_procs3.c:1181). The one space in front of it is do_description's
	// trailing space, written as the first byte of the next line when the
	// destination room lists extra descriptions (src/act.informative.c:2779), and
	// the look emits it through its bare bracket write. A second space means the
	// act re-added a byte the look already wrote.
	assertPeerArrivalLine(t, literal[actor.GetName()], messages[actor.GetName()])
	if !strings.Contains(messages[peer.GetName()], "GaleruCarrier is struck by four beams of colored light and slowly vanishes!") {
		t.Errorf("peer missing actor departure: %q", messages[peer.GetName()])
	}
	if strings.Contains(messages[peer.GetName()], "GaleruPeer is struck") {
		t.Errorf("peer received own departure: %q", messages[peer.GetName()])
	}
	if got := npc.GetRoom(); got != 1372 {
		t.Errorf("NPC room = %d, want 1372", got)
	}
}

// assertPeerArrivalLine pins both halves of C's byte stream: the destination
// look writes do_description's trailing space as the first byte of the next line
// (src/act.informative.c:2779), and the arrival act that follows carries no
// leading space of its own (src/spec_procs3.c:1181). Exactly one space, from
// exactly one writer.
func assertPeerArrivalLine(t *testing.T, literal, messages string) {
	t.Helper()
	const arrival = "GaleruPeer materialises from nowhere in a swirl of colors."
	if got := strings.Count(literal, " "); got != 1 {
		t.Errorf("look's bare writes = %q, want exactly one space (do_description's trailing space)", literal)
	}
	for _, line := range strings.Split(messages, "\n") {
		if !strings.Contains(line, "materialises from nowhere") {
			continue
		}
		if got := strings.TrimRight(line, "\r"); got != arrival {
			t.Errorf("peer arrival line = %q, want %q — the space belongs to the look, not the act", got, arrival)
		}
		return
	}
	t.Errorf("no peer arrival line in %q", messages)
}

func TestSpecElementsGaleruColumn_RejectsNilActor(t *testing.T) {
	w, _, _, _, messages := newGaleruColumnTestWorld(t)
	if specElementsGaleruColumn(w, nil, nil, "say", "hello") {
		t.Fatal("nil actor should not be handled")
	}
	for name, msg := range messages {
		if msg != "" {
			t.Fatalf("nil actor emitted output for %s: %q", name, msg)
		}
	}
}
