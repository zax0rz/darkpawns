package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newSpecProcTestWorld creates a minimal world for spec proc tests.
func newSpecProcTestWorld(t *testing.T) (*World, *Player, func() string) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Dump Room", Zone: 1},
		},
		Objs: []parser.Obj{
			{
				VNum:      3001,
				Keywords:  "sword steel",
				ShortDesc: "a steel sword",
				LongDesc:  "A steel sword lies here.",
				Cost:      100,
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	player := NewPlayer(1, "Tester", 1001)
	player.SetLevel(5)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	return w, player, lastMsg
}

// TestSpecDumpPerformDrop verifies that specDump actually drops the item,
// removes it from inventory, and awards gold for the dumped value (DP-944).
func TestSpecDumpPerformDrop(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)

	sword := NewObjectInstance(w.objs[3001], -1)
	if err := w.MoveObjectToPlayerInventory(sword, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}

	startGold := player.GetGold()
	if got := specDump(w, player, nil, "drop", "sword"); !got {
		t.Fatal("specDump should handle drop command")
	}

	if _, ok := player.Inventory.FindItem("sword"); ok {
		t.Error("dropped sword should no longer be in inventory")
	}

	// Cost 100 → value clamp(100/10, 1, 10) = 10 gold for a level 5 player.
	if player.GetGold() <= startGold {
		t.Errorf("player should receive gold award, got %d want > %d", player.GetGold(), startGold)
	}

	// Room should be cleaned after awarding.
	items := w.GetItemsInRoom(1001)
	for _, item := range items {
		if item == sword {
			t.Error("dropped sword should have been cleaned from room")
		}
	}
}

// TestSpecDumpNonDropPassThrough verifies that specDump returns false for
// commands other than drop, allowing normal command processing.
func TestSpecDumpNonDropPassThrough(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)

	if got := specDump(w, player, nil, "look", ""); got {
		t.Error("specDump should return false for non-drop commands")
	}
}

// TestActMessageAudienceRouting verifies actMessage sends the correct message
// to actor, victim, and bystanders (DP-945 helper regression).
func TestActMessageAudienceRouting(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Theater", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	actor := NewPlayer(1, "Actor", 1001)
	victim := NewPlayer(2, "Victim", 1001)
	bystander := NewPlayer(3, "Bystander", 1001)
	for _, p := range []*Player{actor, victim, bystander} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer failed: %v", err)
		}
	}

	var msgs syncMap
	w.MessageSink = func(name string, msg []byte) { msgs.Store(name, string(msg)) }

	w.actMessage(1001, actor, victim,
		"You poke yourself.",
		"Actor pokes you!",
		"Actor pokes Victim.",
	)

	if got := msgs.Load("Actor"); !strings.Contains(got, "You poke yourself.") {
		t.Errorf("actor message = %q, want containing 'You poke yourself.'", got)
	}
	if got := msgs.Load("Victim"); !strings.Contains(got, "Actor pokes you!") {
		t.Errorf("victim message = %q, want containing 'Actor pokes you!'", got)
	}
	if got := msgs.Load("Bystander"); !strings.Contains(got, "Actor pokes Victim.") {
		t.Errorf("bystander message = %q, want containing 'Actor pokes Victim.'", got)
	}
}

// syncMap is a tiny test helper for collecting per-player messages.
type syncMap struct {
	m map[string]string
}

func (s *syncMap) Store(key, value string) {
	if s.m == nil {
		s.m = make(map[string]string)
	}
	s.m[key] += value
}

func (s *syncMap) Load(key string) string {
	if s.m == nil {
		return ""
	}
	return s.m[key]
}
