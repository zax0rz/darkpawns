package session

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func guestSecurityWorld(t *testing.T) *game.World {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: game.MortalStartRoom, Name: "The Adventurers Guild", Zone: 80}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// A guest login proves no credential, so it must not reset the IP's
// failed-password count: otherwise one free guest entry between bursts of
// wrong passwords keeps the H-15 lockout from ever engaging (VULN-006).
func TestGuestLoginDoesNotResetIPLockout(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long-please")
	m := newTestManager(t, guestSecurityWorld(t), nil)

	const ip = "198.51.100.77"
	for range 9 {
		m.loginAttempts.RecordFailure(ip)
	}
	if locked, _ := m.loginAttempts.IsLocked(ip); locked {
		t.Fatal("precondition: IP locked below the threshold")
	}

	s := m.NewSession()
	s.remoteIP = ip
	if err := s.handleLogin(json.RawMessage(`{"player_name":"guest","password":"","new_char":false}`)); err != nil {
		t.Fatalf("guest login: %v", err)
	}
	if !s.isGuest {
		t.Fatal("precondition: guest login did not complete")
	}

	m.loginAttempts.RecordFailure(ip)
	if locked, _ := m.loginAttempts.IsLocked(ip); !locked {
		t.Fatal("tenth failed password after a guest login did not lock the IP: the guest login reset the failure count")
	}
}

// The guest branch publishes its identity under the manager lock, so a reader
// that synchronizes on that lock (lookups, broadcasts) never races the login's
// writes or sees a half-published guest (VULN-010). Run under -race.
func TestGuestLoginPublishesIdentityUnderManagerLock(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long-please")
	m := newTestManager(t, guestSecurityWorld(t), nil)
	s := m.NewSession()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			m.mu.RLock()
			authenticated, guest, name := s.authenticated, s.isGuest, s.playerName
			m.mu.RUnlock()
			if authenticated && (!guest || name == "") {
				t.Errorf("observed a half-published guest: authenticated=%v isGuest=%v name=%q", authenticated, guest, name)
				return
			}
		}
	}()

	err := s.handleLogin(json.RawMessage(`{"player_name":"guest","password":"","new_char":false}`))
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatalf("guest login: %v", err)
	}
}
