package dbmigrate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// These tests run against a real PostgreSQL database, and they need one to
// themselves: DP_MIGRATE_TEST_DB_URL, not the shared DATABASE_URL.
//
// The reason is not tidiness. A conversion reads and writes the five production
// table names, so these tests create them in their own PostgreSQL schema and drop
// them afterwards. information_schema is per database, not per schema: while a
// test schema holds player_penalties, any other package's test that asks
// information_schema about that table by name alone sees it. pkg/moderation's
// columnExists helper does exactly that, so sharing one database between this
// package and that one makes the moderation tests fail for a reason that has
// nothing to do with moderation. A dedicated database keeps the converter's
// tests honest and its neighbours untouched.
//
// CI creates the database and sets the variable; without it these tests skip.

func requirePostgres(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DP_MIGRATE_TEST_DB_URL")
	if dsn == "" {
		t.Skip("set DP_MIGRATE_TEST_DB_URL to a disposable PostgreSQL database to run the migration tests")
	}
	return dsn
}

func withSearchPath(dsn, schema string) string {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "options=-csearch_path%3D" + schema
}

// newSourceSchema creates a private schema and returns its DSN plus the matching
// plain *sql.DB for assertions. The schema is dropped when the test ends.
func newSourceSchema(t *testing.T) (string, *sql.DB) {
	t.Helper()
	base := requirePostgres(t)
	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	schema := fmt.Sprintf("dp_mig_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		_ = admin.Close()
		t.Fatalf("create schema %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`); err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
		}
		_ = admin.Close()
	})
	dsn := withSearchPath(base, schema)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open schema connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return dsn, conn
}

// buildSourceSchema creates the production schema in the source schema by running
// the same two constructors the server runs at boot.
func buildSourceSchema(t *testing.T, dsn string) {
	t.Helper()
	database, err := db.New(dsn)
	if err != nil {
		t.Fatalf("create source schema: %v", err)
	}
	manager := moderation.NewManager(database.SQLDB(), database.Dialect())
	manager.Close()
	if err := database.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}
}

// fixturePlayers are the player rows every populated fixture carries. They are
// inserted with explicit ids, some non-contiguous, the way a production database
// looks after deletions: ids are what the application keys on.
var fixturePlayers = []struct {
	ID            int
	Name          string
	Password      string
	RoomVNum      int
	Level         int
	Exp           int
	Inventory     string
	Equipment     string
	CharacterData string
	Description   string
	Title         string
	IsAdmin       bool
	Failures      int
	LockedUntil   any
	CreatedAt     string
	UpdatedAt     string
}{
	{
		ID: 7, Name: "Zax", Password: "$2a$10$fixturepasswordhashfixturepasswordhashfixturepassw",
		RoomVNum: 8004, Level: 34, Exp: 1234567,
		Inventory:     `[{"vnum":8023,"count":1,"state":{"name":"a club with unicode ünïcode 日本語"}},{"vnum":8019,"count":1,"locate":1}]`,
		Equipment:     `{"slots":{"wield":{"vnum":8023}},"unicode":"日本語"}`,
		CharacterData: `{"played":3600,"prefs":{"brief":true,"nested":{"deep":[1,2,3]}},"title":"a 日本語 title"}`,
		Description:   "A weathered immortal.", Title: "the Portwright", IsAdmin: true, Failures: 3,
		LockedUntil: "2026-09-26 20:05:00.5-04",
		CreatedAt:   "2026-03-01 12:30:45.123456+00",
		UpdatedAt:   "2026-03-02 08:00:00+00",
	},
	{
		ID: 42, Name: "roamer", Password: "", RoomVNum: -1, Level: 1,
		Inventory: "[]", Equipment: "{}", CharacterData: "{}",
		LockedUntil: nil, CreatedAt: "2026-09-01 00:00:00+02", UpdatedAt: "2026-09-01 00:00:00+02",
	},
}

