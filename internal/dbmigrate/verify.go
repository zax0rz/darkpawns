package dbmigrate

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TableVerification is the independent comparison for one table. Nothing here
// trusts the copy: both sides are re-read from their own connection and digested
// separately. A table is OK only when every check passes; the receipt prints
// which check failed so an operator never has to guess.
type TableVerification struct {
	Table         string   `json:"table"`
	ColumnsCopied []string `json:"columns_copied"`

	SourceRows      int64 `json:"source_rows"`
	DestinationRows int64 `json:"destination_rows"`
	RowsMatch       bool  `json:"rows_match"`

	SourcePrimaryKeyDigest      string `json:"source_primary_key_digest"`
	DestinationPrimaryKeyDigest string `json:"destination_primary_key_digest"`
	PrimaryKeyMatches           bool   `json:"primary_key_matches"`

	SourceContentDigest      string `json:"source_content_digest"`
	DestinationContentDigest string `json:"destination_content_digest"`
	ContentMatches           bool   `json:"content_matches"`

	SourceNullCounts      map[string]int64 `json:"source_null_counts"`
	DestinationNullCounts map[string]int64 `json:"destination_null_counts"`
	NullShapeMatches      bool             `json:"null_shape_matches"`

	// MismatchedColumns names the columns whose content digest differs. Column
	// names are schema, never payload, so this is safe to report and is enough to
	// tell an operator which field to look at without printing a player's data.
	MismatchedColumns []string `json:"mismatched_columns,omitempty"`

	SourceMaxID      *int64 `json:"source_max_id,omitempty"`
	DestinationMaxID *int64 `json:"destination_max_id,omitempty"`
	MaxIDMatches     bool   `json:"max_id_matches"`

	MissingInSource []string `json:"missing_in_source,omitempty"`
	// ExtraInSource is every source column with no destination column.
	ExtraInSource []string `json:"extra_in_source,omitempty"`
	// UnexpectedExtraColumns is the subset of those that no allowance covers.
	// Verification of a table is a claim about all of that table's data, so a
	// source column with nowhere to go breaks the claim: OK is false for it unless
	// a receipt proves an earlier conversion was explicitly told to drop it.
	UnexpectedExtraColumns []string `json:"unexpected_extra_columns,omitempty"`

	// JSONReformattedColumns names JSON columns whose bytes changed while their
	// value did not. That is a dialect re-spacing, not a loss, and it is reported
	// separately from a content mismatch.
	JSONReformattedColumns []string `json:"json_reformatted_columns,omitempty"`

	Notes []string `json:"notes,omitempty"`
	OK    bool     `json:"ok"`
}

// Verification is the whole-run result.
type Verification struct {
	Tables []TableVerification `json:"tables"`
	// UniqueFoldedNamesHolds proves the case-insensitive identity constraint the
	// application relies on (C find_name uses str_cmp): no two rows may share a
	// name ignoring case.
	UniqueFoldedNamesHolds bool     `json:"unique_folded_names_hold"`
	FoldedNameCollisions   int64    `json:"folded_name_collisions"`
	Indexes                []string `json:"indexes"`
	OK                     bool     `json:"ok"`
}

// tableScan is one side of one table, digested.
type tableScan struct {
	rows           int64
	nullCounts     map[string]int64
	rowDigests     []string
	keyDigests     []string
	columnDigests  map[string]string
	rawJSONDigests map[string]string
	maxID          int64
}

// scanTable re-reads one table and digests it. keyDigests are the row identities
// (the primary key), so a missing or extra row is visible even when the table
// digest coincidentally still matches.
func scanTable(ctx context.Context, conn queryer, plan *Table, keyColumns []string) (*tableScan, error) {
	columns := plan.ColumnNames()
	selectList := make([]string, len(columns))
	for i, column := range columns {
		selectList[i] = quoteIdent(column)
	}
	order := make([]string, len(keyColumns))
	for i, column := range keyColumns {
		order[i] = quoteIdent(column)
	}
	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
		strings.Join(selectList, ", "), quoteIdent(plan.Name), strings.Join(order, ", "))

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", plan.Name, err)
	}
	defer func() { _ = rows.Close() }()

	kinds := make(map[string]ValueKind, len(plan.Columns))
	for _, column := range plan.Columns {
		kinds[column.Name] = column.Kind
	}

	scan := &tableScan{
		nullCounts:     make(map[string]int64),
		columnDigests:  make(map[string]string),
		rawJSONDigests: make(map[string]string),
	}
	keySet := make(map[string]bool, len(keyColumns))
	for _, column := range keyColumns {
		keySet[column] = true
	}
	columnValues := make(map[string][]string, len(columns))
	rawJSONValues := make(map[string][]string)

	values := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range values {
		scanTargets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan row of %s: %w", plan.Name, err)
		}
		scan.rows++
		canonical := make([]string, len(columns))
		key := make([]string, 0, len(keyColumns))
		for i, column := range columns {
			if values[i] == nil {
				scan.nullCounts[column]++
			}
			text, err := Canonicalize(kinds[column], values[i])
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", plan.Name, column, err)
			}
			canonical[i] = text
			columnValues[column] = append(columnValues[column], text)
			if keySet[column] {
				key = append(key, column+"="+text)
			}
			if kinds[column] == KindJSON && values[i] != nil {
				raw, err := RawText(values[i])
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", plan.Name, column, err)
				}
				rawJSONValues[column] = append(rawJSONValues[column], raw)
			}
			if plan.GeneratedID != "" && column == plan.GeneratedID {
				n, err := asInt64(values[i])
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", plan.Name, column, err)
				}
				if n > scan.maxID {
					scan.maxID = n
				}
			}
		}
		scan.rowDigests = append(scan.rowDigests, RowDigest(columns, canonical))
		scan.keyDigests = append(scan.keyDigests, strings.Join(key, "\x1e"))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", plan.Name, err)
	}

	for _, column := range columns {
		scan.columnDigests[column] = TableDigest(columnValues[column])
	}
	for column, raws := range rawJSONValues {
		scan.rawJSONDigests[column] = TableDigest(raws)
	}
	return scan, nil
}

