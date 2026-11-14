package telnet

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1529 (divergent-approved): C's ident toggle makes the next connection
// print "Please wait." while it queries the client's identd on port 113
// (src/ident.c:68). Zach ruled the lookup obsolete and a security and privacy
// surface, so the port keeps the toggle's command bytes but never prints the
// notice or connects out. With ident on, a fresh connection goes straight to
// the name prompt.
func TestIdentToggleOnSkipsLookupNotice(t *testing.T) {
	waitForNoConnections(t)
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	god := game.NewCharacter(1, "Identgod", game.ClassWarrior, game.RaceHuman)
	god.SetLevel(game.LVL_IMPL)
	world.ExecGenTog(god, nil, "ident") // ident defaults off; this turns it on
	t.Cleanup(func() { world.ExecGenTog(god, nil, "ident") })

	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)

	addr := ipv4ListenerAddr(t, func() net.Addr { return listener.Addr() })
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	var got []byte
	buf := make([]byte, 4096)
	for !bytes.Contains(got, []byte("By what name do you wish to be known?")) {
		n, err := conn.Read(buf)
		got = append(got, buf[:n]...)
		if err != nil {
			t.Fatalf("no name prompt before %v; got %q", err, got)
		}
	}
	if bytes.Contains(got, []byte("Please wait.")) {
		t.Fatalf("ident notice printed despite DP-1529: %q", got)
	}
}
