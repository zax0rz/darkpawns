# BRIEF (mimo) — deep-dive: boot-time world-reset draw parity (+2 Go-ahead)

**Mode:** open-ended investigation → **written report**. Not a gated merge. Hammer it. Produce a root-cause report precise enough that a follow-up PR is mechanical.
**Owner:** mimo. **Context:** Dark Pawns is a faithful Go port of a C MUD; the North Star is byte-for-byte player-facing 1:1. We verify fidelity with a C↔Go differential harness under a fixed RNG seed. A residual **+2 RNG-draw offset** at boot is the last thing keeping the `combat-swing` combat scenario red. Everything else about the current combat work is green.

## The symptom (precise)
Running the oracle-diff harness on scenario `combat-swing` under `DP_SEED=1` + `DP_CLOCK=1`:
- Both engines now reach the trainee guard and swing (good). They diverge only on the **miss message variant**: C = "You swing your fist at a guard trainee, but miss him!"; Go = "You try to hit a guard trainee who easily avoids the blow." Same attack, same `messages` table (byte-identical) — a *different variant is selected* because the shared RNG stream is at a different position.
- Instrumented draw counts at the swing: **C `drawsBefore` = 4062, Go = 4064.** Go is **+2 ahead**, stable across runs, and the offset is present **before** the per-character settle-pump fires (i.e. it accrues during **boot**, not during creation/movement/combat — those are all draw-aligned).
- The variant selection is `dice(1, number_of_attacks)` in C `skill_message` (`fight.c:1035`); +2 upstream lands it on a different variant.

**Goal: pin exactly where those 2 extra Go draws (or 2 missing C draws) originate at boot, and propose the fix.**

## Already ruled out — do NOT redo
- Creation, movement, the swing itself, and the settle-pump are all draw-faithful (measured). The offset is boot-time.
- For the zones a prior pass flagged (43, 48, 70, 187–191): `.obj` and `.zon` files are **byte-identical** between the C oracle lib and the Go `lib/world`. The `.mob` "differences" are **only additive `Script: *.lua` MobProg annotation lines** in the Go format — not draw-affecting protos. Red herring.
- **Live lead:** the zone *set* differs — the C oracle lib has `150.zon` and `165.zon` that Go's `lib/world` does **not** have. Direction is puzzling (C resetting 2 extra zones would make *C* draw more, i.e. C ahead — but Go is ahead), so the net +2 must involve a countervailing Go-side draw or a per-zone accounting difference. **Measure, don't assume.**
- `quiet-mobs` (see harness) comments out **M/G/E** mob/give/equip resets but **not O/P/D/R** object/put/door/room resets — so boot object-loads still execute in both engines.

## Environment & reproduction
- Repo: this tree. C oracle (builds+boots on this Mac): `~/.openclaw/workspace/darkpawns-c-oracle`. Build it: `cd src && make` → binary at `bin/circle`.
- Both engines honor two getenv seams (byte-identical when unset): **`DP_SEED`** (deterministic RNG; C `comm.c:263`, Go `pkg/dprng` `ConfigureFromEnvironment`) and **`DP_CLOCK`** (freeze wall-clock pulses + settle-pump; C `comm.c:690`, Go `internal/dpclock`).
- Run a scenario:
  `DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" go run ./cmd/dp-oracle-diff --scenario combat-swing`
  The harness (`cmd/dp-oracle-diff/main.go`) builds the Go server from the current tree, boots both engines with `DP_SEED=1` (and sets `DP_CLOCK=1`), drives both over telnet, and diffs normalized output. **Key structural fact:** it feeds the **C oracle its own `lib`** (copied to a throwaway dir) but the **Go port its `lib/world`** — so the two engines load *different world data trees*. Any world-data divergence surfaces here. Fixtures (`spawn-mob`, `quiet-mobs`) are applied to both copies; see `applyMobFixtures`/`applyQuietMobFixtures`/`applyObjectFixtures` in `main.go`.

## Method (the decisive tool)
Differential draw-counter, the technique that cracked the prior offsets:
1. **C:** add a global `unsigned long dp_draw_count` incremented in `prng_next()` (`src/random.c`); `fprintf(stderr, ...)` the count at chosen probe points; rebuild. Both stdout+stderr are captured into the harness's per-process log buffer.
2. **Go:** add an atomic counter in `dprng.Generator.Next()` (`pkg/dprng/cmwc.go`) + a `DrawCount()` getter; print at matching probe points.
3. **Localize by bisection.** Print the draw count **per zone during the boot reset loop** on both sides (C: `db.c` boot loop that calls `reset_zone(i)` for every zone, ~lines 387–392; Go: the corresponding boot zone-reset / spawner path). Emit `zone <vnum> draws=<n>` on each side, diff the progressions, and find the first zone where the C and Go deltas diverge. Then drill into that zone's reset commands / object or mob load to find the exact `number()/dice()` call (or the missing one).
4. If the harness's disposable-world split is the culprit (Go missing 150/165, or any per-zone content delta), confirm by making the two trees match for that zone and re-measuring.

**Hard rules:** temp instrumentation in the C oracle is allowed **only if reverted and the oracle rebuilt clean afterward** (verify no `dp_draw_count`/`DP_DRAW` residue). Do **not** revert `comm.c`'s pre-existing `DP_SEED`/`DP_CLOCK` seams — edit out only your temp hunks. **Never** "fix" the offset with a compensating draw-burn, a fixture hack, or by deleting draws — those defeat the purpose. The fix must be a genuine parity fix (align the world data, or correct whichever engine's reset/load draws wrongly vs the C original).

## Key code locations
- Draw funnels: C `src/random.c` `prng_next`→`cmwc_next`; Go `pkg/dprng/cmwc.go` `Generator.Next`. `number(from,to)` draws even when `from==to` on **both** (verified) — so that's not a source.
- C boot: `src/db.c` `boot_db` (zone-reset loop ~387–392), `reset_zone`, `read_object`, `read_mobile`. Look for any `number()/dice()` in the object/mob load and reset-command execution paths.
- Go boot: zone reset / world population (`pkg/game/spawner.go`, `pkg/game` world load, `cmd/server/main.go` boot). Find every RNG draw on the load/reset path and compare call-for-call to C.
- Harness world plumbing: `cmd/dp-oracle-diff/main.go` (the `oracleData` vs `goWorld` split, fixture application).
- The variant selection that surfaces the offset (for reference only, not the bug): C `fight.c:1035` `skill_message`; Go `pkg/combat/skill_messages.go`.

## Deliverable — the report
A written report (markdown) containing:
1. **Root cause, pinned:** the exact zone(s), reset command, object/mob vnum, and the specific `number()/dice()` draw (or missing draw) that accounts for the +2 — with the per-zone draw-count evidence.
2. **Which side is wrong vs the C original:** is it a Go-lib/C-lib **data** gap (e.g. missing `150.zon`/`165.zon`, or a per-zone content delta) or a **code** divergence in a reset/load path? Cite the C source that defines correct behavior.
3. **Proposed fix**, minimal and faithful (data alignment or code change), with the files/lines to touch. Do not implement a merge here — just make it mechanical for a follow-up.
4. **Blast radius:** does this offset affect other boot states / other scenarios (it should be systemic to any scenario crossing the affected zone reset), and does the fix keep the currently-green scenarios green.
5. Confirmation that all temp instrumentation was reverted and the C oracle rebuilds clean.

If you exhaust the +2 and still have budget: audit the **entire** boot-reset draw path for other latent C↔Go divergences (this is the least-tested surface), and note any you find. Go wide.
