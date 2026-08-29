package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecOroStudyRoom_GatesAudienceHPAndFallthrough(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 18399, Name: "The Coding Room", Zone: 183}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) {
		output[name] += string(msg)
	}
	actor := NewPlayer(1, "Studyactor", 18399)
	actor.SetPosition(combat.PosStanding)
	actor.SetHP(101)
	observer := NewPlayer(2, "Studypeer", 18399)
	observer.SetPosition(combat.PosStanding)
	for _, player := range []*Player{actor, observer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}

	if got := specOroStudyRoom(w, actor, nil, "look", ""); got {
		t.Fatal("non-north command was handled")
	}
	if actor.GetHP() != 101 || output[actor.Name] != "" || output[observer.Name] != "" {
		t.Fatalf("non-north gate changed state: hp=%d actor=%q observer=%q", actor.GetHP(), output[actor.Name], output[observer.Name])
	}

	if got := specOroStudyRoom(w, actor, nil, "north", "ignored argument"); !got {
		t.Fatal("non-Orodreth north command was not handled")
	}
	if got, want := actor.GetHP(), 50; got != want {
		t.Errorf("HP after jolt = %d, want C integer half %d", got, want)
	}
	if got, want := output[actor.Name], "A strong force blocks your way and gives you a nasty jolt.\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	if got, want := output[observer.Name], "A strong force jolts Studyactor in his attempt to leave north.\r\n"; got != want {
		t.Errorf("observer output = %q, want %q", got, want)
	}

	output[actor.Name] = ""
	output[observer.Name] = ""
	orodreth := NewPlayer(3, "Orodreth", 18399)
	if err := w.AddPlayer(orodreth); err != nil {
		t.Fatalf("AddPlayer(Orodreth): %v", err)
	}
	orodreth.SetHP(101)
	if got := specOroStudyRoom(w, orodreth, nil, "north", ""); got {
		t.Fatal("Orodreth north command was handled")
	}
	if got := orodreth.GetHP(); got != 101 {
		t.Errorf("Orodreth HP = %d, want unchanged 101", got)
	}
	if output[orodreth.Name] != "" || output[observer.Name] != "" {
		t.Fatalf("Orodreth fallthrough emitted output: actor=%q observer=%q", output[orodreth.Name], output[observer.Name])
	}
}
