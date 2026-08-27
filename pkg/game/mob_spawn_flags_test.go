package game

import (
	"os"
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

// TestNewMobSeedsInnateAffects pins the affected-by half of C read_mobile's
// prototype copy (db.c:1757): the mob file's AFF words ride onto the instance
// as innate bits in C struct positions, and mag_affects' mob-affection gate
// (magic.c:1387-1394) refuses spells whose bitvector the mob holds innately.
// The names round-trip through the REAL parser so the game and parser tables
// cannot drift apart.
func TestNewMobSeedsInnateAffects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/163.mob"
	record := "#16303\ntestmob\ta\ttest mob\nA test mob is here.\n~\nDetail.\n~\n" +
		"10 0 0 0 128 0 0 0 0 E\n1 19 9 1d5+20 1d4+0\n3 100\n8 8 1\nE\n$\n"
	if err := os.WriteFile(path, []byte(record), 0o600); err != nil {
		t.Fatalf("write fixture mob file: %v", err)
	}
	mobs, err := parser.ParseMobFile(path)
	if err != nil {
		t.Fatalf("ParseMobFile: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("parsed %d mobs, want 1", len(mobs))
	}
	proto := &mobs[0]
	if len(proto.AffectFlags) != 1 || proto.AffectFlags[0] != "SANCTUARY" {
		t.Fatalf("AffectFlags = %v, want [SANCTUARY] (word 5 carries AFF bits)", proto.AffectFlags)
	}
	mob := NewMob(proto, 8105)
	if !mob.IsAffected(7) { // C AFF_SANCTUARY, structs.h
		t.Errorf("mob.IsAffected(7) = false, want innate sanctuary from the mob file")
	}
	if mob.IsAffected(0) {
		t.Errorf("mob.IsAffected(0) = true, want no innate blind bit")
	}
}
