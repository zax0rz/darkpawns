package game

import "testing"

func TestSpecJail_CommandPathFallsThrough(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	player.SetRoom(8118)
	player.SetGold(100)
	player.SetMove(100)
	player.SetLevel(5)

	if specJail(w, player, nil, "say", "release") {
		t.Fatal("jail room special must reject player commands")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("jail command emitted %q", got)
	}
	if got := player.GetRoomVNum(); got != 8118 {
		t.Fatalf("jail command moved player to room %d", got)
	}
	if got := player.GetGold(); got != 100 {
		t.Fatalf("jail command changed gold to %d", got)
	}
	if got := player.GetMove(); got != 100 {
		t.Fatalf("jail command changed movement to %d", got)
	}
}
