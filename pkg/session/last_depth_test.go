package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLastRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["last"]
	if !ok {
		t.Fatal("last command has no C gate")
	}
	if gate.MinLevel != LVL_GOD-1 || gate.MinPosition != combat.PosDead {
		t.Fatalf("last gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GOD-1, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("last")
	if !ok {
		t.Fatal("last command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("last registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestLastUsesCOneArgumentAndMissingPlayerText(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Lastgod", LVL_IMPL, 1001)

	if err := cmdLast(s, []string{"the", "Nobody", "trailing"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	if got, want := readSessionText(t, s), "There is no such player.\r\n"; got != want {
		t.Fatalf("cmdLast missing target = %q, want %q", got, want)
	}
}

// TestLastNoDatabaseReportsMissing pins the deleted no-store branch's
// replacement: C's do_last loads the pfile (src/act.wizard.c:1841-1844), and
// a runtime with no store has no pfiles, so every lookup is the miss line.
// The online-session fallback that used to live here rebuilt C's line from
// the live connection; it existed only for the store-less census (DP-1479).
func TestLastNoDatabaseReportsMissing(t *testing.T) {
	m := makeTestManager(t)
	target := makeTestSession(t, m, "Lastpeer", 1001, true)
	target.player = game.NewCharacter(2, "Lastpeer", game.ClassWarrior, game.RaceHuman)
	if err := m.world.AddPlayer(target.player); err != nil {
		t.Fatalf("add player: %v", err)
	}
	m.mu.Lock()
	m.sessions[target.playerName] = target
	m.mu.Unlock()

	actor := makeTestSession(t, m, "Lastactor", 1001, true)
	actor.player = game.NewCharacter(1, "Lastactor", game.ClassWarrior, game.RaceHuman)
	actor.player.Level = game.LVL_IMPL
	if err := cmdLast(actor, []string{"the", "LASTPEER", "ignored"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	if got, want := readSessionText(t, actor), "There is no such player.\r\n"; got != want {
		t.Fatalf("no-store last = %q, want %q", got, want)
	}
}

// TestLastPrintsSavedHostInCFormat pins do_last's exact line from the saved
// record (src/act.wizard.c:1847): "[%5ld] [%2d %s] %-12s : %-18s : %-20s\r\n"
// with the saved host and ctime(&last_logon). ctime's string ends in \n and
// outgrows the 20-column field, so the output ends "<time>\n\r\n".
// Red against the pre-DP-1479 invented format "[...] : Level N".
func TestLastPrintsSavedHostInCFormat(t *testing.T) {
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "last.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	m := makeTestManagerWithVoidRoomsAndDB(t, database)

	seed := &db.PlayerRecord{
		Name: "Hostman", RoomVNum: game.MortalStartRoom,
		Level: 5, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
		Host:      "127.000.000.001",
		LastLogon: time.Date(2026, time.March, 5, 7, 8, 9, 0, time.Local).Unix(),
	}
	if err := database.CreatePlayer(seed); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}

	actor := makeCommandTestSession(t, m, "Lastgod", game.LVL_IMPL, 1001)
	if err := cmdLast(actor, []string{"hostman"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	want := "[    1] [ 5 Wa] Hostman      : 127.000.000.001    : Thu Mar  5 07:08:09 2026\n\r\n"
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("last = %q, want %q", got, want)
	}
}

// TestLastRecordWithoutHostColumnReadsEmpty simulates a pre-DP-1479 row: the
// column absent (NULL), read back as the empty host C's empty-record host
// prints as.
func TestLastRecordWithoutHostColumnReadsEmpty(t *testing.T) {
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	m := makeTestManagerWithVoidRoomsAndDB(t, database)

	seed := &db.PlayerRecord{
		Name: "Oldhand", RoomVNum: game.MortalStartRoom,
		Level: 3, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassThief, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
		LastLogon: time.Date(2026, time.March, 5, 7, 8, 9, 0, time.Local).Unix(),
	}
	if err := database.CreatePlayer(seed); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	// A row written before the column existed reads NULL.
	if _, err := database.Exec("UPDATE players SET host = NULL"); err != nil {
		t.Fatalf("null out host: %v", err)
	}

	actor := makeCommandTestSession(t, m, "Lastgod", game.LVL_IMPL, 1001)
	if err := cmdLast(actor, []string{"oldhand"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	// %-18s of "" is 18 spaces.
	want := "[    1] [ 3 Th] Oldhand      :                    : Thu Mar  5 07:08:09 2026\n\r\n"
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("last = %q, want %q", got, want)
	}
}

// TestSaveStoresSessionHostAndEmptyWithoutDescriptor pins save_char's host
// rule (src/db.c:2373-2381): the live descriptor's host goes into the
// record; a detached session — the linkless body's later saves — stores the
// empty host.
func TestSaveStoresSessionHostAndEmptyWithoutDescriptor(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Hostsave", game.LVL_IMPL, 1001)
	s.player = game.NewCharacter(1, "Hostsave", game.ClassWarrior, game.RaceHuman)
	s.player.ID = 1
	s.remoteIP = "10.1.2.3"

	live, err := s.playerRecordForSave(s.player, game.MortalStartRoom)
	if err != nil {
		t.Fatalf("playerRecordForSave: %v", err)
	}
	if got, want := live.Host, "010.001.002.003"; got != want {
		t.Fatalf("live save host = %q, want the padded quad %q", got, want)
	}

	s.DetachTransport()
	detached, err := s.playerRecordForSave(s.player, game.MortalStartRoom)
	if err != nil {
		t.Fatalf("playerRecordForSave after detach: %v", err)
	}
	if got := detached.Host; got != "" {
		t.Fatalf("detached save host = %q, want empty (db.c:2379-2381)", got)
	}
}
