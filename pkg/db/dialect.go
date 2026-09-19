package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Dialect selects the SQL flavour for a connection. The game store supports
// both; the research corpus (decision_log, combat_log) is PostgreSQL-only and
// is never reached on the SQLite path.
//
// Exported because pkg/moderation writes its own tables through the same
// connection: one translator for both stores, or the two drift and only one of
// them inherits a fix.
type Dialect int

const (
	DialectPostgres Dialect = iota
	DialectSQLite
)

// SplitDSN sniffs the connection string. postgres:// and postgresql:// select
// the lib/pq driver; anything else, a sqlite:// prefix, a bare file path or
// :memory:, selects modernc.org/sqlite. An empty string is an error so the
// DP_ALLOW_NO_DB fallthrough in cmd/server keeps working: an unconfigured
// operator must get a failure, not an anonymous temporary database.
func SplitDSN(connString string) (Dialect, string, error) {
	if strings.TrimSpace(connString) == "" {
		return 0, "", fmt.Errorf("database connection string is empty")
	}
	if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		return DialectPostgres, connString, nil
	}
	dsn, _ := strings.CutPrefix(connString, "sqlite://")
	return DialectSQLite, dsn, nil
}

// postgresPlaceholder matches $N bind markers. SQLite's positional ? markers
// consume arguments in order of appearance, so a statement's placeholders
// must appear in argument order and each exactly once; RecordLoginFailure is
// written that way on purpose.
var postgresPlaceholder = regexp.MustCompile(`\$\d+`)

// Rebind translates PostgreSQL $N placeholders to SQLite positional markers.
// PostgreSQL text passes through unchanged.
func (d Dialect) Rebind(query string) string {
	if d == DialectPostgres {
		return query
	}
	return postgresPlaceholder.ReplaceAllString(query, "?")
}

// expandRepeatedPlaceholders is what makes the store's argument contract the
// same on both dialects. Callers bind one argument per placeholder *occurrence*,
// in order of appearance — that is exactly how SQLite's positional markers
// work, so `VALUES ($6, $6)` consumes two arguments there. PostgreSQL numbers
// distinct parameters instead, so the identical statement takes one argument
// and lib/pq rejects the call outright ("got 7 parameters but the statement
// requires 6"), which is how a statement that ran on SQLite silently failed
// once a PostgreSQL deployment touched it.
//
// When the argument count matches the number of occurrences, every repeated
// marker is rewritten to a fresh $N so both dialects consume the arguments the
// caller supplied. Statements without repeats, and statements bound
// PostgreSQL-style (one argument per distinct parameter), are returned
// unchanged; the argument-count check is what keeps the classic `$1 ... $1`
// reuse working. The rewrite is a semantic no-op for SQLite: occurrence i is
// still bound to argument i.
func expandRepeatedPlaceholders(query string, args []interface{}) (string, []interface{}) {
	matches := postgresPlaceholder.FindAllStringIndex(query, -1)
	if len(matches) == 0 || len(matches) != len(args) {
		return query, args
	}

	numbers := make([]int, len(matches))
	maxParam := 0
	for i, m := range matches {
		n, err := strconv.Atoi(query[m[0]+1 : m[1]])
		if err != nil || n < 1 || n > len(args) {
			return query, args
		}
		numbers[i] = n
		if n > maxParam {
			maxParam = n
		}
	}
	if maxParam == len(matches) {
		return query, args // already one distinct marker per argument
	}

	var b strings.Builder
	b.Grow(len(query))
	expanded := make([]interface{}, 0, len(matches))
	last := 0
	for i, m := range matches {
		b.WriteString(query[last:m[0]])
		b.WriteString("$")
		b.WriteString(strconv.Itoa(i + 1))
		last = m[1]
		expanded = append(expanded, args[numbers[i]-1])
	}
	b.WriteString(query[last:])
	return b.String(), expanded
}

// DDL rewrites the PostgreSQL-flavoured schema in this package into the
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
func (d Dialect) DDL(query string) string {
	if d == DialectPostgres {
		return query
	}
	return strings.NewReplacer(
		"BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT",
		"SERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT",
		"TIMESTAMPTZ", "TIMESTAMP",
		"DEFAULT NOW()", "DEFAULT CURRENT_TIMESTAMP",
		"DEFAULT now()", "DEFAULT CURRENT_TIMESTAMP",
	).Replace(query)
}

// quoteLiteral renders s as a SQL string literal safe to interpolate into the
// pragma and ALTER statements built by AddColumnIfNotExists. Both values are
// package constants (table and column names), never player input.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// AddColumnIfNotExists applies one migration column idempotently. PostgreSQL
// has ADD COLUMN IF NOT EXISTS natively; SQLite has ADD COLUMN but not the
// IF NOT EXISTS guard, so the column is looked up in pragma_table_info first.
// Verified idempotent across repeat runs on both dialects, and exported so a
// second package with tables of its own (pkg/moderation) reuses this guard
// instead of growing a copy that drifts.
//
// The constructed statement goes through DDL, the same translator CREATE TABLE
// uses, so a migration column may be spelled in the PostgreSQL dialect the way
// the rest of the schema is. It is identity on PostgreSQL; on SQLite it is what
// folds TIMESTAMPTZ to TIMESTAMP and DEFAULT NOW() to DEFAULT CURRENT_TIMESTAMP.
// Skipping it is not harmless: a column declared TIMESTAMPTZ reaches SQLite as
// an unknown type name, which takes no affinity, so modernc hands every value
// back as a string and the first Scan into time.Time fails at runtime.
func AddColumnIfNotExists(conn *sql.DB, d Dialect, table, columnDef string) error {
	column := strings.Fields(columnDef)[0]
	if d == DialectSQLite {
		var n int
		if err := conn.QueryRow(d.Rebind(
			`SELECT COUNT(*) FROM pragma_table_info(` + quoteLiteral(table) + `) WHERE name = ` + quoteLiteral(column),
		)).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		_, err := conn.Exec(d.DDL(d.Rebind(`ALTER TABLE ` + table + ` ADD COLUMN ` + columnDef)))
		return err
	}
	_, err := conn.Exec(d.DDL(d.Rebind(`ALTER TABLE ` + table + ` ADD COLUMN IF NOT EXISTS ` + columnDef)))
	return err
}
