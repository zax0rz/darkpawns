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
		if strings.HasPrefix(lvl, arg) {
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
	target := s.player
	if len(args) > 0 {
		target = nil
		for _, candidate := range s.manager.world.GetAllPlayers() {
			if strings.EqualFold(candidate.GetName(), args[0]) && game.CanSee(s.player, candidate) {
				target = candidate
				break
			}
		}
		if target == nil {
			s.Send("Who is that?\r\n")
			return nil
		}
		if s.player.GetLevel() < target.GetLevel() {
			s.Send("You can't see the commands of people above your level.\r\n")
			return nil
		}
	}

	entries := cmdRegistry.GetAll()

	// Filter by player level and sort alphabetically
	level := 0
	if target != nil {
		level = target.GetLevel()
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

// cmdSocials — "socials" lists the available social commands.
// Source: act.informative.c do_commands/SCMD_SOCIALS — lists commands whose handler is
// do_action (the social handler). The Go equivalent is the game.Socials map.
func cmdSocials(s *Session, args []string) error {
	names := make([]string, 0, len(game.Socials))
	for name := range game.Socials {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		s.Send("No socials available.")
		return nil
	}
	var buf strings.Builder
	buf.WriteString("The following socials are available to you:\r\n")
	for i, name := range names {
		fmt.Fprintf(&buf, "%-16s", name)
		if (i+1)%7 == 0 {
			buf.WriteString("\r\n")
		}
	}
	if len(names)%7 != 0 {
		buf.WriteString("\r\n")
	}
	s.Send(buf.String())
	return nil
}

// cmdWizhelp — "wizhelp" lists the privileged (immortal+) commands.
// Source: act.informative.c do_commands/SCMD_WIZHELP — lists commands whose min level is
// >= LVL_IMMORT. The Go registry carries MinLevel on every entry.
func cmdWizhelp(s *Session, args []string) error {
	entries := cmdRegistry.GetAll()
	var names []string
	for _, e := range entries {
		if e.MinLevel >= LVL_IMMORT {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		s.Send("No privileged commands available.")
		return nil
	}
	var buf strings.Builder
	buf.WriteString("The following privileged commands are available:\r\n")
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

// cmdToggle — show the player's preference-flag grid.
// Source: act.informative.c do_toggle() (the sprintf at lines 2513-2579). It is
// purely informational — it ignores its argument and always prints the grid.
// Individual settings are changed via their own commands (brief, compact,
// notell, …, see commands.go's "Preference toggles" block), matching
// src/interpreter.c where each toggle is its own top-level command, not a
// "toggle <name>" dispatcher. The grid labels, column spacing, ONOFF/YESNO
// choice, and flag inversions (!PRF_*) are copied verbatim from C — do not
// reflow or relabel.
func cmdToggle(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	p := s.player
	g := p.Flags
	bit := func(b int) bool { return g&(1<<b) != 0 }

	// Wimp level: "OFF" when 0, else a 3-wide left-justified number (C buf2).
	wimp := "OFF"
	if p.WimpLevel != 0 {
		wimp = fmt.Sprintf("%-3d", p.WimpLevel)
	}

	// Color level: CAP(ctypes[COLOR_LEV(ch)]) — capitalize the lowercase name.
	color := colorLevelStr(g)
	if color != "" {
		color = strings.ToUpper(color[:1]) + color[1:]
	}

	// NOTE(autoexit): C reads PRF_AUTOEXIT here; Go's "auto" command toggles the
	// dedicated p.AutoExit bool (the PrfAutoexit bit is a separate slot), so the
	// grid reads p.AutoExit to stay consistent with what "auto" actually flips.
	// A fresh mortal has both ON, matching C's do_start default (class.c:589).
	grid := fmt.Sprintf(
		"Hit Pnt Display: %-3s         Brief Mode: %-3s     Summon Protect: %-3s\r\n"+
			"   Move Display: %-3s       Compact Mode: %-3s           On Quest: %-3s\r\n"+
			"   Mana Display: %-3s             NoTell: %-3s       Repeat Comm.: %-3s\r\n"+
			" Auto Show Exit: %-3s          Auto Loot: %-3s          Auto Gold: %-3s\r\n"+
			"     Auto Split: %-3s               Deaf: %-3s         Wimp Level: %-3s\r\n"+
			" Gossip Channel: %-3s    Auction Channel: %-3s      Grats Channel: %-3s\r\n"+
			"  Dsp Tank Stat: %-3s    Dsp Fightg Stat: %-3s        Color Level: %s\r\n"+
			" Newbie Channel: %-3s    Clan tells: %-3s     Broadcasts: %-3s",
		onOff(bit(game.PrfDisphp)), onOff(bit(game.PrfBrief)), onOff(!bit(game.PrfSummonable)),
		onOff(bit(game.PrfDispmove)), onOff(bit(game.PrfCompact)), yesNo(bit(game.PrfQuest)),
		onOff(bit(game.PrfDispmmana)), onOff(bit(game.PrfNotell)), yesNo(!bit(game.PrfNoRepeat)),
		onOff(p.AutoExit), onOff(bit(game.PrfAutoLoot)), onOff(bit(game.PrfAutoGold)),
		onOff(bit(game.PrfAutoSplit)), yesNo(bit(game.PrfDeaf)), wimp,
		onOff(!bit(game.PrfNoGossip)), onOff(!bit(game.PrfNoAuctions)), onOff(!bit(game.PrfNoGratz)),
		onOff(bit(game.PrfDispTank)), onOff(bit(game.PrfDispTarget)), color,
		onOff(!bit(game.PrfNoNewbie)), onOff(!bit(game.PrfNoCTell)), onOff(!bit(game.PrfNoBroad)),
	)
	// C finishes with sprintf(buf, "%s\r\n", buf) — a single trailing CRLF.
	s.Send(grid + "\r\n")
	return nil
}

// onOff mirrors C's ONOFF macro (utils.h:159): true → "ON", else "OFF".
func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// yesNo mirrors C's YESNO macro (utils.h:158): true → "YES", else "NO".
func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
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
