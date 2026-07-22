# BRIEF — Domain 7d: `time` / `weather` snapshot — O16

**For:** codex (frontier). **Owner of gate:** Claude.
**Branch:** `refactor/domain-time-weather` off `main`.
**Findings:** DP-1097 / O16 (session clock+RNG disconnected from world state).
**Part 4 (last) of Character-view.** **⚠ Weak oracle surface — primarily unit-test-gated.**
**Method rules:** read `src/act.informative.c` `do_time`/`do_weather` (1498-1563) + the DP
`time_info`/`weather_info` globals directly.

---

## 1. Oracle observation (`scenarios/character-view.txt`, verified 2026-07-15) — NOT a clean RED
`time [actor]` diff (C `-`, Go `+`):
```
-It is 9 o'clock am, on the Day of Thunder
-The 24th Day of the Month of the Old Forces, Year 1260.
+It is 12 o'clock night am, on the Day of the Dark, the 1st day of New Year Tide, Year 0.
+The moon is not in the sky.
```
Two independent simulations: C reads the **persistent world clock** (Year 1260, loaded at boot);
Go's session owns a **process-start elapsed clock** (Year 0, day 1) and a **separate 10-min random
weather cache** with no indoor gate.

**Why this is not a clean oracle red→green:** the absolute game-time (`Year 1260`) is C's saved
world state; even after Go reads its *own* game clock, the two servers won't share an epoch, so the
`time` block will still differ on the absolute date. Making it diff-clean needs a **clock-alignment
seam** (a boot-time/`time_info` freeze, analogous to the `DP_SEED` seam) — that's harness work
Claude will scope separately. **Do not** try to force the oracle green by matching C's literal
1260. The acceptance gate for THIS PR is the structural fix + unit tests (§4), plus a documented,
localized time diff (format-correct, epoch-unaligned).

## 2. Root cause
Session runs its own clock + RNG weather (pkg/session/time_weather.go:~80-147, ~154-257), divorced
from the canonical advancing time/weather in pkg/game/weather.go:~57-180 that world darkness already
consumes (pkg/game/world.go:~766-791). So `time`/`weather` can contradict actual darkness/moon/
weather events.

## 3. Faithful C reference (`do_time`/`do_weather`, act.informative.c:1498-1563)
- **`do_time`:** prints hour (`%d o'clock %s`, am/pm from `time_info.hours`), the **day-of-week**
  name, then `The %d%s Day of the Month of %s, Year %d.` (day + ordinal suffix + month name +
  year) from the global `time_info`. Read the exact am/pm/midnight/noon phrasing — Go's
  `12 o'clock night am` is malformed; match C's wording precisely. The **moon** line prints only at
  `SUN_DARK` (night) — Go prints `The moon is not in the sky.` unconditionally; gate it like C.
- **`do_weather`:** **outdoor-gated** — if `OUTSIDE(ch)` is false → `"You have no feeling about the
  weather at all.\r\n"` (verify exact string). Otherwise the sky/precip line from `weather_info`
  (`__ Sky is cloudless / cloudy / raining / lightning`, and the pressure-change direction line).
  Read the source for the exact strings and the `SKY_*` / pressure branches.

## 4. Session adoption + acceptance gate
1. Expose a **read-only, locked snapshot** from pkg/game/weather.go (game-owned `TimeSnapshot()` /
   `WeatherSnapshot()`); **delete** the session-side clock + RNG weather sim; render the snapshot.
   Preserve C's **outdoor gate** for weather and the **SUN_DARK moon gate** for time.
2. **Unit tests (the real gate):** hour→am/pm/midnight/noon wording; day-of-week + ordinal-suffix
   + month + year formatting from a fixed `time_info`; moon line present iff SUN_DARK; weather
   outdoor-gate string when indoors; each SKY_/pressure branch string. Exact C wording.
3. `time`/`weather` read the **same** state world-darkness uses (no second sim) — assert via a test
   that advances the game clock and sees `time` reflect it.
4. `make check-fmt vet` + `go test ./...` green; no WS schema break.

## 5. Sequencing note (for Claude, not codex)
O16's oracle parity is **blocked on a clock-alignment seam**. Options: (a) freeze `time_info` to a
fixed epoch in both servers under a `DP_CLOCK`-style env seam so `time` diffs clean; (b) accept
format-only unit verification and keep a documented epoch diff. Prefer (a) as a small harness
follow-on; until then this PR lands on unit tests + structural correctness. Lower priority than
7a/7b/7c — deliver after those.

## 6. Gotchas
- **Don't fake the oracle.** No touching the C world's time file; no hard-coding 1260.
- **One clock.** The whole point of O16 is to kill the second simulation — render the game
  snapshot, don't reconcile two clocks.
- **RNG:** weather transitions are RNG → Tier-2 for oracle; unit-test the render, not the sequence.
