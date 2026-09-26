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
	if receipt.Destination.Installed || receipt.Destination.DurabilityUncertain {
		t.Errorf("verify-only claims it replaced the destination: %+v", receipt.Destination)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Errorf("verify-only created the destination: %v", statErr)
	}
}

// sourceWithExtraTable seeds a source and then adds a table the migration does
// not own, standing in for the legacy moderation script's side tables.
func sourceWithExtraTable(t *testing.T) (string, *sql.DB) {
	t.Helper()
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	if _, err := sourceConn.Exec(`CREATE TABLE chat_logs (id SERIAL PRIMARY KEY, message TEXT, player_name VARCHAR(32))`); err != nil {
		t.Fatalf("create extra source table: %v", err)
	}
	return sourceDSN, sourceConn
}

// writeReceipt writes a receipt the way the command line tool does, so a test can
// hand it back as the proof a verification is allowed to accept.
func writeReceipt(t *testing.T, path string, receipt *Receipt) {
	t.Helper()
	payload, err := receipt.JSON()
	if err != nil {
		t.Fatalf("render receipt: %v", err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
}

func TestVerifyOnlyRefusesAnUnknownSourceTable(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := sourceConn.Exec(`CREATE TABLE chat_logs (id SERIAL PRIMARY KEY, message TEXT)`); err != nil {
		t.Fatalf("create extra source table: %v", err)
	}

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("verify-only accepted a source table outside the migrated set")
	}
	if !strings.Contains(err.Error(), "chat_logs") {
		t.Errorf("refusal does not name the table: %v", err)
	}
	if !strings.Contains(err.Error(), "--conversion-receipt") {
		t.Errorf("refusal does not say how the decision can be proved: %v", err)
	}
	if receipt.Failure == nil || receipt.Failure.Phase != PhasePreflight {
		t.Fatalf("failure = %+v", receipt)
	}
	// The inventory is populated even though the run stopped, because it is what
	// stopped it.
	if len(receipt.Source.Tables) != len(Tables)+1 {
		t.Errorf("source tables = %v, want the five plus chat_logs", receipt.Source.Tables)
	}
	if len(receipt.Source.UnknownTables) != 1 || receipt.Source.UnknownTables[0] != "chat_logs" {
		t.Errorf("unknown tables = %v", receipt.Source.UnknownTables)
	}
	if receipt.Verification != nil {
		t.Error("verification ran despite the inventory stop condition")
	}
}

func TestVerifyOnlyAcceptsUnknownTablesAConversionReceiptProves(t *testing.T) {
	sourceDSN, _ := sourceWithExtraTable(t)
	dir := t.TempDir()
	destination := filepath.Join(dir, "darkpawns.db")
	receiptPath := filepath.Join(dir, "migration-receipt.json")

	converted, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraTables: true,
	})
	if err != nil {
		t.Fatalf("convert with --drop-extra-tables: %v", err)
	}
	if converted.Allowances == nil || len(converted.Allowances.Tables) != 1 {
		t.Fatalf("conversion receipt allowances = %+v", converted.Allowances)
	}
	writeReceipt(t, receiptPath, converted)

	verified, err := verifyOnlyWithReceipt(t, sourceDSN, destination, receiptPath)
	if err != nil {
		t.Fatalf("verify-only with the conversion's own receipt: %v", err)
	}
	if !verified.OK {
		t.Fatalf("receipt = %s", verified.Render())
	}
	if verified.Allowances == nil || !strings.Contains(verified.Allowances.Proof, receiptPath) {
		t.Errorf("verification receipt does not record the proof it used: %+v", verified.Allowances)
	}
	if len(verified.Allowances.Tables) != 1 || verified.Allowances.Tables[0] != "chat_logs" {
		t.Errorf("allowed tables = %v", verified.Allowances.Tables)
	}
	if len(verified.Source.UnknownTables) != 1 || len(verified.Verification.Tables) != len(Tables) {
		t.Errorf("receipt inventory = %v / %d tables", verified.Source.UnknownTables, len(verified.Verification.Tables))
	}
}

