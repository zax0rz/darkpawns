package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Social commands (ported from act.other.c / act.social.c)
// ---------------------------------------------------------------------------

// cmdDream handles the 'dream' command — shows dream state while sleeping.
// Calls game.ProcessDream for dream message generation.
func cmdDream(s *Session, args []string) error {
	_ = args

	pos := s.player.GetPosition()
	if pos != combat.PosSleeping {
		s.Send("You are awake.")
		return nil
	}

	lastDeath := s.player.GetLastDeath()
	result := game.ProcessDream(s.player, lastDeath)
	if result == nil {
		return nil
	}

	if result.PlayerMessage != "" {
		s.Send(result.PlayerMessage)
	}
	if result.RoomMessage != "" {
		broadcastToRoomText(s, s.player.GetRoomVNum(), result.RoomMessage)
	}
	if result.WakeUp {
		s.player.SetPosition(combat.PosStanding)
	}

	return nil
}

// cmdAlias manages a player's command aliases.
func cmdAlias(s *Session, args []string) error {
	player := s.player

	if len(args) == 0 {
		aliases := player.Aliases
		if len(aliases) == 0 {
			s.Send("No aliases defined. Usage: alias <from> <to>  or  alias <from> (to delete)")
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
		aliases := player.Aliases
		for i, a := range aliases {
			if strings.EqualFold(a.Alias, aliasName) {
				player.Aliases = append(aliases[:i], aliases[i+1:]...)
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
	for _, a := range player.Aliases {
		if strings.EqualFold(a.Alias, aliasName) {
			a.Replacement = replacement
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

// cmdBan bans a site. Admin only (LVL_IMMORT+).
func cmdBan(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Usage: ban <site> <type>  (types: new, select, all)")
		return nil
	}

	site := args[0]
	flag := args[1]

	if err := game.AddBan(site, s.player.Name, flag); err != nil {
		s.Send(fmt.Sprintf("Ban failed: %s", err))
		return nil
	}
	s.Send(fmt.Sprintf("%s has been banned (%s).", site, game.BanTypeName(game.IsBanned(site))))
	return nil
}

// cmdUnban unbans a site. Admin only (LVL_IMMORT+).
func cmdUnban(s *Session, args []string) error {
	if len(args) < 1 {
		s.Send("Usage: unban <site>")
		return nil
	}

	site := args[0]
	if err := game.RemoveBan(site); err != nil {
		s.Send(fmt.Sprintf("Unban failed: %s", err))
		return nil
	}
	s.Send(fmt.Sprintf("%s has been unbanned.", site))
	return nil
}

// cmdInsult insults a target in the room.
// Find target in room, send insult emote.
func cmdInsult(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Insult whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	roomVNum := s.player.GetRoomVNum()

	// Look for target in the room (players first, then mobs)
	found := false

	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.Contains(strings.ToLower(p.Name), targetName) {
			s.Send(fmt.Sprintf("You insult %s.", p.Name))
			broadcastToRoomText(s, roomVNum,
				fmt.Sprintf("%s insults %s.", s.player.Name, p.Name))
			found = true
			break
		}
	}

	if !found {
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, m := range mobs {
			if strings.Contains(strings.ToLower(m.GetShortDesc()), targetName) {
				s.Send(fmt.Sprintf("You insult %s.", m.GetShortDesc()))
				broadcastToRoomText(s, roomVNum,
					fmt.Sprintf("%s insults %s.", s.player.Name, m.GetShortDesc()))
				found = true
				break
			}
		}
	}

	if !found {
		s.Send("They aren't here.")
	}

	return nil
}
