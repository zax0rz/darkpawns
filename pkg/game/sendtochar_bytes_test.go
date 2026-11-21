package game

import "testing"

// TestEnterCircleLookBytesMatchC pins the enter-circle look output bytes. C
// builds one line across several send_to_char calls (src/spec_procs.c:1964-1975):
//
//	send_to_char("Looking into the circle at the platform in the middle"
//	             " of the room, you see\n\r", ch);
//	send_to_char("no one", ch);      /* fragment: no terminator */
//	send_to_char(".\n\r\n\r", ch);
//
// The first line ends in LFCR, and the "no one"/name fragment carries no
// line ending. Each fragment must reach the player byte-for-byte.
func TestEnterCircleLookBytesMatchC(t *testing.T) {
	w, actor, sink := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 10)

	if !specEnterCircle(w, actor, mob, "look", "circle") {
		t.Fatal("specEnterCircle(look circle) returned false")
	}

	got := sink()
	want := "Looking into the circle at the platform in the middle of the room, you see\n\r" +
		"no one" +
		".\n\r\n\r"
	if got != want {
		t.Errorf("enter-circle look bytes = %q, want %q", got, want)
	}
}
