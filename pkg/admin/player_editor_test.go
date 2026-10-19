package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

type editorReservation struct {
	blocked bool
	held    bool
}

func (r *editorReservation) ReserveOfflinePlayer(string) (func(), bool) {
	if r.blocked || r.held {
		return nil, false
	}
	r.held = true
	return func() { r.held = false }, true
}

func TestSavedPlayerEditor(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	world := newOLCTestWorld(t)
	record := &db.PlayerRecord{Name: "Returner", Level: 10, Health: 10, MaxHealth: 20, Mana: 10, MaxMana: 20, Move: 10, MaxMove: 20, StatStr: 15, StatCon: 15, StatInt: 15, StatWis: 15, StatDex: 15, StatCha: 15, Inventory: []byte(`[{"vnum":3001,"count":1,"state":{"timer":7}}]`), Equipment: []byte(`[]`), CharacterData: []byte(`{"save_version":2,"gold":123,"skills":{"kick":42}}`), OlcZone: 7}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	reservation := &editorReservation{}
	handler, err := NewRouter(world, nil, NewLogBuffer(10), database, reservation)
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	requests := 0
	request := func(method, path string, body any, credential string) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		requests++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", requests)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Requested-With", "darkpawns-admin")
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	get := func() savedPlayerResponse {
		t.Helper()
		res := request("GET", "/admin/characters/returner", nil, token)
		if res.Code != 200 {
			t.Fatalf("get: %d %s", res.Code, res.Body.String())
		}
		var out savedPlayerResponse
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, credential := range []string{"", generateTestToken(t, "builder")} {
		res := request("GET", "/admin/characters", nil, credential)
		if res.Code != 401 && res.Code != 403 {
			t.Fatalf("unauthorized: %d", res.Code)
		}
	}
	original := get()
	res := request("PATCH", "/admin/characters/Returner", map[string]any{"revision": original.Revision, "values": map[string]int{"str": 20, "health": 19}}, token)
	if res.Code != 200 {
		t.Fatalf("edit: %d %s", res.Code, res.Body.String())
	}
	saved, err := database.GetPlayer("Returner")
	if err != nil {
		t.Fatal(err)
	}
	if saved.StatStr != 20 || saved.Strength != 20 || saved.Health != 19 || saved.OlcZone != 7 || !bytes.Equal(saved.CharacterData, record.CharacterData) {
		t.Fatalf("record not preserved: %+v", saved)
	}
	res = request("PATCH", "/admin/characters/Returner", map[string]any{"revision": original.Revision, "values": map[string]int{"str": 21}}, token)
	if res.Code != 409 {
		t.Fatalf("stale write: %d %s", res.Code, res.Body.String())
	}
	current := get()
	for _, values := range []map[string]int{{"password": 1}, {"str": 26}, {"health": 21}, {"level": game.LVL_IMPL + 1}} {
		res = request("PATCH", "/admin/characters/Returner", map[string]any{"revision": current.Revision, "values": values}, token)
		if res.Code != 400 {
			t.Fatalf("invalid stats %v: %d %s", values, res.Code, res.Body.String())
		}
	}
	reservation.blocked = true
	res = request("PATCH", "/admin/characters/Returner", map[string]any{"revision": current.Revision, "values": map[string]int{"str": 21}}, token)
	if res.Code != 409 {
		t.Fatalf("online edit: %d", res.Code)
	}
	reservation.blocked = false
	res = request("POST", "/admin/characters/Returner/items", map[string]any{"revision": current.Revision, "action": "move", "source": "inventory", "index": 0, "vnum": 0, "locate": 17}, token)
	if res.Code != 200 {
		t.Fatalf("move: %d %s", res.Code, res.Body.String())
	}
	current = get()
	if len(current.Inventory) != 0 || len(current.Equipment) != 1 || current.Equipment[0].Locate != 17 {
		t.Fatalf("equipment: %+v", current)
	}
	snapshot, err := database.GetObjectSave("Returner")
	if err != nil || snapshot == nil || snapshot.Kind != 2 || len(snapshot.Objects) != 1 || snapshot.Objects[0].Locate != 17 || snapshot.Objects[0].State["timer"] != float64(7) {
		t.Fatalf("object snapshot: %+v %v", snapshot, err)
	}
	if _, err := database.Exec("UPDATE players SET failed_login_attempts=10,locked_until=? WHERE id=?", time.Now().Add(time.Hour), record.ID); err != nil {
		t.Fatal(err)
	}
	current = get()
	secret := "a new full password"
	res = request("POST", "/admin/characters/Returner/password", map[string]any{"revision": current.Revision, "password": secret}, token)
	if res.Code != 200 {
		t.Fatalf("password: %d %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), secret) || strings.Contains(res.Body.String(), "$2") {
		t.Fatal("credential leaked")
	}
	saved, err = database.GetPlayer("Returner")
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(saved.Password), []byte(secret)); err != nil {
		t.Fatal(err)
	}
	if saved.FailedLoginAttempts != 0 || saved.LockedUntil != nil {
		t.Fatal("lockout not cleared")
	}
	if reservation.held {
		t.Fatal("reservation leaked")
	}
	// The stored actor level overrides the token's old admin role.
	actor, err := database.GetPlayer("TestPlayer")
	if err != nil {
		t.Fatal(err)
	}
	actor.Level = game.LVL_GOD
	if err := database.SavePlayer(actor); err != nil {
		t.Fatal(err)
	}
	if res := request("GET", "/admin/characters", nil, token); res.Code != 403 {
		t.Fatalf("demoted actor: %d", res.Code)
	}
}

func TestSavedItemContainerEdits(t *testing.T) {
	world := newOLCTestWorld(t)
	r := &db.PlayerRecord{Inventory: []byte(`[{"vnum":3001,"count":1},{"vnum":3001,"count":1,"container_index":1,"state":{"label":"child"}},{"vnum":3001,"count":1}]`), Equipment: []byte(`[]`)}
	in := &editPlayerItemsInput{}
	in.Body.Action = "move"
	in.Body.Source = "inventory"
	in.Body.Locate = 7
	if err := editSavedItems(world, r, in); err != nil {
		t.Fatal(err)
	}
	inv, _ := decodeEditorItems(r.Inventory)
	eq, _ := decodeEditorItems(r.Equipment)
	if len(inv) != 1 || len(eq) != 2 || eq[1].ContainerIndex != 1 || eq[1].State["label"] != "child" {
		t.Fatalf("subtree lost: %s %s", r.Inventory, r.Equipment)
	}
	in.Body.Source = "equipment"
	in.Body.Action = "remove"
	if err := editSavedItems(world, r, in); err != nil {
		t.Fatal(err)
	}
	eq, _ = decodeEditorItems(r.Equipment)
	if len(eq) != 0 {
		t.Fatal("container contents survived removal")
	}
}

func TestPlayerEditorDisabledWithoutSessionReservation(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/characters/TestPlayer", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(t, "admin"))
	req.Header.Set("X-Requested-With", "darkpawns-admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 503 {
		t.Fatalf("without reservation: %d %s", rec.Code, rec.Body.String())
	}
}
