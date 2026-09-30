// fidelitytests indexes Go test declarations without treating comments or strings as proof.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type testDecl struct {
	Name               string   `json:"name"`
	Package            string   `json:"package"`
	File               string   `json:"file"`
	Line               int      `json:"line"`
	Valid              bool     `json:"valid"`
	Subtests           []string `json:"subtests"`
	DynamicSubtests    bool     `json:"dynamic_subtests"`
	Vacuous            bool     `json:"vacuous"`
	VacuousSubtests    []string `json:"vacuous_subtests"`
	DirectFailureCalls int      `json:"direct_failure_calls"`
}

type index struct {
	Tests []testDecl        `json:"tests"`
	Files map[string]string `json:"files"`
}

func isTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	rest := strings.TrimPrefix(name, "Test")
	if rest == "" {
		return true
	}
	return !unicode.IsLower([]rune(rest)[0])
}

func validTest(fn *ast.FuncDecl, testingAlias string) bool {
	if fn.Recv != nil || !isTestName(fn.Name.Name) || fn.Type.TypeParams != nil {
		return false
	}
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	field := fn.Type.Params.List[0]
	if len(field.Names) > 1 {
		return false
	}
	star, ok := field.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	if testingAlias == "." {
		ident, ok := star.X.(*ast.Ident)
		return ok && ident.Name == "T"
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "T" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && testingAlias != "" && ident.Name == testingAlias
}

func parameterName(fn *ast.FuncType) string {
	if fn.Params == nil || len(fn.Params.List) != 1 || len(fn.Params.List[0].Names) != 1 {
		return ""
	}
	return fn.Params.List[0].Names[0].Name
}

// Conservative: only empty bodies or logging-only bodies are confirmed vacuous.
// A helper call may assert or test panic-free execution, so it is never called vacuous here.
func vacuous(body *ast.BlockStmt, param string) bool {
	if body == nil {
		return false
	}
	for _, stmt := range body.List {
		if ret, ok := stmt.(*ast.ReturnStmt); ok && len(ret.Results) == 0 {
			continue
		}
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			return false
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != param || (sel.Sel.Name != "Log" && sel.Sel.Name != "Logf") {
			return false
		}
		// Logging arguments themselves can invoke a checker or panic; fail closed.
		for _, arg := range call.Args {
			if _, ok := arg.(*ast.BasicLit); !ok {
				return false
			}
		}
	}
	return true
}

func subtests(body *ast.BlockStmt, param, prefix string, names *[]string, empty *[]string, dynamic *bool) {
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Run" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != param {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			*dynamic = true
			return true
		}
		name, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		// testing rewrites whitespace in subtest names to underscores.
		name = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return '_'
			}
			return r
		}, name)
		path := prefix + name
		*names = append(*names, path)
		callback, ok := call.Args[1].(*ast.FuncLit)
		if ok {
			if vacuous(callback.Body, parameterName(callback.Type)) {
				*empty = append(*empty, path)
			}
			subtests(callback.Body, parameterName(callback.Type), path+"/", names, empty, dynamic)
		}
		return false
	})
}

func scan(root string) (index, error) {
	result := index{Tests: []testDecl{}, Files: map[string]string{}}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules" || entry.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		result.Files[rel] = string(data)
		file, err := parser.ParseFile(fset, path, data, 0)
		if err != nil {
			return err
		}
		alias := ""
		for _, imp := range file.Imports {
			if imp.Path.Value == `"testing"` {
				alias = "testing"
				if imp.Name != nil {
					alias = imp.Name.Name
				}
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			pkg := filepath.ToSlash(filepath.Dir(rel))
			if pkg == "." {
				pkg = "."
			} else {
				pkg = "./" + pkg
			}
			item := testDecl{Name: fn.Name.Name, Package: pkg, File: rel, Line: fset.Position(fn.Pos()).Line, Valid: validTest(fn, alias), Subtests: []string{}}
			if item.Valid {
				item.Vacuous = vacuous(fn.Body, parameterName(fn.Type))
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					switch sel.Sel.Name {
					case "Fatal", "Fatalf", "Error", "Errorf", "Fail", "FailNow":
						item.DirectFailureCalls++
					}
					return true
				})
				subtests(fn.Body, parameterName(fn.Type), "", &item.Subtests, &item.VacuousSubtests, &item.DynamicSubtests)
				sort.Strings(item.Subtests)
			}
			result.Tests = append(result.Tests, item)
		}
		return nil
	})
	return result, err
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fidelitytests <repository-root>")
		os.Exit(2)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
