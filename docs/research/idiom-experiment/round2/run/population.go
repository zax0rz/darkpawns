// Population rule (round two): inspect every non-test Go function, method,
// and function literal under pkg/game, pkg/combat, and pkg/command. A function
// is in the population when a string literal containing at least one ASCII
// letter is in a message argument position listed in outputArguments,
// including the named output
// callbacks private, output, ack, and sendToOutdoor; assigned to a local string
// expression that is referenced by one of those arguments; or written to a
// local value with String() whose WriteString/Write or fmt.Fprintf output is
// passed to one of those arguments in that same function. The map below gives
// explicit zero-based argument positions for every recognized send, Act,
// tell, echo, room, channel, combat, notification, and message callback API.
// Only string-like local assignment expressions are followed transitively.
// Literals used only in non-message arguments, comments, tests, or functions
// that merely return a message are excluded. Each source string token is
// counted once per function. This is syntactic, same-function dataflow, not
// whole-program or interprocedural analysis.
//
// Run from the repository root: go run ./docs/research/idiom-experiment/round2/run/population.go
// It writes population.tsv and population-literals.tsv beside this file.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type literal struct {
	File   string
	Line   int
	Raw    string
	Value  string
	Offset int
}

type row struct {
	Func     string
	File     string
	Line     int
	Literals []literal
}

var outputArguments = map[string][]int{
	"Act": {0, 6, 7}, "ActWithPrivateSink": {6, 7}, "Echo": {3}, "PrivateOutput": {0},
	"RoomEcho": {1}, "Send": {0}, "SendLiteral": {0}, "SendMessage": {0}, "SendRaw": {0},
	"SendTell": {1}, "SendText": {0}, "SendToAll": {0}, "SendToChar": {1}, "SendToRoom": {1},
	"SendToZone": {1}, "SpellActNotVict": {2}, "SpellActToChar": {3}, "SpellActToRoom": {2},
	"SpellActToVictim": {2}, "SpellObjectAct": {2}, "Tell": {1},
	"ack": {0}, "act": {3}, "actArenaBroadcast": {6, 7}, "actDeliver": {6, 7},
	"actForPrivateBody": {6, 7}, "actInRoom": {2}, "actMessage": {3, 4, 5},
	"actToChar": {1}, "actToRoom": {1, 2}, "actToRoomExclude": {2}, "actToVictim": {2},
	"actorAct": {0}, "broadcast": {1}, "cbBroadcast": {1}, "cbSendRaw": {1}, "cbSendToChar": {1},
	"channelAct": {4}, "channelActWrapped": {4}, "channelSend": {2}, "combatSendMessage": {1},
	"communicationSend": {1}, "conSellerRoomAct": {4}, "conSellerTell": {3}, "damageGateAct": {2}, "mobSendToZone": {1},
	"mobileEquipmentAct": {2}, "movementSendToChar": {1}, "npcEcho": {1}, "npcGlobalEcho": {1},
	"npcTell": {1}, "notifyAdmins": {0}, "notifyPlayer": {1}, "output": {0}, "performTell": {2},
	"positionSelfLine": {2}, "private": {0}, "roomMessage": {1}, "roomMessageExcludeTwo": {1},
	"send": {0}, "sendActLine": {1}, "sendActLineWithWorld": {2}, "sendBodyMessage": {2},
	"sendColorBracketed": {2}, "sendColored": {2}, "sendCombatMessage": {1},
	"sendDefenseObserverMessage": {2}, "sendRoom": {0},
	"sendToChar": {0, 1}, "sendToOutdoor": {0}, "sendToRoomExact": {1}, "sendToZoneExceptRoom": {1},
	"sendWeatherRoom": {2}, "send_to_all": {0}, "send_to_char": {1}, "send_to_room": {1},
	"send_to_zone": {1}, "shopNpcTell": {1}, "tellFromMob": {3}, "tellVictim": {0},
	"tattooRoomAct": {2}, "wakeSelfLine": {2}, "woundBroadcast": {1},
}

