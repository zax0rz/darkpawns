package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests run the game store against real databases: SQLite always (it is
// the dependency-free default), PostgreSQL when DATABASE_URL points at a
// disposable database. The fake-driver tests elsewhere in this package prove
// statements execute; these prove the statements are actually valid SQL on
// both dialects, that ids autoincrement, and that the schema is idempotent.
// Without them the SQLite path rots the way the store this package replaced
// did: pkg/storage (since deleted) only passed where cgo happened to be
// enabled, so nothing noticed that it could not work in the configuration the
// project ships.

type backend struct {
	name string
	dsn  string
}

// gameStoreBackends lists the backends for one test run.
func gameStoreBackends(t *testing.T) []backend {
	t.Helper()
	backends := []backend{{"sqlite", "sqlite://" + filepath.Join(t.TempDir(), "game.db")}}
	if pg := os.Getenv("DATABASE_URL"); pg != "" {
		backends = append(backends, backend{"postgres", pg})
	}
	return backends
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

// catalog lists tables and indexes through each dialect's catalog so schema
// assertions stay exact. Only explicitly named indexes are collected (SQLite
// autoindexes and PostgreSQL constraint indexes are implementation details).
func catalog(t *testing.T, database *DB) (tables, indexes map[string]bool) {
	t.Helper()
	tables = make(map[string]bool)
	indexes = make(map[string]bool)
	var rows *sql.Rows
	var err error
	if database.dialect == DialectSQLite {
		rows, err = database.conn.Query(
			`SELECT type, name FROM sqlite_master WHERE (type = 'table' OR type = 'index') AND name NOT LIKE 'sqlite_%'`)
	} else {
		rows, err = database.conn.Query(
			`SELECT 'table', tablename FROM pg_tables WHERE schemaname = current_schema()
			 UNION ALL SELECT 'index', indexname FROM pg_indexes WHERE schemaname = current_schema()`)
	}
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

var gameStoreTables = []string{"players", "agent_keys", "agent_narrative_memory", "agent_session_summaries"}

// gameStoreIndexes is every index the game store creates. (The migration brief
// said ten; the schema creates eleven, counted here by name.)
var gameStoreIndexes = []string{
	"players_name_folded_key",
	"idx_players_name",
	"idx_players_locked_until",
	"idx_anm_agent_name",
	"idx_anm_event_type",
	"idx_anm_salience",
	"idx_anm_social_event",
	"idx_anm_agent_session",
	"idx_anm_bootstrap",
	"idx_ass_agent_name",
	"idx_ass_session_id",
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

			_, keyID, err := database.CreateAgentKey(p.Name)
			if err != nil {
				t.Fatalf("CreateAgentKey: %v", err)
			}
			if keyID == 0 {
				t.Error("CreateAgentKey returned id 0")
			}

			memID, err := database.WriteNarrativeMemory(&NarrativeMemory{
				AgentName: p.Name,
				EventType: NarrEventMobKill,
				Summary:   "schema proof kill",
				Salience:  1.0,
			})
			if err != nil {
				t.Fatalf("WriteNarrativeMemory: %v", err)
			}
			if memID == 0 {
				t.Error("WriteNarrativeMemory returned id 0")
			}
		})
	}
}

// postgresDSN returns the disposable PostgreSQL DSN, skipping the test when
// DATABASE_URL is not set. The timestamp migration tests are PostgreSQL-only:
// SQLite has no zone-aware type to convert to, and no information_schema.
func postgresDSN(t *testing.T) string {
	t.Helper()
	pg := os.Getenv("DATABASE_URL")
	if pg == "" {
		t.Skip("set DATABASE_URL to run the PostgreSQL timestamp migration tests")
	}
	return pg
}

// wantGameStoreTimestamptz is every game-store column that must be zone-aware:
// the four naive TIMESTAMP sites the DDL used to author (players.locked_until
// via the migration-column list, players.created_at, players.updated_at,
// agent_keys.created_at). Spelled out rather than derived from the production
// list, so a new naive column that never made it into that list fails here
// instead of shipping.
var wantGameStoreTimestamptz = []string{
	"players.locked_until",
	"players.created_at",
	"players.updated_at",
	"agent_keys.created_at",
}

// columnType reads a column's declared type through information_schema, the way
// a DBA would check it.
func columnType(t *testing.T, database *DB, table, column string) string {
	t.Helper()
	var dataType string
	if err := database.queryRow(
		`SELECT data_type FROM information_schema.columns
		  WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`,
		table, column,
	).Scan(&dataType); err != nil {
		t.Fatalf("read type of %s.%s: %v", table, column, err)
	}
	return dataType
}

