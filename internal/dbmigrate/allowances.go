package dbmigrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Allowances are the source tables and columns a run is permitted to leave
// unverified: a source table outside the migrated five, and a source column with
// no destination column. Both are data the destination cannot hold, so both are
// refused by default, and both need an explicit decision before a run may
// proceed past them.
//
// The decision is recorded differently on each side of the contract:
//
//   - A conversion takes its allowances from its own command line
//     (--drop-extra-tables / --drop-extra-columns) and writes them into its
//     receipt. There is no other way for a conversion to earn one.
//   - A verification takes them from that receipt and nowhere else. The generic
//     flags are refused outright in verify-only, so a verify-only run can never
//     be made to ignore something by an argument that belongs to conversion.
//     Without a receipt it accepts nothing.
//
// Allowances never silence a difference in data that does have a destination
// column: they only record that a table or column was knowingly left behind at
// conversion time.
type Allowances struct {
	tables  map[string]bool
	columns map[string]map[string]bool
	proof   string
}

// TableAllowed reports whether the source table may be left behind.
func (a *Allowances) TableAllowed(table string) bool {
	if a == nil {
		return false
	}
	return a.tables[table]
}

// ColumnAllowed reports whether the source column may be left behind.
func (a *Allowances) ColumnAllowed(table, column string) bool {
	if a == nil {
		return false
	}
	return a.columns[table][column]
}

// Empty reports whether this allowance set permits nothing.
func (a *Allowances) Empty() bool {
	if a == nil {
		return true
	}
	if len(a.tables) > 0 {
		return false
	}
	for _, columns := range a.columns {
		if len(columns) > 0 {
			return false
		}
	}
	return true
}

// record renders the allowance set for the receipt: sorted, so two receipts for
// the same decision are byte-comparable.
func (a *Allowances) record() *AllowanceRecord {
	if a.Empty() {
		return nil
	}
	record := &AllowanceRecord{Proof: a.proof}
	record.Tables = a.tableList()
	record.Columns = a.columnList()
	return record
}

func (a *Allowances) tableList() []string {
	tables := make([]string, 0, len(a.tables))
	for table := range a.tables {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	return tables
}

func (a *Allowances) columnList() map[string][]string {
	if len(a.columns) == 0 {
		return nil
	}
	columns := make(map[string][]string, len(a.columns))
	for table, names := range a.columns {
		if len(names) == 0 {
			continue
		}
		sorted := make([]string, 0, len(names))
		for name := range names {
			sorted = append(sorted, name)
		}
		sort.Strings(sorted)
		columns[table] = sorted
	}
	if len(columns) == 0 {
		return nil
	}
	return columns
}

// exercisedRecord renders what a run actually waived: the proof it accepted and
// the allowances it needed. A receipt may prove more than the source needs (it
// names the tables that existed when it was written), and a receipt that waived
// nothing must not read as a decision about tables that are no longer there. The
// proof is recorded either way, so a reader can always see which receipt a run
// accepted.
func exercisedRecord(allow *Allowances, unknownTables []string, plans []*Table) *AllowanceRecord {
	if allow.Empty() {
		return nil
	}
	record := &AllowanceRecord{Proof: allow.proof}
	for _, table := range unknownTables {
		if allow.TableAllowed(table) {
			record.Tables = append(record.Tables, table)
		}
	}
	sort.Strings(record.Tables)
	for _, plan := range plans {
		var columns []string
		for _, column := range plan.ExtraInSource {
			if allow.ColumnAllowed(plan.Name, column) {
				columns = append(columns, column)
			}
		}
		if len(columns) == 0 {
			continue
		}
		sort.Strings(columns)
		if record.Columns == nil {
			record.Columns = make(map[string][]string)
		}
		record.Columns[plan.Name] = columns
	}
	return record
}

// conversionAllowances is what a converting run may leave behind: exactly what the
// operator waved through on this command line. Called after the reconciliation so
// the column half is the real set, and never consulted for a table or column the
// flags do not name.
func conversionAllowances(plans []*Table, unknownTables []string, options Options) *Allowances {
	allowances := &Allowances{
		tables:  make(map[string]bool),
		columns: make(map[string]map[string]bool),
		proof:   "command-line flags on the converting run (--drop-extra-tables / --drop-extra-columns)",
	}
	if options.DropExtraTables {
		for _, table := range unknownTables {
			allowances.tables[table] = true
		}
	}
	if options.DropExtraColumns {
		for _, plan := range plans {
			if len(plan.ExtraInSource) == 0 {
				continue
			}
			allowances.columns[plan.Name] = make(map[string]bool, len(plan.ExtraInSource))
			for _, column := range plan.ExtraInSource {
				allowances.columns[plan.Name][column] = true
			}
		}
	}
	return allowances
}

// loadConversionReceipt reads a receipt written by an earlier run.
func loadConversionReceipt(path string) (*Receipt, error) {
	clean := filepath.Clean(path)
	payload, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read conversion receipt %s: %w", clean, err)
	}
	var receipt Receipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return nil, fmt.Errorf("parse conversion receipt %s: %w", clean, err)
	}
	return &receipt, nil
}

