package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// comm.c:2140-2143: a descriptor that has not heard from the name prompt has no
// character, so close_socket has none to name. The producer is once per
// descriptor even though Go closes a session from both its transport and the
// manager.
func TestLoseDescriptorWithoutChar(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	watcher := mudlogObserver(t, m, "Closewatch", game.LVL_IMMORT, game.PrfLog1, game.PrfLog2)
	file := captureMudlogFile(t)

	s := makeCharSession(t, m)
	s.LoseDescriptor()
	want := "[ Losing descriptor without char. ]\r\n"
	if got := strings.Join(drainSessionText(t, watcher), ""); got != want {
		t.Fatalf("observer = %q, want %q", got, want)
	}
	if !strings.Contains(file.String(), "Losing descriptor without char.") {
		t.Fatalf("file = %q", file.String())
	}
	s.LoseDescriptor()
	if got := strings.Join(drainSessionText(t, watcher), ""); got != "" {
		t.Fatalf("second close repeated the producer: %q", got)
	}
}

// comm.c:2136-2138 with C's CON_MENU '0', which reaches CON_CLOSE while the
// descriptor still holds the character (interpreter.c:2168-2170): the line
// names it. The character cannot be named from the entry name here, because
// completeCharCreation clears it — the attached body supplies it.
func TestLosePlayerOnMenuExit(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	watcher := mudlogObserver(t, m, "Closewatch", game.LVL_IMMORT, game.PrfLog1, game.PrfLog2)
	file := captureMudlogFile(t)

	s := makeCharSession(t, m)
	s.player = game.NewPlayer(1, "Returner", 1001)
	s.authenticated = true
	s.menuActive = true
	s.menuStage = "menu"
	s.terminalNamed = true
	s.markDescriptorBound()
	s.charName = ""
	sendMenuInput(t, s, "0")

	want := "[ Losing player: Returner. ]\r\n"
	if got := strings.Join(drainSessionText(t, watcher), ""); got != want {
		t.Fatalf("observer = %q, want %q", got, want)
	}
	if !strings.Contains(file.String(), "Losing player: Returner.") {
		t.Fatalf("file = %q", file.String())
	}
}

// comm.c:2136-2138 on the frozen delete arm, which also sets CON_CLOSE
// (interpreter.c:2322-2325). Its own self-delete line is a separate producer.
func TestLosePlayerOnFrozenDelete(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	watcher := mudlogObserver(t, m, "Closewatch", game.LVL_IMMORT, game.PrfLog1, game.PrfLog2)

	s := makeCharSession(t, m)
	s.player = game.NewPlayer(1, "Frozen", 1001)
	s.player.SetPlrFlag(game.PlrFrozen, true)
	s.authenticated = true
	s.menuActive = true
	s.menuStage = "delete_confirm"
	s.terminalNamed = true
	s.markDescriptorBound()
	sendMenuInput(t, s, "yes")

	if !s.SendClosed() {
		t.Fatal("frozen delete did not close")
	}
	want := "[ Losing player: Frozen. ]\r\n"
	if got := strings.Join(drainSessionText(t, watcher), ""); got != want {
		t.Fatalf("observer = %q, want %q", got, want)
	}
}

// comm.c:2140-2143 with perform_dupe_check: C clears k->character before it
// sets CON_CLOSE (interpreter.c:1573-1576), so the displaced menu descriptor
// closes without a character.
func TestLoseDescriptorWithoutCharOnDuplicateMenu(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	watcher := mudlogObserver(t, m, "Closewatch", game.LVL_IMMORT, game.PrfLog1, game.PrfLog2)

	old := makeCharSession(t, m)
	old.player = game.NewPlayer(1, "Returner", 1001)
	old.authenticated = true
	old.menuActive = true
	old.menuStage = "menu"
	if err := m.Register("Returner", old); err != nil {
		t.Fatal(err)
	}
	fresh := makeCharSession(t, m)
	fresh.player = game.NewPlayer(1, "Returner", 1001)
	if fresh.performDupeCheck() {
		t.Fatal("menu descriptor supplied a live body")
	}

	want := "[ Losing descriptor without char. ]\r\n"
	if got := strings.Join(drainSessionText(t, watcher), ""); got != want {
		t.Fatalf("observer = %q, want %q", got, want)
	}
}

