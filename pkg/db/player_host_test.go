package db

import (
	"path/filepath"
	"testing"
)

// TestHostRoundTripsThroughSaveAndLoad pins the host column DP-1479 added
// (C char_file_u.host, written by save_char from the descriptor,
// src/db.c:2373-2381): create, update and read all carry it.
func TestHostRoundTripsThroughSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.db")
	database, err := New("sqlite://" + path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer database.Close() //nolint:errcheck // test teardown

	p := &PlayerRecord{
		Name: "Roundtrip", RoomVNum: 8004,
		Level: 2, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: 3, Race: 0,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
		Host: "192.168.000.007",
	}
	if err := database.CreatePlayer(p); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	got, err := database.GetPlayer("roundtrip")
	if err != nil || got == nil {
		t.Fatalf("GetPlayer: %v %v", got, err)
	}
	if got.Host != "192.168.000.007" {
		t.Fatalf("host after create = %q, want the saved padded quad", got.Host)
	}

	got.Host = "resolved.example.net"
	if err := database.SavePlayer(got); err != nil {
		t.Fatalf("SavePlayer: %v", err)
	}
	again, err := database.GetPlayer("roundtrip")
	if err != nil || again == nil {
		t.Fatalf("GetPlayer after save: %v %v", again, err)
	}
	if again.Host != "resolved.example.net" {
		t.Fatalf("host after update = %q, want the resolved name", again.Host)
	}
}