func main() {
	root := "."
	fset := token.NewFileSet()
	var rows []row
	for _, dir := range []string{"pkg/game", "pkg/combat", "pkg/command"} {
		files, err := filepath.Glob(filepath.Join(root, dir, "*.go"))
		check(err)
		sort.Strings(files)
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			check(err)
			rel, err := filepath.Rel(root, path)
			check(err)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				name := functionName(fn)
				rows = append(rows, analyzeFunction(fset, rel, name, fset.Position(fn.Pos()).Line, fn.Body)...)
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].File != rows[j].File {
			return rows[i].File < rows[j].File
		}
		if rows[i].Line != rows[j].Line {
			return rows[i].Line < rows[j].Line
		}
		return rows[i].Func < rows[j].Func
	})

	base := filepath.Dir("docs/research/idiom-experiment/round2/run/population.go")
	population, err := os.Create(filepath.Join(base, "population.tsv"))
	check(err)
	defer population.Close()
	_, err = fmt.Fprintln(population, "func\tfile\tline\tn_literals")
	check(err)

	literalFile, err := os.Create(filepath.Join(base, "population-literals.tsv"))
	check(err)
	defer literalFile.Close()
	_, err = fmt.Fprintln(literalFile, "func\tfile\tline\tliteral\toffset")
	check(err)

	for _, item := range rows {
		_, err = fmt.Fprintf(population, "%s\t%s\t%d\t%d\n", item.Func, item.File, item.Line, len(item.Literals))
		check(err)
		for _, lit := range item.Literals {
			_, err = fmt.Fprintf(literalFile, "%s\t%s\t%d\t%s\t%d\n", item.Func, item.File, lit.Line, escapeTSV(lit.Raw), lit.Offset)
			check(err)
		}
	}
}

func analyzeFunction(fset *token.FileSet, file, name string, line int, body *ast.BlockStmt) []row {
	locals := map[string][]ast.Expr{}
	var closures []*ast.FuncLit
	var calls []*ast.CallExpr
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		if closure, ok := node.(*ast.FuncLit); ok {
			closures = append(closures, closure)
			return false
		}
		switch n := node.(type) {
		case *ast.ValueSpec:
			for i, ident := range n.Names {
				if i < len(n.Values) && isStringMessageExpr(n.Values[i]) {
					locals[ident.Name] = append(locals[ident.Name], n.Values[i])
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if ok && i < len(n.Rhs) && isStringMessageExpr(n.Rhs[i]) {
					locals[ident.Name] = append(locals[ident.Name], n.Rhs[i])
				}
			}
		case *ast.CallExpr:
			calls = append(calls, n)
		}
		return true
	})

	used := map[token.Pos]literal{}
	usedNames := map[string]bool{}
	stringBuffers := map[string]bool{}
	for _, call := range calls {
		indices, ok := outputArguments[callName(call.Fun)]
		if !ok {
			continue
		}
		for _, index := range indices {
			if index >= len(call.Args) {
				continue
			}
			arg := call.Args[index]
			if isEmptyStringLiteral(arg) {
				continue
			}
			for _, ident := range messageIdentifiers(arg) {
				usedNames[ident] = true
			}
			ast.Inspect(arg, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok && callName(call.Fun) == "String" {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							stringBuffers[ident.Name] = true
						}
					}
				}
				return true
			})
			collectLiterals(fset, file, arg, used)
		}
	}

	// Follow values of local identifiers that participate in output arguments.
	queue := make([]string, 0, len(usedNames))
	for ident := range usedNames {
		queue = append(queue, ident)
	}
	seenNames := map[string]bool{}
	for len(queue) > 0 {
		ident := queue[0]
		queue = queue[1:]
		if seenNames[ident] {
			continue
		}
		seenNames[ident] = true
		for _, value := range locals[ident] {
			if !isStringMessageExpr(value) {
				continue
			}
			markStringBuffers(value, stringBuffers)
			collectLiterals(fset, file, value, used)
			for _, dependency := range messageIdentifiers(value) {
				if !seenNames[dependency] {
					queue = append(queue, dependency)
				}
			}
		}
	}

	// A local builder/buffer counts only when its String() is sent above.
	if len(stringBuffers) > 0 {
		for _, call := range calls {
			callee := call.Fun
			if sel, ok := callee.(*ast.SelectorExpr); ok && (sel.Sel.Name == "WriteString" || sel.Sel.Name == "Write") {
				if receiver, ok := sel.X.(*ast.Ident); ok && stringBuffers[receiver.Name] {
					for _, arg := range call.Args {
						collectLiterals(fset, file, arg, used)
					}
				}
			}
			if callName(call.Fun) == "Fprintf" && len(call.Args) >= 2 {
				if isLocalBufferAddress(call.Args[0], stringBuffers) {
					for _, arg := range call.Args[1:] {
						collectLiterals(fset, file, arg, used)
					}
				}
			}
		}
	}

	var out []literal
	for _, lit := range used {
		out = append(out, lit)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Offset < out[j].Offset })
	var result []row
	if len(out) > 0 {
		result = append(result, row{Func: name, File: file, Line: line, Literals: out})
	}
	for _, closure := range closures {
		closureLine := fset.Position(closure.Pos()).Line
		closureName := fmt.Sprintf("%s::func@%d", name, closureLine)
		result = append(result, analyzeFunction(fset, file, closureName, closureLine, closure.Body)...)
	}
	return result
}

