# Brief: Fix cmdTime fidelity — weekday, month names, moon phase

## Context

DP-939, DP-940, DP-941 — three Reek findings on `cmdTime` in `pkg/session/time_weather.go`.
All three are in the same function. Batch into one fix.

## Files

- **Primary:** `pkg/session/time_weather.go` (cmdTime function, ~line 133)
- **Reference:** `pkg/game/constants.go` (WeekdayNames, MonthNames, Phases arrays — all exist but are dead code)

## What's Wrong

### 1. DP-939: Missing weekday display

`cmdTime` shows: "It is 3 o'clock AM, on the 15th day of Winter Deep, Year 1."

C's `do_time` shows: "It is 3 o'clock AM, on the Day of the Dark, the 15th day of Winter Deep, Year 1."

**Fix:** Compute weekday from `(35*month + day + 1) % 7`, index into `game.WeekdayNames`. Add weekday to the output string.

### 2. DP-940: Wrong month names

`monthNames` in time_weather.go has: "January", "February", ... "December", "Frost", "Dark", "Void", "Ash", "Bloom"

C's `month_name[]` (in constants.go as `MonthNames`) has: "New Year Tide", "Winter Deep", "Snow Melt", "Spring Dawning", "Green Field", "Flower Blooms", "High Sun", "Harvest Tide", "Fruit Picking", "Leaf Fall", "First Ice", "Dark Tide"

**Fix:** Replace the local `monthNames` slice with `game.MonthNames`. The C source has exactly 12 months (not 17) — the current Go array has 5 bogus entries (Frost/Dark/Void/Ash/Bloom) that don't exist in C. Use `game.MonthNames` which has the correct 12 C fantasy names.

### 3. DP-941: Missing moon phase display

C's `do_time` shows "The moon is [phase]." when sunlight==SUN_DARK. Go's cmdTime omits this entirely.

**Fix:** After the time line, check if world is dark (hour-based: dark between sunset and sunrise — use existing `timePeriodName` or check hour < 6 || hour > 18). If dark, add: "The moon is [phase]." where phase comes from `game.Phases[(35*month + day) % 8]`.

Note: `game.Phases` currently has astronomical names ("New Moon", "Waxing Crescent"...). C's `phases[]` has descriptive names ("New Moon", "One-quarter full (waxing)", etc.). Check the C source for exact names before using. If they differ, fix the `Phases` array in constants.go too.

## Verification

1. `go build ./... && go vet ./... && go test ./pkg/session/...`
2. Check output format: weekday appears before "day of", month names match C fantasy names, moon phase appears when dark

## Notes

- All three fixes are in the same function (~15 lines of change)
- `game.WeekdayNames`, `game.MonthNames`, `game.Phases` already exist in constants.go — just not wired
- The local `monthNames` variable can be deleted entirely
