//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

// ---------------------------------------------------------------------------
// Fight commands
// ---------------------------------------------------------------------------

func cmdParry(s *Session, args []string) error {
	// Fight.parry would be toggled here if the field existed
	s.Send("Parry toggled.\r\n")
	return nil
}