// TestGameStoreTimestamptzSchema proves the DDL half of the fix: a fresh
// PostgreSQL install authors zone-aware columns, so the defect cannot come back
// through CREATE TABLE or through the ADD COLUMN path that creates
// locked_until. It also pins the migration list against that schema, in both
// directions.
func TestGameStoreTimestamptzSchema(t *testing.T) {
	database := openGameStore(t, postgresDSN(t))
	for _, target := range wantGameStoreTimestamptz {
		if !containsString(gameStoreTimestamptzColumns, target) {
			t.Errorf("%s is zone-aware but missing from gameStoreTimestamptzColumns: an install that already has it naive would never be converted", target)
		}
		table, column, _ := strings.Cut(target, ".")
		if got := columnType(t, database, table, column); got != "timestamp with time zone" {
			t.Errorf("%s is declared %q, want timestamp with time zone", target, got)
		}
	}
	for _, target := range gameStoreTimestamptzColumns {
		if !containsString(wantGameStoreTimestamptz, target) {
			t.Errorf("gameStoreTimestamptzColumns lists %s, which is not a game-store timestamp column", target)
		}
	}
}

// TestMigrateNaiveTimestamptz builds the pre-migration schema in a throwaway
// table, converts it, and checks the three claims the migration owns: the types
// change, a second run moves no value, and rows written before the conversion
// survive with their stored wall clock read in the database session's zone.
func TestMigrateNaiveTimestamptz(t *testing.T) {
	database := openGameStore(t, postgresDSN(t))
	// Unique per process: this database may be shared with a concurrent run.
	probe := fmt.Sprintf("tz_migration_probe_%d", os.Getpid())
	if _, err := database.conn.Exec(`DROP TABLE IF EXISTS ` + probe); err != nil {
		t.Fatalf("drop stale probe table: %v", err)
	}
	t.Cleanup(func() {
		if _, err := database.conn.Exec(`DROP TABLE IF EXISTS ` + probe); err != nil {
			t.Errorf("drop probe table: %v", err)
		}
	})

	// The shape an older build left behind: naive columns, one carrying the
	// CURRENT_TIMESTAMP default the players table used and one the NOW()
	// default agent_keys used.
	if _, err := database.conn.Exec(`CREATE TABLE ` + probe + ` (
		id           SERIAL PRIMARY KEY,
		created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		locked_until TIMESTAMP
	)`); err != nil {
		t.Fatalf("create legacy probe table: %v", err)
	}
	if _, err := database.conn.Exec(
		`INSERT INTO ` + probe + ` (created_at, locked_until)
		 VALUES ('2000-01-01 00:00:00', '2030-06-15 12:30:00')`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	targets := []string{probe + ".created_at", probe + ".locked_until"}
	if err := database.convertNaiveTimestamps(targets); err != nil {
		t.Fatalf("first conversion: %v", err)
	}
	for _, column := range []string{"created_at", "locked_until"} {
		if got := columnType(t, database, probe, column); got != "timestamp with time zone" {
			t.Errorf("%s.%s is %q after conversion, want timestamp with time zone", probe, column, got)
		}
	}

	// The expected instants are derived from the database's own zone rather
	// than written as literals: the conversion attaches
	// current_setting('TimeZone'), so a fixed expectation would only hold on a
	// server whose zone happens to match it.
	var wantCreated, wantLocked time.Time
	if err := database.queryRow(
		`SELECT TIMESTAMP '2000-01-01 00:00:00' AT TIME ZONE current_setting('TimeZone'),
		        TIMESTAMP '2030-06-15 12:30:00' AT TIME ZONE current_setting('TimeZone')`,
	).Scan(&wantCreated, &wantLocked); err != nil {
		t.Fatalf("compute expected instants: %v", err)
	}
	var created, locked time.Time
	if err := database.queryRow(`SELECT created_at, locked_until FROM `+probe+` WHERE id = 1`).Scan(&created, &locked); err != nil {
		t.Fatalf("read converted row: %v", err)
	}
	if !created.Equal(wantCreated) {
		t.Errorf("created_at = %v, want %v: the stored wall clock is kept in the session's zone", created, wantCreated)
	}
	if !locked.Equal(wantLocked) {
		t.Errorf("locked_until = %v, want %v: the stored wall clock is kept in the session's zone", locked, wantLocked)
	}

	// The wall clock the game prints is the part that must not move, and it is
	// not the same claim as the instant above: read back in the session's zone
	// it is still the value that was stored.
	var wallClock string
	if err := database.queryRow(
		`SELECT to_char(created_at AT TIME ZONE current_setting('TimeZone'), 'YYYY-MM-DD HH24:MI:SS')
		   FROM ` + probe + ` WHERE id = 1`,
	).Scan(&wallClock); err != nil {
		t.Fatalf("read converted wall clock: %v", err)
	}
	if wallClock != "2000-01-01 00:00:00" {
		t.Errorf("created_at reads back as %q in the session's zone, want %q", wallClock, "2000-01-01 00:00:00")
	}

	// The second run is the idempotency proof, and it is not cosmetic: on an
	// already-converted column the AT TIME ZONE expression re-interprets the
	// value and shifts it by the host's offset, so without the
	// information_schema guard every boot would move the data again.
	if err := database.convertNaiveTimestamps(targets); err != nil {
		t.Fatalf("second conversion: %v", err)
	}
	var againCreated, againLocked time.Time
	if err := database.queryRow(`SELECT created_at, locked_until FROM `+probe+` WHERE id = 1`).Scan(&againCreated, &againLocked); err != nil {
		t.Fatalf("read row after second conversion: %v", err)
	}
	if !againCreated.Equal(created) || !againLocked.Equal(locked) {
		t.Errorf("second conversion moved values: created %v -> %v, locked %v -> %v",
			created, againCreated, locked, againLocked)
	}

	// A column this install does not have (an older build may predate it) is
	// skipped rather than treated as an error.
	if err := database.convertNaiveTimestamps([]string{probe + "_absent.created_at"}); err != nil {
		t.Errorf("converting an absent column: %v, want a silent skip", err)
	}

	// The column default has to survive the type change: rows after the
	// conversion rely on it.
	var insertedID int
	if err := database.queryRow(
		`INSERT INTO ` + probe + ` (locked_until) VALUES (NULL) RETURNING id`,
	).Scan(&insertedID); err != nil {
		t.Fatalf("insert relying on the default: %v", err)
	}
	var fresh sql.NullTime
	if err := database.queryRow(`SELECT created_at FROM `+probe+` WHERE id = $1`, insertedID).Scan(&fresh); err != nil {
		t.Fatalf("read defaulted created_at: %v", err)
	}
	if !fresh.Valid {
		t.Fatal("created_at default did not fire after the type change")
	}
	if d := time.Since(fresh.Time).Abs(); d > time.Minute {
		t.Errorf("defaulted created_at = %v (%v from now), want about now", fresh.Time, d)
	}
}

// TestMigrateNaiveTimestampsOnCurrentSchema runs the production entry point
// against a schema this build just created, on both backends. On PostgreSQL
// every listed column is already zone-aware, so the guard has to make it a
// silent no-op; on SQLite it has to skip outright, because there is no
// zone-aware type to convert to and no information_schema to ask.
func TestMigrateNaiveTimestampsOnCurrentSchema(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			if err := database.migrateNaiveTimestamps(); err != nil {
				t.Fatalf("migrateNaiveTimestamps on a current schema: %v", err)
			}
		})
	}
}

