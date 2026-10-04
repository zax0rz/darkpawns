package session

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestEntryWizlockNewConfirmation(t *testing.T) {
	for _, threshold := range []int{0, 1, 30, 40} {
		for _, terminal := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/terminal=%v", threshold, terminal), func(t *testing.T) {
				database := entryDatabase(t)
				s := entrySession(t, database)
				if terminal {
					s.TerminalLine("Freshhero")
				} else if err := s.handleLogin(loginMsg("Freshhero", "")); err != nil {
					t.Fatal(err)
				}
				_ = renderedOutput(s)
				// Change it after name lookup: C checks at confirmation, not naming.
				s.manager.wizlockLevel = threshold
				if terminal {
					s.TerminalLine("y")
				} else {
					sendCharInput(t, s, "y")
				}
				got := renderedOutput(s)
				if threshold == 0 {
					if s.SendClosed() || s.charStage != "create_password" || !strings.Contains(got, "Give me a password for Freshhero: ") {
						t.Fatalf("open gate: %q stage=%s", got, s.charStage)
					}
				} else if !s.SendClosed() || got != "Sorry, new players can't be created at the moment.\r\n" || s.player != nil || s.authenticated {
					t.Fatalf("closed gate: %q closed=%v candidate=%v", got, s.SendClosed(), s.player)
				}
				if count, err := database.CountPlayers(); err != nil || count != 0 {
					t.Fatalf("created unauthorized record: %d %v", count, err)
				}
			})
		}
	}
}

func TestEntryWizlockReturningThreshold(t *testing.T) {
	for _, threshold := range []int{0, 1, 30, 40} {
		for _, level := range []int{1, 29, 30, 31, 40} {
			for _, terminal := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%d/terminal=%v", threshold, level, terminal), func(t *testing.T) {
					database := entryDatabase(t)
					rec := entrySeed(t, database, "Aiko")
					rec.Level = level
					if err := database.SavePlayer(rec); err != nil {
						t.Fatal(err)
					}
					for range 2 {
						if _, err := database.RecordLoginFailure("Aiko", 100, 0); err != nil {
							t.Fatal(err)
						}
					}
					s := entrySession(t, database)
					s.manager.accountLockouts = nil
					if terminal {
						s.TerminalLine("aiko")
					} else if err := s.handleLogin(loginMsg("aiko", "")); err != nil {
						t.Fatal(err)
					}
					_ = renderedOutput(s)
					s.manager.wizlockLevel = threshold
					if terminal {
						s.TerminalLine("oraclepass")
					} else {
						sendCharInput(t, s, "oraclepass")
					}
					got := renderedOutput(s)
					if level < threshold {
						if !s.SendClosed() || s.authenticated || s.player != nil || got != "\r\nThe game is temporarily restricted.. try again later.\r\n" {
							t.Fatalf("refusal: %q closed=%v authenticated=%v", got, s.SendClosed(), s.authenticated)
						}
						stored, err := database.GetPlayer("Aiko")
						if err != nil || stored.FailedLoginAttempts != 2 || stored.ID != rec.ID {
							t.Fatalf("refusal mutated record: %+v %v", stored, err)
						}
					} else if s.SendClosed() || !s.authenticated || !s.menuActive || s.menuStage != "motd" {
						t.Fatalf("admission failed: %q stage=%s", got, s.charStage)
					}
					if s.manager.world.GetPlayerCount() != 0 {
						t.Fatal("entry registered a body before menu")
					}
				})
			}
		}
	}
}

func TestEntryWizlockWrongPasswordFirst(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	s := entrySession(t, database)
	s.manager.wizlockLevel = 40
	if err := s.handleLogin(loginMsg("Aiko", "wrongpass")); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); strings.Contains(got, "restricted") || !strings.Contains(got, "Wrong password.") || s.SendClosed() {
		t.Fatalf("restriction leaked before password check: %q", got)
	}
}

func TestEntryWizlockWebSocket(t *testing.T) {
	for _, returning := range []bool{false, true} {
		t.Run(fmt.Sprint(returning), func(t *testing.T) {
			database := entryDatabase(t)
			if returning {
				entrySeed(t, database, "Aiko")
			}
			m := entryTransportManager(t, database)
			m.wizlockLevel = 30
			server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
			defer server.Close()
			conn := entryWebSocketDial(t, server.URL)
			defer conn.Close()
			name, answer := "Freshhero", "Y"
			want := "Sorry, new players can't be created at the moment.\r\n"
			if returning {
				name, answer = "Aiko", "oraclepass"
				want = "The game is temporarily restricted.. try again later.\r\n"
			}
			wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": name})
			_ = wsReadUntilType(t, conn, MsgCharCreate)
			wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": answer})
			var prompt CharCreateData
			entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
			if prompt.Stage != "closing" || prompt.Prompt != want {
				t.Fatalf("browser refusal=%+v", prompt)
			}
			if _, _, err := conn.ReadMessage(); err == nil {
				t.Fatal("refused WebSocket remained open")
			}
		})
	}
}

func TestEntryWizlockGuestOverlayUnchanged(t *testing.T) {
	s := entrySession(t, entryDatabase(t))
	s.manager.wizlockLevel = game.LVL_IMPL
	if err := s.handleLogin(loginMsg("guest_wizlock", "")); err != nil {
		t.Fatal(err)
	}
	if !s.isGuest || !s.authenticated || s.SendClosed() {
		t.Fatal("DP-1379 guest overlay changed")
	}
}
