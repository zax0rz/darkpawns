package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdPractice ports src/act.other.c do_practice(): with no argument it lists the
// skill catalog; with any argument (when NOT standing on a guildmaster — whose
// guild spec proc, pkg/game specGuild, intercepts `practice` first) it directs
// the player to their guild. The catalog rendering + learning both live in
// pkg/game (canonical layer) and are shared with the guild proc.
func cmdPractice(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) > 0 {
		s.Send("You can only practice skills in your guild.\r\n")
		return nil
	}
	s.Send(game.RenderSkillList(s.player))
	return nil
}
