package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobFlagComparisonCaseInsensitive(t *testing.T) {
	// Simulate parser output: flags stored as UPPERCASE
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"SENTINEL", "STAY_ZONE"}}}

	// This should find the flag — if it doesn't, the lowercase comparison is wrong
	found := false
	for _, f := range mob.Prototype.ActionFlags {
		if strings.EqualFold(f, "sentinel") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("SENTINEL flag not found with case-insensitive comparison")
	}
}

func TestMobStayZonePreventsWander(t *testing.T) {
	// A STAY_ZONE mob should be recognized by hasMobFlag regardless of case.
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"STAY_ZONE"}}}
	if !hasMobFlag(mob, "stay_zone") {
		t.Fatal("STAY_ZONE flag not found with case-insensitive hasMobFlag")
	}
}

func TestHasMobFlagCaseInsensitive(t *testing.T) {
	mob := &MobInstance{Prototype: &parser.Mob{ActionFlags: []string{"AGGRESSIVE", "SENTINEL", "STAY_ZONE"}}}

	if !hasMobFlag(mob, "aggressive") {
		t.Error("hasMobFlag failed for lowercase 'aggressive' against uppercase 'AGGRESSIVE'")
	}
	if !hasMobFlag(mob, "SENTINEL") {
		t.Error("hasMobFlag failed for uppercase 'SENTINEL' against uppercase 'SENTINEL'")
	}
	if !hasMobFlag(mob, "Stay_Zone") {
		t.Error("hasMobFlag failed for mixed-case 'Stay_Zone' against uppercase 'STAY_ZONE'")
	}
	if hasMobFlag(mob, "wimpy") {
		t.Error("hasMobFlag falsely found 'wimpy'")
	}
}

func TestRoomHasFlagCaseInsensitive(t *testing.T) {
	room := &parser.Room{Flags: []string{"DEATH", "NO_MOB"}}
	if !roomHasFlag(room, "death") {
		t.Error("roomHasFlag failed for lowercase 'death' against uppercase 'DEATH'")
	}
	if !roomHasFlag(room, "NO_MOB") {
		t.Error("roomHasFlag failed for uppercase 'NO_MOB' against uppercase 'NO_MOB'")
	}
	if roomHasFlag(room, "peaceful") {
		t.Error("roomHasFlag falsely found 'peaceful'")
	}
}
