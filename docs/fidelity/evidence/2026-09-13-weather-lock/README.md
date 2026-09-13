# Weather lock re-entry characterization — 2026-09-13

Status: proof only. This evidence does not implement a production repair, change
weather output, alter the scheduler or RNG stream, or authorize Phase 6.4
global-to-struct injection.

## Reproduction

The proof was run from branch `glm/audit-weather-lock`, based on
`origin/main` at `56ab8634737b66815c4116149a5f4cab9a02b883` (the merge of
#1455), with the primary checkout's pre-existing
`docs/specs/tui-setup-wizard.md` edit left untouched.

```text
PATH=/usr/local/go/bin:$PATH go test ./pkg/game \
  -run '^TestWeatherLockCharacterization$' -count=1 -v
```

The committed test is intentionally a defect-characterization test. Each case
launches the test binary as a subprocess. The worker waits 250 ms, captures
all goroutine stacks, and returns; the parent has a 2 s `CommandContext`
deadline that kills and cleans up the worker if setup or reporting hangs. A
case is accepted as the expected defect only when its output contains the
deadlock marker, `sync.(*RWMutex).RLock`, `WeatherAndTime`, `AnotherHour`, the
expected first event helper, and the callback's post-increment hour/sunlight
markers. A generic timeout, panic, or setup failure cannot satisfy the test.

The complete focused run was also preserved at
`/home/zach/dp-weather-lock-characterization-2026-09-13.log`.

## Matrix result

| case | setup | expected result | observed result |
|---|---|---|---|
| `event-hour-4-nil` | pre-increment hour 4, `weatherWorld == nil` | deadlock in `ghostShipDisappear` | expected deadlock; observed hour 5, sunlight `SunRise`, callback count 1 |
| `event-hour-4-live` | pre-increment hour 4, non-nil `weatherWorld` | same deadlock | expected deadlock; observed hour 5, sunlight `SunRise`, callback count 1 |
| `event-hour-20-nil` | pre-increment hour 20, `weatherWorld == nil` | deadlock in `ghostShipAppear` | expected deadlock; observed hour 21, sunlight `SunSet`, callback count 1 |
| `event-hour-20-live` | pre-increment hour 20, non-nil `weatherWorld` | same deadlock | expected deadlock; observed hour 21, sunlight `SunSet`, callback count 1 |
| `non-event-hour-7-nil` | pre-increment hour 7, nil target, mode `true` | return | returned at hour 8 |
| `non-event-hour-7-live` | pre-increment hour 7, live target, mode `true` | return | returned at hour 8 |
| `mode-false-hour-4-nil` | pre-increment hour 4, nil target, mode `false` | return without event branch | returned at hour 5; callback count 0 |
| `mode-false-hour-20-live` | pre-increment hour 20, live target, mode `false` | return without event branch | returned at hour 21; callback count 0 |

Representative stack evidence from the event-hour workers:

```text
CHARACTERIZATION_DEADLOCK case=event-hour-4-nil observed_hour=5 observed_sunlight=1 callbacks=1
goroutine 8 [sync.RWMutex.RLock]:
sync.(*RWMutex).RLock(...)
github.com/zax0rz/darkpawns/pkg/game.ghostShipDisappear()
github.com/zax0rz/darkpawns/pkg/game.AnotherHour(...)
github.com/zax0rz/darkpawns/pkg/game.WeatherAndTime(...)

CHARACTERIZATION_DEADLOCK case=event-hour-20-live observed_hour=21 observed_sunlight=3 callbacks=1
goroutine 22 [sync.RWMutex.RLock]:
sync.(*RWMutex).RLock(...)
github.com/zax0rz/darkpawns/pkg/game.ghostShipAppear()
github.com/zax0rz/darkpawns/pkg/game.AnotherHour(...)
github.com/zax0rz/darkpawns/pkg/game.WeatherAndTime(...)
```

The nil target does not avoid the defect: every event helper acquires the read
lock before checking `weatherWorld`. The live target likewise cannot avoid it.

## What this proves

The dynamic proof exercises `WeatherAndTime(true, callback)` directly in a
separate process. It proves the lock behavior, the first event reached at each
event hour, and the partial state/callback ordering before the blocked call.

The scheduled and manual entry points are source-inspection findings, not
dynamic claims in this test:

- `pkg/engine/gameloop.go:325-331` runs the weather callback every 63-second
  MUD-hour pulse, then invokes affect, point, hunt-item, and player-file work.
- `cmd/server/main.go:324-326` wires that callback to
  `game.WeatherAndTime(true, manager.SendToOutdoor)`.
- `pkg/session/wiz_system.go:732-744` implements the immortal `tick` command,
  which calls the same weather function before affect and point updates.

At hour 5, `AnotherHour` has already incremented `timeInfo.Hours` to 5, set
sunlight to `SunRise`, and called the outdoor callback before
`ghostShipDisappear` blocks. `removeNightGate`, `WeatherChange`, and every
later scheduled/manual operation are not reached. At hour 21, the equivalent
partial state is hour 21 plus `SunSet` and the outdoor callback; the first
`ghostShipAppear` call blocks before `loadNightGate`, the conditional moon
events, `WeatherChange`, or subsequent heartbeat/manual work.

## Scope boundary

The six event helpers and their callers were audited statically. The existing
`TestWeatherEvents_BroadcastToWorld` calls helpers directly without holding
`weatherMu`, so its green result does not cover this re-entry path. The
weather event output/state differences from C, the lunar RNG path, and the
gate/ghost-ship world side effects are recorded in the dated handoff, but are
not repaired or blessed here.
