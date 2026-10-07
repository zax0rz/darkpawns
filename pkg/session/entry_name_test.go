package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// C src/interpreter.c:1721,1743-1759,1505-1520,853-876.
func TestEntryNameGateMatrix(t *testing.T) {
	names := []struct {
		raw, want     string
		valid, closed bool
	}{
		{"", "", false, true},
		{"   ", "", false, true},
		{"A", "", false, false},
		{"Ab", "Ab", true, false},
		{strings.Repeat("x", 20), "X" + strings.Repeat("x", 19), true, false},
		{strings.Repeat("x", 21), "", false, false},
		{"  aIkO", "AIkO", true, false},
		{"Aiko ", "", false, false},
		{"Aiko B", "", false, false},
		{"Fighter123", "", false, false},
		{"a_b", "", false, false},
		{"12345", "", false, false},
		{"Ai-ko", "", false, false},
		// process_input drops the non-ASCII bytes before the name gate
		// (src/comm.c:1974), so C parses "iko".
		{"Åiko", "Iko", true, false},
	}
	for _, word := range []string{"in", "from", "with", "the", "on", "at", "to", "a", "an", "self", "me", "all", "room", "someone", "something"} {
		names = append(names, struct {
			raw, want     string
			valid, closed bool
		}{strings.ToUpper(word), "", false, false})
	}
	for _, route := range []string{"json", "retry", "terminal"} {
		for _, tc := range names {
			t.Run(route+"/"+tc.raw, func(t *testing.T) {
				s := makeCharSession(t, makeTestManager(t))
				switch route {
				case "json":
					// Through the transport boundary, as a WebSocket frame arrives.
					frame, _ := json.Marshal(ClientMessage{Type: MsgLogin, Data: loginMsg(tc.raw, "")})
					err := s.handleMessage(frame)
					if err != nil && !tc.closed {
						t.Fatal(err)
					}
				case "retry":
					s.charCreating = true
					s.charStage = "get_name"
					if err := entryInput(s, tc.raw); err != nil && !tc.closed {
						t.Fatal(err)
					}
				case "terminal":
					if open := s.TerminalLine(tc.raw); open == tc.closed {
						t.Fatalf("open=%v closed want=%v", open, tc.closed)
					}
				}
				if s.SendClosed() != tc.closed {
					t.Fatalf("closed=%v want %v", s.SendClosed(), tc.closed)
				}
				if tc.closed {
					if raw, ok := drainSend(s); ok && len(raw) > 0 {
						t.Fatalf("C closes without text: %q", raw)
					}
					return
				}
				_, p := unmarshalCharCreate(t, drainMsg(t, s))
				if tc.valid {
					want := "Please remember to choose an appropriate fantasy-oriented name.\r\nDid I get that right, " + tc.want + " (Y/N)? "
					if s.charStage != "confirm_name" || s.charName != tc.want || p.Prompt != want {
						t.Fatalf("accepted: stage=%q name=%q prompt=%q", s.charStage, s.charName, p.Prompt)
					}
				} else if s.charStage != "get_name" || s.charName != "" || p.Prompt != "Invalid name, please try another.\r\nName: " {
					t.Fatalf("rejected: stage=%q name=%q prompt=%q", s.charStage, s.charName, p.Prompt)
				}
			})
		}
	}
}

type entryNameLookupStore struct {
	db.GameStore
	names []string
}

func (d *entryNameLookupStore) GetPlayer(name string) (*db.PlayerRecord, error) {
	d.names = append(d.names, name)
	return d.GameStore.GetPlayer(name)
}

func TestEntryNameGateBeforeSavedLookup(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	for _, name := range []string{"the", "Aiko ", "a_b", strings.Repeat("a", 21)} {
		d := &entryNameLookupStore{GameStore: database}
		s := entrySession(t, d)
		if err := s.handleLogin(loginMsg(name, "")); err != nil {
			t.Fatal(err)
		}
		if len(d.names) != 0 || s.charStage != "get_name" {
			t.Fatalf("%q: lookup=%v stage=%q", name, d.names, s.charStage)
		}
	}
	d := &entryNameLookupStore{GameStore: database}
	s := entrySession(t, d)
	if err := s.handleLogin(loginMsg("  AIKO", "")); err != nil {
		t.Fatal(err)
	}
	_, p := unmarshalCharCreate(t, drainMsg(t, s))
	if len(d.names) != 1 || d.names[0] != "AIKO" || s.charName != "Aiko" || p.Prompt != "Password: " || !p.Secret {
		t.Fatalf("saved route: lookup=%v name=%q prompt=%+v", d.names, s.charName, p)
	}
}

