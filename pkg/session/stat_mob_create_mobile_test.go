package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestStatMobReadsCreateMobileExpOverride proves do_stat_character reads the
// live GET_EXP(k) (src/act.wizard.c:745-748), not the prototype's stored Exp.
// A create_mobile'd mobile is overwritten to XP 0 (src/new_cmds2.c:593), so a
// prototype carrying Exp 100 must still stat as XP 0. Reverting sendStatMob to
// proto.Exp is the revert proof: the line then reads XP 100.
func TestStatMobReadsCreateMobileExpOverride(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Mobs: []parser.Mob{{
			VNum: 69, ShortDesc: "a clone", Keywords: "clone", Exp: 100, Level: 40,
			Damage: parser.DiceRoll{Num: 1, Sides: 4},
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := makeTestSession(t, newTestManager(t, w, nil), "Statgod", 1001, true)

	mob, err := w.CreateMobileI(nil, 69, 1001, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	s.sendStatMob(mob)
	got := statColorOutput(t, s)

	if !strings.Contains(got, "Monster Class: Normal, Lev: [ 1], XP: [      0]") {
		t.Fatalf("stat mob did not read create_mobile's XP override:\n%s", got)
	}
}
