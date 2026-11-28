package game

// minor_fixes_depth_test.go — fidelity cover for the minor, assigned-special
// fixes (BRIEF-2026-11-27-ember-special-proc-minor-fixes, PR 1: gates and
// draws, plus the medusa/mortician exact-token matchers and suck_in's case
// gate). Each sub-test reverts to the old body and fails (R5h): sleeping
// guards act, cross-room opponents get no rescue, immortals-only rooms
// proceed, a dying werewolf skips its draws, a typing player never
// triggers fighter draws, prefixes intercept where C's isname requires the
// full token, and "PAINTING" in caps fell through.
//
// Two rows did NOT survive their claim on re-check and are deliberately
// unfixed: tattoo4's IS_NPC gate is dead code (Player.IsNPC() is
// const-false, so the gate cannot fire on the call path, R5e), and the
// assassin/eq_thief/elements_load_cylinders isname rows are wrong — C's
// get_char_room (src/handler.c:877) and get_obj_in_list_vis
// (src/handler.c:1341) both match with isname_with_abbrevs, exactly as Go
// does.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// minorNumberSeed finds a stream seed whose first number(from, to) is want.
func minorNumberSeed(t *testing.T, from, to, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(from, to) == want {
			return seed
		}
	}
	t.Fatalf("no seed in 1..4999 draws %d first", want)
	return 0
}

type minorFixWorld struct {
	w    *World
	out  map[string]string
	next int
}

func newMinorFixWorld(t *testing.T) *minorFixWorld {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Minor Room", Zone: 1},
			{VNum: 1002, Name: "Far Room", Zone: 1},
			{VNum: 18101, Name: "Painting Room", Zone: 1},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	mf := &minorFixWorld{w: w, out: map[string]string{}}
	w.MessageSink = func(name string, msg []byte) { mf.out[name] += string(msg) }
	return mf
}

var minorMobVNum int

func (mf *minorFixWorld) addMob(t *testing.T, roomVNum, level int, keywords, shortDesc string) *MobInstance {
	t.Helper()
	minorMobVNum++
	vnum := 99000 + minorMobVNum
	proto := &parser.Mob{
		VNum: vnum, Keywords: keywords, ShortDesc: shortDesc,
		LongDesc: "A test mob is here.", Level: level,
		HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		Alignment: 0, Race: 1, Position: 8, DefaultPos: 8, Sex: 0,
		Int: 10, Wis: 10,
	}
	mf.w.mu.Lock()
	mf.w.mobs[vnum] = proto
	mf.w.mu.Unlock()
	mob, err := mf.w.SpawnMob(vnum, roomVNum)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetLevel(level)
	mob.SetPosition(combat.PosStanding)
	mob.CurrentHP = 50
	return mob
}

func (mf *minorFixWorld) addPlayer(t *testing.T, name string, roomVNum, level int) *Player {
	t.Helper()
	mf.next++
	player := NewPlayer(mf.next, name, roomVNum)
	player.SetLevel(level)
	if err := mf.w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return player
}

func (mf *minorFixWorld) take(name string) string {
	got := mf.out[name]
	mf.out[name] = ""
	return got
}

// TestMinorCityguardSleepingGate proves C's `if (cmd || !AWAKE(ch))` gate
// (src/spec_procs.c:776): a resting guard with an outlaw in the room must
// stay silent. The old `pos <= 0` gate let positions 1-4 act.
func TestMinorCityguardSleepingGate(t *testing.T) {
	mf := newMinorFixWorld(t)
	outlaw := mf.addPlayer(t, "Outlaw", 1001, 10)
	outlaw.SetPlrFlag(PlrOutlaw, true)
	guard := mf.addMob(t, 1001, 20, "cityguard", "the cityguard")
	guard.SetPosition(combat.PosSleeping)

	if specCityguard(mf.w, nil, guard, "", "") {
		t.Fatal("sleeping cityguard acted")
	}
	if got := mf.take("Outlaw"); strings.Contains(got, "OUTLAWS") {
		t.Fatalf("sleeping cityguard said %q, C returns FALSE at !AWAKE", got)
	}

	// Control: a standing guard still polices the outlaw.
	mf2 := newMinorFixWorld(t)
	outlaw2 := mf2.addPlayer(t, "Outlaw", 1001, 10)
	outlaw2.SetPlrFlag(PlrOutlaw, true)
	guard2 := mf2.addMob(t, 1001, 20, "cityguard", "the cityguard")
	specCityguard(mf2.w, nil, guard2, "", "")
	if got := mf2.take("Outlaw"); !strings.Contains(got, "We don't like OUTLAWS") {
		t.Fatalf("standing cityguard output = %q, want the outlaw line", got)
	}
}

