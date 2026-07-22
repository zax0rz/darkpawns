# BRIEF (codex) — RANDZON/zone79 placement draw parity + determinism + index-loading

**Owner:** codex. **Gate:** Claude (has `DP_ORACLE_BIN`, oracle is clean & rebuilt). **Branch off `main`, ONE PR, THREE clearly-separable commits** (so the gate can drop the last one if needed). Re-implement in your OWN clean clone from the two reports — kimi's clone is dirty and kimi is rate-limited, so do NOT try to inherit its tree.

## Source of truth (read both)
- `docs/briefs/REPORT-2026-07-18-kimi-randzon-placement-draw-parity.md` — **root cause + the exact fix, verified draw-for-draw.** This is your primary spec.
- `docs/briefs/REPORT-2026-07-18-mimo-perzone-draw-bisection.md` — the per-zone measurements (context).

## Root cause (verified)
C's `reset_zone()` M-command runs unbounded rejection-sampling **placement loops** after `read_mobile`: `MOB_RANDZON` (db.c:2132-2145) and zone79 randload (db.c:2113-2130), each drawing one `number(0, top_of_world)` **per iteration**. That's 83% of the C↔Go boot-draw gap (14,419 of 17,353). Go under-drew because it never ran these loops faithfully. `read_mobile` HP/gold draws are already correct (PR #396 stands; mimo's "6.3/mob" was an artifact).

## Commit 1 — the placement fix (kimi's §2, re-implemented)
In `pkg/game/spawner.go`, exactly as kimi verified:
1. **RANDZON flag case fix (spawner.go:~302):** `mob.HasFlag("randzon")` never matches because `parser/mob.go` stores it uppercase `"RANDZON"` and `HasFlag` is exact-match. Fix the literal to `"RANDZON"`. *(This also fixes a real gameplay bug — RANDZON mobs were never being relocated in Go at all.)*
2. **Unbounded sampling loops:** `pickRandomRoom()` / `pickRandomZoneRoom()` use a 5-attempt cap + linear fallback. Replace with **faithful unbounded do-while** loops — one `dprng.Number(0, top)` draw per iteration, no cap, no fallback — mirroring C db.c:2113-2145.
3. **Rejection predicate:** `isRoomValidForRandZon()` wrongly rejects `SECT_CITY`; C's RANDZON loop (db.c:2133-2141) has **no** sector check. Remove the `sectCity` reject from `isRoomValidForRandZon` ONLY. Keep it in `isRoomValidForSpawn` (zone79, db.c:2125 — C DOES check sector there).

## Commit 2 — determinism (kimi's §6, REQUIRED)
The unbounded loops sample `World.Rooms()` (`pkg/game/world.go:471`), which iterates a Go **map** → the sampling slice's order is randomized per process → the rejection loop's iteration COUNT varies run-to-run (two frozen-seed Go boots gave 57,214 vs 57,387). This breaks PR #398's determinism seam. **Fix:** the placement loops must sample rooms in a **fixed order that matches C's `world[]` RNUM/load order** — i.e. rooms in the order they were parsed from the indexed wld files (index-file order, vnum-ascending within each file), NOT a live map range and NOT a naive global re-sort unless that provably equals load order. C's `number(0,top)` indexes `world[rnum]`, so for the accept/reject decisions to align **draw-for-draw**, Go's sampling array at index k must be the same room C's `world[k]` is. Build this ordered slice once at load and reuse it for placement.

## Commit 3 — index-only world loading (kimi's §5) — **GATE-APPROVED by Claude**
Go's `parser.ParseAllWldFiles` (`pkg/parser/wld.go:321`) globs **all 93** `.wld` files (9,995 rooms) via `os.ReadDir`; `ParseAllObjFiles` does the same. C's `index_boot` loads **only the index-listed files** (68 wld = 8,231 rooms). `ParseAllZonFiles`/`ParseAllMobFiles` already respect the index (`indexedDataFileNames`). **Switch `ParseAllWldFiles` and `ParseAllObjFiles` to `indexedDataFileNames` too**, so Go's runtime world == C's `index_boot` world exactly. This is the North Star (Go must load the same world C does; the 25 extra unindexed zones are not in real Dark Pawns) AND it fixes the RANDZON denominator (`valid/8231` == C, not `valid/9995`) that causes the +1,529 residual.
- Keep this a **separate commit** so the gate can drop it if a reachability problem surfaces.
- After this, Commit 2's ordered slice is naturally the index-loaded rooms in load order — verify they're the same set/order C boots with.

## Hard rules
- Faithful only: match C's loop structure, predicate, and draw order. **No compensating draws, no burn, no normalization.** If a per-zone residual remains after all three commits, **report the per-zone delta table — do not paper it** (Claude pins the remainder).
- No instrumentation left in the PR. `go build ./... && go vet ./... && go test ./pkg/game/...` clean.
- Do NOT touch the C oracle (Claude owns it; it's already clean). Do NOT touch `pkg/dprng/cmwc.go` (kimi's temp DrawCounter is only in its dirty clone, not on main).
- Out of scope (flag as follow-ups, don't do here): the other case-mismatched `HasFlag` literals (`"aggressive"`, `"mountable"`, `"MOB_MEMORY"`); the DP_CLOCK fixed-calendar (Claude's).

## Acceptance (Claude-gated, from a PR-branch worktree)
1. **Determinism restored:** two frozen-seed Go boots (`DP_SEED=1 DP_CLOCK=1`) produce **identical** total/per-zone draw counts (Claude verifies).
2. **Draw parity:** C↔Go per-zone boot-draw delta → **0** (or a small residual you report explicitly). The creation-time stream position aligns ⇒ newbie stat/move draws match.
3. **`hunger-thirst` move line green** (Go 85) and any stat-position-sensitive scenario stops diverging on stats/move. (Its separate `[quit]` block is a different brief — may keep hunger-thirst red until that lands; that's expected.)
4. **Full sweep** stays green — especially anything touching RANDZON/zone79 zones or the now-index-restricted world (watch for a scenario referencing a dropped unindexed room).
5. `DP_SEED`/`DP_CLOCK` unset ⇒ production behavior faithful (RANDZON now actually fires — that's a correct gameplay fix, not a regression).
