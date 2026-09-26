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
	if receipt.Destination.Installed || receipt.Destination.DurabilityUncertain {
		t.Errorf("a failed run claims the destination was replaced: %+v", receipt.Destination)
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
	if receipt.Failure == nil || receipt.Failure.Phase != PhasePreflight {
		t.Fatalf("failure = %+v", receipt.Failure)
	}
	if receipt.Destination.Installed {
		t.Error("a refused run claims the destination was replaced")
	}
	if receipt.Allowances != nil {
		t.Errorf("a run that granted nothing recorded an allowance: %+v", receipt.Allowances)
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
	if allowed.Allowances == nil || len(allowed.Allowances.Tables) != 1 || allowed.Allowances.Tables[0] != "chat_logs" {
		t.Errorf("conversion allowances = %+v", allowed.Allowances)
	}
	if !strings.Contains(allowed.Allowances.Proof, "command-line flags") {
		t.Errorf("allowance proof = %q", allowed.Allowances.Proof)
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
	if allowed.Allowances == nil || len(allowed.Allowances.Columns["word_filters"]) != 1 {
		t.Errorf("conversion allowances = %+v", allowed.Allowances)
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

// stateDirectory creates a directory for the destination and returns both paths,
// with the mode restored before the test's own cleanup removes it.
func stateDirectory(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("create state directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	return dir, filepath.Join(dir, "darkpawns.db")
}

// TestInstalledButDurabilityUncertainIsItsOwnState is the second half of the
// install contract: a rename that lands followed by a directory sync that fails
// is not an install that never happened. The destination holds the verified
// database, and the receipt has to say so in its own words.
//
// The failure is produced the way it really happens -- a destination directory
// that can be written and traversed but not read, so the rename succeeds and the
// directory cannot be opened to be synced.
func TestInstalledButDurabilityUncertainIsItsOwnState(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory mode cannot deny the read the sync needs")
	}
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)
	dir, destination := stateDirectory(t)
	if err := os.Chmod(dir, 0o300); err != nil {
		t.Fatalf("make the destination directory unreadable: %v", err)
	}

	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	// Restore the mode before asserting, because reading the directory back is
	// exactly what the run could not do.
	if chmodErr := os.Chmod(dir, 0o700); chmodErr != nil {
		t.Fatalf("restore the destination directory mode: %v", chmodErr)
	}
	if err == nil {
		t.Fatal("the run reported success although the destination directory could not be synced")
	}
	if receipt == nil {
		t.Fatal("a failure produced no receipt")
	}
	if receipt.Failure == nil || receipt.Failure.Phase != PhaseInstallDurability {
		t.Fatalf("failure = %+v", receipt.Failure)
	}
	if !receipt.Destination.Installed {
		t.Error("receipt does not record that the destination was replaced")
	}
	if !receipt.Destination.DurabilityUncertain {
		t.Error("receipt does not record that durability is uncertain")
	}
	if receipt.OK {
		t.Error("a durability failure reported OK")
	}
	if receipt.Destination.Bytes == 0 {
		t.Error("receipt does not record the size of the installed database")
	}
	if !strings.Contains(err.Error(), "now holds the verified database") {
		t.Errorf("error does not say the destination was replaced: %v", err)
	}
	rendered := receipt.Render()
	if !strings.Contains(rendered, "installed") || !strings.Contains(rendered, "NOT synced") {
		t.Errorf("summary does not distinguish the two install outcomes:\n%s", rendered)
	}

	// The file that is in place is the verified database, not a partial one.
	conn, err := sql.Open("sqlite", "file:"+destination+"?mode=ro")
	if err != nil {
		t.Fatalf("open the installed database: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if got := tableRowCount(t, conn, "players"); got != 2 {
		t.Errorf("installed players = %d, want the two migrated characters", got)
	}
	var integrity string
	if err := conn.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(integrity, "ok") {
		t.Errorf("installed database integrity = %q", integrity)
	}
	if leftovers := temporarySiblings(t, destination); len(leftovers) != 0 {
		t.Errorf("the run left temporary files behind: %v", leftovers)
	}

	// A re-run refuses until the operator says replace, which is the point of
	// reporting the state rather than the error alone.
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err == nil {
		t.Error("a re-run over the installed database was accepted without --replace")
	}
}

// TestInstallRenameFailureReportsTheDestinationAsUnchanged is the first half of
// the contract: a failure before the rename leaves the destination exactly as it
// was, and the receipt says that too.
func TestInstallRenameFailureReportsTheDestinationAsUnchanged(t *testing.T) {
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)

	// A directory in the destination's place: an operator typo that reaches the
	// rename and fails there. It is non-empty, so it also needs --replace.
	parent := t.TempDir()
	destination := filepath.Join(parent, "darkpawns.db")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "marker")
	if err := os.WriteFile(marker, []byte("untouched\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	receipt, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, Replace: true,
	})
	if err == nil {
		t.Fatal("renaming a database over a non-empty directory was accepted")
	}
	if receipt.Failure == nil || receipt.Failure.Phase != PhaseInstall {
		t.Fatalf("failure = %+v", receipt.Failure)
	}
	if receipt.Destination.Installed || receipt.Destination.DurabilityUncertain {
		t.Errorf("a failed rename claimed an install: %+v", receipt.Destination)
	}
	if !strings.Contains(receipt.Render(), "unchanged") {
		t.Errorf("summary does not say the destination was left alone:\n%s", receipt.Render())
	}
	info, err := os.Stat(destination)
	if err != nil || !info.IsDir() {
		t.Fatalf("the destination directory was disturbed: %v (%v)", info, err)
	}
	payload, err := os.ReadFile(marker)
	if err != nil || string(payload) != "untouched\n" {
		t.Errorf("the destination directory's contents changed: %q (%v)", payload, err)
	}
	if leftovers := temporarySiblings(t, destination); len(leftovers) != 0 {
		t.Errorf("the failed run left temporary files behind: %v", leftovers)
	}
}

func TestSyncDirectoryRefusesWhatItCannotSync(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory mode cannot deny the read the sync needs")
	}
	if err := syncDirectory(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("syncing a directory that does not exist was reported as success")
	}

	dir, _ := stateDirectory(t)
	if err := os.Chmod(dir, 0o300); err != nil {
		t.Fatal(err)
	}
	err := syncDirectory(dir)
	if chmodErr := os.Chmod(dir, 0o700); chmodErr != nil {
		t.Fatal(chmodErr)
	}
	if err == nil {
		t.Error("syncing an unreadable directory was reported as success")
	}
}

func TestSuccessfulInstallRecordsItsState(t *testing.T) {
	sourceDSN, destination := migrateFixture(t)
	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !receipt.Destination.Installed {
		t.Error("a successful run does not record that it installed the destination")
	}
	if receipt.Destination.DurabilityUncertain {
		t.Error("a successful run reports uncertain durability")
	}
	if receipt.Allowances != nil {
		t.Errorf("a run that left nothing behind recorded allowances: %+v", receipt.Allowances)
	}
	if !strings.Contains(receipt.Render(), "state:     installed") {
		t.Errorf("summary does not say the destination was installed:\n%s", receipt.Render())
	}
}
