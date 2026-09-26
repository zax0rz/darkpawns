package db

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// DriverName is the only SQL driver this package opens.
//
// Dark Pawns stores its runtime state in an embedded SQLite database: a fresh
// installation needs no external service. PostgreSQL is not a runtime option,
// and the conversion bridge that reads a PostgreSQL database to produce a SQLite
// one lives in its own module (tools/db-migrate), which is the only thing in the
// repository that links a PostgreSQL driver.
const DriverName = "sqlite"

// ErrPostgresDSN reports a PostgreSQL connection string handed to a runtime that
// no longer has a PostgreSQL driver. It is a distinct error so a caller can tell
// "misconfigured for the old architecture" from "path is wrong".
var ErrPostgresDSN = errors.New(
	"PostgreSQL is no longer a runtime database: this server stores its data in SQLite. " +
		"Point it at the SQLite file instead (tools/db-migrate converts an existing PostgreSQL database)")

// SQLitePath resolves the runtime database setting to the SQLite file it names.
//
// Accepted spellings: a bare filesystem path, or the same path with a sqlite://
// prefix. Everything else is refused, and refused with a message that says why:
//
//   - empty: there is no default here. The caller decides the path, so a missing
//     setting is a configuration error rather than a silent in-memory database.
//   - postgres:// or postgresql://: ErrPostgresDSN (see above). This is the case
//     that matters most: a unit file left over from the PostgreSQL era must fail
//     loudly rather than be ignored, because ignoring it would boot the server
//     against a fresh empty SQLite file and look healthy.
//   - :memory: and file::memory:: refused. An in-memory database loses every
//     character at restart; it is not a runtime configuration, and silently
//     accepting it is the same hazard as accepting an ignored DSN.
func SQLitePath(setting string) (string, error) {
	trimmed := strings.TrimSpace(setting)
	if trimmed == "" {
		return "", errors.New("no database configured: set the SQLite path (-db, DP_SQLITE_PATH)")
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return "", fmt.Errorf("%w (got %q)", ErrPostgresDSN, RedactDSN(trimmed))
	}
	path, hadScheme := strings.CutPrefix(trimmed, "sqlite://")
	if !hadScheme {
		path = strings.TrimPrefix(trimmed, "sqlite:")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("database setting %q names no file", trimmed)
	}
	if path == ":memory:" || strings.HasPrefix(path, "file::memory:") || strings.Contains(path, "mode=memory") {
		return "", fmt.Errorf(
			"database setting %q asks for an in-memory database, which would lose every character at restart", trimmed)
	}
	return filepath.Clean(path), nil
}

// RedactDSN keeps a misconfiguration message readable without echoing a password
// back into a log. A SQLite path has nothing to redact, so it is returned as is.
func RedactDSN(setting string) string {
	trimmed := strings.TrimSpace(setting)
	if !strings.Contains(trimmed, "://") {
		return trimmed
	}
	scheme, rest, _ := strings.Cut(trimmed, "://")
	rest, _, _ = strings.Cut(rest, "?")
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	rest = strings.TrimPrefix(rest, ":")
	if userinfoEnd := strings.Index(rest, ":"); userinfoEnd >= 0 && !strings.Contains(rest[:userinfoEnd], "/") {
		rest = rest[userinfoEnd+1:]
	}
	return scheme + "://" + rest
}

// quoteLiteral renders s as a SQL string literal safe to interpolate into the
// pragma statements built by AddColumnIfNotExists. The value is a package
// constant (a table or column name the schema declares), never player input.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// AddColumnIfNotExists applies one migration column idempotently.
//
// SQLite has ADD COLUMN but no IF NOT EXISTS guard, so the column is looked up in
// pragma_table_info first. Verified idempotent across repeat runs. Exported
// because pkg/moderation has migration columns of its own through the same
// connection, and one implementation means one place to fix.
func AddColumnIfNotExists(conn *sql.DB, table, columnDef string) error {
	column := strings.Fields(columnDef)[0]
	var n int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(` + quoteLiteral(table) + `) WHERE name = ` + quoteLiteral(column),
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := conn.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + columnDef)
	return err
}