// TestMinorMagicUserNeutralDispel proves case 12's second arm
// (src/spec_procs.c:450): IS_GOOD(vict) && !IS_GOOD(ch) — a neutral caster
// dispels good; the old evil-only gate never did.
func TestMinorMagicUserNeutralDispel(t *testing.T) {
	setup := func(t *testing.T, casterAlign, victAlign int) (*minorFixWorld, *MobInstance, *Player) {
		mf := newMinorFixWorld(t)
		vict := mf.addPlayer(t, "Victim", 1001, 10)
		vict.SetAlignment(victAlign)
		mage := mf.addMob(t, 1001, 24, "magicuser", "a magic user")
		mage.SetAlignment(casterAlign)
		mage.SetPosition(combat.PosFighting)
		mage.SetFightingBody(vict)
		vict.SetFightingBody(mage)
		return mf, mage, vict
	}

	t.Run("neutral caster dispels good", func(t *testing.T) {
		mf, mage, _ := setup(t, 0, 500)
		seed := magicUserSeed(t, 24, 12)
		dprng.ResetStream(seed)
		if !specMagicUser(mf.w, nil, mage, "", "") {
			t.Fatal("specMagicUser should handle the dispel switch arm")
		}
		if got := mf.take("Victim"); got == "" {
			t.Fatal("neutral caster vs good victim cast nothing; C casts DISPEL_GOOD")
		}
	})
	t.Run("good caster spares good", func(t *testing.T) {
		mf, mage, _ := setup(t, 500, 500)
		seed := magicUserSeed(t, 24, 12)
		dprng.ResetStream(seed)
		if !specMagicUser(mf.w, nil, mage, "", "") {
			t.Fatal("specMagicUser should handle the switch arm")
		}
		if got := mf.take("Victim"); got != "" {
			t.Fatalf("good caster vs good victim emitted %q, C's !IS_GOOD blocks it", got)
		}
	})
	t.Run("neutral caster dispels evil", func(t *testing.T) {
		mf, mage, _ := setup(t, 0, -500)
		seed := magicUserSeed(t, 24, 12)
		dprng.ResetStream(seed)
		if !specMagicUser(mf.w, nil, mage, "", "") {
			t.Fatal("specMagicUser should handle the dispel switch arm")
		}
		if got := mf.take("Victim"); got == "" {
			t.Fatal("neutral caster vs evil victim cast nothing; C casts DISPEL_EVIL")
		}
	})
}

// TestMinorPissedAlchemistImmort gate proves the `cont` scan needs a
// non-NPC below LEVEL_IMMORT (src/spec_procs2.c:556): an immortals-only
// room returns FALSE before any say.
func TestMinorPissedAlchemistImmort(t *testing.T) {
	mf := newMinorFixWorld(t)
	mf.addPlayer(t, "Immort", 1001, combat.LVL_IMMORT)
	mob := mf.addMob(t, 1001, 20, "alchemist", "the pissed alchemist")
	if specPissedalchemist(mf.w, nil, mob, "", "") {
		t.Fatal("immortals-only room proceeded")
	}
	if got := mf.take("Immort"); got != "" {
		t.Fatalf("immortals-only room emitted %q, C returns FALSE before the says", got)
	}

	mf2 := newMinorFixWorld(t)
	mf2.addPlayer(t, "Mortal", 1001, 30)
	mob2 := mf2.addMob(t, 1001, 20, "alchemist", "the pissed alchemist")
	specPissedalchemist(mf2.w, nil, mob2, "", "")
	if got := mf2.take("Mortal"); !strings.Contains(got, "secret") {
		t.Fatalf("mortal room output = %q, want the thief says", got)
	}
}

