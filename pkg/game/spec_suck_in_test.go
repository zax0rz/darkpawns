package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecSuckIn_LookGateAudienceAndRelocation(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{
				VNum: 20073,
				Name: "Eastern Hallway",
				ExtraDescs: []parser.ExtraDesc{{
					Keywords:    "picture painting",
					Description: "The painting shows a strange maze.",
				}},
			},
			{VNum: 18101, Name: "Entrance Tunnel", Description: "A strong wind flows outward.\n"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	actor := NewPlayer(1, "Hallprobe", 20073)
	observer := NewPlayer(2, "Hallwitness", 20073)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) {
		transcript[name] += string(msg)
	}

	if specSuckIn(w, actor, nil, "examine", "painting") {
		t.Fatal("non-look command should fall through")
	}
	if specSuckIn(w, actor, nil, "look", "picture") {
		t.Fatal("picture alias should not trigger the exact painting gate")
	}
	if got := transcript[actor.Name] + transcript[observer.Name]; got != "" {
		t.Fatalf("gate output = %q, want empty", got)
	}

	if !specSuckIn(w, actor, nil, "look", "the painting") {
		t.Fatal("fill-word-prefixed painting should trigger C one_argument path")
	}
	if got := actor.GetRoom(); got != paintingRoom {
		t.Fatalf("actor room = %d, want %d", got, paintingRoom)
	}
	wantActor := "The painting shows a strange maze.\r\n" +
		"\r\n\r\n\r\n\r\nYou suddenly feel very dizzy...\r\n\r\n" +
		"Entrance Tunnel\r\nA strong wind flows outward.\r\n[ Exits: None! ]\r\n"
	if got := transcript[actor.Name]; got != wantActor {
		t.Errorf("actor transcript = %q, want exact C look, dizziness, and landing bytes %q", got, wantActor)
	}
	if got, want := transcript[observer.Name], "Hallprobe suddenly vanishes!\r\n"; got != want {
		t.Errorf("observer output = %q, want exact actor-excluding TO_ROOM line %q", got, want)
	}
}
