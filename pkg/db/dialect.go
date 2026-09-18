package db

import (
	"fmt"
	"regexp"
	"strings"
)

// dialect selects the SQL flavour for a connection. The game store supports
// both; the research corpus (decision_log, combat_log) is PostgreSQL-only and
// is never reached on the SQLite path.
type dialect int

const (
	dialectPostgres dialect = iota
	dialectSQLite
)

// splitDSN sniffs the connection string. postgres:// and postgresql:// select
// the lib/pq driver; anything else, a sqlite:// prefix, a bare file path or
// :memory:, selects modernc.org/sqlite. An empty string is an error so the
// DP_ALLOW_NO_DB fallthrough in cmd/server keeps working: an unconfigured
// operator must get a failure, not an anonymous temporary database.
func splitDSN(connString string) (dialect, string, error) {
	if strings.TrimSpace(connString) == "" {
		return 0, "", fmt.Errorf("database connection string is empty")
	}
	if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		return dialectPostgres, connString, nil
	}
	dsn, _ := strings.CutPrefix(connString, "sqlite://")
	return dialectSQLite, dsn, nil
}

// postgresPlaceholder matches $N bind markers. SQLite's positional ? markers
// consume arguments in order of appearance, so a statement's placeholders
// must appear in argument order and each exactly once; RecordLoginFailure is
// written that way on purpose.
var postgresPlaceholder = regexp.MustCompile(`\$\d+`)

// rebind translates PostgreSQL $N placeholders to SQLite positional markers.
// PostgreSQL text passes through unchanged.
func (d dialect) rebind(query string) string {
	if d == dialectPostgres {
		return query
	}
	return postgresPlaceholder.ReplaceAllString(query, "?")
}

// sqliteDDL rewrites the PostgreSQL-flavoured schema in this package into the
// SQLite syntax verified against modernc.org/sqlite:
//
//   - SERIAL/BIGSERIAL PRIMARY KEY -> INTEGER PRIMARY KEY AUTOINCREMENT.
//     SQLite accepts the unknown SERIAL type silently and the column then
//     never autoincrements, so this must be a real rewrite, not a hope.
//     BIGSERIAL must be listed before SERIAL: the shorter pattern is a
//     substring of the longer one and would otherwise leave a "BI" prefix
//     glued onto the replacement.
//   - DEFAULT NOW() -> DEFAULT CURRENT_TIMESTAMP. SQLite's DEFAULT clause
//     takes a literal or CURRENT_TIMESTAMP, never a bare function call.
//   - TIMESTAMPTZ -> TIMESTAMP. modernc.org/sqlite returns time.Time only for
//     the declared types it recognizes, and TIMESTAMPTZ is not one of them;
//     with the zone-suffixed spelling every timestamp scan comes back as a
//     string. SQLite stores no zone information either way.
//
// CURRENT_TIMESTAMP is valid on both dialects, which is why the DML in this
// package uses it directly instead of routing through this translator.
func sqliteDDL(query string) string {
	return strings.NewReplacer(
		"BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT",
		"SERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT",
		"TIMESTAMPTZ", "TIMESTAMP",
		"DEFAULT NOW()", "DEFAULT CURRENT_TIMESTAMP",
		"DEFAULT now()", "DEFAULT CURRENT_TIMESTAMP",
	).Replace(query)
}

// quoteLiteral renders s as a SQL string literal safe to interpolate into the
// pragma and ALTER statements built by addColumnIfNotExists. Both values are
// package constants (table and column names), never player input.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
