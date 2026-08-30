package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecElementsPlatforms_EntryAudienceAndRelocation(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1314, Name: "Column Base"},
		{VNum: 1326, Name: "Earth Platform"},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "PlatformCarrier", 1326)
	peer := NewPlayer(2, "PlatformPeer", 1326)
	for _, player := range []*Player{actor, peer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.GetName(), err)
		}
	}

	if !specElementsPlatforms(w, actor, nil, "say", "hello") {
		t.Fatal("room special did not consume the command")
	}
	for _, player := range []*Player{actor, peer} {
		if got := player.GetRoomVNum(); got != 1314 {
			t.Errorf("%s room = %d, want 1314", player.GetName(), got)
		}
	}
	if got := messages[actor.GetName()]; !strings.Contains(got, "A wave of power surges through you and you feel dizzy.") {
		t.Errorf("actor missing direct dizzy message: %q", got)
	}
	if !strings.Contains(messages[actor.GetName()], "PlatformPeer appears in a brilliant flash of light.") {
		t.Errorf("actor missing peer arrival: %q", messages[actor.GetName()])
	}
	if strings.Contains(messages[actor.GetName()], "PlatformCarrier disappears") || strings.Contains(messages[actor.GetName()], "PlatformPeer disappears") {
		t.Errorf("actor received a departure self/peer leak: %q", messages[actor.GetName()])
	}
	if got := messages[peer.GetName()]; !strings.Contains(got, "PlatformCarrier disappears in a brilliant flash of light.") {
		t.Errorf("peer missing actor departure: %q", got)
	}
	if !strings.Contains(messages[peer.GetName()], "A wave of power surges through you and you feel dizzy.") {
		t.Errorf("peer missing direct dizzy message: %q", messages[peer.GetName()])
	}
	if strings.Contains(messages[peer.GetName()], "PlatformPeer disappears") || strings.Contains(messages[peer.GetName()], "PlatformCarrier appears") {
		t.Errorf("peer received a departure self/arrival leak: %q", messages[peer.GetName()])
	}
}

func TestSpecElementsPlatforms_RejectsNilActor(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1314, Name: "Column Base"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	if specElementsPlatforms(w, nil, nil, "say", "hello") {
		t.Fatal("nil actor should not be handled")
	}
}
