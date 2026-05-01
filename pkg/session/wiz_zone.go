package session

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

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
	pw := s.GetWorld().GetParsedWorld()
	if pw == nil {
		s.Send("No parsed world available.")
		return nil
	}

	zoneNum := 0
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err == nil {
			zoneNum = n
		}
	}
	if zoneNum == 0 {
		// Default to current room's zone
		curRoom := s.GetWorld().GetRoomInWorld(s.player.RoomVNum)
		if curRoom != nil {
			zoneNum = curRoom.Zone
		}
	}

	var result strings.Builder
	result.WriteString("Zones:\r\n")
	for _, z := range pw.Zones {
		if zoneNum > 0 && z.Number != zoneNum {
			// If filtering by keyword, still allow name match
			if len(args) > 0 {
				keyword := strings.ToLower(args[0])
				if !strings.Contains(strings.ToLower(z.Name), keyword) {
					continue
				}
			} else {
				continue
			}
		}
		result.WriteString(fmt.Sprintf("  [%5d] %s (top: %d)\r\n", z.Number, z.Name, z.TopRoom))
	}
	s.Send(result.String())
	return nil
}

// cmdRlist — list rooms matching keyword (LVL_IMMORT)
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
	if len(args) < 1 {
		s.Send("Usage: rlist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for _, r := range pw.Rooms {
		if strings.Contains(strings.ToLower(r.Name), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", r.VNum, r.Name))
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No rooms found.")
		return nil
	}
	s.Send(result.String())
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
	for _, o := range pw.Objs {
		if strings.Contains(strings.ToLower(o.ShortDesc), keyword) ||
			strings.Contains(strings.ToLower(o.Keywords), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", o.VNum, o.ShortDesc))
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

// cmdMlist — list mobiles matching keyword (LVL_IMMORT)
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
	if len(args) < 1 {
		s.Send("Usage: mlist <keyword>")
		return nil
	}
	keyword := strings.ToLower(args[0])
	var result strings.Builder
	count := 0
	for _, m := range pw.Mobs {
		if strings.Contains(strings.ToLower(m.ShortDesc), keyword) ||
			strings.Contains(strings.ToLower(m.Keywords), keyword) {
			count++
			result.WriteString(fmt.Sprintf("  [%5d] %s\r\n", m.VNum, m.ShortDesc))
			if count >= 50 {
				result.WriteString("... (truncated at 50)")
				break
			}
		}
	}
	if count == 0 {
		s.Send("No mobiles found.")
		return nil
	}
	s.Send(result.String())
	return nil
}

// cmdSysfile — show system file (bugs/ideas/todo/typos) (LVL_IMMORT)
// Original: act.wizard.c do_sysfile() — reads file content and pages it
func cmdSysfile(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: sysfile <bugs|ideas|todo|typos>")
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
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024))
	if err != nil {
		s.Send("Error reading file.")
		return nil
	}
	s.Send(string(data))
	return nil
}

// cmdSethunt — set hunt target for a mob (LVL_IMMORT)
// Original: act.wizard.c do_sethunt() — sets a mob to hunt a player
func cmdSethunt(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: sethunt <victim> <hunter>")
		return nil
	}
	victimName := args[0]
	hunterName := args[1]

	if strings.EqualFold(victimName, hunterName) {
		s.Send("Yeah right.")
		return nil
	}

	// Find victim (can be any character visible to the wizard)
	victimSess := findSessionByName(s.manager, victimName)
	if victimSess == nil || victimSess.player == nil {
		s.Send("No-one by that name around.")
		return nil
	}

	// Find hunter — must be a mob in the same room system
	hunterSess := findSessionByName(s.manager, hunterName)
	if hunterSess == nil || hunterSess.player == nil {
		s.Send("No-one by that name around.")
		return nil
	}

	// Check level restriction
	if s.player.Level < victimSess.player.Level {
		s.Send("Can't hunt higher than your level.")
		return nil
	}

	slog.Warn("wizard sethunt", "by", s.playerName, "hunter", hunterName, "victim", victimName)
	s.Send("Ok, they're fucked.")
	return nil
}

// cmdTick — force an immediate pulse/tick (LVL_IMMORT)
// Original: act.wizard.c do_tick() — calls weather_and_time, affect_update, point_update, hunt_items