func functionName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	recv := types.ExprString(fn.Recv.List[0].Type)
	return recv + "." + fn.Name.Name
}

func callName(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.SelectorExpr:
		return n.Sel.Name
	default:
		return ""
	}
}

func messageIdentifiers(expr ast.Expr) []string {
	set := map[string]bool{}
	var visit func(ast.Expr)
	visit = func(value ast.Expr) {
		switch node := value.(type) {
		case *ast.Ident:
			set[node.Name] = true
		case *ast.BinaryExpr:
			if node.Op == token.ADD {
				visit(node.X)
				visit(node.Y)
			}
		case *ast.CallExpr:
			if isStringMessageExpr(node) {
				for _, arg := range node.Args {
					visit(arg)
				}
				if selector, ok := node.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "String" {
					if receiver, ok := selector.X.(*ast.Ident); ok {
						set[receiver.Name] = true
					}
				}
			}
		case *ast.SelectorExpr:
			visit(node.X)
		}
	}
	visit(expr)
	var result []string
	for ident := range set {
		result = append(result, ident)
	}
	return result
}

func isStringMessageExpr(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.BasicLit:
		return node.Kind == token.STRING
	case *ast.Ident:
		return true
	case *ast.BinaryExpr:
		return node.Op == token.ADD && (isStringMessageExpr(node.X) || isStringMessageExpr(node.Y))
	case *ast.CallExpr:
		switch callName(node.Fun) {
		case "Sprintf", "Sprintln", "Sprint", "Join", "Repeat", "Replace", "ReplaceAll", "TrimSpace", "ToLower", "ToUpper", "Title", "CapitalizeSentence", "ActMessage", "DeleteANSIControls", "String":
			return true
		}
	}
	return false
}

func isEmptyStringLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	value, err := strconv.Unquote(lit.Value)
	return err == nil && value == ""
}

func markStringBuffers(expr ast.Expr, buffers map[string]bool) {
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || callName(call.Fun) != "String" {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if ok {
			buffers[receiver.Name] = true
		}
		return true
	})
}

func collectLiterals(fset *token.FileSet, file string, expr ast.Expr, into map[token.Pos]literal) {
	ast.Inspect(expr, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil || !hasASCIILetter(value) {
			return true
		}
		pos := fset.Position(lit.Pos())
		into[lit.Pos()] = literal{File: file, Line: pos.Line, Raw: lit.Value, Value: value, Offset: pos.Offset}
		return true
	})
}

func hasASCIILetter(value string) bool {
	for _, r := range value {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func isLocalBufferAddress(expr ast.Expr, buffers map[string]bool) bool {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return false
	}
	ident, ok := unary.X.(*ast.Ident)
	return ok && buffers[ident.Name]
}

func escapeTSV(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\t", "\\t", "\r", "\\r", "\n", "\\n").Replace(value)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
