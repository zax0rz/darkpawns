package dbmigrate

import "testing"

// identicalScans builds two scans that agree on everything, so a test can change
// one property at a time and see exactly what it does to the verdict.
func identicalScans() (*tableScan, *tableScan) {
	build := func() *tableScan {
		return &tableScan{
			rows:          2,
			nullCounts:    map[string]int64{"nickname": 1},
			rowDigests:    []string{"row-a", "row-b"},
			keyDigests:    []string{"id=1", "id=2"},
			columnDigests: map[string]string{"id": "a", "nickname": "b"},
			maxID:         2,
		}
	}
	return build(), build()
}

func TestCompareScansMarksIdenticalTablesOK(t *testing.T) {
	source, destination := identicalScans()
	plan := &Table{Name: "players", CopyColumns: []string{"id", "nickname"}, GeneratedID: "id", PrimaryKey: []string{"id"}}
	result := compareScans(plan, source, destination, nil)
	if !result.OK {
		t.Fatalf("identical scans are not OK: %+v", result)
	}
	if len(result.UnexpectedExtraColumns) != 0 {
		t.Errorf("unexpected extra columns = %v", result.UnexpectedExtraColumns)
	}
}

// TestCompareScansRefusesUnallowedExtraColumns is the property the follow-up
// asks for: a table whose rows are byte-identical is still not OK while the
// source holds a column with nowhere to go, because OK is a claim about all of
// the table's data. Asking a conversion's receipt for the decision is the only
// thing that can restore the claim.
func TestCompareScansRefusesUnallowedExtraColumns(t *testing.T) {
	plan := &Table{
		Name:          "word_filters",
		CopyColumns:   []string{"id"},
		ExtraInSource: []string{"is_active"},
		GeneratedID:   "id",
		PrimaryKey:    []string{"id"},
	}
	source, destination := identicalScans()

	unallowed := compareScans(plan, source, destination, nil)
	if unallowed.OK {
		t.Error("a table with an unallowed source column was reported OK")
	}
	if len(unallowed.UnexpectedExtraColumns) != 1 || unallowed.UnexpectedExtraColumns[0] != "is_active" {
		t.Errorf("unexpected extra columns = %v", unallowed.UnexpectedExtraColumns)
	}
	if len(unallowed.ExtraInSource) != 1 || !unallowed.RowsMatch || !unallowed.ContentMatches {
		t.Errorf("the data comparison should still be recorded: %+v", unallowed)
	}
	if summary := FailureSummary(&Verification{Tables: []TableVerification{unallowed}}); summary == "the two databases do not agree" {
		t.Errorf("failure summary does not name the column problem: %s", summary)
	}

	allowed := &Allowances{tables: map[string]bool{}, columns: map[string]map[string]bool{
		"word_filters": {"is_active": true},
	}}
	permitted := compareScans(plan, source, destination, allowed)
	if !permitted.OK {
		t.Errorf("an allowed source column still failed the table: %+v", permitted)
	}
	if len(permitted.UnexpectedExtraColumns) != 0 {
		t.Errorf("unexpected extra columns = %v", permitted.UnexpectedExtraColumns)
	}
	if len(permitted.Notes) == 0 {
		t.Error("the sanctioned omission is not noted")
	}

	// An allowance for a different column, or for the right column of a different
	// table, is not an allowance for this one.
	wrongColumn := &Allowances{tables: map[string]bool{}, columns: map[string]map[string]bool{
		"word_filters": {"severity": true},
	}}
	if compareScans(plan, source, destination, wrongColumn).OK {
		t.Error("an allowance for another column passed this table")
	}
	wrongTable := &Allowances{tables: map[string]bool{}, columns: map[string]map[string]bool{
		"players": {"is_active": true},
	}}
	if compareScans(plan, source, destination, wrongTable).OK {
		t.Error("an allowance for another table passed this table")
	}
}

func TestAllowancesAreNeverInheritedFromTheFlagsAlone(t *testing.T) {
	plans := []*Table{{
		Name:          "word_filters",
		ExtraInSource: []string{"is_active"},
	}}
	// A conversion takes them from its own explicit flags...
	granted := conversionAllowances(plans, []string{"chat_logs"}, Options{DropExtraTables: true, DropExtraColumns: true})
	if !granted.TableAllowed("chat_logs") || !granted.ColumnAllowed("word_filters", "is_active") {
		t.Fatalf("explicit flags did not grant the allowance: %+v", granted.record())
	}
	if granted.Empty() {
		t.Error("a granted allowance set reports itself empty")
	}

	// ...and a run without them grants nothing at all.
	ungranted := conversionAllowances(plans, []string{"chat_logs"}, Options{})
	if !ungranted.Empty() || ungranted.TableAllowed("chat_logs") || ungranted.ColumnAllowed("word_filters", "is_active") {
		t.Fatalf("a run with no flags granted an allowance: %+v", ungranted.record())
	}

	// Nothing is allowed by a nil set, which is what a caller without a proof
	// passes.
	var none *Allowances
	if !none.Empty() || none.TableAllowed("players") || none.ColumnAllowed("players", "name") {
		t.Error("a nil allowance set allowed something")
	}
}

func TestUnexpectedColumnsAndTablesOnlyReportTheUngranted(t *testing.T) {
	plans := []*Table{
		{Name: "players", ExtraInSource: []string{"nickname"}},
		{Name: "word_filters", ExtraInSource: []string{"is_active", "severity"}},
	}
	allow := &Allowances{tables: map[string]bool{"chat_logs": true}, columns: map[string]map[string]bool{
		"word_filters": {"is_active": true},
	}}

	columns := UnexpectedColumns(plans, allow)
	if len(columns) != 2 {
		t.Fatalf("unexpected columns = %v", columns)
	}
	if len(columns["players"]) != 1 || columns["players"][0] != "nickname" {
		t.Errorf("players = %v", columns["players"])
	}
	if len(columns["word_filters"]) != 1 || columns["word_filters"][0] != "severity" {
		t.Errorf("word_filters = %v", columns["word_filters"])
	}
	description := describeUnexpectedColumns(columns)
	if description != "players: nickname; word_filters: severity" {
		t.Errorf("description = %q, want a deterministic, sorted phrase", description)
	}

	tables := UnexpectedTables([]string{"chat_logs", "player_notes"}, allow)
	if len(tables) != 1 || tables[0] != "player_notes" {
		t.Errorf("unexpected tables = %v", tables)
	}
	if got := UnexpectedTables([]string{"chat_logs"}, nil); len(got) != 1 || got[0] != "chat_logs" {
		t.Errorf("without a proof every unknown table is unexpected: %v", got)
	}
}
