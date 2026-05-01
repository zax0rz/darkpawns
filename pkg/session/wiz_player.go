package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdHeal(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Heal whom?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	targetSess.player.Health = targetSess.player.MaxHealth
	targetSess.player.Mana = targetSess.player.MaxMana
	targetSess.player.Move = targetSess.player.MaxMove
	slog.Warn("wizard heal", "by", s.player.Name, "target", targetSess.player.Name)
	s.Send(fmt.Sprintf("You heal %s.", targetSess.player.Name))
	targetSess.Send(fmt.Sprintf("%s has healed you!", s.player.Name))
	return nil
}

// ---------------------------------------------------------------------------
// restore — fully restore target (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdRestore(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	return cmdHeal(s, args)
}

// ---------------------------------------------------------------------------
// set — set a player field (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdSet(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 3 {
		s.Send("Usage: set <player> <field> <value>")
		return nil
	}
	targetName := args[0]
	field := args[1]
	value := strings.Join(args[2:], " ")
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	field = strings.ToLower(field)

	// Validate numeric value before assignment
	val, err := strconv.Atoi(value)
	if err != nil {
		s.Send("Invalid numeric value.")
		return nil
	}

	// Validate field bounds
	switch field {
	case "str", "sta", "dex", "int", "wil", "cha":
		val = clamp(val, 3, 25)
	case "level":
		val = clamp(val, 0, 61)
	case "hp", "mana", "move":
		if val > 10000 && targetSess.player.Level < 60 {
			return fmt.Errorf("cannot set %s above 10000 for non-immortals", field)
		}
	}

	switch field {
	case "level":
		targetSess.player.Level = val
		s.Send(fmt.Sprintf("Level set to %d.", val))
	case "gold":
		targetSess.player.Gold = val
		s.Send(fmt.Sprintf("Gold set to %d.", val))
	case "alignment":
		targetSess.player.Alignment = val
		s.Send(fmt.Sprintf("Alignment set to %d.", val))
	case "str":
		targetSess.player.Stats.Str = val
		targetSess.player.Strength = val
		s.Send(fmt.Sprintf("Strength set to %d.", val))
	case "sta":
		targetSess.player.Stats.Con = val
		s.Send(fmt.Sprintf("Constitution set to %d.", val))
	case "dex":
		targetSess.player.Stats.Dex = val
		s.Send(fmt.Sprintf("Dexterity set to %d.", val))
	case "int":
		targetSess.player.Stats.Int = val
		s.Send(fmt.Sprintf("Intelligence set to %d.", val))
	case "wil":
		targetSess.player.Stats.Wis = val
		s.Send(fmt.Sprintf("Wisdom set to %d.", val))
	case "cha":
		targetSess.player.Stats.Cha = val
		s.Send(fmt.Sprintf("Charisma set to %d.", val))
	case "hp":
		targetSess.player.MaxHealth = val
		targetSess.player.Health = val
		s.Send(fmt.Sprintf("Hit points set to %d.", val))
	case "mana":
		targetSess.player.MaxMana = val
		targetSess.player.Mana = val
		s.Send(fmt.Sprintf("Mana set to %d.", val))
	case "move":
		targetSess.player.MaxMove = val
		targetSess.player.Move = val
		s.Send(fmt.Sprintf("Move points set to %d.", val))
	default:
		s.Send("Unknown field. Try: level, gold, alignment, str, sta, dex, int, wil, cha, hp, mana, move.")
	}
	slog.Warn("wizard set", "by", s.player.Name, "target", targetName, "field", field, "value", value)
	return nil
}

// clamp restricts v to the [min, max] range.
func cmdSwitch(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Switch into whom?\r\n")
		return nil
	}
	targetName := strings.ToLower(args[0])
	roomVNum := s.player.GetRoom()

	// Look for a mob in the room
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			// Store original player reference for return
			s.switchedOriginal = s.player
			s.switchedMob = mob
			s.isSwitched = true
			slog.Info("(GC) switch", "who", s.player.Name, "into", mob.GetShortDesc())
			s.Send(fmt.Sprintf("You switch into %s.\r\n", mob.GetShortDesc()))
			return nil
		}
	}

	// Look for a player in the room
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.ToLower(p.GetName()) == targetName {
			if p.Level >= s.player.Level {
				s.Send("Fuuuuuuuuu!\r\n")
				return nil
			}
			s.switchedOriginal = s.player
			s.switchedPlayer = p
			s.isSwitched = true
			slog.Info("(GC) switch", "who", s.player.Name, "into", p.GetName())
			s.Send(fmt.Sprintf("You switch into %s.\r\n", p.GetName()))
			return nil
		}
	}
	s.Send("No one here by that name.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// return — return to own body (LVL_IMMORT)
// ---------------------------------------------------------------------------
// cmdReturn returns the wizard to their own body after a switch.
// Expected behavior (from original C):
// - Detach the wizard's session from the switched character
// - Re-attach to the wizard's original character
func cmdReturn(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if !s.isSwitched || s.switchedOriginal == nil {
		s.Send("You aren't switched.\r\n")
		return nil
	}
	if s.switchedMob != nil {
		slog.Info("(GC) return", "who", s.player.Name, "from", s.switchedMob.GetShortDesc())
	} else if s.switchedPlayer != nil {
		slog.Info("(GC) return", "who", s.player.Name, "from", s.switchedPlayer.GetName())
	}
	s.isSwitched = false
	s.switchedMob = nil
	s.switchedPlayer = nil
	s.player = s.switchedOriginal
	s.switchedOriginal = nil
	s.Send("You return to your own body.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// invis — toggle invisibility (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdInvis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	// Toggle invisibility
	if s.player.Flags&game.PLR_INVISIBLE != 0 {
		s.player.Flags &^= game.PLR_INVISIBLE
		s.Send("You are now visible.")
	} else {
		s.player.Flags |= game.PLR_INVISIBLE
		s.Send("You are now invisible.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// vis — make invisible players visible (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdVis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Vis whom?")
		return nil
	}
	s.Send("Vis not yet implemented.")
	return nil
}

// ---------------------------------------------------------------------------
// gecho — broadcast to all players (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdAdvance(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Advance whom to what level?")
		return nil
	}
	targetName := args[0]
	newLevel, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("Invalid level.")
		return nil
	}
	if newLevel < 0 || newLevel > 60 {
		s.Send("Level must be between 0 and 60.")
		return nil
	}
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	oldLevel := target.player.Level
	target.player.Level = newLevel
	slog.Warn("player advanced", "target", target.player.Name, "old", oldLevel, "new", newLevel, "by", s.player.Name)
	s.Send(fmt.Sprintf("%s advanced from level %d to %d.", target.player.Name, oldLevel, newLevel))
	target.Send(fmt.Sprintf("You have been advanced from level %d to %d!", oldLevel, newLevel))
	return nil
}

// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// Re-reads world files from disk and replaces the in-memory world.
