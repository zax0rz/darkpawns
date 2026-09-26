package dbmigrate

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// migrateFixture is the common setup for the safety tests: a populated source in
// its own schema and a destination path that does not exist yet.
func migrateFixture(t *testing.T) (string, string) {
	t.Helper()
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)
	return sourceDSN, filepath.Join(t.TempDir(), "darkpawns.db")
}

func temporarySiblings(t *testing.T, destination string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(destination), "."+filepath.Base(destination)+".migrating-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestMigrateRefusesNonEmptyDestination(t *testing.T) {
	sourceDSN, destination := migrateFixture(t)
	first, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("first migration: %v", err)
	}
	before, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err == nil {
		t.Fatal("a second migration over a non-empty destination was accepted")
	}
	if second == nil || second.Failure == nil || second.Failure.Phase != PhasePreflight {
		t.Fatalf("failure = %+v", second)
	}
	if !strings.Contains(err.Error(), "--replace") {
		t.Errorf("refusal does not tell the operator about --replace: %v", err)
	}
	after, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the refused run modified the existing destination")
	}
	if !first.OK {
		t.Error("the first migration did not report OK")
	}
	if leftovers := temporarySiblings(t, destination); len(leftovers) != 0 {
		t.Errorf("refused run left temporary files: %v", leftovers)
	}
}

