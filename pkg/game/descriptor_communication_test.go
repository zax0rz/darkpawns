package game

import (
	"testing"
)

func TestMobGlobalGossipOnlyReachesPlayingDescriptors(t *testing.T) {
	w, _, _ := newSpecProcTestWorld(t)
	players := []*Player{
		NewPlayer(1, "Playing", 1001),
		NewPlayer(2, "Menu", 1001),
		NewPlayer(3, "Editor", 1001),
		NewPlayer(4, "Linkless", 1001),
	}
	for _, player := range players {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.Name, err)
		}
	}
	w.PlayerHasPlayingDescriptor = func(player *Player) bool { return player.Name == "Playing" }
	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] = append(messages[name], string(msg)) }
	mob := newSpecProcTestMob(t, w, 1001, 10)

	w.mobGlobalGossip(mob, "hello")
	if got, want := messages["Playing"], []string{"A test mob gossips, 'hello'\r\n"}; !equalDescriptorMessages(got, want) {
		t.Errorf("playing mob-gossip bytes = %q, want %q", got, want)
	}
	for _, player := range players[1:] {
		if got := messages[player.Name]; len(got) != 0 {
			t.Errorf("non-playing %s received mob gossip: %q", player.Name, got)
		}
	}
}
