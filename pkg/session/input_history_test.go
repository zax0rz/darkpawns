package session

import (
	"strings"
	"testing"
)

// TestPerformSubstMatchesC pins perform_subst (src/comm.c:2037-2080).
func TestPerformSubstMatchesC(t *testing.T) {
	for _, tc := range []struct {
		orig, subst, want string
		ok                bool
	}{
		{"say hello there", "^hello^goodbye", "say goodbye there", true},
		{"say hello hello", "^hello^bye", "say bye hello", true}, // first occurrence only
		{"say hi", "^^x", "xsay hi", true},                       // strstr(orig, "") is orig
		{"say hi", "^hi^", "say ", true},                         // empty replacement
		{"say hi", "^bye^x", "", false},                          // old not in orig
		{"say hi", "^hi", "", false},                             // no second caret
	} {
		got, ok := performSubst(tc.orig, tc.subst)
		if ok != tc.ok || got != tc.want {
			t.Errorf("performSubst(%q, %q) = %q, %v; want %q, %v", tc.orig, tc.subst, got, ok, tc.want, tc.ok)
		}
	}
	long, ok := performSubst("x", "^x^"+strings.Repeat("y", 400))
	if !ok || len(long) != cMaxInputLength-1 {
		t.Fatalf("overlong substitution = %d bytes, want %d (MAX_INPUT_LENGTH-1)", len(long), cMaxInputLength-1)
	}
}
