package session

import (
	"fmt"
	"sort"
	"strings"
)

// cmdColor — toggle ANSI color on or off.
// The client is responsible for rendering; this flag is advisory.
func cmdColor(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Usage: color <on|off>")
		return nil
	}
	switch strings.ToLower(args[0]) {
	case "on":
		s.Send("Color enabled.")
	case "off":
		s.Send("Color disabled.")
	default:
		s.Send("Usage: color <on|off>")
	}
	return nil
}

// cmdCommands — list all commands available at the player's level.
func cmdCommands(s *Session, args []string) error {
	entries := cmdRegistry.GetAll()

	// Filter by player level and sort alphabetically
	level := 0
	if s.player != nil {
		level = s.player.GetLevel()
	}

	var names []string
	for _, e := range entries {
		if level >= e.MinLevel {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		s.Send("No commands available.")
		return nil
	}

	// Print in columns of 5
	var buf strings.Builder
	buf.WriteString("Commands available:\r\n")
	for i, name := range names {
		fmt.Fprintf(&buf, "%-16s", name)
		if (i+1)%5 == 0 {
			buf.WriteString("\r\n")
		}
	}
	if len(names)%5 != 0 {
		buf.WriteString("\r\n")
	}
	s.Send(buf.String())
	return nil
}

// cmdDescription — set the player's character description.
func cmdDescription(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	if len(args) == 0 {
		s.Send("Set your description to what?")
		return nil
	}
	s.player.Description = strings.Join(args, " ")
	s.Send("Description set.")
	return nil
}

// cmdDiagnose — show the health status of a target (or self if no target).
func cmdDiagnose(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	// Default: diagnose self
	if len(args) == 0 {
		pct := 100
		if s.player.MaxHealth > 0 {
			pct = (s.player.Health * 100) / s.player.MaxHealth
		}
		s.Send(fmt.Sprintf("You are %s.\r\n%s has %d/%d hit points.",
			diagnoseLabel(pct), s.player.Name, s.player.Health, s.player.MaxHealth))
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Check players
	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if strings.Contains(strings.ToLower(p.Name), targetName) {
			pct := 100
			if p.MaxHealth > 0 {
				pct = (p.Health * 100) / p.MaxHealth
			}
			s.Send(fmt.Sprintf("%s is %s.\r\n%s has %d/%d hit points.",
				p.Name, diagnoseLabel(pct), p.Name, p.Health, p.MaxHealth))
			return nil
		}
	}

	// Check mobs
	for _, mob := range s.manager.world.GetMobsInRoom(room.VNum) {
		if strings.Contains(strings.ToLower(mob.GetName()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			pct := 100
			max := mob.GetMaxHP()
			cur := mob.GetHP()
			if max > 0 {
				pct = (cur * 100) / max
			}
			s.Send(fmt.Sprintf("%s is %s.", mob.GetShortDesc(), diagnoseLabel(pct)))
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// diagnoseLabel returns a health description based on HP percentage.
func diagnoseLabel(pct int) string {
	switch {
	case pct >= 100:
		return "in excellent condition"
	case pct >= 90:
		return "slightly scratched"
	case pct >= 75:
		return "lightly wounded"
	case pct >= 50:
		return "moderately wounded"
	case pct >= 30:
		return "heavily wounded"
	case pct >= 15:
		return "severely wounded"
	case pct >= 1:
		return "mortally wounded"
	default:
		return "dead"
	}
}

// cmdToggle — toggle a player preference flag.
func cmdToggle(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	if len(args) == 0 {
		// Show current toggles
		var buf strings.Builder
		buf.WriteString("Toggles:\r\n")
		fmt.Fprintf(&buf, "  %-12s : %s\r\n", "autoexit", boolStr(s.player.AutoExit))
		s.Send(buf.String())
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "autoexit":
		s.player.AutoExit = !s.player.AutoExit
		s.Send(fmt.Sprintf("Auto-exit %s.", boolStr(s.player.AutoExit)))
	default:
		s.Send(fmt.Sprintf("Unknown toggle '%s'. Try: autoexit", args[0]))
	}
	return nil
}

// boolStr returns "ON" or "OFF".
func boolStr(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

