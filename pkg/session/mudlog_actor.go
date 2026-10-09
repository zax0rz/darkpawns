package session

// mudlogActor is the acting body a command's producer names (R4): C logs
// GET_NAME(ch) and MAX(level, GET_INVIS_LEV(ch)) of the body the descriptor is
// switched into, not of the immortal behind it. A switched-into mob has no
// invis level — clear_char zeroes its player_specials (src/db.c:2976-2989) — so
// the immortal's own invisibility must not leak into the threshold.
//
// The manager lock is taken only for the read and released before the caller
// calls MudLog, whose delivery takes its own read locks.
func (s *Session) mudlogActor() (name string, invis int) {
	if s.manager != nil {
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
	}
	if s.isSwitched && s.switchedMob != nil {
		return s.switchedMob.GetName(), 0
	}
	if s.player == nil {
		return "", 0
	}
	return s.player.GetName(), s.player.GetInvisLevel()
}
