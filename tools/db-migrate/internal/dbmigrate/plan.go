package dbmigrate

import (
	"context"
	"fmt"
	"sort"
)

// Reconcile builds the copy plan for one table by reading both catalogs. It is
// the whole of this package's schema knowledge: the destination decides which
// columns exist and in what order, the source decides what can fill them, and
// anything that does not line up is named rather than silently reshaped.
func Reconcile(ctx context.Context, source, destination queryer, table string) (*Table, error) {
	destColumns, err := sqliteTableColumns(ctx, destination, table)
	if err != nil {
		return nil, err
	}
	sourceColumns, err := postgresTableColumns(ctx, source, table)
	if err != nil {
		return nil, err
	}
	if len(sourceColumns) == 0 {
		// A table the runtime initialises but the source does not have means the
		// source predates that table: PostgreSQL has never had it created. Say so
		// here, in the schema phase, rather than letting the copy fail on a SQL
		// error that names a statement instead of the problem.
		return nil, fmt.Errorf(
			"source has no table %q; the production schema initialises it, so boot the current binary once against PostgreSQL (or restore the table) before migrating",
			table)
	}
	id, err := generatedIDColumn(ctx, destination, table)
	if err != nil {
		return nil, err
	}
	key, err := primaryKeyColumns(ctx, destination, table)
	if err != nil {
		return nil, err
	}

	plan := &Table{Name: table, GeneratedID: id, PrimaryKey: key}
	for _, column := range destColumns {
		sourceType, ok := sourceColumns[column.Name]
		if !ok {
			plan.MissingInSource = append(plan.MissingInSource, column.Name)
			continue
		}
		kind, err := postgresKind(sourceType)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", table, column.Name, err)
		}
		if kind != column.Kind {
			return nil, fmt.Errorf(
				"%s.%s: PostgreSQL type %q and SQLite type %q are not the same kind (%s vs %s)",
				table, column.Name, sourceType, column.DestinationType, kind, column.Kind)
		}
		column.SourceType = sourceType
		plan.Columns = append(plan.Columns, column)
		plan.CopyColumns = append(plan.CopyColumns, column.Name)
	}
	sort.Strings(plan.MissingInSource)

	for name := range sourceColumns {
		found := false
		for _, column := range destColumns {
			if column.Name == name {
				found = true
				break
			}
		}
		if !found {
			plan.ExtraInSource = append(plan.ExtraInSource, name)
		}
	}
	sort.Strings(plan.ExtraInSource)
	return plan, nil
}

// SourceTables lists the base tables in the source schema, sorted.
func SourceTables(ctx context.Context, conn queryer) ([]string, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables
		  WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'`)
	if err != nil {
		return nil, fmt.Errorf("list postgres tables: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan postgres table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list postgres tables: %w", err)
	}
	sort.Strings(tables)
	return tables, nil
}

// UnknownTables returns the source tables outside the five migrated ones. A table
// this tool does not know about is a stop condition, not something to leave
// behind: it may be production data with no SQLite home yet, and the operator has
// to decide.
func UnknownTables(tables []string) []string {
	known := make(map[string]bool, len(Tables))
	for _, table := range Tables {
		known[table] = true
	}
	var unknown []string
	for _, table := range tables {
		if !known[table] {
			unknown = append(unknown, table)
		}
	}
	return unknown
}

// DestinationIndexes lists the destination's explicitly named indexes, the same
// set the backend tests assert against.
func DestinationIndexes(ctx context.Context, conn queryer) ([]string, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'index' AND name NOT LIKE 'sqlite_autoindex%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list sqlite indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan sqlite index name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sqlite indexes: %w", err)
	}
	return names, nil
}
