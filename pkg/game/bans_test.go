package game

import (
	"testing"
)

func TestIsBannedLiteralAsterisk(t *testing.T) {
	bm := NewBanManager()

	// C isbanned uses strstr, so an asterisk in a stored site is literal.
	err := bm.AddBan("192.168.1.*", BanAll, "Admin")
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}

	// A wildcard interpretation would match this address; C does not.
	if got := bm.IsBanned("192.168.1.100"); got != BanNot {
		t.Errorf("IsBanned(192.168.1.100) = %d, want %d", got, BanNot)
	}

	if got := bm.IsBanned("192.168.1.*"); got != BanAll {
		t.Errorf("IsBanned(192.168.1.*) = %d, want %d", got, BanAll)
	}
}

func TestIsBannedSubstring(t *testing.T) {
	bm := NewBanManager()

	// Add a substring ban
	err := bm.AddBan("aol.com", BanAll, "Admin")
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}

	// Test matching subdomain
	if got := bm.IsBanned("user.aol.com"); got != BanAll {
		t.Errorf("IsBanned(user.aol.com) = %d, want %d", got, BanAll)
	}

	// Test non-matching domain
	if got := bm.IsBanned("google.com"); got != BanNot {
		t.Errorf("IsBanned(google.com) = %d, want %d", got, BanNot)
	}
}

func TestIsBannedEmptyHostname(t *testing.T) {
	bm := NewBanManager()

	// Add a ban
	err := bm.AddBan("192.168.1.1", BanAll, "Admin")
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}

	// Test empty hostname
	if got := bm.IsBanned(""); got != BanNot {
		t.Errorf("IsBanned(\"\") = %d, want %d", got, BanNot)
	}
}
