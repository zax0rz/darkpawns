package session

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdAdmobs mirrors C adjust_mobs() (src/olc.c:279-307). The C handler
// ignores its argument, updates every mob prototype, marks the corresponding
// zones for OLC mobile saves, and sends only OK.
func cmdAdmobs(s *Session) error {
	s.manager.world.AdjustMobPrototypes()
	s.Send("Okay.\r\n")
	return nil
}

func cmdZreset(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("You must specify a zone.")
		return nil
	}

	arg := args[0]
	w := s.GetWorld()
	pw := w.GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}

	// * = reset all zones
	if arg == "*" {
		for _, z := range pw.Zones {
			slog.Warn("wizard zreset all", "by", s.playerName, "zone", z.Number)
		}
		s.Send("Reset world (async).")
		return nil
	}

	// . = current zone
	if arg == "." {
		curRoom := w.GetRoomInWorld(s.player.RoomVNum)
		if curRoom == nil {
			s.Send("Can't determine current zone.")
			return nil
		}
		zoneNum := curRoom.Zone
		z, ok := w.GetZone(zoneNum)
		if !ok || z == nil {
			s.Send("Invalid zone number.")
			return nil
		}
		slog.Warn("wizard zreset", "by", s.playerName, "zone", z.Number, "name", z.Name)
		s.Send(fmt.Sprintf("Reset zone %d (#%d): %s (async).", zoneNum, z.Number, z.Name))
		return nil
	}

	// Numeric zone number
	zoneNum, err := strconv.Atoi(arg)
	if err != nil {
		s.Send("Invalid zone number.")
		return nil
	}
	z, ok := w.GetZone(zoneNum)
	if !ok || z == nil {
		s.Send("Invalid zone number.")
		return nil
	}
	slog.Warn("wizard zreset", "by", s.playerName, "zone", z.Number, "name", z.Name)
	s.Send(fmt.Sprintf("Reset zone %d (#%d): %s (async).", zoneNum, z.Number, z.Name))
	return nil
}

// cmdZlist — list zones (LVL_IMMORT)
// Original: act.wizard.c do_zlist() — shows zone file contents, defaults to current room's zone
func cmdZlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	arg, _ := game.OneArgument(strings.Join(args, " "))
	zoneNum := mlistAtoi(arg)
	if arg == "" {
		if room := s.GetWorld().GetRoomInWorld(s.player.RoomVNum); room != nil {
			zoneNum = room.Zone
		}
	}
	path := filepath.Join(s.GetWorld().WorldPath, "zon", fmt.Sprintf("%d.zon", zoneNum))
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		s.Send("No zone file for that number.\r\n")
		return nil
	}
	PageString(s, string(data))
	return nil
}

// cmdRlist mirrors C do_rlist (src/act.wizard.c:3336-3366): the first
// argument is a numeric zone selector, not a room-name keyword. C atoi accepts
// a signed decimal prefix, and the final list is paged through the descriptor.
func cmdRlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	argument, _ := game.OneArgument(strings.Join(args, " "))
	zoneNumber := mlistAtoi(argument)
	var result strings.Builder
	count := 1
	found := false
	overflow := false
	for i := range pw.Rooms {
		if pw.Rooms[i].Zone != zoneNumber {
			continue
		}
		line := fmt.Sprintf("%3d. [%5d] %s\r\n", count, pw.Rooms[i].VNum, pw.Rooms[i].Name)
		if result.Len()+len(line) >= 8192 {
			overflow = true
			break
		}
		result.WriteString(line)
		count++
		found = true
	}
	if !found {
		s.Send("The desired zone does not exist.\r\n")
		return nil
	}
	if overflow {
		s.Send("Truncating room list due to size.\r\n")
	}
	PageString(s, result.String())
	return nil
}

// cmdOlist — list objects matching keyword (LVL_IMMORT)
func cmdOlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: olist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for i := range pw.Objs {
		if strings.Contains(strings.ToLower(pw.Objs[i].ShortDesc), keyword) ||
			strings.Contains(strings.ToLower(pw.Objs[i].Keywords), keyword) {
			count++
			fmt.Fprintf(&result, "  [%5d] %s\r\n", pw.Objs[i].VNum, pw.Objs[i].ShortDesc)
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No objects found.")
		return nil
	}
	s.Send(result.String())
	return nil
}

