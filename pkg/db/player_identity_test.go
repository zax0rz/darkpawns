package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRenamePlayerIfCurrentMovesPersistentNameKeys(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Exec(`CREATE TABLE player_penalties (player_name TEXT NOT NULL, penalty_type TEXT NOT NULL, issued_at TEXT NOT NULL, PRIMARY KEY(player_name, penalty_type, issued_at))`); err != nil {
		t.Fatal(err)
	}
	original := &PlayerRecord{Name: "Oldname", Inventory: []byte(`[]`), Equipment: []byte(`[]`), CharacterData: []byte(`{"save_version":2,"name":"Oldname","clan_id":5,"clan_rank":2}`)}
	if err := store.CreatePlayer(original); err != nil {
		t.Fatal(err)
	}
	if err := SaveObjectSnapshot(store, original.Name, 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`INSERT INTO player_penalties VALUES ('oldname','ban','2026-10-27T12:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	original, err = store.GetPlayer("Oldname")
	if err != nil {
		t.Fatal(err)
	}
	updated := *original
	updated.Name = "Newname"
	updated.CharacterData, err = game.RenameCharacterData(original.CharacterData, updated.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PrepareCharacterRename(original.Name, "Newname", nil); err != nil {
		t.Fatal(err)
	}
	if err := store.RenamePlayerIfCurrent(&updated, original); err != nil {
		t.Fatal(err)
	}
	if pending, err := store.ListPendingCharacterRenames(); err != nil || len(pending) != 0 {
		t.Fatalf("committed rename journal = %+v err=%v", pending, err)
	}
	got, err := store.GetPlayer("Newname")
	if err != nil || got == nil || got.ID != original.ID {
		t.Fatalf("renamed player identity = %+v, err=%v; want ID %d", got, err, original.ID)
	}
	var savedName struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(got.CharacterData, &savedName); err != nil || savedName.Name != "Newname" {
		t.Fatalf("character_data duplicate name = %q, err=%v", savedName.Name, err)
	}
	if snapshot, err := store.GetObjectSave("Oldname"); err != nil || snapshot != nil {
		t.Fatalf("old object identity remains: %+v %v", snapshot, err)
	}
	if snapshot, err := store.GetObjectSave("Newname"); err != nil || snapshot == nil || snapshot.Kind != 2 {
		t.Fatalf("new object identity missing: %+v %v", snapshot, err)
	}
	var penaltyName string
	if err := store.queryRow(`SELECT player_name FROM player_penalties WHERE penalty_type='ban'`).Scan(&penaltyName); err != nil || penaltyName != "newname" {
		t.Fatalf("active penalty name = %q, err=%v", penaltyName, err)
	}
	var id int
	if err := store.queryRow(`SELECT id FROM players WHERE name='Newname'`).Scan(&id); err != nil || id != original.ID {
		t.Fatalf("stable ID for mail/house references = %d, err=%v", id, err)
	}
}

func TestRenamePlayerIfCurrentRollsBackConflictingReferences(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	original := &PlayerRecord{Name: "Oldname", CharacterData: []byte(`{"save_version":2,"name":"Oldname"}`)}
	if err := store.CreatePlayer(original); err != nil {
		t.Fatal(err)
	}
	if err := SaveObjectSnapshot(store, "Newname", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	original, err = store.GetPlayer(original.Name)
	if err != nil {
		t.Fatal(err)
	}
	updated := *original
	updated.Name = "Newname"
	updated.CharacterData, err = game.RenameCharacterData(original.CharacterData, updated.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PrepareCharacterRename(original.Name, "Newname", nil); err != nil {
		t.Fatal(err)
	}
	if err := store.RenamePlayerIfCurrent(&updated, original); !errors.Is(err, ErrRenameReferenceConflict) {
		t.Fatalf("rename collision error = %v", err)
	}
	got, err := store.GetPlayer("Oldname")
	if err != nil || got == nil || got.ID != original.ID {
		t.Fatalf("character row changed after collision: %+v %v", got, err)
	}
	if err := store.ClearPendingCharacterRename(original.Name, "Newname"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`DROP TABLE IF EXISTS player_penalties`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`CREATE TRIGGER deny_object_move BEFORE UPDATE OF identity ON object_saves BEGIN SELECT RAISE(ABORT,'test object move failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := DeleteObjectSave(store, "Newname"); err != nil {
		t.Fatal(err)
	}
	if err := SaveObjectSnapshot(store, "Oldname", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := store.PrepareCharacterRename(original.Name, "Newname", nil); err != nil {
		t.Fatal(err)
	}
	if err := store.RenamePlayerIfCurrent(&updated, original); err == nil {
		t.Fatal("expected injected object move error")
	}
	got, err = store.GetPlayer("Oldname")
	if err != nil || got == nil || got.Name != "Oldname" {
		t.Fatalf("character row changed after transaction failure: %+v %v", got, err)
	}
	if snapshot, err := store.GetObjectSave("Oldname"); err != nil || snapshot == nil {
		t.Fatalf("old object row lost after failed transaction: %+v %v", snapshot, err)
	}
}

func TestDeleteCharacterSelfIfCurrentRetainsRowAndDeletesObjectSaveAtomically(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	p := game.NewPlayer(17, "Returner", 1001)
	p.SetPlrFlag(game.PlrDeleted, true)
	data, err := game.EncodeCharacterData(p)
	if err != nil {
		t.Fatal(err)
	}
	original := &PlayerRecord{Name: "Returner", Level: 10, Inventory: []byte(`[1]`), Equipment: []byte(`{"hold":2}`), CharacterData: data}
	if err := store.CreatePlayer(original); err != nil {
		t.Fatal(err)
	}
	if err := SaveObjectSnapshot(store, original.Name, 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	original, err = store.GetPlayer(original.Name)
	if err != nil {
		t.Fatal(err)
	}
	updated := *original
	updated.Inventory, updated.Equipment = []byte(`[]`), []byte(`{}`)
	if _, err := store.Exec("UPDATE players SET level=11 WHERE id=?", original.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`CREATE TRIGGER deny_object_delete BEFORE DELETE ON object_saves BEGIN SELECT RAISE(ABORT,'test object delete failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCharacterSelfIfCurrent(&updated, original); err == nil {
		t.Fatal("expected injected object delete error")
	}
	got, err := store.GetPlayer("Returner")
	if err != nil || got == nil || got.ID != original.ID || !bytes.Equal(got.Inventory, original.Inventory) {
		t.Fatalf("partial delete changed row: %+v %v", got, err)
	}
	if snapshot, err := store.GetObjectSave("Returner"); err != nil || snapshot == nil {
		t.Fatalf("partial delete removed object identity: %+v %v", snapshot, err)
	}
	if _, err := store.Exec("DROP TRIGGER deny_object_delete"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCharacterSelfIfCurrent(&updated, got); err != nil {
		t.Fatal(err)
	}
	got, err = store.GetPlayer("Returner")
	if err != nil || got == nil || got.ID != original.ID || string(got.Inventory) != "[]" || string(got.Equipment) != "{}" || !game.CharacterDataDeleted(got.CharacterData) {
		t.Fatalf("C deleted row state = %+v err=%v", got, err)
	}
	if snapshot, err := store.GetObjectSave("Returner"); err != nil || snapshot != nil {
		t.Fatalf("deleted object identity remains: %+v %v", snapshot, err)
	}
}
