package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// combatSpecialWorld builds a world whose combat engine carries C's per-round
// MOB_SPEC seam (src/fight.c:2030-2031): the assigned special runs after the
// mobile's ordinary attack. The MobSpecialFunc body mirrors
// Manager.SetMobSpecialFunc (pkg/session/manager.go), whose flag gates are
// proven there; this fixture proves the engine seam and the spec bodies.
func combatSpecialWorld(t *testing.T, specVNum, mobVNum, level, hp int, keywords string) (*World, *combat.CombatEngine, *Player, *MobInstance) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: mobVNum, Keywords: keywords, ShortDesc: keywords,
			LongDesc: keywords + " is here.", Level: level,
			HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: hp},
			Alignment: 0, Race: 1,
			// The parser's simple-mob defaults (parse_simple_mob): base 11s.
			Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	ce := combat.NewCombatEngine()
	w.SetCombatEngine(ce)
	ce.MobSpecialFunc = func(mob combat.Combatant) bool {
		instance, ok := mob.(*MobInstance)
		if !ok || instance == nil || !instance.HasFlag("spec") || instance.HasMobFlag(MobFlagExtract) {
			return false
		}
		spec := GetMobSpec(instance.GetVNum())
		if spec == nil {
			return false
		}
		return spec(w, nil, instance, "", "")
	}
	if name, _ := GetMobSpecName(specVNum); name == "" {
		t.Fatalf("spec vnum %d has no assignment", specVNum)
	}
	if _, ok := MobSpecAssign[mobVNum]; !ok {
		t.Fatalf("mob vnum %d has no spec assignment", mobVNum)
	}
	hero := NewPlayer(1, "Hero", 1001)
	hero.SetLevel(20)
	hero.SetPosition(combat.PosStanding)
	hero.SetHP(500)
	hero.SetMaxHP(500)
	if err := w.AddPlayer(hero); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(mobVNum, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosStanding)
	mob.SetHealth(hp)
	if hp > 0 {
		mob.SetMaxHP(hp)
	}
	mob.SetMobFlag(MobFlagSpec)
	if name, _ := GetMobSpecName(mobVNum); name == "" {
		t.Fatalf("mob %d resolves no spec", mobVNum)
	}
	return w, ce, hero, mob
}

// TestCombatRoundSnakeSpecialFiresAfterOrdinaryAttack turns
// mob.snake-unreachable green: C perform_violence() calls the assigned snake
// special every round after the mobile's ordinary attack
// (src/fight.c:2030-2031; src/spec_procs.c:330-346). The snake (14103, level
// 5) bites FIGHTING(ch) on number(0,32-level)==0 and casts poison; the bite
// follows the ordinary attack output in the transcript (R1/R3a/R5e).
func TestCombatRoundSnakeSpecialFiresAfterOrdinaryAttack(t *testing.T) {
	const snakeVNum = 14103
	for seed := uint32(1); seed <= 300; seed++ {
		w, ce, hero, mob := combatSpecialWorld(t, snakeVNum, snakeVNum, 5, 50, "snake")
		mutualEnrol(t, ce, mob, hero)
		var roomLog []string
		w.MessageSink = func(name string, msg []byte) {
			if name == hero.GetName() {
				roomLog = append(roomLog, string(msg))
			}
		}

		// Reference pass on a twin fixture without the spec flag: the stream
		// position after the ordinary round is deterministic, and its next
		// value is what the special's number(0,32-level) draw must consume.
		_, ce2, hero2, mob2 := combatSpecialWorld(t, snakeVNum, snakeVNum, 5, 50, "snake")
		mob2.ClearMobFlag(MobFlagSpec)
		mutualEnrol(t, ce2, mob2, hero2)
		dprng.ResetStream(seed)
		ce2.PerformRound()
		ref1 := dprng.Number(0, 27)
		if ref1 != 0 {
			continue // this seed's bite draw is nonzero; try the next
		}

		dprng.ResetStream(seed)
		ce.PerformRound()
		var biteAt, attackAt int
		biteAt, attackAt = -1, -1
		for i, m := range roomLog {
			if strings.Contains(m, "bites you") && biteAt < 0 {
				biteAt = i
			}
			if attackAt < 0 && i != biteAt && len(strings.TrimSpace(m)) > 0 {
				attackAt = i
			}
		}
		if biteAt < 0 {
			t.Fatalf("seed %d: reference bite draw was 0 but no bite ran; transcript %q", seed, roomLog)
		}
		if attackAt >= 0 && attackAt > biteAt {
			t.Fatalf("seed %d: bite at %d followed non-bite output at %d; transcript %q", seed, biteAt, attackAt, roomLog)
		}
		// call_magic's poison allows a save; keep scanning until the victim
		// fails one so the proof covers the affect landing too.
		if !hero.IsAffected(affPoison) {
			continue
		}
		return
	}
	t.Fatal("no seed in 1..300 produced a snake bite; fixture or draw gate broken")
}

