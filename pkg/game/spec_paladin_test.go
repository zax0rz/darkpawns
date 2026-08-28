package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func preparePaladinCombat(t *testing.T, w *World, player *Player) *MobInstance {
	t.Helper()

	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 32)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.GetName())
	player.SetPosition(combat.PosFighting)
	player.SetFighting(mob.GetName())
	return mob
}

// TestSpecPaladin_Golden covers SPECIAL(paladin)'s actual autonomous entry
// gates and verifies that a fully eligible call is consumed even when its
// inclusive dispatch roll lands on a no-op case (spec_procs.c:537-568).
func TestSpecPaladin_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 32)

	if specPaladin(w, nil, mob, "look", "") {
		t.Error("specPaladin should return false for non-empty cmd")
	}
	mob.SetPosition(combat.PosFighting)
	if specPaladin(w, nil, mob, "", "") {
		t.Error("specPaladin should return false without FIGHTING(ch)")
	}

	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	mob.SetWaitState(1)
	if specPaladin(w, nil, mob, "", "") {
		t.Error("specPaladin should return false while GET_MOB_WAIT is nonzero")
	}

	mob.SetWaitState(0)
	mob.CurrentHP = -1
	if specPaladin(w, nil, mob, "", "") {
		t.Error("specPaladin should return false for negative HP")
	}

	mob.CurrentHP = 50
	dprng.ResetStream(1)
	if !specPaladin(w, nil, mob, "", "") {
		t.Error("eligible paladin special should consume the autonomous tick")
	}
}

func TestSpecPaladin_DefaultDispatch(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := preparePaladinCombat(t, w, player)

	var seed uint32
	for seed = 1; seed < 10000; seed++ {
		rng := dprng.New(seed)
		if rng.Number(0, 8) == 4 {
			break
		}
	}
	if seed == 10000 {
		t.Fatal("could not find a seed for paladin's default dispatch arm")
	}

	_ = lastMsg() // discard spawn/setup bytes
	dprng.ResetStream(seed)
	if !specPaladin(w, nil, mob, "", "") {
		t.Fatal("paladin default dispatch should return TRUE")
	}
	if got := lastMsg(); got != "" {
		t.Errorf("paladin default dispatch invented player bytes: %q", got)
	}
}

func TestSpecPaladin_DispelAlignment(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 32)

	mob.Prototype.Alignment = -350
	if got := paladinDispelSpell(mob); got != spells.SpellDispelGood {
		t.Errorf("evil paladin spell = %d, want SPELL_DISPEL_GOOD (%d)", got, spells.SpellDispelGood)
	}

	mob.Prototype.Alignment = 0
	if got := paladinDispelSpell(mob); got != spells.SpellDispelEvil {
		t.Errorf("neutral paladin spell = %d, want SPELL_DISPEL_EVIL (%d)", got, spells.SpellDispelEvil)
	}

	_ = player
}

func TestSpecPaladin_ChargeNativeArithmetic(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := preparePaladinCombat(t, w, player)
	weaponProto := &parser.Obj{
		VNum:      3002,
		Keywords:  "lance",
		ShortDesc: "a lance",
		Values:    [4]int{0, 2, 4, 12},
	}
	weapon := NewObjectInstance(weaponProto, -1)
	mob.EquipItem(weapon, int(SlotWield))

	const seed = 17
	reference := dprng.New(seed)
	reference.Number(1, 101) // C's charge percent draw.
	wantDamage := 2 * reference.Dice(2, 4)
	dprng.ResetStream(seed)
	startHP := player.GetHP()
	mobCharge(w, mob, player)

	if got := startHP - player.GetHP(); got != wantDamage {
		t.Errorf("charge damage = %d, want %d", got, wantDamage)
	}
	if got := mob.GetPosition(); got != combat.PosFighting {
		t.Errorf("successful charge moved mob to position %d, want fighting", got)
	}
	if got := mob.GetWaitState(); got != 0 {
		t.Errorf("charge changed GET_MOB_WAIT to %d, want 0", got)
	}

	player.SetHP(startHP)
	player.SetAC(-1000) // Forces percent > the subcmd=1 probability 131.
	mob.SetPosition(combat.PosFighting)
	dprng.ResetStream(seed)
	mobCharge(w, mob, player)
	if got := player.GetHP(); got != startHP {
		t.Errorf("failed charge changed victim HP to %d, want %d", got, startHP)
	}
	if got := mob.GetPosition(); got != combat.PosSitting {
		t.Errorf("failed unmounted charge position = %d, want sitting", got)
	}
}

func TestSpecPaladin_DisarmNativeAudienceAndState(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := preparePaladinCombat(t, w, player)
	weapon := NewObjectInstance(w.objs[3001], -1)
	weapon.Location = LocEquippedPlayer(player.GetName(), SlotWield)
	if err := player.Equipment.SetSlot(SlotWield, weapon); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}
	_ = lastMsg() // discard spawn/setup bytes

	dprng.ResetStream(23)
	mobDisarm(w, mob, player)
	if _, ok := player.Equipment.GetItemInSlot(SlotWield); ok {
		t.Fatal("disarm left the weapon equipped")
	}
	if got, ok := player.Inventory.FindItem("sword"); !ok || got != weapon {
		t.Fatal("disarm did not move the weapon to the victim's inventory")
	}

	msg := lastMsg()
	if !strings.Contains(msg, "A test mob deftly disarms you, knocking a steel sword from your hand!") {
		t.Errorf("victim disarm message = %q", msg)
	}
	if !strings.Contains(msg, "A test mob knocks a steel sword from Tester's hand!") {
		t.Errorf("room disarm message = %q", msg)
	}
}

func TestSpecPaladin_DisarmFailure(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := preparePaladinCombat(t, w, player)
	player.SetLevel(200) // Makes number(1, 101+GET_LEVEL(vict)) reach 200.
	weapon := NewObjectInstance(w.objs[3001], -1)
	weapon.Location = LocEquippedPlayer(player.GetName(), SlotWield)
	if err := player.Equipment.SetSlot(SlotWield, weapon); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}
	_ = lastMsg()

	var seed uint32
	for seed = 1; seed < 10000; seed++ {
		rng := dprng.New(seed)
		if rng.Number(1, 301) >= 200 {
			break
		}
	}
	if seed == 10000 {
		t.Fatal("could not find a deterministic disarm failure seed")
	}

	dprng.ResetStream(seed)
	mobDisarm(w, mob, player)
	if _, ok := player.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Fatal("failed disarm removed the victim's weapon")
	}
	if got := mob.GetPosition(); got != combat.PosSitting {
		t.Errorf("failed disarm position = %d, want sitting", got)
	}
	if got := lastMsg(); !strings.Contains(got, "tries to disarm you but fails and falls flat") {
		t.Errorf("failed disarm audience = %q", got)
	}
}