// TestGameStoreSchemaIdempotent proves the migration shim: running the schema
// a second time over the same database must be a clean no-op on both dialects.
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

// TestGameStoreSessionSummaryUpsert covers ON CONFLICT DO UPDATE and the
// NULLS LAST ordering of GetSessionSummaries.
func TestGameStoreSessionSummaryUpsert(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			agent := uniqueName("agent")
			now := time.Now()

			if err := database.WriteSessionSummary(agent, "s1", "first draft", 3, now.Add(-2*time.Hour), now.Add(-time.Hour)); err != nil {
				t.Fatalf("WriteSessionSummary: %v", err)
			}
			if err := database.WriteSessionSummary(agent, "s1", "revised", 5, now.Add(-2*time.Hour), now); err != nil {
				t.Fatalf("WriteSessionSummary upsert: %v", err)
			}
			if err := database.WriteSessionSummary(agent, "s2", "second session", 2, now, now.Add(30*time.Minute)); err != nil {
				t.Fatalf("WriteSessionSummary: %v", err)
			}

			summaries, err := database.GetSessionSummaries(agent, 10)
			if err != nil {
				t.Fatalf("GetSessionSummaries: %v", err)
			}
			if len(summaries) != 2 {
				t.Fatalf("summaries = %v, want 2 entries (upsert collapsed s1)", summaries)
			}
			if summaries[0] != "second session" {
				t.Errorf("most recent first: got %q", summaries[0])
			}
		})
	}
}