func insertFixturePlayers(t *testing.T, conn *sql.DB) {
	t.Helper()
	for i := range fixturePlayers {
		player := &fixturePlayers[i]
		password := any(nil)
		if player.Password != "" {
			password = player.Password
		}
		if _, err := conn.Exec(`
			INSERT INTO players (id, name, password_hash, room_vnum, level, exp, health, max_health, mana, max_mana,
			  move, max_move, strength, class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con,
			  stat_cha, hunger, thirst, drunk, hometown, olc_zone, inventory, equipment, description, title,
			  created_at, updated_at, is_admin, failed_login_attempts, locked_until, character_data)
			VALUES ($1,$2,$3,$4,$5,$6,250,300,100,100,110,120,18,3,1,18,76,12,13,14,15,16,24,24,0,0,0,
			  $7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			player.ID, player.Name, password, player.RoomVNum, player.Level, player.Exp,
			player.Inventory, player.Equipment, player.Description, player.Title,
			player.CreatedAt, player.UpdatedAt, player.IsAdmin, player.Failures, player.LockedUntil,
			player.CharacterData); err != nil {
			t.Fatalf("insert player %s: %v", player.Name, err)
		}
	}
}

func insertFixtureModeration(t *testing.T, conn *sql.DB) {
	t.Helper()
	for _, statement := range moderationFixture {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("insert moderation fixture: %v", err)
		}
	}
}

// moderationFixture is the moderation half of the fixture: two reports (one
// reviewed, one pending, one with NULLs), two admin-log rows (one with a NULL
// interval and a NULL address), two penalties (one permanent, one expired with a
// NULL expiry), and two filters (one regex, one literal).
var moderationFixture = []string{
	`INSERT INTO abuse_reports (id, reporter, target, report_type, description, room_vnum, timestamp, status, reviewed_by, reviewed_at, resolution)
	 VALUES (3, 'Zax', 'roamer', 'spam', 'unicode éè evidence', 8162, '2026-09-20 10:00:00+00', 'pending', NULL, NULL, NULL)`,
	`INSERT INTO abuse_reports (id, reporter, target, report_type, description, room_vnum, timestamp, status, reviewed_by, reviewed_at, resolution)
	 VALUES (9, 'roamer', 'Zax', 'harassment', 'second', 0, '2026-09-21 11:00:00+00', 'resolved', 'Zax', '2026-09-22 12:00:00+00', 'no action')`,
	`INSERT INTO admin_log (id, admin, action, target, reason, duration, timestamp, ip_address)
	 VALUES (1, 'Zax', 'mute', 'roamer', 'spam', '01:30:00', '2026-09-20 10:05:00+00', '127.0.0.1')`,
	`INSERT INTO admin_log (id, admin, action, target, reason, duration, timestamp, ip_address)
	 VALUES (5, 'Zax', 'ban', 'roamer', 'repeat', NULL, '2026-09-20 10:06:00+00', NULL)`,
	`INSERT INTO player_penalties (player_name, penalty_type, issued_at, expires_at, expired_at, status, reason, issued_by)
	 VALUES ('roamer', 'mute', '2026-09-20 10:05:00+00', '2026-09-20 11:05:00+00', NULL, 'active', 'spam', 'Zax')`,
	`INSERT INTO player_penalties (player_name, penalty_type, issued_at, expires_at, expired_at, status, reason, issued_by)
	 VALUES ('roamer', 'ban', '2026-09-19 09:00:00+00', NULL, '2026-09-20 00:00:00+00', 'expired', 'old', 'Zax')`,
	`INSERT INTO word_filters (id, pattern, is_regex, action, created_by, created_at)
	 VALUES (2, 'badword', false, 'censor', 'system', '2026-01-01 00:00:00+00')`,
	`INSERT INTO word_filters (id, pattern, is_regex, action, created_by, created_at)
	 VALUES (11, '(?i)hate.*speech', true, 'block', 'system', '2026-01-02 00:00:00+00')`,
}

// seedSource builds the production schema and loads the full fixture.
func seedSource(t *testing.T, dsn string) {
	t.Helper()
	buildSourceSchema(t, dsn)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer func() { _ = conn.Close() }()
	insertFixturePlayers(t, conn)
	insertFixtureModeration(t, conn)
}

// migrationOptions is the shared option set: verify, then install.
func migrationOptions(source, destination string) Options {
	return Options{Source: source, Destination: destination, Verify: true}
}

func tableRowCount(t *testing.T, conn *sql.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := conn.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func verificationFor(t *testing.T, receipt *Receipt, table string) TableVerification {
	t.Helper()
	if receipt == nil || receipt.Verification == nil {
		if receipt != nil && receipt.Failure != nil {
			t.Fatalf("receipt has no verification; the run failed in phase %s: %s",
				receipt.Failure.Phase, receipt.Failure.Message)
		}
		t.Fatalf("receipt has no verification: %+v", receipt)
	}
	for i := range receipt.Verification.Tables {
		if entry := &receipt.Verification.Tables[i]; entry.Table == table {
			return *entry
		}
	}
	t.Fatalf("verification has no row for %s", table)
	return TableVerification{}
}

func TestMigrateEmptySourceInstallsVerifiedDatabase(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	buildSourceSchema(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("migrate an empty source: %v", err)
	}
	if !receipt.OK || receipt.Destination.Integrity != "ok" {
		t.Fatalf("receipt = %+v", receipt)
	}
	if len(receipt.Copied) != len(Tables) {
		t.Fatalf("copied %d table(s), want %d", len(receipt.Copied), len(Tables))
	}
	for _, entry := range receipt.Copied {
		if entry.Rows != 0 {
			t.Errorf("empty source copied %d row(s) from %s", entry.Rows, entry.Table)
		}
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatalf("destination was not installed: %v", err)
	}
	for _, table := range Tables {
		if got := tableRowCount(t, sourceConn, table); got != 0 {
			t.Errorf("source %s has %d rows after migrating an empty database", table, got)
		}
	}

	// The destination carries the production schema, not a migration-specific one.
	destinationConn, err := sql.Open("sqlite", "file:"+destination+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = destinationConn.Close() }()
	for _, table := range Tables {
		var name string
		if err := destinationConn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Errorf("destination is missing %s: %v", table, err)
		}
	}
	var indexes int
	if err := destinationConn.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name NOT LIKE 'sqlite_autoindex%'`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes == 0 {
		t.Error("destination carries no explicitly named index")
	}
	var integrity string
	if err := destinationConn.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(integrity, "ok") {
		t.Errorf("PRAGMA integrity_check = %q", integrity)
	}
}

func TestMigratePopulatedDatabaseVerifies(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !receipt.OK {
		t.Fatalf("receipt not ok: %s", receipt.Render())
	}
	if receipt.Destination.Integrity != "ok" {
		t.Errorf("integrity = %q", receipt.Destination.Integrity)
	}

	wantRows := map[string]int64{
		"players": 2, "abuse_reports": 2, "admin_log": 2, "player_penalties": 2, "word_filters": 2,
	}
	destinationConn, err := sql.Open("sqlite", "file:"+destination+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = destinationConn.Close() }()
	for table, want := range wantRows {
		if got := tableRowCount(t, sourceConn, table); got != want {
			t.Errorf("source %s has %d rows, want %d", table, got, want)
		}
		if got := tableRowCount(t, destinationConn, table); got != want {
			t.Errorf("destination %s has %d rows, want %d", table, got, want)
		}
		entry := verificationFor(t, receipt, table)
		if !entry.OK || entry.SourceRows != want || entry.DestinationRows != want {
			t.Errorf("%s verification = %+v", table, entry)
		}
		if entry.SourceContentDigest != entry.DestinationContentDigest {
			t.Errorf("%s content digests differ", table)
		}
	}

	// The receipt is machine-readable, and it carries no credentials.
	payload, err := receipt.JSON()
	if err != nil {
		t.Fatal(err)
	}
	// The receipt names columns (that is schema) but must never carry a stored
	// value: no password hash, no player payload, no DSN credential.
	for _, forbidden := range []string{
		fixturePlayers[0].Password,
		fixturePlayers[0].CharacterData,
		"A weathered immortal.",
		"secret",
	} {
		if strings.Contains(string(payload), forbidden) {
			t.Errorf("receipt leaks a stored value (%q): %s", forbidden, payload)
		}
	}
	var decoded Receipt
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("receipt is not valid JSON: %v", err)
	}
	if !decoded.OK || len(decoded.Verification.Tables) != len(Tables) {
		t.Errorf("decoded receipt = %+v", decoded)
	}
}

func TestMigratePreservesEveryPlayerValue(t *testing.T) {
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	conn, err := sql.Open("sqlite", "file:"+destination+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	var (
		id          int64
		name        string
		password    sql.NullString
		roomVNum    int64
		level       int64
		exp         int64
		health      int64
		strength    int64
		statStrAdd  int64
		hunger      int64
		inventory   []byte
		equipment   []byte
		character   []byte
		description string
		title       string
		isAdmin     bool
		failures    int64
		lockedUntil sql.NullTime
		createdAt   time.Time
		updatedAt   time.Time
	)
	err = conn.QueryRow(`SELECT id, name, password_hash, room_vnum, level, exp, health, strength, stat_str_add,
		hunger, inventory, equipment, character_data, description, title, is_admin, failed_login_attempts,
		locked_until, created_at, updated_at FROM players WHERE name = 'Zax'`).Scan(
		&id, &name, &password, &roomVNum, &level, &exp, &health, &strength, &statStrAdd,
		&hunger, &inventory, &equipment, &character, &description, &title, &isAdmin, &failures,
		&lockedUntil, &createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("read migrated player: %v", err)
	}

	if id != 7 || name != "Zax" || level != 34 || exp != 1234567 || health != 250 || strength != 18 || statStrAdd != 76 || hunger != 24 {
		t.Errorf("scalar mismatch: id=%d name=%q level=%d exp=%d health=%d str=%d stradd=%d hunger=%d",
			id, name, level, exp, health, strength, statStrAdd, hunger)
	}
	if roomVNum != 8004 {
		t.Errorf("room_vnum = %d, want 8004", roomVNum)
	}
	if !password.Valid || password.String != fixturePlayers[0].Password {
		t.Errorf("password hash was not preserved byte for byte")
	}
	if description != fixturePlayers[0].Description || title != fixturePlayers[0].Title {
		t.Errorf("description/title = %q/%q", description, title)
	}
	if !isAdmin || failures != 3 {
		t.Errorf("is_admin = %t, failed_login_attempts = %d", isAdmin, failures)
	}
	for _, pair := range []struct {
		name string
		got  []byte
		want string
	}{
		{"inventory", inventory, fixturePlayers[0].Inventory},
		{"equipment", equipment, fixturePlayers[0].Equipment},
		{"character_data", character, fixturePlayers[0].CharacterData},
	} {
		if !JSONReformatOnly(string(pair.got), pair.want) {
			t.Errorf("%s did not survive semantically: %s", pair.name, pair.got)
		}
	}
	if !strings.Contains(string(inventory), "ünïcode") {
		t.Errorf("unicode in inventory was mangled: %s", inventory)
	}

	// Timestamps keep their instant, including the one written with an offset.
	lockedWant := time.Date(2026, 9, 27, 0, 5, 0, 500000000, time.UTC)
	if !lockedUntil.Valid || !lockedUntil.Time.UTC().Equal(lockedWant) {
		t.Errorf("locked_until = %v, want %v", lockedUntil, lockedWant)
	}
	createdWant := time.Date(2026, 3, 1, 12, 30, 45, 123456000, time.UTC)
	if !createdAt.UTC().Equal(createdWant) {
		t.Errorf("created_at = %v, want %v", createdAt, createdWant)
	}

	// The null shape is preserved: a NULL password and the negative room sentinel.
	var nullPassword sql.NullString
	var negativeRoom int64
	var lockedNull sql.NullTime
	if err := conn.QueryRow(`SELECT password_hash, room_vnum, locked_until FROM players WHERE name = 'roamer'`).Scan(
		&nullPassword, &negativeRoom, &lockedNull); err != nil {
		t.Fatalf("read roamer: %v", err)
	}
	if nullPassword.Valid {
		t.Errorf("NULL password_hash became %q", nullPassword.String)
	}
	if negativeRoom != -1 {
		t.Errorf("negative room sentinel = %d, want -1", negativeRoom)
	}
	if lockedNull.Valid {
		t.Errorf("NULL locked_until became %v", lockedNull.Time)
	}
}

func TestMigratePreservesIDsAndAllocatesAboveThem(t *testing.T) {
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// The id watermark is reported per generated-id table and is the proof that
	// the next insert cannot reuse a migrated id.
	wantMax := map[string]int64{"players": 42, "abuse_reports": 9, "admin_log": 5, "word_filters": 11}
	if len(receipt.IDSequences) != len(wantMax) {
		t.Fatalf("id sequences = %+v, want %d entries", receipt.IDSequences, len(wantMax))
	}
	for _, sequence := range receipt.IDSequences {
		if sequence.MaxID != wantMax[sequence.Table] {
			t.Errorf("%s max id = %d, want %d", sequence.Table, sequence.MaxID, wantMax[sequence.Table])
		}
		if !sequence.SequenceAboveMax {
			t.Errorf("%s sequence %d is not above max id %d", sequence.Table, sequence.Sequence, sequence.MaxID)
		}
	}

	// The application's own insert path allocates above every migrated id.
	database, err := db.New("sqlite://" + destination)
	if err != nil {
		t.Fatalf("open migrated database: %v", err)
	}
	defer func() { _ = database.Close() }()
	created := &db.PlayerRecord{Name: "newcomer", Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: []byte("{}")}
	if err := database.CreatePlayer(created); err != nil {
		t.Fatalf("create player after migration: %v", err)
	}
	if created.ID <= 42 {
		t.Fatalf("new player id = %d, want above the highest migrated id 42", created.ID)
	}

	// Reports and filters allocate the same way.
	for table, wantAbove := range map[string]int64{"abuse_reports": 9, "word_filters": 11} {
		result, err := database.Exec(
			`INSERT INTO ` + table + ` (reporter, target, report_type, description) VALUES ('a','b','spam','c')`)
		if table == "word_filters" {
			result, err = database.Exec(
				`INSERT INTO ` + table + ` (pattern, is_regex, action, created_by) VALUES ('x', false, 'warn', 'system')`)
		}
		if err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("last insert id for %s: %v", table, err)
		}
		if id <= wantAbove {
			t.Errorf("%s new id = %d, want above %d", table, id, wantAbove)
		}
	}
}
