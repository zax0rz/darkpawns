package main

import "testing"

func TestShellSingleQuote(t *testing.T) {
	cases := map[string]string{
		"Zork":        `'Zork'`,
		"a b":         `'a b'`,
		"$(rm -rf ~)": `'$(rm -rf ~)'`,
		"O'Brien":     `'O'\''Brien'`,
	}
	for in, want := range cases {
		if got := shellSingleQuote(in); got != want {
			t.Errorf("shellSingleQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