// TestGameStoreBootstrapMemories covers the salience-ordered bootstrap query
// and the partial-index social event filter.
func TestGameStoreBootstrapMemories(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			agent := uniqueName("agent")

			faint := &NarrativeMemory{AgentName: agent, EventType: NarrEventRoomVisit, Summary: "faint", Salience: 0.2}
			bright := &NarrativeMemory{AgentName: agent, EventType: NarrEventMobKill, Summary: "bright", Salience: 0.9}
			social := &NarrativeMemory{AgentName: agent, EventType: NarrEventPlayerEncounter, Summary: "social", Salience: 0.5, SocialEventID: "evt-1"}
			if _, err := database.WriteNarrativeMemory(faint); err != nil {
				t.Fatalf("WriteNarrativeMemory: %v", err)
			}
			if _, err := database.WriteNarrativeMemory(bright); err != nil {
				t.Fatalf("WriteNarrativeMemory: %v", err)
			}
			if _, err := database.WriteNarrativeMemory(social); err != nil {
				t.Fatalf("WriteNarrativeMemory: %v", err)
			}

			memories, err := database.BootstrapMemories(agent, 10)
			if err != nil {
				t.Fatalf("BootstrapMemories: %v", err)
			}
			if len(memories) != 3 {
				t.Fatalf("memories = %d, want 3", len(memories))
			}
			if memories[0].Summary != "bright" || memories[1].Summary != "social" {
				t.Errorf("salience order wrong: %q then %q", memories[0].Summary, memories[1].Summary)
			}
			if memories[0].CreatedAt.IsZero() || memories[0].UpdatedAt.IsZero() {
				t.Error("timestamps did not round-trip")
			}

			socialMemories, err := database.SocialEventMemories("evt-1")
			if err != nil {
				t.Fatalf("SocialEventMemories: %v", err)
			}
			if len(socialMemories) != 1 || socialMemories[0].Summary != "social" {
				t.Errorf("social event memories = %+v", socialMemories)
			}

			recent, err := database.RecentMemories(agent, memories[0].SessionID)
			if err != nil {
				t.Fatalf("RecentMemories: %v", err)
			}
			_ = recent // empty session id is a valid query; just prove it runs
		})
	}
}

