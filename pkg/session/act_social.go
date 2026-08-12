package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Social commands (ported from act.other.c / act.social.c)
// ---------------------------------------------------------------------------

// cmdInsult hurls a randomized insult at a target in the room.
// Source: src/act.social.c do_insult() — game.DoInsult implements the logic.
func cmdInsult(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.manager.world.ExecInsult(s.player, strings.Join(args, " "))
	return nil
}

// cmdDream prints the daydream/sleep-dream flavor message.
// Source: src/act.social.c do_dream() — game.DoDream implements the logic.
func cmdDream(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.manager.world.ExecDream(s.player)
	return nil
}

// cmdAlias manages a player's command aliases.
func cmdAlias(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	player := s.player

	if len(args) == 0 {
		aliases := player.Aliases
		if len(aliases) == 0 {
			s.Send("Currently defined aliases:\r\n None.")
			return nil
		}
		for _, a := range aliases {
			s.Send(fmt.Sprintf("%s -> %s", a.Alias, a.Replacement))
		}
		return nil
	}

	if len(args) == 1 {
		// Delete alias
		aliasName := strings.ToLower(args[0])
		for i, a := range player.Aliases {
			if strings.EqualFold(a.Alias, aliasName) {
				player.Aliases = append(player.Aliases[:i], player.Aliases[i+1:]...)
				if err := game.WriteAliases(player.Name, player.Aliases); err != nil {
					s.Send("Error saving aliases.")
					return nil
				}
				s.Send("Alias deleted.")
				return nil
			}
		}
		s.Send(fmt.Sprintf("Alias '%s' not found.", args[0]))
		return nil
	}

	// Set alias: args[0] = from, args[1] = to
	aliasName := strings.ToLower(args[0])
	replacement := " " + strings.Join(args[1:], " ") // skip initial space

	// Check for duplicate
	for i := range player.Aliases {
		if strings.EqualFold(player.Aliases[i].Alias, aliasName) {
			player.Aliases[i].Replacement = replacement
			if err := game.WriteAliases(player.Name, player.Aliases); err != nil {
				s.Send("Error saving aliases.")
				return nil
			}
			s.Send("Alias updated.")
			return nil
		}
	}

	// Add new alias
	player.Aliases = append(player.Aliases, game.PlayerAlias{
		Alias:       aliasName,
		Replacement: replacement,
		Type:        0,
	})
	if err := game.WriteAliases(player.Name, player.Aliases); err != nil {
		s.Send("Error saving aliases.")
		return nil
	}
	s.Send("Alias added.")
	return nil
}
