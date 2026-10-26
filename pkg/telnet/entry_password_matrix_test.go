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
	steps := []struct {
		input, prompt string
		secret        bool
	}{
		{"Hero", "Did I get that right, Hero (Y/N)? ", false},
		{"Y", "New character.\r\nGive me a password for Hero: ", true},
		{"ab", "\r\nIllegal password.\r\nPassword: ", true},
		{"hERO", "\r\nIllegal password.\r\nPassword: ", true},
		{"12345678901", "\r\nIllegal password.\r\nPassword: ", true},
		{"  abc ", "\r\nPlease retype password: ", true},
		{"abc", "\r\nPasswords don't match... start over.\r\nPassword: ", true},
		{"abc", "\r\nPlease retype password: ", true},
		{"  abc", "\r\nDo you want ANSI color (Y/N)? ", false},
	}
	echoOn := false
	for _, step := range steps {
		if echoOn {
			if _, err := conn.Write([]byte{IAC, DO, OPT_ECHO}); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := fmt.Fprintf(conn, "%s\r\n", step.input); err != nil {
			t.Fatal(err)
		}
		got, raw := readTelnetUntilRaw(t, conn, step.prompt)
		if got != step.prompt && step.input != "Hero" {
			t.Fatalf("input %q: %q want %q", step.input, got, step.prompt)
		}
		willEcho := countTelnetCommand(raw, WILL, OPT_ECHO)
		wontEcho := countTelnetCommand(raw, WONT, OPT_ECHO)
		wantWill, wantWont := 0, 0
		if step.secret && !echoOn {
			wantWill = 1
		}
		if !step.secret && echoOn {
			wantWont = 1
		}
		if willEcho != wantWill || wontEcho != wantWont {
			t.Fatalf("input %q: ECHO negotiation WILL=%d WONT=%d, want WILL=%d WONT=%d (raw %v)",
				step.input, willEcho, wontEcho, wantWill, wantWont, raw)
		}
		echoOn = step.secret
	}
}
