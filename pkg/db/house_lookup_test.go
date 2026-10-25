package db

// house_lookup_test.go — the player-id lookups the house system registers at
// startup (cmd/server/main.go → game.RegisterHousePlayerLookup). C's helpers
// read the player index file (src/house.c:41-58); the port reads the players
// table with GetPlayer's case-insensitive name matching.

import (
	"path/filepath"
	"testing"
)

func TestHousePlayerLookups(t *testing.T) {
	database := openGameStore(t, "sqlite://"+filepath.Join(t.TempDir(), "game.db"))
	for _, name := range []string{"Houselord", "Egon"} {
		if err := database.CreatePlayer(&PlayerRecord{Name: name, Level: 1}); err != nil {
			t.Fatalf("CreatePlayer(%q): %v", name, err)
		}
	}

	nameByID, idByName := database.HousePlayerLookups()

	lordID := idByName("houselord") // case-insensitive, like GetPlayer
	if lordID <= 0 {
		t.Fatalf("idByName(houselord) = %d, want a real id", lordID)
	}
	if got, want := nameByID(lordID), "Houselord"; got != want {
		t.Fatalf("nameByID(%d) = %q, want %q", lordID, got, want)
	}
	if got := idByName("nobody"); got != -1 {
		t.Fatalf("idByName(nobody) = %d, want -1", got)
	}
	if got := nameByID(999999); got != "" {
		t.Fatalf("nameByID(999999) = %q, want empty", got)
	}
}
