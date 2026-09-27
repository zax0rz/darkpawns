package db

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests run the game store against a real SQLite database, which is the
// only backend there is. The fake-driver tests elsewhere in this package prove
// statements execute; these prove the statements are valid SQL, that ids
// autoincrement, and that the schema is idempotent. Without them the store rots
// the way the package this one replaced did: pkg/storage (since deleted) only
// passed where cgo happened to be enabled, so nothing noticed that it could not
// work in the configuration the project ships.

type backend struct {
	name string
	dsn  string
}

// gameStoreBackends lists the backends for one test run: the embedded SQLite
// file, which is the runtime's only database. The slice stays (rather than a
// single inlined DSN) because the tests below iterate it, and a second backend
// would be a second schema to keep in step again.
func gameStoreBackends(t *testing.T) []backend {
	t.Helper()
	return []backend{{"sqlite", "sqlite://" + filepath.Join(t.TempDir(), "game.db")}}
}

func openGameStore(t *testing.T, dsn string) *DB {
	t.Helper()
	database, err := New(dsn)
	if err != nil {
		t.Fatalf("New(%q): %v", dsn, err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// catalog lists tables and explicitly named indexes through SQLite's catalog so
// schema assertions stay exact. SQLite's autoindexes are implementation details
// and are not collected.
func catalog(t *testing.T, database *DB) (tables, indexes map[string]bool) {
	t.Helper()
	tables = make(map[string]bool)
	indexes = make(map[string]bool)
	rows, err := database.conn.Query(
		`SELECT type, name FROM sqlite_master WHERE (type = 'table' OR type = 'index') AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind, name string
		if err := rows.Scan(&kind, &name); err != nil {
			t.Fatalf("scan catalog: %v", err)
		}
		if kind == "table" {
			tables[name] = true
		} else {
			indexes[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate catalog: %v", err)
	}
	return tables, indexes
}

var gameStoreTables = []string{"players"}

// gameStoreIndexes is every index the game store creates, counted by name.
var gameStoreIndexes = []string{
	"players_name_folded_key",
	"idx_players_name",
	"idx_players_locked_until",
}

// TestGameStoreSchema is the delta-1 proof: it asserts on a real INSERT ...
// RETURNING id producing a non-zero id, never on CREATE TABLE succeeding.
// SQLite accepts SERIAL PRIMARY KEY silently and never autoincrements, so a
// test that only checked the DDL would ship null ids.
func TestGameStoreSchema(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)

			tables, indexes := catalog(t, database)
			for _, table := range gameStoreTables {
				if !tables[table] {
					t.Errorf("missing table %s", table)
				}
			}
			for _, index := range gameStoreIndexes {
				if !indexes[index] {
					t.Errorf("missing index %s", index)
				}
			}

			p := &PlayerRecord{
				Name:      uniqueName("schema"),
				Password:  "hash",
				RoomVNum:  8004,
				Inventory: []byte("[]"),
				Equipment: []byte("{}"),
			}
			if err := database.CreatePlayer(p); err != nil {
				t.Fatalf("CreatePlayer: %v", err)
			}
			if p.ID == 0 {
				t.Error("CreatePlayer returned id 0; SERIAL was not translated and the column never autoincrements")
			}
		})
	}
}

func TestGameStoreSchemaIdempotent(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			first := openGameStore(t, be.dsn)
			_ = first
			// Second open of the same DSN re-runs createTables, the 23 guarded
			// ALTERs and every CREATE INDEX over the existing schema.
			second, err := New(be.dsn)
			if err != nil {
				t.Fatalf("re-open after schema already applied: %v", err)
			}
			_ = second.Close()
		})
	}
}

// TestGameStoreIdsIncrease proves autoincrement further than "id is non-zero":
// two inserts of the same shape must yield distinct increasing ids.
func TestGameStoreIdsIncrease(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			var ids []int
			for i := 0; i < 3; i++ {
				p := &PlayerRecord{Name: uniqueName("ids"), Inventory: []byte("[]"), Equipment: []byte("{}")}
				if err := database.CreatePlayer(p); err != nil {
					t.Fatalf("CreatePlayer: %v", err)
				}
				ids = append(ids, p.ID)
			}
			for i := 1; i < len(ids); i++ {
				if ids[i] <= ids[i-1] {
					t.Errorf("ids not increasing: %v", ids)
				}
			}
		})
	}
}

func TestGameStorePlayerRoundTrip(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			name := uniqueName("zara")

			p := &PlayerRecord{
				Name:       name,
				Password:   "hash",
				RoomVNum:   3001,
				Level:      5,
				Exp:        1200,
				Health:     42,
				MaxHealth:  60,
				Mana:       10,
				MaxMana:    100,
				Move:       90,
				MaxMove:    100,
				Strength:   16,
				Class:      3,
				Race:       2,
				StatStr:    16,
				StatStrAdd: 2,
				StatInt:    8,
				StatWis:    9,
				StatDex:    14,
				StatCon:    15,
				StatCha:    11,
				Hunger:     20,
				Thirst:     20,
				Drunk:      1,
				Hometown:   7,
				OlcZone:    30,
				Inventory:  []byte(`[{"vnum":3032,"count":1,"locate":0,"state":null}]`),
				Equipment:  []byte("{}"),
			}
			if err := database.CreatePlayer(p); err != nil {
				t.Fatalf("CreatePlayer: %v", err)
			}

			// lower(name) lookup: a differently-cased name must resolve.
			got, err := database.GetPlayer(strings.ToUpper(name[:1]) + name[1:])
			if err != nil {
				t.Fatalf("GetPlayer case-insensitive: %v", err)
			}
			if got == nil || got.Name != name || got.Level != 5 || got.StatStrAdd != 2 || got.OlcZone != 30 || !bytes.Equal(got.Inventory, p.Inventory) {
				t.Errorf("round trip mismatch: %+v", got)
			}

			// The folded unique index must reject a case-colliding name.
			dup := *p
			dup.Name = name[:1] + strings.ToUpper(name[1:])
			if err := database.CreatePlayer(&dup); err == nil {
				t.Error("case-colliding CreatePlayer succeeded; folded unique index not enforced")
			}

			p.Level = 9
			p.Exp = 5000
			p.OlcZone = 31
			if err := database.SavePlayer(p); err != nil {
				t.Fatalf("SavePlayer: %v", err)
			}
			got, err = database.GetPlayer(name)
			if err != nil {
				t.Fatalf("GetPlayer after save: %v", err)
			}
			if got.Level != 9 || got.Exp != 5000 || got.OlcZone != 31 {
				t.Errorf("save did not persist: level=%d exp=%d", got.Level, got.Exp)
			}

			names, err := database.ListPlayerNames()
			if err != nil {
				t.Fatalf("ListPlayerNames: %v", err)
			}
			if !containsString(names, name) {
				t.Errorf("ListPlayerNames missing %q: %v", name, names)
			}

			n, err := database.CountPlayers()
			if err != nil {
				t.Fatalf("CountPlayers: %v", err)
			}
			if n < 1 {
				t.Errorf("CountPlayers = %d, want >= 1", n)
			}

			if err := database.UpdateDescription(p.ID, "A weathered traveler."); err != nil {
				t.Fatalf("UpdateDescription: %v", err)
			}
			if err := database.UpdatePassword(p.ID, "newhash"); err != nil {
				t.Fatalf("UpdatePassword: %v", err)
			}
			got, err = database.GetPlayer(name)
			if err != nil {
				t.Fatalf("GetPlayer after updates: %v", err)
			}
			if got.Description != "A weathered traveler." || got.Password != "newhash" {
				t.Errorf("updates did not persist: %+v", got)
			}
		})
	}
}

// TestGameStoreAccountLockout exercises the rewritten RecordLoginFailure: the
// lockout deadline is computed in Go and passed as a parameter, which is what
// makes the statement portable across dialects.
func TestGameStoreAccountLockout(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			name := uniqueName("lockout")
			p := &PlayerRecord{Name: name, Inventory: []byte("[]"), Equipment: []byte("{}")}
			if err := database.CreatePlayer(p); err != nil {
				t.Fatalf("CreatePlayer: %v", err)
			}

			const threshold = 3
			var locked bool
			var err error
			for i := 0; i < threshold; i++ {
				locked, err = database.RecordLoginFailure(name, threshold, 15*time.Minute)
				if err != nil {
					t.Fatalf("RecordLoginFailure: %v", err)
				}
			}
			if !locked {
				t.Error("account not locked at threshold")
			}

			attempts, until, err := database.GetAccountLockout(name)
			if err != nil {
				t.Fatalf("GetAccountLockout: %v", err)
			}
			if attempts != threshold {
				t.Errorf("attempts = %d, want %d", attempts, threshold)
			}
			if until == nil {
				t.Fatal("locked_until is nil after threshold")
			}
			if time.Until(*until) < 10*time.Minute {
				t.Errorf("locked_until too soon: %v", until)
			}

			if err := database.RecordLoginSuccess(name); err != nil {
				t.Fatalf("RecordLoginSuccess: %v", err)
			}
			attempts, until, err = database.GetAccountLockout(name)
			if err != nil {
				t.Fatalf("GetAccountLockout after success: %v", err)
			}
			if attempts != 0 || until != nil {
				t.Errorf("lockout not cleared: attempts=%d until=%v", attempts, until)
			}
		})
	}
}

// TestSQLiteJournalMode proves the WAL setup on the file-backed default.
func TestSQLiteJournalMode(t *testing.T) {
	database := openGameStore(t, "sqlite://"+filepath.Join(t.TempDir(), "game.db"))
	var mode string
	if err := database.conn.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

var nameCounter int

// uniqueName returns a player name unique to this test process, so two runs that
// share a database file do not collide across tests.
func uniqueName(prefix string) string {
	nameCounter++
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()+int64(nameCounter))
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
