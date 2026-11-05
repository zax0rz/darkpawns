// Command metrics measures how C-shaped a set of Go packages is, for the
// idiomatic-Go experiment (docs/research/idiom-experiment/PREDICTIONS.md).
//
// Usage: go run . <repo-root> <pkg-dir>...
//
// It counts non-test .go files directly inside each package directory and
// prints one tab-separated line per package plus a total. The same binary is
// run before and after the experiment so both numbers come from one
// definition.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type stats struct {
	files, lines, funcs                int
	over100, over200, maxLen, deepNest int
	snakeIdents, emptyIface, pkgVars   int
	customData                         int
	lens                               []int
}

func (s *stats) add(o stats) {
	s.files += o.files
	s.lines += o.lines
	s.funcs += o.funcs
	s.over100 += o.over100
	s.over200 += o.over200
	s.deepNest += o.deepNest
	s.snakeIdents += o.snakeIdents
	s.emptyIface += o.emptyIface
	s.pkgVars += o.pkgVars
	s.customData += o.customData
	s.lens = append(s.lens, o.lens...)
	if o.maxLen > s.maxLen {
		s.maxLen = o.maxLen
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: metrics <repo-root> <pkg-dir>...")
		os.Exit(2)
	}
	root := os.Args[1]
	fmt.Println("pkg\tfiles\tlines\tfuncs\tmedian_func\tp90_func\tfuncs_over_100\tfuncs_over_200\tmax_func\tnest_ge_5\tsnake_idents\tempty_iface\tpkg_vars\tcustomdata_refs")
	var total stats
	for _, dir := range os.Args[2:] {
		s, err := measure(filepath.Join(root, dir))
		if err != nil {
			fmt.Fprintf(os.Stderr, "metrics: %s: %v\n", dir, err)
			os.Exit(1)
		}
		print(dir, s)
		total.add(s)
	}
	print("TOTAL", total)
}

func print(name string, s stats) {
	sort.Ints(s.lens)
	med, p90 := 0, 0
	if n := len(s.lens); n > 0 {
		med, p90 = s.lens[n/2], s.lens[n*9/10]
	}
	fmt.Printf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		name, s.files, s.lines, s.funcs, med, p90, s.over100, s.over200,
		s.maxLen, s.deepNest, s.snakeIdents, s.emptyIface, s.pkgVars, s.customData)
}

func measure(dir string) (stats, error) {
	var s stats
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return s, err
	}
	fset := token.NewFileSet()
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return s, err
		}
		f, err := parser.ParseFile(fset, p, src, 0)
		if err != nil {
			return s, err
		}
		s.files++
		s.lines += strings.Count(string(src), "\n")
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR {
				s.pkgVars += len(g.Specs)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Body == nil {
					return true
				}
				s.funcs++
				l := fset.Position(x.End()).Line - fset.Position(x.Pos()).Line + 1
				s.lens = append(s.lens, l)
				if l > 100 {
					s.over100++
				}
				if l > 200 {
					s.over200++
				}
				if l > s.maxLen {
					s.maxLen = l
				}
				if depth(x.Body, 0) >= 5 {
					s.deepNest++
				}
			case *ast.Ident:
				// Count declarations only, not every use.
				if x.Obj != nil && x.Obj.Pos() == x.Pos() && strings.Contains(strings.Trim(x.Name, "_"), "_") {
					s.snakeIdents++
				}
				if x.Name == "CustomData" {
					s.customData++
				}
			case *ast.InterfaceType:
				if len(x.Methods.List) == 0 {
					s.emptyIface++
				}
			}
			return true
		})
	}
	return s, nil
}

// depth returns the deepest nesting of if/for/switch/select below n,
// not descending into function literals.
func depth(n ast.Node, d int) int {
	m := d
	ast.Inspect(n, func(c ast.Node) bool {
		if c == n {
			return true
		}
		switch c.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt:
			if v := depth(c, d+1); v > m {
				m = v
			}
			return false
		case *ast.FuncLit:
			return false
		}
		return true
	})
	return m
}
