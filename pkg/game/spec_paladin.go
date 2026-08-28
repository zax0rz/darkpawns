package game

import (
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

const skillChargeNum = 147 // src/spells.h: SKILL_CHARGE

// mobCharge ports do_charge(ch, ..., subcmd=1). The player-facing command
// prelude is suppressed for an NPC, but damage() still emits the native charge
// skill message and applies the same weapon-dice arithmetic.
func mobCharge(w *World, me *MobInstance, vict combat.Combatant) {
	weapon, wielded := me.Equipment[int(SlotWield)]
	if !wielded || weapon == nil || weapon.Prototype == nil {
		return
	}

	weaponType := weapon.Prototype.Values[3]
	if weaponType != 3 && weaponType != 12 {
		return
	}

	percent := ((5 - (vict.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	mounted := me.IsAffected(affMounted)
	if mounted {
		percent += 5
	}
	if mob, ok := vict.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
		percent += 25
	}

	if percent > 131 {
		w.mobSkillDamage(me, vict, 0, skillChargeNum)
		if !mounted {
			me.SetPosition(combat.PosSitting)
		}
		return
	}

	dam := 2 * dprng.Dice(weapon.Prototype.Values[1], weapon.Prototype.Values[2])
	if mounted {
		dam += 50
	}
	w.mobSkillDamage(me, vict, dam, skillChargeNum)
}

// mobDisarm ports do_disarm(ch, ..., subcmd=1). Paladin's subcmd probability
// is 200, so normal-level targets with a wielded weapon always enter the C
// success branch; the strict comparison and target-level range are retained
// for high-level and focused cases.
func mobDisarm(w *World, me *MobInstance, vict combat.Combatant) {
	if me.GetFighting() != vict.GetName() || vict.GetFighting() != me.GetName() {
		return
	}

	var weapon *ObjectInstance
	switch target := vict.(type) {
	case *MobInstance:
		weapon = target.Equipment[int(SlotWield)]
	case *Player:
		weapon, _ = target.Equipment.GetItemInSlot(SlotWield)
	default:
		return
	}
	if weapon == nil {
		return
	}

	percent := dprng.Number(1, 101+vict.GetLevel())
	if percent < 200 {
		if target, ok := vict.(*MobInstance); ok {
			target.UnequipItem(int(SlotWield))
		} else if target, ok := vict.(*Player); ok {
			if err := target.Equipment.Unequip(SlotWield, target.Inventory); err != nil {
				return
			}
		}
		Act(w, true, me, vict, weapon, nil,
			"$n knocks $p from $N's hand!", "", ToNotVict)
		Act(w, true, me, vict, weapon, nil,
			"$n deftly disarms you, knocking $p from your hand!", "", ToVict)
		return
	}

	Act(w, true, me, vict, nil, nil,
		"$n tries to disarm you but fails and falls flat on $s face!", "", ToVict)
	Act(w, true, me, vict, nil, nil,
		"$n tries to disarm $N, but fails and falls flat on $s face!", "", ToNotVict)
	me.SetPosition(combat.PosSitting)
}
