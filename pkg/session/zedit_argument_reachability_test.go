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

// A bounded source audit of the current C argument-dispatch graph. Changed
// shapes invalidate this proof; this is not a general C parser.
func auditCZeditArgumentBranch(source, branch string) error {
	if strings.Count(source, "OLC_CMD(d).command =") != 1 ||
		!strings.Contains(source, `strchr("MOPEDGRL", OLC_CMD(d).command) == NULL`) {
		return fmt.Errorf("changed command-type writer/validation boundary")
	}
	// All four ARG3 calls are in ARG2 handling. M/O/G finish there;
	// the P/E group and D/R/L arms account for those four calls.
	start2 := strings.Index(source, "case ZEDIT_ARG2:")
	start3 := strings.Index(source, "case ZEDIT_ARG3:")
	if start2 < 0 || start3 <= start2 || strings.Count(source, "zedit_disp_arg3(d);") != 4 ||
		strings.Count(source[start2:start3], "zedit_disp_arg3(d);") != 4 {
		return fmt.Errorf("changed ARG3 caller ownership")
	}
	if strings.Count(source, "OLC_MODE(d) = ZEDIT_ARG1;") != 2 ||
		strings.Count(source, "OLC_MODE(d) = ZEDIT_ARG2;") != 1 ||
		strings.Count(source, "OLC_MODE(d) = ZEDIT_ARG3;") != 1 ||
		strings.Count(source, "zedit_disp_arg1(d);") != 2 ||
		strings.Count(source, "zedit_disp_arg2(d);") != 4 {
		return fmt.Errorf("changed argument-mode writer/caller set")
	}
	var body, incoming string
	if strings.HasPrefix(branch, "display") {
		n := strings.TrimPrefix(branch, "display")
		definition := regexp.MustCompile(`void zedit_disp_arg` + n + `\([^;{}]*\)\s*\{`).FindStringIndex(source)
		if definition == nil {
			return fmt.Errorf("missing argument display definition")
		}
		body = source[definition[0]:]
		end := strings.Index(body, "/*-------------------------------------------------------------------*/")
		if end < 0 {
			return fmt.Errorf("changed argument function delimiter")
		}
		body = body[:end]
		incoming = "MOPEDGRL"
		if n == "3" {
			// M/O/G explicitly join the failure default, unlike E/P/D/R/L.
			end := strings.LastIndex(body, "case 'M':")
			if end < 0 {
				return fmt.Errorf("changed ARG3 failure group")
			}
			body = body[:end]
			incoming = "EPDRL"
			if !strings.Contains(source, "case 'P':\n      case 'E':") ||
				!strings.Contains(source, "if (pos)\n            zedit_disp_menu(d);\n          else\n            zedit_disp_arg3(d);") {
				return fmt.Errorf("changed ARG2-to-ARG3 transitions")
			}
		}
	} else {
		n := strings.TrimPrefix(branch, "parse")
		start := strings.Index(source, "case ZEDIT_ARG"+n+":")
		endMarker := "case ZEDIT_ARG2:"
		incoming = "MOPEG"
		if n == "2" {
			endMarker = "case ZEDIT_ARG3:"
			incoming = "MOPEDGRL"
		}
		if start < 0 {
			return fmt.Errorf("missing parser branch")
		}
		body = source[start:]
		end := strings.Index(body, endMarker)
		if end < 0 {
			return fmt.Errorf("changed parser branch delimiter")
		}
		body = body[:end]
		// D/R/L bypass ARG1 at its display function; they intentionally
		// join the ARG1 parser's failure default.
		if n == "1" {
			end := strings.LastIndex(body, "case 'D':")
			if end < 0 || !strings.Contains(source, "OLC_CMD(d).arg1 = real_room(OLC_NUM(d));\n      zedit_disp_arg2(d);") {
				return fmt.Errorf("changed room-command ARG1 bypass")
			}
			body = body[:end]
		}
	}
	seen := map[string]bool{}
	for _, match := range regexp.MustCompile(`case '([A-Z])':`).FindAllStringSubmatch(body, -1) {
		seen[match[1]] = true
	}
	for _, command := range incoming {
		if !seen[string(command)] {
			return fmt.Errorf("incoming %c reaches %s default", command, branch)
		}
	}
	return nil
}

func proveCZeditArgumentBranch(t *testing.T, branch string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("source path unavailable")
	}
	for _, ext := range []string{"*.c", "*.h"} {
		paths, err := filepath.Glob(filepath.Join(filepath.Dir(file), "../../src", ext))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			if filepath.Base(path) == "zedit.c" {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if regexp.MustCompile(`\bzedit_disp_arg[123]\b`).Match(data) {
				t.Fatalf("new argument-dispatch reference outside audited owner: %s", path)
			}
		}
	}
	source := controlCSource(t, "zedit.c")
	if err := auditCZeditArgumentBranch(source, branch); err != nil {
		t.Fatal(err)
	}
	// Keep the call graph and source shape but remove a reachable handler.
	command := "M"
	if branch == "display3" {
		command = "E"
	}
	// Change only the target arm, leaving its incoming call graph intact.
	// This distinguishes branch coverage from the separate caller guards.
	start := strings.Index(source, "case ZEDIT_ARG"+strings.TrimPrefix(branch, "parse")+":")
	if strings.HasPrefix(branch, "display") {
		match := regexp.MustCompile(`void zedit_disp_arg` + strings.TrimPrefix(branch, "display") + `\([^;{}]*\)\s*\{`).FindStringIndex(source)
		if match == nil {
			t.Fatal("missing display definition")
		}
		start = match[0]
	}
	if start < 0 {
		t.Fatal("missing branch")
	}
	pos := strings.Index(source[start:], "case '"+command+"':")
	if pos < 0 {
		t.Fatal("missing reachable case")
	}
	pos += start
	local := source[:pos] + strings.Replace(source[pos:], "case '"+command+"':", "case 'X':", 1)
	if err := auditCZeditArgumentBranch(local, branch); err == nil {
		t.Fatal("argument audit accepted a missing local handler with intact caller graph")
	}
	changed := strings.ReplaceAll(source, "case '"+command+"':", "case 'X':")
	if err := auditCZeditArgumentBranch(changed, branch); err == nil {
		t.Fatal("argument audit accepted a missing reachable command handler")
	}
	changed = strings.ReplaceAll(source, `strchr("MOPEDGRL", OLC_CMD(d).command) == NULL`, "0")
	if err := auditCZeditArgumentBranch(changed, branch); err == nil {
		t.Fatal("argument audit accepted a missing command validation gate")
	}
}

func TestCZeditArg1DisplayDefaultUnreachable(t *testing.T) {
	proveCZeditArgumentBranch(t, "display1")
}

func TestCZeditArg2DisplayDefaultUnreachable(t *testing.T) { proveCZeditArgumentBranch(t, "display2") }

func TestCZeditArg3DisplayDefaultUnreachable(t *testing.T) { proveCZeditArgumentBranch(t, "display3") }
