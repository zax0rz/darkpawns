// Population rule (round two): inspect every non-test Go function, method,
// and function literal under pkg/game, pkg/combat, and pkg/command. A function
// is in the population when a string literal is (a) nested in an argument to
// one of the player-output calls listed in outputCalls, including the named
// output callbacks private, output, ack, and sendToOutdoor, (b) in a local value
// assigned to an identifier referenced by such an argument, or (c) written
// to a local value with String() whose WriteString/Write or fmt.Fprintf output
// is passed to such an argument in that same function. The output-call catalog is explicit
// and includes the project send, Act, tell, echo, room, channel, combat, and
// notification APIs and their in-scope player-output wrappers. For (b), local
// assignments are followed transitively by identifier; literals used only in
// unrelated arguments, comments, tests, or functions that merely return a
// message are excluded. Each source string token is counted once per function.
// This is a syntactic census, not whole-program or interprocedural dataflow.
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

var outputCalls = stringSet(
	"Act", "ActActor", "ActFlagged", "ActMessage", "ActWithPrivateSink",
	"Broadcast", "RoomEcho", "Echo", "PrivateOutput",
	"Send", "SendLiteral", "SendMessage", "SendRaw", "SendTell", "SendText",
	"SendToAll", "SendToChar", "SendToRoom", "SendToZone", "SendWeaponMessage",
	"SpellActNotVict", "SpellActToChar", "SpellActToRoom", "SpellActToVictim", "SpellCreatedObjectAct", "SpellObjectAct",
	"act", "actArenaBroadcast", "actChar", "actDeliver", "actForPrivateBody", "actInRoom", "actMessage",
	"actToChar", "actToRoom", "actToRoomExclude", "actToVictim", "actorAct",
	"ack", "broadcast", "cbBroadcast", "cbSendRaw", "cbSendToChar", "channelAct", "channelActWrapped", "channelSend",
	"combatSendMessage", "communicationSend", "conSellerRoomAct", "conSellerTell", "damageGateAct",
	"do_echo", "do_tell", "message", "mobSendToZone", "mobileEquipmentAct", "movementSendToChar",
	"npcEcho", "npcGlobalEcho", "npcTell", "notifyAdmins", "notifyPlayer", "output", "performTell", "private",
	"roomMessage", "roomMessageExcludeTwo", "send", "sendActLine", "sendActLineWithWorld", "sendBodyMessage",
	"sendClanFormat", "sendColorBracketed", "sendColored", "sendCombatMessage", "sendDefenseObserverMessage",
	"sendHitMessage", "sendLiteralMessages", "sendMissMessage", "sendOk", "sendRoom", "sendSkillMessage",
	"sendSkillResult", "sendToChar", "sendToOutdoor", "sendToRoomExact", "sendToZoneExceptRoom",
	"sendWeatherRoom", "send_to_all", "send_to_char", "send_to_room", "send_to_zone", "shopNpcTell",
	"skillActSendOK", "tellFromMob", "tellVictim", "tattooRoomAct", "woundBroadcast",
	"Tell", "positionSelfLine", "wakeSelfLine",
)

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
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
				if i < len(n.Values) {
					locals[ident.Name] = append(locals[ident.Name], n.Values[i])
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if ok && i < len(n.Rhs) {
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
		if !outputCalls[callName(call.Fun)] {
			continue
		}
		for _, arg := range call.Args {
			for _, ident := range identifiers(arg) {
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
			collectLiterals(fset, file, value, used)
			for _, dependency := range identifiers(value) {
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

func identifiers(expr ast.Expr) []string {
	set := map[string]bool{}
	ast.Inspect(expr, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			set[ident.Name] = true
		}
		return true
	})
	var result []string
	for ident := range set {
		result = append(result, ident)
	}
	return result
}

func collectLiterals(fset *token.FileSet, file string, expr ast.Expr, into map[token.Pos]literal) {
	ast.Inspect(expr, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		pos := fset.Position(lit.Pos())
		into[lit.Pos()] = literal{File: file, Line: pos.Line, Raw: lit.Value, Value: value, Offset: pos.Offset}
		return true
	})
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
