// Package engine — gameloop.go: game loop orchestrator.
//
// Ported from comm.c:heartbeat(). Uses a 100ms ticker and dispatches
// sub-functions on preset pulse intervals.
//
// Pulse constants:
//
//	PASSES_PER_SEC  = 10   → 100ms ticker interval
//	PULSE_ZONE      = 600  → every 60 seconds
//	PULSE_MOBILE    = 40   → every 4 seconds
//	PULSE_VIOLENCE  = 20   → every 2 seconds
//	PULSE_TICK      = 300  → every 30 seconds
//	SECS_PER_MUD_HOUR = 60 → real seconds per Mud hour

package engine

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// Pulse constants — matching comm.c PASSES_PER_SEC = 10.
const (
	PASSES_PER_SEC    = 10                  // 100ms ticker intervals per second
	PULSE_ZONE        = 60 * PASSES_PER_SEC // 600 → 60s
	PULSE_MOBILE      = 4 * PASSES_PER_SEC  // 40  → 4s
	PULSE_VIOLENCE    = 2 * PASSES_PER_SEC  // 20  → 2s
	PULSE_TICK        = 30 * PASSES_PER_SEC // 300 → 30s
	SECS_PER_MUD_HOUR = 60                  // 60 real seconds per Mud hour
)

// DefaultCrashSaveInterval is the number of minutes between automatic crash saves.
const DefaultCrashSaveInterval = 15

// UptimeSnapshot records a server uptime reading.
type UptimeSnapshot struct {
	StartedAt    time.Time
	CurrentPulse int64
	Elapsed      time.Duration
}

// ---------------------------------------------------------------------------
// Callback types — heartbeat dispatch uses these for loose coupling.
// ---------------------------------------------------------------------------

// GameLoopCallbacks groups all optional heartbeat dispatch functions.
// Each is called on its corresponding pulse cycle from the main ticker.
type GameLoopCallbacks struct {
	// OnZoneUpdate — called every PULSE_ZONE (60s). Ported from zone_update().
	OnZoneUpdate func()

	// OnCheckIdlePasswords — called every 15 * PASSES_PER_SEC (1.5s).
	// Ported from check_idle_passwords() in comm.c.
	OnCheckIdlePasswords func()

	// OnMobileActivity — called every PULSE_MOBILE (4s). Ported from mobile_activity().
	OnMobileActivity func()
	// OnRoomActivity — called every PULSE_MOBILE (4s). Ported from room_activity().
	OnRoomActivity func()
	// OnObjectActivity — called every PULSE_MOBILE (4s). Ported from object_activity().
	OnObjectActivity func()

	// OnPerformViolence — called every PULSE_VIOLENCE (2s). Ported from perform_violence().
	OnPerformViolence func()

	// OnWeatherAndTime — called every SECS_PER_MUD_HOUR * PASSES_PER_SEC (60s).
	// Ported from weather_and_time(1).
	OnWeatherAndTime func()
	// OnAffectUpdate — called every Mud hour. Ported from affect_update().
	OnAffectUpdate func()
	// OnPointUpdate — called every Mud hour. Ported from point_update().
	OnPointUpdate func()
	// OnHuntItems — called every Mud hour. Ported from hunt_items().
	OnHuntItems func()
	// OnFlushPlayerFile — called every Mud hour. Ported from fflush(player_fl).
	OnFlushPlayerFile func()

	// OnAutoSave — called every 60 * PASSES_PER_SEC (60s) only if auto_save is true.
	// ported from Crash_save_all().
	OnAutoSave func()
	// getAutoSave returns the current auto_save state.
	AutoSaveEnabled func() bool
	// getAutoSaveTime returns the configured autosave interval in minutes.
	// Ported from 'autosave_time' — usually 15 minutes.
	AutoSaveIntervalMinutes func() int

	// OnRecordUsage — called every 5 * 60 * PASSES_PER_SEC (5 min).
	// Ported from record_usage() in comm.c.
	OnRecordUsage func()

	// OnWriteMudDate — called every 60 * 60 * PASSES_PER_SEC (60 min).
	// Ported from write_mud_date_to_file().
	OnWriteMudDate func()

	// OnEventProcess — called every heartbeat tick (100ms).
	// Ported from event_process().
	OnEventProcess func()

	// OnExtractPending — called every heartbeat tick (100ms).
	// Ported from extract_pending_chars().
	OnExtractPending func()
}

// ---------------------------------------------------------------------------
// GameLoop
// ---------------------------------------------------------------------------

// GameLoop is the main server heartbeat orchestrator.
// It runs a 100ms ticker and dispatches callbacks at configured pulse intervals.
type GameLoop struct {
	// Pulse is the current pulse counter, atomically read/written.
	Pulse atomic.Int64

	// startedAt records when the loop began.
	startedAt time.Time

	// callbacks contains the dispatch functions.
	callbacks GameLoopCallbacks

	// tickerInterval is the base tick interval (100ms / PASSES_PER_SEC).
	tickerInterval time.Duration

	// stopCh signals the goroutine to exit.
	stopCh chan struct{}
	// doneCh is closed when the goroutine exits.
	doneCh chan struct{}
}

