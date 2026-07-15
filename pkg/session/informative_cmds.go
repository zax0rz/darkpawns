package session

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

func cmdTitle(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Set your title to what?")
		return nil
	}
	s.player.Title = args[0]
	s.Send("Title set.")
	return nil
}

func cmdDescribe(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Describe yourself with what?")
		return nil
	}
	s.player.Description = args[0]
	s.Send("Description set.")
	return nil
}
