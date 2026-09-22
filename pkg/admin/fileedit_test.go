package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/fileedit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type forgetRecorder struct{ calls int }

func (f *forgetRecorder) RunScript(*game.ScriptContext, string, string) (bool, error) {
	return false, nil
}
func (f *forgetRecorder) ForgetFailures() { f.calls++ }

type fileEditFixture struct {
	t       *testing.T
	world   *game.World
	handler http.Handler
	token   string
}

func newFileEditFixture(t *testing.T, level int) *fileEditFixture {
	t.Helper()
	setJWTSecret(t)
	world := newOLCTestWorld(t)
	world.ScriptsDir = t.TempDir()
	world.LibTextDir = t.TempDir()
	for _, dir := range []string{"mob", "room"} {
		if err := os.MkdirAll(filepath.Join(world.ScriptsDir, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{
		"mob/guard.lua":  "return true\n",
		"room/guard.lua": "return 'shadowed'\n",
	} {
		if err := os.WriteFile(filepath.Join(world.ScriptsDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"news", "credits"} {
		if err := os.WriteFile(filepath.Join(world.LibTextDir, name), []byte("Old "+name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	database := newOLCTestDatabase(t, level, 1)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin-store.json"))
	handler, err := NewRouter(world, nil, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &fileEditFixture{t: t, world: world, handler: handler, token: generateTestToken(t, "builder")}
}

func (f *fileEditFixture) do(method, target string, headers map[string]string, content *string) *httptest.ResponseRecorder {
	f.t.Helper()
	var body *bytes.Reader
	if content == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, _ := json.Marshal(map[string]string{"content": *content})
		body = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func contentURL(root, path string) string {
	return "/admin/files/" + root + "/content?path=" + url.QueryEscape(path)
}

func ptr(s string) *string { return &s }

func TestFileEditRoundTripsAndConflicts(t *testing.T) {
	f := newFileEditFixture(t, game.LVL_HIGOD)
	read := f.do(http.MethodGet, contentURL("lua", "mob/guard.lua"), nil, nil)
	etag := read.Header().Get("ETag")
	if read.Code != http.StatusOK || etag != fileedit.ETag([]byte("return true\n")) {
		t.Fatalf("read = %d etag=%q body=%s", read.Code, etag, read.Body.String())
	}
	recorder := &forgetRecorder{}
	previousEngine := game.ScriptEngine
	game.ScriptEngine = recorder
	t.Cleanup(func() { game.ScriptEngine = previousEngine })

	broken := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), map[string]string{"If-Match": etag}, ptr("local x = 1\nfunction broken(\n"))
	if broken.Code != http.StatusUnprocessableEntity || !strings.Contains(broken.Body.String(), "line 2 (end of file)") {
		t.Fatalf("broken Lua = %d %q; want 422 naming the line", broken.Code, broken.Body.String())
	}
	midFile := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), map[string]string{"If-Match": etag}, ptr("local y = 1\nx = = 2\n"))
	if midFile.Code != http.StatusUnprocessableEntity || !strings.Contains(midFile.Body.String(), "line 2, column 5") {
		t.Fatalf("mid-file error = %d %q", midFile.Code, midFile.Body.String())
	}
	if unchanged, _ := os.ReadFile(filepath.Join(f.world.ScriptsDir, "mob", "guard.lua")); string(unchanged) != "return true\n" {
		t.Fatalf("broken Lua touched disk: %q", unchanged)
	}
	if recorder.calls != 0 {
		t.Fatalf("a refused save cleared the failed-scripts cache")
	}

	saved := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), map[string]string{"If-Match": etag}, ptr("return false\n"))
	if saved.Code != http.StatusOK || recorder.calls != 1 {
		t.Fatalf("save = %d, forget calls=%d, body=%s", saved.Code, recorder.calls, saved.Body.String())
	}
	stale := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), map[string]string{"If-Match": etag}, ptr("return true\n"))
	if stale.Code != http.StatusPreconditionFailed || !strings.Contains(stale.Body.String(), "reload to compare") {
		t.Fatalf("stale save = %d %q", stale.Code, stale.Body.String())
	}
	if none := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), nil, ptr("return 1\n")); none.Code != http.StatusPreconditionRequired {
		t.Fatalf("unconditional save = %d; want 428, never last-writer-wins", none.Code)
	}

	created := f.do(http.MethodPut, contentURL("lua", "mob/new.lua"), map[string]string{"If-None-Match": "*"}, ptr("return 1\n"))
	if created.Code != http.StatusOK {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if again := f.do(http.MethodPut, contentURL("lua", "mob/new.lua"), map[string]string{"If-None-Match": "*"}, ptr("return 2\n")); again.Code != http.StatusPreconditionFailed {
		t.Fatalf("create over an existing file = %d; want 412", again.Code)
	}
	del := f.do(http.MethodDelete, contentURL("lua", "mob/new.lua"), map[string]string{"If-Match": created.Header().Get("ETag")}, nil)
	if del.Code != http.StatusNoContent || recorder.calls != 3 {
		t.Fatalf("delete = %d forget calls=%d", del.Code, recorder.calls)
	}
}

// A web tedit save must refresh the live text exactly like the telnet save;
// otherwise players read the old MOTD until reboot.
func TestTeditSaveStripsCRAndRefreshesLiveText(t *testing.T) {
	f := newFileEditFixture(t, game.LVL_HIGOD)
	var gotFile, gotText string
	fileedit.RegisterTextSavedHook(func(_ *game.World, filename, text string) { gotFile, gotText = filename, text })
	t.Cleanup(func() { fileedit.RegisterTextSavedHook(nil) })

	read := f.do(http.MethodGet, contentURL("tedit", "news"), nil, nil)
	saved := f.do(http.MethodPut, contentURL("tedit", "news"), map[string]string{"If-Match": read.Header().Get("ETag")}, ptr("New news\r\nline two\r\n"))
	if saved.Code != http.StatusOK {
		t.Fatalf("tedit save = %d %s", saved.Code, saved.Body.String())
	}
	disk, _ := os.ReadFile(filepath.Join(f.world.LibTextDir, "news"))
	if string(disk) != "New news\nline two\n" {
		t.Fatalf("disk = %q; C's strip_string leaves LF-only", disk)
	}
	if gotFile != "news" || gotText != string(disk) {
		t.Fatalf("live text hook got (%q, %q)", gotFile, gotText)
	}
	tooLong := strings.Repeat("x", 8193)
	if rec := f.do(http.MethodPut, contentURL("tedit", "news"), map[string]string{"If-Match": saved.Header().Get("ETag")}, &tooLong); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("over-limit news = %d; want 422", rec.Code)
	}
	if rec := f.do(http.MethodDelete, contentURL("tedit", "news"), map[string]string{"If-Match": saved.Header().Get("ETag")}, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("tedit delete = %d; want 400", rec.Code)
	}
}

