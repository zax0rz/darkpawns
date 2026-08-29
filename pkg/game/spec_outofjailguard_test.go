package game

import (
	"strings"
	"testing"
)

// TestSpecOutOfJailGuard_EntryGates covers SPECIAL(outofjailguard)'s command,
// player, level, and room gates (src/spec_procs.c:1765-1771).
func TestSpecOutOfJailGuard_EntryGates(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 35)
	_ = lastMsg() // discard the spawn announcement

	player.SetRoom(8117)
	mob.SetRoom(8117)

	tests := []struct {
		name  string
		ch    *Player
		cmd   string
		room  int
		level int
		want  bool
	}{
		{name: "autonomous", ch: nil, cmd: "", room: 8117, level: 5},
		{name: "non-movement", ch: player, cmd: "look", room: 8117, level: 5},
		{name: "missing-player", ch: nil, cmd: "south", room: 8117, level: 5},
		{name: "immortal", ch: player, cmd: "south", room: 8117, level: lvlImmort},
		{name: "wrong-room", ch: player, cmd: "south", room: 8116, level: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ch != nil {
				tt.ch.SetRoom(tt.room)
				tt.ch.SetLevel(tt.level)
			}
			if got := specOutOfJailGuard(w, tt.ch, mob, tt.cmd, "ignored"); got != tt.want {
				t.Errorf("handled = %v, want %v", got, tt.want)
			}
			if got := lastMsg(); got != "" {
				t.Errorf("output = %q, want empty", got)
			}
		})
	}
}

// TestSpecOutOfJailGuard_Audience pins C's TO_ROOM substitution and its
// separate TO_CHAR line, including one CRLF per message. The actor must not
// receive the room audience message (src/spec_procs.c:1773-1778).
func TestSpecOutOfJailGuard_Audience(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 8117)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 35)
	_ = lastMsg() // discard the spawn announcement

	player.SetRoom(8117)
	mob.SetRoom(8117)
	player.SetLevel(5)
	observer.SetRoom(8117)

	transcript := make(map[string]string)
	w.MessageSink = func(playerName string, msg []byte) {
		transcript[playerName] += string(msg)
	}

	if got := specOutOfJailGuard(w, player, mob, "south", ""); !got {
		t.Fatal("south at the jail entrance should be intercepted")
	}

	if got, want := transcript[player.GetName()], "The guard stops you from entering with one quick jerk of your collar.\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	peer := transcript[observer.GetName()]
	if want := "The guard grabs " + player.GetName() + " by the collar and blocks his way.\r\n"; peer != want {
		t.Errorf("observer output = %q, want %q", peer, want)
	}
	if strings.Count(transcript[player.GetName()], "blocks") != 0 {
		t.Error("actor received the TO_ROOM guard message")
	}
}
