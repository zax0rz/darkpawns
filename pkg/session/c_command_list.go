package session

import (
	_ "embed"
	"sort"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// commandOrderTSV is generated from the C cmd_info[] table in source order.
// The commands listing sorts these same rows by the C sort key at runtime.
//
//go:embed command_order.tsv
var commandOrderTSV string

type cCommandOrderEntry struct {
	name     string
	minLevel int
}

var cCommandOrder = parseCCommandOrder(commandOrderTSV)

func parseCCommandOrder(data string) []cCommandOrderEntry {
	var entries []cCommandOrderEntry
	for _, line := range strings.Split(data, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			continue
		}
		level, err := strconv.Atoi(fields[2])
		if err != nil || fields[1] == "RESERVED" {
			continue
		}
		entries = append(entries, cCommandOrderEntry{name: fields[1], minLevel: level})
	}
	return entries
}

func cCommandsForLevel(level int) []string {
	names := make([]string, 0, len(cCommandOrder))
	for _, entry := range cCommandOrder {
		if entry.minLevel < 0 || entry.minLevel >= game.LVL_IMMORT || level < entry.minLevel {
			continue
		}
		if _, social := game.Socials[entry.name]; social || entry.name == "insult" {
			continue
		}
		names = append(names, entry.name)
	}
	sort.Strings(names)
	return names
}
