package session

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

// command_order.tsv is the C cmd_info[] table emitted in SOURCE ORDER (one row
// per entry: socials and the qui/shutdow abbreviation stubs included). Order is
// load-bearing — C's command_interpreter prefix scan matches by table line
// number, not alphabetically (R2d). Regenerate with:
//
//	go run ./cmd/dp-command-gates -oracle /path/to/darkpawns-c-oracle/src/interpreter.c
//
//go:embed command_order.tsv
var commandOrderGolden string

// orderEntry is one row of the ordered resolution table. Level drives the
// level-filter-during-scan (interpreter.c:910-911); position is intentionally
// absent because it is a post-resolution gate, not part of resolution.
type orderEntry struct {
	name  string
	level int
}

// commandOrder is the parsed ordered table, in cmd_info[] source order.
var commandOrder = mustParseCommandOrder(commandOrderGolden)

func mustParseCommandOrder(raw string) []orderEntry {
	var entries []orderEntry
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// seq⇥name⇥level — seq is present for human/audit readability; the slice
		// already preserves order, so it is not stored on the entry.
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			panic(fmt.Sprintf("command_order.tsv line %d: got %d fields, want 3", lineNo, len(fields)))
		}
		if _, err := strconv.Atoi(fields[0]); err != nil {
			panic(fmt.Sprintf("command_order.tsv line %d: invalid seq: %v", lineNo, err))
		}
		name := strings.ToLower(strings.TrimSpace(fields[1]))
		level, err := strconv.Atoi(fields[2])
		if err != nil {
			panic(fmt.Sprintf("command_order.tsv line %d: invalid level: %v", lineNo, err))
		}
		if name == "" || level < 0 {
			panic(fmt.Sprintf("command_order.tsv line %d: invalid entry", lineNo))
		}
		entries = append(entries, orderEntry{name: name, level: level})
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("read command_order.tsv: %v", err))
	}
	if len(entries) == 0 {
		panic("command_order.tsv: no entries parsed")
	}
	return entries
}

// resolveCommandPrefix resolves a typed command word to its canonical C table
// name using the three laws of command_interpreter (interpreter.c:909-912):
//
//  1. Prefix, one direction — typed must be a prefix of the table name
//     (`mur` → `murder`); typed-longer-than-entry never matches.
//  2. Table order wins — first matching entry in cmd_info[] source order. This
//     is why `qui`/`shutdow` stubs (which precede `quit`/`shutdown`) force exact
//     typing, and why `gri` → grin (not grimace).
//  3. Level filters DURING the scan — an over-level matching entry is skipped
//     and the scan continues. A mortal and an immortal typing the same prefix
//     can resolve to DIFFERENT commands (e.g. `go` → gossip for a mortal, goto
//     for an immortal).
//
// Go-only commands (ignore, map, autoloot, …) are not in the C table, so a
// prefix that only matches them misses here; the caller falls back to the
// exact-match registry, which is the only way Go-only names resolve (R4 —
// abbreviations are C surface; Go-only names are not).
//
// Returns ("", false) when no reachable entry matches.
func resolveCommandPrefix(typed string, level int) (string, bool) {
	if typed == "" {
		return "", false
	}
	for _, entry := range commandOrder { // law 2: source order
		if strings.HasPrefix(entry.name, typed) { // law 1: one-direction prefix
			if level >= entry.level { // law 3: level filter DURING the scan
				return entry.name, true
			}
		}
	}
	return "", false
}
