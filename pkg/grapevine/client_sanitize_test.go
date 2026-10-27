package grapevine

import (
	"strings"
	"testing"
	"unicode"
)

// Peer-MUD players author broadcast content; control characters in it must
// never reach a player terminal (VULN-039). The strip must remove C0 and DEL
// while preserving printable text — including the multi-byte runes a
// legitimate gossip message carries.
//
// It must also remove the C1 block and the Unicode format/separator
// characters: U+0085 NEL and U+009B CSI are terminal controls in their own
// right, and the bidi/zero-width characters can reorder or hide a rendered
// line without any C0 byte at all.
func TestStripControlChars(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"clean", "clean message", "clean message"},
		{"empty", "", ""},
		{"csi erase", "esc\x1b[2Jclear", "esc[2Jclear"},
		{"osc title", "osc\x1b]0;title\x07end", "osc]0;titleend"},
		{"crlf line forgery", "cr lf \r\n stripped", "cr lf  stripped"},
		{"del", "del\x7fchar", "delchar"},
		{"utf8 preserved", "utf8 ünïcödé 漢字", "utf8 ünïcödé 漢字"},
		{"c1 nel", "nel\u0085next", "nelnext"},
		{"c1 csi", "csi\u009b2Jclear", "csi2Jclear"},
		{"bidi override", "pay\u202epal", "paypal"},
		{"bidi isolate", "a\u2066b\u2069c", "abc"},
		{"zero width", "zero\u200bwidth\u200djoin", "zerowidthjoin"},
		{"bom", "\ufeffleading", "leading"},
		{"line separator", "one\u2028two", "onetwo"},
		{"paragraph separator", "one\u2029two", "onetwo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripControlChars(tc.in); got != tc.want {
				t.Errorf("stripControlChars(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	// Nothing in the removed classes may survive, whatever the input order.
	for _, bad := range []rune{'\x00', '\x01', '\x1b', '\x7f', '\u0085', '\u009b', '\u200b', '\u202e', '\u2066', '\ufeff', '\u2028', '\u2029'} {
		if strings.ContainsRune(stripControlChars("a"+string(bad)+"b"), bad) {
			t.Errorf("U+%04X survived the strip", bad)
		}
	}

	// The rule must stay a whitelist of printable text, not a blacklist that
	// misses a class: every remaining rune is non-control, non-format.
	for _, r := range stripControlChars("ok\u0085\u202e\u200b ünïcödé 漢字") {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) ||
			unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			t.Errorf("strip left a non-printable rune U+%04X", r)
		}
	}
}
