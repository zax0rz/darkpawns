package game

import (
	"context"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newAmbushDepthWorld(t *testing.T) (*World, *Player, *MobInstance) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	// Drive the queue manually so the test pins exact game-pulse timing rather
	// than racing the real-time queue goroutine.
	w.EventQueue.Stop()
	ch.SetSkill(SkillAmbush, 100)
	target := spawnTargetMob(t, w)
	return w, ch, target
}

func TestAmbushDelayedActionAndLostPrey(t *testing.T) {
	w, ch, target := newAmbushDepthWorld(t)
	startHP := target.GetHP()

	w.PlanAmbush(ch, target)
	if ch.GetAmbushAction() == 0 {
		t.Fatal("PlanAmbush did not store the pending GET_ACTION equivalent")
	}
	if got := ch.GetWaitState(); got != 0 {
		t.Fatalf("ambush plan wait state = %d, want 0 before event resolution", got)
	}

	for i := 0; i < 39; i++ {
		w.EventQueue.Process(context.Background())
	}
	if ch.GetAmbushAction() == 0 {
		t.Fatal("ambush resolved before PULSE_VIOLENCE*2")
	}
	if target.GetHP() != startHP {
		t.Fatalf("target HP changed before ambush event: got %d, want %d", target.GetHP(), startHP)
	}

	ch.SetRoom(9999)
	w.EventQueue.Process(context.Background())
	if ch.GetAmbushAction() != 0 {
		t.Fatal("ambush action was not cleared at event entry")
	}
	if target.GetHP() != startHP {
		t.Fatalf("lost-prey event changed target HP: got %d, want %d", target.GetHP(), startHP)
	}
}

func TestAmbushDamageFormulaAndHiddenClear(t *testing.T) {
	w, ch, target := newAmbushDepthWorld(t)
	originalCallbacks := combat.GetCallbacks()
	defer combat.SetCallbacks(originalCallbacks)
	combat.SetCallbacks(w.WireCombatCallbacks())

	ch.Level = 10
	ch.Stats.Str = 18
	ch.SetDamroll(5)
	ch.SetAffect(affHide, true)
	weapon := NewObjectInstance(&parser.Obj{
		VNum:      9001,
		Keywords:  "dagger",
		ShortDesc: "a dagger",
		TypeFlag:  5,
		Values:    [4]int{0, 1, 1, 3},
		WearFlags: [4]int{1 << 13},
	}, -1)
	if err := ch.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add dagger: %v", err)
	}
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip dagger: %v", err)
	}
	target.SetHealth(1000)
	target.SetMaxHP(1000)

	seed := ambushSuccessSeed(t)
	dprng.ResetStream(seed)
	expectedRaw := ch.GetStrToDam() + ch.GetDamroll() + 1 + int(float64(ch.GetLevel())*2.6)
	expectedDamage := expectedRaw + int(float64(expectedRaw)*0.10)

	dprng.ResetStream(seed)
	w.resolveAmbush(ch, target, target.GetRoom())
	if got := target.GetHP(); got != 1000-expectedDamage {
		t.Fatalf("ambush damage = %d, want %d (raw %d with hidden bonus)", 1000-got, expectedDamage, expectedRaw)
	}
	if ch.IsAffected(affHide) {
		t.Fatal("ambush damage did not clear AFF_HIDE")
	}
	if got := target.GetWaitState(); got != 1 {
		t.Fatalf("target wait state = %d, want 1 round", got)
	}
	if got := ch.GetWaitState(); got != 20 {
		t.Fatalf("actor wait state = %d, want 20 pulses", got)
	}
}

func TestAmbushAwareForcesFailureAndStartsCombat(t *testing.T) {
	w, ch, target := newAmbushDepthWorld(t)
	target.Prototype.ActionFlags = []string{"AWARE"}
	startHP := target.GetHP()

	dprng.ResetStream(1)
	w.resolveAmbush(ch, target, target.GetRoom())
	if target.GetHP() != startHP {
		t.Fatalf("aware ambush changed target HP: got %d, want %d", target.GetHP(), startHP)
	}
	if got := ch.GetFighting(); got != target.GetName() {
		t.Fatalf("aware failure actor fighting = %q, want %q", got, target.GetName())
	}
	if got := target.GetFighting(); got != ch.GetName() {
		t.Fatalf("aware failure target fighting = %q, want %q", got, ch.GetName())
	}
	if got := target.GetWaitState(); got != 0 {
		t.Fatalf("aware failure target wait state = %d, want 0", got)
	}
	if got := ch.GetWaitState(); got != 20 {
		t.Fatalf("aware failure actor wait state = %d, want 20 pulses", got)
	}
}

func TestAmbushEventStopsWhenActorIsFighting(t *testing.T) {
	w, ch, target := newAmbushDepthWorld(t)
	startHP := target.GetHP()
	ch.SetFighting("another target")

	w.resolveAmbush(ch, target, target.GetRoom())
	if target.GetHP() != startHP {
		t.Fatalf("fighting actor's ambush changed target HP: got %d, want %d", target.GetHP(), startHP)
	}
	if got := ch.GetWaitState(); got != 0 {
		t.Fatalf("fighting actor ambush wait state = %d, want 0", got)
	}
}

func ambushSuccessSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 131) <= 100 {
			return seed
		}
	}
	t.Fatal("no deterministic ambush success seed found")
	return 0
}
