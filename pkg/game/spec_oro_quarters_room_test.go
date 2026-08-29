package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecOroQuartersRoom_GatesAudienceHPAndFallthrough(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 18397, Name: "Orodreth's Quarters", Zone: 183}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	output := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) {
		output[name] += string(msg)
	}
	actor := NewPlayer(1, "Quartersactor", 18397)
	actor.SetPosition(combat.PosStanding)
	actor.SetHP(101)
	observer := NewPlayer(2, "Quarterspeer", 18397)
	observer.SetPosition(combat.PosStanding)
	for _, player := range []*Player{actor, observer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}

	if got := specOroQuartersRoom(w, actor, nil, "look", ""); got {
		t.Fatal("non-south command was handled")
	}
	if actor.GetHP() != 101 || output[actor.Name] != "" || output[observer.Name] != "" {
		t.Fatalf("non-south gate changed state: hp=%d actor=%q observer=%q", actor.GetHP(), output[actor.Name], output[observer.Name])
	}

	if got := specOroQuartersRoom(w, actor, nil, "south", "ignored argument"); !got {
		t.Fatal("non-Orodreth south command was not handled")
	}
	if got, want := actor.GetHP(), 50; got != want {
		t.Errorf("HP after jolt = %d, want C integer half %d", got, want)
	}
	if got, want := output[actor.Name], "A strong force blocks your way and gives you a nasty jolt.\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	if got, want := output[observer.Name], "A strong force jolts Quartersactor in his attempt to leave south.\r\n"; got != want {
		t.Errorf("observer output = %q, want %q", got, want)
	}

	output[actor.Name] = ""
	output[observer.Name] = ""
	orodreth := NewPlayer(3, "Orodreth", 18397)
	if err := w.AddPlayer(orodreth); err != nil {
		t.Fatalf("AddPlayer(Orodreth): %v", err)
	}
	orodreth.SetHP(101)
	if got := specOroQuartersRoom(w, orodreth, nil, "south", ""); got {
		t.Fatal("Orodreth south command was handled")
	}
	if got := orodreth.GetHP(); got != 101 {
		t.Errorf("Orodreth HP = %d, want unchanged 101", got)
	}
	if output[orodreth.Name] != "" || output[observer.Name] != "" {
		t.Fatalf("Orodreth fallthrough emitted output: actor=%q observer=%q", output[orodreth.Name], output[observer.Name])
	}
}
