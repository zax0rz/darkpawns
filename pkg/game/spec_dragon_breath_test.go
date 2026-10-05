package game

import (
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
	w.StopAITicker()
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 24)
	mob.VNum = 4209     // C's frost-breath arm.
	player.SetLevel(10) // no low-level damage protection
	mob.SetPosition(combat.PosFighting)
	mob.SetFightingBody(player)
	lastMsg()

	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	cb := w.WireCombatCallbacks()
	messages := loadMessagesFile(t)
	combat.InitFightMessages(cb, messages)
	combat.SetCallbacks(cb)
	variants, ok := messages.Variants(spells.SpellFrostBreath)
	if !ok || len(variants) == 0 {
		t.Fatal("frost breath messages missing")
	}
	message := cb.SkillMessage
	breaths := 0
	cb.SkillMessage = func(dam int, attacker, victim string, attackType, room int) bool {
		if attackType == spells.SpellFrostBreath {
			breaths++
			if dam != 0 || attacker != mob.GetName() || victim != player.Name || room != 1001 {
				t.Errorf("breath boundary: %d %q %q %d", dam, attacker, victim, room)
			}
		}
		return message(dam, attacker, victim, attackType, room)
	}
	// A failed breath roll consumes only number(0,3).
	failSeed := uint32(1)
	failRNG := dprng.New(failSeed)
	if failRNG.Number(0, 3) == 0 {
		t.Fatal("failed seed no longer rejects breath")
	}
	wantFailNext := failRNG.Number(0, 999)
	dprng.ResetStream(failSeed)
	if !specDragonBreath(w, nil, mob, "", "") || breaths != 0 || lastMsg() != "" || player.GetFighting() != "" {
		t.Fatal("failed breath roll changed state/output or shared return")
	}
	if got := dprng.Number(0, 999); got != wantFailNext {
		t.Fatalf("failed breath next draw = %d want %d", got, wantFailNext)
	}

	// Seed the success arm, then choose magic_user roll 12 with neutral
	// alignment, so the shared tail returns TRUE without another spell.
	// C draws: breath roll; mag_areas SAVING_SPELL; message selector;
	// magic_user victim choice; magic_user spell choice (spec_procs.c:418-452).
	var seed uint32
	var wantNext int
	for candidate := uint32(1); candidate < 10000; candidate++ {
		rng := dprng.New(candidate)
		if rng.Number(0, 3) != 0 {
			continue
		}
		rng.Number(0, 99)
		rng.Dice(1, len(variants))
		rng.Number(0, 4)
		if rng.Number(0, 12)+12 != 12 {
			continue
		}
		seed, wantNext = candidate, rng.Number(0, 999)
		break
	}
	if seed == 0 {
		t.Fatal("no success seed")
	}
	hp := player.GetHP()
	dprng.ResetStream(seed)
	if !specDragonBreath(w, nil, mob, "", "") {
		t.Fatal("successful breath must return shared magic_user result")
	}
	if breaths != 1 || player.GetFighting() != mob.GetName() || player.GetHP() != hp {
		t.Fatalf("breath effect: calls=%d fighting=%q HP=%d want one zero-damage breath and combat enrollment", breaths, player.GetFighting(), player.GetHP())
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("successful breath draw sequence: next=%d want=%d seed=%d", got, wantNext, seed)
	}
	t.Logf("breath success seed=%d next=%d", seed, wantNext)
}

func TestSpecDragonBreath_StandingRecovery(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.SetPosition(combat.PosSitting)
	mob.SetFightingBody(player)
	player.SetFightingBody(mob)
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
