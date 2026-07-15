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

// cmdAutoExit toggles automatic exit display after movement (Player.AutoExit,
// consulted by look.go). Not in src/interpreter.c's command table — do_gen_tog
// has an SCMD_AUTOEXIT case but nothing wires a command name to it there — so
// this is a Go-only convenience; its message text matches do_gen_tog's anyway.
func cmdAutoExit(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.player.SetAutoExit(!s.player.GetAutoExit())
	if s.player.GetAutoExit() {
		s.Send("Autoexits enabled.")
	} else {
		s.Send("Autoexits disabled.")
	}
	return nil
}

// cmdTitle sets the player's title, matching C do_title() (src/act.other.c:595-620).
func cmdTitle(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	if len(args) == 0 {
		s.Send("Set your title to what?\r\n")
		return nil
	}

	// Recover the full argument, then apply C's preprocessing.
	title := strings.Join(args, " ")
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "$$", "$")
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
