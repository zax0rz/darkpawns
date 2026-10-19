package db

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAdminPlayerEditAtomicity(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	original := &PlayerRecord{Name: "Offline", Level: 1, Inventory: []byte(`[]`), Equipment: []byte(`[]`)}
	if err := store.CreatePlayer(original); err != nil {
		t.Fatal(err)
	}
	original, err = store.GetPlayer("Offline")
	if err != nil {
		t.Fatal(err)
	}
	updated := *original
	updated.Level = 2
	secret := "new-hash"
	// A failure in the second half must roll back character and password too.
	if _, err := store.Exec(`CREATE TRIGGER refuse_objects BEFORE INSERT ON object_saves BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.AdminSavePlayer(&updated, original, &secret, true); err == nil {
		t.Fatal("expected object write failure")
	}
	got, err := store.GetPlayer("Offline")
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != original.Level || got.Password != original.Password {
		t.Fatal("partial character/password commit")
	}
	if _, err := store.Exec("DROP TRIGGER refuse_objects"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdatePassword(original.ID, "intervening-hash"); err != nil {
		t.Fatal(err)
	}
	if err := store.AdminSavePlayer(&updated, original, &secret, true); !errors.Is(err, ErrPlayerRecordChanged) {
		t.Fatalf("concurrent password: %v", err)
	}
	if snapshot, err := store.GetObjectSave("Offline"); err != nil || snapshot != nil {
		t.Fatalf("stale edit created snapshot: %+v %v", snapshot, err)
	}
	current, err := store.GetPlayer("Offline")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AdminSavePlayer(&updated, current, nil, true); err != nil {
		t.Fatal(err)
	}
	if snapshot, err := store.GetObjectSave("Offline"); err != nil || snapshot == nil || snapshot.Kind != 2 {
		t.Fatalf("missing new snapshot: %+v %v", snapshot, err)
	}
	if _, err := store.Exec("UPDATE object_saves SET kind=3"); err != nil {
		t.Fatal(err)
	}
	current, err = store.GetPlayer("Offline")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AdminSavePlayer(&updated, current, nil, true); err != nil {
		t.Fatal(err)
	}
	if snapshot, err := store.GetObjectSave("Offline"); err != nil || snapshot.Kind != 3 {
		t.Fatalf("cryo kind overwritten: %+v %v", snapshot, err)
	}
}
