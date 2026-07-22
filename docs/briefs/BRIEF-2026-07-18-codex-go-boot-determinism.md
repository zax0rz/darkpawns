# BRIEF (codex) — Go port drifts run-to-run despite DP_SEED=1 (boot draw-order non-determinism)

**Owner:** codex. **Gate:** Claude runs the oracle re-baseline; but **this bug reproduces in Go alone — no `DP_ORACLE_BIN` needed.** **Branch off `main`, one PR.** Tracks a new determinism ticket (Go half of DP-1177/1178).

## The bug (measured 2026-07-18, after the C seam was restored in PR #397)
With the C oracle now deterministic (`hunger-thirst` C HP = **21/21/21** across three runs), the **Go port still drifts**: newbie Human-Warrior HP = **23 / 23 / 25**, move = **83 / 86 / 84** across three identical runs under `DP_SEED=1 DP_CLOCK=1`. Go seeds the process-wide stream from `DP_SEED=1` at boot (`cmd/server/main.go:80 dprng.ConfigureFromEnvironment`, before world load at :142/:377), so a fixed seed SHOULD yield identical stats every run. It doesn't ⇒ **something consumes a non-deterministic number/order of PRNG draws during boot**, shifting the shared stream position by the time `RollRealAbils` fires during character creation. HP (from CON) and move both derive from the stat roll, so both drifting = a stream-position drift, not an arithmetic bug.

## Reproduce (Go-only, no oracle)
Boot the Go port twice with `DP_SEED=1 DP_CLOCK=1`, create a fresh Human Warrior (same keystrokes both times — see the char-creation setup letters in the oracle cookbook / `character-creation.txt`), and read `score`. The rolled stats/HP differ run-to-run. That's the whole bug; you don't need C. (You can drive it via a scripted telnet session against the Go port, or via an existing Go integration test harness if one fits.)

## Prime hypothesis — Go map-iteration randomization in the boot draw path
Go **deliberately randomizes `map` iteration order** each run. If any boot step draws from the PRNG *while iterating a map*, the draw-to-entity assignment (and often the draw count) becomes non-deterministic. Suspect the boot reset / world-load path:
- `game.LoadWorld` (`cmd/server/main.go:377`) and whatever it calls for **zone reset / mob spawn / object load** — does it iterate zones, mobs, objects, or rooms via a `map` (e.g. `map[int]*Zone`, `map[string]...`) and draw (mob HP `dice`, gold `number`, `percent_load`, stat boosts) inside that loop? If so, sort the iteration by **vnum** (ascending, matching C's file/index order) before drawing.
- Any `for k := range someMap` in boot where the body reaches a `dprng` call is a bug. Grep the boot path for `range` over maps that transitively draw.

## Secondary suspects (rule out)
- **Goroutine concurrency at boot:** if zone resets / spawns run across goroutines that share the global `dprng` stream, the mutex keeps it safe but the *acquisition order* is non-deterministic → drift. Boot draws must occur in a single deterministic sequence. Check for `go func()` / errgroup / parallel loops in the boot/reset path.
- **A stray `time.Now()` / wall-clock draw** during boot (a secondary RNG, a timestamp feeding a draw-gated branch, a `rand` from `math/rand` instead of `dprng`). Grep boot for `time.Now`, `math/rand`, `rand.`.
- **Map-ordered registration** feeding later ordered draws (e.g. building a slice by ranging a map, then drawing over the slice).

## Method (your proven draw-counter approach)
1. Add the draw counter (`dprng.Generator.DrawCount` / process-stream getter — a `DrawCount` field may already exist in `pkg/dprng/cmwc.go`; the process stream needs a getter too) and print `drawsBefore` at `RollRealAbils` entry.
2. Boot Go **twice**, same inputs, compare `drawsBefore` at the stat roll. If it differs → confirms boot draw-count non-determinism; bisect *backward* through the boot phases (world parse → zone reset → spawn → creation) to the first phase whose draw count varies run-to-run.
3. Within that phase, find the map-range-with-draw (or goroutine, or wall-clock) and make it deterministic (sorted-by-vnum iteration / single-goroutine sequence / remove the stray draw).
4. Confirm: two Go boots now give **identical** `drawsBefore` and identical stats. Revert the counter.

## Hard rules
- Faithful to C's draw ORDER: C processes zones/resets in index order (`db.c` boot loops zones in `zone_table` order; `index_boot` loads files in index-manifest order). Match that order, don't invent one. No compensating draws, no normalization.
- Do NOT change the seed, the CMWC algorithm, or draw *values* — only the *order/count determinism* of the boot path.
- `DP_SEED`/`DP_CLOCK` unset ⇒ no behavioral change (production still fine; determinism only matters under the seam).

## Acceptance (Claude-gated)
1. **Go is deterministic:** two Go boots under `DP_SEED=1 DP_CLOCK=1` produce byte-identical newbie stats/HP/move (Claude verifies via `hunger-thirst` — Go's `+` side must stop drifting; was 23/23/25).
2. With both engines now deterministic, `hunger-thirst`/`guild-practice` expose a **single stable C↔Go gap** (the real per-command draw diff, hunted separately — do NOT fix that here).
3. Full committed sweep stays green (or its greens become *stably* reproducible run-to-run).
4. Draw counter reverted; `go build ./... && go vet ./...` clean.

## Note for the gate (Claude)
This is the Go twin of the C restoration. Once it lands, BOTH engines are deterministic and mimo's per-zone bisection (`BRIEF-2026-07-18-mimo-followup-perzone-draw-bisection.md`) can finally measure a fixed target to pin the real +2.