// credits is LVL_IMPL in C's tedit table. The web door is HIGOD, but a HIGOD
// must not reach a file the telnet command would refuse them.
func TestTeditKeepsCFieldLevelAboveTheWebDoor(t *testing.T) {
	f := newFileEditFixture(t, game.LVL_HIGOD)
	listing := f.do(http.MethodGet, "/admin/files/tedit/listing", nil, nil)
	var body fileListingResponse
	if err := json.Unmarshal(listing.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, e := range body.Entries {
		if e.Name == "credits" && (e.Allowed || e.RequiredLevel != game.LVL_IMPL) {
			t.Fatalf("credits entry = %+v; want grayed at IMPL", e)
		}
		if e.Name == "news" && !e.Allowed {
			t.Fatalf("news entry = %+v; want writable at HIGOD", e)
		}
	}
	read := f.do(http.MethodGet, contentURL("tedit", "credits"), nil, nil)
	rec := f.do(http.MethodPut, contentURL("tedit", "credits"), map[string]string{"If-Match": read.Header().Get("ETag")}, ptr("mine now\n"))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "IMPL (40)") {
		t.Fatalf("HIGOD credits write = %d %q", rec.Code, rec.Body.String())
	}
}

func TestFileEditBuilderCanReadButCannotWrite(t *testing.T) {
	f := newFileEditFixture(t, game.LVL_IMMORT)
	read := f.do(http.MethodGet, contentURL("lua", "mob/guard.lua"), nil, nil)
	if read.Code != http.StatusOK {
		t.Fatalf("builder read = %d", read.Code)
	}
	listing := f.do(http.MethodGet, "/admin/files/lua/listing?directory=mob", nil, nil)
	if !strings.Contains(listing.Body.String(), `"required_label":"HIGOD"`) || !strings.Contains(listing.Body.String(), `"allowed":false`) {
		t.Fatalf("builder listing lacks the grayed HIGOD action: %s", listing.Body.String())
	}
	rec := f.do(http.MethodPut, contentURL("lua", "mob/guard.lua"), map[string]string{"If-Match": read.Header().Get("ETag")}, ptr("return false\n"))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "HIGOD (36)") {
		t.Fatalf("builder write = %d %q", rec.Code, rec.Body.String())
	}
}

// Usage attributes a script_name to the one file the engine resolves it to:
// "guard.lua" runs mob/guard.lua (flat, then mob, room, obj), so
// room/guard.lua must not claim the mob.
func TestScriptUsageFollowsEngineResolution(t *testing.T) {
	f := newFileEditFixture(t, game.LVL_IMMORT)
	if !f.world.SetMobScript(2001, "guard.lua", 0) {
		t.Fatal("SetMobScript(2001) failed")
	}
	type usage = []scriptUsage
	get := func(path string) usage {
		rec := f.do(http.MethodGet, "/admin/files/lua/usage?path="+url.QueryEscape(path), nil, nil)
		var out usage
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("usage %s: %d %s", path, rec.Code, rec.Body.String())
		}
		return out
	}
	if got := get("mob/guard.lua"); len(got) != 1 || got[0].Kind != "mob" || got[0].VNum != 2001 {
		t.Fatalf("mob/guard.lua usage = %+v", got)
	}
	if got := get("room/guard.lua"); len(got) != 0 {
		t.Fatalf("room/guard.lua usage = %+v; the engine never runs it for guard.lua", got)
	}

	rec := f.do(http.MethodGet, "/admin/files/lua/resolve?name=guard.lua&kind=room", nil, nil)
	var resolved scriptResolveResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resolved)
	if resolved.Path != "mob/guard.lua" || !resolved.Exists {
		t.Fatalf("resolve guard.lua = %+v", resolved)
	}
	rec = f.do(http.MethodGet, "/admin/files/lua/resolve?name=fresh.lua&kind=obj", nil, nil)
	resolved = scriptResolveResponse{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resolved)
	if resolved.Path != "obj/fresh.lua" || resolved.Exists {
		t.Fatalf("resolve fresh.lua = %+v; want an obj/ creation proposal", resolved)
	}
}
