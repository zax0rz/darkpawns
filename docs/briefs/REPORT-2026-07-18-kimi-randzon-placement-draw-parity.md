# Report: Per-Zone Draw Bisection — ROOT CAUSE FOUND (RANDZON/zone79 placement loops)

**Status:** Root cause pinned and verified by direct measurement. Fix implemented and validated (gap −17,353 → +1,529). **Two follow-ups remain** (see §6, §7): a determinism regression introduced by the fix, and a structural world-size residual. **Instrumentation NOT yet reverted** in either tree (see §8).

Supersedes the open question in `REPORT-2026-07-18-mimo-perzone-draw-bisection.md`.

## 1. Executive summary

Mimo's "6.3 extra draws per mob in C" was an **attribution artifact**. C's `read_mobile()` draws **exactly** `hpnum + 2·(gold≠0)` per spawn — verified with per-spawn instrumentation on the C oracle: 2695 spawns logged at boot, **zero deviations** from the model. The real gap is the **random-placement loops after `read_mobile`** in C's `reset_zone()` M-command: the `MOB_RANDZON` do-while (db.c:2132-2145) and the zone79 randload do-while (db.c:2113-2130). These loops consume one `number(0, top_of_world)` draw **per rejection-sampling iteration** — tens to hundreds of draws per mob. Measured on the C oracle: **664 M commands with placement draws, totaling 14,419 draws — 83% of the 17,353 gap**, and 100% of those 664 commands were RANDZON-flagged or zone79 mobs (no exceptions).

Per-zone confirmation: zone 151 (Great Plains) placement draws = 3839, exactly mimo's C−Go delta for that zone (3839). Zone 122 (Swamp): 1572 vs 1572. Zone 91 (Great Forest): 2345 vs 2354.

## 2. Go bugs found and fixed (in working tree, `pkg/game/spawner.go`)

1. **`mob.HasFlag("randzon")` case mismatch (spawner.go:302).** `parser/mob.go`'s `actionBitNames` stores the flag as `"RANDZON"` (uppercase); `HasFlag` is an exact-match loop, so Go **never fired RANDZON placement at all** — zero placement draws, and RANDZON mobs were never relocated (also a gameplay bug, not just a draw bug). Fixed to `"RANDZON"`. Note: the same case-mismatch pattern exists elsewhere (`"aggressive"`, `"mountable"`, `"MOB_MEMORY"` in spec_procs3.go / other_mount.go / combat_wire.go) — out of scope, flagged for follow-up.
2. **Capped sampling loops.** `pickRandomRoom()` / `pickRandomZoneRoom()` used a 5-attempt cap + linear-scan fallback (≤5 draws). C's loops are **unbounded do-while**. Replaced with faithful unbounded loops (one `dprng.Number(0, len(rooms)-1)` draw per iteration, no cap, no fallback).
3. **Wrong rejection predicate.** `isRoomValidForRandZon()` rejected `SECT_CITY` rooms; C's RANDZON loop (db.c:2133-2141) has **no** sector check (only the zone79 loop does, db.c:2125). Removed the `sectCity` check from `isRoomValidForRandZon` only; `isRoomValidForSpawn` (zone79) keeps it, matching C.

Mimo's hypothesis (HP dice guard skipping draws / mob-file HP mismatch) is **disproven**: all 1319 shared mob prototypes have identical HP dice in both trees, and C's per-spawn draws match `hpnum + 2·goldflag` exactly.

## 3. Verdict on the brief's leads

- **Lead (i) — zone-reset mob-spawn path is the culprit: CONFIRMED**, but the divergent site is the *placement* loops, not HP/gold/stat-boost draws.
- **Lead (ii) — PR #396's Fix B was wrong: RULED OUT.** C's `read_mobile` does not re-roll stat boosts at spawn; per-spawn draws are HP dice + gold only. Fix B stands.

## 4. Verification (both engines frozen: DP_SEED=1, DP_CLOCK=1)

C oracle instrumented with a draw counter in `prng_next()` and per-command/per-zone logging in `reset_zone()`; same per-command logging added temporarily to Go's `ExecuteZoneReset`. C totals reproduced mimo's numbers exactly (55,685), validating the method.

After the fix, per-zone draw totals (68 zones reset on both):

- **Total: C = 55,685, Go = 57,214, delta = +1,529** (was −17,353 before the fix — 97% of the gap closed).
- Worst remaining per-zone deltas (run 1): zone 91 +2,846; zone 151 +544; zone 80 −475; zone 182 −422; zone 142 −348; zone 27 −255. Mixed signs.
- `go build ./...`, `go vet ./pkg/game/`, `go test ./pkg/game/...` all green after the fix (pre-instrumentation).

## 5. Why any residual remains: structural world-size difference