// cmdMlist mirrors C do_mlist (src/act.wizard.c:3376-3402): the first
// argument is a zone number, not a keyword, and output is paged through the
// descriptor. C atoi accepts a signed decimal prefix, including an empty
// string as zero.
func cmdMlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}
	argument, _ := game.OneArgument(strings.Join(args, " "))
	zoneNumber := mlistAtoi(argument)
	start := zoneNumber * 100
	end := start + 99
	var result strings.Builder
	count := 0
	for i := range pw.Mobs {
		if pw.Mobs[i].VNum >= start && pw.Mobs[i].VNum <= end {
			count++
		}
	}
	if count == 0 {
		result.WriteString("Sorry, there are no mobs in that zone.\r\n")
	} else {
		// The C loop builds each row with sprintf(buf, "%s...", buf, ...),
		// overlapping the destination and source (src/act.wizard.c:3396-3401).
		// The oracle's compiled behavior retains only this final footer; the
		// port must reproduce those player-facing bytes (R1/R5e).
		fmt.Fprintf(&result, " %d Mobiles found in Zone %d\r\n", count, zoneNumber)
	}
	PageString(s, result.String())
	return nil
}

func mlistAtoi(value string) int {
	if value == "" {
		return 0
	}
	sign := 1
	switch value[0] {
	case '-':
		sign = -1
		value = value[1:]
	case '+':
		value = value[1:]
	}
	return sign * loadAtoi(value)
}

// cmdSysfile — show system file (bugs/ideas/todo/typos) (LVL_IMMORT)
// Original: act.wizard.c do_sysfile() — reads file content and pages it
func cmdSysfile(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("That isn't a file!\r\n")
		return nil
	}
	section := strings.ToLower(args[0])

	// Map section names to data directory paths relative to server working dir
	var filePath string
	switch section {
	case "bugs":
		filePath = "data/bugs.txt"
	case "ideas":
		filePath = "data/ideas.txt"
	case "todo":
		filePath = "data/todo.txt"
	case "typos":
		filePath = "data/typos.txt"
	default:
		s.Send("That isn't a file!")
		return nil
	}

	if filePath == "" {
		s.Send("That isn't a file!")
		return nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		s.Send("File does not exist.")
		return nil
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024))
	if err != nil {
		s.Send("Error reading file.")
		return nil
	}
	s.Send(string(data))
	return nil
}

// cmdSethunt — set hunt target for a mob (LVL_GRGOD)
// Original: act.wizard.c do_sethunt() — sets a mob to hunt a visible character
func cmdSethunt(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Who do you wish to hunt?\n\r")
		return nil
	}
	victimName := args[0]
	hunterName := ""
	if len(args) > 1 {
		hunterName = strings.TrimSpace(strings.Join(args[1:], " "))
	}

	// C resolves both names through get_char_vis before comparing pointers.
	// This ordering matters: "sethunt Nobody Nobody" is a victim miss, not
	// the same-character branch (src/act.wizard.c:3452-3460).
	victim, ok := s.manager.world.ResolveCharWorld(s.player, victimName)
	if !ok {
		s.Send("No-one by that name around.\n\r")
		return nil
	}
	hunter, ok := s.manager.world.ResolveCharWorld(s.player, hunterName)
	if !ok {
		s.Send("Who shall be the hunter?\n\r")
		return nil
	}

	if hunter.Combatant == victim.Combatant {
		s.Send("Yeah right.\n\r")
		return nil
	}
	if hunter.Mob == nil {
		s.Send("PCs can't be made to hunt.\n\r")
		return nil
	}

	// C compares the wizard's level to the resolved victim, not the raw name,
	// then mutates the live mob's hunter flag and hunting pointer.
	if s.player.Level < victim.Combatant.GetLevel() {
		s.Send("Cant hunt higher than your level.\n\r")
		return nil
	}
	hunter.Mob.SetMobFlag(game.MobFlagHunter)
	hunter.Mob.SetHunting(victim.Combatant.GetName())

	slog.Warn("wizard sethunt", "by", s.playerName, "hunter", hunter.Combatant.GetName(), "victim", victim.Combatant.GetName())
	s.Send("Ok, they're fucked.\n\r")
	return nil
}

// cmdTick — force an immediate pulse/tick (LVL_IMMORT)
// Original: act.wizard.c do_tick() — calls weather_and_time, affect_update, point_update, hunt_items
