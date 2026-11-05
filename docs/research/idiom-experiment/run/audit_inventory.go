// Audit inventory helper for the pre-refactor idiom experiment.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("kind\tfile\tline\tname\tlines_or_depth")
	for _, pkg := range []string{"pkg/game", "pkg/combat", "pkg/command"} {
		files, _ := filepath.Glob(filepath.Join(pkg, "*.go"))
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			set := token.NewFileSet()
			f, err := parser.ParseFile(set, path, nil, 0)
			if err != nil {
				panic(err)
			}
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				length := set.Position(fn.End()).Line - set.Position(fn.Pos()).Line + 1
				depth := nesting(fn.Body, 0)
				name := fn.Name.Name
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					name = receiver(fn.Recv.List[0].Type) + "." + name
				}
				if length > 100 {
					fmt.Printf("long\t%s\t%d\t%s\t%d\n", path, set.Position(fn.Pos()).Line, name, length)
				}
				if depth >= 5 {
					fmt.Printf("deep\t%s\t%d\t%s\t%d\n", path, set.Position(fn.Pos()).Line, name, depth)
				}
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					if x.Obj != nil && x.Obj.Pos() == x.Pos() && strings.Contains(strings.Trim(x.Name, "_"), "_") {
						fmt.Printf("snake\t%s\t%d\t%s\t-\n", path, set.Position(x.Pos()).Line, x.Name)
					}
				case *ast.InterfaceType:
					if len(x.Methods.List) == 0 {
						fmt.Printf("interface\t%s\t%d\tinterface{}\t-\n", path, set.Position(x.Pos()).Line)
					}
				}
				return true
			})
		}
	}
}

func receiver(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return receiver(x.X)
	case *ast.Ident:
		return x.Name
	case *ast.IndexExpr:
		return receiver(x.X)
	case *ast.IndexListExpr:
		return receiver(x.X)
	default:
		return "?"
	}
}

func nesting(n ast.Node, depth int) int {
	max := depth
	ast.Inspect(n, func(child ast.Node) bool {
		if child == n {
			return true
		}
		switch child.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			if got := nesting(child, depth+1); got > max {
				max = got
			}
			return false
		case *ast.FuncLit:
			return false
		}
		return true
	})
	return max
}

var _ = os.Stdout