`parser.ParseAllWldFiles` (`pkg/parser/wld.go:321`) **globs all 93 `.wld` files** (9,995 rooms) via `os.ReadDir`, while C's `index_boot` loads only the **68 index-listed files** (8,231 rooms). `ParseAllZonFiles` and `ParseAllMobFiles` respect the index (`indexedDataFileNames`); `ParseAllWldFiles` and `ParseAllObjFiles` do not. Both trees' `wld/index` files are identical (68 files, same 8,231 rooms; room records byte-identical).

Effect: Go's RANDZON acceptance probability is `valid/9995` vs C's `valid/8231` — Go expects ~21% more iterations per placement loop (~+3,000 draws over 14,419, partially masked by noise). This is the bulk of the +1,529 residual. Fixing it means switching the wld (and probably obj) loaders to the index — that **removes 25 extra zones' rooms from Go's runtime world**, a gameplay-visible change (reachability, teleports) that needs a gate decision. Per the brief's hard rules (no zone-set changes), I did not make this change.

Smaller unexplained pre-fix discrepancies (Alaozar −152, Kir Drax'in −273 vs data model) may be capped-spawn/if-flag ordering differences; re-check after §6 lands.

## 6. NEW ISSUE introduced by the fix: run-to-run nondeterminism

Two consecutive frozen-seed Go boots (post-fix) produced **different draw totals** (57,214 vs 57,387; 33 zones differ). Cause: `World.Rooms()` (`pkg/game/world.go:471`) iterates a Go **map** — the sampling slice's order is randomized per process, so the unbounded rejection loop's iteration count varies run to run. The old 5-attempt cap masked this (draw count bounded regardless of order). **This must be fixed before merge** or the determinism seam (PR #398) is broken for any scenario touching RANDZON/zone79 resets: build the sampling slice in a deterministic order (e.g., rooms sorted by vnum once at load, mirroring C's `world[]` array order — C's `world[to_room]` indexing is load-order, which is vnum-ordered per indexed file sequence).

## 7. Recommended next steps (in order)

1. Deterministic room ordering for `Rooms()` (or a dedicated rnum-ordered slice for the placement loops). Re-run two frozen boots; expect zero per-zone deltas between Go runs.
2. Re-measure C vs Go per-zone; residual should shrink to the wld-glob effect (~+21% on placement loops, ≈ +3k draws).
3. Gate decision: switch `ParseAllWldFiles`/`ParseAllObjFiles` to `indexedDataFileNames` (matches C's `index_boot`; removes unindexed content from runtime) or accept the residual.
4. Re-check small residuals (Alaozar, Kir Drax'in) after 1–3.
5. Follow-up bug (separate PR): case-mismatched `HasFlag` string literals (`"aggressive"`, `"mountable"`, `"MOB_MEMORY"`).

## 8. Tree state (NOT clean — instrumentation still in place)

**Go repo (this tree):**
- `pkg/game/spawner.go` — the fix (§2) **plus** temp DPDRAW per-command/per-zone logging + `fmt`/`os` imports. Revert the logging only; keep the fix.
- `pkg/dprng/cmwc.go` — temp `DrawCounter atomic.Uint64` + increment in `Generator.Next()`. Revert entirely.
- Fix-only state passed `go build ./... && go vet ./pkg/game/ && go test ./pkg/game/...`.

**C oracle (`~/.openclaw/workspace/darkpawns-c-oracle`):**
- `src/random.c` (`dp_draw_counter`), `src/random.h` (extern), `src/db.c` (`#include "random.h"`, per-spawn `DPDRAW M`, per-command `DPDRAW CMD`, per-zone `DPDRAW ZONE` logging) — all temp, **not reverted**; `bin/circle` currently contains the instrumented build. To restore: revert those three files (`git checkout -- src/db.c src/random.c src/random.h` — the oracle is a git repo, clean except `lib/etc/date_record`) and `make`.

**Measurement logs (tmp):** `/tmp/dp-c-boot2.log` (C per-command, authoritative), `/tmp/dp-go-boot2.log` / `/tmp/dp-go-boot3.log` (Go runs 1/2 — differ, see §6), `/tmp/dp-c-boot.log` (C per-spawn read_mobile proof).

## Deliverable status vs brief

| Deliverable | Status |
|---|---|
| 1. Per-zone draw-delta table | Done (mimo's) + post-fix table (§4) |
| 2. Divergent command type + call-sequence diff | Done — M command, but the placement loops (db.c:2113-2145), not `read_mobile` |
| 3. Verdict on lead (i)/(ii) | (i) confirmed with corrected site; (ii) ruled out — PR #396 stands |
| 4. Minimal faithful fix | Implemented (3 changes, §2); **not mergeable until §6 determinism fix lands** |
| 5. Instrumentation reverted, both build clean | **Not done** — see §8 |