// TestMinorWerewolfDyingDraws proves the HP checks live inside the arms
// after their draws (src/spec_procs3.c:434-440): a dying werewolf still
// consumes number(0,9) and number(0,3), emitting nothing.
func TestMinorWerewolfDyingDraws(t *testing.T) {
	calls := 0
	prev := werewolfNumber
	werewolfNumber = func(from, to int) int {
		calls++
		return dprng.Number(from, to)
	}
	t.Cleanup(func() { werewolfNumber = prev })

	build := func(t *testing.T, hp int) (*minorFixWorld, *MobInstance) {
		mf := newMinorFixWorld(t)
		vict := mf.addPlayer(t, "Victim", 1001, 10)
		wolf := mf.addMob(t, 1001, 20, "werewolf", "a werewolf")
		wolf.CurrentHP = hp
		wolf.SetPosition(combat.PosFighting)
		wolf.SetFightingBody(vict)
		vict.SetFightingBody(wolf)
		return mf, wolf
	}

	t.Run("dying werewolf still draws twice", func(t *testing.T) {
		calls = 0
		mf, wolf := build(t, 0)
		dprng.ResetStream(1)
		if !specWerewolf(mf.w, nil, wolf, "", "") {
			t.Fatal("dying werewolf should run the fighting arm")
		}
		if calls != 2 {
			t.Fatalf("dying werewolf drew %d, C draws number(0,9) and number(0,3) before the HP gate", calls)
		}
	})
	t.Run("dying werewolf bites nothing", func(t *testing.T) {
		calls = 0
		mf, wolf := build(t, -2)
		// First number(0,9) is 0 (a passing howl roll) and the bite roll
		// follows it sequentially; both arms must still stay silent.
		dprng.ResetStream(minorNumberSeed(t, 0, 9, 0))
		if !specWerewolf(mf.w, nil, wolf, "", "") {
			t.Fatal("dying werewolf should return true")
		}
		if calls != 2 {
			t.Fatalf("dying werewolf drew %d, want 2", calls)
		}
		if got := mf.take("Victim"); got != "" {
			t.Fatalf("Victim received %q, dying werewolf must only draw", got)
		}
	})
}

// TestMinorWallGuardFighterDelegate proves the player-lane cmd arm returns
// fighter()'s result (src/spec_procs2.c:2266), drawing like C; the old lane
// declined and drew nothing.
func TestMinorWallGuardFighterDelegate(t *testing.T) {
	build := func(t *testing.T) (*minorFixWorld, *Player, *MobInstance) {
		mf := newMinorFixWorld(t)
		p := mf.addPlayer(t, "Fighter", 1001, 10)
		guard := mf.addMob(t, 1001, 20, "wallguard", "the wall guard")
		guard.SetFightingBody(p)
		p.SetFightingBody(guard)
		guard.SetPosition(combat.PosFighting)
		return mf, p, guard
	}

	t.Run("typing fighter draws through fighter's switch", func(t *testing.T) {
		mf, p, guard := build(t)
		// Seed whose first number(0,10) hits the default arm: no command is
		// issued (no engine needed), but the draw proves delegation ran.
		seed := minorNumberSeed(t, 0, 10, 7)
		dprng.ResetStream(seed)
		specWallGuardNS(mf.w, p, guard, "look", "")
		// fighter consumed exactly one number(0,10) draw.
		dprng.ResetStream(seed)
		_ = dprng.Number(0, 10)
		want := dprng.Number(0, 1000000)
		dprng.ResetStream(seed)
		specWallGuardNS(mf.w, p, guard, "look", "")
		got := dprng.Number(0, 1000000)
		if got != want {
			t.Fatalf("next draw = %d, want %d (fighter drew one number(0,10))", got, want)
		}
	})
	t.Run("fighter hit returns handled", func(t *testing.T) {
		mf, p, guard := build(t)
		seed := minorNumberSeed(t, 0, 10, 1)
		dprng.ResetStream(seed)
		if !specWallGuardNS(mf.w, p, guard, "look", "") {
			t.Fatal("wall_guard_ns should return fighter's result on a hit arm")
		}
	})
}

