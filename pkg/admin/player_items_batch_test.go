package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSavedPlayerItemBatch(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	world := newOLCTestWorld(t)
	proto, _ := world.GetObjPrototype(3001)
	proto.TypeFlag = game.ITEM_CONTAINER
	proto.ShortDesc = "a test bag"
	original := &db.PlayerRecord{Name: "Returner", Level: 10, Health: 10, MaxHealth: 20, Inventory: []byte(`[{"vnum":3001,"count":1,"state":{"timer":7}},{"vnum":3001,"count":1,"container_index":1,"state":{"timer":9}},{"vnum":3001,"count":2}]`), Equipment: []byte(`[]`), CharacterData: []byte(`{"save_version":2,"gold":123}`)}
	if err := database.CreatePlayer(original); err != nil {
		t.Fatal(err)
	}
	reservation := &editorReservation{}
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	logger, err := audit.NewAuditLogger(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	handler, err := NewRouter(world, logger, NewLogBuffer(10), database, reservation)
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	requests := 0
	request := func(method string, body any) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, "/admin/characters/Returner", bytes.NewReader(raw))
		requests++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", requests)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Requested-With", "darkpawns-admin")
		req.Header.Set("Authorization", "Bearer "+token)
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, req)
		return out
	}
	get := func() savedPlayerResponse {
		t.Helper()
		out := request("GET", nil)
		if out.Code != 200 {
			t.Fatalf("get: %d %s", out.Code, out.Body.String())
		}
		var view savedPlayerResponse
		if err := json.Unmarshal(out.Body.Bytes(), &view); err != nil {
			t.Fatal(err)
		}
		return view
	}
	initial := get()
	if len(initial.Slots) != 22 || !initial.Inventory[0].Container || initial.Inventory[1].Parent != 1 {
		t.Fatalf("catalog/tree: %+v", initial)
	}
	// Stats and an earlier valid item change must roll back if a later edit fails.
	failed := request("PATCH", map[string]any{"revision": initial.Revision, "values": map[string]int{"health": 19}, "item_changes": []playerItemChange{{Action: "move", Source: "inventory", Index: 0, Locate: 17}, {Action: "grant", Source: "inventory", VNum: 999999, Locate: 1}}})
	if failed.Code != 400 {
		t.Fatalf("invalid batch: %d %s", failed.Code, failed.Body.String())
	}
	if current := get(); current.Revision != initial.Revision {
		t.Fatal("failed batch changed the character")
	}
	if save, err := database.GetObjectSave("Returner"); err != nil || save != nil {
		t.Fatalf("failed batch wrote object save: %+v %v", save, err)
	}
	// Each operation addresses the preceding result: the stack reindexes to zero,
	// and a newly granted object is then moved to another slot in the same save.
	changes := []playerItemChange{{Action: "move", Source: "inventory", Index: 0, Locate: 17}, {Action: "grant", Source: "inventory", VNum: 3001}, {Action: "move", Source: "inventory", Index: 1, Locate: 1}}
	success := request("PATCH", map[string]any{"revision": initial.Revision, "values": map[string]int{"health": 19}, "item_changes": changes})
	if success.Code != 200 {
		t.Fatalf("batch: %d %s", success.Code, success.Body.String())
	}
	saved := get()
	if saved.Values["health"] != 19 || len(saved.Equipment) != 3 || len(saved.Inventory) != 1 || saved.Inventory[0].Count != 2 || saved.Equipment[1].Parent != 1 || saved.Equipment[2].Locate != 1 {
		t.Fatalf("saved result: %+v", saved)
	}
	snapshot, err := database.GetObjectSave("Returner")
	if err != nil || snapshot == nil || len(snapshot.Objects) != 5 {
		t.Fatalf("object save: %+v %v", snapshot, err)
	}
	record, err := database.GetPlayer("Returner")
	if err != nil {
		t.Fatal(err)
	}
	var equipment []game.SaveItemData
	if err := json.Unmarshal(record.Equipment, &equipment); err != nil {
		t.Fatal(err)
	}
	if equipment[0].State["timer"] != float64(7) || equipment[1].State["timer"] != float64(9) || equipment[1].ContainerIndex != 1 || !bytes.Equal(record.CharacterData, original.CharacterData) {
		t.Fatalf("instance state lost: %+v", equipment)
	}
	stale := request("PATCH", map[string]any{"revision": initial.Revision, "values": map[string]int{}, "item_changes": changes})
	if stale.Code != 409 {
		t.Fatalf("stale: %d", stale.Code)
	}
	reservation.blocked = true
	online := request("PATCH", map[string]any{"revision": saved.Revision, "values": map[string]int{}, "item_changes": changes})
	if online.Code != 409 {
		t.Fatalf("online: %d", online.Code)
	}
	reservation.blocked = false
	for _, invalid := range []any{
		[]playerItemChange{{Action: "move", Source: "inventory", Index: 0, Locate: 2}},     // stack
		[]playerItemChange{{Action: "grant", Source: "inventory", VNum: 3001, Locate: 17}}, // filled
		[]playerItemChange{{Action: "explode", Source: "inventory"}},
		make([]playerItemChange, 101),
	} {
		out := request("PATCH", map[string]any{"revision": saved.Revision, "values": map[string]int{}, "item_changes": invalid})
		if out.Code != 400 && out.Code != 422 {
			t.Fatalf("invalid: %d %s", out.Code, out.Body.String())
		}
		if get().Revision != saved.Revision {
			t.Fatal("invalid edit changed saved state")
		}
	}
	// Move a worn bag and its subtree back, then remove it in one ordered batch.
	out := request("PATCH", map[string]any{"revision": saved.Revision, "values": map[string]int{}, "item_changes": []playerItemChange{{Action: "move", Source: "equipment", Index: 0}, {Action: "remove", Source: "inventory", Index: 1}}})
	if out.Code != 200 {
		t.Fatalf("remove tree: %d %s", out.Code, out.Body.String())
	}
	final := get()
	if len(final.Inventory) != 1 || len(final.Equipment) != 1 || final.Equipment[0].Locate != 1 {
		t.Fatalf("subtree removal: %+v", final)
	}
	if !strings.Contains(string(mustReadFile(t, auditPath)), "admin_character_items") {
		t.Fatal("item save was not audited")
	}
}
