package parser

import "testing"

// TestCFNameParser mirrors C's fname() (src/handler.c:68-79) in the parser
// package. It must agree with game.CFName; the two are kept in step because the
// load-time article fix below needs the same rule and pkg/parser (a leaf) cannot
// import pkg/game.
func TestCFNameParser(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sword", "sword"},
		{"black-banded haute", "black"},
		{"D'haran breast plate armor", "D"},
		{"-x", ""},
		{"40 oz", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := cFName(tc.in); got != tc.want {
			t.Fatalf("cFName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCFixShortDescArticle pins the load-time article fix C applies to object
// short_descriptions (src/db.c:1388-1392) and mobile short_descrs
// (src/db.c:1294-1298): lower the first byte only when fname(sd) is a/an/the.
// Revert proof: before this port, Go's obj loader applied no strip at all, so
// "A sword" stayed "A sword"; the mobile loader matched a whole first space-word
// and lowercased all of it, so it would turn "A-bomb" into no change and "THE
// bird" into "the bird". The "A-bomb" and "THE bird" rows fail both old paths.
func TestCFixShortDescArticle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"A plus noun", "A sword", "a sword"},
		{"An plus noun", "An apple", "an apple"},
		{"The plus noun", "The Axe", "the Axe"},
		{"already lowercased", "an Exoskeletal Pack", "an Exoskeletal Pack"},
		{"only first byte lowercased", "THE bird", "tHE bird"},
		{"article then non-letter", "A-bomb", "a-bomb"},
		{"leading non-letter", "-thing", "-thing"},
		{"leading space", "  A sword", "  A sword"},
		{"no article", "sword", "sword"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cFixShortDescArticle(tc.in); got != tc.want {
				t.Fatalf("cFixShortDescArticle(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
