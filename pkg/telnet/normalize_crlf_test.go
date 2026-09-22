package telnet

import "testing"

func TestNormalizeCRLFTreatsCLineEndingsAsSingleBreaks(t *testing.T) {
	for _, input := range []string{"a\n\rb", "a\r\nb", "a\nb", "a\rb"} {
		if got, want := normalizeCRLF(input), "a\r\nb"; got != want {
			t.Errorf("normalizeCRLF(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestEnsureLineEndedKeepsCLineEndingsIntact guards the telnet write path for
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
