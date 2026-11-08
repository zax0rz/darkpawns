package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// stubTipsterNumber pins the tipster's single draw and records its range.
func stubTipsterNumber(t *testing.T, value int) *mobactDrawRange {
	t.Helper()
	old := tipsterNumber
	var seen mobactDrawRange
	tipsterNumber = func(from, to int) int {
		seen = mobactDrawRange{from: from, to: to}
		return value
	}
	t.Cleanup(func() { tipsterNumber = old })
	return &seen
}

// TestSpecTipsterSpeaksThroughNpcSay pins that every tip is a C do_say line
// routed through npcSay (src/spec_procs2.c:475-514): the verb follows the final
// punctuation, case 6 speaks two lines, draws 13..50 are silent, and the draw
// range is number(0, 50). Before the fix the port always spoke, used
// number(0, 12), joined case 6 into one line, and printed "Npc says '...'"
// without a comma. Revert proof (R5h): the pre-fix body fails every case below.
func TestSpecTipsterSpeaksThroughNpcSay(t *testing.T) {
	for _, tc := range []struct {
		name   string
		draw   int
		intel  int
		want   []string
		silent bool
	}{
		{
			name: "case 0 exclaims", draw: 0, intel: 11,
			want: []string{"A gossip exclaims, 'For ansi color, type COLOR COMPLETE!'\r\n"},
		},
		{
			name: "case 5 states", draw: 5, intel: 11,
			want: []string{"A gossip states, 'You can hire a mercenary by giving him 100 coins.'\r\n"},
		},
		{
			name: "case 6 speaks twice", draw: 6, intel: 11,
			want: []string{
				"A gossip states, 'A bribe of a couple hundred coins will make a guard look the other way while you fight.'\r\n",
				"A gossip states, 'Guards will attack you if you fight in front of them.'\r\n",
			},
		},
		{
			name: "case 11 keeps C's single space", draw: 11, intel: 11,
			want: []string{"A gossip says, 'Use an identify scroll on yourself to see your numerical statistics. (Available at your local magick shop.)'\r\n"},
		},
		{
			name: "draw past case 12 is silent", draw: 13, intel: 11, silent: true,
		},
		{
			name: "numbed tipster is silent", draw: 0, intel: 0, silent: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newActTemplateWorld(1001)
			observer := addCapturedPlayer(t, w, "Pracobserver", 1001)
			wPlayer(t, w, "Pracobserver").SetPosition(8)
			gossip := NewMob(&parser.Mob{
				VNum: 9001, Keywords: "gossip", ShortDesc: "a gossip",
				Int: tc.intel, Wis: 11,
			}, 1001)
			gossip.SetPosition(8)

			seen := stubTipsterNumber(t, tc.draw)
			specTipster(w, nil, gossip, "", "")

			if seen.from != 0 || seen.to != 50 {
				t.Fatalf("tipster draw = number(%d, %d), want number(0, 50)", seen.from, seen.to)
			}
			if tc.silent {
				if observer.msgs != "" {
					t.Fatalf("observer got %q, want silence", observer.msgs)
				}
				return
			}
			for _, line := range tc.want {
				if !strings.Contains(observer.msgs, line) {
					t.Fatalf("observer got %q, want line %q", observer.msgs, line)
				}
			}
			if strings.Contains(observer.msgs, "$n") || strings.Contains(observer.msgs, "says '") {
				t.Fatalf("observer got a non-do_say line: %q", observer.msgs)
			}
		})
	}
}
