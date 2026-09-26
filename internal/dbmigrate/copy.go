package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// copyTable copies one table's rows, in primary-key order, into the destination
// transaction. Values are bound as the driver handed them over: the canonical
// form is the verifier's comparison form, never the storage form, so nothing is
// rewritten on the way across.
func copyTable(ctx context.Context, source queryer, tx *sql.Tx, plan *Table, batchSize int) (int64, error) {
	columns := plan.ColumnNames()
	if len(columns) == 0 {
		return 0, fmt.Errorf("table %s has no copyable columns", plan.Name)
	}
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quoteIdent(column)
	}
	order := make([]string, len(plan.PrimaryKey))
	for i, column := range plan.PrimaryKey {
		order[i] = quoteIdent(column)
	}

	rows, err := source.QueryContext(ctx, fmt.Sprintf(
		"SELECT %s FROM %s ORDER BY %s",
		strings.Join(quoted, ", "), quoteIdent(plan.Name), strings.Join(order, ", ")))
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", plan.Name, err)
	}
	defer func() { _ = rows.Close() }()

	insertPrefix := fmt.Sprintf("INSERT INTO %s (%s) VALUES ",
		quoteIdent(plan.Name), strings.Join(quoted, ", "))
	rowPlaceholder := "(" + strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",") + ")"

	values := make([]any, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}

	var copied int64
	pending := 0
	args := make([]any, 0, len(columns)*batchSize)
	flush := func() error {
		if pending == 0 {
			return nil
		}
		statement := insertPrefix + strings.TrimSuffix(strings.Repeat(rowPlaceholder+",", pending), ",")
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return fmt.Errorf("insert %d row(s) into %s: %w", pending, plan.Name, err)
		}
		args = args[:0]
		pending = 0
		return nil
	}

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return copied, fmt.Errorf("scan %s: %w", plan.Name, err)
		}
		args = append(args, values...)
		pending++
		copied++
		if pending >= batchSize {
			if err := flush(); err != nil {
				return copied, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return copied, fmt.Errorf("read %s: %w", plan.Name, err)
	}
	if err := flush(); err != nil {
		return copied, err
	}
	return copied, nil
}

// syncSequences makes each AUTOINCREMENT table's stored sequence at least the
// highest id copied. Inserting a row with an explicit rowid already moves the
// sequence when that rowid is larger, so on a healthy copy this is a no-op; it
// exists so the guarantee is explicit and checked rather than inferred, and it
// reports the sequence it left behind for the receipt.
func syncSequences(ctx context.Context, tx *sql.Tx, plans []*Table) (map[string]int64, error) {
	sequences := make(map[string]int64, len(plans))
	for _, plan := range plans {
		if plan.GeneratedID == "" {
			continue
		}
		var maxID sql.NullInt64
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(
			"SELECT MAX(%s) FROM %s", quoteIdent(plan.GeneratedID), quoteIdent(plan.Name))).Scan(&maxID); err != nil {
			return nil, fmt.Errorf("read max %s.%s: %w", plan.Name, plan.GeneratedID, err)
		}
		if !maxID.Valid {
			sequences[plan.Name] = 0
			continue
		}
		var seq sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT seq FROM sqlite_sequence WHERE name = ?`, plan.Name).Scan(&seq)
		switch {
		case err == sql.ErrNoRows:
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO sqlite_sequence (name, seq) VALUES (?, ?)`, plan.Name, maxID.Int64); err != nil {
				return nil, fmt.Errorf("seed sqlite_sequence for %s: %w", plan.Name, err)
			}
		case err != nil:
			return nil, fmt.Errorf("read sqlite_sequence for %s: %w", plan.Name, err)
		case seq.Int64 < maxID.Int64:
			if _, err := tx.ExecContext(ctx,
				`UPDATE sqlite_sequence SET seq = ? WHERE name = ?`, maxID.Int64, plan.Name); err != nil {
				return nil, fmt.Errorf("raise sqlite_sequence for %s: %w", plan.Name, err)
			}
		}
		sequences[plan.Name] = maxID.Int64
	}
	return sequences, nil
}

// integrityCheck runs SQLite's own consistency check. It is the one check that
// can catch a corrupt page the SQL layer would otherwise paper over.
func integrityCheck(ctx context.Context, conn queryer) (string, error) {
	var result string
	if err := conn.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return "", fmt.Errorf("integrity_check: %w", err)
	}
	if !strings.EqualFold(result, "ok") {
		return result, fmt.Errorf("sqlite integrity_check reported %q", result)
	}
	rows, err := conn.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return result, fmt.Errorf("foreign_key_check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return result, fmt.Errorf("sqlite foreign_key_check reported violations")
	}
	return result, rows.Err()
}

// finalizeDatabase makes the destination a single self-contained file. WAL mode
// is right for a running server and wrong for an artifact that is about to be
// renamed into place: the newest commits live in the -wal sidecar, so renaming
// only the .db would install a database missing its tail. Checkpointing and
// switching the journal mode folds everything back into the one file. The runtime
// turns WAL on again when it opens the file.
func finalizeDatabase(ctx context.Context, conn queryer) error {
	// Both pragmas return a row, and the connection allows exactly one open
	// statement at a time, so each result set is drained and closed here.
	checkpoint, err := conn.QueryContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	if err != nil {
		return fmt.Errorf("wal_checkpoint: %w", err)
	}
	if err := checkpoint.Close(); err != nil {
		return fmt.Errorf("close wal_checkpoint: %w", err)
	}
	mode, err := conn.QueryContext(ctx, `PRAGMA journal_mode = DELETE`)
	if err != nil {
		return fmt.Errorf("switch journal mode: %w", err)
	}
	return mode.Close()
}

// installFile moves the verified database into place. Both paths are operator
// supplied; the temporary file is a sibling of the destination so the rename is
// within one filesystem and therefore atomic.
func installFile(temporary, destination string) error {
	file, err := os.OpenFile(filepath.Clean(temporary), os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open temporary database: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary database: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary database: %w", err)
	}
	if err := os.Chmod(filepath.Clean(temporary), 0o600); err != nil {
		return fmt.Errorf("set destination permissions: %w", err)
	}
	if err := os.Rename(filepath.Clean(temporary), filepath.Clean(destination)); err != nil {
		return fmt.Errorf("install destination: %w", err)
	}
	// A directory entry rename is only durable once the directory is synced; a
	// crash after the rename but before the sync can lose it.
	dir, err := os.Open(filepath.Dir(filepath.Clean(destination)))
	if err != nil {
		return fmt.Errorf("open destination directory: %w", err)
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	if syncErr != nil {
		return fmt.Errorf("sync destination directory: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close destination directory: %w", closeErr)
	}
	return nil
}

// removeDatabase deletes a SQLite database and its sidecars, so a failed run
// leaves nothing that could be mistaken for a result.
func removeDatabase(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}