func TestVerifyOnlyRefusesAnUnknownTableTheReceiptDoesNotCover(t *testing.T) {
	sourceDSN, sourceConn := sourceWithExtraTable(t)
	dir := t.TempDir()
	destination := filepath.Join(dir, "darkpawns.db")
	receiptPath := filepath.Join(dir, "migration-receipt.json")

	converted, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraTables: true,
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	writeReceipt(t, receiptPath, converted)

	// A second table with no decision behind it. The receipt proves chat_logs and
	// cannot be stretched to cover anything else.
	if _, err := sourceConn.Exec(`CREATE TABLE player_notes (id SERIAL PRIMARY KEY, note TEXT)`); err != nil {
		t.Fatalf("create second extra table: %v", err)
	}

	receipt, err := verifyOnlyWithReceipt(t, sourceDSN, destination, receiptPath)
	if err == nil {
		t.Fatal("verify-only accepted a table the receipt does not cover")
	}
	if !strings.Contains(err.Error(), "player_notes") {
		t.Errorf("refusal does not name the uncovered table: %v", err)
	}
	if strings.Contains(err.Error(), "chat_logs") {
		t.Errorf("refusal names the covered table as a problem: %v", err)
	}
	if receipt.Failure == nil || receipt.Failure.Phase != PhasePreflight {
		t.Fatalf("failure = %+v", receipt)
	}
}

func TestVerifyOnlyRefusesAnExtraSourceColumn(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The legacy moderation script's column: real data with no destination.
	if _, err := sourceConn.Exec(`ALTER TABLE word_filters ADD COLUMN is_active BOOLEAN DEFAULT true`); err != nil {
		t.Fatalf("add legacy column: %v", err)
	}

	receipt, err := verifyOnly(t, sourceDSN, destination)
	if err == nil {
		t.Fatal("verify-only ignored a source column with no destination column")
	}
	if !strings.Contains(err.Error(), "is_active") {
		t.Errorf("refusal does not name the column: %v", err)
	}
	if !strings.Contains(err.Error(), "--conversion-receipt") {
		t.Errorf("refusal does not say how the decision can be proved: %v", err)
	}
	if receipt.Failure == nil || receipt.Failure.Phase != PhaseSchema {
		t.Fatalf("failure = %+v", receipt)
	}
}

func TestVerifyOnlyAcceptsExtraColumnsAConversionReceiptProves(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	if _, err := sourceConn.Exec(`ALTER TABLE word_filters ADD COLUMN is_active BOOLEAN DEFAULT true`); err != nil {
		t.Fatalf("add legacy column: %v", err)
	}
	dir := t.TempDir()
	destination := filepath.Join(dir, "darkpawns.db")
	receiptPath := filepath.Join(dir, "migration-receipt.json")

	converted, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraColumns: true,
	})
	if err != nil {
		t.Fatalf("convert with --drop-extra-columns: %v", err)
	}
	if converted.Allowances == nil || len(converted.Allowances.Columns["word_filters"]) != 1 {
		t.Fatalf("conversion receipt allowances = %+v", converted.Allowances)
	}
	writeReceipt(t, receiptPath, converted)

	verified, err := verifyOnlyWithReceipt(t, sourceDSN, destination, receiptPath)
	if err != nil {
		t.Fatalf("verify-only with the conversion's own receipt: %v", err)
	}
	entry := verificationFor(t, verified, "word_filters")
	if !entry.OK {
		t.Errorf("word_filters verification = %+v", entry)
	}
	if len(entry.UnexpectedExtraColumns) != 0 {
		t.Errorf("unexpected extra columns = %v", entry.UnexpectedExtraColumns)
	}
	if len(entry.ExtraInSource) != 1 || entry.ExtraInSource[0] != "is_active" {
		t.Errorf("extra_in_source = %v", entry.ExtraInSource)
	}
}