// DP-1378 approves this exact overlay; it is not C name parity.
func TestEntrySecurityReservedNames(t *testing.T) {
	for _, name := range []string{"admin", "system", "root", "server", "null", "undefined", "gm", "moderator", "god", "implementor", "imp", "staff", "dev", "bot", "agent", "zax0rz"} {
		t.Run(name, func(t *testing.T) {
			if !validation.IsReservedPlayerName(strings.ToUpper(name)) {
				t.Fatalf("approved overlay list lost %q", name)
			}
			s := makeCharSession(t, makeTestManager(t))
			if err := s.handleLogin(loginMsg(strings.ToUpper(name), "")); err != nil {
				t.Fatal(err)
			}
			_, p := unmarshalCharCreate(t, drainMsg(t, s))
			if s.charStage != "get_name" || p.Prompt != "Invalid name, please try another.\r\nName: " || s.authenticated {
				t.Fatalf("reserved %q admitted", name)
			}
		})
	}
}

// C src/ban.c:266-269 allows CON_PLAYING and rejects creation descriptors.
func TestEntryNameDescriptorOwnership(t *testing.T) {
	m := makeTestManager(t)
	owner := makeCharSession(t, m)
	if err := owner.handleLogin(loginMsg("Aiko", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, owner)
	rejected := makeCharSession(t, m)
	if err := rejected.handleLogin(loginMsg("AIKO", "")); err != nil {
		t.Fatal(err)
	}
	if rejected.charStage != "get_name" {
		t.Fatal("creation-held name accepted")
	}
	if err := entryInput(owner, "N"); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(rejected, "aiko"); err != nil {
		t.Fatal(err)
	}
	if rejected.charStage != "confirm_name" {
		t.Fatal("name not released after N")
	}
	rejected.CloseSend()
	afterClose := makeCharSession(t, m)
	if err := afterClose.handleLogin(loginMsg("aiko", "")); err != nil {
		t.Fatal(err)
	}
	if afterClose.charStage != "confirm_name" {
		t.Fatal("name not released after close")
	}
	afterClose.CloseSend()
	playing := makeTestSession(t, m, "Aiko", 1001, true)
	m.sessions["Aiko"] = playing
	reconnect := makeCharSession(t, m)
	if err := reconnect.handleLogin(loginMsg("AIKO", "")); err != nil {
		t.Fatal(err)
	}
	if reconnect.charStage != "confirm_name" {
		t.Fatal("playing descriptor cannot reconnect")
	}
	reconnect.CloseSend()
	playing.menuActive = true
	menu := makeCharSession(t, m)
	if err := menu.handleLogin(loginMsg("Aiko", "")); err != nil {
		t.Fatal(err)
	}
	if menu.charStage != "get_name" {
		t.Fatal("menu descriptor treated as playing")
	}
}

func TestEntryNameBannedSubstringAndPlayingOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "text"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "text", "xnames"), []byte("bad\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	game.SetBanFilePaths(root)
	t.Cleanup(func() {
		game.SetBanFilePaths("lib")
		if err := game.ReadInvalidList(); err != nil {
			t.Error(err)
		}
	})
	m := makeTestManager(t)
	for _, name := range []string{"Badwolf", "AbaDorn"} {
		s := makeCharSession(t, m)
		if err := s.handleLogin(loginMsg(name, "")); err != nil {
			t.Fatal(err)
		}
		if s.charStage != "get_name" {
			t.Fatalf("banned substring admitted: %q", name)
		}
	}
	control := makeCharSession(t, m)
	if err := control.handleLogin(loginMsg("Goodwolf", "")); err != nil {
		t.Fatal(err)
	}
	if control.charStage != "confirm_name" {
		t.Fatal("nonmatching name rejected")
	}
	live := makeTestSession(t, m, "Badwolf", 1001, true)
	m.sessions["Badwolf"] = live
	reconnect := makeCharSession(t, m)
	if err := reconnect.handleLogin(loginMsg("BADWOLF", "")); err != nil {
		t.Fatal(err)
	}
	if reconnect.charStage != "confirm_name" {
		t.Fatal("C playing descriptor did not override invalid list")
	}
}