// comm.c:2140-2143 for the displaced descriptors: unswitch, usurp and an OLC
// editor all have k->character cleared before CON_CLOSE
// (interpreter.c:1551-1576). A RECON target is a linkless body with no
// descriptor at all, so C's loop never reaches close_socket for it.
func TestLoseDescriptorWithoutCharOnDisplacedDescriptor(t *testing.T) {
	for _, mode := range []string{"reconnect", "usurp", "unswitch", "editor"} {
		t.Run(mode, func(t *testing.T) {
			_, old, observer, fresh := dupeCheckFixture(t)
			observer.player.SetLevel(game.LVL_IMMORT)
			observer.player.SetPlrFlag(game.PrfLog1, true)
			observer.player.SetPlrFlag(game.PrfLog2, true)
			drainSessionText(t, observer)
			switch mode {
			case "reconnect":
				old.player.SetLinkless(true)
				old.DetachTransport()
			case "unswitch":
				old.isSwitched = true
			case "editor":
				old.textEdit = &textEditState{}
			}
			if !fresh.performDupeCheck() {
				t.Fatal("dupe check found no target")
			}
			got := strings.Join(drainSessionText(t, observer), "")
			count := strings.Count(got, "Losing descriptor without char.")
			if mode == "reconnect" {
				if count != 0 {
					t.Fatalf("a linkless descriptor produced a close line: %q", got)
				}
				return
			}
			if count != 1 || !strings.Contains(got, "[ Losing descriptor without char. ]\r\n") {
				t.Fatalf("observer = %q, want exactly one descriptor-loss line", got)
			}
		})
	}
}

// TestLoginStateDisconnectsMatchCCloseSocket pins C's close_socket bytes and
// side effects (src/comm.c:2121-2143) across every login state in nanny
// (src/interpreter.c:1743-2352): CON_GET_NAME through CON_MENU.
// Oracle-unreachable class (2026-11-11 rule; DP-1560): dp-oracle-diff requires
// setup/peer connections to enter the game before comparing blocks.
func TestLoginStateDisconnectsMatchCCloseSocket(t *testing.T) {
	type loginCase struct {
		name      string
		existing  bool
		inputs    []string
		wantLog   string
		checkName string
	}

	cases := []loginCase{
		{
			name:    "CON_GET_NAME_before_input",
			inputs:  nil,
			wantLog: "Losing descriptor without char.",
		},
		{
			name:    "CON_GET_NAME_invalid_name",
			inputs:  []string{"123"},
			wantLog: "Losing player: <null>.",
		},
		{
			name:      "CON_NAME_CNFRM",
			inputs:    []string{"LoginDisc"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_PASSWORD",
			existing:  true,
			inputs:    []string{"Returner"},
			wantLog:   "Losing player: Returner.",
			checkName: "Returner",
		},
		{
			name:      "CON_NEWPASSWD",
			inputs:    []string{"LoginDisc", "y"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_CNFPASSWD",
			inputs:    []string{"LoginDisc", "y", "pass123"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_QSEX",
			inputs:    []string{"LoginDisc", "y", "pass123", "pass123"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_QRACE",
			inputs:    []string{"LoginDisc", "y", "pass123", "pass123", "m"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_QCLASS",
			inputs:    []string{"LoginDisc", "y", "pass123", "pass123", "m", "h"},
			wantLog:   "Losing player: LoginDisc.",
			checkName: "LoginDisc",
		},
		{
			name:      "CON_RMOTD",
			existing:  true,
			inputs:    []string{"Returner", "oraclepass"},
			wantLog:   "Losing player: Returner.",
			checkName: "Returner",
		},
		{
			name:      "CON_MENU",
			existing:  true,
			inputs:    []string{"Returner", "oraclepass", ""},
			wantLog:   "Losing player: Returner.",
			checkName: "Returner",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			database := entryDatabase(t)
			if tc.existing {
				entrySeed(t, database, "Returner")
			}
			m := entryTransportManager(t, database)
			watcher := mudlogObserver(t, m, "Closewatch", game.LVL_IMMORT, game.PrfLog1, game.PrfLog2)
			file := captureMudlogFile(t)
			drainSessionText(t, watcher)

			s := makeCharSession(t, m)
			for _, in := range tc.inputs {
				if !s.TerminalLine(in) {
					t.Fatalf("TerminalLine(%q) closed session unexpectedly", in)
				}
			}
			drainSessionText(t, watcher)

			// Simulate transport disconnect (pkg/telnet/listener.go:500,513).
			s.LoseDescriptor()
			s.Close()
			m.UnregisterSession(s)

			gotObs := strings.Join(drainSessionText(t, watcher), "")
			wantObs := "[ " + tc.wantLog + " ]\r\n"
			if gotObs != wantObs {
				t.Fatalf("observer = %q, want %q", gotObs, wantObs)
			}
			if !strings.Contains(file.String(), tc.wantLog) {
				t.Fatalf("file log = %q, want %q", file.String(), tc.wantLog)
			}
			if strings.Contains(gotObs, "Closing link to:") {
				t.Fatalf("non-playing disconnect emitted Closing link to: %q", gotObs)
			}
			if tc.checkName != "" {
				if _, ok := m.world.GetPlayer(tc.checkName); ok {
					t.Fatalf("non-playing disconnect left %q in world", tc.checkName)
				}
				if _, ok := m.GetSession(tc.checkName); ok {
					t.Fatalf("non-playing disconnect left %q registered in manager", tc.checkName)
				}
			}
			// Second close is a no-op (C's close_socket runs once per descriptor).
			s.LoseDescriptor()
			if again := strings.Join(drainSessionText(t, watcher), ""); again != "" {
				t.Fatalf("second LoseDescriptor repeated output: %q", again)
			}
		})
	}
}
