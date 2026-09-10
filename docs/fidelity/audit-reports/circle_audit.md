# Port Fidelity Audit: Module 16 (`circle.c` / `gameloop.go`)

This audit examines the port fidelity between the legacy C source file `src/circle.c` (the main entrypoint wrapper that calls `dp_main()`) and its Go counterparts: the heartbeat orchestrator in `pkg/engine/gameloop.go` and the server bootstrapper in `cmd/server/main.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/circle.c` (30 lines)
- **Functions**: `main` (the primary application entrypoint; immediately delegates execution to `dp_main` in `src/comm.c` to bootstrap socket descriptors, catch signals, and spin the game loop).

### Go Port Files
- **Orchestration & Timing**:
  - `pkg/engine/gameloop.go` (Active; implements `GameLoop`, `GameLoopCallbacks`, and the `heartbeat()` pulse dispatcher which fires every 100ms)
  - `cmd/server/main.go` (Active; the main Go program entrypoint that imperative-boots all database, scripting, and socket systems, starts the HTTP/Telnet listeners, catches signals, and instantiates the `GameLoop`)
  - `pkg/game/ai.go` (Defines fallback world tickers `StartAITicker` and `StartPointUpdateTicker` to run mob AI and regens independently)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Game Heartbeat is a Hollow Shell (Time is Frozen, Spells Last Forever, Events are Dead)
- **Source Context**: `cmd/server/main.go#L161-L175`, `pkg/engine/gameloop.go#L186-L275` (`heartbeat`)
- **Fidelity Bug**: The heartbeat loop orchestrator `gameloop.go` is a beautifully written, extremely faithful pulse-timer system. However, the server's main boot routine in `main.go` instantiates `GameLoopCallbacks` with **only three** fields: `OnPointUpdate` (health/mana regen), `OnPerformViolence` (stubbed/empty), and `OnMobileActivity` (stubbed/empty). 
  
  All other critical MUD system ticks are left completely `nil`:
  - **`OnEventProcess` (100ms) is `nil`**: The entire priority **events queue** (spells casting delays, combat delay ticks, timer scripts) **never ticks** in the main loop. Any event added to the queue is never processed and sits there forever.
  - **`OnWeatherAndTime` (60s) is `nil`**: The MUD game clock **never advances** (it is perpetual day or night), and weather states (clouds, pressure changes, rain) never tick.
  - **`OnAffectUpdate` (mud hour) is `nil`**: Timed player affects (bless, armor, blind, poison, stat boosts) **never expire or decay**. Buffs and debuffs last infinitely until the player logs out.
  - **`OnCheckIdlePasswords` is `nil`**: Connection handshakes never time out.

### 2. Standalone World Tickers are Never Started (Mobs are Statues)
- **Source Context**: `pkg/game/ai.go#L181-L224` (`StartAITicker`), `cmd/server/main.go`
- **Fidelity Bug**: To make matters worse, the backup background tick routines implemented on the world struct in `ai.go` (`StartAITicker` which ticks mob AI/wandering and starts the standalone `EventQueue.Start(ctx)` loop) are **never called anywhere in the server bootstrap**.
  As a result:
  - Mobs **never act or wander**; they stand completely still like statues forever.
  - The standalone event queue processing goroutine is **never spawned**, compounding the dead-events-queue bug.

### 3. Modulo Documentation Contradiction
- **Source Context**: `pkg/engine/gameloop.go#L207-L210`
- **Fidelity Bug**: The Go check-idle callback timer contains a comment typo:
  ```go
  // 15 * PASSES_PER_SEC → every 1.5 seconds
  if pulse%(15*PASSES_PER_SEC) == 0 && cb.OnCheckIdlePasswords != nil {
  ```
  Since `PASSES_PER_SEC = 10`, `15 * PASSES_PER_SEC` is `150` passes. At 100ms per pass, this is exactly `15 seconds`. The comment erroneously states this runs every `1.5 seconds`.

### 4. Mud Hour Timing Drift
- **Source Context**: `pkg/engine/gameloop.go#L30` (`SECS_PER_MUD_HOUR`)
- **Logic Gap**: In stock CircleMUD, the default configuration has `#define SECS_PER_MUD_HOUR 75` (75 real-world seconds per Mud hour).
- **Fidelity Bug**: Go hardcodes `SECS_PER_MUD_HOUR = 60` (60 real-world seconds). This speeds up time progression by 25%, causing weather and affect updates to cycle slightly too quickly compared to the legacy MUD speed.

---

## 3. Go Improvements Over C

### 1. Robust Lifecycle Signals
- **Fidelity Improvement**: Legacy C handled shutdowns brutally or required custom signal piping in `comm.c` which could crash thread states. Go's `main.go` cleanly registers signal listening (`signal.Notify` for SIGINT/SIGTERM) and triggers a thread-safe graceful session save (`manager.ShutdownGracefully`) and dynamic world dump (`game.SaveWorld`) before closing files.

### 2. Type-Safe Heartbeat Registry
- **Fidelity Improvement**: C's heartbeat loop mixed low-level network I/O select loops directly with game ticks in a single massive while loop in `comm.c`. Go decouples this cleanly: `gameloop.go` manages only the timing and delegates business logic cleanly via Go interfaces (`GameLoopCallbacks`).

---

## 4. Concurrency & Thread Safety

- **Atomic Pulse Counters**:
  - The pulse timer uses `atomic.Int64` inside `GameLoop` for increments and reads. This is fully thread-safe for concurrent read queries (like uptime queries from web admin panel).
- **Heartbeat Callback Deadlocks**:
  - Heartbeat callbacks (`OnPointUpdate`, etc.) run inside a single, dedicated game loop goroutine. 
  - Care must be taken that these callbacks do not perform blocking operations or acquire world locks in a way that causes deadlocks with session reader/writer threads.

---

## 5. Summary of Recommended Fixes

1. **Wire All Dead Heartbeat Callbacks**:
   Update `cmd/server/main.go#L161-L175` to register all missing `GameLoopCallbacks` properly to their respective world and session routines:
   - Wire `OnEventProcess` to `func() { gameWorld.EventQueue.Process(context.Background()) }`.
   - Wire `OnAffectUpdate` to `func() { gameWorld.AffectUpdate() }`.
   - Wire `OnWeatherAndTime` to execute weather ticking.
   - Wire `OnCheckIdlePasswords` to session idle checks.
   - Wire `OnMobileActivity` to `func() { gameWorld.AITick() }` so mobs can wander and execute AI.
2. **Correct the Modulo Comment**:
   Fix the typo in `gameloop.go` to correctly document `15 * PASSES_PER_SEC` as `15 seconds`.
3. **Harmonize `SECS_PER_MUD_HOUR`**:
   Adjust `SECS_PER_MUD_HOUR` to `75` in `gameloop.go` to match standard CircleMUD time speeds.