func TestVerifyOnlyRejectsAReceiptThatProvesNothing(t *testing.T) {
	sourceDSN, sourceConn := sourceWithExtraTable(t)
	dir := t.TempDir()
	destination := filepath.Join(dir, "darkpawns.db")
	if _, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraTables: true,
	}); err != nil {
		t.Fatalf("convert: %v", err)
	}

	// A clean conversion elsewhere: a real receipt, for a source with nothing to
	// allow, which therefore proves nothing about this source.
	cleanDSN, _ := newSourceSchema(t)
	seedSource(t, cleanDSN)
	clean := filepath.Join(dir, "clean.db")
	cleanReceipt := filepath.Join(dir, "clean-receipt.json")
	cleanConversion, err := Run(context.Background(), migrationOptions(cleanDSN, clean))
	if err != nil {
		t.Fatalf("clean convert: %v", err)
	}
	writeReceipt(t, cleanReceipt, cleanConversion)

	// A successful verification receipt: verification is not a conversion, and it
	// allowed nothing, so it cannot prove anything about a later run. It is taken
	// from the clean pair, which is the only pair here a verification can pass.
	verifyReceipt := filepath.Join(dir, "verify-receipt.json")
	successfulVerification, err := verifyOnly(t, cleanDSN, clean)
	if err != nil {
		t.Fatalf("clean verify-only: %v", err)
	}
	if successfulVerification.Mode != ModeVerifyOnly {
		t.Fatalf("verify-only receipt mode = %q", successfulVerification.Mode)
	}
	writeReceipt(t, verifyReceipt, successfulVerification)

	// A failed conversion: a run that did not complete allowed nothing.
	failedDSN, _ := sourceWithExtraTable(t)
	failureReceipt := filepath.Join(dir, "failed-receipt.json")
	failed, _ := Run(context.Background(), Options{
		Source: failedDSN, Destination: filepath.Join(dir, "never-installed.db"), Verify: true,
	})
	writeReceipt(t, failureReceipt, failed)

	garbage := filepath.Join(dir, "garbage.json")
	if err := os.WriteFile(garbage, []byte("this is not a receipt\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"not a receipt":              garbage,
		"missing file":               filepath.Join(dir, "absent.json"),
		"a verification receipt":     verifyReceipt,
		"a failed conversion":        failureReceipt,
		"a conversion with no needs": cleanReceipt,
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			receipt, err := verifyOnlyWithReceipt(t, sourceDSN, destination, path)
			if err == nil {
				t.Fatalf("verify-only accepted a proof that proves nothing: %s", receipt.Render())
			}
			if receipt == nil || receipt.Failure == nil || receipt.Failure.Phase != PhaseValidate {
				t.Fatalf("failure = %+v", receipt)
			}
		})
	}
	if _, err := sourceConn.Exec(`SELECT 1`); err != nil {
		t.Fatalf("source unusable after the rejected runs: %v", err)
	}
}

func verifyOnlyWithReceipt(t *testing.T, sourceDSN, destination, receiptPath string) (*Receipt, error) {
	t.Helper()
	return Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, VerifyOnly: true, ConversionReceipt: receiptPath,
	})
}

// TestVerifyOnlyRecordsOnlyWhatItActuallyWaived covers a receipt that proves more
// than the source still needs: it names a table that existed when it was written
// and is gone now. The proof is still recorded, because a reader must be able to
// see which receipt the run accepted, but the run must not report having waived
// anything — nobody should read "tables left behind" out of a run that left
// nothing.
func TestVerifyOnlyRecordsOnlyWhatItActuallyWaived(t *testing.T) {
	sourceDSN, sourceConn := sourceWithExtraTable(t)
	dir := t.TempDir()
	destination := filepath.Join(dir, "darkpawns.db")
	receiptPath := filepath.Join(dir, "migration-receipt.json")

	converted, err := Run(context.Background(), Options{
		Source: sourceDSN, Destination: destination, Verify: true, DropExtraTables: true,
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if converted.Allowances == nil || len(converted.Allowances.Tables) != 1 {
		t.Fatalf("conversion allowances = %+v", converted.Allowances)
	}
	writeReceipt(t, receiptPath, converted)

	// The table the receipt names is no longer there: the decision it records is
	// now moot.
	if _, err := sourceConn.Exec(`DROP TABLE chat_logs`); err != nil {
		t.Fatalf("drop the extra table: %v", err)
	}

	verified, err := verifyOnlyWithReceipt(t, sourceDSN, destination, receiptPath)
	if err != nil {
		t.Fatalf("verify-only: %v", err)
	}
	if !verified.OK {
		t.Fatalf("receipt = %s", verified.Render())
	}
	if verified.Allowances == nil {
		t.Fatal("the accepted proof is not recorded at all")
	}
	if !strings.Contains(verified.Allowances.Proof, receiptPath) {
		t.Errorf("allowance proof = %q", verified.Allowances.Proof)
	}
	if len(verified.Allowances.Tables) != 0 || len(verified.Allowances.Columns) != 0 {
		t.Errorf("a run that waived nothing recorded waivers: %+v", verified.Allowances)
	}
	if strings.Contains(verified.Render(), "left behind") {
		t.Errorf("summary claims something was left behind:\n%s", verified.Render())
	}
}
