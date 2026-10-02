package session

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// DP-1380 approves full-secret comparison, unlike C DES's first eight bytes
// (src/interpreter.c:1873,1964,2292,2305). Both positive and suffix controls
// run through each actual entry comparison, not a bcrypt-only assertion.
func TestEntryFullPasswordComparison(t *testing.T) {
	const secret = "12345678ab"
	const changed = "12345678cd"
	t.Run("creation-confirm", func(t *testing.T) {
		s := makeCharSession(t, makeTestManager(t))
		s.charCreating, s.charStage, s.charName = true, "create_password", "Hero"
		sendCharInput(t, s, secret)
		drainAllFrames(t, s)
		sendCharInput(t, s, changed)
		drainAllFrames(t, s)
		if s.charStage != "create_password" {
			t.Fatal("suffix mismatch accepted during creation")
		}
		sendCharInput(t, s, secret)
		drainAllFrames(t, s)
		sendCharInput(t, s, secret)
		drainAllFrames(t, s)
		if s.charStage != "color" {
			t.Fatal("full matching secret rejected")
		}
	})
	for _, stage := range []string{"password_old", "delete_password", "password_confirm"} {
		t.Run(stage, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charName = "Hero"
			hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
			if err != nil {
				t.Fatal(err)
			}
			s.charPassword = string(hash)
			s.menuNewPasswordHash = string(hash)
			s.menuActive = true
			s.menuStage = stage
			sendMenuInput(t, s, changed)
			drainAllFrames(t, s)
			if s.menuStage != "menu" && s.menuStage != "password_new" {
				t.Fatal("suffix mismatch accepted", s.menuStage)
			}
			s.menuStage = stage
			s.menuNewPasswordHash = string(hash)
			sendMenuInput(t, s, secret)
			drainAllFrames(t, s)
			want := map[string]string{"password_old": "password_new", "delete_password": "delete_confirm", "password_confirm": "menu"}[stage]
			if s.menuStage != want {
				t.Fatal("full matching secret rejected", s.menuStage)
			}
		})
	}
	t.Run("stored-login", func(t *testing.T) {
		database := entryDatabase(t)
		record := entrySeed(t, database, "Hero")
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		if err := database.UpdatePassword(record.ID, string(hash)); err != nil {
			t.Fatal(err)
		}
		s := entrySession(t, database)
		if err := s.handleLogin(loginMsg("Hero", changed)); err != nil {
			t.Fatal(err)
		}
		if s.authenticated || s.player != nil {
			t.Fatal("suffix mismatch authenticated")
		}
		drainAllFrames(t, s)
		if err := s.handleLogin(loginMsg("Hero", secret)); err != nil {
			t.Fatal(err)
		}
		if !s.authenticated || s.player == nil || s.menuStage != "motd" {
			t.Fatal("full matching secret rejected")
		}
	})
}
