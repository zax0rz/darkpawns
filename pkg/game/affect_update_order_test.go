package game

// DP-1530: C's affected list is newest-first (affect_to_char prepends,
// src/handler.c:391-392) and affect_update walks from the head
// (src/magic.c:437-456), so same-tick wear-off messages print newest-first.
// ActiveAffects is stored oldest-first, so AffectUpdate walks its snapshot in
// reverse and suppresses the duplicate message on the newer of two same-spell
// expirations (src/magic.c:446-448).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestAffectUpdateSameTickWearOffNewestFirst casts Hydrogen's three detects
// (alignment, invisibility, magic) with zero durations and requires the
// wear-off messages in C's order: newest spell first.
func TestAffectUpdateSameTickWearOffNewestFirst(t *testing.T) {
	world, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()

	var messages []string
	world.MessageSink = func(_ string, msg []byte) {
		messages = append(messages, string(msg))
	}

	p := NewPlayer(1, "Tester", 3001)
	world.AddPlayer(p)

	// Cast order: alignment oldest, magic newest. Duration 0 expires all
	// three on the same AffectUpdate tick.
	p.AddAffect(engine.NewAffect(spells.SpellDetectAlign, engine.ApplyNone, 0, 0, "detect alignment"))
	p.AddAffect(engine.NewAffect(spells.SpellDetectInvis, engine.ApplyNone, 0, 0, "detect invis"))
	p.AddAffect(engine.NewAffect(spells.SpellDetectMagic, engine.ApplyNone, 0, 0, "detect magic"))

	world.AffectUpdate()

	if len(p.ActiveAffects) != 0 {
		t.Errorf("expected all three affects to expire, got %d", len(p.ActiveAffects))
	}

	want := []string{
		SpellWearOffMsg(spells.SpellDetectMagic) + "\r\n",
		SpellWearOffMsg(spells.SpellDetectInvis) + "\r\n",
		SpellWearOffMsg(spells.SpellDetectAlign) + "\r\n",
	}
	var got []string
	for _, m := range messages {
		if strings.TrimSpace(m) != "" {
			got = append(got, m)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d wear-off messages, got %d: %q", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("message %d = %q, want %q (C prints newest-first)", i, got[i], want[i])
		}
	}
}

// TestAffectUpdateDuplicateSpellWearsOffOnce pins C's duplicate guard
// (src/magic.c:446-448): when two affects of the same spell expire on the
// same tick, only the oldest prints the wear-off message.
func TestAffectUpdateDuplicateSpellWearsOffOnce(t *testing.T) {
	world, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()

	var messages []string
	world.MessageSink = func(_ string, msg []byte) {
		messages = append(messages, string(msg))
	}

	p := NewPlayer(1, "Tester", 3001)
	world.AddPlayer(p)

	// Two SPELL_ARMOR affects on different locations (affect_join merges only
	// same spell+location, so this coexistence is legal), both expiring now.
	p.AddAffect(engine.NewAffect(1, engine.ApplyStr, 0, 1, "armor older"))
	p.AddAffect(engine.NewAffect(1, engine.ApplyDex, 0, 1, "armor newer"))

	world.AffectUpdate()

	count := 0
	for _, m := range messages {
		if strings.Contains(m, "You feel less protected.") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 armor wear-off message (C's oldest-duplicate guard), got %d: %q", count, messages)
	}
}
