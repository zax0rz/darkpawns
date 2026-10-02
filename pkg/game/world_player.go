package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func (w *World) OnPlayerEnterRoom(player *Player, roomVNum int, ce CombatEngine) bool {
	mobs := w.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		// Check if mob is aggressive
		if hasMobFlag(mob, "aggressive") && !player.IsFighting() {
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

	// src/class.c:506-533: allocate the pack before class objects, and
	// insert thief picks before bread/water. obj_to_obj prepends (handler.c:948).
	var pack *ObjectInstance
	if proto, ok := w.GetObjPrototype(8038); ok {
		pack = w.newObjectInstance(proto, -1)
	}
	switch p.Class {
	case ClassThief:
		if pack != nil {
			if picks, ok := w.GetObjPrototype(8027); ok {
				w.giveStartingPackItem(p, pack, picks)
			}
		}
		w.giveItem(p, 8036)
	case ClassMageUser:
		w.giveItem(p, 8036)
		w.giveItem(p, 1239)
		w.giveItem(p, 1239)
	case ClassNinja:
		w.giveItem(p, 8036)
	case ClassWarrior, ClassPsionic:
		w.giveItem(p, 8037)
	default:
		w.giveItem(p, 8023)
	}
	w.giveItem(p, 8019)
	if pack != nil {
		if bread, ok := w.GetObjPrototype(8010); ok {
			w.giveStartingPackItem(p, pack, bread)
		}
		if water, ok := w.GetObjPrototype(8063); ok {
			w.giveStartingPackItem(p, pack, water)
		}
		if err := w.MoveObjectToPlayerInventory(pack, p); err != nil {
			slog.Warn("starting pack failed", "player", p.Name, "error", err)
		}
	}
}

func (w *World) giveStartingPackItem(p *Player, pack *ObjectInstance, proto *parser.Obj) {
	item := w.newObjectInstance(proto, -1)
	if err := w.MoveObjectToContainer(item, pack); err != nil {
		slog.Warn("starting pack item failed", "player", p.Name, "vnum", proto.VNum, "error", err)
		w.ExtractObject(item, p.GetRoom())
	}
}

// giveItem creates an ObjectInstance from a prototype vnum and adds it to player inventory.
func (w *World) giveItem(p *Player, vnum int) {
	proto, ok := w.GetObjPrototype(vnum)
	if !ok {
		return
	}
	obj := w.newObjectInstance(proto, -1)
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

func (w *World) countMobInstances(vnum int) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	count := 0
	for _, mob := range w.activeMobs {
		// VNum is immutable after construction. Read it directly so a global
		// world lock never nests a mob lock.
		if mob != nil && mob.VNum == vnum {
			count++
		}
	}
	return count
}

func (w *World) countObjectInstances(vnum int) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	count := 0
	for _, obj := range w.objectInstances {
		if obj != nil && obj.VNum == vnum {
			count++
		}
	}
	return count
}

// ScriptableWorld interface implementation

// GetPlayersInRoomScriptable returns all players in a given room as ScriptablePlayer.
