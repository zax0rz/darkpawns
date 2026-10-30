package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func dyingArmWorld(t *testing.T, mobVNum int, shortDesc string) (*World, *MobInstance, *Player, func() string) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Arena", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      mobVNum,
			Keywords:  strings.ToLower(shortDesc),
			ShortDesc: shortDesc,
			LongDesc:  shortDesc + " is here.",
			Level:     2,
			HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	killer := NewPlayer(70, "Slayer", 1001)
	killer.SetLevel(10)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	mob, err := w.SpawnMob(mobVNum, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetMobFlag(MobFlagSpec)
	lastMsg()
	return w, mob, killer, lastMsg
}

// die_with_killer runs a spec-flagged victim's own special once more, with
// FIGHTING set to the killer (src/fight.c:590-594), before the HP<0 death
// tail. Both Go death paths used to skip the arm entirely (#2221 handoff).
// R5h: with the arm reverted, the puff test loses the dying line and the
// never_die test finds a corpse.
func TestHandleDeathDyingSpecArmPuffSaysLine(t *testing.T) {
	w, mob, killer, lastMsg := dyingArmWorld(t, 1, "Puff")
	mob.SetHealth(-15)

	w.HandleDeath(mob, killer, TypeSlash)

	if got := lastMsg(); !strings.Contains(got, "Shit, I'm dead.") {
		t.Fatalf("dying spec arm output = %q, want puff's dying line (spec_procs.c:619-623)", got)
	}
	if mob.IsAlive() {
		t.Error("puff should still die after the dying line")
	}
	corpses := 0
	for _, obj := range w.GetItemsInRoom(1001) {
		if obj.IsCorpse {
			corpses++
		}
	}
	if corpses != 1 {
		t.Errorf("corpse count = %d, want 1 (the death tail still ran)", corpses)
	}
}

// A dying never_die body heals to full and die_with_killer's HP<0 tail is
// canceled (src/fight.c:599; src/spec_procs2.c:2075-2090): no stop_fighting,
// no POS_DEAD, no corpse, and the body keeps fighting its killer.
func TestHandleDeathNeverDieCancelsDeathTail(t *testing.T) {
	w, mob, killer, _ := dyingArmWorld(t, 14401, "the Gatekeeper")
	maxHP := mob.GetMaxHP()
	mob.SetHealth(-15)

	w.HandleDeath(mob, killer, TypeSlash)

	if mob.GetHP() != maxHP {
		t.Fatalf("never_die healed to %d, want %d", mob.GetHP(), maxHP)
	}
	if mob.GetPosition() != combat.PosStanding {
		t.Fatalf("healed body position = %d, want standing", mob.GetPosition())
	}
	if !mob.IsAlive() {
		t.Error("never_die body must stay in the world")
	}
	if mob.GetFightingBody() != killer {
		t.Errorf("healed body fighting %v, want the killer (fight.c:592)", mob.GetFightingBody())
	}
	for _, obj := range w.GetItemsInRoom(1001) {
		if obj.IsCorpse {
			t.Error("never_die cancel must not leave a corpse")
		}
	}
}

func TestZzzCanaryTwo(t *testing.T) { t.Log("v2") }
