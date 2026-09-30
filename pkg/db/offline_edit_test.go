package db

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestOfflineEditRejectsNewerSQLiteRecord(t *testing.T) {
	store := openGameStore(t, filepath.Join(t.TempDir(), "edit.db"))
	r := &PlayerRecord{Name: "Offline", Level: 1, Health: 10, MaxHealth: 10, Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: []byte("{}"), OlcZone: 7}
	if err := store.CreatePlayer(r); err != nil {
		t.Fatal(err)
	}
	original, err := store.GetPlayer(r.Name)
	if err != nil {
		t.Fatal(err)
	}
	edit := *original
	edit.Health = 5
	newer := *original
	newer.OlcZone = 99
	newer.Health = 8
	if err := store.SavePlayer(&newer); err != nil {
		t.Fatal(err)
	}
	if err := SavePlayerIfCurrent(store, &edit, original); !errors.Is(err, ErrPlayerRecordChanged) {
		t.Fatalf("stale edit error=%v", err)
	}
	got, err := store.GetPlayer(r.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Health != 8 || got.OlcZone != 99 {
		t.Fatal("stale edit overwrote newer save")
	}
	edit = *got
	edit.Health = 5
	if err := SavePlayerIfCurrent(store, &edit, got); err != nil {
		t.Fatal(err)
	}
	got, err = store.GetPlayer(r.Name)
	if err != nil || got.Health != 5 || got.OlcZone != 99 {
		t.Fatal("current edit failed")
	}
}
