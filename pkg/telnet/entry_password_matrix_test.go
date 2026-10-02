package telnet

import (
	"fmt"
	"net"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

func TestEntryNewPasswordTelnet(t *testing.T) {
	database := listenerEntryDatabase(t)
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
	steps := []struct{ input, prompt string }{
		{"Hero", "Did I get that right, Hero (Y/N)? "},
		{"Y", "New character.\r\nGive me a password for Hero: "},
		{"ab", "\r\nIllegal password.\r\nPassword: "},
		{"hERO", "\r\nIllegal password.\r\nPassword: "},
		{"12345678901", "\r\nIllegal password.\r\nPassword: "},
		{"  abc ", "\r\nPlease retype password: "},
		{"abc", "\r\nPasswords don't match... start over.\r\nPassword: "},
		{"abc", "\r\nPlease retype password: "},
		{"  abc", "\r\nDo you want ANSI color (Y/N)? "},
	}
	for _, step := range steps {
		if _, err := fmt.Fprintf(conn, "%s\r\n", step.input); err != nil {
			t.Fatal(err)
		}
		got := readTelnetUntil(t, conn, step.prompt)
		if got != step.prompt && step.input != "Hero" {
			t.Fatalf("input %q: %q want %q", step.input, got, step.prompt)
		}
	}
}
