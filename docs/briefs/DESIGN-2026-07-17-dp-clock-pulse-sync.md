# DESIGN NOTE — DP_CLOCK deterministic pulse-sync seam (DP-1162)

**Status:** proposal for Zach's approval before build. **Owner:** Claude (harness + both engines).
**Why now:** measured — combat/death fidelity is un-gateable without it (see Problem).

## Problem (measured 2026-07-17)
`combat-swing` diverges on the miss message ("swing your fist…but miss" [C] vs "wildly punch at the air, missing" [Go]). Draw-counter instrumentation proved the swing *logic* is faithful (identical `messages` file, faithful `Dice(1,N)` variant selection, consistent attack type) — the divergence is a **stream-position drift of +2 draws** at the swing. Isolation:
- **Mob-free scenarios are perfectly draw-aligned** through creation + movement + look (C=Go at every checkpoint).
- The +2 appears **only with a persistently-alive mob**, and the absolute draw counts are **scenario-structure-dependent** (identical creation gave entry draws `4066` in combat-swing vs `4058` in a scratch scenario = exactly 4 mob-ticks' difference).

**Root cause:** the trainee guard runs `mobile_activity` on a real-time `PULSE_MOBILE` cadence. C paces pulses off wall-clock (`comm.c:688` fires `heartbeat` `missed_pulses` times); the Go port runs independent real-time `time.NewTicker` goroutines. During the harness's quiescence waits, a nondeterministic, scenario-sensitive number of pulses fire and interleave differently across two independently-clocked processes. Per-draw parity (already achieved by #374/#375) cannot fix this — only a **deterministic clock** can. Every real combat scenario (multi-round fights, kills) needs a live mob, so this blocks the entire combat/death domain.

## Mechanism: env-gated deterministic clock, harness-pumped
Model on the existing `DP_SEED` seam (`comm.c:263`, getenv-gated, byte-identical when unset). Add `DP_CLOCK`:

- **When `DP_CLOCK` is set, both processes stop advancing pulses off wall-clock.** The heartbeat only fires when the harness explicitly pumps it. No real-time timers drive game state.
- **The harness pumps N pulses at deterministic scenario points**, sending the *same* pump to *both* processes. Both execute exactly N `heartbeat()` iterations at identical points relative to player commands → mob-tick draws become deterministic and aligned.
- The scenario author controls pumping via a new primitive (below). Absent any pump, mobs never tick — which is exactly what makes `combat-swing`'s single initiating swing align.

### Two phases (phase 1 de-risks before the big investment)
**Phase 1 — freeze only (small).** Under `DP_CLOCK`, don't fire heartbeat off wall-clock in either process; add no pump yet. The mob loads at boot (parity, 3 draws) but never ticks → the stream stays mob-free-aligned → `combat-swing`'s swing draws from an aligned stream → **GREEN**. Proves the diagnosis and the seam end-to-end with minimal surface.

**Phase 2 — harness-controlled pump (larger).** Add a control channel + scenario directive so multi-round and mob-active scenarios advance deterministically (drive fight rounds via `PULSE_VIOLENCE`, mob wandering via `PULSE_MOBILE`). Unblocks `combat-death` and the worker units.

## Touch points

**C oracle** (`~/.../darkpawns-c-oracle/src`, never edit outside the seam):
- `comm.c:688-689` — gate the `while (missed_pulses--) heartbeat(++pulse);` behind `!dp_clock`; under `dp_clock`, fire `heartbeat` only from pending pumped pulses. Getenv read next to the `DP_SEED` block (`comm.c:263`).
- Control input: recognize a pump trigger. Simplest = a reserved telnet line handled before `command_interpreter` so it **draws nothing** (must NOT consume the per-command `number(0,3)` hide-clear at `interpreter.c:889`).

**Go port** (`pkg/engine/gameloop.go` is already a structural analog of C's `heartbeat`):
- `GameLoop.Start` (gameloop.go:151, 100ms ticker → `heartbeat(pulse)` at :204) — under `DP_CLOCK`, don't start the real-time ticker; expose `PumpPulses(n)` that calls `heartbeat(pulse)` n times.
- **Unify the currently-separate tickers under the pumped heartbeat**, in C's dispatch order: `CombatEngine`'s own 2s tick (main.go:243 `OnPerformViolence` is a no-op today — must become a real pumped `perform_violence`), the 63s point-update ticker (`world.go:197`), and `Spawner.StartPeriodicResets` (`spawner.go:673`). This is the main Go refactor: one pumped clock, dispatch order matching `comm.c:810 heartbeat()`.
- `cmd/server/main.go` — gate ticker startup on `DP_CLOCK`; wire the pump trigger into the telnet input path (pre-interpreter, draw-neutral).

**Harness** (`cmd/dp-oracle-diff`):
- Set `DP_CLOCK=1` in both processes' env (alongside `DP_SEED`, main.go:184/193).
- New scenario section, e.g. `[pulse]` directives (`mobile N`, `violence N`, or raw `N`) interleaved with probe steps; harness sends the pump to both connections and drains.
- Quiescence still works (frozen clock just means fewer/no async bursts).

## The fidelity-critical invariant
Within a single pumped `heartbeat(pulse)`, the **dispatch order of sub-activities must byte-match C's** (`comm.c:810`: zone_update → 15s → mobile_activity → violence → point_update → …), because each draws from the shared stream. The Go refactor's ordering is the highest-risk correctness point — it must be verified draw-for-draw against C using the same draw-counter method (temp, reverted).

## Proof plan
- **Phase 1:** `combat-swing` red→green under `DP_CLOCK` (no pump). Re-run the full committed suite under `DP_CLOCK` — all currently-green scenarios must stay green (they're mob-free/quiet-mobs and draw-insensitive at the probe, so freezing the clock must not move them).
- **Phase 2:** a new `combat-death` scenario: `hit`, then `[pulse violence N]` to drive rounds to the kill, diffing each round's messages + the death/corpse output. This becomes k3's gate.

## Effort & risk
- **Phase 1:** small — ~1 getenv gate + ticker-startup guard per side (~40 lines C, ~40 Go, ~20 harness). Low risk. Highest value/effort ratio (unblocks combat-swing + proves the seam).
- **Phase 2:** medium — the Go ticker-unification refactor (fold combat/point/reset into the pumped heartbeat in C's order) + control-channel + scenario primitive. Main risk = draw-order fidelity of the unified dispatch; mitigated by draw-counter verification.

## GATE RESULT on PR #384 (2026-07-17) — freeze-only is INSUFFICIENT; needs a settle-pump
Codex's PR #384 correctly freezes the real-time pulses on both engines. Gating `combat-swing` from a worktree exposed a design gap in *this note's* Phase 1, not a coding error:
- **When the stream aligns, Go now emits C's faithful message** ("You swing your fist at a guard trainee, but miss him!") — the +2 drift is gone. **The seam concept works.**
- **But `combat-swing` still fails**, because **C's newbie birth is pulse-driven**: `SPECIAL(start_room)` (`spec_procs.c:2204`) shows the entry "dream" and *teleports* the fresh char from the staging room to their hometown infirmary (8162). It fires on a heartbeat pulse. With the clock frozen, the char is stuck in the staging room until the **first command** triggers the spec (consuming it). The Go port transports at creation, so it's already at 8162 → the warmup `north` is eaten on C only → navigation desyncs → `hit trainee` finds nothing. Proven via per-move probe diff: C's `[north]` flushes the dream + stays at Temple Infirmary while Go reaches Western Vestibule.
- Other scenarios (e.g. `movement`) pass only because their *diffed* probes come late, after some command already flushed the birth. Any scenario whose diffed probe is near entry breaks.

**Revised approach — the pump is NOT deferrable.** "Freeze-only" cannot birth a newbie. The minimal viable seam = **freeze real-time pulses + a deterministic settle-pump**: a control primitive that fires `heartbeat` a fixed N times, which the harness pumps at defined points (after boot for zone spawns; after each character's entry to fire the start_room spec + flush entry events) **identically on both engines**. Go's pump must dispatch the same activities in C's `heartbeat` order (spec_procs/event-queue included). This merges old-Phase-2's pump into the baseline. Draw-neutrality of the pump trigger still required.

## Phase 1 — worker handoff (codex)  [SUPERSEDED by the gate result above — freeze-only is not enough; implement freeze + settle-pump]
**Owner:** codex. **Gate:** Claude runs `combat-swing` red→green (worker has no `DP_ORACLE_BIN`). **Branch off `main`, one PR.**
Scope is *freeze only* — no pump, no ticker-unification (that's Phase 2). Deliverables:
1. **C** (`comm.c`): read `DP_CLOCK` via getenv beside the `DP_SEED` block (:263). When set, skip `while (missed_pulses--) heartbeat(++pulse);` (:688-689) so no wall-clock pulses fire. Byte-identical behavior when unset.
2. **Go** (`cmd/server/main.go` + `pkg/engine/gameloop.go`): when `DP_CLOCK` is set, do **not** start the real-time `GameLoop` ticker, the `CombatEngine` 2s tick, the 63s point-update ticker (`world.go:197`), or `Spawner.StartPeriodicResets` (`spawner.go:673`). No new pump API yet — just don't start the timers.
3. **Harness** (`cmd/dp-oracle-diff/main.go`): add `DP_CLOCK=1` to both processes' env (beside `DP_SEED=`, :184/:193).
4. **Do NOT** touch draw sites, message tables, or combat logic. This PR only stops timers.
**Acceptance (Claude-gated):** `combat-swing` → `no normalized divergence`; full committed suite re-run under `DP_CLOCK` stays green (mob-free/quiet-mobs scenarios must not move). Note the initiating `hit` is synchronous (not pulse-driven), so it still fires with the clock frozen — that's the point.

## Open decisions for Zach
1. **Phase 1 first, then reassess Phase 2?** (Recommended — cheap proof before the refactor.)
2. **Control channel:** reserved telnet command (simplest, reuses the existing telnet drive) vs. a control FD/signal. Recommend telnet, with explicit draw-neutrality.
3. **Pump granularity in scenarios:** explicit `[pulse violence N]` directives (precise, verbose) vs. auto-pump one PULSE between every probe step (terse, less control). Recommend explicit.
