package db

import (
	"testing"
)

func TestPlayerLastLogonMigrationAndWrites(t *testing.T) {
	d := openGameStore(t, gameStoreBackends(t)[0].dsn)
	p := &PlayerRecord{Name: "Metadata", LastLogon: 12345}
	if err := d.CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("UPDATE players SET last_logon=NULL, updated_at='2001-01-01 00:00:00' WHERE id=?", p.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.createTables(); err != nil {
		t.Fatal(err)
	}
	r, err := d.GetPlayer("metadata")
	if err != nil {
		t.Fatal(err)
	}
	if r.LastLogon != 978307200 {
		t.Fatalf("legacy date=%d", r.LastLogon)
	}
	r.LastLogon = 123456
	if err := d.SavePlayer(r); err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateDescription(p.ID, "edited"); err != nil {
		t.Fatal(err)
	}
	if err := d.createTables(); err != nil {
		t.Fatal(err)
	}
	saved, err := d.GetPlayer(p.Name)
	if err != nil {
		t.Fatal(err)
	}
	if saved.LastLogon != 123456 {
		t.Fatal("general mutation or restart overwrote C timestamp")
	}
	newer := *saved
	newer.LastLogon = 123457
	if err := d.SavePlayer(&newer); err != nil {
		t.Fatal(err)
	}
	saved.Level = 3
	if err := SavePlayerIfCurrent(d, saved, saved); err != ErrPlayerRecordChanged {
		t.Fatalf("stale timestamp edit=%v", err)
	}
}
