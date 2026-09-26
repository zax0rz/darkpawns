package dbmigrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// migratedPair migrates a populated fixture and returns both sides, for the
// verify-only tests that compare an existing pair.
func migratedPair(t *testing.T) (string, string) {
	t.Helper()
	sourceDSN, destination := migrateFixture(t)
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return sourceDSN, destination
}

// mutate applies one statement to the migrated SQLite file, standing in for
// whatever changed it after the migration.
func mutate(t *testing.T, destination, statement string) {
	t.Helper()
	conn, err := sql.Open("sqlite", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Exec(statement); err != nil {
		t.Fatalf("mutate destination: %v", err)
	}
}

func verifyOnly(t *testing.T, sourceDSN, destination string) (*Receipt, error) {
	t.Helper()
	return Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, VerifyOnly: true,
	})
}

func TestVerifyOnlyAcceptsAMigratedPair(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err != nil {
		t.Fatalf("verify-only on a fresh migration: %v", err)
	}
	if !receipt.OK || receipt.Mode != ModeVerifyOnly {
		t.Fatalf("receipt = %+v", receipt)
	}
	if len(receipt.Copied) != 0 {
		t.Error("verify-only reported copied rows")
	}
	if !receipt.Verification.UniqueFoldedNamesHolds || len(receipt.Verification.Indexes) == 0 {
		t.Errorf("verification = %+v", receipt.Verification)
	}
}

func TestVerifyOnlyDetectsAChangedScalar(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	mutate(t, destination, `UPDATE players SET level = 35 WHERE name = 'Zax'`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("a changed level was not detected")
	}
	entry := verificationFor(t, receipt, "players")
	if entry.ContentMatches {
		t.Error("players content digest matched after a scalar change")
	}
	if strings.Join(entry.MismatchedColumns, ",") != "level" {
		t.Errorf("mismatched columns = %v, want [level]", entry.MismatchedColumns)
	}
	if !strings.Contains(FailureSummary(receipt.Verification), "level") {
		t.Errorf("failure summary does not name the column: %s", FailureSummary(receipt.Verification))
	}
}

func TestVerifyOnlyDetectsAChangedJSONValue(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	mutate(t, destination, `UPDATE players SET character_data = '{"played":7200}' WHERE name = 'Zax'`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("a changed character_data value was not detected")
	}
	entry := verificationFor(t, receipt, "players")
	if strings.Join(entry.MismatchedColumns, ",") != "character_data" {
		t.Errorf("mismatched columns = %v, want [character_data]", entry.MismatchedColumns)
	}
}

func TestVerifyOnlyAcceptsJSONReformatting(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	// Same value, different bytes: key order and whitespace only. A byte
	// comparison would call this a difference; the verifier must not, and it must
	// still say out loud that the bytes moved.
	mutate(t, destination, `UPDATE players SET character_data =
		'{ "played" : 3600 , "prefs" : { "nested" : { "deep" : [ 1 , 2 , 3 ] } , "brief" : true } , "title" : "a 日本語 title" }'
		WHERE name = 'Zax'`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err != nil {
		t.Fatalf("reformatting was reported as a difference: %v", err)
	}
	entry := verificationFor(t, receipt, "players")
	if len(entry.JSONReformattedColumns) != 1 || entry.JSONReformattedColumns[0] != "character_data" {
		t.Errorf("json_reformatted_columns = %v", entry.JSONReformattedColumns)
	}
	if len(entry.Notes) == 0 {
		t.Error("receipt does not note the reformatting")
	}
}

func TestVerifyOnlyDetectsAChangedTimestampInstant(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	// The same wall clock one hour off: only an instant comparison catches this.
	mutate(t, destination, `UPDATE players SET locked_until = '2026-09-26 21:05:00.5-04' WHERE name = 'Zax'`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("a shifted lockout instant was not detected")
	}
	entry := verificationFor(t, receipt, "players")
	if strings.Join(entry.MismatchedColumns, ",") != "locked_until" {
		t.Errorf("mismatched columns = %v, want [locked_until]", entry.MismatchedColumns)
	}
}

func TestVerifyOnlyDetectsAMissingRow(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	mutate(t, destination, `DELETE FROM word_filters WHERE id = 11`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("a deleted row was not detected")
	}
	entry := verificationFor(t, receipt, "word_filters")
	if entry.RowsMatch || entry.DestinationRows != 1 {
		t.Errorf("row counts = %d/%d", entry.SourceRows, entry.DestinationRows)
	}
	if !strings.Contains(FailureSummary(receipt.Verification), "word_filters") {
		t.Errorf("failure summary = %s", FailureSummary(receipt.Verification))
	}
}

func TestVerifyOnlyDetectsAnExtraRow(t *testing.T) {
	sourceDSN, destination := migratedPair(t)
	mutate(t, destination, `INSERT INTO abuse_reports (reporter, target, report_type, description, room_vnum, timestamp, status)
		VALUES ('nobody', 'nobody', 'spam', 'extra', 0, '2026-09-25 00:00:00+00', 'pending')`)

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("an extra row was not detected")
	}
	entry := verificationFor(t, receipt, "abuse_reports")
	if entry.RowsMatch || entry.DestinationRows != 3 {
		t.Errorf("row counts = %d/%d", entry.SourceRows, entry.DestinationRows)
	}
	if entry.PrimaryKeyMatches {
		t.Error("primary-key digests matched with an extra row present")
	}
}

func TestVerifyOnlyRefusesAMissingDestination(t *testing.T) {
	sourceDSN, _ := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "absent.db")
	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("verifying an absent destination succeeded")
	}
	if receipt == nil || receipt.Failure == nil || receipt.Failure.Phase != PhasePreflight {
		t.Fatalf("failure = %+v", receipt)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("verify-only created the destination: %v", statErr)
	}
}
