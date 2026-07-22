# BRIEF (mimo) — the constant +2 newbie-HP gap & validating the fixed-clock seam

**Mode:** open-ended measured investigation → written report. Same shape as your boot-draw-parity win. Do NOT merge; make the fix mechanical for a follow-up. **You have `DP_ORACLE_BIN` / the C oracle; workers like codex/k3 do not.**
**Owner:** mimo. **Gate:** Claude runs the oracle red→green after a fix lands. Tracks **DP-1177** (the +2 gap) and informs **DP-1178** (the fixed-clock seam Claude owns).

## The reframe (measured by Claude 2026-07-18 — start here, don't re-derive)
Running `hunger-thirst` (Human Warrior "Oracletst") three times back-to-back, capturing the score line from BOTH engines:
- run 1: **C HP 21**, Go HP 23
- run 2: **C HP 23**, Go HP 25
- run 3: **C HP 22**, Go HP 24

Two facts fall out:
1. **Both engines drift run-to-run** (C: 21/23/22; Go: 23/25/24) — NOT just C. The prior "C is non-deterministic" note was half the picture.
2. **The C→Go gap is a CONSTANT +2 every run** (Go = C+2 in all three). The drift is *correlated*; the +2 is *stable*.

Interpretation (your job to confirm or overturn): there are TWO independent phenomena tangled together —
- **(A) A shared wall-clock draw drift.** Both engines consume a `time(0)`-dependent number of PRNG draws during boot (C: `prng_seed(time(0))` at comm.c:263 AND `reset_time()` = `mud_time_passed(time(0), …)` at db.c:415-420; Go: whatever mirrors this). Because the calendar/daylight state changes minute-to-minute, the count of *conditionally-executed* boot draws changes, shifting the shared stream position by the time `roll_real_abils` fires. This is what Claude's **fixed-clock seam** (DP-1178) will freeze.
- **(B) A real, constant +2 Go-over-C HP fidelity bug** that survives the drift. This is a genuine divergence — Go draws/computes 2 more HP than C at the same stream position — and it will gate cleanly once (A) is frozen.

A parallel DeepSeek enumeration is done at `docs/reports/2026-07-18-c-boot-walltime-and-draws.md` — the inventory of C boot wall-clock reads and boot PRNG draw sites. **Read it first.** Its headline results and TWO surprises you must chase:
- 18 boot draw sites; **16 always execute** regardless of calendar. Only **db.c:449 / db.c:451** are calendar-gated: `dice(1,50)` in months 7–12 vs `dice(1,80)` otherwise (weather pressure). Note: both branches draw *exactly once*, so on their face they change the draw *value* but not the *count*. **Chase whether that pressure value transitively gates a LATER conditional draw** — a value-only change can't shift stream position, yet we observe a position-like drift, so either there's a downstream count dependency or the seed itself is moving.
- **db.c:421 `read_mud_date_from_file()` reads `etc/date_record`** to override the calendar. There IS an untracked `lib/etc/date_record` in the oracle tree (`git status` shows it). This file may be rewritten each boot and is a prime suspect for the run-to-run calendar (hence month-branch) movement. Inspect it and how it's written.

**Contradiction to resolve (measured, with the real oracle — this is the crux):** C's HP drift is a *tight ±1 band* (21/23/22), not the wild scatter a fully `time(0)`-reseeded CMWC stream would produce (xorshift seeding makes nearby seeds yield unrelated streams). So either (i) the harness's `DP_SEED=1` IS being honored by the binary after all — despite the pristine source and absent seam strings — and the drift is a *single* wall-clock-gated draw shifting position by one; or (ii) the seed moves but only slightly. **Pin which.** This determines whether the fixed-clock seam alone suffices, or whether a seed seam must also be built. Confirm empirically (does the C binary honor `DP_SEED`? test a fixed vs varied `DP_SEED` and watch HP).

## Mission
### PRIMARY — pin the constant +2 (B)
With the wall-clock drift held fixed (see "How to freeze it" below), the C→Go gap should collapse to a single stable delta. Find where Go gets +2 HP over C for the newbie warrior:
- Is it a **draw-count** delta right at/around `roll_real_abils` → different CON rolled → `conApp[con].Hitp` differs? Or is it **downstream of the roll** (same CON, but Go's `advance_level` / `do_start` adds 2 more — e.g. an off-by-one in the `number(11,14)` HP dice, a double-add, or a base `max_hit` mismatch)?
- Use the differential draw-counter (your proven method): temp counter in C `prng_next()` (random.c) and Go `dprng.Generator.Next()` (`pkg/dprng/cmwc.go` — note mimo's `DrawCount` field may still be present), print `drawsBefore` at `roll_real_abils`/`RollRealAbils` entry on BOTH engines under the SAME frozen clock. If `drawsBefore` matches but HP still differs by 2 → it's downstream arithmetic (compare `do_start` class.c:501 + `advance_level` class.c:600 line-by-line against `pkg/game/level.go`). If `drawsBefore` differs → it's a creation-flow draw gap (bisect the nanny states as before).
- **Hard rule:** never paper over with a compensating draw-burn or normalization. The fix is the genuine missing/extra draw or the genuine arithmetic correction.

### SECONDARY — validate the fixed-clock model (A), for Claude's seam design
Confirm the drift really is wall-clock-driven and quantify it, so Claude can build the seam with confidence:
- Show that pinning the clock (freeze `time(0)`/`time_info` to a fixed value on the C side, and the equivalent on Go) makes BOTH engines' HP reproducible run-to-run (drift → 0), leaving only the constant +2.
- Report the exact C boot site(s) whose draw *execution count* varies with the calendar (from the DeepSeek inventory + your own confirmation). Name the file:line and the wall-clock-dependent condition. This is the precise thing the seam must neutralize.

## How to freeze the clock for your measurements (temporary, your side only)
C has NO existing DP_CLOCK seam in `~/.openclaw/workspace/darkpawns-c-oracle` (verified: source is pristine `prng_seed(time(0))` / `reset_time()` with `time(0)`; binary has no DP_SEED/DP_CLOCK strings — the "seed matching" some scenarios show is normalization, not a seed seam). So to hold the clock fixed for a measurement, temporarily hardcode the two `time(0)` reads you care about (comm.c:263 seed, db.c:420 reset_time) to a FIXED constant, rebuild (`cd src && make`), and rerun. **This is throwaway instrumentation — revert both hunks and `make` clean afterward, exactly like your draw-counter cleanup rule.** Do NOT design the permanent seam — that's Claude's DP-1178. Just use a temporary freeze to isolate (B) from (A).

## Deliverable — the report
1. **The +2, pinned:** exact site and mechanism (draw gap vs downstream arithmetic), with measured `drawsBefore` on both engines under a frozen clock, and the C source that defines correct behavior.
2. **Which side is wrong** and the minimal faithful fix (files/lines).
3. **The wall-clock-gated boot draw site(s)** confirming model (A), for the seam spec.
4. **Blast radius:** the fix should move any stat-derived value (HP/mana/hitroll/saves/practices); confirm currently-green scenarios stay green under a frozen clock.
5. Confirmation all temp instrumentation (C freeze + draw counters) reverted and C rebuilds clean.
