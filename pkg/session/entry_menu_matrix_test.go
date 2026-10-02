package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

// src/interpreter.c:1721,1942-1988,2165-2350. Ordinary secret mismatch
// excludes the pending DES/bcrypt equivalence decision.
func TestEntryMenuMatrix(t *testing.T) {
	for _, suffix := range []string{"", "suffix", " trailing "} {
		t.Run(suffix, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charName = "Cipher"
			hash, err := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
			if err != nil {
				t.Fatal(err)
			}
			s.charPassword = string(hash)
			s.showMainMenu()
			_ = drainMsg(t, s)
			sendMenuInput(t, s, "\t4"+suffix)
			_, p := unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "\r\nEnter your old password: " || !p.Secret {
				t.Fatalf("old: %+v", p)
			}
			sendMenuInput(t, s, " \toldpass")
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "\r\nEnter a new password: " || !p.Secret {
				t.Fatalf("new: %+v", p)
			}
			for _, bad := range []string{"", "ab", "12345678901", "cIpHeR"} {
				sendMenuInput(t, s, bad)
				_, p = unmarshalCharCreate(t, drainMsg(t, s))
				if p.Prompt != "\r\nIllegal password.\r\nPassword: " || !p.Secret || s.menuStage != "password_new" {
					t.Fatalf("gate %q: %+v", bad, p)
				}
			}
			sendMenuInput(t, s, "  abc ")
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "\r\nPlease retype password: " || !p.Secret {
				t.Fatalf("confirm: %+v", p)
			}
			sendMenuInput(t, s, "abc")
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "\r\nPasswords don't match... start over.\r\nPassword: " || s.menuStage != "password_new" {
				t.Fatalf("mismatch: %+v", p)
			}
			sendMenuInput(t, s, "  newpass")
			_ = drainMsg(t, s)
			sendMenuInput(t, s, "\tnewpass")
			if got := entryMenuText(t, s); got != "\r\n\r\n\r\nDone.\n\r" {
				t.Fatalf("done: %q", got)
			}
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Stage != "menu" || !s.passwordMatches("newpass") || s.passwordMatches("oldpass") {
				t.Fatal("password install/menu")
			}
			sendMenuInput(t, s, " 5"+suffix)
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "\r\nEnter your password for verification: " || !p.Secret {
				t.Fatalf("delete: %+v", p)
			}
			sendMenuInput(t, s, "bad")
			if got := entryMenuText(t, s); got != "\r\n\r\nIncorrect password.\r\n" {
				t.Fatalf("delete denial: %q", got)
			}
			_ = drainMsg(t, s)
			sendMenuInput(t, s, "0"+suffix)
			_, p = unmarshalCharCreate(t, drainMsg(t, s))
			if p.Prompt != "Goodbye.\r\n" || !s.SendClosed() {
				t.Fatalf("exit: %+v", p)
			}
		})
	}
	for _, choice := range []string{"yes", "YES", "Yes", "yEs", "yes ", " YES", ""} {
		t.Run("delete/"+choice, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.player = game.NewPlayer(1, "Frozen", 1001)
			s.player.SetPlrFlag(game.PlrFrozen, true)
			s.menuActive = true
			s.menuStage = "delete_confirm"
			sendMenuInput(t, s, choice)
			accepted := choice == "yes" || choice == "YES" || choice == " YES"
			if s.SendClosed() != accepted {
				t.Fatalf("delete %q closes=%v, want %v", choice, s.SendClosed(), accepted)
			}
			got := entryMenuText(t, s)
			want := "\r\nCharacter not deleted.\r\n"
			if accepted {
				want = "You try to kill yourself, but the ice stops you.\r\nCharacter not deleted.\r\n"
			}
			if got != want {
				t.Fatalf("delete %q: %q", choice, got)
			}
		})
	}
}

func TestEntryMenuBackgroundPager(t *testing.T) {
	for _, long := range []bool{false, true} {
		t.Run(map[bool]string{false: "short", true: "paged"}[long], func(t *testing.T) {
			m := makeTestManager(t)
			dir := t.TempDir()
			text := "Background\r\n"
			if long {
				text = strings.Repeat("History\r\n", 50)
			}
			if err := os.WriteFile(filepath.Join(dir, "background"), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			m.world.LibTextDir = dir
			setTeditTestCache(t, "background", "")
			s := makeCharSession(t, m)
			s.player = game.NewPlayer(1, "Reader", 1001)
			s.menuActive = true
			s.menuStage = "menu"
			s.terminalNamed = true
			sendMenuInput(t, s, " 3story")
			if s.menuStage != "background" || s.IsPaging() != long {
				t.Fatalf("background: stage=%s paging=%v", s.menuStage, s.IsPaging())
			}
			drainAllFrames(t, s)
			if long {
				s.TerminalLine("q")
				if s.IsPaging() || s.menuStage != "background" {
					t.Fatal("pager quit must leave CON_RMOTD")
				}
				drainAllFrames(t, s)
			}
			sendMenuInput(t, s, "")
			_, p := unmarshalCharCreate(t, drainMsg(t, s))
			if p.Stage != "menu" || p.Prompt != menuText {
				t.Fatalf("background return: %+v", p)
			}
		})
	}
}

func entryMenuText(t *testing.T, s *Session) string {
	t.Helper()
	var msg struct {
		Type string `json:"type"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(drainMsg(t, s), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != MsgEvent && msg.Type != MsgText {
		t.Fatalf("expected text/event: %+v", msg)
	}
	return msg.Data.Text
}

func TestEntryMenuStoredPassword(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Cipher")
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Cipher", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	drainAllFrames(t, s)
	sendMenuInput(t, s, "")
	drainAllFrames(t, s)
	sendMenuInput(t, s, "4change")
	drainAllFrames(t, s)
	sendMenuInput(t, s, " oraclepass")
	drainAllFrames(t, s)
	sendMenuInput(t, s, " newpass")
	drainAllFrames(t, s)
	sendMenuInput(t, s, "newpass")
	drainAllFrames(t, s)
	rec, err := database.GetPlayer("CIPHER")
	if err != nil || rec == nil {
		t.Fatalf("record: %+v %v", rec, err)
	}
	if bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte("newpass")) != nil || bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte("oraclepass")) == nil {
		t.Fatal("stored password did not change")
	}
	if !s.menuActive || s.menuStage != "menu" {
		t.Fatal("password change entered world")
	}
	if _, ok := s.manager.world.GetPlayer("Cipher"); ok {
		t.Fatal("menu player admitted to world")
	}
}
