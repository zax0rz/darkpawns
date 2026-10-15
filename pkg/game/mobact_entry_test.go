package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1438 mobile half: every mobact arm's hit() draws number(1,20) before
// damage()'s protection block, so a first swing refused by the peaceful gate
// still advances the shared stream and enrols nobody (fight.c:1815, 1335-1342;
// mobact.c:223/227/254/279/299/320/334). The old StartCombat arms enrolled
// eagerly — no swing draws, and the gate's FIGHTING(victim)==ch exemption let
// the damage land. Both refused tests fail on that shape (R5h); room 8162 is
// the aggr24-weaker case (flags 28, lib/world/wld/80.wld).

func newRealPulseWorld(t *testing.T, peaceful bool, mobs ...parser.Mob) (*World, *combat.CombatEngine) {
	t.Helper()
	flags := []string{"0", "0", "0", "0"}
	if peaceful {
		flags[0] = "16" // ROOM_PEACEFUL (combat_gates_test's parsed-world pattern)
	}
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Pulse Room", Zone: 1, Flags: flags}},
		Mobs:  mobs,
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	oldCB := combat.GetCallbacks()
	ce := combat.NewCombatEngine()
	ce.SetCallbacks(w.WireCombatCallbacks())
	w.SetCombatEngine(ce)
	t.Cleanup(func() { ce.Stop(); combat.SetCallbacks(oldCB) })
	return w, ce
}

// missAfterPulseDraws finds a seed whose stream, after the pulse's
// unconditional door and sound draws, rolls a natural 1 for hit()'s
// number(1,20) — a miss, so no damage-dice draws follow.
func missAfterPulseDraws(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 8192; seed++ {
		dprng.ResetStream(seed)
		dprng.Number(0, 18)
		dprng.Number(0, 15)
		if dprng.Number(1, 20) == 1 {
			return seed
		}
	}
	t.Fatal("no seed found with a missed swing after the pulse draws")
	return 0
}

func replayMissedSwing(seed uint32) int {
	dprng.ResetStream(seed)
	dprng.Number(0, 18)
	dprng.Number(0, 15)
	dprng.Number(1, 20)
	return dprng.Number(0, 999)
}

func TestMobactAggr24WeakerMobilePeacefulFirstHitRefused(t *testing.T) {
	w, ce := newRealPulseWorld(t, true,
		parser.Mob{
			VNum: 9601, Keywords: "lord", ShortDesc: "the lord", Level: 33,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 500}, ActionFlags: []string{"aggr24", "aggressive", "sentinel"},
		},
		parser.Mob{
			VNum: 9602, Keywords: "courier", ShortDesc: "the courier", Level: 5,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
		},
	)
	lord, err := w.SpawnMob(9601, 1001)
	if err != nil {
		t.Fatalf("SpawnMob lord: %v", err)
	}
	courier, err := w.SpawnMob(9602, 1001)
	if err != nil {
		t.Fatalf("SpawnMob courier: %v", err)
	}

	seed := missAfterPulseDraws(t)
	dprng.ResetStream(seed)
	w.mobileActivityForMob(lord)
	gotNext := dprng.Number(0, 999)
	if want := replayMissedSwing(seed); gotNext != want {
		t.Fatalf("peaceful first hit changed the draw stream: next=%d, want %d (door, sound, number(1,20) consumed)", gotNext, want)
	}
	if ce.IsFighting(lord) || courier.GetFightingBody() != nil {
		t.Fatal("peaceful first hit enrolled anyone (C: damage() refuses before set_fighting)")
	}
}

func TestMobactAggressivePeacefulRoomRefusesAndDraws(t *testing.T) {
	w, ce := newRealPulseWorld(t, true,
		parser.Mob{
			VNum: 9603, Keywords: "brute", ShortDesc: "the brute", Level: 10,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 100}, ActionFlags: []string{"aggressive", "sentinel"},
		},
	)
	mob, err := w.SpawnMob(9603, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	p := NewPlayer(1, "Bystander", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	seed := missAfterPulseDraws(t)
	dprng.ResetStream(seed)
	w.mobileActivityForMob(mob)
	gotNext := dprng.Number(0, 999)
	if want := replayMissedSwing(seed); gotNext != want {
		t.Fatalf("peaceful aggressive swing changed the draw stream: next=%d, want %d", gotNext, want)
	}
	if ce.IsFighting(mob) || p.GetFightingBody() != nil {
		t.Fatal("peaceful aggressive swing enrolled anyone")
	}
}

func TestMobactAggressiveNormalRoomSwingsAndEnrols(t *testing.T) {
	w, ce := newRealPulseWorld(t, false,
		parser.Mob{
			VNum: 9604, Keywords: "brute", ShortDesc: "the brute", Level: 10,
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 100}, ActionFlags: []string{"aggressive", "sentinel"},
		},
	)
	mob, err := w.SpawnMob(9604, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	p := NewPlayer(1, "Victim", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	seed := missAfterPulseDraws(t)
	dprng.ResetStream(seed)
	w.mobileActivityForMob(mob)
	gotNext := dprng.Number(0, 999)
	if want := replayMissedSwing(seed); gotNext != want {
		t.Fatalf("aggressive swing changed the draw stream: next=%d, want %d", gotNext, want)
	}
	// C's miss arm still calls damage(ch, vict, 0, w_type), which enrols both
	// sides (fight.c:1408, 1445).
	if !ce.IsFighting(mob) || p.GetFightingBody() != mob {
		t.Fatal("missed first swing failed to enrol the pair")
	}
}
