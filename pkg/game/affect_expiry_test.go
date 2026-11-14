package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// affectExpiryFixture builds a one-player world whose MessageSink captures the
// player's own output for wear-off assertions.
func affectExpiryFixture(t *testing.T) (*World, *Player, func() []string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	p := NewPlayer(1, "Wearee", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, p, func() []string {
		var lines []string
		for _, l := range strings.Split(strings.TrimRight(out.String(), "\r\n"), "\r\n") {
			if l != "" {
				lines = append(lines, l+"\r\n")
			}
		}
		out.Reset()
		return lines
	}
}

// TestPlayerWearOffMultiRecordSpeaksOnce pins C's last-record condition on the
// player path (src/magic.c:439-444): a spell stored as two records — bless, one
// per location — sends its wear-off once when both expire the same tick. The
// pre-fix loop had no guard and printed one line per record (DP-1478).
func TestPlayerWearOffMultiRecordSpeaksOnce(t *testing.T) {
	w, p, drain := affectExpiryFixture(t)

	p.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplyHitroll, 0, 2, 0, "bless"))
	p.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplySavingSpell, 0, -2, 0, "bless"))

	w.AffectUpdate()

	got := drain()
	if len(got) != 1 || got[0] != "You feel less righteous.\r\n" {
		t.Fatalf("two-record bless wear-off = %q, want exactly one line", got)
	}
	if remaining := p.ActiveAffects; len(remaining) != 0 {
		t.Fatalf("expired records left on the player: %d", len(remaining))
	}
}

// TestPlayerAffectExpiryRemovesOneRecord pins C's affect_remove on the expiry
// path (src/handler.c:428-438): when one record of a stacked spell expires, the
// other record survives with its own modifier. The pre-fix call to
// engine.AffectFromChar here was affect_from_char — C's remove-all — which is
// not on the expiry path in C (DP-1478).
func TestPlayerAffectExpiryRemovesOneRecord(t *testing.T) {
	w, p, drain := affectExpiryFixture(t)
	_ = w

	baseHitroll := p.GetHitroll()
	// Oldest-first list: the hitroll record (alive, duration 3) is the older
	// neighbour the C guard consults; the saves record expires this tick.
	p.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplyHitroll, 3, 2, 0, "bless"))
	p.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplySavingSpell, 0, -2, 0, "bless"))
	if got := p.GetHitroll(); got != baseHitroll+2 {
		t.Fatalf("hitroll with bless = %d, want %d", got, baseHitroll+2)
	}

	w.AffectUpdate()

	if remaining := p.ActiveAffects; len(remaining) != 1 || remaining[0].Location != engine.ApplyHitroll {
		t.Fatalf("remaining records = %+v, want exactly the hitroll record", remaining)
	}
	if got := p.GetHitroll(); got != baseHitroll+2 {
		t.Fatalf("hitroll after the saves record expired = %d, want %d (survivor's modifier stays)", got, baseHitroll+2)
	}
	got := drain()
	if len(got) != 1 || got[0] != "You feel less righteous.\r\n" {
		t.Fatalf("unequal-duration bless wear-off = %q, want exactly one line", got)
	}
}

// TestMobAffectExpiryRemovesOneRecord is the mob twin: RemoveAffectRecord
// unlinks exactly the expiring node (src/handler.c:428-438); the pre-fix
// RemoveAffectBySpell removed every record of the spell at the first expiry
// (DP-1478).
func TestMobAffectExpiryRemovesOneRecord(t *testing.T) {
	w, mob, drain := mobWearOffFixture(t)

	baseHitroll := mob.GetHitroll()
	// OrderedAffects is newest-first: the saves record (expiring) is walked
	// first and its guard sees the older hitroll record still alive.
	mob.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplyHitroll, 3, 2, 0, "bless"))
	mob.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplySavingSpell, 0, -2, 0, "bless"))
	if got := mob.GetHitroll(); got != baseHitroll+2 {
		t.Fatalf("hitroll with bless = %d, want %d", got, baseHitroll+2)
	}

	w.AffectUpdate()

	remaining := mob.OrderedAffects()
	if len(remaining) != 1 || remaining[0].Location != engine.ApplyHitroll {
		t.Fatalf("remaining mob records = %+v, want exactly the hitroll record", remaining)
	}
	if remaining[0].Duration != 2 {
		t.Fatalf("survivor duration = %d, want 2 (3 decremented once)", remaining[0].Duration)
	}
	if got := mob.GetHitroll(); got != baseHitroll+2 {
		t.Fatalf("hitroll after the saves record expired = %d, want %d (survivor's modifier stays)", got, baseHitroll+2)
	}
	got := drain()
	if len(got) != 1 || got[0] != "You feel less righteous.\r\n" {
		t.Fatalf("unequal-duration bless wear-off = %q, want exactly one line", got)
	}
}
