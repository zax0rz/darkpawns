package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/fileedit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C src/interpreter.c:764 gates tedit at GRGOD; src/tedit.c:49-50 then checks
// the field. The dedicated page uses these same routes, not a second write door.
func TestMOTDFileEditCLevelAndConditionalLiveSave(t *testing.T) {
	for _, level := range []int{game.LVL_HIGOD, game.LVL_GRGOD - 1, game.LVL_GRGOD} {
		t.Run(fileEditLevelLabel(level), func(t *testing.T) {
			f := newFileEditFixture(t, level)
			for _, name := range []string{"motd", "imotd"} {
				if err := os.WriteFile(filepath.Join(f.world.LibTextDir, name), []byte("Old welcome\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			listing := f.do(http.MethodGet, "/admin/files/tedit/listing", nil, nil)
			var body fileListingResponse
			if err := json.Unmarshal(listing.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			for _, e := range body.Entries {
				if e.Name == "motd" || e.Name == "imotd" {
					if e.RequiredLevel != game.LVL_GRGOD || e.Allowed != (level >= game.LVL_GRGOD) {
						t.Fatalf("wrong permission: %+v", e)
					}
				}
			}
			var savedFile, live string
			fileedit.RegisterTextSavedHook(func(_ *game.World, filename, text string) { savedFile, live = filename, text })
			defer fileedit.RegisterTextSavedHook(nil)
			for _, name := range []string{"motd", "imotd"} {
				target := contentURL("tedit", name)
				read := f.do(http.MethodGet, target, nil, nil)
				if read.Code != http.StatusOK {
					t.Fatalf("read = %d", read.Code)
				}
				etag := read.Header().Get("ETag")
				saved := f.do(http.MethodPut, target, map[string]string{"If-Match": etag}, ptr("&CWelcome&n\r\n"))
				if level < game.LVL_GRGOD {
					if saved.Code != http.StatusForbidden || live != "" {
						t.Fatalf("lower-level save = %d, live = %q", saved.Code, live)
					}
					continue
				}
				if saved.Code != http.StatusOK || savedFile != name || live != "&CWelcome&n\n" {
					t.Fatalf("live save %s: %d %q", name, saved.Code, live)
				}
				if disk, err := os.ReadFile(filepath.Join(f.world.LibTextDir, name)); err != nil || string(disk) != live {
					t.Fatalf("disk = %q, %v", disk, err)
				}
				stale := f.do(http.MethodPut, target, map[string]string{"If-Match": etag}, ptr("stale draft\n"))
				if stale.Code != http.StatusPreconditionFailed {
					t.Fatalf("stale save = %d", stale.Code)
				}
				if savedFile != name || live != "&CWelcome&n\n" {
					t.Fatal("stale save changed live text")
				}
				long := f.do(http.MethodPut, target, map[string]string{"If-Match": saved.Header().Get("ETag")}, ptr(strings.Repeat("x", 2399)+"\n"))
				if long.Code != http.StatusUnprocessableEntity {
					t.Fatalf("over-limit = %d", long.Code)
				}
			}
		})
	}
}

func TestMOTDSavesUseFileEditAudit(t *testing.T) {
	setJWTSecret(t)
	world := newOLCTestWorld(t)
	world.LibTextDir = t.TempDir()
	database := newOLCTestDatabase(t, game.LVL_GRGOD, 1)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "store.json"))
	path := filepath.Join(t.TempDir(), "audit.log")
	logger, err := audit.NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	handler, err := NewRouter(world, logger, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &fileEditFixture{t: t, world: world, handler: handler, token: generateTestToken(t, "builder")}
	for _, name := range []string{"motd", "imotd"} {
		if err := os.WriteFile(filepath.Join(world.LibTextDir, name), []byte("Old\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		target := contentURL("tedit", name)
		read := f.do(http.MethodGet, target, nil, nil)
		saved := f.do(http.MethodPut, target, map[string]string{"If-Match": read.Header().Get("ETag")}, ptr("Welcome\n"))
		if saved.Code != http.StatusOK {
			t.Fatalf("save = %d %s", saved.Code, saved.Body.String())
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"file_edit_save", "tedit:motd", "tedit:imotd"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing audit %q in %s", want, raw)
		}
	}
}
