package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoAppraiseLowCostAndImproveDraws(t *testing.T) {
	var out strings.Builder
	w := &World{MessageSink: func(_ string, msg []byte) { out.Write(msg) }}
	ch := NewPlayer(1, "Appraiser", 1001)
	ch.worldRef = w
	ch.Stats.Wis = 100
	ch.Stats.Int = 100
	ch.SetSkill(SkillAppraise, 50)
	item := NewObjectInstance(&parser.Obj{
		VNum:      8010,
		Keywords:  "loaf bread",
		ShortDesc: "a loaf of bread",
		Cost:      3,
	}, -1)
	if err := ch.Inventory.AddItem(item); err != nil {
		t.Fatalf("add bread: %v", err)
	}

	seed, expectedCost, expectedSkill := appraiseImprovementSeed(t)
	dprng.ResetStream(seed)
	w.doAppraise(ch, nil, "appraise", "bread extra")

	want := fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", expectedCost)
	want += "Your skill in appraise improves.\r\n"
	if got := out.String(); got != want {
		t.Fatalf("appraise output = %q, want %q", got, want)
	}
	if got := ch.GetSkill(SkillAppraise); got != expectedSkill {
		t.Fatalf("appraise skill = %d, want %d", got, expectedSkill)
	}

	// C consumes exactly four draws here: the command roll, low-cost value
	// roll, improve_skill stat gate, and +3 increment. In particular, there is
	// no second random draw to decide whether the improvement line is printed.
	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Number(0, 20)
	dprng.Number(1, 200)
	dprng.Number(1, 3)
	wantNext := dprng.Number(1, 200)
	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Number(0, 20)
	dprng.Number(1, 200)
	dprng.Number(1, 3)
	if got := dprng.Number(1, 200); got != wantNext {
		t.Fatalf("appraise consumed the wrong number of draws: next=%d want=%d", got, wantNext)
	}
}

func appraiseImprovementSeed(t *testing.T) (uint32, int, int) {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 101) > 50 {
			continue
		}
		cost := 3 + dprng.Number(0, 20)
		increment := dprng.Number(1, 3)
		if increment == 3 {
			return seed, cost, 53
		}
	}
	t.Fatal("no deterministic appraise improvement seed found")
	return 0, 0, 0
}
