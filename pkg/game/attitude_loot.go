package game

import (
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// attitudeLootMob is the game-layer implementation of fight.c's
// attitude_loot(). The combat package can describe the command sequence, but
// only the game layer owns mob inventories, corpse objects, and equipment.
func (w *World) attitudeLootMob(killer *MobInstance, victim combat.Combatant) {
	if w == nil || killer == nil || victim == nil {
		return
	}

	for range 2 {
		corpse := w.findAttitudeLootCorpse(killer.GetRoom(), victim.GetName())
		if corpse != nil {
			items := append([]*ObjectInstance(nil), corpse.Contains...)
			// C get_from_container walks contains forward (act.item.c:246-253).
			for _, item := range items {
				if item == nil || !item.IsTakeable() {
					continue
				}
				if err := w.MoveObjectToMobInventoryFront(item, killer); err != nil {
					slog.Warn("attitude loot get failed",
						"mob", killer.GetName(), "victim", victim.GetName(),
						"obj_vnum", item.GetVNum(), "error", err)
					continue
				}
				Act(w, true, killer, nil, item, corpse,
					"$n gets $p from $P.", "", ToRoom)
			}
		}

		// C's fake junking excludes containers and keys, and does not award
		// the normal player junk reward.
		for _, item := range append([]*ObjectInstance(nil), killer.Inventory...) {
			if item == nil || item.IsContainer() || item.GetTypeFlag() == ITEM_KEY || item.GetCost() > 150 {
				continue
			}
			Act(w, false, killer, nil, item, nil,
				"$n junks $p. It vanishes in a puff of smoke!", "", ToRoom)
			w.ExtractObject(item, killer.GetRoom())
		}

		// do_wear(ch, \"all\") uses find_eq_pos() and emits the ordinary room
		// wear act. Mob equipment has the same C WEAR_* slot numbering.
		for _, item := range append([]*ObjectInstance(nil), killer.Inventory...) {
			if item == nil || item.Prototype == nil {
				continue
			}
			if !CanSeeObject(killer, item) {
				continue
			}
			where := findEqPos(item, "")
			if where >= 0 {
				w.performMobileWear(killer, item, where)
			}
		}
	}
}

func (w *World) findAttitudeLootCorpse(roomVNum int, victimName string) *ObjectInstance {
	victimName = strings.ToLower(victimName)
	for _, item := range w.GetItemsInRoom(roomVNum) {
		if item == nil || !item.IsCorpse {
			continue
		}
		if victimName == "" || strings.Contains(strings.ToLower(item.GetShortDesc()), victimName) {
			return item
		}
	}
	return nil
}

// performMobileWear mirrors perform_wear for NPC do_wear("all") callers.
// Every gate applies to NPCs; invalid_class alone always permits NPCs.
// C act.item.c:1416-1517: wear_message BEFORE obj_from_char/equip_char.
func (w *World) performMobileWear(m *MobInstance, obj *ObjectInstance, where int) {
	if where < 0 || where >= NumWears {
		return
	}
	actorAct := func(text string) { w.mobileEquipmentAct(m, obj, text) }
	actorText := func(text string) {
		if w.MobileMessageSink != nil {
			w.MobileMessageSink(m, []byte(text))
		}
	}
	if !canWearAtPosition(obj, where) || where == eqWearLight {
		actorAct("You can't wear $p there.")
		return
	}
	if where == eqWearFingerR || where == eqWearNeck1 || where == eqWearWristR {
		if m.Equipped(where) != nil {
			where++
		}
	}
	if m.Equipped(where) != nil {
		actorText(alreadyWearing[where])
		return
	}
	switch where {
	case eqWearWield:
		if !canWearObject(obj, eqWearWield) {
			actorText("You can't wield that.\r\n")
			return
		}
		if m.IsAffected(affFleshAlter) {
			actorText("Your flesh is altered, you can't wield anything!\r\n")
			return
		}
		// constants.c str_app[].wield_w, indexed by live STRENGTH_APPLY_INDEX.
		weights := [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 20, 40, 40, 40, 40, 40, 40, 40, 22, 24, 26, 28, 30}
		if obj.GetWeight() > weights[combat.StrengthIndex(m.GetStr(), m.GetStrAdd())] {
			actorText("It is too heavy for you to use.\r\n")
			return
		}
		if obj.HasExtraFlag(0, extraFlagTwoHanded) && (m.Equipped(eqWearHold) != nil || m.Equipped(eqWearShield) != nil) {
			actorText("Both hands must be free to wield that.\r\n")
			return
		}
	case eqWearHold, eqWearShield:
		if weapon := m.Equipped(eqWearWield); weapon != nil && weapon.HasExtraFlag(0, extraFlagTwoHanded) {
			actorText("Both your hands are occupied with your weapon at the moment.\r\n")
			return
		}
	}
	Act(w, true, m, nil, obj, nil, wearMessages[where][0], "", ToRoom)
	actorAct(wearMessages[where][1])
	w.equipMobileFromInventory(m, obj, where)
}