// receiptAllowances is the only allowance a verification accepts: the record a
// successful conversion wrote about what it was explicitly told to leave behind.
//
// A receipt that does not prove an explicit decision proves nothing. A conversion
// that failed allowed nothing, a verification receipt allowed nothing (it is not a
// conversion), and an allowance is only real when the flag that granted it was
// set — so a hand-edited list cannot smuggle a table past the check.
func receiptAllowances(proof *Receipt, path string) (*Allowances, error) {
	if proof == nil {
		return nil, errors.New("no conversion receipt to read")
	}
	if proof.Mode != ModeConvert {
		return nil, fmt.Errorf(
			"%s is a %q receipt; only a conversion receipt can prove what an earlier conversion was allowed to leave behind",
			path, proof.Mode)
	}
	if !proof.OK {
		return nil, fmt.Errorf(
			"%s records a failed conversion (%s); a run that did not complete allowed nothing",
			path, describeFailure(proof.Failure))
	}
	if proof.Allowances == nil {
		return nil, fmt.Errorf(
			"%s records no allowances; it was written by a conversion that left nothing behind, so it proves nothing about the source's extra tables or columns", path)
	}

	allowances := &Allowances{
		tables:  make(map[string]bool),
		columns: make(map[string]map[string]bool),
		proof:   fmt.Sprintf("conversion receipt %s", path),
	}
	if proof.Options.DropExtraTables && len(proof.Allowances.Tables) > 0 {
		for _, table := range proof.Allowances.Tables {
			allowances.tables[table] = true
		}
	}
	if proof.Options.DropExtraColumns && len(proof.Allowances.Columns) > 0 {
		for table, columns := range proof.Allowances.Columns {
			allowances.columns[table] = make(map[string]bool, len(columns))
			for _, column := range columns {
				allowances.columns[table][column] = true
			}
		}
	}
	if allowances.Empty() {
		return nil, fmt.Errorf(
			"%s names allowances but does not record the flag that granted them; it proves nothing", path)
	}
	return allowances, nil
}

func describeFailure(failure *Failure) string {
	if failure == nil {
		return "no failure recorded"
	}
	return failure.Phase + ": " + failure.Message
}

// UnexpectedTables returns the source tables outside the migrated five that no
// allowance covers. It is the one implementation of that stop condition, so a
// conversion and a verification cannot disagree about it.
func UnexpectedTables(unknownTables []string, allow *Allowances) []string {
	var unexpected []string
	for _, table := range unknownTables {
		if !allow.TableAllowed(table) {
			unexpected = append(unexpected, table)
		}
	}
	sort.Strings(unexpected)
	return unexpected
}

// UnexpectedColumns returns, per table, the source columns with no destination
// column that no allowance covers.
func UnexpectedColumns(plans []*Table, allow *Allowances) map[string][]string {
	var unexpected map[string][]string
	for _, plan := range plans {
		var columns []string
		for _, column := range plan.ExtraInSource {
			if !allow.ColumnAllowed(plan.Name, column) {
				columns = append(columns, column)
			}
		}
		if len(columns) == 0 {
			continue
		}
		if unexpected == nil {
			unexpected = make(map[string][]string)
		}
		sort.Strings(columns)
		unexpected[plan.Name] = columns
	}
	return unexpected
}

// describeUnexpectedColumns renders the per-table column sets as one readable,
// deterministic phrase for a failure message.
func describeUnexpectedColumns(unexpected map[string][]string) string {
	tables := make([]string, 0, len(unexpected))
	for table := range unexpected {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	descriptions := make([]string, 0, len(tables))
	for _, table := range tables {
		descriptions = append(descriptions, fmt.Sprintf("%s: %s", table, strings.Join(unexpected[table], ", ")))
	}
	return strings.Join(descriptions, "; ")
}
