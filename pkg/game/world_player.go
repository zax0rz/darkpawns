package game

import (
	"fmt"
	"log/slog"
)

func (w *World) OnPlayerEnterRoom(player *Player, roomVNum int, ce CombatEngine) bool {
	mobs := w.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		// Check if mob is aggressive
		isAggressive := false
		if mob.Prototype != nil {
			for _, flag := range mob.Prototype.ActionFlags {
				if flag == "aggressive" {
					isAggressive = true
					break
				}
			}
		}

		if isAggressive && !player.IsFighting() {
			// Check if mob is already fighting
			if !ce.IsFighting(mob.GetName()) {
				go func(m *MobInstance) {
					if err := ce.StartCombat(m, player); err != nil {
						slog.Debug("aggro combat start failed", "mob", m.GetName(), "target", player.Name, "error", err)
					}
				}(mob)
				return true
			}
		}
	}
	return false
}

// GiveStartingItems implements do_start() item distribution from class.c lines 506-532.
// Creates ObjectInstance items from prototypes and adds them to player inventory.
// Source: class.c do_start()
func (w *World) GiveStartingItems(p *Player) {
	// Pack (8038) is created first, filled with bread (8010) + waterskin (8063)
	// then given to player

	packProto, packOK := w.GetObjPrototype(8038)

	// Class-specific items (given directly to player)
	switch p.Class {
	case ClassThief, ClassAssassin:
		w.giveItem(p, 8036) // dagger
		if packOK {
			// lockpicks (8027) go INTO the pack — handled after pack creation
			_ = packProto // suppress unused warning, used below
		}
	case ClassMageUser, ClassMagus:
		w.giveItem(p, 8036) // dagger
		w.giveItem(p, 1239) // obsidian
		w.giveItem(p, 1239) // obsidian (2x)
	case ClassNinja:
		w.giveItem(p, 8036) // dagger
	case ClassWarrior, ClassPsionic:
		w.giveItem(p, 8037) // small sword
	default:
		w.giveItem(p, 8023) // club
	}

	w.giveItem(p, 8019) // tunic (all classes)

	// Create pack and fill it
	if packOK {
		pack := NewObjectInstance(packProto, -1)
		pack.Contains = make([]*ObjectInstance, 0)

		// bread + waterskin always in pack
		if bread, ok := w.GetObjPrototype(8010); ok {
			pack.Contains = append(pack.Contains, NewObjectInstance(bread, -1))
		}
		if water, ok := w.GetObjPrototype(8063); ok {
			pack.Contains = append(pack.Contains, NewObjectInstance(water, -1))
		}
		// lockpicks in pack for thieves/assassins
		if p.Class == ClassThief || p.Class == ClassAssassin {
			if picks, ok := w.GetObjPrototype(8027); ok {
				pack.Contains = append(pack.Contains, NewObjectInstance(picks, -1))
			}
		}

		if err := w.MoveObjectToPlayerInventory(pack, p); err != nil {
			slog.Warn("starting pack failed", "player", p.Name, "error", err)
			return
		}
	}
}

// giveItem creates an ObjectInstance from a prototype vnum and adds it to player inventory.
func (w *World) giveItem(p *Player, vnum int) {
	proto, ok := w.GetObjPrototype(vnum)
	if !ok {
		return
	}
	obj := NewObjectInstance(proto, -1)
	if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
		slog.Warn("giveItem failed", "player", p.Name, "vnum", vnum, "error", err)
	}
}

// Stats returns world statistics.
func (w *World) Stats() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return fmt.Sprintf(
		"World: %d rooms, %d mobs (%d active), %d objects, %d zones, %d players online",
		len(w.rooms), len(w.mobs), len(w.activeMobs), len(w.objs), len(w.zones), len(w.players),
	)
}

// ScriptableWorld interface implementation

// GetPlayersInRoomScriptable returns all players in a given room as ScriptablePlayer.
