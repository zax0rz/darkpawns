package stringcensus

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// extractGoFiles parses each Go source file and reports the literals that reach
// a sink, plus the sink arguments it could not resolve.
func extractGoFiles(files []sourceFile) (cands []candidate, unresolved []unresolvedSite, err error) {
	sinks := goSinkIndex()
	for _, f := range files {
		fset := token.NewFileSet()
		parsed, perr := parser.ParseFile(fset, f.rel, f.data, parser.SkipObjectResolution)
		if perr != nil {
			return nil, nil, perr
		}
		g := &goExtractor{fset: fset}
		for _, decl := range parsed.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			idx := newLocalIndex(fd.Body)
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := goCallee(call.Fun, sinks)
				if !ok {
					return true
				}
				line := fset.Position(call.Pos()).Line
				for _, arg := range call.Args {
					v := g.classify(arg, idx, 0)
					sink := name
					if v.via != "" {
						sink += "(via " + v.via + ")"
					}
					for _, lit := range v.literals {
						cands = append(cands, candidate{
							file: f.rel, line: line, sink: sink, raw: lit,
						})
					}
					if v.unresolved != "" {
						unresolved = append(unresolved, unresolvedSite{
							file: f.rel, line: line, sink: name, expr: v.unresolved,
						})
					}
				}
				return true
			})
		}
	}
	return cands, unresolved, nil
}

// goCallee reports whether a call expression targets a player-output sink, and
// under which report label (".Send" for a method, "Act" for a function). The
// census does not type-check, so a Send-shaped call on an unrelated receiver is
// still attributed: the table is the census's declared scope, and whether the
// receiver can print is a review question.
func goCallee(fun ast.Expr, sinks map[string]goSink) (string, bool) {
	switch node := fun.(type) {
	case *ast.Ident:
		if s, ok := sinks[node.Name]; ok && s.Kind != goSinkMethod {
			return s.Name, true
		}
	case *ast.SelectorExpr:
		if s, ok := sinks["."+node.Sel.Name]; ok && s.Kind != goSinkFunc {
			return "." + s.Name, true
		}
	}
	return "", false
}
