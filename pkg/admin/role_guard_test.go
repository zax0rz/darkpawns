package admin

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// L2-030: a demotion must take effect on the next authenticated request —
// the token's role claim is a login-time hint, never the authority — and a
// deleted account loses the panel entirely. Promotions keep needing a fresh
// login (the outer claim check applies).
func TestRoleGuardDemotionTakesEffectImmediately(t *testing.T) {
	setJWTSecret(t)
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "guard.db"))
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	record := &db.PlayerRecord{Name: "Demotedme", Password: "x", Level: 40}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}

	called := false
	handler := roleGuard(database)("admin", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	request := func() *httptest.ResponseRecorder {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/admin/mudlog", nil)
		req.Header.Set(adminCSRFHeader, "1")
		claims := &auth.Claims{PlayerName: "Demotedme", Role: "admin"}
		req = req.WithContext(auth.SetClaimsOnContext(req.Context(), claims))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	if rec := request(); rec.Code != http.StatusOK || !called {
		t.Fatalf("level-40 admin request = %d (called=%v), want 200", rec.Code, called)
	}

	// Demote to a mortal level: the same token, on the very next request,
	// must be refused by the re-resolved role.
	fetched, err := database.GetPlayer("Demotedme")
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	fetched.Level = 1
	if err := database.SavePlayer(fetched); err != nil {
		t.Fatalf("SavePlayer: %v", err)
	}
	if rec := request(); rec.Code != http.StatusForbidden || called {
		t.Fatalf("demoted request = %d (called=%v), want 403", rec.Code, called)
	}

	// A deleted account loses the panel entirely, even with a valid token.
	if err := database.DeletePlayer(fetched.ID); err != nil {
		t.Fatalf("DeletePlayer: %v", err)
	}
	if rec := request(); rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("deleted-account request = %d (called=%v), want 401", rec.Code, called)
	}
}

// A soft-deleted character's record lingers until purge with its old level;
// the guard must treat it as gone (the deleted-marker path, not row removal).
func TestRoleGuardDeletedCharacterLosesPanel(t *testing.T) {
	setJWTSecret(t)
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "guard-deleted.db"))
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	p := game.NewCharacter(0, "Tombstone", game.ClassThief, game.RaceKender)
	p.Level = 40
	p.SetPlrFlag(game.PlrDeleted, true)
	record, err := db.PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord: %v", err)
	}
	record.Password = "x"
	if err := database.CreatePlayer(record); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}

	called := false
	handler := roleGuard(database)("admin", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/mudlog", nil)
	req.Header.Set(adminCSRFHeader, "1")
	claims := &auth.Claims{PlayerName: "Tombstone", Role: "admin"}
	req = req.WithContext(auth.SetClaimsOnContext(req.Context(), claims))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("soft-deleted admin request = %d (called=%v), want 401", rec.Code, called)
	}
}
