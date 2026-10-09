package game

import "testing"

// TestEnterCircleLookBytesMatchC pins the sendToChar append class at the game
// layer for the enter-circle special, whose C original builds one output line
// across several send_to_char calls (src/spec_procs.c:1964-1975):
//
//	send_to_char("Looking into the circle at the platform in the middle"
//	             " of the room, you see\n\r", ch);
//	send_to_char("no one", ch);      /* fragment: no terminator */
//	send_to_char(".\n\r\n\r", ch);
//
// The "no one"/name fragment carries no line ending, so it must reach the
// player verbatim. A sendToChar append on the fragment inserts a line break
// the C oracle never wrote.
func TestEnterCircleLookBytesMatchC(t *testing.T) {
	w, actor, sink := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 10)

	if !specEnterCircle(w, actor, mob, "look", "circle") {
		t.Fatal("specEnterCircle(look circle) returned false")
	}

	got := sink()
	// CRLF-normalized C bytes: leading line, the fragment, then the tail.
	want := "Looking into the circle at the platform in the middle of the room, you see\r\n" +
		"no one" +
		".\r\n\r\n"
	if got != want {
		t.Errorf("enter-circle look bytes = %q, want %q", got, want)
	}
}
