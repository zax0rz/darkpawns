package session

import "fmt"

// ---------------------------------------------------------------------------
// Informative command stubs (act.informative.c)
// These are referenced in commands.go but have partial implementations
// elsewhere that may not compile. Provide minimal stubs for now.
// ---------------------------------------------------------------------------

func cmdConsider(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Consider killing who?")
		return nil
	}
	s.Send(fmt.Sprintf("You consider %s.", args[0]))
	return nil
}

func cmdExamine(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Examine what?")
		return nil
	}
	s.Send(fmt.Sprintf("You examine %s.", args[0]))
	return nil
}

func cmdTime(s *Session, args []string) error {
	s.Send("The current time is unknown.")
	return nil
}

func cmdWeather(s *Session, args []string) error {
	s.Send("The weather is fine.")
	return nil
}

func cmdAffects(s *Session, args []string) error {
	s.Send("You have no active affects.")
	return nil
}

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
