package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// R5h: the shared JSON/terminal input handler owns raw password bytes.
// C gate and confirmation: src/interpreter.c:1942-1980; MAX_PWD_LENGTH: src/structs.h:648.
func TestEntryNewPasswordMatrix(t *testing.T) {
	for _, password := range []string{"", "a", "ab", "12345678901", "hErO", "abc", "1234567890", " abc ", "AbC", "\t\vabc", "          abc"} {
		t.Run(password, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charCreating, s.charStage, s.charName = true, "create_password", "Hero"
			sendCharInput(t, s, password)
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			effective := strings.TrimLeft(password, " \t\n\r\v\f")
			valid := len(effective) >= 3 && len(effective) <= 10 && !strings.EqualFold(effective, "Hero")
			want, stage := "\r\nIllegal password.\r\nPassword: ", "create_password"
			if valid {
				want, stage = "\r\nPlease retype password: ", "confirm_password"
			}
			if prompt.Prompt != want || prompt.Stage != stage || !prompt.Secret || s.charStage != stage {
				t.Fatalf("gate: %+v state=%q", prompt, s.charStage)
			}
			if !valid {
				return
			}
			sendCharInput(t, s, "different")
			_, prompt = unmarshalCharCreate(t, drainMsg(t, s))
			if prompt.Prompt != "\r\nPasswords don't match... start over.\r\nPassword: " || !prompt.Secret || s.charStage != "create_password" {
				t.Fatalf("mismatch: %+v", prompt)
			}
			sendCharInput(t, s, password)
			_ = drainMsg(t, s)
			sendCharInput(t, s, password)
			_, prompt = unmarshalCharCreate(t, drainMsg(t, s))
			if prompt.Prompt != "\r\nDo you want ANSI color (Y/N)? " || prompt.Secret || s.charStage != "color" {
				t.Fatalf("confirmation: %+v", prompt)
			}
			if bcrypt.CompareHashAndPassword([]byte(s.charPassword), []byte(effective)) != nil {
				t.Fatal("raw accepted password was not hashed")
			}
		})
	}
}

// Real browser frames must carry the same gate prompts and secret flags.
func TestEntryNewPasswordWebSocket(t *testing.T) {
	m := entryTransportManager(t, entryDatabase(t))
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer server.Close()
	conn := entryWebSocketDial(t, server.URL)
	defer func() { _ = conn.Close() }()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": "Hero"})
	_ = wsReadUntilType(t, conn, MsgCharCreate)
	steps := []struct {
		input, stage, prompt string
		secret               bool
	}{
		{"Y", "create_password", "New character.\r\nGive me a password for Hero: ", true},
		{"ab", "create_password", "\r\nIllegal password.\r\nPassword: ", true},
		{"hERO", "create_password", "\r\nIllegal password.\r\nPassword: ", true},
		{"12345678901", "create_password", "\r\nIllegal password.\r\nPassword: ", true},
		{"  abc ", "confirm_password", "\r\nPlease retype password: ", true},
		{"abc", "create_password", "\r\nPasswords don't match... start over.\r\nPassword: ", true},
		{"abc", "confirm_password", "\r\nPlease retype password: ", true},
		{"  abc", "color", "\r\nDo you want ANSI color (Y/N)? ", false},
	}
	for _, step := range steps {
		wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": step.input})
		var prompt CharCreateData
		entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
		if prompt.Stage != step.stage || prompt.Prompt != step.prompt || prompt.Secret != step.secret {
			t.Fatalf("input %q: %+v", step.input, prompt)
		}
	}
}

