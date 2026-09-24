package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Informative command stubs (act.informative.c)
// These are referenced in commands.go but have partial implementations
// elsewhere that may not compile. Provide minimal stubs for now.
// ---------------------------------------------------------------------------

// cmdTitle sets the player's title, matching C do_title() (src/act.other.c:595-620).
func cmdTitle(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}

	// Recover the full argument, then apply C's preprocessing. The live C
	// command path preserves doubled dollars on the direct send_to_char title
	// acknowledgement, so do not collapse them here.
	title := strings.Join(args, " ")
	title = strings.TrimSpace(title)
	title = game.DeleteANSIControls(title)

	switch {
	case s.player.IsNPC():
		s.Send("Your title is fine... go away.\r\n")
	case s.player.GetFlags()&(1<<uint(game.PlrNotitle)) != 0:
		s.Send("You can't title yourself -- you shouldn't have abused it!\r\n")
	case strings.Contains(title, "(") || strings.Contains(title, ")"):
		s.Send("Titles can't contain the ( or ) characters.\r\n")
	case len(title) > game.MAX_TITLE_LENGTH:
		s.Send(fmt.Sprintf("Sorry, titles can't be longer than %d characters.\r\n", game.MAX_TITLE_LENGTH))
	default:
		game.SetTitle(s.player, title)
		s.Send(fmt.Sprintf("Okay, you're now %s %s.\r\n", s.player.Name, s.player.Title))
	}
	return nil
}