// TestGameStoreMemoryDecay covers the decay UPDATE, including its
// CURRENT_TIMESTAMP assignment (delta: NOW() in DML is PostgreSQL-only), and
// the prune pass. Old rows are seeded with an explicit created_at through the
// package-internal exec so the rebind applies on both dialects.
func TestGameStoreMemoryDecay(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			database := openGameStore(t, be.dsn)
			agent := uniqueName("agent")
			old := time.Now().AddDate(0, 0, -60)

			seed := func(summary string, salience float64, valence int) {
				t.Helper()
				_, err := database.exec(
					`INSERT INTO agent_narrative_memory
						(agent_name, event_type, summary, valence, salience, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $6)`,
					agent, NarrEventRoomVisit, summary, valence, salience, old, old,
				)
				if err != nil {
					t.Fatalf("seed memory: %v", err)
				}
			}
			seed("decays only", 0.5, 0)      // 0.5 * 0.5 = 0.25, survives prune
			seed("decays to prune", 0.08, 0) // 0.08 * 0.5 = 0.04, pruned

			decayed, pruned, err := database.DecayStaleMemories(30)
			if err != nil {
				t.Fatalf("DecayStaleMemories: %v", err)
			}
			if decayed != 2 {
				t.Errorf("decayed = %d, want 2", decayed)
			}
			if pruned != 1 {
				t.Errorf("pruned = %d, want 1", pruned)
			}

			var survivors int
			if err := database.queryRow(
				`SELECT COUNT(*) FROM agent_narrative_memory WHERE agent_name = $1 AND summary = 'decays only'`,
				agent,
			).Scan(&survivors); err != nil {
				t.Fatalf("count survivors: %v", err)
			}
			if survivors != 1 {
				t.Errorf("survivors = %d, want 1", survivors)
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

func TestSplitDSN(t *testing.T) {
	tests := []struct {
		in      string
		dialect Dialect
		dsn     string
		wantErr bool
	}{
		{"postgres://u:p@host/db", DialectPostgres, "postgres://u:p@host/db", false},
		{"postgres:///darkpawns?host=/var/run/postgresql", DialectPostgres, "postgres:///darkpawns?host=/var/run/postgresql", false},
		{"postgresql://u:p@host/db", DialectPostgres, "postgresql://u:p@host/db", false},
		{"sqlite:///tmp/game.db", DialectSQLite, "/tmp/game.db", false},
		{"sqlite://:memory:", DialectSQLite, ":memory:", false},
		{"data/darkpawns.db", DialectSQLite, "data/darkpawns.db", false},
		{":memory:", DialectSQLite, ":memory:", false},
		{"", 0, "", true},
		{"   ", 0, "", true},
	}
	for _, tt := range tests {
		d, dsn, err := SplitDSN(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("SplitDSN(%q): want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("SplitDSN(%q): %v", tt.in, err)
			continue
		}
		if d != tt.dialect || dsn != tt.dsn {
			t.Errorf("SplitDSN(%q) = (%v, %q), want (%v, %q)", tt.in, d, dsn, tt.dialect, tt.dsn)
		}
	}
}

func TestRebind(t *testing.T) {
	const q = `INSERT INTO players (name, password_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	if got := DialectPostgres.Rebind(q); got != q {
		t.Errorf("postgres rebind changed the query: %q", got)
	}
	want := `INSERT INTO players (name, password_hash) VALUES (?, ?) ON CONFLICT DO NOTHING`
	if got := DialectSQLite.Rebind(q); got != want {
		t.Errorf("sqlite rebind = %q, want %q", got, want)
	}
}

// TestExpandRepeatedPlaceholders covers the argument contract shared by the
// dialect choke points: one argument per placeholder occurrence, in order.
// SQLite binds positionally and already behaves that way; PostgreSQL numbers
// distinct parameters, so a repeated marker has to be expanded or lib/pq
// rejects the call before it reaches the server.
func TestExpandRepeatedPlaceholders(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		args      []interface{}
		wantQuery string
		wantArgs  []interface{}
	}{
		{
			name:      "no placeholders",
			query:     `SELECT 1`,
			args:      nil,
			wantQuery: `SELECT 1`,
			wantArgs:  nil,
		},
		{
			name:      "distinct placeholders unchanged",
			query:     `SELECT $1, $2`,
			args:      []interface{}{"a", "b"},
			wantQuery: `SELECT $1, $2`,
			wantArgs:  []interface{}{"a", "b"},
		},
		{
			name:      "repeated marker expands one per occurrence",
			query:     `INSERT INTO t (a, b, c) VALUES ($1, $2, $2)`,
			args:      []interface{}{1, "same", "same"},
			wantQuery: `INSERT INTO t (a, b, c) VALUES ($1, $2, $3)`,
			wantArgs:  []interface{}{1, "same", "same"},
		},
		{
			// The shape TestGameStoreMemoryDecay seeds with: $6 twice, one
			// argument per occurrence.
			name:      "trailing repeated marker (decay seed shape)",
			query:     `VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			args:      []interface{}{1, 2, 3, 4, 5, "old", "old"},
			wantQuery: `VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			wantArgs:  []interface{}{1, 2, 3, 4, 5, "old", "old"},
		},
		{
			// The classic PostgreSQL idiom passes one argument for a marker
			// used twice; that must pass through untouched.
			name:      "postgres-style reuse untouched",
			query:     `SELECT $1, $1`,
			args:      []interface{}{"only"},
			wantQuery: `SELECT $1, $1`,
			wantArgs:  []interface{}{"only"},
		},
		{
			name:      "argument count mismatch untouched",
			query:     `SELECT $1, $2`,
			args:      []interface{}{"only"},
			wantQuery: `SELECT $1, $2`,
			wantArgs:  []interface{}{"only"},
		},
		{
			name:      "out-of-range marker untouched",
			query:     `SELECT $1, $3`,
			args:      []interface{}{"a", "b"},
			wantQuery: `SELECT $1, $3`,
			wantArgs:  []interface{}{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotArgs := expandRepeatedPlaceholders(tt.query, tt.args)
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			// The rewrite must never leave the positional dialect with a
			// marker/argument mismatch. Only statements following the
			// one-argument-per-occurrence convention are checkable here; the
			// PostgreSQL-style cases above are meant to pass through.
			if occurrences := len(postgresPlaceholder.FindAllStringIndex(gotQuery, -1)); occurrences == len(gotArgs) {
				if sqlite := DialectSQLite.Rebind(gotQuery); strings.Count(sqlite, "?") != len(gotArgs) {
					t.Errorf("sqlite markers = %d, args = %d", strings.Count(sqlite, "?"), len(gotArgs))
				}
			}
		})
	}
}

var nameCounter int

// uniqueName returns a player name unique to this test process so PostgreSQL
// runs (which share one disposable database) do not collide across tests.
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

// TestMigrateCanonicalizingJSON pins the jsonb→json half of the first-boot
// migration: a legacy install's canonicalizing columns become exact-text json,
// content survives, and a second boot is a no-op.
func TestMigrateCanonicalizingJSON(t *testing.T) {
	database := openGameStore(t, postgresDSN(t))
	probe := fmt.Sprintf("json_migration_probe_%d", os.Getpid())
	if _, err := database.conn.Exec(`DROP TABLE IF EXISTS ` + probe); err != nil {
		t.Fatalf("drop stale probe table: %v", err)
	}
	t.Cleanup(func() {
		if _, err := database.conn.Exec(`DROP TABLE IF EXISTS ` + probe); err != nil {
			t.Errorf("drop probe table: %v", err)
		}
	})
	// The shape an older build left behind: canonicalizing jsonb columns.
	if _, err := database.conn.Exec(`CREATE TABLE ` + probe + ` (
		id        SERIAL PRIMARY KEY,
		inventory JSONB,
		equipment JSONB
	)`); err != nil {
		t.Fatalf("create legacy probe table: %v", err)
	}
	const seed = `[{"vnum":3032,"count":1,"locate":0,"state":null}]`
	if _, err := database.conn.Exec(
		`INSERT INTO `+probe+` (inventory, equipment) VALUES ($1::jsonb, '{}'::jsonb)`, seed,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// The legacy row's canonicalized content must survive the conversion
	// unchanged: capture what jsonb rendered before the migration runs, and
	// require the same text after. (The seeded literal itself is gone the
	// moment jsonb stores it — key order and spacing were normalized at
	// insert — so "unchanged" is the claim, not "restored".)
	var before, got string
	if err := database.conn.QueryRow(`SELECT inventory FROM ` + probe + ` WHERE id = 1`).Scan(&before); err != nil {
		t.Fatalf("read legacy row: %v", err)
	}

	targets := []string{probe + ".inventory", probe + ".equipment"}
	if err := database.convertColumns(targets, "jsonb", "JSON", func(column string) string {
		return column + "::json"
	}); err != nil {
		t.Fatalf("first conversion: %v", err)
	}
	for _, column := range []string{"inventory", "equipment"} {
		var dataType string
		if err := database.conn.QueryRow(
			`SELECT data_type FROM information_schema.columns
			  WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`,
			probe, column,
		).Scan(&dataType); err != nil || dataType != "json" {
			t.Errorf("%s.%s is %q (err %v) after conversion, want json", probe, column, dataType, err)
		}
	}
	if err := database.conn.QueryRow(`SELECT inventory FROM ` + probe + ` WHERE id = 1`).Scan(&got); err != nil {
		t.Fatalf("read converted row: %v", err)
	}
	if got != before {
		t.Errorf("inventory content changed across conversion: before %q, after %q", before, got)
	}
	// After the conversion the column is exact-text json: a write comes back
	// byte-identical, which is the property jsonb never had.
	if _, err := database.conn.Exec(`UPDATE `+probe+` SET inventory = $1 WHERE id = 1`, seed); err != nil {
		t.Fatalf("write exact text: %v", err)
	}
	if err := database.conn.QueryRow(`SELECT inventory FROM ` + probe + ` WHERE id = 1`).Scan(&got); err != nil || got != seed {
		t.Errorf("inventory content = %q (err %v) after exact-text write, want %q", got, err, seed)
	}
	// Second boot: the type guard finds nothing to convert, and the row is
	// untouched.
	if err := database.convertColumns(targets, "jsonb", "JSON", func(column string) string {
		return column + "::json"
	}); err != nil {
		t.Fatalf("second conversion: %v", err)
	}
	if err := database.conn.QueryRow(`SELECT inventory FROM ` + probe + ` WHERE id = 1`).Scan(&got); err != nil || got != seed {
		t.Errorf("inventory content = %q (err %v) after second boot, want unchanged", got, err)
	}
}
