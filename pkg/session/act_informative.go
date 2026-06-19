package session

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdColor — set the player's ANSI color level (off/sparse/normal/complete).
// Source: act.informative.c do_color(). The client renders; PrfColor1/2 are
// advisory state mirrored back via the "toggle" listing.
func cmdColor(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	levels := []string{"off", "sparse", "normal", "complete"}
	if len(args) == 0 {
		s.Send(fmt.Sprintf("Your current color level is %s.", colorLevelStr(s.player.Flags)))
		return nil
	}
	arg := strings.ToLower(args[0])
	if arg == "on" {
		arg = "complete"
	}
	tp := -1
	for i, lvl := range levels {
		if lvl == arg {
			tp = i
			break
		}
	}
	if tp == -1 {
		s.Send("Usage: color { Off | Sparse | Normal | Complete }")
		return nil
	}
	s.player.SetPlrFlag(game.PrfColor1, tp&1 != 0)
	s.player.SetPlrFlag(game.PrfColor2, tp&2 != 0)
	s.Send(fmt.Sprintf("Your color is now %s.", levels[tp]))
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

// cmdToggle — show the player's preference-flag grid.
// Source: act.informative.c do_toggle(), which is purely informational —
// it ignores its argument and always prints the full grid. Individual
// settings are changed via their own commands (brief, compact, notell, …,
// see commands.go's "Preference toggles" block), matching src/interpreter.c
// where each toggle is its own top-level command, not a "toggle <name>"
// dispatcher.
func cmdToggle(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	p := s.player
	g := p.Flags
	var buf strings.Builder
	buf.WriteString("Toggles:\r\n")
	// Row 1
	toggleRow(&buf, "hitpoint", prfOnOff(g, game.PrfDisphp), "brief", prfOnOff(g, game.PrfBrief), "summonable", prfOnOff(g, game.PrfSummonable))
	// Row 2
	toggleRow(&buf, "move", prfOnOff(g, game.PrfDispmove), "compact", prfOnOff(g, game.PrfCompact), "quest", prfOnOff(g, game.PrfQuest))
	// Row 3
	toggleRow(&buf, "mana", prfOnOff(g, game.PrfDispmmana), "notell", prfOnOff(g, game.PrfNotell), "norepeat", prfOnOff(g, game.PrfNoRepeat))
	// Row 4
	toggleRow(&buf, "autoexit", prfOnOffStr(p.AutoExit), "autoloot", prfOnOff(g, game.PrfAutoLoot), "autogold", prfOnOff(g, game.PrfAutoGold))
	// Row 5
	toggleRow(&buf, "autosplit", prfOnOff(g, game.PrfAutoSplit), "deaf", prfOnOff(g, game.PrfDeaf), "wimpy", wimpyStr(p.WimpLevel))
	// Row 6
	toggleRow(&buf, "nogossip", prfOnOff(g, game.PrfNoGossip), "noauction", prfOnOff(g, game.PrfNoAuctions), "nogratz", prfOnOff(g, game.PrfNoGratz))
	// Row 7
	toggleRow(&buf, "disptank", prfOnOff(g, game.PrfDispTank), "disptarget", prfOnOff(g, game.PrfDispTarget), "color", colorLevelStr(g))
	// Row 8
	toggleRow(&buf, "newbie", prfOnOff(g, game.PrfNoNewbie), "noctell", prfOnOff(g, game.PrfNoCTell), "nobroad", prfOnOff(g, game.PrfNoBroad))
	s.Send(buf.String())
	return nil
}

// toggleRow writes one row of 3 toggles with fixed-width formatting.
func toggleRow(buf *strings.Builder, name1, val1, name2, val2, name3, val3 string) {
	fmt.Fprintf(buf, "  %-12s : %-6s   %-12s : %-6s   %-12s : %s\r\n", name1, val1, name2, val2, name3, val3)
}

// prfOnOff returns "ON" or "OFF" based on a PRF flag bit in the flags bitmask.
func prfOnOff(flags uint64, bit int) string {
	if flags&(1<<bit) != 0 {
		return "ON"
	}
	return "OFF"
}

// prfOnOffStr returns "ON" or "OFF" for a bool.
func prfOnOffStr(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// wimpyStr returns the wimpy level as a string, or "OFF" if 0.
func wimpyStr(level int) string {
	if level == 0 {
		return "OFF"
	}
	return fmt.Sprintf("%d", level)
}

// colorLevelStr returns the color level string based on PRF_COLOR flags.
// 0=off, 1=sparse, 2=normal, 3=complete
func colorLevelStr(flags uint64) string {
	on1 := flags&(1<<game.PrfColor1) != 0
	on2 := flags&(1<<game.PrfColor2) != 0
	if on1 && on2 {
		return "complete"
	}
	if on2 {
		return "normal"
	}
	if on1 {
		return "sparse"
	}
	return "off"
}
