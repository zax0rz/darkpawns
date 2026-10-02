package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
)

// parseEntryName follows src/interpreter.c:1721,1505-1520,1743-1759.
// C nanny skips leading spaces before checking for the empty-name close.
func parseEntryName(raw string) (name string, valid bool) {
	name = strings.TrimLeft(raw, " \t\n\r\v\f")
	if len(name) < 2 || len(name) > 20 {
		return name, false
	}
	for i := 0; i < len(name); i++ {
		letter := name[i]
		if (letter < 'a' || letter > 'z') && (letter < 'A' || letter > 'Z') {
			return name, false
		}
	}
	switch strings.ToLower(name) {
	// C src/interpreter.c:853-876, fill_word/reserved_word exact matches.
	case "in", "from", "with", "the", "on", "at", "to", "a", "an", "self", "me", "all", "room", "someone", "something":
		return name, false
	}
	return name, true
}

// claimEntryName mirrors src/ban.c:257-291 before saved lookup. Entry names
// remain descriptor-owned through the menu; a playing descriptor can reconnect.
// Keep its lock separate from m.mu: Register may close an old descriptor.
func (s *Session) claimEntryName(name string) bool {
	m := s.manager
	m.mu.RLock()
	active := make([]*Session, 0, len(m.sessions))
	for _, candidate := range m.sessions {
		active = append(active, candidate)
	}
	m.mu.RUnlock()

	m.entryNameMu.Lock()
	defer m.entryNameMu.Unlock()
	for owner, held := range m.entryNames {
		if owner != s && strings.EqualFold(held, name) {
			return false
		}
	}
	allowed := false
	for _, candidate := range active {
		if candidate == s || candidate.player == nil || !candidate.hasTransport() || candidate.SendClosed() || !strings.EqualFold(candidate.player.Name, name) {
			continue
		}
		if candidate.charCreating || candidate.menuActive || candidate.inOLCEditorState() {
			return false
		}
		allowed = true
	}
	// A playing descriptor wins before invalid_list in C Valid_Name.
	if !allowed && !game.ValidNameNoActive(name) {
		return false
	}
	if m.entryNames == nil {
		m.entryNames = make(map[*Session]string)
	}
	m.entryNames[s] = name
	return true
}

func (s *Session) releaseEntryName() {
	if s.manager == nil {
		return
	}
	s.manager.entryNameMu.Lock()
	delete(s.manager.entryNames, s)
	s.manager.entryNameMu.Unlock()
}

// auditSafeName returns the typed login name for audit records only when it
// passes the entry gate's parse and staff-reserved checks. Raw prompt input
// is frequently a mistyped password, and rejected input must never reach the
// audit file.
func auditSafeName(raw string) string {
	name, valid := parseEntryName(raw)
	if !valid || validation.IsReservedPlayerName(name) {
		return ""
	}
	return name
}

func (s *Session) acceptEntryName(raw string) (string, bool) {
	name, valid := parseEntryName(raw)
	if name == "" {
		s.CloseSend()
		return name, false
	}
	if !valid || validation.IsReservedPlayerName(name) || !s.claimEntryName(name) {
		s.restartNameEntry()
		return name, false
	}
	return name, true
}

// reserveOfflineRename implements DP-1381. The entry-name lock serializes
// the availability check and file edit against claimEntryName. Retained keys
// catch live wizard renames whose stored and displayed names differ. Guest
// numeric IDs are process-local and cannot identify an offline store record.
func (m *Manager) reserveOfflineRename(oldName, newName string) (release func(), allowed bool) {
	m.entryNameMu.Lock()
	release = m.entryNameMu.Unlock
	held := func(name string) bool { return strings.EqualFold(name, oldName) || strings.EqualFold(name, newName) }
	for _, name := range m.entryNames {
		if held(name) {
			release()
			return nil, false
		}
	}
	m.mu.RLock()
	blocked := false
	for key, s := range m.sessions {
		if held(key) || held(s.playerName) || (s.player != nil && held(s.player.GetName())) ||
			(s.switchedOriginal != nil && held(s.switchedOriginal.GetName())) {
			blocked = true
			break
		}
	}
	m.mu.RUnlock()
	// Preserve retained/linkdead bodies too; renaming their stored identity
	// would otherwise invalidate the next reconnect just like a live socket.
	if !blocked && m.world != nil {
		// Live edits leave the world's original entry key intact. Check the
		// stored spelling too, even during the transient teardown interval.
		if _, present := m.world.GetPlayer(oldName); present {
			blocked = true
		}
		for _, p := range m.world.GetAllPlayers() {
			if held(p.GetName()) {
				blocked = true
				break
			}
		}
	}
	if blocked {
		release()
		return nil, false
	}
	return release, true
}
