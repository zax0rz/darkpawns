# Brief: Fix moon phase hour gate to match C exactly

## Context

The cmdTime moon phase display (just shipped in the previous commit) uses `hour < 6 || hour > 18` to decide when to show the moon. This is wrong — it's wider than C's actual sunlight state machine.

## File

- `pkg/session/time_weather.go` — cmdTime function, the `if hour < 6 || hour > 18` block

## What's Wrong

**Go (current):** `hour < 6 || hour > 18` → moon shows at hours 0–5 and 19–23

**C source (`src/weather.c:57-82`):**
- Hour 5: `SUN_RISE` — suns rise, not dark
- Hour 6: `SUN_LIGHT` — day begins
- Hour 21: `SUN_SET` — suns set, not yet dark
- Hour 22: `SUN_DARK` — night begins

Moon shows when `weather_info.sunlight == SUN_DARK`, which is **hours 22 through 4** (inclusive).

**Fix:** Change the condition to:

```go
if hour >= 22 || hour <= 4 {
```

This matches C's exact SUN_DARK window.

## Cite

`src/weather.c:57-82` — `update_weather_and_time()` switch on `time_info.hours`. Hours 5/6/21/22 set sunlight state. SUN_DARK starts at hour 22, ends at hour 4 (SUN_RISE at 5).

## Verification

`go build ./... && go vet ./... && go test ./pkg/session/...`
