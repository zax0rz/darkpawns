package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestListsHiddenBodyMatchesCHolylightClause pins list_char_to_char's second
// disjunct: a body carrying AFF_HIDE is listed only for a looker with
// PRF_HOLYLIGHT whose real level covers the body's invis level
// (src/act.informative.c:635-638). Without the clause the port dropped the
// " (hidden)" listing entirely.
func TestListsHiddenBodyMatchesCHolylightClause(t *testing.T) {
	viewer := NewPlayer(1, "Viewer", 1001)
	peer := NewPlayer(2, "Peer", 1001)

	if listsHiddenBody(viewer, peer) {
		t.Fatal("a looker without holylight must not list a hidden body")
	}

	viewer.SetHolyLight(true)
	if !listsHiddenBody(viewer, peer) {
		t.Fatal("holylight must list a hidden body at equal level")
	}

	// C compares GET_REAL_LEVEL(sub) >= GET_INVIS_LEV(obj); a body hidden above
	// the looker's level stays filtered.
	peer.SetInvisLevel(viewer.GetLevel() + 1)
	if listsHiddenBody(viewer, peer) {
		t.Fatal("a body whose invis level exceeds the looker's level must stay filtered")
	}

	// MobInstance carries no invis level, which is C's zero default.
	viewer.SetInvisLevel(0)
	peer.SetInvisLevel(0)
	mob := NewMob(&parser.Mob{Keywords: "probe", ShortDesc: "a probe", HP: parser.DiceRoll{Plus: 100}}, 1001)
	if !listsHiddenBody(viewer, mob) {
		t.Fatal("holylight must list a hidden mobile")
	}
	viewer.SetHolyLight(false)
	if listsHiddenBody(viewer, mob) {
		t.Fatal("a looker without holylight must not list a hidden mobile")
	}
}
