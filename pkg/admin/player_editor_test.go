package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/boards"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/moderation"
	"golang.org/x/crypto/bcrypt"
)

type editorReservation struct {
	blocked        bool
	held           bool
	renameObserver func(oldName, newName string)
}

type editorRenameObserverFunc func(oldName, newName string)

func (f editorRenameObserverFunc) PlayerRenamed(oldName, newName string) {
	f(oldName, newName)
}

func (r *editorReservation) ReserveOfflinePlayer(string) (func(), bool) {
	if r.blocked || r.held {
		return nil, false
	}
	r.held = true
	return func() { r.held = false }, true
}

func (r *editorReservation) ReserveOfflineRename(oldName, newName string) (func(), bool) {
	return r.ReserveOfflinePlayer(oldName)
}

func (r *editorReservation) PlayerRenamed(oldName, newName string) {
	if r.renameObserver != nil {
		r.renameObserver(oldName, newName)
	}
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
	if unknown := get(); unknown.LastSavedAt != nil {
		t.Fatal("invented timestamp for unsaved character")
	}
	record.LastLogon = 1735689600
	if err := database.SavePlayer(record); err != nil {
		t.Fatal(err)
	}
	original := get()
	if original.LastSavedAt == nil || original.LastSavedAt.Unix() != record.LastLogon {
		t.Fatal("last save timestamp missing")
	}
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

type editorBoardPlayer struct {
	name  string
	level int
	out   strings.Builder
}

func (p *editorBoardPlayer) GetLevel() int           { return p.level }
func (p *editorBoardPlayer) GetName() string         { return p.name }
func (*editorBoardPlayer) GetRoomVNum() int          { return 1001 }
func (p *editorBoardPlayer) SendMessage(text string) { p.out.WriteString(text) }

func TestSavedPlayerRenameAndDeleteAcrossReferences(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Error(err)
		}
	})
	xnameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(xnameRoot, "text"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xnameRoot, "text", "xnames"), []byte("blocked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	game.SetBanFilePaths(xnameRoot)
	if err := game.ReadInvalidList(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(filepath.Join(xnameRoot, "text", "xnames"), nil, 0o600); err != nil {
			t.Error(err)
		}
		game.SetBanFilePaths(xnameRoot)
		if err := game.ReadInvalidList(); err != nil {
			t.Error(err)
		}
	})
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	t.Cleanup(func() { _ = database.Close() })
	world := newOLCTestWorld(t)
	if proto, ok := world.GetObjPrototype(3001); ok {
		proto.ExtraFlags[0] |= game.FlagNoRent
	}
	world.Clans = game.NewClanManager()
	world.Clans.Clans = []*game.Clan{{ID: 3, Members: 1, Power: 10}, {ID: 42, Members: 1, Power: 5}}
	world.HouseControl = []game.HouseControl{{VNum: 1001, Owner: 2, NumOfGuests: 1, Guests: [50]int64{1}}}
	world.Boards = boards.InitBoards(filepath.Join(t.TempDir(), "boards"))
	boardAuthor := &editorBoardPlayer{name: "Returner", level: 10}
	magic := world.Boards.WriteMessage(0, boardAuthor, "A historical message")
	if magic != boards.BoardMagic {
		t.Fatalf("board write returned %d", magic)
	}
	world.Boards.AppendBoardLine(magic, boardAuthor, "The text stays attached.")
	world.Boards.FinalizeBoardWrite(magic, boardAuthor)
	otherBoardAuthor := &editorBoardPlayer{name: "Newname", level: 10}
	otherMagic := world.Boards.WriteMessage(0, otherBoardAuthor, "An older post by the destination name")
	if otherMagic != boards.BoardMagic {
		t.Fatalf("destination-name board write returned %d", otherMagic)
	}
	world.Boards.AppendBoardLine(otherMagic, otherBoardAuthor, "Keep this historical author.")
	world.Boards.FinalizeBoardWrite(otherMagic, otherBoardAuthor)

	record := &db.PlayerRecord{
		Name: "Returner", Level: 10, Health: 10, MaxHealth: 20, Mana: 10, MaxMana: 20, Move: 10, MaxMove: 20,
		StatStr: 15, StatCon: 15, StatInt: 15, StatWis: 15, StatDex: 15, StatCha: 15,
		Inventory: []byte(`[{"vnum":3001,"count":1}]`), Equipment: []byte(`[]`),
		CharacterData: []byte(`{"save_version":2,"name":"Returner","flags":0,"clan_id":3,"clan_rank":3,"future_field":"preserved"}`),
	}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	world.HouseControl[0].Owner = int64(record.ID)
	world.HouseControl[0].Guests[0] = int64(record.ID)
	if err := db.SaveObjectSnapshot(database, record.Name, 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := game.WriteAliases(record.Name, []game.Alias{{Alias: "hi", Replacement: " hello", Type: game.AliasSimple}}); err != nil {
		t.Fatal(err)
	}
	mod := moderation.NewManager(database.SQLDB())
	t.Cleanup(mod.Close)
	issued := time.Now().UTC().Truncate(time.Second)
	if err := mod.AddPenalty(moderation.PlayerPenalty{PlayerName: record.Name, PenaltyType: moderation.ActionBan, IssuedAt: issued, Reason: "test", IssuedBy: "admin"}); err != nil {
		t.Fatal(err)
	}
	reservation := &editorReservation{}
	reservation.renameObserver = mod.RenamePlayerName
	observedRename := false
	mailNameCacheObserver := editorRenameObserverFunc(func(oldName, newName string) {
		observedRename = oldName == "Returner" && newName == "Newname"
	})
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	auditLogger, err := audit.NewAuditLogger(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = auditLogger.Close() })
	handler, err := NewRouter(world, auditLogger, NewLogBuffer(10), database, reservation, WithPlayerRenameObserver(mailNameCacheObserver))
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	requests := 0
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		requests++
		req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", requests)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Requested-With", "darkpawns-admin")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	get := func(name string) savedPlayerResponse {
		t.Helper()
		res := request("GET", "/admin/characters/"+name, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("get %s: %d %s", name, res.Code, res.Body.String())
		}
		var out savedPlayerResponse
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	current := get("Returner")
	if len(current.Inventory) != 1 || !current.Inventory[0].NoRent {
		t.Fatalf("player item rent flag not surfaced: %+v", current.Inventory)
	}
	for _, invalid := range []string{"the", "Admin", "Has Digit1", "Blockedname"} {
		res := request("POST", "/admin/characters/Returner/rename", map[string]any{"revision": current.Revision, "new_name": invalid})
		if res.Code != http.StatusBadRequest {
			t.Fatalf("rename invalid %q: %d %s", invalid, res.Code, res.Body.String())
		}
	}
	reservation.blocked = true
	res := request("POST", "/admin/characters/Returner/rename", map[string]any{"revision": current.Revision, "new_name": "Newname"})
	if res.Code != http.StatusConflict {
		t.Fatalf("rename while online: %d %s", res.Code, res.Body.String())
	}
	reservation.blocked = false
	// Simulate process interruption after board and alias files moved but before
	// the SQLite identity transaction. Rebuilding the router runs boot recovery.
	if _, err := world.Boards.RenameAuthorWithPrepare("Returner", "Newname", func(snapshot []byte) error {
		return database.PrepareCharacterRename("Returner", "Newname", snapshot)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.RenameAliases("Returner", "Newname"); err != nil {
		t.Fatal(err)
	}
	handler, err = NewRouter(world, auditLogger, NewLogBuffer(10), database, reservation, WithPlayerRenameObserver(mailNameCacheObserver))
	if err != nil {
		t.Fatalf("startup rename recovery: %v", err)
	}
	if aliases, err := game.ReadAliases("Returner"); err != nil || len(aliases) != 1 {
		t.Fatalf("startup recovery did not restore aliases: %+v err=%v", aliases, err)
	}
	if aliases, err := game.ReadAliases("Newname"); err != nil || len(aliases) != 0 {
		t.Fatalf("startup recovery left new alias identity: %+v err=%v", aliases, err)
	}
	reader := &editorBoardPlayer{name: "Reader", level: 10}
	if !world.Boards.DisplayMsg(0, reader, "1") || !strings.Contains(reader.out.String(), "(Returner)") {
		t.Fatalf("startup recovery did not restore original board heading: %q", reader.out.String())
	}
	reader.out.Reset()
	if !world.Boards.DisplayMsg(0, reader, "2") || !strings.Contains(reader.out.String(), "(Newname)") || !strings.Contains(reader.out.String(), "Keep this historical author.") {
		t.Fatalf("startup recovery changed unrelated destination-name post: %q", reader.out.String())
	}
	res = request("POST", "/admin/characters/Returner/rename", map[string]any{"revision": current.Revision, "new_name": "Newname"})
	if res.Code != http.StatusOK {
		t.Fatalf("rename: %d %s", res.Code, res.Body.String())
	}
	var renamed savedPlayerResponse
	if err := json.Unmarshal(res.Body.Bytes(), &renamed); err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Newname" {
		t.Fatalf("rename response name = %q", renamed.Name)
	}
	if !observedRename {
		t.Fatal("rename did not notify the external stable-ID name cache")
	}
	saved, err := database.GetPlayer("Newname")
	if err != nil || saved == nil || saved.ID != record.ID {
		t.Fatalf("stable player ID after rename: %+v err=%v", saved, err)
	}
	var embeddedName struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(saved.CharacterData, &embeddedName); err != nil || embeddedName.Name != "Newname" {
		t.Fatalf("character_data name = %q err=%v", embeddedName.Name, err)
	}
	clanID, clanRank, err := game.CharacterDataClan(saved.CharacterData)
	if err != nil || clanID != 3 || clanRank != 3 {
		t.Fatalf("rename changed clan membership id/rank: %d/%d err=%v", clanID, clanRank, err)
	}
	if _, err := database.GetPlayer("Returner"); err != nil {
		t.Fatal(err)
	} else if old, _ := database.GetPlayer("Returner"); old != nil {
		t.Fatal("old player-name key remained")
	}
	if snapshot, err := database.GetObjectSave("Newname"); err != nil || snapshot == nil {
		t.Fatalf("object-save key was not moved: %+v err=%v", snapshot, err)
	}
	if aliases, err := game.ReadAliases("Newname"); err != nil || len(aliases) != 1 || aliases[0].Alias != "hi" {
		t.Fatalf("alias file not moved: %+v err=%v", aliases, err)
	}
	if aliases, err := game.ReadAliases("Returner"); err != nil || len(aliases) != 0 {
		t.Fatalf("old alias identity remains: %+v err=%v", aliases, err)
	}
	if len(mod.GetPlayerPenalties("Newname")) != 1 || len(mod.GetPlayerPenalties("Returner")) != 0 {
		t.Fatal("active moderation penalty cache did not follow the rename")
	}
	reader = &editorBoardPlayer{name: "Reader", level: 10}
	if !world.Boards.DisplayMsg(0, reader, "1") || !strings.Contains(reader.out.String(), "(Newname)") || !strings.Contains(reader.out.String(), "The text stays attached.") {
		t.Fatalf("board author heading not renamed: %q", reader.out.String())
	}
	if world.HouseControl[0].Owner != int64(record.ID) || world.HouseControl[0].Guests[0] != int64(record.ID) {
		t.Fatalf("house IDs changed during rename: %+v", world.HouseControl[0])
	}
	var penaltyName string
	if err := database.SQLDB().QueryRow(`SELECT player_name FROM player_penalties WHERE penalty_type='ban'`).Scan(&penaltyName); err != nil || penaltyName != "newname" {
		t.Fatalf("persistent penalty key = %q err=%v", penaltyName, err)
	}
	current = get("Newname")
	flagKeys := make(map[string]bool)
	for _, flag := range current.Flags {
		flagKeys[flag.Key] = true
	}
	for _, key := range []string{"brief", "invstart", "can_be_summoned", "nohassle", "frozen", "outlaw", "roomflag", "siteok", "nowizlist", "quest", "color", "nodelete", "chosen", "cryo", "deleted"} {
		if !flagKeys[key] {
			t.Fatalf("flag catalog is missing %q: %v", key, flagKeys)
		}
	}
	res = request("PATCH", "/admin/characters/Newname/flags", map[string]any{"revision": current.Revision, "flags": map[string]bool{"frozen": true}})
	if res.Code != http.StatusOK {
		t.Fatalf("enable frozen: %d %s", res.Code, res.Body.String())
	}
	current = get("Newname")
	res = request("PATCH", "/admin/characters/Newname/flags", map[string]any{"revision": current.Revision, "flags": map[string]bool{"frozen": false, "nodelete": true, "cryo": true}})
	if res.Code != http.StatusOK {
		t.Fatalf("update player flags: %d %s", res.Code, res.Body.String())
	}
	updatedFlags, err := game.CharacterDataFlags(func() []byte {
		row, lookupErr := database.GetPlayer("Newname")
		if lookupErr != nil {
			t.Fatal(lookupErr)
		}
		return row.CharacterData
	}())
	if err != nil || updatedFlags&(uint64(1)<<uint(game.PlrNODELETE)) == 0 || updatedFlags&(uint64(1)<<uint(game.PlrCRYO)) == 0 || updatedFlags&(uint64(1)<<uint(game.PlrFrozen)) != 0 {
		t.Fatalf("saved flags = %#x err=%v", updatedFlags, err)
	}
	current = get("Newname")
	res = request("PATCH", "/admin/characters/Newname/flags", map[string]any{"revision": current.Revision, "flags": map[string]bool{"frozen": true}})
	if res.Code != http.StatusOK {
		t.Fatalf("freeze character before admin delete: %d %s", res.Code, res.Body.String())
	}
	current = get("Newname")
	reservation.blocked = true
	res = request("POST", "/admin/characters/Newname/delete", map[string]any{"revision": current.Revision})
	if res.Code != http.StatusConflict {
		t.Fatalf("delete while online: %d %s", res.Code, res.Body.String())
	}
	reservation.blocked = false
	res = request("POST", "/admin/characters/Newname/delete", map[string]any{"revision": current.Revision})
	if res.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", res.Code, res.Body.String())
	}
	var deleted deleteSavedPlayerResponse
	if err := json.Unmarshal(res.Body.Bytes(), &deleted); err != nil {
		t.Fatal(err)
	}
	if !deleted.MarkedDeleted || !deleted.AliasesRemoved || !deleted.ClanCleared {
		t.Fatalf("delete result = %+v", deleted)
	}
	saved, err = database.GetPlayer("Newname")
	if err != nil || saved == nil || saved.ID != record.ID || !game.CharacterDataDeleted(saved.CharacterData) || string(saved.Inventory) != "[]" || string(saved.Equipment) != "{}" {
		t.Fatalf("admin-deleted character row = %+v err=%v", saved, err)
	}
	clanID, clanRank, err = game.CharacterDataClan(saved.CharacterData)
	if err != nil || clanID != 0 || clanRank != 0 || world.Clans.Clans[0].Members != 0 || world.Clans.Clans[0].Power != 0 {
		t.Fatalf("clan cleanup id/rank=%d/%d clan=%+v err=%v", clanID, clanRank, world.Clans.Clans[0], err)
	}
	if snapshot, err := database.GetObjectSave("Newname"); err != nil || snapshot != nil {
		t.Fatalf("delete retained saved items: %+v err=%v", snapshot, err)
	}
	if aliases, err := game.ReadAliases("Newname"); err != nil || len(aliases) != 0 {
		t.Fatalf("delete retained aliases: %+v err=%v", aliases, err)
	}
	if world.HouseControl[0].Owner != int64(record.ID) || world.HouseControl[0].Guests[0] != int64(record.ID) {
		t.Fatal("admin delete unexpectedly rewrote house ownership IDs")
	}
	if len(mod.GetPlayerPenalties("Newname")) != 1 {
		t.Fatal("admin delete unexpectedly removed historical moderation data")
	}
	reader = &editorBoardPlayer{name: "Reader", level: 10}
	if !world.Boards.DisplayMsg(0, reader, "1") || !strings.Contains(reader.out.String(), "(Newname)") || !strings.Contains(reader.out.String(), "The text stays attached.") {
		t.Fatalf("admin delete unexpectedly removed the historical board post: %q", reader.out.String())
	}
	names, err := database.ListActivePlayerNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if strings.EqualFold(name, "Newname") {
			t.Fatalf("deleted character still listed: %v", names)
		}
	}
	if !strings.Contains(string(mustReadFile(t, auditPath)), "admin_character_rename") || !strings.Contains(string(mustReadFile(t, auditPath)), "admin_character_delete") {
		t.Fatal("rename or delete did not append an admin audit entry")
	}
	applicant := &db.PlayerRecord{
		Name: "Applicant", Level: 5, Inventory: []byte("[]"), Equipment: []byte("{}"),
		CharacterData: []byte(`{"save_version":2,"name":"Applicant","flags":0,"clan_id":3,"clan_rank":0}`),
	}
	if err := database.CreatePlayer(applicant); err != nil {
		t.Fatal(err)
	}
	applicantView := get("Applicant")
	res = request("POST", "/admin/characters/Applicant/delete", map[string]any{"revision": applicantView.Revision})
	if res.Code != http.StatusOK {
		t.Fatalf("delete clan applicant: %d %s", res.Code, res.Body.String())
	}
	var deletedApplicant deleteSavedPlayerResponse
	if err := json.Unmarshal(res.Body.Bytes(), &deletedApplicant); err != nil {
		t.Fatal(err)
	}
	applicantSaved, err := database.GetPlayer("Applicant")
	if err != nil || applicantSaved == nil {
		t.Fatalf("load deleted applicant: %+v err=%v", applicantSaved, err)
	}
	applicantClanID, applicantClanRank, err := game.CharacterDataClan(applicantSaved.CharacterData)
	if err != nil || !deletedApplicant.ClanCleared || applicantClanID != 0 || applicantClanRank != 0 || world.Clans.Clans[0].Members != 0 || world.Clans.Clans[0].Power != 0 {
		t.Fatalf("applicant cleanup id/rank=%d/%d clan=%+v result=%+v err=%v", applicantClanID, applicantClanRank, world.Clans.Clans[0], deletedApplicant, err)
	}
	oldIdentity, err := database.GetPlayer("Newname")
	if err != nil || oldIdentity == nil || !game.CharacterDataDeleted(oldIdentity.CharacterData) {
		t.Fatalf("deleted name was not retained for safe reuse: %+v err=%v", oldIdentity, err)
	}
	replacement := &db.PlayerRecord{Name: "Newname", Level: 1, Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: []byte(`{"save_version":2,"name":"Newname"}`)}
	if err := db.ReplaceDeletedPlayer(database, replacement, oldIdentity); err != nil {
		t.Fatalf("reuse deleted name: %v", err)
	}
	if replacement.ID == oldIdentity.ID {
		t.Fatalf("reused name kept old ID %d", replacement.ID)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSavedPlayerAdminDeleteAtImmortalLevelRequiresTypedName(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Error(err)
		}
	})
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	player := game.NewPlayer(91, "Highlevel", 1001)
	player.SetLevel(game.LVL_IMPL - 1)
	data, err := game.EncodeCharacterData(player)
	if err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{Name: player.Name, Level: game.LVL_IMPL - 1, Inventory: []byte(`[{}]`), Equipment: []byte(`{}`), CharacterData: data}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveObjectSnapshot(database, record.Name, 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := game.WriteAliases(record.Name, []game.Alias{{Alias: "hi", Replacement: " hello", Type: game.AliasSimple}}); err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, &editorReservation{})
	if err != nil {
		t.Fatal(err)
	}
	token := generateTestToken(t, "admin")
	get := httptest.NewRequest(http.MethodGet, "/admin/characters/Highlevel", nil)
	get.Header.Set("Authorization", "Bearer "+token)
	get.Header.Set("X-Requested-With", "darkpawns-admin")
	getResult := httptest.NewRecorder()
	handler.ServeHTTP(getResult, get)
	if getResult.Code != http.StatusOK {
		t.Fatalf("get character: %d %s", getResult.Code, getResult.Body.String())
	}
	var view savedPlayerResponse
	if err := json.Unmarshal(getResult.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{"revision": view.Revision})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/characters/Highlevel/delete", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Requested-With", "darkpawns-admin")
	req.Header.Set("Content-Type", "application/json")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	if result.Code != http.StatusBadRequest {
		t.Fatalf("delete level-39 character without typed name: %d %s", result.Code, result.Body.String())
	}
	body, err = json.Marshal(map[string]string{"revision": view.Revision, "confirm_name": strings.ToLower(record.Name)})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/characters/Highlevel/delete", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Requested-With", "darkpawns-admin")
	req.Header.Set("Content-Type", "application/json")
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	if result.Code != http.StatusBadRequest {
		t.Fatalf("delete level-39 character with case-mismatched name: %d %s", result.Code, result.Body.String())
	}
	body, err = json.Marshal(map[string]string{"revision": view.Revision, "confirm_name": "Highlevel"})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/characters/Highlevel/delete", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Requested-With", "darkpawns-admin")
	req.Header.Set("Content-Type", "application/json")
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	if result.Code != http.StatusOK {
		t.Fatalf("delete level-39 character after typing name: %d %s", result.Code, result.Body.String())
	}
	var deleted deleteSavedPlayerResponse
	if err := json.Unmarshal(result.Body.Bytes(), &deleted); err != nil {
		t.Fatal(err)
	}
	if !deleted.MarkedDeleted {
		t.Fatal("admin delete did not mark the level-39 character deleted")
	}
	saved, err := database.GetPlayer("Highlevel")
	if err != nil || saved == nil || !game.CharacterDataDeleted(saved.CharacterData) {
		t.Fatalf("high-level character delete state = %+v err=%v", saved, err)
	}
	if snapshot, err := database.GetObjectSave("Highlevel"); err != nil || snapshot != nil {
		t.Fatalf("high-level object save remains: %+v err=%v", snapshot, err)
	}
	if aliases, err := game.ReadAliases("Highlevel"); err != nil || len(aliases) != 0 {
		t.Fatalf("high-level aliases remain: %+v err=%v", aliases, err)
	}
	names, err := database.ListActivePlayerNames()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(names, "Highlevel") {
		t.Fatalf("admin-deleted character remains active: %v", names)
	}
}

