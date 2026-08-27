package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestNewMobAppliesLoadpos pins C read_mobile's prototype copy (db.c:1757
// *mob = mob_proto[i]): the mob file's loadpos is the spawn position. A
// sleeping loader must spawn asleep — mobile_activity's position gates (the
// !AWAKE skip and the wander gate's POS_STANDING check) then keep it in
// place, exactly like C.
func TestNewMobAppliesLoadpos(t *testing.T) {
	cases := []struct {
		name    string
		loadpos int
		want    int
	}{
		{"sleeping", 4, combat.PosSleeping},
		{"resting", 5, combat.PosResting},
		{"sitting", 6, combat.PosSitting},
		{"standing", 8, combat.PosStanding},
		// C clear_char defaults position to POS_STANDING before the file
		// line overwrites it (utils.c:2997), so a record without the line
		// never spawns at POS_DEAD.
		{"unset defaults to standing", 0, combat.PosStanding},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mob := NewMob(&parser.Mob{VNum: 2109, Keywords: "sleeping orc worker", Position: tc.loadpos}, 8105)
			if got := mob.GetPosition(); got != tc.want {
				t.Errorf("spawn position = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestNewMobCarriesActFlags pins the round-14 fix: a mob whose record carries
// MOB_AWARE (harbor guard 18223, act=6232 = bits 1,3,4,6,12) must expose it
// through HasMobFlag — C's backstab aware-guard reads this bit.
func TestNewMobCarriesActFlags(t *testing.T) {
	proto := &parser.Mob{
		VNum: 18223, Keywords: "harbor guard", Position: 8,
		ActionFlags: []string{"SENTINEL", "ISNPC", "AWARE", "STAY_ZONE", "HELPER"},
	}
	mob := NewMob(proto, 8105)
	for _, want := range []struct {
		bit  int
		name string
	}{
		{4, "AWARE"}, {1, "SENTINEL"}, {6, "STAY_ZONE"}, {12, "HELPER"},
	} {
		if !mob.HasMobFlag(want.bit) {
			t.Errorf("HasMobFlag(%s) = false, want true", want.name)
		}
	}
	if mob.HasMobFlag(5) {
		t.Error("HasMobFlag(AGGRESSIVE) = true, want false")
	}
}