// TestSnakeSpecialDrawsOnlyWithSameRoomVictim pins the draw gate
// (src/spec_procs.c:338-339): FIGHTING(ch) and the same-room check
// short-circuit before number(0,32-level), so a snake whose victim left the
// room consumes no draw (R3a).
func TestSnakeSpecialDrawsOnlyWithSameRoomVictim(t *testing.T) {
	w, _, _, mob := combatSpecialWorld(t, 14103, 14103, 5, 50, "snake")
	away := NewPlayer(2, "Away", 1002)
	if err := w.AddPlayer(away); err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosFighting)
	mob.SetFightingBody(away) // FIGHTING(ch) set, but in another room

	// A refusing snake consumes nothing, so the next draw is the stream's
	// first value.
	dprng.ResetStream(163)
	wantNext := dprng.Number(0, 27)

	dprng.ResetStream(163)
	if specSnake(w, nil, mob, "", "") {
		t.Fatal("snake bit a victim in another room")
	}
	if got := dprng.Number(0, 27); got != wantNext {
		t.Fatalf("next draw after refused snake = %d, want %d (the gate must not draw)", got, wantNext)
	}
}

// TestCombatRoundDelegatorsRunFighterSpecial proves the three guard
// specials C routes through fighter() while fighting — cityguard
// (src/spec_procs.c:779-780), take_to_jail (src/spec_procs2.c:1435-1436) and
// breed_killer (src/spec_procs2.c:1688-1690) — delegate to specFighter with
// no gate or draw of their own: on twin fixtures and identical seeds the
// delegated call returns specFighter's result and leaves the draw stream at
// specFighter's position (R1/R3/R5e).
func TestCombatRoundDelegatorsRunFighterSpecial(t *testing.T) {
	for _, tc := range []struct {
		name string
		vnum int
	}{
		{name: "cityguard", vnum: 2747},
		{name: "take_to_jail", vnum: 8027},
		{name: "breed_killer", vnum: 7900},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := GetMobSpec(tc.vnum)
			if spec == nil {
				t.Fatalf("vnum %d unassigned", tc.vnum)
			}
			for seed := uint32(1); seed <= 12; seed++ {
				w, _, hero, mob := combatSpecialWorld(t, tc.vnum, tc.vnum, 15, 120, "guard")
				mob.SetFightingBody(hero)
				hero.SetFightingBody(mob)
				mob.SetPosition(combat.PosFighting)
				hero.SetPosition(combat.PosFighting)
				dprng.ResetStream(seed)
				gotRet := spec(w, nil, mob, "", "")
				gotNext := dprng.Number(0, 10)

				world2, _, hero2, mob2 := combatSpecialWorld(t, tc.vnum, tc.vnum, 15, 120, "guard")
				mob2.SetFightingBody(hero2)
				hero2.SetFightingBody(mob2)
				mob2.SetPosition(combat.PosFighting)
				hero2.SetPosition(combat.PosFighting)
				dprng.ResetStream(seed)
				wantRet := specFighter(world2, nil, mob2, "", "")
				wantNext := dprng.Number(0, 10)

				if gotRet != wantRet || gotNext != wantNext {
					t.Fatalf("seed %d: delegated ret/tail = (%v, %d), fighter's = (%v, %d)", seed, gotRet, gotNext, wantRet, wantNext)
				}
				if seed == 1 && !wantRet && !gotRet {
					continue
				}
			}
		})
	}
}

