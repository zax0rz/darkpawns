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

func auditCEditorModeClosure(source, prefix string) error {
	lower := strings.ToLower(prefix)
	definition := regexp.MustCompile(`\b` + lower + `_parse\s*\([^;{}]*\)\s*\{`).FindStringIndex(source)
	cleanup := regexp.MustCompile(`\b` + lower + `_string_cleanup\s*\([^;{}]*\)\s*\{`).FindStringIndex(source)
	if definition == nil || cleanup == nil || definition[0] >= cleanup[0] {
		return fmt.Errorf("changed %s parser/cleanup shape", prefix)
	}
	parser := source[definition[0]:cleanup[0]]
	covered := map[string]bool{}
	for _, m := range regexp.MustCompile(`\bcase\s+(`+prefix+`_[A-Z_]+)\s*:`).FindAllStringSubmatch(parser, -1) {
		covered[m[1]] = true
	}
	// This mode has no parser arm: input goes to string_add until its
	// synchronous cleanup restores the extra-description menu.
	if prefix == "REDIT" {
		setup := regexp.MustCompile(`OLC_MODE\s*\(d\)\s*=\s*REDIT_EXTRADESC_DESCRIPTION;[\s\S]*?return;`).FindString(source)
		cleanupBody := source[cleanup[0]:]
		if !strings.Contains(setup, "string_write(d,") ||
			!strings.Contains(cleanupBody, "case REDIT_EXTRADESC_DESCRIPTION:") ||
			!strings.Contains(cleanupBody, "redit_disp_extradesc_menu(d);") {
			return fmt.Errorf("changed extra-description string routing")
		}
		covered["REDIT_EXTRADESC_DESCRIPTION"] = true
	}
	writes := regexp.MustCompile(`OLC_MODE\s*\(d\)\s*=\s*([^;]+);`).FindAllStringSubmatch(source, -1)
	if len(writes) == 0 {
		return fmt.Errorf("no mode writers")
	}
	if regexp.MustCompile(`OLC_MODE\s*\(d\)\s*(\+\+|--|\+=|-=)`).MatchString(source) {
		return fmt.Errorf("dynamic mode writer")
	}
	for _, m := range writes {
		mode := strings.TrimSpace(m[1])
		if !covered[mode] {
			return fmt.Errorf("unhandled/dynamic mode %q", mode)
		}
	}
	return nil
}

func proveCEditorModeClosure(t *testing.T, prefix string) {
	t.Helper()
	source := controlCSource(t, strings.ToLower(prefix)+".c")
	if err := auditCEditorModeClosure(source, prefix); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(source, "OLC_MODE(d) = "+prefix+"_MAIN_MENU;", "OLC_MODE(d) = 999;", 1)
	if changed == source {
		t.Fatal("missing main-menu assignment")
	}
	if err := auditCEditorModeClosure(changed, prefix); err == nil {
		t.Fatal("mode audit accepted an unknown mode")
	}
	changed = strings.ReplaceAll(source, "case "+prefix+"_ALIAS:", "case 999:")
	if prefix == "REDIT" {
		changed = strings.ReplaceAll(source, "case REDIT_NAME:", "case 999:")
	}
	if changed == source {
		t.Fatal("missing parser case")
	}
	if err := auditCEditorModeClosure(changed, prefix); err == nil {
		t.Fatal("mode audit accepted an unhandled assigned mode")
	}
}

func TestCReditDefaultUnreachable(t *testing.T) { proveCEditorModeClosure(t, "REDIT") }

// The audit follows installation, input precedence and synchronous save/abort
// cleanup. It deliberately rejects changed source shapes instead of treating
// arbitrary injected descriptor states as reachable C gameplay.
func auditCDescriptionRoute(editor, comm, modify, mode, menu string) error {
	prefix := strings.Split(mode, "_")[0]
	lower := strings.ToLower(prefix)
	assignment := "OLC_MODE(d) = " + mode + ";"
	if strings.Count(editor, assignment) != 1 {
		return fmt.Errorf("changed description entry set")
	}
	start := strings.Index(editor, assignment)
	end := strings.Index(editor[start:], "\n    case ")
	if end < 0 || !strings.Contains(editor[start:start+end], "string_write(d,") {
		return fmt.Errorf("description mode entered without string editor")
	}
	priority := regexp.MustCompile(`if \(d->str\)[\s\S]*?string_add\(d, comm\);\s*else if \(d->showstr_count\)[\s\S]*?else if \(d->connected != CON_PLAYING\)[\s\S]*?nanny\(d, comm\);`)
	if !priority.MatchString(comm) {
		return fmt.Errorf("string input no longer preempts nanny")
	}
	if !strings.Contains(modify, "if (action == STRINGADD_SAVE || action == STRINGADD_ABORT)") ||
		!strings.Contains(modify, "{ CON_"+prefix+"  , "+lower+"_string_cleanup }") ||
		!strings.Contains(modify, "(*cleanup_modes[i].func)(d, action);") {
		return fmt.Errorf("save/abort no longer invokes editor cleanup synchronously")
	}
	cleanupStart := regexp.MustCompile(`void ` + lower + `_string_cleanup\([^;{}]*\)\s*\{`).FindStringIndex(editor)
	if cleanupStart == nil {
		return fmt.Errorf("missing cleanup definition")
	}
	cleanup := editor[cleanupStart[0]:]
	if prefix == "REDIT" {
		if !regexp.MustCompile(`case ` + mode + `:\s*` + menu + `\(d\);\s*break;`).MatchString(cleanup) {
			return fmt.Errorf("cleanup no longer restores menu for description mode")
		}
	} else if !strings.Contains(cleanup, menu+"(d);") {
		return fmt.Errorf("cleanup no longer restores mob menu")
	}
	return nil
}

func proveCDescriptionRoute(t *testing.T, mode, menu string) {
	t.Helper()
	prefix := strings.ToLower(strings.Split(mode, "_")[0])
	editor, comm, modify := controlCSource(t, prefix+".c"), controlCSource(t, "comm.c"), controlCSource(t, "modify.c")
	audit := func(e, c, m string) error { return auditCDescriptionRoute(e, c, m, mode, menu) }
	if err := audit(editor, comm, modify); err != nil {
		t.Fatal(err)
	}
	for name, inputs := range map[string][3]string{
		"installation": {strings.ReplaceAll(editor, "string_write(d,", "removed_install(d,"), comm, modify},
		"precedence":   {editor, strings.ReplaceAll(comm, "if (d->str)", "if (0)"), modify},
		"abort":        {editor, comm, strings.ReplaceAll(modify, "action == STRINGADD_SAVE || action == STRINGADD_ABORT", "action == STRINGADD_SAVE")},
		"cleanup":      {strings.ReplaceAll(editor, menu+"(d);", "removed_menu(d);"), comm, modify},
	} {
		if err := audit(inputs[0], inputs[1], inputs[2]); err == nil {
			t.Fatalf("description audit accepted broken %s", name)
		}
	}
}

func TestCReditDescriptionUnreachable(t *testing.T) {
	proveCDescriptionRoute(t, "REDIT_DESC", "redit_disp_menu")
}
