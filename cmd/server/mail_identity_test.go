package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

func TestMailIdentityUsesPersistentRecordsForOfflineAndRestartLookups(t *testing.T) {
	database := testutil.NewMockDatabase()
	sender := &db.PlayerRecord{Name: "Sender"}
	recipient := &db.PlayerRecord{Name: "Recipient"}
	if err := database.CreatePlayer(sender); err != nil {
		t.Fatalf("create sender: %v", err)
	}
	if err := database.CreatePlayer(recipient); err != nil {
		t.Fatalf("create recipient: %v", err)
	}

	identity, err := newMailIdentity(database)
	if err != nil {
		t.Fatalf("newMailIdentity: %v", err)
	}
	if got := identity.idByName("recipient"); got != recipient.ID {
		t.Fatalf("case-insensitive recipient ID = %d, want %d", got, recipient.ID)
	}
	if got := identity.nameByID(sender.ID); got != sender.Name {
		t.Fatalf("offline sender name = %q, want %q", got, sender.Name)
	}

	// A record created after boot is resolved from the same persistent source
	// on a reverse-lookup miss; no online World entry is required.
	late := &db.PlayerRecord{Name: "LatePlayer"}
	if err := database.CreatePlayer(late); err != nil {
		t.Fatalf("create late player: %v", err)
	}
	if got := identity.nameByID(late.ID); got != late.Name {
		t.Fatalf("post-boot persistent name = %q, want %q", got, late.Name)
	}
}

func TestMailIdentityRenameRefreshesStableIDNameCache(t *testing.T) {
	database, err := db.New(filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	player := &db.PlayerRecord{Name: "Oldname"}
	if err := database.CreatePlayer(player); err != nil {
		t.Fatal(err)
	}
	identity, err := newMailIdentity(database)
	if err != nil {
		t.Fatal(err)
	}
	if got := identity.nameByID(player.ID); got != "Oldname" {
		t.Fatalf("cached pre-rename name = %q", got)
	}
	if _, err := database.Exec("UPDATE players SET name=? WHERE id=?", "Newname", player.ID); err != nil {
		t.Fatal(err)
	}
	identity.PlayerRenamed("Oldname", "Newname")
	if got := identity.nameByID(player.ID); got != "Newname" {
		t.Fatalf("mail sender/recipient name after rename = %q, want Newname", got)
	}
}

func TestMailIdentityRejectsIncompletePersistentAuthority(t *testing.T) {
	listErr := errors.New("player index unavailable")
	_, err := newMailIdentity(&mailIdentityFaultDB{
		GameStore: testutil.NewMockDatabase(),
		listErr:   listErr,
	})
	if !errors.Is(err, listErr) {
		t.Fatalf("newMailIdentity error = %v, want %v", err, listErr)
	}

	database := testutil.NewMockDatabase()
	identity, err := newMailIdentity(database)
	if err != nil {
		t.Fatalf("newMailIdentity with empty database: %v", err)
	}
	lookupErr := errors.New("player lookup unavailable")
	identity.database = &mailIdentityFaultDB{
		GameStore: database,
		getErr:    lookupErr,
	}
	if got := identity.idByName("Recipient"); got != -1 {
		t.Fatalf("failed ID lookup = %d, want -1", got)
	}

	_, err = newMailIdentity(&mailIdentityFaultDB{
		GameStore: testutil.NewMockDatabase(),
		listNames: []string{"Ghost"},
	})
	if err == nil {
		t.Fatal("newMailIdentity accepted a listed name with no persistent record")
	}
}

type mailIdentityFaultDB struct {
	db.GameStore
	listErr   error
	getErr    error
	listNames []string
}

func (f *mailIdentityFaultDB) ListPlayerNames() ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listNames != nil {
		return f.listNames, nil
	}
	return f.GameStore.ListPlayerNames()
}

func (f *mailIdentityFaultDB) GetPlayer(name string) (*db.PlayerRecord, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.GameStore.GetPlayer(name)
}
