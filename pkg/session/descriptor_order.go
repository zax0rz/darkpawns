package session

import "sort"

// sortSessionsDescriptorOrder sorts a session slice into newest-connection-first
// order (descending connectionNumber, with connectedAt descending and
// playerName descending as deterministic tie-breaks).
func sortSessionsDescriptorOrder(sessions []*Session) {
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].connectionNumber != sessions[j].connectionNumber {
			return sessions[i].connectionNumber > sessions[j].connectionNumber
		}
		if !sessions[i].connectedAt.Equal(sessions[j].connectedAt) {
			return sessions[i].connectedAt.After(sessions[j].connectedAt)
		}
		return sessions[i].playerName > sessions[j].playerName
	})
}

// sessionsInDescriptorOrderLocked snapshots m.sessions in newest-connection-first
// order. Caller must hold m.mu (read or write).
func (m *Manager) sessionsInDescriptorOrderLocked() []*Session {
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		if s != nil {
			sessions = append(sessions, s)
		}
	}
	sortSessionsDescriptorOrder(sessions)
	return sessions
}

// sessionsInDescriptorOrder snapshots m.sessions in newest-connection-first order.
func (m *Manager) sessionsInDescriptorOrder() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionsInDescriptorOrderLocked()
}

// playingSessionsInDescriptorOrder returns all CON_PLAYING sessions in
// newest-connection-first order.
func (m *Manager) playingSessionsInDescriptorOrder() []*Session {
	all := m.sessionsInDescriptorOrder()
	playing := make([]*Session, 0, len(all))
	for _, s := range all {
		if s.isConPlaying() {
			playing = append(playing, s)
		}
	}
	return playing
}
