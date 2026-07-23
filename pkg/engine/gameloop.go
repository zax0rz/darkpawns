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
//	SECS_PER_MUD_HOUR = 75 → real seconds per Mud hour (C default)

package engine

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"
)

// Pulse constants — matching comm.c PASSES_PER_SEC = 10.
const (
	PASSES_PER_SEC    = 10                  // 100ms ticker intervals per second
	PULSE_ZONE        = 60 * PASSES_PER_SEC // 600 → 60s
	PULSE_MOBILE      = 4 * PASSES_PER_SEC  // 40  → 4s
	PULSE_VIOLENCE    = 2 * PASSES_PER_SEC  // 20  → 2s
	PULSE_TICK        = 30 * PASSES_PER_SEC // 300 → 30s
	SECS_PER_MUD_HOUR = 63                  // 63 real seconds per Mud hour (Dark Pawns override, src/utils.h:135)
)

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
	// OnDrainInput — called every heartbeat tick (100ms), at the TOP of
	// heartbeat before any other dispatch. Port of comm.c:603: the game_loop
	// drains pending player input (one command per session) BEFORE
	// perform_violence within a pass. The manager wires this to
	// DrainInputQueues, which decrements each player's wait by one pulse and
	// drains one queued command per session when wait reaches 0 (DP-1201).
	OnDrainInput func()

	// OnZoneUpdate — called every PULSE_ZONE (60s). Ported from zone_update().
	OnZoneUpdate func()

	// OnCheckIdlePasswords — called every 15 * PASSES_PER_SEC (15s).
	// Ported from check_idle_passwords() in comm.c.
	OnCheckIdlePasswords func()

	// OnReapLinkdeadSessions — called every 15 * PASSES_PER_SEC (15s).
	// Detects dead TCP sockets and extracts ghost players (DP-902).
	OnReapLinkdeadSessions func()

	// OnMobileActivity — called every PULSE_MOBILE (4s). Ported from mobile_activity().
	OnMobileActivity func()
	// OnRoomActivity — called every PULSE_MOBILE (4s). Ported from room_activity().
	OnRoomActivity func()
	// OnObjectActivity — called every PULSE_MOBILE (4s). Ported from object_activity().
	OnObjectActivity func()

	// OnPerformViolence — called every PULSE_VIOLENCE (2s). Ported from perform_violence().
	OnPerformViolence func()

	// OnWeatherAndTime — called every SECS_PER_MUD_HOUR * PASSES_PER_SEC (63s).
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
	pumpMu    sync.Mutex

	// tickerInterval is the base tick interval (100ms / PASSES_PER_SEC).
	tickerInterval time.Duration

	// stopCh signals the goroutine to exit.
	stopCh chan struct{}
	// doneCh is closed when the goroutine exits.
	doneCh chan struct{}

	// started is set when Start has launched the goroutine.
	started atomic.Bool

	// lifecycle guards Start/Stop transitions.
	startOnce sync.Once
	stopOnce  sync.Once
}

const maxPumpPulses = 100_000

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
// dispatches heartbeat callbacks. Start is idempotent: repeated calls are
// ignored after the loop has been started once.
//
// The loop stops when ctx is canceled or when Stop is called, whichever comes
// first — so a server-lifetime context cancellation drains the heartbeat the
// same way an explicit Stop does (DP-892). Pass context.Background() if you
// only want signal/Stop-driven shutdown.
func (gl *GameLoop) Start(ctx context.Context) {
	if dpclock.Frozen() {
		slog.Info("game loop wall-clock ticker frozen", "env", "DP_CLOCK")
		return
	}
	gl.startOnce.Do(func() {
		gl.startedAt = time.Now()
		gl.started.Store(true)
		slog.Info(
			"game loop starting",
			"tickerInterval", gl.tickerInterval,
			"pulsesPerSec", PASSES_PER_SEC,
		)
		go gl.run(ctx)
	})
}

// PumpPulses advances the deterministic clock synchronously. The telnet
// oracle-control seam calls this only while DP_CLOCK has disabled the
// wall-clock loop, so callbacks cannot interleave with ticker-driven pulses.
func (gl *GameLoop) PumpPulses(n int) error {
	if !dpclock.Frozen() {
		return fmt.Errorf("pulse pump requires DP_CLOCK")
	}
	if n <= 0 || n > maxPumpPulses {
		return fmt.Errorf("pulse count must be between 1 and %d", maxPumpPulses)
	}

	gl.pumpMu.Lock()
	defer gl.pumpMu.Unlock()
	for range n {
		pulse := gl.Pulse.Add(1)
		gl.heartbeat(pulse)
	}
	return nil
}

