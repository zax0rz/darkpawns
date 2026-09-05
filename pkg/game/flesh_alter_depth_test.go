package game

import (
	"slices"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func fleshAlterSuccessSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 1000; seed++ {
		if dprng.New(seed).Number(0, 111) <= 100 {
			return seed
		}
	}
	t.Fatal("could not find a flesh-alter success seed")
	return 0
}

func TestFleshAlterWeaponBands(t *testing.T) {
	for _, tc := range []struct {
		level int
		want  string
	}{
		{1, "studded wooden club"},
		{3, "studded wooden club"},
		{4, "razor-sharp dagger"},
		{6, "razor-sharp dagger"},
		{7, "steel-shafted axe"},
		{12, "studded steel mace"},
		{15, "battle flail"},
		{18, "steel-shafted battle axe"},
		{21, "double-headed battle axe"},
		{24, "studded morning-star"},
		{27, "gleaming broad sword"},
		{29, "gleaming long sword"},
		{30, "gleaming scythe"},
		{40, "gleaming scythe"},
	} {
		if got := fleshAlterWeapon(tc.level); got != tc.want {
			t.Errorf("fleshAlterWeapon(%d) = %q, want %q", tc.level, got, tc.want)
		}
	}
}

func TestDoFleshAlterToggleAndUnequip(t *testing.T) {
	seed := fleshAlterSuccessSeed(t)
	ch := NewPlayer(1, "Flesh", 1001)
	ch.Level = 1
	ch.Position = combat.PosFighting
	ch.SetFighting("target")
	ch.SetSkill(SkillFleshAlter, 100)
	ch.SetHitroll(4)
	ch.SetDamroll(6)

	dagger := NewObjectInstance(&parser.Obj{
		VNum: 9001, Keywords: "dagger", ShortDesc: "a dagger",
		TypeFlag: 5, WearFlags: [4]int{1 << 13},
	}, -1)
	if err := ch.Inventory.AddItem(dagger); err != nil {
		t.Fatalf("add dagger: %v", err)
	}
	if err := ch.Equipment.Equip(dagger, ch.Inventory); err != nil {
		t.Fatalf("equip dagger: %v", err)
	}

	dprng.ResetStream(seed)
	on := DoFleshAlter(ch)
	if !on.Success || !ch.IsAffected(affFleshAlter) {
		t.Fatalf("flesh alter on result/state = %+v/%v", on, ch.IsAffected(affFleshAlter))
	}
	if got, want := ch.GetHitroll(), 5; got != want {
		t.Errorf("hitroll after alter = %d, want %d", got, want)
	}
	if got, want := ch.GetDamroll(), 7; got != want {
		t.Errorf("damroll after alter = %d, want %d", got, want)
	}
	if got, want := on.MessageToCh, "You stop using a dagger.\r\nYour hand turns into a studded wooden club!"; got != want {
		t.Errorf("on actor message = %q, want %q", got, want)
	}
	if got, want := on.MessageToRoom, "Flesh stops using a dagger.\r\nFlesh's hand turns into a studded wooden club!"; got != want {
		t.Errorf("on room message = %q, want %q", got, want)
	}
	if _, equipped := ch.Equipment.GetItemInSlot(SlotWield); equipped {
		t.Fatal("flesh alter should unequip the wielded item")
	}
	if _, carried := ch.Inventory.FindItem("dagger"); !carried {
		t.Fatal("unequipped dagger should return to inventory")
	}

	dprng.ResetStream(seed)
	off := DoFleshAlter(ch)
	if !off.Success || ch.IsAffected(affFleshAlter) {
		t.Fatalf("flesh alter off result/state = %+v/%v", off, ch.IsAffected(affFleshAlter))
	}
	if got, want := ch.GetHitroll(), 4; got != want {
		t.Errorf("hitroll after revert = %d, want %d", got, want)
	}
	if got, want := ch.GetDamroll(), 6; got != want {
		t.Errorf("damroll after revert = %d, want %d", got, want)
	}
	if got, want := off.MessageToCh, "You shift your molecules back to normal.\r\nYour hand reverts from a studded wooden club."; got != want {
		t.Errorf("off actor message = %q, want %q", got, want)
	}
	if got, want := off.MessageToRoom, "Flesh's hand reverts from a studded wooden club!"; got != want {
		t.Errorf("off room message = %q, want %q", got, want)
	}
}

func TestDoFleshAlterFailureWaitAndImprovement(t *testing.T) {
	var seed uint32
	for candidate := uint32(1); candidate < 1000; candidate++ {
		if dprng.New(candidate).Number(0, 111) > 1 {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("could not find a flesh-alter failure seed")
	}

	ch := NewPlayer(1, "Flesh", 1001)
	ch.Position = combat.PosFighting
	ch.SetFighting("target")
	ch.SetSkill(SkillFleshAlter, 1)
	dprng.ResetStream(seed)
	result := DoFleshAlter(ch)

	if result.Success {
		t.Fatal("flesh alter should fail on the selected seed")
	}
	if result.MessageToCh != "You lose your concentration!" {
		t.Errorf("failure message = %q", result.MessageToCh)
	}
	if result.WaitCh != 2 {
		t.Errorf("failure wait = %d, want 2", result.WaitCh)
	}
	if !slices.Equal(result.DeferredImprove, []string{SkillFleshAlter}) {
		t.Errorf("failure deferred improvement = %v", result.DeferredImprove)
	}
	if ch.IsAffected(affFleshAlter) {
		t.Error("failed flesh alter must not apply the affect")
	}
}
