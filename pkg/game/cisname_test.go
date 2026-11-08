package game

import (
	"strings"
	"testing"
)

// TestCIsNameTable pins C's isname() (src/handler.c:81-111): the typed word must
// equal a whole keyword word, never a prefix. Revert proof: the removed isName()
// matched prefixes, so "d"/"door" and "doo"/"door" returned true; every false
// row below fails against it.
func TestCIsNameTable(t *testing.T) {
	cases := []struct {
		name, nameList string
		want           bool
	}{
		{"door", "door", true},        // exact match
		{"d", "door", false},          // prefix is NOT a match (the bug)
		{"doo", "door", false},        // prefix is NOT a match
		{"doors", "door", false},      // longer than the word
		{"door", "door dark", true},   // whole word in a list
		{"dark", "door dark", true},   // whole word in a list
		{"do", "door dark", false},    // prefix in a list
		{"", "door", false},           // empty typed word
		{"door", "", false},           // empty list
		{"D", "D'haran", true},        // apostrophe splits the run: "D"
		{"haran", "D'haran", true},    // apostrophe splits the run: "haran"
		{"D'haran", "D'haran", false}, // the whole token is not a letter-run
		{"corpse", "corpse", true},    // janitor literal
		{"corp", "corpse", false},     // prefix of the literal
		{"Door", "door", true},        // case-insensitive
	}
	for _, tc := range cases {
		if got := cIsName(tc.name, tc.nameList); got != tc.want {
			t.Fatalf("cIsName(%q, %q) = %v, want %v", tc.name, tc.nameList, got, tc.want)
		}
	}
}

// TestCIsNumberTable pins C's is_number() (src/interpreter.c:1175-1182): true
// only when every byte is a digit, and the empty string is true. Revert proof:
// the removed fmt.Sscanf isNumber() accepted "-5" and "12abc" and rejected "".
func TestCIsNumberTable(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123", true},
		{"0", true},
		{"", true},    // C's loop never runs -> true
		{"-5", false}, // sign is not a digit
		{"12abc", false},
		{"abc", false},
		{" 12", false},
		{"1.5", false},
	}
	for _, tc := range cases {
		if got := CIsNumber(tc.in); got != tc.want {
			t.Fatalf("CIsNumber(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestDoOpenRejectsKeywordPrefix proves the door path uses cIsName: a door keyed
// "door" does not respond to the prefix "d". C answers as if no door matched
// (find_door, src/act.movement.c:413-420). Revert proof: the pre-fix prefix
// matcher opened/locked the door instead of refusing.
func TestDoOpenRejectsKeywordPrefix(t *testing.T) {
	w, p, messages := newDoorHandlerTestWorld(t)

	w.mu.Lock()
	near := w.rooms[100].Exits["north"]
	near.Keywords = "door"
	w.rooms[100].Exits["north"] = near
	w.mu.Unlock()

	w.DoOpen(p, "d")

	got := joinedDoorMessages(messages, p.Name)
	if !strings.Contains(got, "There doesn't seem to be a d here.") {
		t.Fatalf("open d output = %q, want a no-such-door refusal", got)
	}
	if strings.Contains(got, "Okay.") {
		t.Fatalf("open d opened the door on a prefix match: %q", got)
	}
}
