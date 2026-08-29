package game

import "testing"

// TestSpecJailGuard_EntryGates covers SPECIAL(jailguard)'s command, player,
// level, and room gates (src/spec_procs.c:1783-1789).
func TestSpecJailGuard_EntryGates(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 29)
	_ = lastMsg() // discard the spawn announcement

	player.SetRoom(8118)
	mob.SetRoom(8118)

	tests := []struct {
		name  string
		ch    *Player
		cmd   string
		room  int
		level int
		want  bool
	}{
		{name: "autonomous", ch: nil, cmd: "", room: 8118, level: 5},
		{name: "non-movement", ch: player, cmd: "look", room: 8118, level: 5},
		{name: "missing-player", ch: nil, cmd: "north", room: 8118, level: 5},
		{name: "immortal", ch: player, cmd: "north", room: 8118, level: lvlImmort},
		{name: "wrong-room", ch: player, cmd: "north", room: 8117, level: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ch != nil {
				tt.ch.SetRoom(tt.room)
				tt.ch.SetLevel(tt.level)
			}
			if got := specJailGuard(w, tt.ch, mob, tt.cmd, "ignored"); got != tt.want {
				t.Errorf("handled = %v, want %v", got, tt.want)
			}
			if got := lastMsg(); got != "" {
				t.Errorf("output = %q, want empty", got)
			}
		})
	}
}

// TestSpecJailGuard_Audience pins C's TO_ROOM substitution and actor
// exclusion alongside its separate TO_CHAR warning (src/spec_procs.c:1791-1796).
func TestSpecJailGuard_Audience(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 8118)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 29)
	_ = lastMsg() // discard the spawn announcement

	player.SetRoom(8118)
	mob.SetRoom(8118)
	player.SetLevel(5)
	observer.SetRoom(8118)

	transcript := make(map[string]string)
	w.MessageSink = func(playerName string, msg []byte) {
		transcript[playerName] += string(msg)
	}

	if got := specJailGuard(w, player, mob, "north", "ignored"); !got {
		t.Fatal("north in the holding cell should be intercepted")
	}

	if got, want := transcript[player.GetName()], "The guard stops you from leaving with one flabby hand.\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	if got, want := transcript[observer.GetName()], "The guard grabs "+player.GetName()+" with one hand and throws him back in the room.\r\n"; got != want {
		t.Errorf("observer output = %q, want %q", got, want)
	}
}