func TestSavedPlayerAdminDeleteRefusesOwnAndLevelFortyCharacters(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	target := game.NewPlayer(92, "Otherimpl", 1001)
	target.SetLevel(game.LVL_IMPL)
	data, err := game.EncodeCharacterData(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePlayer(&db.PlayerRecord{Name: target.Name, Level: game.LVL_IMPL, Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: data}); err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, &editorReservation{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(name string) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(map[string]string{"revision": strings.Repeat("0", 64)})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/characters/"+name+"/delete", strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer "+generateTestToken(t, "admin"))
		req.Header.Set("X-Requested-With", "darkpawns-admin")
		req.Header.Set("Content-Type", "application/json")
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		return result
	}
	if result := request("TestPlayer"); result.Code != http.StatusForbidden || !strings.Contains(result.Body.String(), "own character") {
		t.Fatalf("delete own character: %d %s", result.Code, result.Body.String())
	}
	if result := request("Otherimpl"); result.Code != http.StatusForbidden || !strings.Contains(result.Body.String(), "Level 40") {
		t.Fatalf("delete level-40 character: %d %s", result.Code, result.Body.String())
	}
	for _, name := range []string{"TestPlayer", "Otherimpl"} {
		saved, err := database.GetPlayer(name)
		if err != nil || saved == nil || game.CharacterDataDeleted(saved.CharacterData) {
			t.Fatalf("refused delete changed %s: %+v err=%v", name, saved, err)
		}
	}
}
