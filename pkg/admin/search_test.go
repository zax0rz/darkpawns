package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSearchWorldAcrossEditorKinds(t *testing.T) {
	parsed := &parser.World{
		Zones: []parser.Zone{{Number: 1, Name: "Dragon Reach"}},
		Rooms: []parser.Room{{VNum: 101, Name: "Dragon's Lair", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 201, ShortDesc: "a red dragon"}},
		Objs:  []parser.Obj{{VNum: 301, ShortDesc: "dragon-scaled leggings"}},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	shops := game.NewShopManager()
	shops.AddShop(&game.Shop{VNum: 401, KeeperVNum: 201, KeeperName: "Dragon trader"})
	world.SetShopManager(shops)

	setJWTSecret(t)
	router, err := NewRouter(world, nil, NewLogBuffer(10), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	request := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/admin/search?q=dragon", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	if got := request("").Code; got != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", got)
	}
	rec := request(generateTestToken(t, "builder"))
	if rec.Code != http.StatusOK {
		t.Fatalf("builder status = %d: %s", rec.Code, rec.Body.String())
	}
	var response worldSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Total != 5 || len(response.Results) != 5 {
		t.Fatalf("expected all five editor kinds, got %+v", response)
	}
	kinds := map[string]bool{}
	for _, hit := range response.Results {
		kinds[hit.Kind] = true
	}
	for _, kind := range []string{"zone", "room", "mob", "object", "shop"} {
		if !kinds[kind] {
			t.Errorf("missing %s", kind)
		}
	}
	if got := searchWorld(world, "dragon leggings"); got.Total != 1 || got.Results[0].Kind != "object" {
		t.Fatalf("multiword search = %+v", got)
	}
}
