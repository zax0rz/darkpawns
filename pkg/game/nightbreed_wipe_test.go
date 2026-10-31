package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func makeNightbreedObj(proto *parser.Obj) *ObjectInstance {
	return NewObjectInstance(proto, -1)
}

func nightbreedRobes() *ObjectInstance {
	return makeNightbreedObj(&parser.Obj{
		VNum:      4331,
		Keywords:  "robes",
		ShortDesc: "the dragon Emperor robes",
		WearFlags: [4]int{1 << 17, 0, 0, 0},
		Affects: []parser.ObjAffect{
			{Location: ApplySpell, Modifier: affVampire},
			{Location: ApplySpell, Modifier: affWerewolf},
		},
	})
}

// TestApplySpellUnequipWipesPersistentNightbreedBit pins C's unequip half of
// the APPLY_SPELL arm (src/handler.c:749-782, 240-270): transform sets the
// persistent AFF_VAMPIRE with no timed record (src/act.other.c:1503), and
// removing an item whose APPLY_SPELL carries the same bit runs the negative
// arm, finds no timed provider, and wipes the real form with it. A timed
// affect providing the bitvector suppresses the wipe (src/handler.c:259-266).
// Proven live by nightbreed-unequip-wipe-depth: Lothar's look grants
// PLR_VAMPIRE, night transform sets the form, and the robes' removal ends it.
func TestApplySpellUnequipWipesPersistentNightbreedBit(t *testing.T) {
	p := NewPlayer(1, "Fangbearer", 1001)
	robes := nightbreedRobes()

	p.SetAffect(affVampire, true)
	p.applySpellUnequipWipe(robes)
	if p.IsAffected(affVampire) {
		t.Fatal("persistent AFF_VAMPIRE survived the robes' removal without a timed provider")
	}

	p.SetAffect(affVampire, true)
	p.ActiveAffects = []*engine.Affect{{Flags: AffBitToEngineFlag[affVampire]}}
	p.applySpellUnequipWipe(robes)
	if !p.IsAffected(affVampire) {
		t.Fatal("timed provider did not suppress the unequip wipe (src/handler.c:259-266)")
	}

	// Non-spell affects and out-of-range bits never touch the flags.
	p.SetAffect(affInvisible, true)
	p.applySpellUnequipWipe(makeNightbreedObj(&parser.Obj{VNum: 1, Affects: []parser.ObjAffect{{Location: 1, Modifier: affInvisible}}}))
	if !p.IsAffected(affInvisible) {
		t.Fatal("non-APPLY_SPELL affect wiped a bit")
	}
}
