package session

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C's do_stat_character prints k->player.time.birth for both the Created stamp
// and age() (src/act.wizard.c:754, 764), and APPLY_AGE has already shifted that
// field in place (src/handler.c:160-161):
//
//	case APPLY_AGE:  ch->player.time.birth -= (mod * SECS_PER_MUD_YEAR);
//
// so a character wearing an APPLY_AGE item is statted at the shifted age. The
// port printed the raw Birth for both lines.
func TestStatPlayerAgeAndCreatedUseTheAffectedBirth(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Ageview", 1001, true)

	// A birth well in the past, so the APPLY_AGE shift moves the reported age.
	s.player.Birth = time.Now().AddDate(-20, 0, 0).Unix()
	baseAge := game.Age(s.player.EffectiveBirth()).Year

	worn := game.NewObjectInstance(&parser.Obj{
		VNum: 9001, Keywords: "ring", ShortDesc: "a ring", TypeFlag: game.ITEM_ARMOR,
		WearFlags: [4]int{1 | 2}, // TAKE | FINGER
		Affects:   []parser.ObjAffect{{Location: engine.ApplyAge, Modifier: 5}},
	}, -1)
	if err := s.player.Equipment.Equip(worn, s.player.Inventory); err != nil {
		t.Fatalf("Equip: %v", err)
	}
	if s.player.EffectiveBirth() == s.player.Birth {
		t.Fatal("the worn APPLY_AGE item did not shift the effective birth")
	}

	s.sendStatPlayerReport(s.player, 1001, true, 0)
	got := statColorOutput(t, s)

	shiftedAge := game.Age(s.player.EffectiveBirth()).Year
	if shiftedAge == baseAge {
		t.Fatalf("test setup is vacuous: the shift did not change the age (%d)", baseAge)
	}
	if want := fmt.Sprintf("Played [0d 0h], Age [%d]\r\n", shiftedAge); !strings.Contains(got, want) {
		t.Fatalf("stat age line = %q, want %q (raw birth would give %d)", got, want, baseAge)
	}
	// The Created stamp is the same field, so it moves too.
	created := time.Unix(s.player.EffectiveBirth(), 0).Format("Mon Jan _2 15:04:05 2006")
	if !strings.Contains(got, "Created: ["+created+"]") {
		t.Fatalf("stat created line = %q, want the affected stamp %s", got, created)
	}
}
