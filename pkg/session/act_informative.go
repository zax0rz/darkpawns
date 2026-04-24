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
		buf.WriteString(fmt.Sprintf("%-16s", name))
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

// cmdSkills — show the player's known skills and their proficiency.
func cmdSkills(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	sm := s.player.SkillManager
	if sm == nil {
		s.Send("You have no skills.")
		return nil
	}

	learned := sm.GetLearnedSkills()
	if len(learned) == 0 {
		s.Send("You have not yet learned any skills.")
		return nil
	}

	var buf strings.Builder
	buf.WriteString("Your skills:\r\n")
	buf.WriteString(fmt.Sprintf("%-20s %s\r\n", "Skill", "Level"))
	buf.WriteString(strings.Repeat("-", 30) + "\r\n")
	for _, sk := range learned {
		name := sk.DisplayName
		if name == "" {
			name = sk.Name
		}
		buf.WriteString(fmt.Sprintf("%-20s %d%%\r\n", name, sk.Level))
	}
	buf.WriteString(fmt.Sprintf("\r\nSlots used: %d/%d\r\n",
		sm.GetUsedSlots(), sm.GetSlots()))
	s.Send(buf.String())
	return nil
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
		buf.WriteString(fmt.Sprintf("  %-12s : %s\r\n", "autoexit", boolStr(s.player.AutoExit)))
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

// cmdUsers — list connected players with level and IP info (LVL_IMMORT).
func cmdUsers(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}

	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%-15s %-6s %-20s\r\n", "Name", "Level", "Remote Addr"))
	buf.WriteString(strings.Repeat("-", 45) + "\r\n")

	count := 0
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		name := sess.player.Name
		level := sess.player.GetLevel()
		ip := "unknown"
		if sess.request != nil {
			ip = sess.request.RemoteAddr
			if fwd := sess.request.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = fwd
			}
		}

		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}

		buf.WriteString(fmt.Sprintf("%-15s %-6d %-20s\r\n", name, level, ip))
		count++
	}

	buf.WriteString(fmt.Sprintf("\r\n%d player(s) connected.\r\n", count))
	s.Send(buf.String())
	return nil
}

// cmdAbils shows the player's raw abilities/stats.
// Source: act.informative.c do_abilities()
func cmdAbils(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	p := s.player
	var buf strings.Builder
	buf.WriteString("Abilities:\r\n")
	buf.WriteString(strings.Repeat("-", 30) + "\r\n")
	buf.WriteString(fmt.Sprintf("Level: %d\r\n", p.Level))
	buf.WriteString(fmt.Sprintf("Health: %d/%d  Mana: %d/%d  Move: %d/%d\r\n",
		p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
	buf.WriteString(fmt.Sprintf("STR: %d  INT: %d  WIS: %d  DEX: %d  CON: %d  CHA: %d\r\n",
		p.Stats.Str, p.Stats.Int, p.Stats.Wis, p.Stats.Dex, p.Stats.Con, p.Stats.Cha))
	buf.WriteString(fmt.Sprintf("AC: %d  Hitroll: %d  Damroll: %d\r\n", p.AC, p.Hitroll, p.Damroll))
	buf.WriteString(fmt.Sprintf("Gold: %d  XP: %d  Alignment: %d\r\n", p.Gold, p.Exp, p.Alignment))
	s.Send(buf.String())
	return nil
}

// cmdAutoExits toggles auto-exit display.
// Source: act.informative.c do_auto_exits()
func cmdAutoExits(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.player.AutoExit = !s.player.AutoExit
	state := "OFF"
	if s.player.AutoExit {
		state = "ON"
	}
	s.Send(fmt.Sprintf("Auto-exits %s.", state))
	return nil
}
