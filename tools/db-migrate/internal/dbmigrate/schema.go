// Package dbmigrate converts one PostgreSQL Dark Pawns database into one SQLite
// database, and proves the conversion.
//
// Dark Pawns stores everything durable in one game-store database. This package
// is the operator half of the move to SQLite as the sole long-term backend: it
// copies five tables, verifies the copy independently of "the inserts returned no
// error", and installs the result atomically. It is not part of the server binary
// and adds no runtime dependency to it.
//
// The destination schema is never hand-written here. It comes from pkg/db.New on
// the destination DSN, which is the same schema initialisation the SQLite runtime
// uses on a fresh install, so a migrated database and a fresh one cannot drift.
package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// queryer is the read seam shared by *sql.DB and *sql.Tx. The whole run reads
// the source through one read-only repeatable-read transaction, so the catalog
// it reconciles against and the rows it copies and verifies all come from the
// same snapshot of the source.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Tables in copy order. players is copied first because it is the table the
// application reads at boot; the moderation tables follow. The order is fixed so
// a receipt is comparable between runs.
var Tables = []string{"players", "abuse_reports", "admin_log", "player_penalties", "word_filters"}

// ValueKind is the canonical comparison family of a column, agreed between the
// PostgreSQL source type and the SQLite destination type. Two dialects spell the
// same semantics differently (boolean vs BOOLEAN, timestamp with time zone vs
// TIMESTAMP); the kind is what the verifier compares by, and a disagreement is a
// schema difference, not a formatting difference.
type ValueKind string

const (
	KindText      ValueKind = "text"
	KindInteger   ValueKind = "integer"
	KindBoolean   ValueKind = "boolean"
	KindTimestamp ValueKind = "timestamp"
	KindJSON      ValueKind = "json"
	KindInterval  ValueKind = "interval"
)

// postgresKind maps information_schema.columns.data_type to a canonical kind.
func postgresKind(dataType string) (ValueKind, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "boolean":
		return KindBoolean, nil
	case "smallint", "integer", "bigint":
		return KindInteger, nil
	case "timestamp with time zone", "timestamp without time zone", "date":
		return KindTimestamp, nil
	case "json", "jsonb":
		return KindJSON, nil
	case "interval":
		return KindInterval, nil
	case "text", "character varying", "character", "name":
		return KindText, nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL column type %q", dataType)
	}
}

// sqliteKind maps a declared SQLite column type (pragma_table_info.type) to a
// canonical kind. SQLite applies type affinity, so the declared name is the only
// statement of intent: a column declared TIMESTAMP is what modernc.org/sqlite
// hands back as time.Time, and one declared JSON comes back as bytes.
func sqliteKind(declaredType string) (ValueKind, error) {
	upper := strings.ToUpper(strings.TrimSpace(declaredType))
	switch {
	case upper == "":
		return KindText, nil
	// INTERVAL is checked before the integer prefixes: "INT" is a prefix of
	// "INTERVAL", and matching the shorter one first would read an interval
	// column as an integer and make the reconciliation refuse a table that is
	// in fact identical on both sides.
	case strings.HasPrefix(upper, "INTERVAL"):
		return KindInterval, nil
	case strings.HasPrefix(upper, "INTEGER"), upper == "INT", strings.HasPrefix(upper, "INT("), strings.HasPrefix(upper, "INT "):
		return KindInteger, nil
	case strings.HasPrefix(upper, "BOOLEAN"), strings.HasPrefix(upper, "BOOL"):
		return KindBoolean, nil
	case strings.HasPrefix(upper, "TIMESTAMP"), strings.HasPrefix(upper, "DATETIME"), upper == "DATE":
		return KindTimestamp, nil
	case strings.HasPrefix(upper, "JSON"):
		return KindJSON, nil
	default:
		// VARCHAR(n), TEXT, CHARACTER VARYING and anything else behaves as text.
		return KindText, nil
	}
}

