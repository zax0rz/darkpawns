package session

import (
	"fmt"
	"strings"
)

// cmdPassword handles password changes.
// Usage: password <old> <new>
// TODO: Wire to DB password storage when implemented.
func cmdPassword(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Usage: password <old> <new>")
		return nil
	}

	newPass := args[1]

	if args[0] == newPass {
		s.Send("That's the same as your old password!")
		return nil
	}

	if len(newPass) < 4 {
		s.Send("Password must be at least 4 characters.")
		return nil
	}

	// TODO: Wire to DB password storage when implemented.
	s.Send("Password change is not yet implemented. Coming soon.")
	return nil
}

// cmdPrompt sets or toggles the player's prompt display.
// Usage: prompt              — toggle prompt on/off
//        prompt <string>     — set custom prompt (supports %h/%m/%v/%H/%M/%V)
//        prompt all          — show all stats (hp/mana/move)
func cmdPrompt(s *Session, args []string) error {
	arg := strings.Join(args, " ")

	if arg == "" {
		s.player.PromptOn = !s.player.PromptOn
		if s.player.PromptOn {
			s.Send("Prompt now on.")
		} else {
			s.Send("Prompt now off.")
		}
		return nil
	}

	if strings.EqualFold(arg, "all") {
		s.player.PromptStr = "%h/%H hp %m/%M mana %v/%V mv > "
		s.player.PromptOn = true
		s.Send("Prompt set to show all.")
		return nil
	}

	if strings.EqualFold(arg, "off") {
		s.player.PromptOn = false
		s.Send("Prompt now off.")
		return nil
	}

	if strings.EqualFold(arg, "on") {
		s.player.PromptOn = true
		s.Send("Prompt now on.")
		return nil
	}

	s.player.PromptStr = arg
	s.player.PromptOn = true
	s.Send(fmt.Sprintf("Prompt set to: %s", arg))
	return nil
}
