package dbmigrate

import (
	"context"
	"fmt"
	"strings"
)

// Verify compares every planned table between a source and a destination and
// returns the independent result. Both sides are read through their own
// connection and digested separately; nothing here consults the copy log.
//
// allow carries the proofs a caller has for source tables and columns the
// destination cannot hold. It is the only thing that can keep a table's OK true
// while part of its source has nowhere to go, and it never affects a value that
// does have a destination column.
func Verify(ctx context.Context, source, destination queryer, plans []*Table, allow *Allowances) (*Verification, error) {
	result := &Verification{}
	for _, plan := range plans {
		sourceScan, err := scanTable(ctx, source, plan, plan.PrimaryKey)
		if err != nil {
			return nil, fmt.Errorf("scan source %s: %w", plan.Name, err)
		}
		destinationScan, err := scanTable(ctx, destination, plan, plan.PrimaryKey)
		if err != nil {
			return nil, fmt.Errorf("scan destination %s: %w", plan.Name, err)
		}
		result.Tables = append(result.Tables, compareScans(plan, sourceScan, destinationScan, allow))
	}

	collisions, err := foldedNameCollisions(ctx, destination)
	if err != nil {
		return nil, err
	}
	result.FoldedNameCollisions = collisions
	result.UniqueFoldedNamesHolds = collisions == 0

	indexes, err := DestinationIndexes(ctx, destination)
	if err != nil {
		return nil, err
	}
	result.Indexes = indexes

	result.OK = result.UniqueFoldedNamesHolds && len(result.Tables) > 0
	for i := range result.Tables {
		if !result.Tables[i].OK {
			result.OK = false
		}
	}
	return result, nil
}

// foldedNameCollisions counts player names that collide when folded to lower
// case. C's find_name compares with str_cmp, so two rows differing only in case
// are two logins for one character; the destination must not hold any.
func foldedNameCollisions(ctx context.Context, conn queryer) (int64, error) {
	var collisions int64
	err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (SELECT lower(name) AS folded FROM players GROUP BY folded HAVING COUNT(*) > 1)`,
	).Scan(&collisions)
	if err != nil {
		return 0, fmt.Errorf("count folded-name collisions: %w", err)
	}
	return collisions, nil
}

// FailureSummary names what disagreed, so a failed verification is actionable
// without opening the JSON receipt. It reports table and column names only:
// schema, never payload.
func FailureSummary(verification *Verification) string {
	if verification == nil {
		return "no verification was run"
	}
	// Reported as a list rather than one problem per table: a table can be both
	// missing data and carrying a column with no destination, and an operator
	// needs to see the kind of problem that a receipt has to prove away.
	var problems []string
	for i := range verification.Tables {
		table := &verification.Tables[i]
		if !table.RowsMatch {
			problems = append(problems, fmt.Sprintf("%s: %d source row(s) vs %d destination row(s)",
				table.Table, table.SourceRows, table.DestinationRows))
		}
		if !table.PrimaryKeyMatches {
			problems = append(problems, fmt.Sprintf("%s: primary-key sets differ", table.Table))
		}
		if len(table.MismatchedColumns) > 0 {
			problems = append(problems, fmt.Sprintf("%s: column(s) %s differ",
				table.Table, strings.Join(table.MismatchedColumns, ", ")))
		}
		if !table.NullShapeMatches {
			problems = append(problems, fmt.Sprintf("%s: null shape differs", table.Table))
		}
		if !table.MaxIDMatches {
			problems = append(problems, fmt.Sprintf("%s: highest migrated id differs", table.Table))
		}
		if len(table.UnexpectedExtraColumns) > 0 {
			problems = append(problems, fmt.Sprintf(
				"%s: source column(s) %s have no destination column and no proof they were dropped deliberately",
				table.Table, strings.Join(table.UnexpectedExtraColumns, ", ")))
		}
	}
	if !verification.UniqueFoldedNamesHolds {
		problems = append(problems, fmt.Sprintf("players: %d case-folding name collision(s)",
			verification.FoldedNameCollisions))
	}
	if len(problems) == 0 {
		return "the two databases do not agree"
	}
	return strings.Join(problems, "; ")
}
