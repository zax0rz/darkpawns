package game

import (
	"strings"
	"testing"
)

// C look_at_char wraps the word "glows" in the tattoo line with
// CCWHT(ch, C_CMP) ... CCNRM(ch, C_CMP) (src/act.informative.c:415-417):
//
//	sprintf(buf, "On %s right arm is a tattoo %s... it %sglows%s softly.\r\n",
//	    HSHR(i), tattoos[GET_TATTOO(i)], CCWHT(ch, C_CMP), CCNRM(ch, C_CMP));
//
// C_CMP is _clrlevel(ch) >= 3 (src/screen.h:45-49), so `complete` is the only
// level that shows it and levels 0-2 must stay byte-identical to each other.
//
// The other C_CMP pair the inventory attributes to look is list_char_to_char's
// infravision line (src/act.informative.c:646-650). That one is unreachable:
// look_at_room returns at line 731-732 whenever
// `IS_DARK(ch->in_room) && !CAN_SEE_IN_DARK(ch)`, which is exactly the branch's
// own condition, and look_at_room:835 is the function's only caller. The port
// must not reproduce dead C output (R4), and the reachable dark-room eyes line
// (line 755) is uncoloured, which appendDarkOccupants already matches.
func TestLookAtCharTattooColorLevelsMatchC(t *testing.T) {
	for _, test := range []struct {
		level int
		name  string
		want  string
	}{
		{level: 0, name: "off", want: "On his right arm is a tattoo of a green dragon... it glows softly."},
		{level: 1, name: "sparse", want: "On his right arm is a tattoo of a green dragon... it glows softly."},
		{level: 2, name: "normal", want: "On his right arm is a tattoo of a green dragon... it glows softly."},
		{level: 3, name: "complete", want: "On his right arm is a tattoo of a green dragon... it \x1b[37mglows\x1b[0m softly."},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, p, _ := newViewTestWorld(t)
			setViewerColorLevel(p, test.level)

			target := NewPlayer(2, "Target", 100)
			target.Tattoo = 1 // "of a green dragon"
			if err := w.AddPlayer(target); err != nil {
				t.Fatal(err)
			}

			got := observationMessageText(w.DoLookTarget(p, "Target"))
			if !strings.Contains(got, test.want) {
				t.Fatalf("tattoo line at level %d = %q, want substring %q", test.level, got, test.want)
			}
			if test.level < 3 && strings.Contains(got, "\x1b[37m") {
				t.Fatalf("tattoo line emitted the white sandwich below C_CMP: %q", got)
			}
		})
	}
}
