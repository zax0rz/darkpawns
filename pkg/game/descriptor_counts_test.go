package game

import "testing"

func TestCounterProcsBlessingOnlyReachesPlayingDescriptors(t *testing.T) {
	w := newCounterProcsTestWorld(t)
	players := []*Player{
		NewPlayer(1, "Actor", 1001),
		NewPlayer(2, "Playing", 1001),
		NewPlayer(3, "Menu", 1001),
		NewPlayer(4, "Editor", 1001),
		NewPlayer(5, "Linkless", 1001),
	}
	for _, player := range players {
		player.MaxHealth = 100
		player.Health = 1
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.Name, err)
		}
	}
	playing := map[*Player]bool{players[0]: true, players[1]: true}
	w.PlayerHasPlayingDescriptor = func(player *Player) bool { return playing[player] }
	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] = append(messages[name], string(msg)) }

	w.counter_procs(players[0], 5000)
	if got, want := players[1].Health, 100; got != want {
		t.Errorf("playing descriptor HP = %d, want full heal %d", got, want)
	}
	if got, want := messages["Playing"], []string{"The gods of war and death bestow a blessing upon you.\r\n"}; !equalDescriptorMessages(got, want) {
		t.Errorf("playing descriptor blessing = %q, want %q", got, want)
	}
	for _, player := range players[2:] {
		if player.Health != 1 || len(messages[player.Name]) != 0 {
			t.Errorf("non-playing %s state: HP=%d messages=%q; want HP=1 and no blessing", player.Name, player.Health, messages[player.Name])
		}
	}
}

func TestCombatHealAllCallbackOnlyHealsPlayingDescriptors(t *testing.T) {
	w := newCounterProcsTestWorld(t)
	playing := []*Player{
		NewPlayer(1, "Actor", 1001),
		NewPlayer(2, "Playing", 1001),
		NewPlayer(3, "Menu", 1001),
		NewPlayer(4, "Editor", 1001),
		NewPlayer(5, "Linkless", 1001),
	}
	for _, player := range playing {
		player.MaxHealth = 100
		player.Health = 1
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.Name, err)
		}
	}
	w.PlayerHasPlayingDescriptor = func(player *Player) bool {
		return player.Name == "Actor" || player.Name == "Playing"
	}
	w.WireCombatCallbacks().HealAllPlayers()
	if got := playing[0].Health; got != 100 {
		t.Errorf("actor HP = %d, want 100", got)
	}
	if got := playing[1].Health; got != 100 {
		t.Errorf("playing descriptor HP = %d, want 100", got)
	}
	for _, player := range playing[2:] {
		if got := player.Health; got != 1 {
			t.Errorf("non-playing %s HP = %d, want unchanged 1", player.Name, got)
		}
	}
}
