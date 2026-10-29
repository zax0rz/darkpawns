package game

// SPECIAL(werewolf) bite proof (src/spec_procs3.c:427-453): the bite is an
// ordinary damage() call whose lethal tail is the shared kill seam, and the
// leg tear subtracts level*1.5 from ANY victim's move as floating point
// before truncation.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newWerewolfWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Forest", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 1, ShortDesc: "a werewolf", Keywords: "werewolf", Level: 11, Exp: 0},
			{VNum: 2, ShortDesc: "a victim", Keywords: "victim", Level: 1, Exp: 100},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// The bite's leg tear subtracts level*1.5 in floating point: a level-11
// werewolf takes 16.5 move, so 50 move becomes int(33.5) = 33, not the 34
// an integer level*3/2 would give (src/spec_procs3.c:447-449).
func TestWerewolfBiteReducesMobVictimMoveWithFloatTruncation(t *testing.T) {
	w := newWerewolfWorld(t)
	wolf, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob wolf: %v", err)
	}
	victim, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob victim: %v", err)
	}
	wolf.SetFightingBody(victim)

	// Force the howl roll to miss and the bite roll to hit.
	call := 0
	werewolfNumber = func(_, _ int) int {
		call++
		if call == 1 {
			return 1 // howl roll misses
		}
		return 0 // bite roll hits
	}
	t.Cleanup(func() { werewolfNumber = dprng.Number })

	if !specWerewolf(w, nil, wolf, "", "") {
		t.Fatal("specWerewolf declined a fighting pulse")
	}
	if got, want := victim.GetMove(), 33; got != want {
		t.Fatalf("victim move after bite = %d, want %d (50 - 11*1.5 truncated)", got, want)
	}
	if got := wolf.GetMove(); got != 50 {
		t.Fatalf("attacker move changed: %d", got)
	}
}

// A lethal bite on a mobile victim runs the shared kill seam: the victim's
// experience is drained by 1/37 at die_with_killer (src/fight.c:588), which
// HandleDeath owns — 100 becomes 98.
func TestWerewolfBiteMobVictimDeathRunsKillSeam(t *testing.T) {
	w := newWerewolfWorld(t)
	wolf, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob wolf: %v", err)
	}
	victim, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob victim: %v", err)
	}
	victim.SetHealth(1)
	wolf.SetFightingBody(victim)

	call := 0
	werewolfNumber = func(_, _ int) int {
		call++
		if call == 1 {
			return 1 // howl roll misses
		}
		return 0 // bite roll hits
	}
	t.Cleanup(func() { werewolfNumber = dprng.Number })

	specWerewolf(w, nil, wolf, "", "")

	if victim.IsAlive() {
		t.Fatal("victim survived a lethal bite")
	}
	if got, want := victim.GetExp(), 98; got != want {
		t.Fatalf("victim exp after lethal bite = %d, want %d (die_with_killer drains 1/37)", got, want)
	}
}

// die_with_killer's drain applies to every mobile death through HandleDeath,
// not just the werewolf (src/fight.c:588; src/limits.c:287-291 — gain_exp's
// immortal guard only skips PCs, so mobiles drain too).
func TestHandleDeathDrainsMobVictimExpBy37(t *testing.T) {
	w := newWerewolfWorld(t)
	victim, err := w.SpawnMob(2, 1001)
	if err != nil {
		t.Fatalf("SpawnMob victim: %v", err)
	}
	killer, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob killer: %v", err)
	}

	w.HandleDeath(victim, killer, combat.TYPE_BITE)

	if got, want := victim.GetExp(), 98; got != want {
		t.Fatalf("victim exp after HandleDeath = %d, want %d", got, want)
	}
}
