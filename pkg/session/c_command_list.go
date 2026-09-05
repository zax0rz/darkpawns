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

// socialCommandOrderTSV is generated from the C cmd_info[] do_action rows,
// plus the explicit insult entry marked social by sort_commands(). It is
// already in the lexical order produced by C's cmd_sort_info sort.
//
//go:embed social_command_order.tsv
var socialCommandOrderTSV string

type cSocialOrderEntry struct {
	name     string
	minLevel int
}

var cSocialCommandOrder = parseCSocialCommandOrder(socialCommandOrderTSV)

func parseCSocialCommandOrder(data string) []cSocialOrderEntry {
	var entries []cSocialOrderEntry
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			continue
		}
		level, err := strconv.Atoi(fields[1])
		if err != nil || fields[0] == "" {
			continue
		}
		entries = append(entries, cSocialOrderEntry{name: fields[0], minLevel: level})
	}
	return entries
}

func cSocialsForLevel(level int) []string {
	names := make([]string, 0, len(cSocialCommandOrder))
	for _, entry := range cSocialCommandOrder {
		// C's socials listing has wizhelp=false, so immortal-only social rows
		// are excluded even when the viewer is immortal. The caller's level
		// still filters ordinary level-gated social rows during the scan.
		if entry.minLevel >= game.LVL_IMMORT || level < entry.minLevel {
			continue
		}
		names = append(names, entry.name)
	}
	return names
}
