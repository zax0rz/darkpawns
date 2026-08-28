package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func TestDragonBreathSpellSelection(t *testing.T) {
	tests := []struct {
		vnum int
		want int
	}{
		{vnum: 4209, want: spells.SpellFrostBreath},
		{vnum: 4705, want: spells.SpellFrostBreath},
		{vnum: 11000, want: spells.SpellAcidBreath},
		{vnum: 11001, want: spells.SpellLightningBreath},
		{vnum: 11002, want: spells.SpellFireBreath},
		{vnum: 20027, want: spells.SpellLightningBreath},
		{vnum: 99999, want: spells.SpellFireBreath},
	}
	for _, tt := range tests {
		if got := dragonBreathSpell(tt.vnum); got != tt.want {
			t.Errorf("dragonBreathSpell(%d) = %d, want %d", tt.vnum, got, tt.want)
		}
	}
}

func TestSpecDragonBreath_EntryGatesAndRoomThreat(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	lastMsg() // discard spawn announcement

	if specDragonBreath(w, nil, mob, "look", "") {
		t.Fatal("dragon breath should reject a non-empty command")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("command gate output = %q, want empty", got)
	}

	mob.SetPosition(combat.PosSleeping)
	if specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("dragon breath should reject a sleeping mob")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("sleeping gate output = %q, want empty", got)
	}

	mob.SetPosition(combat.PosStanding)
	mob.CurrentHP = -1
	if specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("dragon breath should reject a negative-HP mob")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("negative-HP gate output = %q, want empty", got)
	}

	mob.CurrentHP = 50
	mob.SetAffected(affBlind)
	if specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("blind dragon should not select a room victim")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("blind gate output = %q, want empty", got)
	}

	mob.RemoveAffected(affBlind)
	player.SetPlrFlag(PrfNohassle, true)
	if specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("nohassle player should not be selected")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("nohassle gate output = %q, want empty", got)
	}

	player.SetPlrFlag(PrfNohassle, false)
	if !specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("eligible room victim should be handled")
	}
	if got, want := lastMsg(), "A test mob looks at you.\r\nA test mob growls, 'So, you have found my lair...'\r\nA test mob exclaims, 'For that you must die!'\r\n"; got != want {
		t.Fatalf("room threat = %q, want %q", got, want)
	}
}

func TestSpecDragonBreath_CombatRollAndSharedReturn(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	lastMsg()

	var failSeed uint32
	for seed := uint32(1); seed < 10000; seed++ {
		rng := dprng.New(seed)
		if rng.Number(0, 3) != 0 {
			failSeed = seed
			break
		}
	}
	if failSeed == 0 {
		t.Fatal("could not find a seed for dragon breath's failed combat roll")
	}
	dprng.ResetStream(failSeed)
	if !specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("failed combat breath roll should still consume the special")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("failed combat breath invented output = %q", got)
	}

	var successSeed uint32
	for seed := uint32(1); seed < 10000; seed++ {
		rng := dprng.New(seed)
		if rng.Number(0, 3) == 0 {
			successSeed = seed
			break
		}
	}
	if successSeed == 0 {
		t.Fatal("could not find a seed for dragon breath's successful combat roll")
	}
	dprng.ResetStream(successSeed)
	if !specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("successful combat breath should return the shared magic_user result")
	}
	if got := lastMsg(); strings.Contains(got, "breathes") {
		t.Fatalf("combat breath invented output = %q", got)
	}
}

func TestSpecDragonBreath_StandingRecovery(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.SetPosition(combat.PosSitting)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	lastMsg()

	if !specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("standing recovery should consume the special")
	}
	if got, want := mob.GetPosition(), combat.PosStanding; got != want {
		t.Fatalf("dragon position after do_stand = %d, want %d", got, want)
	}
	if got, want := lastMsg(), "A test mob clambers to its feet.\r\n"; got != want {
		t.Fatalf("standing recovery room output = %q, want %q", got, want)
	}
}
