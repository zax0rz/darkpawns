package session

// ---------------------------------------------------------------------------
// Shop commands
// ---------------------------------------------------------------------------

func cmdNotBuy(s *Session, args []string) error {
	s.Send("Not buying.\r\n")
	return nil
}
