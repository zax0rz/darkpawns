package session

import "testing"

func TestManager_SessionCount(t *testing.T) {
	m := makeTestManager(t)
	if got := m.SessionCount(); got != 0 {
		t.Errorf("SessionCount on empty manager = %d, want 0", got)
	}

	s1 := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s1
	m.mu.Unlock()

	if got := m.SessionCount(); got != 1 {
		t.Errorf("SessionCount after 1 session = %d, want 1", got)
	}

	s2 := makeTestSession(t, m, "Bob", 1001, true)
	m.mu.Lock()
	m.sessions["bob"] = s2
	m.mu.Unlock()

	if got := m.SessionCount(); got != 2 {
		t.Errorf("SessionCount after 2 sessions = %d, want 2", got)
	}
}

func TestManager_CountSessions(t *testing.T) {
	m := makeTestManager(t)

	connected, playing := m.CountSessions()
	if connected != 0 || playing != 0 {
		t.Errorf("empty manager: connected=%d playing=%d, want 0,0", connected, playing)
	}

	s1 := makeTestSession(t, m, "Alice", 1001, true)     // authenticated + has player
	s2 := makeTestSession(t, m, "Bob", 1001, true)       // authenticated + has player
	s3 := makeTestSession(t, m, "Guest", 1001, false)    // not authenticated
	s4 := makeTestSession(t, m, "NoPlayer", 1001, false) // not authenticated

	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.sessions["guest"] = s3
	m.sessions["noplayer"] = s4
	m.mu.Unlock()

	connected, playing = m.CountSessions()
	if connected != 4 {
		t.Errorf("connected = %d, want 4", connected)
	}
	if playing != 2 {
		t.Errorf("playing = %d, want 2 (Alice and Bob authenticated with players)", playing)
	}
}

func TestManager_IsWizlocked_Default(t *testing.T) {
	m := makeTestManager(t)
	if m.IsWizlocked() {
		t.Error("IsWizlocked should default to false")
	}
}

func TestManager_SetWizlock(t *testing.T) {
	m := makeTestManager(t)

	m.SetWizlock(true)
	if !m.IsWizlocked() {
		t.Error("IsWizlocked should be true after SetWizlock(true)")
	}

	m.SetWizlock(false)
	if m.IsWizlocked() {
		t.Error("IsWizlocked should be false after SetWizlock(false)")
	}
}

func TestManager_HasDB(t *testing.T) {
	m := makeTestManager(t)
	// makeTestManager uses newTestManager with nil DB, so HasDB should be false
	if m.HasDB() {
		t.Error("HasDB should be false when no database is configured")
	}
}

func TestManager_GetShopManager(t *testing.T) {
	m := makeTestManager(t)
	sm := m.GetShopManager()
	if sm == nil {
		t.Error("GetShopManager should return a non-nil ShopManager")
	}
}

func TestManager_GetBanManager(t *testing.T) {
	m := makeTestManager(t)
	bm := m.GetBanManager()
	if bm == nil {
		t.Error("GetBanManager should return a non-nil BanManager")
	}
}

func TestManager_GetCombatEngine(t *testing.T) {
	m := makeTestManager(t)
	ce := m.GetCombatEngine()
	if ce == nil {
		t.Error("GetCombatEngine should return a non-nil CombatEngine")
	}
}
