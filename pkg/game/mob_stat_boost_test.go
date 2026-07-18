package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestNewMobDoesNotReRollStatBoosts locks in Fix B: the level>15 random stat
// boosts must be applied ONCE, at parse/boot time on the prototype (matching C
// parse_simple_mob, db.c:1053-1062). read_mobile (db.c:1725-1776) only copies
// the prototype — it does not re-roll stats. A prior version of NewMob
// re-applied the boosts at spawn time, double-boosting stats and burning 6
// extra PRNG draws per high-level mob, which desynced the shared stream before
// character creation (the newbie stat-roll divergence).
//
// This test fails if anyone re-introduces a spawn-time stat boost in NewMob,
// detected via instance stats diverging from the already-boosted prototype.
func TestNewMobDoesNotReRollStatBoosts(t *testing.T) {
	// A high-level prototype whose stats have ALREADY been boosted at parse time
	// (simulating what parser/mob.go does). Spawn must copy these verbatim.
	proto := &parser.Mob{
		VNum:      99901,
		ShortDesc: "a high-level sentinel",
		Level:     30, // > 15, so the old bug's boost branch would have fired
		Str:       18,
		Int:       17,
		Wis:       16,
		Dex:       15,
		Con:       19,
		Cha:       14,
		HP:        parser.DiceRoll{Num: 0, Sides: 0, Plus: 100}, // no HP dice draws
		Gold:      0,                                            // no gold-variance draws
	}

	mob := NewMob(proto, 1200)

	// Instance ability scores must equal the prototype's already-boosted values.
	// A re-introduced spawn-time boost would mutate these away from proto.
	if mob.Str != proto.Str {
		t.Errorf("Str mutated at spawn: got %d, want %d (prototype already boosted)", mob.Str, proto.Str)
	}
	if mob.Intel != proto.Int {
		t.Errorf("Int mutated at spawn: got %d, want %d (prototype already boosted)", mob.Intel, proto.Int)
	}
	if mob.Wis != proto.Wis {
		t.Errorf("Wis mutated at spawn: got %d, want %d (prototype already boosted)", mob.Wis, proto.Wis)
	}
	if mob.Dex != proto.Dex {
		t.Errorf("Dex mutated at spawn: got %d, want %d (prototype already boosted)", mob.Dex, proto.Dex)
	}
	if mob.Con != proto.Con {
		t.Errorf("Con mutated at spawn: got %d, want %d (prototype already boosted)", mob.Con, proto.Con)
	}
	if mob.Cha != proto.Cha {
		t.Errorf("Cha mutated at spawn: got %d, want %d (prototype already boosted)", mob.Cha, proto.Cha)
	}
}

// TestNewMobLowLevelStatsEqualPrototype confirms low-level mobs (level <= 15)
// also copy prototype stats verbatim — C applies no boosts to them at any point.
func TestNewMobLowLevelStatsEqualPrototype(t *testing.T) {
	proto := &parser.Mob{
		VNum: 99902, ShortDesc: "a rat", Level: 3,
		Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
		HP: parser.DiceRoll{Num: 0, Sides: 0, Plus: 10},
	}
	mob := NewMob(proto, 1201)
	if mob.Str != proto.Str || mob.Con != proto.Con || mob.Wis != proto.Wis {
		t.Errorf("low-level mob stats diverged from prototype: got str/con/wis = %d/%d/%d, want %d/%d/%d",
			mob.Str, mob.Con, mob.Wis, proto.Str, proto.Con, proto.Wis)
	}
}
