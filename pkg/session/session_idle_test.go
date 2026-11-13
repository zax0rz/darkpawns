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

func TestCheckIdlePasswordsMenuSession(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "MenuIdler", 1001, true)
	closed := false
	s.SetCloseFunc(func() { closed = true })
	s.showMainMenu()
	drainSendChannel(t, s)

	m.mu.Lock()
	m.sessions["menuidler"] = s
	m.mu.Unlock()

	// First 15s tic sets idleTicsSet without disconnecting.
	m.CheckIdlePasswords()
	if closed || !s.idleTicsSet {
		t.Fatalf("after 1st tic: closed=%v idleTicsSet=%v, want false/true", closed, s.idleTicsSet)
	}

	// Second 15s tic emits "\r\nTimed out... goodbye.\r\n", closes, and unregisters.
	m.CheckIdlePasswords()
	if !closed {
		t.Fatal("after 2nd tic: expected session to close")
	}
	if !s.SendClosed() {
		t.Fatal("after 2nd tic: expected send channel to close")
	}
	if _, ok := m.sessions["menuidler"]; ok {
		t.Fatal("after 2nd tic: expected session to be removed from manager")
	}
	var got string
	for msg := range s.send {
		if f, ok := RenderTerminalFrame(msg); ok {
			got += f.Text
		}
	}
	if got != "\r\nTimed out... goodbye.\r\n" {
		t.Fatalf("timeout bytes = %q, want %q", got, "\r\nTimed out... goodbye.\r\n")
	}
}
