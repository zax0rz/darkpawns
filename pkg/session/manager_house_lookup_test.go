package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestManagerUsesDBLookupForOfflinePlayers pins DP-1572 at the session seam:
// when NewManager receives a database, the house get_id_by_name / get_name_by_id
// pair must resolve offline players (src/db.c:2313-2336). The old online-only
// overwrite made house guest/transfer and hcontrol build/show answer
// "No such player." for seeded offline records.
func TestManagerUsesDBLookupForOfflinePlayers(t *testing.T) {
	// DoHouse saves the control file under ./house; keep it out of the tree.
	t.Chdir(t.TempDir())
	t.Cleanup(func() { _ = os.RemoveAll("house") })
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.CreatePlayer(&db.PlayerRecord{Name: "Offlineorak"}); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	nameByID, idByName := database.HousePlayerLookups()
	offlineID := idByName("offlineorak")
	if offlineID <= 0 {
		t.Fatalf("idByName(offlineorak) = %d, want a real id", offlineID)
	}
	// World without the offline player online; owner is present.
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 8162, Name: "House", Zone: 1, Flags: []string{"house"}},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	owner := game.NewPlayer(1, "Houseowner", 8162)
	owner.SetLevel(40)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatalf("AddPlayer owner: %v", err)
	}
	// Seed a house owned by the owner so DoHouse has a control row.
	w.HouseControl = []game.HouseControl{{VNum: 8162, Owner: 1}}
	// NewManager must wire the DB pair, not the online scan.
	_ = newTestManager(t, w, database)
	// Sanity: the DB pair itself resolves the offline record.
	if got := nameByID(offlineID); got != "Offlineorak" {
		t.Fatalf("nameByID(offline) = %q, want Offlineorak", got)
	}
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	// If NewManager fell back to the online scan this would be
	// "No such player." even though the DB row exists.
	w.DoHouse(owner, "guest Offlineorak")
	if !strings.Contains(out.String(), "Guest added.") {
		t.Fatalf("house guest offline = %q, want Guest added", out.String())
	}
	if got := w.HouseControl[0].Guests[0]; got != offlineID {
		t.Fatalf("guest id = %d, want %d", got, offlineID)
	}
	// Case-insensitive like C str_cmp: lower-case still resolves offline.
	out.Reset()
	w.DoHouse(owner, "guest offlineorak")
	if !strings.Contains(out.String(), "Guest deleted.") {
		t.Fatalf("toggle offline guest = %q, want Guest deleted", out.String())
	}
}

// TestManagerNilDBFallsBackToOnlineScan documents the no-DB fallback: offline
// lookups correctly miss when there is no store (production always has one).
func TestManagerNilDBFallsBackToOnlineScan(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Cleanup(func() { _ = os.RemoveAll("house") })
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 8162, Name: "House", Zone: 1, Flags: []string{"house"}},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	owner := game.NewPlayer(1, "Houseowner", 8162)
	owner.SetLevel(40)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.HouseControl = []game.HouseControl{{VNum: 8162, Owner: 1}}
	_ = newTestManager(t, w, nil)
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	w.DoHouse(owner, "guest Ghostplayer")
	if !strings.Contains(out.String(), "No such player.") {
		t.Fatalf("offline without DB = %q, want No such player", out.String())
	}
	// Online player still resolves via the scan.
	online := game.NewPlayer(2, "Onlinepal", 8162)
	online.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(online); err != nil {
		t.Fatalf("AddPlayer online: %v", err)
	}
	out.Reset()
	w.DoHouse(owner, "guest Onlinepal")
	if !strings.Contains(out.String(), "Guest added.") {
		t.Fatalf("online guest = %q, want Guest added", out.String())
	}
}
