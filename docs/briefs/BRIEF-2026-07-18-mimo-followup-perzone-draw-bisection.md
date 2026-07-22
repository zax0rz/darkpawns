# BRIEF (mimo, follow-up) — the 17366-draw gap is per-command, NOT missing zones

**Owner:** mimo. **Gate:** Claude. Follow-up to `REPORT-2026-07-18-mimo-hp-plus2-and-fixed-clock.md`. **Keep everything you measured; only the fix/attribution is wrong.**

## ⚡ RE-ACTIVATED & now tractable (2026-07-18, PRIORITY task)
Context changed in your favor. **Both engines are now DETERMINISTIC** (C seam restored PR #397; Go boot map-order fixed PR #398). So the per-zone draw counts you measure will be **STABLE run-to-run** — no more moving target. This is now the top priority; it supersedes the standalone "−1 move" fix.
**Fresh confirmation the gap is real (codex, live counters):** at do_start, base max_move=82 on BOTH (C db.c:3053 == Go player.go:325), HP draw=11 on both, but **move draw C=3 vs Go=2** — the draws are adjacent in both engines, so a disagreeing draw immediately after a matching one proves **C and Go sit at different stream positions entering character creation** (the earlier HP=21 "match" was a ~25% coincidence, not alignment). Your job: pin the exact upstream boot-draw-count delta between C and Go and the faithful fix. Re-measure the gap cleanly now that both are deterministic (the old 8904/17366 numbers predate the determinism fixes — treat them as historical, remeasure).

## What you nailed (keep it)
- The +2 is a **stream offset**, not arithmetic — C 59394 vs Go 42028 draws at the stat roll under frozen seed+clock. HP math (`do_start`+`advance_level`) verified identical. 
- **Model A confirmed:** one wall-clock-gated draw, `dice(1,50)`/`dice(1,80)` in `reset_time()` gated by `time_info.month`; Go hardcodes month 0. That's the seam target. 

## What's disproven (Claude verified, 2026-07-18 — do not repeat)
The fix "copy 150.zon/165.zon into Go" is **wrong**, same trap as PR #396:
1. `grep -E '^(150|165)\.'` on `lib/world/{zon,mob,wld,obj}/index` in the C oracle returns **nothing** — 150/165 are in **no** index manifest, so CircleMUD `index_boot()` **never loads them**. C is not running 150/165 reset commands. The `~179 extra commands` you attributed to them come from elsewhere.
2. Comparing C's `zon/index` zone list to Go's `lib/world/zon/`: Go is missing **zero** zones that C's index lists. Go holds a **superset** (extras: 33,34,57,58,85,93,94,95,97,131,132,140,160,161,162,164,180,181,184,190,213,300,301,302,303).

⇒ Go has **more** zones than C yet draws **fewer**. The 17366 gap is therefore a **per-zone-reset-command draw-rate difference** (C draws more per command), not a missing-zone problem. **Do NOT add zones to Go** — that would be a compensating draw-burn against the fidelity rule.

## A concrete lead
The gap **grew** from your prior 8904 to 17366 — exactly across PR #396, which **removed Go's mob spawn-time stat-boost double-draw** (`mob.go:113-129`, Fix B). So Fix B moved Go *away* from C on this axis. Two possibilities to resolve:
- **(i)** C legitimately draws mob HP/gold/stat-boosts *during zone-reset M-command spawns* (DeepSeek §2 lists 5 draws in `read_mobile` + 6 in `parse_simple_mob`), and Go's spawn path no longer replicates the per-spawn draws → Go under-draws on every mob the zone resets. If so, Fix B fixed a *character-creation-time* double-draw but the *zone-reset spawn* path is a separate, still-broken draw site.
- **(ii)** Fix B was wrong for this path. Less likely (its C-source reasoning was sound for creation), but rule it out.

## Method — per-zone (then per-command) draw bisection
1. Instrument `reset_zone()` on BOTH engines to emit **draws-consumed per zone vnum** (wrap the reset call, read the draw counter before/after). C: `src/db.c reset_zone`; Go: the equivalent world-reset routine. Use your existing draw-counter seam.
2. Diff per-zone draws for the zones C and Go **share** (C's index set). Rank zones by |C−Go|. The top few localize the bulk of 17366.
3. Within the worst zone(s), instrument **per-command-type** (M/O/G/E/P/D/R) draw counts. Find which command type C draws more on. Prime suspects from DeepSeek §2: `read_mobile` HP/gold (5), `parse_simple_mob` boosts (6, level>15), `init_rare` (2), `percent_load` (1), R-command random placement (2, `do-while` retry vs Go's capped attempts).
4. For the divergent command type, compare the exact `number()`/`dice()` call sequence C vs Go, line by line. Report which side is faithful to the C original and the minimal fix (a real missing/extra draw — never a burn).

## Hard rules
- Faithful fix only: match C's actual per-command draw sequence. No compensating draws, no added zones, no normalization.
- Temp instrumentation reverted + C rebuilt clean afterward (your standard).
- If a residual remains after the top-zone fix, report the per-zone delta table — don't paper it.

## Deliverable
1. Per-zone draw-delta table (C vs Go, shared zones), worst offenders ranked.
2. The divergent command type + exact call-sequence diff, with the C source that defines correct behavior.
3. Verdict on lead (i)/(ii): is the zone-reset mob-spawn draw path the culprit, and does it interact with PR #396's Fix B?
4. Minimal faithful fix (files/lines), and confirmation currently-green scenarios stay green under frozen seed+clock.
5. Instrumentation reverted, both engines build clean.
