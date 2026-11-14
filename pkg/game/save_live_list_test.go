package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// DP-1533: C char_to_store strips the live affects and re-adds the stored
// array in order (src/db.c:2576-2577, 2627-2631); affect_to_char prepends
// (src/handler.c:415-416), so every save_char leaves the live list reversed
// and truncated at MAX_AFFECT (src/structs.h:655; db.c:2547's copy bound).
// The stored bytes keep the pre-strip list order (st->affected[]).
func TestPlayerToSaveDataReversesLiveAffectList(t *testing.T) {
	p := NewPlayer(1, "Flip", 1001)
	p.AddAffect(engine.NewAffectDirect(3, ApplyNone, 30, 0, engine.AFFSanctuary, "bless"))
	p.AddAffect(engine.NewAffectDirect(97, ApplyNone, 20, 0, engine.AFFNothing, "haste"))

	playerToSaveData(p)
	p.ApplyCharToStoreSideEffects()

	live := p.ActiveAffects
	if len(live) != 2 {
		t.Fatalf("live affects after save = %d records, want 2", len(live))
	}
	if live[0].SpellID != 3 || live[1].SpellID != 97 {
		t.Errorf("live order after save = [%d, %d], want C's re-add order [3, 97] (the array re-prepended: the oldest record lands at the head)",
			live[0].SpellID, live[1].SpellID)
	}
}

// The stored array keeps the pre-strip list order (db.c:2547-2558 copies
// ch->affected head-first), so the file's first slot is the newest record.
func TestPlayerToSaveDataKeepsStoredArrayOrder(t *testing.T) {
	p := NewPlayer(1, "Flip", 1001)
	p.AddAffect(engine.NewAffectDirect(3, ApplyNone, 30, 0, engine.AFFSanctuary, "bless"))
	p.AddAffect(engine.NewAffectDirect(97, ApplyNone, 20, 0, engine.AFFNothing, "haste"))

	data := playerToSaveData(p)

	if len(data.Affects) != 2 {
		t.Fatalf("stored affects = %d records, want 2", len(data.Affects))
	}
	if data.Affects[0].SpellID != 97 || data.Affects[1].SpellID != 3 {
		t.Errorf("stored order = [%d, %d], want the pre-strip list order [97, 3]",
			data.Affects[0].SpellID, data.Affects[1].SpellID)
	}
}

// db.c:2547 copies at most MAX_AFFECT (32) records and the re-add never
// restores the rest, so a save drops record 33+ from the live character.
func TestPlayerToSaveDataTruncatesLiveListAtMaxAffect(t *testing.T) {
	p := NewPlayer(1, "Flip", 1001)
	for i := range cMaxAffect + 3 {
		p.AddAffect(engine.NewAffectDirect(900+i%100, ApplyNone, 100, 0, engine.AFFNothing, "stack"))
	}

	playerToSaveData(p)
	p.ApplyCharToStoreSideEffects()

	if len(p.ActiveAffects) != cMaxAffect {
		t.Fatalf("live affects after save = %d, want the MAX_AFFECT truncation (%d)",
			len(p.ActiveAffects), cMaxAffect)
	}
	// The survivors are the newest 32 records (the list head side), reversed:
	// the head-side record 971 lands last after the re-add.
	if p.ActiveAffects[0].SpellID != 903 {
		t.Errorf("first survivor = spell %d, want the oldest kept record 903 (the last re-prepended array slot)",
			p.ActiveAffects[0].SpellID)
	}
	if got := p.ActiveAffects[len(p.ActiveAffects)-1].SpellID; got != 934 {
		t.Errorf("last survivor = spell %d, want the newest kept record 934", got)
	}
}

// The records past MAX_AFFECT leave through affect_remove, which runs
// affect_total, so their modifiers stop applying once the save truncates.
func TestCharToStoreTruncationDropsStatModifiers(t *testing.T) {
	p := NewPlayer(1, "Flip", 1001)
	p.Stats.Str = 10
	p.CopyBaseAttributes()
	// The oldest record is the list tail, so it is the one truncated.
	p.AddAffect(engine.NewAffectDirect(0, ApplyStr, 100, 2, 0, "strength"))
	for i := range cMaxAffect {
		p.AddAffect(engine.NewAffectDirect(900+i, ApplyNone, 100, 0, engine.AFFNothing, "stack"))
	}
	if got := p.GetStr(); got != 12 {
		t.Fatalf("GetStr before save = %d, want 12", got)
	}

	p.ApplyCharToStoreSideEffects()

	if got := p.GetStr(); got != 10 {
		t.Errorf("GetStr after truncating save = %d, want 10: the dropped strength record must stop applying", got)
	}
}
