package game

// spec_fearface_test.go — fearface port cover (src/spec_procs.c:2171-2194).
// C assigns this special nowhere (src/spec_assign.c declares it without
// ASSIGNMOB), so direct calls carry the proof; each behavior test fails on
// the former always-false stub.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// fearSaveSeed.find a stream whose first number(0,99) makes a level-1
// warrior fail (wantPanic) or pass a SAVING_SPELL throw: the warrior
// threshold is 85, so the throw saves exactly when roll > 85.
func fearSaveSeed(t *testing.T, wantPanic bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 1000; seed++ {
		roll := dprng.New(seed).Number(0, 99)
		if (roll <= 85) == wantPanic {
			return seed
		}
	}
	t.Fatal("no deterministic fear save seed found")
	return 0
}

// fearfaceWorld returns a world with a two-room map (so do_flee has
// somewhere to go), a fighting fear mob, and one player.
func fearfaceWorld(t *testing.T) (*World, *MobInstance, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Bear Room", Zone: 1, Exits: map[string]parser.Exit{"east": {Direction: "east", ToRoom: 1002}}},
		{VNum: 1002, Name: "East Room", Zone: 1, Exits: map[string]parser.Exit{"west": {Direction: "west", ToRoom: 1001}}},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	victim := NewPlayer(1, "Fearvictim", 1001)
	victim.SetClass(ClassWarrior)
	victim.SetLevel(1)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob.SetFightingBody(victim)
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	return w, mob, victim, &out
}

func TestSpecFearfacePanicsOnFailedSave(t *testing.T) {
	w, mob, victim, out := fearfaceWorld(t)
	dprng.ResetStream(fearSaveSeed(t, true))
	if !specFearface(w, victim, mob, "", "") {
		t.Fatal("specFearface did not handle the failed save")
	}
	// src/spec_procs.c:2190: the stare line, C's TO_ROOM audience.
	if got := out.String(); !strings.Contains(got, "stares into the eyes of") {
		t.Fatalf("room output = %q, want the stare line", got)
	}
}

func TestSpecFearfaceSavedVictimEndsTheWalk(t *testing.T) {
	// a saved victim still commits (C sets victim before the save), so a
	// save returns FALSE and no second target is considered.
	w, mob, victim, out := fearfaceWorld(t)
	dprng.ResetStream(fearSaveSeed(t, false))
	if specFearface(w, victim, mob, "", "") {
		t.Fatal("saved fear victim should not be panicked")
	}
	if got := out.String(); got != "" {
		t.Fatalf("saved fear output = %q, want empty", got)
	}
}

func TestSpecFearfaceNohassleVictimIsSkipped(t *testing.T) {
	w, mob, victim, out := fearfaceWorld(t)
	victim.SetPlrFlag(PrfNohassle, true)
	dprng.ResetStream(fearSaveSeed(t, true))
	if specFearface(w, victim, mob, "", "") {
		t.Fatal("nohassle victim should be skipped")
	}
	if got := out.String(); got != "" {
		t.Fatalf("nohassle output = %q, want empty", got)
	}
}

func TestSpecFearfaceSeatedMobStandsInstead(t *testing.T) {
	// src/spec_procs.c:2179: posture between sleeping and fighting stands
	// first; the scare does not run on that pulse.
	w, mob, victim, _ := fearfaceWorld(t)
	mob.SetPosition(combat.PosResting)
	dprng.ResetStream(fearSaveSeed(t, true))
	if specFearface(w, victim, mob, "", "") {
		t.Fatal("seated fearface should stand, not scare")
	}
	if got := mob.GetPosition(); got != combat.PosStanding {
		t.Fatalf("fearface position = %d, want standing", got)
	}
}

func TestSpecFearfaceGatesDecline(t *testing.T) {
	w, mob, victim, _ := fearfaceWorld(t)
	// FIGHTING(ch) is required (src/spec_procs.c:2178).
	idle := newSpecProcTestMob(t, w, 1001, 10)
	if specFearface(w, victim, idle, "", "") {
		t.Fatal("non-fighting fearface should decline")
	}
	// A typed command always declines (C requires cmd == 0).
	if specFearface(w, victim, mob, "south", "") {
		t.Fatal("commanded fearface should decline")
	}
	// A sleeping fearface declines.
	sleeper := newSpecProcTestMob(t, w, 1001, 10)
	sleeper.SetPosition(combat.PosSleeping)
	sleeper.SetFightingBody(victim)
	if specFearface(w, victim, sleeper, "", "") {
		t.Fatal("sleeping fearface should decline")
	}
}
