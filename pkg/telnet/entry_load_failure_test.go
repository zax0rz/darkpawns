package telnet

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

func TestEntryRestoreFailureTelnet(t *testing.T) {
	database := listenerEntryDatabase(t)
	listenerEntrySeed(t, database)
	saved, err := database.GetPlayer("Aiko")
	if err != nil {
		t.Fatal(err)
	}
	saved.CharacterData = []byte(`{"skills":{"track":"bad"}}`)
	if err := database.SavePlayer(saved); err != nil {
		t.Fatal(err)
	}
	before, err := database.GetPlayer("Aiko")
	if err != nil {
		t.Fatal(err)
	}
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
	readTelnetUntil(t, conn, "By what name do you wish to be known? ")
	if _, err := fmt.Fprint(conn, "AIKO\r\n"); err != nil {
		t.Fatal(err)
	}
	if got := readNextLoginPrompt(t, conn); got != "Password: " {
		t.Fatalf("saved corrupt name: %q", got)
	}
	if _, err := fmt.Fprint(conn, "oraclepass\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("failed restore did not close TCP: %v", err)
	}
	text := string(stripTelnetCommands(raw))
	if !strings.Contains(text, "restore character: decode character data:") || strings.Contains(text, "PRESS RETURN") {
		t.Fatalf("failed restore response: %q", text)
	}
	if _, ok := world.GetPlayer("Aiko"); ok {
		t.Fatal("failed telnet restore entered world")
	}
	after, err := database.GetPlayer("Aiko")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("telnet failure rewrote corrupt row")
	}
}
