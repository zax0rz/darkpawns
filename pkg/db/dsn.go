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

// ErrUnsupportedScheme reports a setting that names some other database, cache or
// service over a URI. It exists because a URI-shaped string is not a filename: a
// setting like mysql://host/db once silently became a file called
// "mysql:/host/db" in the current directory, so a wrong DSN produced a working,
// empty server instead of a refusal.
var ErrUnsupportedScheme = errors.New(
	"unsupported database setting: the supported spellings are a filesystem path, sqlite://<path> and sqlite:<path>")

// SQLitePath resolves the runtime database setting to the SQLite file it names.
//
// Accepted spellings, and nothing else:
//
//	/var/lib/darkpawns/darkpawns.db   a filesystem path
//	sqlite:///var/lib/darkpawns/…      the same path with a sqlite:// prefix
//	sqlite:/var/lib/darkpawns/…        the same path with a sqlite: prefix
//
// A setting that carries a URI scheme instead is refused, whatever the scheme is:
//
//   - postgres:// or postgresql://: ErrPostgresDSN (see above). This is the case
//     that matters most: a unit file left over from the PostgreSQL era must fail
//     loudly rather than be ignored, because ignoring it would boot the server
//     against a fresh empty SQLite file and look healthy.
//   - any other scheme (mysql://, redis://, https://, sqlite+something://, or a
//     bare "scheme:value"): ErrUnsupportedScheme. A URI is not a filename, and
//     treating one as a filename created a file named after the URI in the
//     current directory, which is the same silent-empty-server hazard as an
//     ignored DSN.
//   - empty: there is no default here. The caller decides the path, so a missing
//     setting is a configuration error rather than a silent in-memory database.
//   - :memory: and file::memory:: refused. An in-memory database loses every
//     character at restart, which is not a runtime configuration.
//
// A single-letter scheme is read as a Windows drive letter (C:\data\darkpawns.db
// is a path), because refusing every valid Windows path to catch a hypothetical
// one-letter DSN is the worse trade.
func SQLitePath(setting string) (string, error) {
	trimmed := strings.TrimSpace(setting)
	if trimmed == "" {
		return "", errors.New("no database configured: set the SQLite path (-db, DP_SQLITE_PATH)")
	}

	scheme, rest := uriScheme(trimmed)
	path := trimmed
	switch strings.ToLower(scheme) {
	case "postgres", "postgresql":
		return "", fmt.Errorf("%w (got %q)", ErrPostgresDSN, RedactDSN(trimmed))
	case "sqlite":
		// Accept sqlite:<path> and sqlite://<path> alike: the authority slashes of
		// a URL are optional here, and a path that begins with them is not.
		path = strings.TrimPrefix(rest, "//")
	case "":
		// No scheme at all: a plain filesystem path.
	default:
		return "", fmt.Errorf("%w (got %q, a %q URI)", ErrUnsupportedScheme, RedactDSN(trimmed), scheme)
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

// uriScheme splits an RFC 3986 scheme off a setting, returning ("", setting) when
// there is none. A single letter followed by a colon is a Windows drive letter
// rather than a scheme, and a colon after a path separator can never be part of
// one.
func uriScheme(setting string) (scheme, rest string) {
	colon := strings.IndexByte(setting, ':')
	if colon <= 0 {
		return "", setting
	}
	candidate := setting[:colon]
	for i, r := range candidate {
		valid := r == '+' || r == '-' || r == '.'
		if i == 0 {
			valid = valid || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		} else {
			valid = valid || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		}
		if !valid {
			return "", setting
		}
	}
	if len(candidate) == 1 {
		return "", setting
	}
	return candidate, setting[colon+1:]
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
