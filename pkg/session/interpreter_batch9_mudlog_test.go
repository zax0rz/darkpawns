package session

import (
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1371 batch 9 — the six interpreter.c denial / bad-password / self-delete
// rows, all file TRUE and all NRM at LVL_GOD except the bad-password BRF.
//
// C (src/interpreter.c) → Go producer:
//
//	:1826  "Request for new char %s denied from [%s] (siteban)"      NRM / LVL_GOD
//	       -> entry_restrictions.go:60, refuseNewSiteBan
//	:1838  "Request for new char %s denied from [%s] (wizlock)"      NRM / LVL_GOD
//	       -> entry_restrictions.go:16, refuseNewWizlock
//	:1879  "Bad PW: %s [%s]"                                         BRF / LVL_GOD
//	       -> session_login.go:212, handleLogin's password arm
//	:1903  "Connection attempt for %s denied from %s"                NRM / LVL_GOD
//	       -> entry_restrictions.go:71, refuseReturningSiteBan
//	:1912  "Request for login denied for %s [%s] (wizlock)"          NRM / LVL_GOD
//	       -> entry_restrictions.go:28, refuseReturningWizlock
//	:2346  "%s (lev %d) has self-deleted."                           NRM / LVL_GOD
//	       -> menu.go:389, confirmDelete
//
// The near-identical "Connection attempt denied from [%s]" text in
// pkg/telnet/listener.go:244 and pkg/session/manager.go:1265 is NOT this row: it
// mirrors src/comm.c:1573 and src/ident.c:234, different C sites.
//
// Six producers share one shape (NRM except :1879, fixed LVL_GOD, file TRUE), so
// the discriminators are: a LOG2-only observer receives and a LOG1-only one does
// not (for BRF the reverse), and the level is silent at LVL_GOD-1 and receiving
// at LVL_GOD. Assertions are per producer on its exact bracketed bytes.

func refNewSiteBanPayload(name, host string) string {
	return "Request for new char " + name + " denied from [" + host + "] (siteban)"
}

func refNewWizlockPayload(name, host string) string {
	return "Request for new char " + name + " denied from [" + host + "] (wizlock)"
}

func refLoginWizlockPayload(name, host string) string {
	return "Request for login denied for " + name + " [" + host + "] (wizlock)"
}

func refConnDeniedPayload(name, host string) string {
	return "Connection attempt for " + name + " denied from " + host
}
func badPasswordPayload(name, host string) string { return "Bad PW: " + name + " [" + host + "]" }
func selfDeletePayload(name string, level int) string {
	return name + " (lev " + strconv.Itoa(level) + ") has self-deleted."
}

// TestEntryRestrictionMudlogProducers drives the four refusal producers.
func TestEntryRestrictionMudlogProducers(t *testing.T) {
	cases := []struct {
		name    string
		drive   func(m *Manager, s *Session) bool
		payload func(host string) string
	}{
		{
			"siteban_1826",
			func(m *Manager, s *Session) bool {
				s.banLevel = game.BanNew
				return s.refuseNewSiteBan()
			},
			func(host string) string { return refNewSiteBanPayload("Newbie", host) },
		},
		{
			"wizlock_new_1838",
			func(m *Manager, s *Session) bool {
				m.wizlockLevel = 31
				return s.refuseNewWizlock()
			},
			func(host string) string { return refNewWizlockPayload("Newbie", host) },
		},
		{
			"login_wizlock_1912",
			func(m *Manager, s *Session) bool {
				m.wizlockLevel = 31
				return s.refuseReturningWizlock("Returner", 1)
			},
			func(host string) string { return refLoginWizlockPayload("Returner", host) },
		},
		{
			"conn_denied_1903",
			func(m *Manager, s *Session) bool {
				s.banLevel = game.BanSelect
				return s.refuseReturningSiteBan("Returner", nil)
			},
			func(host string) string { return refConnDeniedPayload("Returner", host) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			rig, file := newI8Rig(t, m)
			s := makeCharSession(t, m)
			s.charName = "Newbie"
			if !tc.drive(m, s) {
				t.Fatal("the refusal helper did not refuse; producer not reached")
			}
			payload := tc.payload(s.MudHost())
			if got := strings.Count(file.String(), payload); got != 1 {
				t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
			}
			if !i8Saw(t, rig.log2, payload) {
				t.Fatalf("LOG2-only observer did not receive %q", bracketOf(payload))
			}
			if i8Saw(t, rig.log1, payload) {
				t.Fatalf("LOG1-only observer received %q: type <= BRF", bracketOf(payload))
			}
			if i8Saw(t, rig.underGod, payload) {
				t.Fatalf("level-%d observer received %q", game.LVL_GOD-1, bracketOf(payload))
			}
			if !i8Saw(t, rig.god, payload) {
				t.Fatalf("level-%d observer did not receive %q", game.LVL_GOD, bracketOf(payload))
			}
		})
	}
}

// TestBadPasswordMudlog pins the :1879 producer (BRF at LVL_GOD).
func TestBadPasswordMudlog(t *testing.T) {
	m, name := connectFixture(t)
	rig, file := newI8Rig(t, m)
	s := makeCharSession(t, m)
	if err, panicked := callHandleLogin(s, loginMsg(name, "wrong-password")); panicked || err != nil {
		t.Fatalf("callHandleLogin = (%v, panicked=%v)", err, panicked)
	}
	payload := badPasswordPayload(name, s.MudHost())
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	if !i8Saw(t, rig.log1, payload) {
		t.Fatalf("LOG1-only observer did not receive %q", bracketOf(payload))
	}
	if i8Saw(t, rig.none, payload) {
		t.Fatalf("no-flag observer received %q: type would be OFF", bracketOf(payload))
	}
	if i8Saw(t, rig.underGod, payload) {
		t.Fatalf("level-%d observer received %q", game.LVL_GOD-1, bracketOf(payload))
	}
	if !i8Saw(t, rig.god, payload) {
		t.Fatalf("level-%d observer did not receive %q", game.LVL_GOD, bracketOf(payload))
	}
}

// TestSelfDeleteMudlog pins the :2346 producer.
func TestSelfDeleteMudlog(t *testing.T) {
	m := makeTestManager(t)
	rig, file := newI8Rig(t, m)
	s := makeCharSession(t, m)
	s.charName = "Selfdel"
	s.menuActive = true
	if err := s.confirmDelete("yes"); err != nil {
		t.Fatalf("confirmDelete = %v", err)
	}
	payload := selfDeletePayload("Selfdel", 0)
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	if !i8Saw(t, rig.log2, payload) {
		t.Fatalf("LOG2-only observer did not receive %q", bracketOf(payload))
	}
	if i8Saw(t, rig.log1, payload) {
		t.Fatalf("LOG1-only observer received %q: type <= BRF", bracketOf(payload))
	}
	if i8Saw(t, rig.underGod, payload) {
		t.Fatalf("level-%d observer received %q", game.LVL_GOD-1, bracketOf(payload))
	}
	if !i8Saw(t, rig.god, payload) {
		t.Fatalf("level-%d observer did not receive %q", game.LVL_GOD, bracketOf(payload))
	}
}
