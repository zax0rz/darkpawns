package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestScriptZoneEchoOnlyVisitsAwakePlayingDescriptors(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Origin", Zone: 1},
		{VNum: 1002, Name: "Listener room", Zone: 1},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	players := []*Player{
		NewPlayer(1, "Origin", 1001),
		NewPlayer(2, "Playing", 1002),
		NewPlayer(3, "SameRoom", 1001),
		NewPlayer(4, "Sleeping", 1002),
		NewPlayer(5, "Menu", 1002),
		NewPlayer(6, "Editor", 1002),
		NewPlayer(7, "Linkless", 1002),
	}
	for _, player := range players {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.Name, err)
		}
	}
	players[3].SetPosition(combat.PosSleeping)
	w.PlayerHasPlayingDescriptor = func(player *Player) bool {
		return player.Name == "Origin" || player.Name == "Playing" || player.Name == "SameRoom" || player.Name == "Sleeping"
	}
	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] = append(messages[name], string(msg)) }

	actor := scripting.CharRef{ID: players[0].GetID()}
	const message = "A script zone echo.\r\n"
	NewWorldScriptableAdapter(w).Echo("zone", nil, &actor, message)
	if got, want := messages["Playing"], []string{message}; !equalDescriptorMessages(got, want) {
		t.Errorf("awake different-room playing recipient = %q, want %q", got, want)
	}
	for _, player := range []*Player{players[0], players[2], players[3], players[4], players[5], players[6]} {
		if got := messages[player.Name]; len(got) != 0 {
			t.Errorf("%s received zone echo: %q", player.Name, got)
		}
	}
}
