package telnet

import "testing"

func TestNormalizeCRLFTreatsCLineEndingsAsSingleBreaks(t *testing.T) {
	for _, input := range []string{"a\n\rb", "a\r\nb", "a\nb", "a\rb"} {
		if got, want := normalizeCRLF(input), "a\r\nb"; got != want {
			t.Errorf("normalizeCRLF(%q) = %q, want %q", input, got, want)
		}
	}
}
