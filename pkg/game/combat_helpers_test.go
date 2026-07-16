package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestImproveSkill_DrawParity pins improveSkill to the C draw contract in
// src/act.other.c:1704 improve_skill(). The prior implementation was invented
// (number(1,100)>cur then number(0,99)<chance; bounds checked first; always +1;
// wrong message) and fails every assertion here. Proof is against the real shared
// CMWC stream and is self-referencing — no hard-coded golden numbers. A bare
// *World (not NewWorld) supplies the message sink without starting any ticker or
// spawner goroutine that would draw from the shared stream mid-assertion.
func TestImproveSkill_DrawParity(t *testing.T) {
	var out strings.Builder
	w := &World{}
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	player := NewPlayer(1, "Tester", 1001)
	player.Class = ClassThief
	player.worldRef = w

	// (1) Draw-ORDER invariant. C rolls number(1,200) BEFORE the percent bounds
	// check (line 1713 before 1715), so a mastered skill (>=97) still consumes
	// exactly one draw. The old code checked `cur>=100` first and drew zero,
	// desyncing the seeded stream on every use of a mastered skill. Whether the
	// stat gate passes or fails, the function must return after exactly one draw.
	player.Stats.Int = 12
	player.Stats.Wis = 12
	player.SetSkill("backstab", 99)

	dprng.ResetStream(1)
	dprng.Number(1, 200) // reference: consume exactly one draw
	wantNext := dprng.Number(1, 200)

	dprng.ResetStream(1)
	improveSkill(player, "backstab")
	if got := dprng.Number(1, 200); got != wantNext {
		t.Fatalf("mastered-skill improveSkill drew wrong count (next=%d want=%d)", got, wantNext)
	}
	if got := player.GetSkill("backstab"); got != 99 {
		t.Errorf("mastered skill mutated: got %d want 99", got)
	}
	if msg := lastMsg(); msg != "" {
		t.Errorf("mastered skill must be silent, got %q", msg)
	}

	// (2) Gain path. Force the stat gate to PASS (WIS+INT=200 >= any number(1,200)):
	// C then rolls number(1,3), adds it, and prints "Your skill in %s improves.\r\n"
	// only on a +3 roll (spells[skill] == the Skill* constant). Derive the expected
	// increment from the stream itself.
	player.Stats.Int = 100
	player.Stats.Wis = 100
	dprng.ResetStream(1)
	dprng.Number(1, 200)      // gate draw (passes)
	inc := dprng.Number(1, 3) // the increment C would roll

	player.SetSkill("backstab", 50)
	dprng.ResetStream(1)
	improveSkill(player, "backstab")
	if got := player.GetSkill("backstab"); got != 50+inc {
		t.Fatalf("gain: skill=%d want %d (50 + number(1,3)=%d)", got, 50+inc, inc)
	}
	msg := lastMsg()
	if inc == 3 {
		if want := "Your skill in backstab improves.\r\n"; msg != want {
			t.Errorf("+3 gain message: got %q want %q", msg, want)
		}
	} else if msg != "" {
		t.Errorf("non-+3 gain must be silent, got %q", msg)
	}
}
