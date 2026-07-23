package session

import "testing"

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "hello world", want: "hello world"},
		{name: "preserves newline", input: "line1\nline2", want: "line1\nline2"},
		{name: "preserves carriage return", input: "line1\r\nline2", want: "line1\r\nline2"},
		{name: "strips null byte", input: "hel\x00lo", want: "hello"},
		{name: "strips bell", input: "\aalert", want: "alert"},
		{name: "strips escape", input: "\x1b[31mred", want: "[31mred"},
		{name: "strips DEL", input: "abc\x7fdef", want: "abcdef"},
		{name: "strips tab (control char)", input: "\thello", want: "hello"},
		{name: "preserves space", input: "a b c", want: "a b c"},
		{name: "empty string", input: "", want: ""},
		{name: "only control chars", input: "\x00\x01\x02", want: ""},
		{name: "mixed printable and control", input: "a\x00b\x01c", want: "abc"},
		{name: "preserves tilde", input: "hello~world", want: "hello~world"},
		{name: "strips vertical tab", input: "a\x0bb", want: "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeMessage(tt.input); got != tt.want {
				t.Errorf("sanitizeMessage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
