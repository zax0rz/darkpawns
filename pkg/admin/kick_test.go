package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

type fakeKicker struct {
	online    map[string]bool
	gotName   string
	gotNotice string
}

func (f *fakeKicker) Kick(playerName string, notice string) bool {
	f.gotName, f.gotNotice = playerName, notice
	return f.online[playerName]
}

func adminContext(role string) context.Context {
	return auth.SetClaimsOnContext(context.Background(), &auth.Claims{PlayerName: "Admin", Role: role})
}

func serveKick(t *testing.T, kicker sessionKicker, role string) *httptest.ResponseRecorder {
	t.Helper()
	handler := handlePlayerDetail(testWorld(t), nil, kicker)
	req := httptest.NewRequest(http.MethodPost, "/admin/players/TestPlayer/kick", nil).WithContext(adminContext(role))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHandlePlayerKick(t *testing.T) {
	kicker := &fakeKicker{online: map[string]bool{"TestPlayer": true}}
	rec := serveKick(t, kicker, "admin")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "kicked" {
		t.Fatalf("status field = %q, want %q", body["status"], "kicked")
	}
	if kicker.gotName != "TestPlayer" {
		t.Fatalf("kicked name = %q, want TestPlayer", kicker.gotName)
	}
	if !strings.Contains(kicker.gotNotice, "disconnected by an administrator") {
		t.Fatalf("notice = %q, want the administrator disconnect notice", kicker.gotNotice)
	}
}

func TestHandlePlayerKick_OfflineIsNotFound(t *testing.T) {
	kicker := &fakeKicker{online: map[string]bool{}}
	rec := serveKick(t, kicker, "admin")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for an offline player", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "player not online") {
		t.Fatalf("body = %q, want 'player not online'", rec.Body.String())
	}
}

func TestHandlePlayerKick_RequiresAdminRole(t *testing.T) {
	kicker := &fakeKicker{online: map[string]bool{"TestPlayer": true}}
	rec := serveKick(t, kicker, "builder")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a builder-role token", rec.Code)
	}
	if kicker.gotName != "" {
		t.Fatalf("builder role reached the kicker for %q", kicker.gotName)
	}
}
