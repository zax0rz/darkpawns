package admin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

// fakeLoginDB implements loginPlayerDB for testing.
type fakeLoginDB struct {
	rec *db.PlayerRecord
	err error
}

func (f *fakeLoginDB) GetPlayer(name string) (*db.PlayerRecord, error) {
	return f.rec, f.err
}

func TestHandleLogin_PlayerNotFound_Returns401(t *testing.T) {
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
		Threshold: 3,
		Lockout:   15 * time.Minute,
	})
	t.Cleanup(tracker.Stop)

	// Simulate GetPlayer returning nil, nil for an unknown player.
	handler := handleLogin(&fakeLoginDB{rec: nil, err: nil}, tracker)

	body := strings.NewReader(`{"player_name":"nobody","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "player not found") {
		t.Errorf("body should not reveal whether the player exists: %s", rec.Body.String())
	}

	testIP := "192.0.2.1"
	if locked, _ := tracker.IsLocked(testIP); locked {
		t.Errorf("a single failure should not lock the IP")
	}
}

func TestHandleLogin_DatabaseError_Returns401(t *testing.T) {
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
		Threshold: 3,
		Lockout:   15 * time.Minute,
	})
	t.Cleanup(tracker.Stop)

	handler := handleLogin(&fakeLoginDB{rec: nil, err: errors.New("db unavailable")}, tracker)

	body := strings.NewReader(`{"player_name":"anyone","password":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestHandleLogin_AuthFailuresAreIndistinguishable pins the property that made
// the earlier responses a user-enumeration oracle: an unknown character, a
// character with no password set, and a real character given the wrong password
// all answered differently, so a stranger could map which names exist by
// reading the reply. Every authentication failure must be byte-identical.
func TestHandleLogin_AuthFailuresAreIndistinguishable(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-horse"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashing test password: %v", err)
	}

	cases := []struct {
		name string
		db   *fakeLoginDB
		body string
	}{
		{
			name: "unknown character",
			db:   &fakeLoginDB{rec: nil, err: nil},
			body: `{"player_name":"nobody","password":"whatever"}`,
		},
		{
			name: "character exists but has no password",
			db:   &fakeLoginDB{rec: &db.PlayerRecord{Name: "Aidan", Password: "", Level: 40}},
			body: `{"player_name":"Aidan","password":"whatever"}`,
		},
		{
			name: "character exists, wrong password",
			db:   &fakeLoginDB{rec: &db.PlayerRecord{Name: "Aidan", Password: string(hash), Level: 40}},
			body: `{"player_name":"Aidan","password":"wrong"}`,
		},
		{
			name: "lookup failed",
			db:   &fakeLoginDB{rec: nil, err: errors.New("database is down")},
			body: `{"player_name":"Aidan","password":"whatever"}`,
		},
	}

	type reply struct {
		code int
		body string
	}
	replies := make([]reply, 0, len(cases))

	for _, tc := range cases {
		tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
			Threshold: 100,
			Lockout:   time.Minute,
		})
		t.Cleanup(tracker.Stop)

		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleLogin(tc.db, tracker).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d (body: %s)", tc.name, rec.Code, rec.Body.String())
		}
		replies = append(replies, reply{code: rec.Code, body: rec.Body.String()})
	}

	first := replies[0]
	for i, got := range replies[1:] {
		if got != first {
			t.Errorf("%q answers differently from %q:\n  %q\n  %q",
				cases[i+1].name, cases[0].name, got.body, first.body)
		}
	}

	// And the shared answer must not name which check failed.
	for _, leak := range []string{"password", "not found", "no such", "exists"} {
		if strings.Contains(strings.ToLower(first.body), leak) {
			t.Errorf("shared failure body mentions %q, which narrows the cause: %s", leak, first.body)
		}
	}
}