// NewGameLoop creates a new GameLoop with the given callbacks.
func NewGameLoop(callbacks GameLoopCallbacks) *GameLoop {
	return &GameLoop{
		callbacks:      callbacks,
		tickerInterval: time.Second / PASSES_PER_SEC, // 100ms
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
}

// Start begins the game loop in a new goroutine. Returns immediately.
// The ticker runs every 100ms. Each tick increments the pulse counter and
// dispatches heartbeat callbacks.
func (gl *GameLoop) Start() {
	gl.startedAt = time.Now()
	slog.Info("game loop starting",
		"tickerInterval", gl.tickerInterval,
		"pulsesPerSec", PASSES_PER_SEC,
	)
	go gl.run()
}

// Stop signals the loop goroutine to stop and waits for it to finish.
func (gl *GameLoop) Stop() {
	close(gl.stopCh)
	<-gl.doneCh
	slog.Info("game loop stopped")
}

// Uptime returns a snapshot of the server uptime.
func (gl *GameLoop) Uptime() UptimeSnapshot {
	return UptimeSnapshot{
		StartedAt:    gl.startedAt,
		CurrentPulse: gl.Pulse.Load(),
		Elapsed:      time.Since(gl.startedAt),
	}
}

// run is the main goroutine body.
func (gl *GameLoop) run() {
	defer close(gl.doneCh)

	ticker := time.NewTicker(gl.tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-gl.stopCh:
			return
		case <-ticker.C:
			pulse := gl.Pulse.Add(1)
			gl.heartbeat(pulse)
		}
	}
}

// heartbeat dispatches all sub-functions based on the current pulse.
// Ported from comm.c:heartbeat().
//
// The pulse counter starts at 1 (the first Add call) and increments each tick.
// Pulse modulo checks use the same logic as the C source.
func (gl *GameLoop) heartbeat(pulse int64) {
	cb := gl.callbacks

	// Every tick (100ms)
	if cb.OnEventProcess != nil {
		cb.OnEventProcess()
	}
	if cb.OnExtractPending != nil {
		cb.OnExtractPending()
	}

	// PULSE_ZONE → every 60 seconds
	if pulse%PULSE_ZONE == 0 && cb.OnZoneUpdate != nil {
		cb.OnZoneUpdate()
	}

	// 15 * PASSES_PER_SEC → every 1.5 seconds
	if pulse%(15*PASSES_PER_SEC) == 0 && cb.OnCheckIdlePasswords != nil {
		cb.OnCheckIdlePasswords()
	}

	// PULSE_MOBILE → every 4 seconds
	if pulse%PULSE_MOBILE == 0 {
		if cb.OnMobileActivity != nil {
			cb.OnMobileActivity()
		}
		if cb.OnRoomActivity != nil {
			cb.OnRoomActivity()
		}
		if cb.OnObjectActivity != nil {
			cb.OnObjectActivity()
		}
	}

	// PULSE_VIOLENCE → every 2 seconds
	if pulse%PULSE_VIOLENCE == 0 && cb.OnPerformViolence != nil {
		cb.OnPerformViolence()
	}

	// SECS_PER_MUD_HOUR * PASSES_PER_SEC → every 60 real seconds
	if pulse%(SECS_PER_MUD_HOUR*PASSES_PER_SEC) == 0 {
		if cb.OnWeatherAndTime != nil {
			cb.OnWeatherAndTime()
		}
		if cb.OnAffectUpdate != nil {
			cb.OnAffectUpdate()
		}
		if cb.OnPointUpdate != nil {
			cb.OnPointUpdate()
		}
		if cb.OnHuntItems != nil {
			cb.OnHuntItems()
		}
		if cb.OnFlushPlayerFile != nil {
			cb.OnFlushPlayerFile()
		}
	}

	// Auto-save: every 60 * PASSES_PER_SEC (60s) if auto_save is on
	if pulse%(60*PASSES_PER_SEC) == 0 {
		autoSave := false
		if cb.AutoSaveEnabled != nil {
			autoSave = cb.AutoSaveEnabled()
		}
		if autoSave && cb.AutoSaveIntervalMinutes != nil && cb.OnAutoSave != nil {
			interval := cb.AutoSaveIntervalMinutes()
			// We track crash-save ticks via the pulse counter:
			// autosave_time is in minutes, so we need
			// interval * 60 * PASSES_PER_SEC pulses.
			if interval > 0 && pulse%(int64(interval)*60*PASSES_PER_SEC) == 0 {
				cb.OnAutoSave()
			}
		}
	}

	// Record usage every 5 minutes
	if pulse%(5*60*PASSES_PER_SEC) == 0 && cb.OnRecordUsage != nil {
		cb.OnRecordUsage()
	}

	// Write Mud date every 60 minutes
	if pulse%(60*60*PASSES_PER_SEC) == 0 && cb.OnWriteMudDate != nil {
		cb.OnWriteMudDate()
	}
}
