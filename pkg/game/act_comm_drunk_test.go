package game

import "testing"

// TestSpeakDrunkChainSemantics pins the C speak_drunk loop's no-break scan
// (act.comm.c:1389-1441). Every expectation below was produced by compiling
// the exact C loop standalone and is byte-verified oracle behavior — the
// live proof is the comm-drunk scenario (say.drunk-speech). The chain quirk:
// after a match advances the cursor the table scan CONTINUES from the next
// entry, so "kill" can never fire after "the " matched in the same pass
// ("th' killer ish here", NOT longest-match's "th' murderizeer ish here"),
// and each pass appends at most one unmatched byte ("Sick" → "SHick" — the
// 'c' falls back before "ck" can ever be seen).
func TestSpeakDrunkChainSemantics(t *testing.T) {
	cases := []struct{ in, want string }{
		{"the killer is here", "th' killer ish here"},
		{"what is this!", "wha' ish thhish!"},
		{"how are you?", "howsh arsh you?"},
		{"Sick duck sits", "SHick duck shiths"},
		{"ss tt uu", "shs tht uu"},
		{"  double space", "  double shpace"},
	}
	for _, tc := range cases {
		if got := speakDrunk(tc.in); got != tc.want {
			t.Errorf("speakDrunk(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
