package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// C stores d->host as the zero-padded dotted quad (src/comm.c:1536-1539).
func TestCConnectionHostPadsIPv4(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1":  "127.000.000.001",
		"192.0.2.10": "192.000.002.010",
		// Already-padded and non-IPv4 input is passed through: C only ever
		// stores the padded quad or a resolved name, and IPv6 has no C form.
		"127.000.000.001": "127.000.000.001",
		"client.example":  "client.example",
		"2001:db8::1":     "2001:db8::1",
		"":                "",
	}
	for in, want := range cases {
		if got := CConnectionHost(in); got != want {
			t.Errorf("CConnectionHost(%q) = %q, want %q", in, got, want)
		}
	}
}

// MudHost is C's d->host: a stored resolved name wins, otherwise the padded
// quad of the connection's address.
func TestMudHostPrefersResolvedName(t *testing.T) {
	s := entrySession(t, entryDatabase(t))
	s.request = nil
	s.SetRemoteIP("192.0.2.10")
	if got := s.MudHost(); got != "192.000.002.010" {
		t.Fatalf("MudHost() = %q, want the padded quad", got)
	}
	s.SetMudHost("client.example")
	if got := s.MudHost(); got != "client.example" {
		t.Fatalf("MudHost() = %q, want the resolved name", got)
	}
	// An empty value never clears the recorded host: SetMudHost ignores it so a
	// caller that has no resolved name cannot erase C's padded quad.
	s.SetMudHost("")
	if got := s.MudHost(); got != "client.example" {
		t.Fatalf("MudHost() after an empty set = %q, want the recorded name", got)
	}
}

// The nanny checks isbanned(d->host) only (src/interpreter.c:1822, :1896), so
// the retained entry identity is C's host string: a padded ban matches and the
// pre-fix raw-IP spelling does not.
func TestEntryBanLevelUsesMudHost(t *testing.T) {
	t.Run("padded ban matches", func(t *testing.T) {
		s := entrySession(t, entryDatabase(t))
		s.request = nil
		s.SetRemoteIP("192.0.2.10")
		s.SetBanHosts([]string{s.MudHost()})
		if err := s.manager.GetBanManager().AddBan("192.000.002", game.BanAll, "God"); err != nil {
			t.Fatal(err)
		}
		if got := s.entryBanLevel(); got != game.BanAll {
			t.Fatalf("entryBanLevel() = %d, want BanAll", got)
		}
	})
	t.Run("raw-IP ban does not match", func(t *testing.T) {
		s := entrySession(t, entryDatabase(t))
		s.request = nil
		s.SetRemoteIP("192.0.2.10")
		s.SetBanHosts([]string{s.MudHost()})
		if err := s.manager.GetBanManager().AddBan("192.0.2.10", game.BanAll, "God"); err != nil {
			t.Fatal(err)
		}
		if got := s.entryBanLevel(); got != game.BanNot {
			t.Fatalf("entryBanLevel() = %d, want BanNot for a raw-IP ban", got)
		}
	})
}

// Accept-time isbanned() covers d->host and both wildcard strings
// (src/comm.c:1567-1569); after a successful lookup C compares the name alone
// because wildhost and double_wild are uninitialised stack (R1a).
func TestAcceptBanLevelStrings(t *testing.T) {
	bm := game.NewBanManager()
	if err := bm.AddBan("010.000.020.*", game.BanAll, "God"); err != nil {
		t.Fatal(err)
	}
	if got := AcceptBanLevel(bm, "010.000.020.005", false); got != game.BanAll {
		t.Fatalf("unresolved AcceptBanLevel = %d, want BanAll", got)
	}
	if got := AcceptBanLevel(bm, "host.example", true); got != game.BanNot {
		t.Fatalf("resolved AcceptBanLevel = %d, want BanNot (host-only)", got)
	}
	if got := AcceptBanLevel(nil, "010.000.020.005", false); got != game.BanNot {
		t.Fatalf("AcceptBanLevel without a ban manager = %d, want BanNot", got)
	}
}

// WildBanHosts is C's two derived strings, and it declines non-quad input.
func TestWildBanHosts(t *testing.T) {
	wild := WildBanHosts("198.051.100.005")
	if len(wild) != 2 || wild[0] != "198.051.100.*" || wild[1] != "198.051.*.*" {
		t.Fatalf("WildBanHosts = %q", wild)
	}
	if got := WildBanHosts("client.example"); got != nil {
		t.Fatalf("WildBanHosts(non-quad) = %q, want nil", got)
	}
}
