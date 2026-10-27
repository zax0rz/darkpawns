//go:build adminpreview

package admin

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/notify"
)

type consolePreviewSessions struct{ *fakeOLCWriteStateProvider }

func (*consolePreviewSessions) ReserveOfflinePlayer(string) (func(), bool) { return func() {}, true }

// TestConsolePreview is an opt-in, disposable real-API preview for visual QA.
// Run with -tags adminpreview -run '^TestConsolePreview$' -timeout 0.
// All mutable data belongs to t.TempDir; no production credentials are used.
func TestConsolePreview(t *testing.T) {
	if os.Getenv("ADMIN_PREVIEW") != "1" {
		t.Skip("set ADMIN_PREVIEW=1 to serve the fixture console")
	}
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	world := newOLCTestWorld(t)
	world.StopAITicker()
	// Fill optional fixture fields before serving; real parsed records include them.
	if mob, ok := world.GetMobPrototype(2001); ok {
		mob.Keywords = "guard"
		mob.LongDesc = "A guard stands here."
		mob.ActionFlags = []string{"SENTINEL"}
		mob.AffectFlags = []string{}
	}
	if obj, ok := world.GetObjPrototype(3001); ok {
		obj.ShortDesc = "a steel sword"
		obj.LongDesc = "A steel sword lies here."
	}
	for _, vnum := range []int{1001, 1002} {
		if room := world.GetRoomInWorld(vnum); room != nil {
			room.Flags = []string{"0", "0", "0", "0"}
			room.Description = "A room in the synthetic console test world."
		}
	}
	world.ScriptsDir = t.TempDir()
	world.LibTextDir = t.TempDir()
	for path, body := range map[string]string{filepath.Join(world.ScriptsDir, "guard.lua"): "return true\n", filepath.Join(world.LibTextDir, "news"): "Local console preview. Synthetic test world.\n", filepath.Join(world.LibTextDir, "credits"): "Dark Pawns console fixture.\n"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	online := game.NewPlayer(1, "TestPlayer", 1001)
	online.Level = game.LVL_IMPL
	if err := world.AddPlayer(online); err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{Name: "Returner", Level: 10, Health: 10, MaxHealth: 20, Mana: 10, MaxMana: 20, Move: 10, MaxMove: 20, StatStr: 15, StatCon: 15, StatInt: 15, StatWis: 15, StatDex: 15, StatCha: 15, Inventory: []byte(`[]`), Equipment: []byte(`[]`), CharacterData: []byte(`{"save_version":2}`)}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	service, err := notify.New(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	logs := NewLogBuffer(20)
	if _, err := fmt.Fprintln(logs, "INFO 2026-10-27T12:00:00-04:00 Local console preview started (synthetic fixture)"); err != nil {
		t.Fatal(err)
	}
	doc := apidoc.New()
	handler, err := NewRouter(world, nil, logs, database, &consolePreviewSessions{newFakeOLCWriteStateProvider()}, WithSharedSpec(doc), WithNotifications(service))
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	mux := http.NewServeMux()
	apidoc.RegisterHealth(doc.NewAPI(mux))
	mux.HandleFunc("/api/openapi.json", doc.Handler())
	mux.HandleFunc("/preview-session", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: token, Path: "/admin", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		w.WriteHeader(http.StatusNoContent)
	})
	var active atomic.Value
	active.Store(handler)
	mux.HandleFunc("/preview-reset", func(w http.ResponseWriter, r *http.Request) {
		next, err := NewRouter(world, nil, logs, database, &consolePreviewSessions{newFakeOLCWriteStateProvider()}, WithNotifications(service))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		active.Store(next)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { active.Load().(http.Handler).ServeHTTP(w, r) })
	// Parallel agents each need their own fixture: 4350 stays the default, and
	// ADMIN_PREVIEW_PORT lets a second run take another port.
	port := os.Getenv("ADMIN_PREVIEW_PORT")
	if port == "" {
		port = "4350"
	}
	t.Fatal(http.ListenAndServe("127.0.0.1:"+port, mux))
}
