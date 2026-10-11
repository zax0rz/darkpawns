package session

import "testing"

// TestNormalizeCRLFExactBytes pins the exact output of every line-ending
// shape. Both complete pairs — CRLF and C's historical LFCR — pass through
// verbatim: C's final write path emits a string's own spelling untransformed
// (send_to_char src/comm.c:2335 → write_to_output :1348 → write_to_descriptor
// :1380), so do_compare's "They're the same thing!\n\r" (src/new_cmds.c:1991)
// must reach the wire as LFCR. The old chain folded the pair to CRLF, which
// no caller needed (the documented purpose is the bare-LF anti-staircase
// rule, and a pair already contains both controls). Lone controls still
// become CRLF. Fail this by reverting pair preservation.
func TestNormalizeCRLFExactBytes(t *testing.T) {
	tests := []struct{ in, want string }{
		// both pair spellings, verbatim
		{"a\r\nb", "a\r\nb"},
		{"a\n\rb", "a\n\rb"},
		{"They're the same thing!\n\r", "They're the same thing!\n\r"},
		{"Looks like you don't have those objects..\n\r", "Looks like you don't have those objects..\n\r"},
		// lone controls become CRLF
		{"a\nb", "a\r\nb"},
		{"a\rb", "a\r\nb"},
		{"one\ntwo", "one\r\ntwo"},
		// adjacent and mixed endings
		{"a\n\rb\r\nc", "a\n\rb\r\nc"},
		{"a\r\r\nb", "a\r\n\r\nb"},   // lone CR then a CRLF pair
		{"a\n\r\nb", "a\n\r\r\nb"},   // an LFCR pair then a lone LF
		{"a\r\n\n\rb", "a\r\n\n\rb"}, // a CRLF pair then an LFCR pair
		{"a\r\n\rb", "a\r\n\r\nb"},   // a CRLF pair then a lone CR
		// no controls at all and empty input
		{"plain text", "plain text"},
		{"", ""},
		{"\r\n", "\r\n"},
		{"\n\r", "\n\r"},
	}
	for _, tt := range tests {
		if got := NormalizeCRLF(tt.in); got != tt.want {
			t.Errorf("NormalizeCRLF(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestNormalizeCRLFIdempotent proves a second pass never rewrites: the output
// contains only pairs and canonical CRLFs, so re-normalizing must be a
// fixed point.
func TestNormalizeCRLFIdempotent(t *testing.T) {
	for _, in := range []string{
		"a\n\rb\r\nc\nd\re",
		"They're the same thing!\n\r",
		"\n\r\r\n\n\r\n\r",
		"x\n\n\n\ry",
	} {
		once := NormalizeCRLF(in)
		if twice := NormalizeCRLF(once); twice != once {
			t.Errorf("NormalizeCRLF not idempotent: %q -> %q -> %q", in, once, twice)
		}
	}
}

// TestEnsureLineEndedKeepsCLineEndingsIntact guards the terminal write path for
// handler text that already ends a line. C's LFCR pair ("\n\r") ends most
// handler output; treating the trailing '\r' as unterminated appended a second
// CRLF, which showed up as an extra blank line between the two messages of one
// command (do_string's WARNING and Ok: modify.c:632,765).
func TestEnsureLineEndedKeepsCLineEndingsIntact(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"WARNING: You have changed the name of a player.\n\r", "WARNING: You have changed the name of a player.\n\r"},
		{"Ok.\r\n", "Ok.\r\n"},
		{"bare\n", "bare\n"},
		{"bare\r", "bare\r"},
		{"unterminated", "unterminated\r\n"},
	}
	for _, tt := range tests {
		if got := ensureLineEnded(tt.text); got != tt.want {
			t.Errorf("ensureLineEnded(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}
