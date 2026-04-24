package session

// ---------------------------------------------------------------------------
// Fight commands
// ---------------------------------------------------------------------------

func cmdParry(s *Session, args []string) error {
	// Fight.parry would be toggled here if the field existed
	s.Send("Parry toggled.\r\n")
	return nil
}
