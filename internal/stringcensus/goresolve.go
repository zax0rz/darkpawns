package stringcensus

import (
	"go/ast"
	"go/token"
	"strconv"
)

// localIndex records the single-assignment locals of one function body. Only
// unambiguous single assignments resolve; anything assigned twice, declared
// without a value, or destructured is recorded so the census can say it could
// not resolve a variable instead of guessing which value reached the sink.
type localIndex struct {
	single map[string]ast.Expr
	multi  map[string]bool
}

func newLocalIndex(body *ast.BlockStmt) *localIndex {
	idx := &localIndex{single: map[string]ast.Expr{}, multi: map[string]bool{}}
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			idx.recordAssign(node)
		case *ast.DeclStmt:
			idx.recordDecl(node)
		}
		return true
	})
	return idx
}

func (idx *localIndex) record(name string, value ast.Expr, ok bool) {
	if name == "" || name == "_" {
		return
	}
	if !ok {
		delete(idx.single, name)
		idx.multi[name] = true
		return
	}
	if _, dup := idx.single[name]; dup {
		delete(idx.single, name)
		idx.multi[name] = true
		return
	}
	if idx.multi[name] {
		return
	}
	idx.single[name] = value
}

func (idx *localIndex) recordAssign(a *ast.AssignStmt) {
	pairs := len(a.Lhs) == len(a.Rhs)
	for i, lhs := range a.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		if !pairs {
			idx.record(id.Name, nil, false)
			continue
		}
		idx.record(id.Name, a.Rhs[i], true)
	}
}

func (idx *localIndex) recordDecl(d *ast.DeclStmt) {
	gen, ok := d.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR {
		return
	}
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			switch {
			case i < len(vs.Values):
				idx.record(name.Name, vs.Values[i], true)
			default:
				idx.record(name.Name, nil, false)
			}
		}
	}
}

// classify resolves one sink argument into literals, or into an unresolved
// report. It never guesses: anything it cannot follow statically comes back as
// unresolved (when a local variable is involved) or as nothing at all (when it
// is a parameter, a field, or a non-string literal, which the report ignores).
func (g *goExtractor) classify(e ast.Expr, idx *localIndex, depth int) goValue {
	if depth > 4 {
		return goValue{}
	}
	switch node := e.(type) {
	case *ast.BasicLit:
		if node.Kind != token.STRING {
			return goValue{}
		}
		s, err := strconv.Unquote(node.Value)
		if err != nil {
			return goValue{unresolved: g.render(node)}
		}
		if s == "" {
			// An empty literal can never produce a segment; C's act() calls
			// carry several of them as unused arguments.
			return goValue{}
		}
		return goValue{literals: []string{s}}
	case *ast.ParenExpr:
		return g.classify(node.X, idx, depth)
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return g.unresolvedIfLocal(node, idx)
		}
		return concatGoValues(
			g.classify(node.X, idx, depth),
			g.classify(node.Y, idx, depth),
		)
	case *ast.Ident:
		return g.classifyIdent(node, idx, depth)
	case *ast.CallExpr:
		return g.classifyCall(node, idx, depth)
	default:
		return g.unresolvedIfLocal(e, idx)
	}
}

// classifyIdent resolves a bare identifier. A local assigned exactly once
// resolves through that assignment; a local assigned more than once is
// reported unresolved rather than guessed at; a local whose value is computed
// (a function call, a field read) is reported unresolved too, because the
// literal is somewhere inside it and the census cannot follow it. Parameters,
// globals and literals such as nil are ignored: nothing in the file set proves
// they hold player text.
func (g *goExtractor) classifyIdent(id *ast.Ident, idx *localIndex, depth int) goValue {
	switch id.Name {
	case "nil", "true", "false", "_", "":
		return goValue{}
	}
	if idx.multi[id.Name] {
		return goValue{unresolved: id.Name}
	}
	value, ok := idx.single[id.Name]
	if !ok {
		return goValue{}
	}
	v := g.classify(value, idx, depth+1)
	if len(v.literals) > 0 && v.via == "" && v.unresolved == "" {
		v.via = id.Name
	}
	if len(v.literals) == 0 && v.unresolved == "" {
		return goValue{unresolved: id.Name}
	}
	return v
}

// classifyCall follows the two formatting calls the brief names
// (fmt.Sprintf/fmt.Sprint) when their result is the sink argument. Any other
// call is a black box: if it mentions a local variable, the site is reported
// unresolved rather than guessed at.
func (g *goExtractor) classifyCall(call *ast.CallExpr, idx *localIndex, depth int) goValue {
	if isFmtFormat(g.render(call.Fun)) {
		var v goValue
		for _, arg := range call.Args {
			v = mergeGoValues(v, g.classify(arg, idx, depth+1))
		}
		return v
	}
	return g.unresolvedIfLocal(call, idx)
}

// unresolvedIfLocal reports the expression only when it mentions a local
// variable (something that provably holds runtime text this file set). A
// parameter, a struct field, or a control constant is not a lead.
func (g *goExtractor) unresolvedIfLocal(e ast.Expr, idx *localIndex) goValue {
	name := localRef(e, idx)
	if name == "" {
		return goValue{}
	}
	return goValue{unresolved: name}
}

// localRef returns the name of a local variable the expression mentions, or "".
func localRef(e ast.Expr, idx *localIndex) string {
	var found string
	ast.Inspect(e, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := idx.single[id.Name]; ok {
			found = id.Name
			return false
		}
		if idx.multi[id.Name] {
			found = id.Name
			return false
		}
		return true
	})
	return found
}
