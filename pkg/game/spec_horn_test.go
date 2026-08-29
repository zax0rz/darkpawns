package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecHornObjectReceiverAudienceAndGates(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Horn Room", Zone: 1},
		{VNum: 1002, Name: "Same Zone", Zone: 1},
		{VNum: 2001, Name: "Other Zone", Zone: 2},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] += string(msg)
	}
	actor := NewPlayer(1, "Hornactor", 1001)
	roomPeer := NewPlayer(2, "Roompeer", 1001)
	zonePeer := NewPlayer(3, "Zonepeer", 1002)
	otherZonePeer := NewPlayer(4, "Otherzonepeer", 2001)
	for _, player := range []*Player{actor, roomPeer, zonePeer, otherZonePeer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}

	horn := NewObjectInstance(&parser.Obj{
		VNum:      14415,
		Keywords:  "silver horn",
		ShortDesc: "a silver horn",
		WearFlags: [4]int{16385},
	}, -1)
	if err := actor.Inventory.AddItem(horn); err != nil {
		t.Fatalf("add horn to inventory: %v", err)
	}
	if err := actor.Equipment.Equip(horn, actor.Inventory); err != nil {
		t.Fatalf("hold horn: %v", err)
	}

	if fn := GetObjSpecForObject(14415); fn == nil {
		t.Fatal("horn object-special lookup returned nil")
	}
	if got := specHornObject(w, actor, horn, "use", "silver"); !got {
		t.Fatal("matching held horn command was not handled")
	}

	if got, want := messages[actor.Name], "You inhale deeply then blow hard!\r\nA blaring note resounds through the air.\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	if got, want := messages[roomPeer.Name], "Hornactor blows into a silver horn.\r\nA silver horn lets out a blaring note...\r\n"; got != want {
		t.Errorf("same-room output = %q, want %q", got, want)
	}
	if got, want := messages[zonePeer.Name], "You hear the blaring of a loud horn.\r\n"; got != want {
		t.Errorf("same-zone output = %q, want %q", got, want)
	}
	if got := messages[otherZonePeer.Name]; got != "" {
		t.Errorf("other-zone output = %q, want empty", got)
	}

	for name := range messages {
		messages[name] = ""
	}
	if got := specHornObject(w, actor, horn, "use", "horn extra"); got {
		t.Fatal("multi-word horn argument was handled")
	}
	for name, got := range messages {
		if got != "" {
			t.Errorf("keyword-gate output for %s = %q, want empty", name, got)
		}
	}

	if ok := actor.Equipment.UnequipItem(horn, actor.Inventory); !ok {
		t.Fatal("remove horn from hold failed")
	}
	if got := specHornObject(w, actor, horn, "use", "horn"); got {
		t.Fatal("un-held horn command was handled")
	}
}
