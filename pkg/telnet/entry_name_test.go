package telnet

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// C-backed rejection/prompt/lookup expectations at the real TCP boundary.
func TestEntryNameTelnetBoundary(t *testing.T) {
	database := listenerEntryDatabase(t)
	listenerEntrySeed(t, database)
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	manager := session.NewManager(world, database)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	readTelnetUntil(t, conn, "By what name do you wish to be known?")
	for _, name := range []string{"Aiko ", "Fighter123", "the"} {
		if _, err := fmt.Fprintf(conn, "%s\r\n", name); err != nil {
			t.Fatal(err)
		}
		if got := readNextLoginPrompt(t, conn); got != "Invalid name, please try another.\r\nName: " {
			t.Fatalf("%q: prompt=%q", name, got)
		}
	}
	if _, err := fmt.Fprint(conn, "  AIKO\r\n"); err != nil {
		t.Fatal(err)
	}
	if got := readNextLoginPrompt(t, conn); got != "Password: " {
		t.Fatalf("leading/casefold route=%q", got)
	}
	if n, err := database.CountPlayers(); err != nil || n != 1 {
		t.Fatalf("name rejection changed rows: n=%d err=%v", n, err)
	}
	blank, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blank.Close() }()
	readTelnetUntil(t, blank, "By what name do you wish to be known?")
	if _, err := fmt.Fprint(blank, "   \r\n"); err != nil {
		t.Fatal(err)
	}
	if err := blank.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var one [1]byte
	if n, err := blank.Read(one[:]); n != 0 || err == nil {
		t.Fatalf("empty-name close emitted text: n=%d err=%v byte=%q", n, err, one)
	}
}