func TestMigrateReplaceModeInstallsTheNewDatabase(t *testing.T) {
	sourceDSN, destination := migrateFixture(t)
	// A destination that is clearly not the migrated database: a valid SQLite
	// file with a marker table.
	marker, err := sql.Open("sqlite", destination)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := marker.Exec(`CREATE TABLE old_marker (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := marker.Close(); err != nil {
		t.Fatal(err)
	}

	receipt, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, Replace: true,
	})
	if err != nil {
		t.Fatalf("replace migration: %v", err)
	}
	if !receipt.OK {
		t.Fatalf("replace receipt = %s", receipt.Render())
	}

	conn, err := sql.Open("sqlite", "file:"+destination+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var tables int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='old_marker'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Error("the replaced destination still holds the old table")
	}
	if got := tableRowCount(t, conn, "players"); got != 2 {
		t.Errorf("replaced destination has %d players, want 2", got)
	}
}

func TestFailedMigrationLeavesNoDestination(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	// A case-folding name collision is what the runtime refuses to arbitrate, and
	// the schema prevents one from being created now: the folded-name index exists
	// on PostgreSQL too. A source that predates that index can still hold one, so
	// the fixture drops the index to reproduce exactly that install. The copy must
	// then fail on the destination's own constraint rather than pick an account.
	if _, err := sourceConn.Exec(`DROP INDEX players_name_folded_key`); err != nil {
		t.Fatalf("drop folded-name index: %v", err)
	}
	if _, err := sourceConn.Exec(`INSERT INTO players (id, name, room_vnum, inventory, equipment, character_data)
		VALUES (99, 'ZAX', 8004, '[]', '{}', '{}')`); err != nil {
		t.Fatalf("insert collision fixture: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err == nil {
		t.Fatal("a folded-name collision was migrated without error")
	}
	if receipt == nil || receipt.OK {
		t.Fatalf("receipt = %+v", receipt)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("failed migration left a destination: %v", statErr)
	}
	if leftovers := temporarySiblings(t, destination); len(leftovers) != 0 {
		t.Errorf("failed migration left temporary files: %v", leftovers)
	}
	// The source still holds every row it had, including the colliding one.
	if got := tableRowCount(t, sourceConn, "players"); got != 3 {
		t.Errorf("source players = %d, want 3", got)
	}
}

func TestMigrateRefusesUnknownSourceTable(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	if _, err := sourceConn.Exec(`CREATE TABLE chat_logs (id SERIAL PRIMARY KEY, message TEXT)`); err != nil {
		t.Fatalf("create extra table: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err == nil {
		t.Fatal("a source table outside the migrated set was accepted")
	}
	if !strings.Contains(err.Error(), "chat_logs") {
		t.Errorf("refusal does not name the table: %v", err)
	}
	if receipt.Source.UnknownTables == nil || receipt.Source.UnknownTables[0] != "chat_logs" {
		t.Errorf("receipt unknown tables = %v", receipt.Source.UnknownTables)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("refused run created a destination: %v", statErr)
	}

	allowed, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraTables: true,
	})
	if err != nil {
		t.Fatalf("explicit --drop-extra-tables run failed: %v", err)
	}
	if !allowed.OK || len(allowed.Source.UnknownTables) != 1 {
		t.Fatalf("allowed receipt = %+v", allowed)
	}
}

func TestMigrateRefusesExtraSourceColumn(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	// The legacy moderation script defines columns the runtime never writes.
	if _, err := sourceConn.Exec(`ALTER TABLE word_filters ADD COLUMN is_active BOOLEAN DEFAULT true`); err != nil {
		t.Fatalf("add legacy column: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	refused, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err == nil {
		t.Fatal("a source column with no destination column was dropped silently")
	}
	if refused == nil || refused.Failure.Phase != PhaseSchema {
		t.Fatalf("failure = %+v", refused)
	}
	if !strings.Contains(err.Error(), "is_active") {
		t.Errorf("refusal does not name the column: %v", err)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("refused run created a destination: %v", statErr)
	}

	allowed, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraColumns: true,
	})
	if err != nil {
		t.Fatalf("explicit --drop-extra-columns run failed: %v", err)
	}
	if !allowed.OK {
		t.Fatalf("allowed receipt = %s", allowed.Render())
	}
	entry := verificationFor(t, allowed, "word_filters")
	if len(entry.ExtraInSource) != 1 || entry.ExtraInSource[0] != "is_active" {
		t.Errorf("receipt extra_in_source = %v", entry.ExtraInSource)
	}
	if len(entry.Notes) == 0 {
		t.Error("receipt does not note the dropped column")
	}
}

func TestMigrateRefusesSourceMissingADestinationTable(t *testing.T) {
	sourceDSN, _ := newSourceSchema(t)
	// Only the game store: the moderation tables were never created.
	database, err := db.New(sourceDSN)
	if err != nil {
		t.Fatalf("create players-only source: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "darkpawns.db")

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err == nil {
		t.Fatal("a source missing a table was accepted")
	}
	if !strings.Contains(err.Error(), "abuse_reports") {
		t.Errorf("refusal does not name the missing table: %v", err)
	}
	if receipt == nil || receipt.Failure == nil || receipt.Failure.Phase != PhaseSchema {
		t.Fatalf("failure = %+v", receipt)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("failed run created a destination: %v", statErr)
	}
}

func TestMigrationDoesNotMutateTheSource(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)

	snapshot := func() map[string]string {
		values := make(map[string]string)
		for _, table := range Tables {
			var count int64
			if err := sourceConn.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			values[table+".rows"] = strconv.FormatInt(count, 10)
		}
		var sequence int64
		if err := sourceConn.QueryRow(`SELECT last_value FROM players_id_seq`).Scan(&sequence); err != nil {
			t.Fatalf("read players sequence: %v", err)
		}
		values["players_id_seq"] = strconv.FormatInt(sequence, 10)
		var digest string
		if err := sourceConn.QueryRow(
			`SELECT md5(string_agg(id::text || name || coalesce(password_hash,'') || room_vnum::text || level::text || inventory::text, '|' ORDER BY id)) FROM players`,
		).Scan(&digest); err != nil {
			t.Fatalf("digest players: %v", err)
		}
		values["players.digest"] = digest
		return values
	}
	before := snapshot()

	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	after := snapshot()
	for key, want := range before {
		if after[key] != want {
			t.Errorf("source %s changed during migration: %q -> %q", key, want, after[key])
		}
	}
}
