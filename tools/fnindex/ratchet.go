package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type goPortPair struct {
	file     string
	function string
}

type duplicatePair struct {
	goPortPair
	cFunction   string
	duplicateOf goPortPair
	basis       string
}

func checkRatchet(root, baselinePath, duplicatesPath string, functions []goFunction) error {
	baseline, err := readBaseline(filepath.Join(root, baselinePath))
	if err != nil {
		return err
	}
	duplicates, err := readDuplicates(filepath.Join(root, duplicatesPath))
	if err != nil {
		return err
	}

	current := make(map[string]map[goPortPair]bool)
	for i := range functions {
		fn := functions[i]
		if strings.HasSuffix(fn.File, "_test.go") {
			continue
		}
		pair := goPortPair{file: fn.File, function: fn.Name}
		for _, c := range fn.CFunctions {
			if current[c] == nil {
				current[c] = make(map[goPortPair]bool)
			}
			current[c][pair] = true
		}
	}

	var violations []string
	for c, pairs := range current {
		for pair := range pairs {
			if _, approved := baseline[c][pair]; approved {
				continue
			}
			if len(pairs) > 1 || len(baseline[c]) > 0 {
				violations = append(violations, fmt.Sprintf("unbaselined C association: %s -> %s:%s", c, pair.file, pair.function))
			}
		}
	}

	activeDuplicates := make(map[string]map[goPortPair]bool)
	for _, duplicate := range duplicates {
		baseClass, approved := baseline[duplicate.cFunction][duplicate.goPortPair]
		if !approved || baseClass != "duplicate" {
			violations = append(violations, fmt.Sprintf("duplicate is not marked duplicate in baseline: %s -> %s:%s", duplicate.cFunction, duplicate.file, duplicate.function))
			continue
		}
		c := duplicate.cFunction
		if activeDuplicates[c] == nil {
			activeDuplicates[c] = make(map[goPortPair]bool)
		}
		if activeDuplicates[c][duplicate.goPortPair] {
			violations = append(violations, fmt.Sprintf("duplicate row appears more than once: %s -> %s:%s", c, duplicate.file, duplicate.function))
		}
		activeDuplicates[c][duplicate.goPortPair] = true
		if !current[c][duplicate.goPortPair] {
			violations = append(violations, fmt.Sprintf("inactive duplicate must be removed from duplicates.tsv: %s -> %s:%s", c, duplicate.file, duplicate.function))
		}
		if !current[c][duplicate.duplicateOf] {
			violations = append(violations, fmt.Sprintf("duplicate target is not an active Go association: %s -> %s:%s", c, duplicate.duplicateOf.file, duplicate.duplicateOf.function))
		}
		if _, targetApproved := baseline[c][duplicate.duplicateOf]; !targetApproved {
			violations = append(violations, fmt.Sprintf("duplicate target is not in baseline: %s -> %s:%s", c, duplicate.duplicateOf.file, duplicate.duplicateOf.function))
		}
	}

	for c, pairs := range baseline {
		for pair, class := range pairs {
			if class != "duplicate" || !current[c][pair] {
				continue
			}
			if !activeDuplicates[c][pair] {
				violations = append(violations, fmt.Sprintf("active duplicate missing from duplicates.tsv: %s -> %s:%s", c, pair.file, pair.function))
			}
		}
	}

	if len(violations) == 0 {
		return nil
	}
	sort.Strings(violations)
	return fmt.Errorf("function-index ratchet failed:\n  %s", strings.Join(violations, "\n  "))
}

func readBaseline(path string) (map[string]map[goPortPair]string, error) {
	rows, err := readTSV(path, 4)
	if err != nil {
		return nil, err
	}
	baseline := make(map[string]map[goPortPair]string)
	for line, row := range rows {
		c := row[0]
		pair := goPortPair{file: row[1], function: row[2]}
		class := row[3]
		if c == "" || pair.file == "" || pair.function == "" {
			return nil, fmt.Errorf("%s:%d: baseline fields may not be empty", path, line+2)
		}
		if class != "primary" && class != "split" && class != "duplicate" {
			return nil, fmt.Errorf("%s:%d: invalid baseline class %q", path, line+2, class)
		}
		if baseline[c] == nil {
			baseline[c] = make(map[goPortPair]string)
		}
		if _, exists := baseline[c][pair]; exists {
			return nil, fmt.Errorf("%s:%d: duplicate baseline association %s -> %s:%s", path, line+2, c, pair.file, pair.function)
		}
		baseline[c][pair] = class
	}
	return baseline, nil
}

func readDuplicates(path string) ([]duplicatePair, error) {
	rows, err := readTSV(path, 6)
	if err != nil {
		return nil, err
	}
	duplicates := make([]duplicatePair, 0, len(rows))
	for line, row := range rows {
		duplicate := duplicatePair{
			file: row[1], function: row[2],
			duplicateOf: goPortPair{file: row[3], function: row[4]},
			cFunction:   row[0],
			basis:       row[5],
		}
		if duplicate.cFunction == "" || duplicate.file == "" || duplicate.function == "" || duplicate.duplicateOf.file == "" || duplicate.duplicateOf.function == "" || duplicate.basis == "" {
			return nil, fmt.Errorf("%s:%d: duplicate fields may not be empty", path, line+2)
		}
		duplicates = append(duplicates, duplicate)
	}
	return duplicates, nil
}

func readTSV(path string, columns int) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		return nil, fmt.Errorf("%s is empty", path)
	}
	rows := [][]string{}
	for line := 2; scanner.Scan(); line++ {
		row := strings.Split(scanner.Text(), "\t")
		if len(row) != columns {
			return nil, fmt.Errorf("%s:%d: got %d TSV columns, want %d", path, line, len(row), columns)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return rows, nil
}
