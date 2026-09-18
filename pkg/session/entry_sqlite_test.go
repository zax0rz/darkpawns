package session

import (
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// TestEntryAcceptedStatsPersistSQLite proves the session entry path runs
// against the embedded SQLite store, not only against the PostgreSQL contract
// suite in entry_persistence_test.go. The flow mirrors
// TestEntryAcceptedStatsPersistBeforeMenu.
func TestEntryAcceptedStatsPersistSQLite(t *testing.T) {
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "entry.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	entrySeed(t, database, "Founder") // Exercise the mortal path.
	s := entrySession(t, database)
	s.startNewCharFlow("Newhero")
	for _, line := range []string{"Y", "oraclepass", "oraclepass", "N", "M", "K", "T", "K", "Y"} {
		sendCharInput(t, s, line)
	}
	p, err := database.GetPlayer("Newhero")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("accepted stats reached MOTD without persisting the character")
	}
}
