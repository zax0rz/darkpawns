package session

import (
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
