package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type cityguardTestCombatEngine struct {
	starts         [][2]string
	initialAttacks [][2]string
}

func (e *cityguardTestCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *cityguardTestCombatEngine) PerformInitialAttack(attacker, defender combat.Combatant) error {
	e.initialAttacks = append(e.initialAttacks, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *cityguardTestCombatEngine) IsFighting(string) bool { return false }

func (e *cityguardTestCombatEngine) GetCombatTarget(string) (combat.Combatant, bool) {
	return nil, false
}

func TestSpecCityguard_EntryGates(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)

	if specCityguard(w, player, guard, "look", "") {
		t.Fatal("cityguard should reject command dispatch")
	}

	guard.SetStatus("sleeping")
	if specCityguard(w, nil, guard, "", "") {
		t.Fatal("cityguard should reject a sleeping mob")
	}
}

func TestSpecCityguard_OutlawVisibility(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	_ = lastMsg()
	player.SetPlrFlag(PlrOutlaw, true)

	guard.SetAffected(affBlind)
	if specCityguard(w, nil, guard, "", "") {
		t.Fatal("blind cityguard should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("blind cityguard emitted %q", got)
	}

	guard.RemoveAffected(affBlind)
	specCityguard(w, nil, guard, "", "")
	if got := lastMsg(); !strings.Contains(got, "A test mob says, 'We don't like OUTLAWS like you in this city!'") {
		t.Fatalf("outlaw warning = %q", got)
	}
}

func TestSpecCityguard_ProtectionSelectionAndHitBoundary(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, 1001, 20)
	protected := NewPlayer(2, "Protected", 1001)
	protected.SetAlignment(100)
	if err := w.AddPlayer(protected); err != nil {
		t.Fatalf("AddPlayer protected: %v", err)
	}

	lessEvil := newSpecProcTestMob(t, w, 1001, 10)
	lessEvil.Prototype.Alignment = -100
	lessEvil.Prototype.ShortDesc = "a less evil mob"
	lessEvil.SetFighting(protected.GetName())
	mostEvil := newSpecProcTestMob(t, w, 1001, 10)
	mostEvil.Prototype.Alignment = -500
	mostEvil.Prototype.ShortDesc = "the most evil mob"
	mostEvil.SetFighting(protected.GetName())

	engine := &cityguardTestCombatEngine{}
	w.SetCombatEngine(engine)
	specCityguard(w, nil, guard, "", "")

	if got, want := len(engine.starts), 1; got != want {
		t.Fatalf("combat starts = %d, want %d", got, want)
	}
	if got, want := engine.starts[0][1], mostEvil.GetName(); got != want {
		t.Fatalf("selected attacker = %q, want lowest alignment %q", got, want)
	}
	if got, want := len(engine.initialAttacks), 1; got != want {
		t.Fatalf("initial attacks = %d, want %d", got, want)
	}
	if got, want := engine.initialAttacks[0][1], mostEvil.GetName(); got != want {
		t.Fatalf("initial hit target = %q, want %q", got, want)
	}
	if got := lastMsg(); !strings.Contains(got, "You just pissed me off") {
		t.Fatalf("protection warning = %q", got)
	}
}
