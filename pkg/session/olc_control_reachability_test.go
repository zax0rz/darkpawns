package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func controlCSource(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("source path unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../src", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// This is a bounded source-shape audit, not a general C parser. Any new call
// form or dynamic mode writer invalidates the exclusion and requires review.
func auditParseActionClosedCalls(source string) error {
	calls := regexp.MustCompile(`\bparse_action\s*\(([^,]+),`).FindAllStringSubmatch(source, -1)
	cases := regexp.MustCompile(`\bcase\s+(PARSE_[A-Z_]+)\s*:`).FindAllStringSubmatch(source, -1)
	covered := map[string]bool{}
	for _, m := range cases {
		covered[m[1]] = true
	}
	count := 0
	for _, m := range calls {
		argument := strings.TrimSpace(m[1])
		if argument == "int command" {
			continue
		}
		if !covered[argument] {
			return fmt.Errorf("uncovered parse_action argument %q", argument)
		}
		count++
	}
	if count != 8 || len(covered) != 8 {
		return fmt.Errorf("changed parse_action call/branch set: %d/%d", count, len(covered))
	}
	return nil
}

func TestCImprovedEditorDefaultUnreachable(t *testing.T) {
	source := controlCSource(t, "improved-edit.c")
	if err := auditParseActionClosedCalls(source); err != nil {
		t.Fatal(err)
	}
	// A ninth/unknown caller must invalidate the audit, not silently exclude it.
	if err := auditParseActionClosedCalls(source + "\nparse_action(999, actions, d);\n"); err == nil {
		t.Fatal("audit accepted an unknown action caller")
	}
	if err := auditParseActionClosedCalls(strings.Replace(source, "parse_action(PARSE_DELETE,", "parse_action(999,", 1)); err == nil {
		t.Fatal("audit accepted an unknown caller with unchanged call count")
	}
	_, file, _, _ := runtime.Caller(0)
	for _, extension := range []string{"*.c", "*.h"} {
		names, err := filepath.Glob(filepath.Join(filepath.Dir(file), "../../src", extension))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if filepath.Base(name) == "improved-edit.c" || filepath.Base(name) == "improved-edit.h" {
				continue
			}
			text := controlCSource(t, filepath.Base(name))
			if regexp.MustCompile(`\bparse_action\b`).MatchString(text) {
				t.Fatalf("new parse_action reference outside audited owner: %s", name)
			}
		}
	}
	if !strings.Contains(source, "case PARSE_EDIT:") {
		t.Fatal("missing handled edit action")
	}
	if err := auditParseActionClosedCalls(strings.ReplaceAll(source, "case PARSE_EDIT:", "case 999:")); err == nil {
		t.Fatal("audit accepted an unhandled real caller")
	}
}
