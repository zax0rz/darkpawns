package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

const elementsMasterColumnRoom = 1315

func newElementsMasterColumnTestWorld(t *testing.T) (*World, *Player, *Player, map[string]string) {
	t.Helper()

	rooms := []parser.Room{
		{VNum: elementsMasterColumnRoom, Name: "Master Column", Description: "The master column."},
		{VNum: 1320, Name: "Earth Plane", Description: "Earth surrounds you."},
		{VNum: 1331, Name: "Air Plane", Description: "Air surrounds you."},
		{VNum: 1342, Name: "Fire Plane", Description: "Fire surrounds you."},
		{VNum: 1353, Name: "Water Plane", Description: "Water surrounds you."},
		{VNum: 1372, Name: "Elemental Chamber", Description: "The chamber surrounds you."},
	}
	w, err := NewWorld(&parser.World{Rooms: rooms})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "ColumnCarrier", elementsMasterColumnRoom)
	peer := NewPlayer(2, "ColumnPeer", elementsMasterColumnRoom)
	actor.SetAutoExit(false)
	peer.SetAutoExit(false)
	for _, player := range []*Player{actor, peer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.GetName(), err)
		}
	}
	return w, actor, peer, messages
}

func elementsTalisman(vnum int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{VNum: vnum, Keywords: "talisman", ShortDesc: "a talisman"}, -1)
}

func TestSpecElementsMasterColumn_EntryAndTalismanDestinations(t *testing.T) {
	tests := []struct {
		name       string
		actorItems []int
		peerItems  []int
		actorRoom  int
		actorText  string
	}{
		{
			name:      "no talismans",
			actorRoom: 1320,
			actorText: "You feel a tingling sensation and your vision fades. When you wake...",
		},
		{
			name:       "earth only uses missing air plane",
			actorItems: []int{1300},
			actorRoom:  1331,
			actorText:  "The talisman of earth glows softly and your vision fades. When you wake...",
		},
		{
			name:       "earth and air use missing fire plane",
			actorItems: []int{1300, 1301},
			actorRoom:  1342,
			actorText:  "The talisman of air glows softly and your vision fades. When you wake...",
		},
		{
			name:       "all four uses elemental chamber",
			actorItems: []int{1300, 1301, 1302, 1303},
			actorRoom:  1372,
			actorText:  "The four talismans glow softly and your vision fades. When you wake...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, actor, peer, messages := newElementsMasterColumnTestWorld(t)
			for _, vnum := range tc.actorItems {
				if err := actor.Inventory.AddItem(elementsTalisman(vnum)); err != nil {
					t.Fatalf("add actor talisman %d: %v", vnum, err)
				}
			}
			for _, vnum := range tc.peerItems {
				if err := peer.Inventory.AddItem(elementsTalisman(vnum)); err != nil {
					t.Fatalf("add peer talisman %d: %v", vnum, err)
				}
			}

			if !specElementsMasterColumn(w, actor, nil, "look", "") {
				t.Fatal("room special did not consume the command")
			}
			if got := actor.GetRoomVNum(); got != tc.actorRoom {
				t.Errorf("actor room = %d, want %d", got, tc.actorRoom)
			}
			if !strings.Contains(messages[actor.GetName()], tc.actorText) {
				t.Errorf("actor output missing %q: %q", tc.actorText, messages[actor.GetName()])
			}
		})
	}
}

func TestSpecElementsMasterColumn_PreservesCStaleCarryStateAndAudience(t *testing.T) {
	w, actor, peer, messages := newElementsMasterColumnTestWorld(t)
	if err := actor.Inventory.AddItem(elementsTalisman(1301)); err != nil {
		t.Fatalf("add actor air talisman: %v", err)
	}
	if err := peer.Inventory.AddItem(elementsTalisman(1300)); err != nil {
		t.Fatalf("add peer earth talisman: %v", err)
	}

	if !specElementsMasterColumn(w, actor, nil, "say", "hello") {
		t.Fatal("room special did not consume the command")
	}
	if got, want := actor.GetRoomVNum(), 1320; got != want {
		t.Errorf("actor room = %d, want %d", got, want)
	}
	if got, want := peer.GetRoomVNum(), 1342; got != want {
		t.Errorf("peer room = %d, want %d after C stale carry state", got, want)
	}
	if !strings.Contains(messages[actor.GetName()], "You feel a tingling sensation") {
		t.Errorf("actor output missing no-talisman message: %q", messages[actor.GetName()])
	}
	if !strings.Contains(messages[peer.GetName()], "The talisman of air glows softly") {
		t.Errorf("peer output missing stale air message: %q", messages[peer.GetName()])
	}
	if !strings.Contains(messages[peer.GetName()], "ColumnCarrier vanishes in a brilliant flash of light.") {
		t.Errorf("peer output missing actor departure: %q", messages[peer.GetName()])
	}
}

func TestSpecElementsMasterColumn_RejectsNilActor(t *testing.T) {
	w, _, _, messages := newElementsMasterColumnTestWorld(t)
	if specElementsMasterColumn(w, nil, nil, "look", "") {
		t.Fatal("nil actor should not be handled")
	}
	if len(messages) != 0 {
		t.Fatalf("nil actor emitted output: %v", messages)
	}
}
