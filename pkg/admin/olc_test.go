package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type fakeOLCReadStateProvider struct {
	claims []olc.ClaimEntry
	dirty  []olc.DirtyEntry
}

func (f *fakeOLCReadStateProvider) GetLiveAgentSessions() []LiveAgentSession { return nil }

func (f *fakeOLCReadStateProvider) EnableDecisionCapture() bool { return false }

func (f *fakeOLCReadStateProvider) DisableDecisionCapture() {}

func (f *fakeOLCReadStateProvider) DecisionCaptureEnabled() bool { return false }

func (f *fakeOLCReadStateProvider) DecisionCaptureAvailable() bool { return false }

func (f *fakeOLCReadStateProvider) GetOLCClaims() []olc.ClaimEntry { return f.claims }

func (f *fakeOLCReadStateProvider) GetOLCDirtyZones() []olc.DirtyEntry { return f.dirty }

func newOLCTestDatabase(t *testing.T, level, zone int) *db.DB {
	t.Helper()
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "olc.db"))
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	player := &db.PlayerRecord{
		Name:      "TestPlayer",
		Level:     level,
		OlcZone:   zone,
		Inventory: []byte("[]"),
		Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(player); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	return database
}

func newOLCTestWorld(t *testing.T) *game.World {
	t.Helper()
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "First Room", Zone: 1, Exits: map[string]parser.Exit{}},
			{VNum: 1002, Name: "Second Room", Zone: 1, Exits: map[string]parser.Exit{}},
		},
		Mobs: []parser.Mob{{VNum: 2001, ShortDesc: "a guard"}},
		Objs: []parser.Obj{{VNum: 3001, Keywords: "sword"}},
		Zones: []parser.Zone{{
			Number:  1,
			Name:    "Test Zone",
			TopRoom: 2000,
			Commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 2001, Arg3: 1001},
				{Command: "G", Arg1: 3001},
				{Command: "M", Arg1: 2001, Arg3: 1002},
			},
		}},
	})
	if err != nil {
		t.Fatalf("game.NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	return world
}

func newOLCTestRouter(t *testing.T, level, zone int, state LiveSessionProvider) http.Handler {
	t.Helper()
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin-store.json"))
	database := newOLCTestDatabase(t, level, zone)
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, state)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return handler
}

func doOLCTestRequest(t *testing.T, handler http.Handler, path string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+generateTestToken(t, "builder"))
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestOLCGateAuthorizationBoundaryAndUnauthenticated(t *testing.T) {
	handler := newOLCTestRouter(t, 32, 2, nil)

	for _, path := range []string{
		"/admin/olc/room/1001/preview",
		"/admin/olc/held",
		"/admin/olc/pending",
	} {
		if rec := doOLCTestRequest(t, handler, path, false); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s unauthenticated status = %d, want 401; body: %s", path, rec.Code, rec.Body.String())
		}
	}

	rec := doOLCTestRequest(t, handler, "/admin/olc/room/1001/preview", true)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("outside-zone status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
	var refusal struct {
		Error string `json:"error"`
		Zone  int    `json:"zone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("unmarshal refusal: %v", err)
	}
	if refusal.Zone != 1 || !strings.Contains(refusal.Error, "OLC zone") {
		t.Fatalf("refusal = %+v, want zone 1", refusal)
	}
}

func TestOLCGateLevel35IsUnconfined(t *testing.T) {
	handler := newOLCTestRouter(t, 35, 2, nil)
	rec := doOLCTestRequest(t, handler, "/admin/olc/room/1001/preview", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("level 35 outside-zone status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestOLCPreviewAndZoneCommandView(t *testing.T) {
	handler := newOLCTestRouter(t, 31, 1, nil)
	rec := doOLCTestRequest(t, handler, "/admin/olc/zone/1001/preview", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("zone preview status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Kind string `json:"kind"`
		VNum int    `json:"vnum"`
		Zone struct {
			Commands []parser.ZoneCommand `json:"commands"`
		} `json:"zone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal zone preview: %v", err)
	}
	if body.Kind != "zone" || body.VNum != 1001 {
		t.Fatalf("zone preview identity = %+v", body)
	}
	if len(body.Zone.Commands) != 2 || body.Zone.Commands[0].Command != "M" || body.Zone.Commands[1].Command != "G" {
		t.Fatalf("zone commands = %+v, want M/G room-scoped view", body.Zone.Commands)
	}
}

func TestOLCReadLists(t *testing.T) {
	state := &fakeOLCReadStateProvider{
		claims: []olc.ClaimEntry{{
			Kind:             olc.KindRoom,
			Number:           1001,
			OwnerIdentity:    "builder-1",
			OwnerDisplayName: "Builder",
			OwnerFrontend:    olc.FrontendTelnet,
		}},
		dirty: []olc.DirtyEntry{{Kind: olc.KindRoom, Zone: 1}},
	}
	handler := newOLCTestRouter(t, 35, 99, state)

	for _, test := range []struct {
		path string
		want string
	}{
		{path: "/admin/olc/held", want: "builder-1"},
		{path: "/admin/olc/pending", want: "\"zone\":1"},
	} {
		t.Run(test.path, func(t *testing.T) {
			rec := doOLCTestRequest(t, handler, test.path, true)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), test.want) {
				t.Fatalf("body = %s, want substring %q", rec.Body.String(), test.want)
			}
		})
	}
}
