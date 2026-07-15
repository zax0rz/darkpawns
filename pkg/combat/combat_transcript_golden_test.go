package combat

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tier-3 fidelity prototype: a deterministic BEHAVIORAL golden transcript.
//
// We run a fixed combat encounter under a fixed-seed Roller (the RNG seam's payoff — combat is now
// reproducible) and record the round-by-round outcome. The transcript is compared to a committed
// golden file. Any change to the hit/damage formulas, the message tiers, OR the RNG seam that alters
// combat behavior changes the transcript and fails this test — a Go-vs-Go regression tripwire.
//
// Regenerate the golden intentionally with:  go test ./pkg/combat/ -run TestCombatTranscript -update
var updateGolden = flag.Bool("update", false, "rewrite combat transcript golden file")

func simulateCombat() string {
	hero := &mockCombatant{
		name: "Hero", npc: false, class: ClassWarrior, room: 100, level: 10,
		hp: 100, maxHP: 100, ac: 50, thac0: 15, position: PosStanding, sex: 1,
		str: 16, intVal: 12, wis: 10, hitroll: 1, damroll: 2, damageRoll: DiceRoll{},
	}
	orc := &mockCombatant{
		name: "Orc", npc: true, room: 100, level: 5,
		hp: 60, maxHP: 60, ac: 60, thac0: 20, position: PosStanding, sex: 0,
		str: 15, intVal: 10, wis: 10, hitroll: 0, damroll: 1, damageRoll: DiceRoll{},
	}
	weapon := DiceRoll{Num: 2, Sides: 4}

	var b strings.Builder
	b.WriteString("# Deterministic combat transcript (CMWC seed 1337) — Hero(Warrior L10) vs Orc(L5)\n")
	// Fixed seed → fixed RNG sequence → reproducible encounter. The seam makes this possible.
	WithRoller(NewSeededRoller(1337), func() {
		for round := 1; round <= 15 && orc.hp > 0; round++ {
			if CalculateHitChance(hero, orc, HitModifiers{}) {
				dam := CalculateDamage(hero, orc, weapon, AttackNormal)
				orc.hp -= dam
				fmt.Fprintf(&b, "round %2d: HIT  %3d dmg  (Orc hp -> %d)\n", round, dam, orc.hp)
			} else {
				fmt.Fprintf(&b, "round %2d: MISS\n", round)
			}
		}
	})
	if orc.hp <= 0 {
		b.WriteString("result: Orc slain.\n")
	} else {
		b.WriteString("result: Orc survives.\n")
	}
	return b.String()
}

func TestCombatTranscript_Golden(t *testing.T) {
	got := simulateCombat()
	goldenPath := filepath.Join("testdata", "combat_transcript.golden")

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote golden: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run once with -update to create it): %v", err)
	}
	if got != string(want) {
		t.Errorf("combat transcript diverged from golden — a combat-formula or RNG-seam regression.\n"+
			"--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

// TestCombatTranscript_Deterministic guards the harness itself: the same seed must produce the same
// transcript every time, or the golden approach is meaningless (something outside the Roller leaks).
func TestCombatTranscript_Deterministic(t *testing.T) {
	if a, b := simulateCombat(), simulateCombat(); a != b {
		t.Errorf("non-deterministic transcript — RNG leaks outside the seam.\nA:\n%s\nB:\n%s", a, b)
	}
}
