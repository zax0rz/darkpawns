package grapevine

import (
	"strings"
	"testing"
)

// Peer-MUD players author broadcast content; control characters in it must
// never reach a player terminal (VULN-039). The strip must remove every
// byte < 0x20 and 0x7f while preserving printable text — including the
// multi-byte runes a legitimate gossip message carries.
func TestStripControlChars(t *testing.T) {
	cases := []struct{ in, want string }{
		{"clean message", "clean message"},
		{"esc\x1b[2Jclear", "esc[2Jclear"},
		{"osc\x1b]0;title\x07end", "osc]0;titleend"},
		{"cr lf \r\n stripped", "cr lf  stripped"},
		{"del\x7fchar", "delchar"},
		{"utf8 ünïcödé 漢字", "utf8 ünïcödé 漢字"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stripControlChars(tc.in); got != tc.want {
			t.Errorf("stripControlChars(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if strings.ContainsRune(stripControlChars("\x01\x02\x1b\x7f"), 0x1b) {
		t.Fatal("control characters survived the strip")
	}
}
