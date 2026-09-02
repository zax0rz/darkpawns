package game

import "testing"

func TestDoRoflSelfTargetUsesCCharAutoBytes(t *testing.T) {
	w, actor, _, _, output := newChannelWorld(t)

	DoAction(w, actor, "rofl", actor.Name)

	want := "$n rolls on the floor laughing at $mself.\r\n"
	if got := channelOutput(output, actor.Name); got != want {
		t.Fatalf("self-target output = %q, want literal C char_auto bytes %q", got, want)
	}
}
