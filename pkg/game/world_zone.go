package game

import (
	"fmt"
	"sort"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func (w *World) GetMobPrototype(vnum int) (*parser.Mob, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	mob, ok := w.mobs[vnum]
	return mob, ok
}

// GetObjPrototype returns an object prototype by VNum.
func (w *World) GetObjPrototype(vnum int) (*parser.Obj, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	obj, ok := w.objs[vnum]
	return obj, ok
}

// GetZone returns a zone by number.
func (w *World) GetZone(number int) (*parser.Zone, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zone, ok := w.zones[number]
	return zone, ok
}

// GetShopManager returns the shop manager.
func (w *World) GetShopManager() common.ShopManager {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.shopManager
}

// ShopBitvectorForKeeper returns the shop behavior bitvector for a
// shopkeeper mob vnum (shop.h: WILL_START_FIGHT=1, WILL_BANK_MONEY=2).
// The keeper set comes from the .shp files at boot — C's equivalent of
// is_shopkeeper's GET_MOB_SPEC == shop_keeper membership test.
func (w *World) ShopBitvectorForKeeper(vnum int) (int, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	bits, ok := w.shopKeepers[vnum]
	return bits, ok
}

// SetShopManager sets the shop manager.
func (w *World) SetShopManager(manager common.ShopManager) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.shopManager = manager
}

// legacyShopManagerLocked returns the C-record-backed shop view used by the
// session and admin compatibility paths. Tests and older callers may install
// the legacy manager directly; production installs systems.ShopManager, so
// fall back to the parsed .shp index in that case.
func (w *World) legacyShopManagerLocked() *ShopManager {
	if sm, ok := w.shopManager.(*ShopManager); ok {
		return sm
	}
	return w.cShopManager
}

// GetShopByKeeper returns a shop by keeper NPC VNum.
// Uses the concrete ShopManager if available.
func (w *World) GetShopByKeeper(vnum int) (*Shop, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if sm := w.legacyShopManagerLocked(); sm != nil {
		shop := sm.GetShopByKeeper(vnum)
		return shop, shop != nil
	}

	return nil, false
}

// GetAllShops returns all registered shops.
func (w *World) GetAllShops() []*Shop {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if sm := w.legacyShopManagerLocked(); sm != nil {
		return sm.GetAllShops()
	}
	return nil
}

// ShopBuysType returns true if the shopkeeper mob (by VNum) runs a shop
// that buys items of the given type flag.
// Source: scripts.c lua_item_check() — iterates SHOP_BUYTYPE().
func (w *World) ShopBuysType(mobVNum int, itemType int) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if sm := w.legacyShopManagerLocked(); sm != nil {
		shop := sm.GetShopByKeeper(mobVNum)
		if shop == nil {
			return false
		}
		return shop.WillBuyType(itemType)
	}
	return false
}

// GetAllMobPrototypes returns all mob prototypes from the parsed world data.
func (w *World) GetAllMobPrototypes() []*parser.Mob {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]*parser.Mob, 0, len(w.mobs))
	for _, m := range w.mobs {
		result = append(result, m)
	}
	return result
}

// GetAllObjPrototypes returns all object prototypes from the parsed world data.
func (w *World) GetAllObjPrototypes() []*parser.Obj {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]*parser.Obj, 0, len(w.objs))
	for _, o := range w.objs {
		result = append(result, o)
	}
	return result
}

// GetAllZones returns all zones.
func (w *World) GetAllZones() []*parser.Zone {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zones := make([]*parser.Zone, 0, len(w.zones))
	for _, zone := range w.zones {
		zones = append(zones, zone)
	}
	sort.Slice(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
	return zones
}

// StartZoneResets starts all zone resets.
func (w *World) StartZoneResets() error {
	if w.spawner == nil {
		w.spawner = NewSpawner(w)
	}

	zones := w.GetAllZones()
	for _, zone := range zones {
		if err := w.spawner.ExecuteZoneReset(zone); err != nil {
			return fmt.Errorf("zone %d reset failed: %w", zone.Number, err)
		}
	}
	return nil
}

// StartPeriodicResets starts periodic zone reset checks.
func (w *World) StartPeriodicResets(interval time.Duration) {
	if w.spawner == nil {
		w.spawner = NewSpawner(w)
	}
	w.spawner.StartPeriodicResets(interval)
}

// StopPeriodicResets signals the periodic zone reset goroutine to exit cleanly.
func (w *World) StopPeriodicResets() {
	if w.spawner != nil {
		w.spawner.StopPeriodicResets()
	}
}

// GetSpawner returns the world's spawner.
func (w *World) GetSpawner() *Spawner {
	return w.spawner
}

// OnPlayerEnterRoom handles player entering a room (for aggressive mobs).
// Returns true if combat was initiated.
