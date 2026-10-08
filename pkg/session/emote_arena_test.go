package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/act.wizard.c:144-151 calls shared act, whose arena hook is
// src/comm.c:2529-2556. Exercise command dispatch and the terminal sink.
func TestEmoteCommandArenaBroadcast(t *testing.T) {
	m := makeTestManager(t)
	m.world.GetRoomInWorld(1001).Flags = []string{"134217728"} // ROOM_ARENA, parsed C bitvector
	actor := makeTestSession(t, m, "Actor", 1001, true)
	local := makeTestSession(t, m, "Local", 1001, true)
	remote := makeTestSession(t, m, "Remote", 1002, true)
	for _, s := range []*Session{actor, local, remote} {
		s.player.Stats.Int = 10
		s.player.CopyBaseAttributes()
		registerTestSession(t, m, s, s.playerName)
	}
	emote := func() {
		t.Helper()
		if err := ExecuteCommand(actor, "emote", []string{"waves."}); err != nil {
			t.Fatal(err)
		}
	}
	const broadcast = "&RBroadcast: Actor waves.&n\r\n"
	emote()
	if got := renderedOutput(remote); got != broadcast {
		t.Fatalf("remote command output = %q, want %q", got, broadcast)
	}
	if got := renderedOutput(local); got != broadcast+"Actor waves.\r\n" {
		t.Fatalf("local broadcast then act = %q", got)
	}
	if got := renderedOutput(actor); got != "Actor waves.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	m.world.ExecGenTog(remote.player, "nobroadcast")
	renderedOutput(remote)
	emote()
	if got := renderedOutput(remote); got != "" {
		t.Fatalf("NOBROAD command output = %q", got)
	}
	renderedOutput(local)
	renderedOutput(actor)
	m.world.ExecGenTog(remote.player, "nobroadcast")
	renderedOutput(remote)
	remote.player.SetPosition(game.PosSleeping)
	actor.player.SetPlrFlag(game.PrfNoRepeat, true)
	emote()
	if got := renderedOutput(remote); got != broadcast {
		t.Fatalf("sleeping command output = %q", got)
	}
	if got := renderedOutput(actor); got != "Okay.\r\n" {
		t.Fatalf("NOREPEAT output = %q", got)
	}
}