// C src/interpreter.c:1871-1893,1929-1937: persistent failures, empty
// echo/close, successful reset, and singular/plural warning at the MOTD.
func TestEntryPasswordAccounting(t *testing.T) {
	for _, overlay := range []bool{false, true} {
		t.Run(map[bool]string{false: "C-only", true: "security-overlay"}[overlay], func(t *testing.T) {
			database := entryDatabase(t)
			entrySeed(t, database, "Aiko")
			for attempt := 1; attempt <= 2; attempt++ {
				s := entrySession(t, database)
				if !overlay {
					s.manager.accountLockouts = nil
				}
				if err := s.handleLogin(loginMsg("aiko", "")); err != nil {
					t.Fatal(err)
				}
				_ = drainMsg(t, s)
				sendCharInput(t, s, "wrongpass")
				_ = drainMsg(t, s)
				_ = drainMsg(t, s)
				rec, err := database.GetPlayer("AIKO")
				if err != nil || rec.FailedLoginAttempts != attempt {
					t.Fatalf("durable failures=%+v err=%v", rec, err)
				}
				s.CloseSend()
				good := entrySession(t, database)
				if !overlay {
					good.manager.accountLockouts = nil
				}
				if attempt == 1 {
					if err := good.handleLogin(loginMsg("aiko", "  oraclepass")); err != nil {
						t.Fatal(err)
					}
					_ = drainMsg(t, good)
					_, prompt := unmarshalCharCreate(t, drainMsg(t, good))
					if !strings.Contains(prompt.Prompt, "\r\n\r\n\007\007\0071 LOGIN FAILURE SINCE LAST SUCCESSFUL LOGIN.\r\n\r\n\n*** PRESS RETURN: ") {
						t.Fatalf("singular warning: %q", prompt.Prompt)
					}
					rec, err = database.GetPlayer("Aiko")
					if err != nil || rec.FailedLoginAttempts != 0 {
						t.Fatalf("success did not reset: %+v %v", rec, err)
					}
					// Prepare one prior failure for the next iteration's plural control.
					if _, err := database.RecordLoginFailure("Aiko", 100, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := good.handleLogin(loginMsg("aiko", "oraclepass")); err != nil {
						t.Fatal(err)
					}
					_ = drainMsg(t, good)
					_, prompt := unmarshalCharCreate(t, drainMsg(t, good))
					if !strings.Contains(prompt.Prompt, "2 LOGIN FAILURES SINCE LAST SUCCESSFUL LOGIN.") {
						t.Fatalf("plural warning: %q", prompt.Prompt)
					}
				}
			}
		})
	}
}

func TestEntryPasswordEmptyEchoClose(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Aiko", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, s)
	sendCharInput(t, s, "   ")
	if raw := drainMsg(t, s); !strings.Contains(string(raw), `"type":"raw"`) || !strings.Contains(string(raw), `"text":"\r\n"`) {
		t.Fatalf("empty password echo: %s", raw)
	}
	if !s.SendClosed() || s.authenticated || s.player != nil {
		t.Fatal("empty password admitted or did not close")
	}
	rec, err := database.GetPlayer("Aiko")
	if err != nil || rec.FailedLoginAttempts != 0 {
		t.Fatalf("empty counted as wrong: %+v %v", rec, err)
	}
}

func TestEntryPasswordRetryWebSocket(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	m := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer server.Close()
	conn := entryWebSocketDial(t, server.URL)
	defer func() { _ = conn.Close() }()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": "aiko"})
	_ = wsReadUntilType(t, conn, MsgCharCreate)
	for attempt := 1; attempt <= 3; attempt++ {
		wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": "wrongpass"})
		kind, raw := entryWebSocketReadOne(t, conn)
		if kind != MsgEvent || !strings.Contains(string(mustMarshalEntry(t, raw)), `"text":"\r\n"`) {
			t.Fatalf("echo before refusal: %s %+v", kind, raw)
		}
		var prompt CharCreateData
		entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
		want := "Wrong password.\r\nPassword: "
		if attempt == 3 {
			want = "Wrong password... disconnecting.\r\n"
		}
		if prompt.Prompt != want || prompt.Secret != (attempt < 3) {
			t.Fatalf("attempt %d: %+v", attempt, prompt)
		}
		rec, err := database.GetPlayer("Aiko")
		if err != nil || rec.FailedLoginAttempts != attempt {
			t.Fatalf("attempt count: %+v %v", rec, err)
		}
	}
}

func mustMarshalEntry(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// C's historical failure counter is an unsigned byte (src/structs.h:978).
// The approved security counter stays unbounded; only the C warning view wraps.
func TestEntryPasswordCounterByteView(t *testing.T) {
	for _, count := range []int{255, 256, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			database := entryDatabase(t)
			entrySeed(t, database, "Aiko")
			if _, err := database.SQLDB().Exec("UPDATE players SET failed_login_attempts = ? WHERE name = ?", count, "Aiko"); err != nil {
				t.Fatal(err)
			}
			s := entrySession(t, database)
			s.manager.accountLockouts = nil
			if err := s.handleLogin(loginMsg("Aiko", "oraclepass")); err != nil {
				t.Fatal(err)
			}
			_ = drainMsg(t, s)
			_, p := unmarshalCharCreate(t, drainMsg(t, s))
			if count == 256 {
				if strings.Contains(p.Prompt, "LOGIN FAILURE") {
					t.Fatalf("C byte wraps to zero: %q", p.Prompt)
				}
			} else {
				want := fmt.Sprintf("%d LOGIN FAILURE", count&255)
				if !strings.Contains(p.Prompt, want) {
					t.Fatalf("warning view %d: %q", count, p.Prompt)
				}
			}
			record, err := database.GetPlayer("Aiko")
			if err != nil || record.FailedLoginAttempts != 0 {
				t.Fatal("successful login failed to reset shared counter")
			}
		})
	}
}