// Column is one column of the destination schema, with the kind agreed against
// the source.
type Column struct {
	Name string
	Kind ValueKind
	// DestinationType is the declared SQLite type; SourceType is the PostgreSQL
	// type. Both are reported so an operator can see what was reconciled.
	DestinationType string
	SourceType      string
	// NotNull reports the destination's NOT NULL flag. The null-shape check
	// compares null counts per column, and this is the expected flag.
	NotNull bool
}

// Table is the reconciled copy plan for one table.
type Table struct {
	Name string
	// Columns copied, in destination schema order. Columns the source lacks are
	// omitted so the destination's own default applies; they are listed in
	// MissingInSource.
	Columns []Column
	// CopyColumns is the copied column names in order, for statement building.
	CopyColumns []string
	// MissingInSource lists destination columns the source does not have. The
	// destination default applies; this is a schema evolution, not data loss.
	MissingInSource []string
	// ExtraInSource lists source columns the destination does not have. Copying
	// cannot represent them: they are dropped only when the operator asked for
	// that, and the receipt records it.
	ExtraInSource []string
	// GeneratedID is the autoincrement column, or "" when the table has none.
	GeneratedID string
	// PrimaryKey is the row-identity column list (a single integer column, or the
	// composite natural key).
	PrimaryKey []string
}

// ColumnNames returns a copy of the copied column names.
func (t *Table) ColumnNames() []string { return append([]string(nil), t.CopyColumns...) }

// sqliteTableColumns reads the destination catalog: every column the production
// schema initialisation created, in declared order.
func sqliteTableColumns(ctx context.Context, conn queryer, table string) ([]Column, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT name, type, "notnull" FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, fmt.Errorf("read sqlite columns for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var columns []Column
	for rows.Next() {
		var name, declaredType string
		var notNull int
		if err := rows.Scan(&name, &declaredType, &notNull); err != nil {
			return nil, fmt.Errorf("scan sqlite column for %s: %w", table, err)
		}
		kind, err := sqliteKind(declaredType)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", table, name, err)
		}
		columns = append(columns, Column{
			Name:            name,
			Kind:            kind,
			DestinationType: declaredType,
			NotNull:         notNull == 1,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read sqlite columns for %s: %w", table, err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("destination has no table %q", table)
	}
	return columns, nil
}

// postgresTableColumns reads the source catalog for one table.
func postgresTableColumns(ctx context.Context, conn queryer, table string) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT column_name, data_type FROM information_schema.columns
		  WHERE table_schema = current_schema() AND table_name = $1`, table)
	if err != nil {
		return nil, fmt.Errorf("read postgres columns for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	columns := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, fmt.Errorf("scan postgres column for %s: %w", table, err)
		}
		columns[name] = dataType
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read postgres columns for %s: %w", table, err)
	}
	return columns, nil
}

// generatedIDColumn returns the table's AUTOINCREMENT column, or "". It is read
// from the destination's own DDL rather than assumed: AUTOINCREMENT is what makes
// SQLite keep a sequence above the highest id ever inserted, and the
// post-migration id proof depends on that being true of the real schema.
func generatedIDColumn(ctx context.Context, conn queryer, table string) (string, error) {
	var ddl sql.NullString
	if err := conn.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&ddl); err != nil {
		return "", fmt.Errorf("read sqlite ddl for %s: %w", table, err)
	}
	if !strings.Contains(strings.ToUpper(ddl.String), "AUTOINCREMENT") {
		return "", nil
	}
	var column string
	if err := conn.QueryRowContext(ctx,
		`SELECT name FROM pragma_table_info(?) WHERE pk = 1 ORDER BY pk LIMIT 1`, table).Scan(&column); err != nil {
		return "", fmt.Errorf("find primary key of %s: %w", table, err)
	}
	return column, nil
}

// primaryKeyColumns returns the table's key columns in key order.
func primaryKeyColumns(ctx context.Context, conn queryer, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT name FROM pragma_table_info(?) WHERE pk > 0 ORDER BY pk`, table)
	if err != nil {
		return nil, fmt.Errorf("read primary key of %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan primary key of %s: %w", table, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read primary key of %s: %w", table, err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("destination table %q has no primary key", table)
	}
	return columns, nil
}