// TestCombatRoundSpecialRunsAfterOrdinaryAttack proves the per-round seam's
// order with a drawless special: C perform_violence() invokes
// perform_violence's special after the mobile's attack loop
// (src/fight.c:2009-2031), so teleporter's "My work here is done." (no RNG,
// src/spec_procs2.c:2021-2030) follows the ordinary attack output.
func TestCombatRoundSpecialRunsAfterOrdinaryAttack(t *testing.T) {
	const teleporterVNum = 14411
	if _, ok := MobSpecAssign[teleporterVNum]; !ok {
		t.Fatal("teleporter vnum not assigned")
	}
	for seed := uint32(1); seed <= 40; seed++ {
		w, ce, hero, mob := combatSpecialWorld(t, teleporterVNum, teleporterVNum, 15, 100, "wizard")
		// Below half health: the teleport arm fires deterministically.
		mob.SetMaxHP(200)
		mutualEnrol(t, ce, mob, hero)
		var roomLog []string
		w.MessageSink = func(name string, msg []byte) {
			if name == hero.GetName() {
				roomLog = append(roomLog, string(msg))
			}
		}
		dprng.ResetStream(seed)
		ce.PerformRound()
		doneAt := -1
		for i, m := range roomLog {
			if strings.Contains(m, "My work here is done") {
				doneAt = i
				break
			}
		}
		if doneAt < 0 {
			continue
		}
		if doneAt == 0 {
			t.Fatalf("seed %d: special fired before any ordinary attack output: %q", seed, roomLog)
		}
		return
	}
	t.Fatal("no seed in 1..40 ran the teleport arm")
}

// TestMagicUserSelectsFightingMobVictim pins the victim walk
// (src/spec_procs.c:417-425): C iterates every body in the room — mobiles
// included — and falls back to FIGHTING(ch), so a caster fighting another
// mobile still casts. Go previously only considered players and declined.
func TestMagicUserSelectsFightingMobVictim(t *testing.T) {
	for seed := uint32(1); seed <= 60; seed++ {
		w, _, hero, caster := combatSpecialWorld(t, 1307, 1307, 20, 200, "mage")
		_ = hero
		foe, err := w.SpawnMob(1307, 1001)
		if err != nil {
			t.Fatal(err)
		}
		foe.SetPosition(combat.PosFighting)
		foe.SetHealth(100)
		caster.SetPosition(combat.PosFighting)
		caster.SetFightingBody(foe)
		foe.SetFightingBody(caster)
		var roomLog []string
		w.MessageSink = func(name string, msg []byte) {
			roomLog = append(roomLog, fmt.Sprintf("%s: %s", name, msg))
		}
		dprng.ResetStream(seed)
		specMagicUser(w, nil, caster, "", "")
		for _, m := range roomLog {
			// castMobSpell emits the verbal component before call_magic;
			// an utterance proves the cast ran with the mob foe as victim
			// (no player fights the caster, so the fallback is FIGHTING).
			if strings.Contains(m, "utters") {
				return
			}
		}
	}
	t.Fatal("no seed in 1..60 produced a cast at the mob victim")
}

func mutualEnrol(t *testing.T, ce *combat.CombatEngine, mob *MobInstance, hero *Player) {
	t.Helper()
	if err := ce.StartCombat(mob, hero); err != nil {
		t.Fatalf("enrol mob: %v", err)
	}
	if err := ce.StartCombat(hero, mob); err != nil {
		t.Fatalf("enrol hero: %v", err)
	}
	mob.SetPosition(combat.PosFighting)
	hero.SetPosition(combat.PosFighting)
}
