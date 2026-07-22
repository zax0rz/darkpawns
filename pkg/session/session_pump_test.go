package session

import (
	"testing"
)

func TestStripANSIString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "hello world", want: "hello world"},
		{name: "green text", input: "\x1b[32mgreen\x1b[0m", want: "green"},
		{name: "red text", input: "\x1b[31mred\x1b[0m", want: "red"},
		{name: "bold+color", input: "\x1b[1;31mbold red\x1b[0m", want: "bold red"},
		{name: "multiple escapes", input: "\x1b[32mhello\x1b[33m world\x1b[0m", want: "hello world"},
		{name: "cursor movement", input: "\x1b[5;10Hpositioned", want: "positioned"},
		{name: "empty string", input: "", want: ""},
		{name: "only escapes", input: "\x1b[32m\x1b[0m", want: ""},
		{name: "no trailing letter", input: "abc\x1b[32", want: "abc\x1b[32"},
		{name: "mixed content", input: "\x1b[1mBOLD\x1b[22m and \x1b[4munderlined\x1b[24m", want: "BOLD and underlined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripANSIString(tt.input); got != tt.want {
				t.Errorf("stripANSIString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripANSIRecursive(t *testing.T) {
	t.Run("map values", func(t *testing.T) {
		input := map[string]interface{}{
			"name":  "\x1b[32mHero\x1b[0m",
			"desc":  "a \x1b[1mlegendary\x1b[22m warrior",
			"age":   25,
			"alive": true,
		}
		want := map[string]interface{}{
			"name":  "Hero",
			"desc":  "a legendary warrior",
			"age":   25,
			"alive": true,
		}
		stripANSIRecursive(input)
		for k, v := range want {
			if input[k] != v {
				t.Errorf("key %q = %v (%T), want %v (%T)", k, input[k], input[k], v, v)
			}
		}
	})

	t.Run("nested map", func(t *testing.T) {
		input := map[string]interface{}{
			"inner": map[string]interface{}{
				"text": "\x1b[33mwarning\x1b[0m",
			},
		}
		stripANSIRecursive(input)
		inner := input["inner"].(map[string]interface{})
		if got := inner["text"].(string); got != "warning" {
			t.Errorf("nested text = %q, want %q", got, "warning")
		}
	})

	t.Run("slice values", func(t *testing.T) {
		input := []interface{}{
			"\x1b[31mred\x1b[0m",
			"\x1b[32mgreen\x1b[0m",
			42,
		}
		want := []interface{}{
			"red",
			"green",
			42,
		}
		stripANSIRecursive(input)
		for i, v := range want {
			if input[i] != v {
				t.Errorf("index %d = %v (%T), want %v (%T)", i, input[i], input[i], v, v)
			}
		}
	})

	t.Run("nil input does not panic", func(t *testing.T) {
		stripANSIRecursive(nil)
	})
}