// Stop signals the loop goroutine to stop and waits for it to finish.
// Stop is idempotent and safe to call before Start; it returns immediately
// if the loop was never started. Stop is also safe to call after the loop has
// already exited via context cancellation — it just observes the closed
// doneCh and returns.
func (gl *GameLoop) Stop() {
	if !gl.started.Load() {
		return
	}
	gl.stopOnce.Do(func() {
		close(gl.stopCh)
		<-gl.doneCh
		slog.Info("game loop stopped")
	})
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
func (gl *GameLoop) run(ctx context.Context) {
	defer close(gl.doneCh)

	ticker := time.NewTicker(gl.tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-gl.stopCh:
			return
		case <-ctx.Done():
			slog.Info("game loop stopping: context canceled", "err", ctx.Err())
			return
		case <-ticker.C:
			pulse := gl.Pulse.Add(1)
			gl.heartbeat(pulse)
		}
	}
}

// safeInvoke runs a single heartbeat callback under a recover guard. A panic in
// any callback (e.g. a nil dereference in AffectUpdate) would otherwise unwind
// the loop goroutine and take the whole MUD down with no restart mechanism
// (DP-1019). Instead we log the offending callback, pulse, panic value and
// stack, then continue — so one bad callback is isolated and the remaining
// callbacks for this tick, and every future tick, still run.
func (gl *GameLoop) safeInvoke(name string, pulse int64, fn func()) {
	if fn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error(
				"heartbeat callback panicked; loop continues",
				"callback", name,
				"pulse", pulse,
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()
	fn()
}

// heartbeat dispatches all sub-functions based on the current pulse.
// Ported from comm.c:heartbeat().
//
// The pulse counter starts at 1 (the first Add call) and increments each tick.
// Pulse modulo checks use the same logic as the C source.
//
// Every callback is dispatched through safeInvoke so a panic in one cannot kill
// the loop goroutine (DP-1019).
func (gl *GameLoop) heartbeat(pulse int64) {
	cb := gl.callbacks

	// Per-pulse command drain — FIRST, before perform_violence (comm.c:603
	// drains input before the violence pass within a heartbeat). DP-1201.
	gl.safeInvoke("DrainInput", pulse, cb.OnDrainInput)

	// Every tick (100ms)
	gl.safeInvoke("EventProcess", pulse, cb.OnEventProcess)
	gl.safeInvoke("ExtractPending", pulse, cb.OnExtractPending)

	// PULSE_ZONE → every 60 seconds
	if pulse%PULSE_ZONE == 0 {
		gl.safeInvoke("ZoneUpdate", pulse, cb.OnZoneUpdate)
	}

	// 15 * PASSES_PER_SEC → every 15 seconds
	if pulse%(15*PASSES_PER_SEC) == 0 {
		gl.safeInvoke("CheckIdlePasswords", pulse, cb.OnCheckIdlePasswords)
		gl.safeInvoke("ReapLinkdeadSessions", pulse, cb.OnReapLinkdeadSessions)
	}

	// PULSE_MOBILE → every 4 seconds
	if pulse%PULSE_MOBILE == 0 {
		gl.safeInvoke("MobileActivity", pulse, cb.OnMobileActivity)
		gl.safeInvoke("RoomActivity", pulse, cb.OnRoomActivity)
		gl.safeInvoke("ObjectActivity", pulse, cb.OnObjectActivity)
	}

	// PULSE_VIOLENCE → every 2 seconds
	if pulse%PULSE_VIOLENCE == 0 {
		gl.safeInvoke("PerformViolence", pulse, cb.OnPerformViolence)
	}

	// SECS_PER_MUD_HOUR * PASSES_PER_SEC → every 63 real seconds
	if pulse%(SECS_PER_MUD_HOUR*PASSES_PER_SEC) == 0 {
		gl.safeInvoke("WeatherAndTime", pulse, cb.OnWeatherAndTime)
		gl.safeInvoke("AffectUpdate", pulse, cb.OnAffectUpdate)
		gl.safeInvoke("PointUpdate", pulse, cb.OnPointUpdate)
		gl.safeInvoke("HuntItems", pulse, cb.OnHuntItems)
		gl.safeInvoke("FlushPlayerFile", pulse, cb.OnFlushPlayerFile)
	}

	// Record usage every 5 minutes
	if pulse%(5*60*PASSES_PER_SEC) == 0 {
		gl.safeInvoke("RecordUsage", pulse, cb.OnRecordUsage)
	}

	// Write Mud date every 60 minutes
	if pulse%(60*60*PASSES_PER_SEC) == 0 {
		gl.safeInvoke("WriteMudDate", pulse, cb.OnWriteMudDate)
	}
}
