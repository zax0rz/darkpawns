package admin

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

func deletedAdminData(t *testing.T) []byte {
	t.Helper()
	p := game.NewCharacter(1, "TestPlayer", game.ClassWarrior, game.RaceHuman)
	p.SetPlrFlag(game.PlrDeleted, true)
	data, err := game.EncodeCharacterData(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDeletedAdminLoginDenied(t *testing.T) {
	setJWTSecret(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{Threshold: 10})
	t.Cleanup(tracker.Stop)
	for _, record := range []*db.PlayerRecord{nil, {Name: "TestPlayer", Level: game.LVL_GOD, Password: string(hash), CharacterData: deletedAdminData(t)}} {
		response := httptest.NewRecorder()
		handleLogin(&fakeLoginDB{rec: record}, tracker).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"player_name":"TestPlayer","password":"correct-password"}`)))
		if response.Code != http.StatusUnauthorized || response.Body.String() != authFailureBody+"\n" {
			t.Fatalf("deleted/missing login = %d %q", response.Code, response.Body.String())
		}
	}
}

func TestDeletedAdminExistingTokenDenied(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	world := newOLCTestWorld(t)
	world.ScriptsDir, world.LibTextDir = t.TempDir(), t.TempDir()
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	handler, err := NewRouter(world, nil, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The token predates deletion: authorization must re-read the retained row.
	token := generateTestToken(t, "builder")
	record, err := database.GetPlayer("TestPlayer")
	if err != nil {
		t.Fatal(err)
	}
	record.CharacterData = deletedAdminData(t)
	if err := database.SavePlayer(record); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/admin/olc/schema/zone"},
		{http.MethodGet, "/admin/olc/room/100/preview"},
		{http.MethodPost, "/admin/olc/zones/50"},
		{http.MethodGet, "/admin/files/lua/content?path=mob/guard.lua"},
		{http.MethodPut, "/admin/files/lua/content?path=mob/guard.lua"},
	} {
		t.Run(route.method+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"content":"return true"}`))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("deleted token = %d: %s", response.Code, response.Body.String())
			}
		})
	}
}
