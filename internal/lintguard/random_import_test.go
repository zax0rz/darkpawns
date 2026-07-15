package lintguard

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestProductionCodeUsesOneRandomStream prevents new math/rand generators
// from bypassing pkg/dprng and silently breaking seeded draw-order parity.
func TestProductionCodeUsesOneRandomStream(t *testing.T) {
	root := repoRoot(t)
	var offenders []string
	for _, path := range walkGoFiles(root) {
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, "pkg/dprng/") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse imports in %s: %v", rel, err)
			continue
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if importPath == "math/rand" || importPath == "math/rand/v2" {
				offenders = append(offenders, filepath.ToSlash(rel))
				break
			}
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("production files bypass pkg/dprng with math/rand imports:\n  %s", strings.Join(offenders, "\n  "))
	}
}
