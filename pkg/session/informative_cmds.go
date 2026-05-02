package session

import "fmt"

// ---------------------------------------------------------------------------
// Informative command stubs (act.informative.c)
// These are referenced in commands.go but have partial implementations
// elsewhere that may not compile. Provide minimal stubs for now.
// ---------------------------------------------------------------------------

func cmdAutoExit(s *Session, args []string) error {
	s.Send("Auto-exit toggled.")
	return nil
}

func cmdTitle(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Set your title to what?")
		return nil
	}
	s.player.Title = fmt.Sprintf("%s", args[0])
	s.Send("Title set.")
	return nil
}

func cmdDescribe(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Describe yourself with what?")
		return nil
	}
	s.player.Description = fmt.Sprintf("%s", args[0])
	s.Send("Description set.")
	return nil
}

func cmdSpells(s *Session, args []string) error {
	s.Send("You know no spells.")
	return nil
}
