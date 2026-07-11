package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mockAreaWorld satisfies worldAoe (affect_spells.go) for castHellfire /
// castMeteorSwarm regression tests.
type mockAreaWorld struct {
	room  *parser.Room
	chars []interface{}
	calls []struct {
		Attacker, Victim interface{}
		Dam              int
		Skill            string
	}
}

func (w *mockAreaWorld) GetAllCharsInRoom(roomVNum int) []interface{} { return w.chars }
func (w *mockAreaWorld) GetRoomInWorld(vnum int) *parser.Room         { return w.room }
func (w *mockAreaWorld) DoSpellDamage(attacker, victim interface{}, dam int, skill string) bool {
	w.calls = append(w.calls, struct {
		Attacker, Victim interface{}
		Dam              int
		Skill            string
	}{attacker, victim, dam, skill})
	if d, ok := victim.(*mockSpellsChar); ok {
		newHP := d.hp - dam
		if newHP < 0 {
			newHP = 0
		}
		d.hp = newHP
	}
	return true
}

// ---------------------------------------------------------------------------
// DP-938: these tests cover the *real* ported area/multi-cast spells
// (castHellfire, castMeteorSwarm, castCalliope) that MagDamage's fabricated
// dice-table cases were duplicating/shadowing. They already exist and are
// wired through CallMagic -> RoutineManual -> ExecuteManualSpell, but had no
// test coverage before this ticket.
// ---------------------------------------------------------------------------

func TestCastHellfire_BlocksPeacefulRoom(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 30, roomVNum: 100}
	victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 200, hp: 200}
	// RoomPeaceful = 4 -> word = 4/16 = 0, bitPos = 4 -> value with bit 4 set = 16.
	world := &mockAreaWorld{
		room:  &parser.Room{Flags: []string{"16", "0", "0", "0"}},
		chars: []interface{}{caster, victim},
	}

	castHellfire(30, caster, world)

	if len(world.calls) != 0 {
		t.Errorf("expected no damage in a peaceful room, got %d DoSpellDamage calls", len(world.calls))
	}
	if victim.hp != 200 {
		t.Errorf("expected victim HP untouched, got %d", victim.hp)
	}
}

func TestCastHellfire_DealsAreaDamage_InstakillsLowLevel_SkipsCasterAndGroup(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 30, roomVNum: 100, inGroup: true}
	victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 200, hp: 200}
	weakling := &mockSpellsChar{name: "Weakling", level: 3, maxHP: 50, hp: 50}
	ally := &mockSpellsChar{name: "Ally", level: 30, maxHP: 200, hp: 200, inGroup: true, following: "Caster"}

	world := &mockAreaWorld{
		room:  &parser.Room{},
		chars: []interface{}{caster, victim, weakling, ally},
	}

	castHellfire(30, caster, world)

	// Victim (level > 4): dam = dice(12,5) + (2*30) - 10, bounded [12,60]+50 = [62,110].
	if victim.hp >= 200 || victim.hp < 200-110 {
		t.Errorf("expected victim to take hellfire damage in [90,138] HP remaining, got %d", victim.hp)
	}
	// Weakling (level <= 4): C instakills via GET_MAX_HIT*12.
	if weakling.hp != 0 {
		t.Errorf("DP-938: expected level<=4 victim to be instakilled by hellfire, got %d HP left", weakling.hp)
	}
	// Grouped ally must be skipped entirely.
	if ally.hp != 200 {
		t.Errorf("expected grouped ally to be skipped by hellfire, got %d HP left", ally.hp)
	}
}

func TestCastMeteorSwarm_DealsAreaDamage_SkipsImmortalsAndGroup(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 30, roomVNum: 100, inGroup: true}
	victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 500, hp: 500}
	immortal := &mockSpellsChar{name: "God", level: 100, maxHP: 5000, hp: 5000}
	ally := &mockSpellsChar{name: "Ally", level: 30, maxHP: 500, hp: 500, inGroup: true, following: "Caster"}

	world := &mockAreaWorld{
		room:  &parser.Room{},
		chars: []interface{}{caster, victim, immortal, ally},
	}

	castMeteorSwarm(30, caster, world)

	// dam = level*6 + rand.IntN(level*3+11) - 10 = 180 + [0,100] - 10, so the
	// damage range is [170,270] for level 30 (max when rand.IntN(101) == 100).
	// The old bound 500-269 flaked ~1% of runs on the max roll.
	if victim.hp >= 500 || victim.hp < 500-270 {
		t.Errorf("expected victim to take meteor swarm damage, got %d HP left", victim.hp)
	}
	if immortal.hp != 5000 {
		t.Errorf("DP-938: expected non-NPC immortal (level>=100) to be skipped by meteor swarm, got %d HP left", immortal.hp)
	}
	if ally.hp != 500 {
		t.Errorf("expected grouped ally to be skipped by meteor swarm, got %d HP left", ally.hp)
	}
}

// TestCastCalliope_FiresMultipleMissiles: castCalliope fires
// MAX(4, number(level/6, level*2)) magic missiles at the target. At level 0,
// lo=hi=0 so the MAX(4,...) floor makes this exactly 4 missiles — deterministic.
func TestCastCalliope_FiresMultipleMissiles(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 0, class: 99, position: int(PosFighting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, maxHP: 1000, hp: 1000, position: int(PosFighting)}

	castCalliope(0, caster, victim)

	// Each missile: dice(4,3)+level(0), bounded [4,12]. 4 missiles -> [16,48] before saves.
	// magSavingThrow can halve individual missile damage (min 1), so floor is 4×1 = 4.
	dealt := 1000 - victim.hp
	if dealt < 4 || dealt > 48 {
		t.Errorf("DP-938: expected ~4 magic missiles worth of damage (4-48 after saves), got %d", dealt)
	}
}

// mockCharmWorld drives MagAreas'/MagAffectsMass' charm-skip path (DP-1015).
// IsCharmedI mirrors game.World.IsCharmedI: it consults the target's internal
// charm bit (affCharm == index 21 in pkg/game/affects_constants.go).
type mockCharmWorld struct {
	chars []interface{}
}

func (w *mockCharmWorld) GetAllCharsInRoom(int) []interface{} { return w.chars }

func (w *mockCharmWorld) IsCharmedI(ch interface{}) bool {
	if c, ok := ch.(interface{ IsAffected(int) bool }); ok {
		return c.IsAffected(21) // affCharm bit index
	}
	return false
}

// TestMagAreas_SkipsCharmedNPC is the DP-1015 regression: area spells must skip
// charmed NPCs (C: mag_areas -> if (IS_NPC(tch) && IS_AFFECTED(tch, AFF_CHARM))
// continue;). The old code passed the engine AFF_CHARM mask as a bit index, so
// the guard was always false and charmed pets got hit.
func TestMagAreas_SkipsCharmedNPC(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 40, roomVNum: 100}
	pet := &mockSpellsChar{name: "Pet", npc: true, level: 10, maxHP: 200, hp: 200, aff: 1 << 21}
	enemy := &mockSpellsChar{name: "Enemy", npc: true, level: 10, maxHP: 200, hp: 200}
	world := &mockCharmWorld{chars: []interface{}{caster, pet, enemy}}

	MagAreas(40, caster, SpellEarthquake, 0, world)

	if pet.hp != 200 {
		t.Errorf("charmed pet must be skipped by area spell, but it took %d damage", 200-pet.hp)
	}
	if enemy.hp >= 200 {
		t.Errorf("non-charmed enemy should be damaged by area spell, hp still %d", enemy.hp)
	}
}