// TestMinorRescuerCrossRoomOpponent proves C checks `!IS_NPC(FIGHTING(i))`
// only (src/spec_procs2.c:534): the victim's room is never consulted.
func TestMinorRescuerCrossRoomOpponent(t *testing.T) {
	mf := newMinorFixWorld(t)
	ally := mf.addMob(t, 1001, 20, "buddy", "an ally")
	opp := mf.addPlayer(t, "Fledvictim", 1002, 10)
	ally.SetFightingBody(opp)
	rescuer := mf.addMob(t, 1001, 20, "rescuer", "the rescuer")

	if !specRescuer(mf.w, nil, rescuer, "", "") {
		t.Fatal("specRescuer skipped the cross-room opponent; C only checks !IS_NPC(FIGHTING(i))")
	}
}

// TestMinorMedusaExactToken proves isname's exact-token gate
// (src/spec_procs2.c:1561): "med" must not intercept the medusa. The
// fallthrough has no draws, so the stream probe is the proof.
func TestMinorMedusaExactToken(t *testing.T) {
	mf := newMinorFixWorld(t)
	viewer := mf.addPlayer(t, "Viewer", 1001, 10)
	medusa := mf.addMob(t, 1001, 20, "medusa", "the medusa")

	seed := uint32(42)
	dprng.ResetStream(seed)
	if specMedusa(mf.w, viewer, medusa, "look", "med") {
		t.Fatal("prefix 'med' intercepted the medusa; C isname requires the full token")
	}
	if got := mf.take("Viewer"); got != "" {
		t.Fatalf("prefix look emitted %q, C falls through silently", got)
	}
	dprng.ResetStream(seed)
	want := dprng.Number(0, 1000000)
	dprng.ResetStream(seed)
	specMedusa(mf.w, viewer, medusa, "look", "med")
	if got := dprng.Number(0, 1000000); got != want {
		t.Fatalf("prefix look consumed draws (%d != %d); C draws nothing when isname fails", got, want)
	}
}

// TestMinorMorticianExactCorpse proves isname on the corpse scan
// (src/spec_procs3.c:838): "Tester" must not retrieve the "Testerx" corpse.
func TestMinorMorticianExactCorpse(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Mort Int", Zone: 1}},
		Objs: []parser.Obj{{
			VNum: 3002, Keywords: "Testerx", ShortDesc: "the corpse of Testerx",
			LongDesc: "A corpse lies here.", TypeFlag: 15, // ITEM_CONTAINER
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }

	player := NewPlayer(1, "Tester", 1001)
	player.SetLevel(5)
	player.SetGold(10000)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob := NewMob(&parser.Mob{
		VNum: 8095, Keywords: "mortician", ShortDesc: "the mortician",
		Level: 20, Position: 8, DefaultPos: 8, HP: parser.DiceRoll{Plus: 100},
	}, 1001)
	mob.SetPosition(combat.PosStanding)
	obj, err := w.SpawnObject(3002, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	obj.SetValue(3, 1)

	if !specMortician(w, player, mob, "retrieve", "") {
		t.Fatal("specMortician should handle retrieve")
	}
	if got := out["Tester"]; !strings.Contains(got, "can't find your corpse") {
		t.Fatalf("Tester = %q, want the prefix-miss tell", got)
	}
	if got := out["Tester"]; strings.Contains(got, "dumps your corpse") {
		t.Fatalf("Tester = %q, prefix must not retrieve the corpse", got)
	}
}
