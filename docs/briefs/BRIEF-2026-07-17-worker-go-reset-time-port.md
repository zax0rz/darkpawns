# BRIEF (worker) — port C `reset_time()` / `mud_time_passed()` into Go (game-clock fidelity)

**Owner:** any available worker (codex/glm/k3). **Gate:** Claude (needs a DP_CLOCK fixed-time seam first — see note; the worker builds the fidelity port now, Claude adds the seam + gates). **Branch off `main`, one PR.** Tracks **DP-1178** (fidelity half only).

## The gap
Go's game clock is an admitted placeholder: `pkg/game/weather.go:79` initializes `timeInfo` to `{Hours:0, Day:0, Month:0, Year:0}` and never derives it from the C epoch (comment at weather.go:98-99 says so explicitly). C computes the calendar from real time at boot.

## C truth — reproduce exactly
- `src/db.c:415-420` `reset_time()`:
  ```c
  long beginning_of_time = 650336715;
  time_info = mud_time_passed(time(0), beginning_of_time);
  ```
- `src/utils.c:306` `mud_time_passed(t2, t1)` — converts elapsed seconds `(t2 - t1)` into `{hours, day, month, year}` using the MUD calendar constants (`SECS_PER_MUD_HOUR`, hours/day, days/month=`SECS_PER_MUD_DAY`, months/year=`SECS_PER_MUD_YEAR`). **Port this function faithfully**, including integer-division/truncation behavior, so the same `(now - beginning_of_time)` yields the identical `{hours, day, month, year}` as C.

## The work
1. Port `mud_time_passed` into Go (e.g. `pkg/game/weather.go`), using the same `beginning_of_time = 650336715` and the same calendar constants as C (`src/structs.h` / `src/utils.c` — copy the constant values, don't invent them).
2. Add a `ResetTime()` that sets `timeInfo = mudTimePassed(now, beginningOfTime)` at boot, and call it where the world initializes its clock (replacing the static Year-0 placeholder). Match C's boot ordering relative to weather init.
3. Confirm `time`, `look` (day/night), and any sunlight/weather-dependent rendering now read from the derived clock.

## ⚡ 2026-07-18 — the DP_CLOCK contract is now DEFINED (Claude's C side is committed). Implement to THIS.
Claude added the C-side seam (oracle branch `dp-oracle-seam`, commit 9462763). Under `DP_CLOCK`, C's `reset_time` now does:
```c
time_info = mud_time_passed(1770838461L, 650336715);   /* fixed timestamp */
/* and SKIPS read_mud_date_from_file() */
```
so C is deterministic. **Your Go port MUST mirror this exactly:**
1. Port `mud_time_passed` faithfully (constants: `SECS_PER_MUD_HOUR=63`, DAY=24h, MONTH=35d, YEAR=17mo; hours→day→month→year truncating in that order — see C `utils.c`).
2. In `ResetTime`, gate on `DP_CLOCK` presence (mirror `internal/dpclock.Frozen()`):
   - **`DP_CLOCK` set:** `timeInfo = mudTimePassed(1770838461, 650336715)` → **`{hours:14, day:17, month:8, year:1245}`**. Do NOT read any date_record / apply any override (C skips it too).
   - **`DP_CLOCK` unset:** faithful production behavior (compute from real time). *(Note: Go's `time` today is wall-clock-derived and non-deterministic — hours drifted 9/10/10am across runs, year/month/day 1260/Sun/30. That path is fine to keep for production; it's just not gated.)*
3. **Exact target — `character-view`'s `[time]` block, C side (match byte-for-byte):**
   ```
   It is 2 o'clock pm, on the Day of Freedom
   The 18th Day of the Month of the Dragon, Year 1245.
   ```
   (No moon line — at 2pm/daytime C's `do_time` omits it. So your `do_time` must ALSO omit the moon line in daytime; don't emit one.)
4. **Name-table gotcha (verify — the gate will catch it):** the strings above come from C's weekday / month-name tables (`constants.c`). Confirm Go's tables produce `Day of Freedom` for this weekday, `Month of the Dragon` for month index 8, and the "18th" ordinal for day 17 — byte-for-byte with C. Mismatched name tables are the likeliest failure. (day index 17 renders "18th" ⇒ display is `day+1`; match C's ordinal logic.)

### Gating
`character-view`'s `[time [actor]]` and `[time [observer]]` blocks → `no normalized divergence` (Claude runs it against the committed C seam). This is the only remaining red in the suite — closing it is 31/31.

## Historical note (superseded by the section above)
C's calendar is **wall-clock-derived**, so the `time` output is non-deterministic run-to-run. **Claude will add a DP_CLOCK extension** that feeds `reset_time` a FIXED timestamp in test mode (both engines) so `character-view`'s `time` block becomes deterministic and gateable. Until then, verify your port by unit test: assert `mudTimePassed(650336715 + KNOWN_SECONDS, 650336715)` produces the expected `{hours,day,month,year}` for a couple of hand-computed timestamps. **Do NOT** hardcode the current date or normalize the time away.

## Acceptance
1. `go build ./... && go vet ./...` clean; unit test on `mudTimePassed` with known inputs passes.
2. A live boot shows `time` advancing from a real-time-derived date (not Year 0).
3. `DP_SEED`/`DP_CLOCK` unset ⇒ behaves like C's normal `reset_time`. (Claude gates the oracle `time` block after adding the deterministic seam.)
