package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

func TestActivityEndpointVisibilityFiltersAndStablePagination(t *testing.T) {
	h := telemetry.New(time.Now())
	for _, name := range []string{"Avery", "Mira", "Avery"} {
		telemetry.RecordLogin(telemetry.Activity{Player: name, Class: "Warrior", MinLevel: 31})
	}
	telemetry.RecordLogin(telemetry.Activity{Player: "Hidden", MinLevel: 40})
	telemetry.RecordCharacter(telemetry.Activity{Player: "Newcomer", Class: "Mage", MinLevel: 31})
	telemetry.RecordPlayerDeath(telemetry.Activity{Player: "Avery", Cause: "mob", MobName: "a guard", MobVNum: 300, RoomName: "Arena", RoomVNum: 1001, MinLevel: 31})
	h.Sample(time.Now(), 1)
	mux := http.NewServeMux()
	doc := apidoc.New()
	level := 31
	registerActivity(doc.NewInternalAPI(mux), h, func(context.Context) int { return level })
	get := func(path string) (int, activityResponse) {
		t.Helper()
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
		var body activityResponse
		if rr.Code == 200 {
			if rr.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("activity response is cacheable")
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
		}
		return rr.Code, body
	}
	code, first := get("/admin/activity/events?kind=login&limit=2")
	if code != 200 || first.Total != 3 || len(first.Entries) != 2 || first.NextBefore == 0 || first.Since == nil {
		t.Fatalf("first: %d %+v", code, first)
	}
	telemetry.RecordLogin(telemetry.Activity{Player: "Later", MinLevel: 31})
	h.Sample(time.Now(), 1)
	next := activitySnapshot(h, &activityInput{Range: "24h", Kind: "login", Before: first.NextBefore, Limit: 2}, 31, time.Now())
	if len(next.Entries) != 1 || next.Entries[0].ID >= first.NextBefore || next.NextBefore != 0 {
		t.Fatalf("unstable page: %+v", next)
	}
	_, filtered := get("/admin/activity/events?kind=login&search=mIrA")
	if filtered.Total != 1 || filtered.Entries[0].Player != "Mira" {
		t.Fatalf("filter: %+v", filtered)
	}
	_, death := get("/admin/activity/events?kind=death")
	if len(death.Entries) != 1 || death.Entries[0].RoomVNum != 1001 || death.Entries[0].MobVNum != 300 {
		t.Fatalf("death: %+v", death)
	}
	_, creation := get("/admin/activity/events?kind=character")
	if len(creation.Entries) != 1 || creation.Entries[0].Class != "Mage" {
		t.Fatalf("creation: %+v", creation)
	}
	level = 40
	_, all := get("/admin/activity/events?kind=login")
	if all.Total != 5 {
		t.Fatalf("implementor visibility: %+v", all)
	}
	for _, path := range []string{"?kind=invalid", "?range=1y", "?limit=101", "?before=-1"} {
		if code, _ := get("/admin/activity/events" + path); code != 422 {
			t.Fatalf("invalid query accepted: %s %d", path, code)
		}
	}
	absent := activitySnapshot(nil, &activityInput{}, 31, time.Now())
	if absent.Recording || absent.Since != nil || len(absent.Entries) != 0 {
		t.Fatal("missing recorder looked populated")
	}
	if doc.OpenAPI().Paths["/admin/activity/events"].Get == nil {
		t.Fatal("missing typed operation")
	}
}

func TestActivityUsesCurrentStoredImmortalLevel(t *testing.T) {
	setJWTSecret(t)
	database := newOLCTestDatabase(t, game.LVL_IMMORT, 1)
	h := telemetry.New(time.Now())
	telemetry.RecordLogin(telemetry.Activity{Player: "Visible", MinLevel: 31})
	telemetry.RecordLogin(telemetry.Activity{Player: "Hidden", MinLevel: 40})
	h.Sample(time.Now(), 0)
	handler, err := NewRouter(testWorld(t), nil, NewLogBuffer(1), database, nil, WithHistory(h))
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	request := func() (int, activityResponse) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/admin/activity/events", nil)
		req.Header.Set("X-Requested-With", "darkpawns-admin")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		var body activityResponse
		if rr.Code == 200 {
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
		}
		return rr.Code, body
	}
	code, body := request()
	if code != 200 || body.Total != 1 || body.Entries[0].Player != "Visible" {
		t.Fatalf("claimed role bypassed actual level: %d %+v", code, body)
	}
	if _, err := database.Exec("UPDATE players SET level=? WHERE name=?", 40, "TestPlayer"); err != nil {
		t.Fatal(err)
	}
	code, body = request()
	if code != 200 || body.Total != 2 {
		t.Fatalf("level not resolved per request: %d %+v", code, body)
	}
	if _, err := database.Exec("UPDATE players SET level=? WHERE name=?", 1, "TestPlayer"); err != nil {
		t.Fatal(err)
	}
	code, _ = request()
	if code != 403 {
		t.Fatalf("demoted player retained access: %d", code)
	}
}
