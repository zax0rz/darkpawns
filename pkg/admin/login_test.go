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
