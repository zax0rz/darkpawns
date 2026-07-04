// Package lintguard holds meta-tests that gate codebase-wide invariants.
//
// The U1000 ratchet enforces that the number of blanket
// `//lint:file-ignore U1000` suppressions only ever goes DOWN. These
// suppressions hide the shadow C-port layer's not-yet-wired functions from
// the unused-symbol linter (see docs/briefs/BRIEF-2026-07-03-dp904-u1000-inventory.md
// and DP-904). A full `staticcheck U1000` CI gate can't be turned on until
// that backlog burns down, so this ratchet is the interim guard: it stops the
// dead-code surface from growing while the per-system cleanup proceeds.
//
// When you clean up a file and remove its suppression, LOWER maxFileIgnores
// to match — that locks in the progress so it can't regress.
package lintguard

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// maxFileIgnores is the current count of `//lint:file-ignore U1000` blanket
// suppressions in the tree. It must only decrease. Retire a suppression only
// once its file is genuinely U1000-clean (`staticcheck -checks U1000 ./...`),
// then drop this number.
const maxFileIgnores = 40

const suppressionMarker = "//lint:file-ignore U1000"

// skipDirs are non-first-party or generated trees excluded from the walk,
// mirroring the `paths` exclusions in .golangci.yml.
var skipDirs = map[string]bool{
	".git":         true,
	"admin-ui":     true,
	"node_modules": true,
	"website":      true,
	"vendor":       true,
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives at <root>/internal/lintguard/u1000_ratchet_test.go
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("could not locate repo root (no go.mod at %s): %v", root, err)
	}
	return root
}

func suppressedFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var hits []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Match only real directives — a line whose trimmed form begins with
		// the marker — so string literals and prose mentions (e.g. in this
		// test file) don't self-count.
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), suppressionMarker) {
				rel, _ := filepath.Rel(root, path)
				hits = append(hits, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	sort.Strings(hits)
	return hits
}

// TestU1000SuppressionRatchet fails if the number of blanket U1000
// file-ignores exceeds the ratchet. This is the DP-904 regression guard: dead
// code can grow behind these suppressions without tripping the linter, so we
// cap and ratchet the count instead.
func TestU1000SuppressionRatchet(t *testing.T) {
	files := suppressedFiles(t)
	if len(files) > maxFileIgnores {
		t.Errorf("U1000 file-ignore suppressions grew to %d, ratchet is %d.\n"+
			"Do not add new blanket //lint:file-ignore U1000 comments — wire the code or delete it.\n"+
			"Suppressed files:\n  %s",
			len(files), maxFileIgnores, strings.Join(files, "\n  "))
	}
	// If cleanup dropped the count, nudge to lower the ratchet so progress locks in.
	if len(files) < maxFileIgnores {
		t.Errorf("U1000 file-ignores dropped to %d (ratchet is %d). Lower maxFileIgnores to %d to lock in the cleanup.",
			len(files), maxFileIgnores, len(files))
	}
}
