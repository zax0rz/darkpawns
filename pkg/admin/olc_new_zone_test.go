package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestOLCSchemaSurfacesNewZoneCapability(t *testing.T) {
	for _, test := range []struct {
		name    string
		level   int
		allowed bool
	}{
		{name: "below HIGOD", level: olc.NewZoneRequiredLevel - 1},
		{name: "HIGOD", level: olc.NewZoneRequiredLevel, allowed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := newOLCTestRouter(t, test.level, 1, nil)
			rec := doOLCTestRequest(t, handler, "/admin/olc/schema/zone", true)
			if rec.Code != http.StatusOK {
				t.Fatalf("schema status = %d; body: %s", rec.Code, rec.Body.String())
			}
			var body struct {
				Actions []struct {
					Key           string `json:"key"`
					Allowed       bool   `json:"allowed"`
					RequiredLevel int    `json:"required_level"`
				} `json:"actions"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode schema: %v", err)
			}
			if len(body.Actions) != 1 || body.Actions[0].Key != "create_zone" {
				t.Fatalf("actions = %+v", body.Actions)
			}
			if body.Actions[0].RequiredLevel != olc.NewZoneRequiredLevel || body.Actions[0].Allowed != test.allowed {
				t.Fatalf("new-zone capability = %+v; want level %d allowed %v", body.Actions[0], olc.NewZoneRequiredLevel, test.allowed)
			}
		})
	}
}

func TestOLCNewZoneRequiresHIGODAndCreatesCFiles(t *testing.T) {
	world := newOLCTestWorld(t)
	world.WorldPath = t.TempDir()
	for _, extension := range []string{"zon", "wld", "mob", "obj", "shp"} {
		directory := filepath.Join(world.WorldPath, extension)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", extension, err)
		}
		if err := os.WriteFile(filepath.Join(directory, "index"), []byte("1."+extension+"\n$\n"), 0o644); err != nil {
			t.Fatalf("write %s index: %v", extension, err)
		}
	}

	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin-store.json"))
	database := newOLCTestDatabase(t, olc.NewZoneRequiredLevel-1, 1)
	lowHandler, err := NewRouter(world, nil, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatalf("NewRouter low-level: %v", err)
	}
	low := doOLCJSONRequest(t, lowHandler, http.MethodPost, "/admin/olc/zones/50", nil)
	if low.Code != http.StatusForbidden {
		t.Fatalf("low-level status = %d; body: %s", low.Code, low.Body.String())
	}
	_ = database.Close()

	database = newOLCTestDatabase(t, olc.NewZoneRequiredLevel, 1)
	t.Cleanup(func() { _ = database.Close() })
	highHandler, err := NewRouter(world, nil, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatalf("NewRouter HIGOD: %v", err)
	}
	high := doOLCJSONRequest(t, highHandler, http.MethodPost, "/admin/olc/zones/50", nil)
	if high.Code != http.StatusOK {
		t.Fatalf("HIGOD status = %d; body: %s", high.Code, high.Body.String())
	}
	for _, file := range olc.NewZoneFiles(50) {
		path := filepath.Join(world.WorldPath, file.Extension, "50."+file.Extension)
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		if string(contents) != file.Contents {
			t.Fatalf("contents of %s = %q, want %q", path, contents, file.Contents)
		}
	}
	if _, ok := world.SnapshotZone(50); !ok {
		t.Fatal("created zone is missing from the live world")
	}
}
