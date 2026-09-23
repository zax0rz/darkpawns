package stringcensus

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// goSinkKind says how a sink is spelled at a call site.
type goSinkKind int

const (
	// goSinkMethod is a call on a receiver: x.Send(...).
	goSinkMethod goSinkKind = iota
	// goSinkFunc is a package-level call: Act(...).
	goSinkFunc
	// goSinkBoth is a name that appears as both (SendToRoom does).
	goSinkBoth
)

// goSink is one player-output sink in the Go tree. This table is the whole
// scope of the Go side of the census; every string the tool reports as
// player-facing came through one of these. Counts in Why are the brief's grep
// counts for 2026-09-23, kept so the table can be re-derived.
type goSink struct {
	// Name is the bare call name, without a receiver dot.
	Name string
	// Kind says how the name is spelled at a call site.
	Kind goSinkKind
	// Why says what the sink is, for the sink table in the report.
	Why string
}

// goSinks is the Go sink table. It is the brief's list plus the thin
// forwarding helpers this tree wrapped around those primitives (a literal
// passed to sendToChar is a literal send_to_char receives), each verified to
// forward straight into a listed sink.
var goSinks = []goSink{
	{Name: "Send", Kind: goSinkMethod, Why: "Session.Send and wrapper sessions (679 call sites)"},
	{Name: "SendMessage", Kind: goSinkMethod, Why: "Player/MobInstance.SendMessage (599 call sites)"},
	{Name: "Act", Kind: goSinkFunc, Why: "pkg/game Act(): the act()-style primitive (427 call sites)"},
	{Name: "sendText", Kind: goSinkMethod, Why: "Session.sendText, group and mail text (73 call sites)"},
	{Name: "SendToRoom", Kind: goSinkBoth, Why: "World.SendToRoom and act.go SendToRoom(): room broadcast (24 call sites)"},
	{Name: "BroadcastToRoom", Kind: goSinkMethod, Why: "SessionManager.BroadcastToRoom, socket-level room broadcast (22 call sites)"},
	{Name: "actToRoom", Kind: goSinkMethod, Why: "World.actToRoom, raw room line (20 call sites)"},
	{Name: "SendToAll", Kind: goSinkMethod, Why: "World.SendToAll and SessionManager.SendToAll (10 call sites)"},
	{Name: "sendPromptText", Kind: goSinkMethod, Why: "Session.sendPromptText: prompt bytes are player-facing"},
	{Name: "SendToZone", Kind: goSinkMethod, Why: "World.SendToZone and the script adapter"},
	{Name: "SendToOutdoor", Kind: goSinkMethod, Why: "SessionManager.SendToOutdoor, weather broadcast"},
	{Name: "channelAct", Kind: goSinkMethod, Why: "pkg/game/out_of_band.go channelAct(): channel lines"},
	{Name: "channelSend", Kind: goSinkMethod, Why: "pkg/game/out_of_band.go channelSend(): the sender's own channel echo"},
	{Name: "SendToChar", Kind: goSinkFunc, Why: "act.go:600 SendToChar(): one-line Act(ToChar) wrapper"},
	{Name: "SendToVict", Kind: goSinkFunc, Why: "act.go:605 SendToVict(): one-line Act(ToVict) wrapper"},
	{Name: "sendToChar", Kind: goSinkFunc, Why: "spec_procs.go:85 sendToChar(): one-line SendMessage wrapper"},
	{Name: "movementSendToChar", Kind: goSinkFunc, Why: "movement_commands.go:12: one-line Act(ToChar|ToSleep) wrapper"},
	{Name: "communicationSend", Kind: goSinkFunc, Why: "directed_speech.go:56: one-line Act(ToChar|ToSleep) wrapper"},
}

// goSinkIndex maps every call spelling ("Send" as a function, ".Send" as a
// method) to its sink, so one table entry covers both forms.
func goSinkIndex() map[string]goSink {
	m := make(map[string]goSink, len(goSinks)*2)
	for _, s := range goSinks {
		if s.Kind != goSinkFunc {
			m["."+s.Name] = s
		}
		if s.Kind != goSinkMethod {
			m[s.Name] = s
		}
	}
	return m
}

// sortedGoSinkNames returns the sink table's names in report order.
func sortedGoSinkNames() []string {
	names := make([]string, 0, len(goSinks))
	for _, s := range goSinks {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

// unresolvedSite is a sink argument the census refused to guess about: an
// expression that mentions a local variable whose value it could not resolve to
// literals. It is reported so a human can tighten the sink table or the
// extractor; it never gates the ratchet.
type unresolvedSite struct {
	file string
	line int
	sink string
	expr string
}

// goValue is the classification of one sink argument.
type goValue struct {
	// literals are the unescaped literals that reach the sink.
	literals []string
	// via names the local variable a resolved literal travelled through.
	via string
	// unresolved is the source text of an argument the census could not
	// resolve (empty when there is nothing to report).
	unresolved string
}

// goExtractor renders expressions for the unresolved report.
type goExtractor struct {
	fset *token.FileSet
}

func (g *goExtractor) render(n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, g.fset, n); err != nil {
		return fmt.Sprintf("<%T>", n)
	}
	out := buf.String()
	if len(out) > 120 {
		out = out[:117] + "..."
	}
	return out
}

func mergeGoValues(a, b goValue) goValue {
	out := goValue{literals: append(append([]string{}, a.literals...), b.literals...)}
	switch {
	case a.unresolved != "":
		out.unresolved = a.unresolved
	case b.unresolved != "":
		out.unresolved = b.unresolved
	}
	if a.via == b.via {
		out.via = a.via
	}
	return out
}

// concatGoValues joins the two sides of a Go "+". When both sides are pure
// literals the result is one literal, which is what C's adjacent literals "a"
// "b" produce; when either side is runtime text the fragments stay separate,
// because joining them would invent a string the output never contains.
func concatGoValues(a, b goValue) goValue {
	if a.via == "" && b.via == "" && a.unresolved == "" && b.unresolved == "" {
		return goValue{literals: []string{strings.Join(append(append([]string{}, a.literals...), b.literals...), "")}}
	}
	return mergeGoValues(a, b)
}

func isFmtFormat(callee string) bool {
	return callee == "fmt.Sprintf" || callee == "fmt.Sprint"
}
