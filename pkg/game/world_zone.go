package game

import (
	"fmt"
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

// SetShopManager sets the shop manager.
func (w *World) SetShopManager(manager common.ShopManager) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.shopManager = manager
}

// GetShopByKeeper returns a shop by keeper NPC VNum.
// Uses the concrete *ShopManager if available.
func (w *World) GetShopByKeeper(vnum int) (*Shop, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Try the concrete ShopManager first
	if sm, ok := w.shopManager.(*ShopManager); ok {
		shop := sm.GetShopByKeeper(vnum)
		return shop, shop != nil
	}

	return nil, false
}

// GetAllZones returns all zones.
func (w *World) GetAllZones() []*parser.Zone {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zones := make([]*parser.Zone, 0, len(w.zones))
	for _, zone := range w.zones {
		zones = append(zones, zone)
	}
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

// StartZoneDispatcher starts per-zone goroutines for resets and AI.
func (w *World) StartZoneDispatcher() {
	if w.zoneDispatcher != nil {
		w.zoneDispatcher.Start()
	}
}

// StopZoneDispatcher gracefully stops all zone goroutines.
func (w *World) StopZoneDispatcher() {
	if w.zoneDispatcher != nil {
		w.zoneDispatcher.Stop()
	}
}

// GetZoneDispatcher returns the zone dispatcher.
func (w *World) GetZoneDispatcher() *ZoneDispatcher {
	return w.zoneDispatcher
}

// StartPeriodicResets starts periodic zone reset checks.
func (w *World) StartPeriodicResets(interval time.Duration) {
	if w.spawner == nil {
		w.spawner = NewSpawner(w)
	}
	w.spawner.StartPeriodicResets(interval)
}

// GetSpawner returns the world's spawner.
func (w *World) GetSpawner() *Spawner {
	return w.spawner
}

// OnPlayerEnterRoom handles player entering a room (for aggressive mobs).
// Returns true if combat was initiated.
