package session

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/command"
)

// C and Go use the same numeric player-level scale at the dispatcher boundary:
//
//	C LVL_IMMORT / Go LVL_IMMORT = 31
//	C LVL_GOD    / Go LVL_GOD    = 34
//	C LVL_HIGOD                  = 36
//	C LVL_GRGOD  / Go LVL_GRGOD  = 38
//	C LVL_IMPL   / Go LVL_IMPL   = 40
//
// C's LVL_BUILDER aliases LVL_IMMORT (31), and LVL_FREEZE aliases LVL_GRGOD
// (38). command_gates.tsv stores the evaluated numeric values so production
// dispatch does not depend on C preprocessor names.

type commandGate struct {
	MinLevel    int
	MinPosition int
	Source      string
}

//go:embed command_gates.tsv
var commandGateGolden string

var commandGates = mustParseCommandGates(commandGateGolden)

// registerCommand registers a built-in handler using the checked-in C-derived
// gate table. Call sites intentionally cannot supply ad hoc gate values.
func registerCommand(name string, handler command.Handler, helpText string, aliases ...string) {
	gate := mustCommandGate(name)
	cmdRegistry.Register(name, handler, helpText, gate.MinLevel, gate.MinPosition, aliases...)
}

func mustCommandGate(name string) commandGate {
	gate, ok := commandGates[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		panic(fmt.Sprintf("command %q has no authoritative gate", name))
	}
	return gate
}

func mustParseCommandGates(raw string) map[string]commandGate {
	gates := make(map[string]commandGate)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			panic(fmt.Sprintf("command_gates.tsv line %d: got %d fields, want 4", lineNo, len(fields)))
		}
		name := strings.ToLower(strings.TrimSpace(fields[0]))
		level, err := strconv.Atoi(fields[1])
		if err != nil {
			panic(fmt.Sprintf("command_gates.tsv line %d: invalid level: %v", lineNo, err))
		}
		position, err := strconv.Atoi(fields[2])
		if err != nil {
			panic(fmt.Sprintf("command_gates.tsv line %d: invalid position: %v", lineNo, err))
		}
		if name == "" || level < 0 || position < 0 || position > 8 {
			panic(fmt.Sprintf("command_gates.tsv line %d: invalid gate", lineNo))
		}
		if _, duplicate := gates[name]; duplicate {
			panic(fmt.Sprintf("command_gates.tsv line %d: duplicate command %q", lineNo, name))
		}
		gates[name] = commandGate{MinLevel: level, MinPosition: position, Source: fields[3]}
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("read command_gates.tsv: %v", err))
	}
	return gates
}