// compareScans turns two scans into the verification record.
func compareScans(plan *Table, source, destination *tableScan, allow *Allowances) TableVerification {
	result := TableVerification{
		Table:                       plan.Name,
		ColumnsCopied:               plan.ColumnNames(),
		SourceRows:                  source.rows,
		DestinationRows:             destination.rows,
		RowsMatch:                   source.rows == destination.rows,
		SourcePrimaryKeyDigest:      SetDigest(source.keyDigests),
		DestinationPrimaryKeyDigest: SetDigest(destination.keyDigests),
		SourceContentDigest:         TableDigest(source.rowDigests),
		DestinationContentDigest:    TableDigest(destination.rowDigests),
		SourceNullCounts:            source.nullCounts,
		DestinationNullCounts:       destination.nullCounts,
		MissingInSource:             plan.MissingInSource,
		ExtraInSource:               plan.ExtraInSource,
	}
	result.PrimaryKeyMatches = result.SourcePrimaryKeyDigest == result.DestinationPrimaryKeyDigest
	result.ContentMatches = result.SourceContentDigest == result.DestinationContentDigest
	result.NullShapeMatches = equalNullCounts(source.nullCounts, destination.nullCounts)
	switch {
	case plan.GeneratedID == "":
		// player_penalties keys on (player_name, penalty_type, issued_at), so there
		// is no generated watermark to compare; the primary-key digest above is
		// what proves its identity.
		result.MaxIDMatches = true
	case source.rows == 0 && destination.rows == 0:
		result.MaxIDMatches = true
	default:
		sourceMax, destinationMax := source.maxID, destination.maxID
		result.SourceMaxID, result.DestinationMaxID = &sourceMax, &destinationMax
		result.MaxIDMatches = sourceMax == destinationMax
	}

	for _, column := range plan.ColumnNames() {
		if source.columnDigests[column] != destination.columnDigests[column] {
			result.MismatchedColumns = append(result.MismatchedColumns, column)
			continue
		}
		// Content equal but bytes different: a dialect re-spacing, reported so the
		// operator sees it without it counting as loss.
		if source.rawJSONDigests[column] != destination.rawJSONDigests[column] &&
			source.rawJSONDigests[column] != "" && destination.rawJSONDigests[column] != "" {
			result.JSONReformattedColumns = append(result.JSONReformattedColumns, column)
		}
	}

	for _, column := range plan.ExtraInSource {
		if !allow.ColumnAllowed(plan.Name, column) {
			result.UnexpectedExtraColumns = append(result.UnexpectedExtraColumns, column)
		}
	}
	sort.Strings(result.UnexpectedExtraColumns)

	result.OK = result.RowsMatch && result.PrimaryKeyMatches && result.ContentMatches &&
		result.NullShapeMatches && result.MaxIDMatches && len(result.MismatchedColumns) == 0 &&
		len(result.UnexpectedExtraColumns) == 0
	if len(result.MissingInSource) > 0 {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"%d destination column(s) absent from the source took their schema default: %s",
			len(result.MissingInSource), strings.Join(result.MissingInSource, ", ")))
	}
	if len(result.ExtraInSource) > 0 {
		covered := len(result.ExtraInSource) - len(result.UnexpectedExtraColumns)
		result.Notes = append(result.Notes, fmt.Sprintf(
			"%d source column(s) have no destination column and were not copied: %s (%d covered by an explicit allowance)",
			len(result.ExtraInSource), strings.Join(result.ExtraInSource, ", "), covered))
	}
	if len(result.JSONReformattedColumns) > 0 {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"JSON bytes differ while values match (dialect re-spacing): %s",
			strings.Join(result.JSONReformattedColumns, ", ")))
	}
	return result
}

func equalNullCounts(left, right map[string]int64) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

// quoteIdent quotes an identifier for a statement this package builds from the
// catalog. Every input is a schema name read from the database, never request or
// operator text, and quoting is what keeps a name like "unique" safe.
func quoteIdent(name string) string {
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}
