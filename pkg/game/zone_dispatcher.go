package game

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ZoneDispatcher manages per-zone goroutines for resets, mob AI, and state.
// Each zone gets its own goroutine for isolated processing.
// This replaces the single-threaded serial zone reset loop in StartZoneResets.
type ZoneDispatcher struct {
	world  *World
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.RWMutex
	zones    map[int]*zoneWorker
	interval time.Duration
}

// zoneWorker holds per-zone goroutine state.
type zoneWorker struct {
	zone   *parser.Zone
	ticks  uint64
	ctx    context.Context
	cancel context.CancelFunc
}

// NewZoneDispatcher creates a new zone dispatcher.
// interval is the tick rate for zone processing (typically same as game pulse, ~100ms).
func NewZoneDispatcher(w *World, interval time.Duration) *ZoneDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &ZoneDispatcher{
		world:    w,
		ctx:      ctx,
		cancel:   cancel,
		zones:    make(map[int]*zoneWorker),
		interval: interval,
	}
}

// Start launches per-zone goroutines for all registered zones.
func (zd *ZoneDispatcher) Start() {
	zd.mu.Lock()
	defer zd.mu.Unlock()

	zones := zd.world.GetAllZones()
	for _, zone := range zones {
		zd.startZoneLocked(zone)
	}
	slog.Info("zone dispatcher started", "zone_count", len(zd.zones))
}

// startZoneLocked creates and starts a goroutine for a single zone.
func (zd *ZoneDispatcher) startZoneLocked(zone *parser.Zone) {
	if _, exists := zd.zones[zone.Number]; exists {
		return // already started
	}

	ctx, cancel := context.WithCancel(zd.ctx)
	worker := &zoneWorker{
		zone:   zone,
		ctx:    ctx,
		cancel: cancel,
	}
	zd.zones[zone.Number] = worker

	zd.wg.Add(1)
	go zd.zoneLoop(worker)
}

// zoneLoop is the main goroutine for a single zone.
// It handles periodic resets, mob AI, and respawn checks for that zone.
func (zd *ZoneDispatcher) zoneLoop(worker *zoneWorker) {
	defer zd.wg.Done()

	zoneNum := worker.zone.Number
	slog.Debug("zone goroutine started", "zone", zoneNum)

	// Zone-specific tick interval (base rate * zone's own reset timing if available)
	ticker := time.NewTicker(zd.interval)
	defer ticker.Stop()

	resetInterval := zd.zoneResetInterval(worker.zone)
	lastReset := time.Now()

	for {
		select {
		case <-worker.ctx.Done():
			slog.Debug("zone goroutine stopped", "zone", zoneNum)
			return

		case <-ticker.C:
			worker.ticks++

			// Run zone reset on interval
			if time.Since(lastReset) >= resetInterval {
				zd.runZoneReset(worker.zone)
				lastReset = time.Now()
			}

			// Run per-zone mob AI processing
			zd.runZoneMobAI(worker.zone)
		}
	}
}

// runZoneReset triggers a zone reset/respawn.
func (zd *ZoneDispatcher) runZoneReset(zone *parser.Zone) {
	if zd.world.spawner == nil {
		return
	}
	if err := zd.world.spawner.ExecuteZoneReset(zone); err != nil {
		slog.Warn("zone reset failed", "zone", zone.Number, "error", err)
	}
}

// runZoneMobAI processes mob AI ticks for all mobs in a zone.
// Iterates active mobs that reside in rooms belonging to this zone and
// dispatches basic AI behaviors. Placeholder — expand as AI systems land.
func (zd *ZoneDispatcher) runZoneMobAI(zone *parser.Zone) {
	mobs := zd.world.GetAllMobs()
	zoneNum := zone.Number

	for _, mob := range mobs {
		roomVNum := mob.GetRoom()
		if roomVNum < 0 {
			continue
		}
		room := zd.world.GetRoomInWorld(roomVNum)
		if room == nil || room.Zone != zoneNum {
			continue
		}

		// Check respawn triggers — room_reset_vnum list
		// TODO: Scan zone reset commands for the mob's room to respawn if the
		// mob is below its expected count and the respawn timer has elapsed.

		// Move wandering mobs
		if mob.HasFlag("wander") {
			// TODO: Pick a random exit and move the mob to the adjacent room.
			// See src/mobact.c:mob_activity() for the original C logic.
		}

		// Check aggro ranges
		if mob.HasFlag("aggressive") && !mob.Fighting {
			// TODO: Scan room for players below the aggro level threshold
			// and initiate combat via w.AttackMobOnPlayer(mob, target).
			// See src/mobact.c:hitprcnt() and do_hunt_victim() for thresholds.
		}

		_ = mob // mob fully used in the TODO expansion above
	}

	// TODO: Process zone-wide events (evacuation, invasion triggers)
	// These are spec_proc-based script events triggered by zone-level time
	// or condition changes, not per-mob loop.
}

// zoneResetInterval returns the reset interval for a zone.
// Defaults to the dispatcher interval if the zone doesn't specify one.
func (zd *ZoneDispatcher) zoneResetInterval(zone *parser.Zone) time.Duration {
	if zone.Lifespan > 0 {
		return time.Duration(zone.Lifespan) * time.Minute
	}
	return zd.interval * 30 // default: ~every 30 ticks
}

// Stop gracefully shuts down all zone goroutines and waits for them to finish.
func (zd *ZoneDispatcher) Stop() {
	zd.cancel()
	zd.wg.Wait()
	zd.mu.Lock()
	zd.zones = make(map[int]*zoneWorker)
	zd.mu.Unlock()
	slog.Info("zone dispatcher stopped")
}

// ZoneCount returns the number of active zone goroutines.
func (zd *ZoneDispatcher) ZoneCount() int {
	zd.mu.RLock()
	defer zd.mu.RUnlock()
	return len(zd.zones)
}

// ZoneTicks returns the total ticks processed by a specific zone worker.
func (zd *ZoneDispatcher) ZoneTicks(zoneNum int) uint64 {
	zd.mu.RLock()
	defer zd.mu.RUnlock()
	if w, ok := zd.zones[zoneNum]; ok {
		return w.ticks
	}
	return 0
}