// DP-1379 approves guest-prefix entry; command scope is deliberately unchanged.
func TestEntryGuestPrefixApproved(t *testing.T) {
	database := entryDatabase(t)
	manager := entryTransportManager(t, database)
	t.Setenv("JWT_SECRET", "entry-guest-approved-secret-at-least-32-characters")
	// Typed suffixes are never used: they bypass the C name gate, and alias
	// files are keyed by name, so a suffix could carry path separators,
	// control bytes or a lookalike identity.
	for _, name := range []string{"guest", "GuEsT", "guest_test_user", "Guest123!", "guest/../../../tmp/x", "guest\x1b[31mAiko", "GuestAiko"} {
		s := makeCharSession(t, manager)
		if err := s.handleLogin(loginMsg(name, "")); err != nil {
			t.Fatal(err)
		}
		if !s.isGuest || !s.authenticated || s.player == nil || s.player.GetRoom() != game.MortalStartRoom || s.charCreating {
			t.Fatalf("guest path lost for %q", name)
		}
		if !strings.HasPrefix(s.playerName, "Guest_") || s.player.Name != s.playerName {
			t.Fatalf("guest %q kept a typed name: session=%q player=%q", name, s.playerName, s.player.Name)
		}
		for i := len("Guest_"); i < len(s.playerName); i++ {
			if c := s.playerName[i]; c < '0' || c > '9' {
				t.Fatalf("guest %q name has a non-generated suffix: %q", name, s.playerName)
			}
		}
		if s.player.Class != game.ClassWarrior || s.player.Health != 100 || s.player.MaxMana != 20 || s.player.MaxMove != 100 {
			t.Fatal("guest initial state changed")
		}
	}
	if n, err := database.CountPlayers(); err != nil || n != 0 {
		t.Fatalf("guest persisted: n=%d err=%v", n, err)
	}
}

func TestEntryNameWorldHandoffReleasesClaim(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	m := entryTransportManager(t, database)
	s := makeCharSession(t, m)
	if err := s.handleLogin(loginMsg("Biko", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, s)
	for _, input := range []string{"Y", "oraclepass", "oraclepass", "N", "M", "K", "T", "K", "Y", "", "1"} {
		if err := entryInput(s, input); err != nil {
			t.Fatal(err)
		}
	}
	if !s.authenticated || s.charCreating || s.menuActive || s.player == nil {
		t.Fatal("creation did not enter world")
	}
	next := makeCharSession(t, m)
	if err := next.handleLogin(loginMsg("BIKO", "")); err != nil {
		t.Fatal(err)
	}
	if next.charStage != "login_password" || next.charName != "Biko" {
		t.Fatal("playing creator still holds an entry claim")
	}
}

func TestEntryNameConcurrentClaims(t *testing.T) {
	m := makeTestManager(t)
	owners := []*Session{makeCharSession(t, m), makeCharSession(t, m)}
	var wg sync.WaitGroup
	for i, s := range owners {
		wg.Add(1)
		go func(i int, s *Session) {
			defer wg.Done()
			name := []string{"Aiko", "AIKO"}[i]
			if err := s.handleLogin(loginMsg(name, "")); err != nil {
				t.Error(err)
			}
		}(i, s)
	}
	wg.Wait()
	accepted := 0
	for _, s := range owners {
		if s.charStage == "confirm_name" {
			accepted++
		} else if s.charStage != "get_name" {
			t.Fatalf("unexpected losing state %q", s.charStage)
		}
		s.CloseSend()
	}
	if accepted != 1 {
		t.Fatalf("casefolded concurrent owners=%d want 1", accepted)
	}
}

func TestEntryNamePasswordKeepsDescriptorIdentity(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	d := &entryNameLookupStore{GameStore: database}
	s := entrySession(t, d)
	if err := s.handleLogin(loginMsg("AIKO", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, s)
	if err := s.handleLogin(loginMsg("a_b", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if len(d.names) != 2 || d.names[1] != "Aiko" || !s.authenticated || s.player == nil || s.player.Name != "Aiko" {
		t.Fatalf("password changed descriptor identity: lookups=%v player=%v", d.names, s.player)
	}
}

func TestEntryGuestHandoffReleasesName(t *testing.T) {
	database := entryDatabase(t)
	m := entryTransportManager(t, database)
	t.Setenv("JWT_SECRET", "entry-guest-handoff-secret-at-least-32-characters")
	s := makeCharSession(t, m)
	if err := s.handleLogin(loginMsg("Freedname", "")); err != nil {
		t.Fatal(err)
	}
	if s.charStage != "confirm_name" {
		t.Fatal("C entry name was not held")
	}
	if err := s.handleLogin(loginMsg("guest", "")); err != nil {
		t.Fatal(err)
	}
	if !s.authenticated || !s.isGuest {
		t.Fatal("approved guest path did not enter")
	}
	next := makeCharSession(t, m)
	if err := next.handleLogin(loginMsg("Freedname", "")); err != nil {
		t.Fatal(err)
	}
	if next.charStage != "confirm_name" {
		t.Fatal("guest world entry left the former C name held")
	}
}
