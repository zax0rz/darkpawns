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
			for i := len(items) - 1; i >= 0; i-- {
				item := items[i]
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
			where := findEqPos(item, "")
			if where < 0 || where >= len(wearMessages) {
				continue
			}
			if _, occupied := killer.Equipment[where]; occupied {
				continue
			}
			if !killer.EquipItem(item, where) {
				continue
			}
			Act(w, false, killer, nil, item, nil, wearMessages[where][0], "", ToRoom)
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
