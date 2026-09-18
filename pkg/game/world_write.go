// Package game provides write methods for mutating in-memory world data.
// These methods hold the world write lock and modify runtime state only —
// they do NOT persist changes to disk (persistence is a future phase).
package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// --------------------------------------------------------------------------
// Room write methods
// --------------------------------------------------------------------------

// SetRoomFlagBit sets a single runtime C ROOM_* bit in a room's flag words
// under the world lock. Callers must NOT mutate room.Flags through the
// *parser.Room handed out by GetRoomInWorld — that pointer escapes the read
// lock, so lock-free writes race with locked readers. Returns false if the
// room doesn't exist.
func (w *World) SetRoomFlagBit(vnum int, flagBit int) bool {
	// Runtime room bits (the search skill's secret mark) are routine C
	// world[] mutations, not editor definitions; take the cheap path.
	return w.mutateRoom(vnum, func(room *parser.Room) bool {
		setRoomFlagBit(room, flagBit)
		return true
	})
}

// CreateRoomExit replaces an exit with the bare runtime record created by C's
// do_dig. It intentionally clears any prior door metadata and descriptions.
func (w *World) CreateRoomExit(vnum int, direction string, toRoom int) bool {
	return w.updateRoom(vnum, func(room *parser.Room) {
		if room.Exits == nil {
			room.Exits = make(map[string]parser.Exit)
		}
		room.Exits[direction] = parser.Exit{Direction: direction, ToRoom: toRoom}
	})
}

// --------------------------------------------------------------------------
// Mob write methods
// --------------------------------------------------------------------------

// AdjustMobPrototypes mirrors C adjust_mobs() (src/olc.c:279-307). It
// recalculates each prototype's damroll and hit-point addend from its level.
// The C command also marks each prototype's zone for a later OLC save; Go's
// world persistence currently has no equivalent dirty-zone queue, so this
// operation intentionally changes only the in-memory prototypes.
func (w *World) AdjustMobPrototypes() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, mob := range w.mobs {
		if mob.Level > 10 {
			// C's / 1.50 expression is converted back to int on assignment.
			mob.Damage.Plus = int(float64(mob.Level) / 1.50)
		} else {
			mob.Damage.Plus = mob.Level / 2
		}

		addHP := 10*mob.Level + 10
		if mob.Level > 22 {
			addHP += 13 * (mob.Level - 22)
		}
		if mob.Level > 30 {
			addHP += 560 * (mob.Level - 30)
		}
		mob.HP.Plus = addHP
	}

	return len(w.mobs)
}

// --------------------------------------------------------------------------
// Object write methods
// --------------------------------------------------------------------------

// --------------------------------------------------------------------------
// Shop write methods
// --------------------------------------------------------------------------

// SetShopBuyTypes sets the buy types for the shop run by the given keeper NPC.
// Returns false if no shop exists for that keeper.
func (w *World) SetShopBuyTypes(keeperVNum int, buyTypes []int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.BuyTypes = buyTypes
	return true
}

// SetShopSellTypes sets the sell types for the shop run by the given keeper NPC.
// Returns false if no shop exists for that keeper.
func (w *World) SetShopSellTypes(keeperVNum int, sellTypes []int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.SellTypes = sellTypes
	return true
}

// SetShopProfit sets the buy and sell profit multipliers for the shop run by
// the given keeper NPC. Returns false if no shop exists for that keeper.
func (w *World) SetShopProfit(keeperVNum int, buyProfit, sellProfit float64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.ProfitBuy = buyProfit
	shop.ProfitSell = sellProfit
	return true
}

// ResetZone triggers a manual zone reset. Returns an error if the zone or spawner is unavailable.
func (w *World) ResetZone(number int) error {
	w.mu.RLock()
	zone, ok := w.zones[number]
	w.mu.RUnlock()
	if !ok {
		return fmt.Errorf("zone %d not found", number)
	}
	if w.spawner == nil {
		return fmt.Errorf("spawner not initialized")
	}
	return w.spawner.ExecuteZoneReset(zone)
}

// The three setters below outlived the HTTP write path that was their only
// production caller. They stay because redit's concurrency tests use them to
// simulate an external write landing on a live room while the editor holds a
// working copy — pkg/game/world_redit_test.go drives them in a loop against
// CommitEditedRoom, and pkg/session/redit_test.go covers the D5 manifest
// scenario redit.overlap-web-admin.
//
// That is a different thing from the defect this file's other forty-odd
// setters were part of: a route that wrote to the world without passing
// through the command parser, where dp-oracle-diff could not observe it.
// Simulating a concurrent writer inside a test is not a bypass. Nothing
// outside a test reaches these.

// SetRoomName updates a room's name. Returns false if the room doesn't exist.
func (w *World) SetRoomName(vnum int, name string) bool {
	return w.updateRoom(vnum, func(room *parser.Room) { room.Name = name })
}

// SetRoomDescription updates a room's description. Returns false if the room doesn't exist.
func (w *World) SetRoomDescription(vnum int, desc string) bool {
	return w.updateRoom(vnum, func(room *parser.Room) { room.Description = desc })
}

// SetRoomSector sets a room's sector type. Returns false if the room doesn't exist.
func (w *World) SetRoomSector(vnum int, sector int) bool {
	return w.updateRoom(vnum, func(room *parser.Room) { room.Sector = sector })
}
