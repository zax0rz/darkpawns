package game

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoDreamMatchesCOrderAudienceAndVisibility(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Dream Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	actor := NewPlayer(1, "Dreamer", 1001)
	awake := NewPlayer(2, "Awake", 1001)
	sleeper := NewPlayer(3, "Sleeper", 1001)
	blind := NewPlayer(4, "Blind", 1001)
	actor.SetPosition(combat.PosSleeping)
	sleeper.SetPosition(combat.PosSleeping)
	blind.SetAffect(affBlind, true)
	for _, p := range []*Player{actor, awake, sleeper, blind} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) failed: %v", p.Name, err)
		}
	}

	var events []string
	w.MessageSink = func(name string, message []byte) {
		events = append(events, name+":"+string(message))
	}

	DoDream(w, actor)

	want := []string{
		"Awake:Dreamer dreams of running naked through a field of tulips.\r\n",
		"Dreamer:You dream of running naked through a field of tulips.\r\n",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("dream events = %#v, want %#v", events, want)
	}
}
