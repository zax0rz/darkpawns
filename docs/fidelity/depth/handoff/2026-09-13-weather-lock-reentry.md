# Weather lock re-entry proof handoff — 2026-09-13

## Disposition

This is one bounded availability/fidelity proof PR for the weather lock
re-entry defect recorded by merged PR #1455. It is not Phase 6.4
global-to-struct injection. It contains no production code change and must
remain unmerged for human review.

The proof vehicle is the isolated subprocess matrix in
`pkg/game/weather_lock_characterization_test.go`. Its durable run summary is
in [`docs/fidelity/evidence/2026-09-13-weather-lock/README.md`](../../evidence/2026-09-13-weather-lock/README.md).

The worktree was created from fresh `origin/main` at
`56ab8634737b66815c4116149a5f4cab9a02b883`, the merge commit for #1455, on
`glm/audit-weather-lock`. The primary checkout's existing
`docs/specs/tui-setup-wizard.md` edit was preserved. `gh pr list --state open`
returned no open PRs on 2026-09-13, so no overlap was found and nothing was
merged.

Governing boundaries read before work: `AGENTS.md`, the amended standing
charter `/home/zach/dp-audit-PICKUP.md`, `docs/modernization/06-roadmap.md`,
`/home/zach/dp-handoff-brief-2026-09-05.md`,
`docs/fidelity/RULEBOOK.md`,
`docs/fidelity/DEPTH_TESTING.md`, and the amended
`docs/fidelity/depth/handoff/2026-09-13-modernization-phase6-4-audit.md`.
The oracle sources under `src/` and `darkpawns-c-oracle/` were read-only.

## Proven defect

The current Go ownership is:

1. `pkg/game/weather.go:170` declares the process-wide `weatherMu`.
2. `WeatherAndTime` at `:313-320` takes `weatherMu.Lock()` and holds it
   through `AnotherHour` and, when `mode` is true, `WeatherChange`.
3. `AnotherHour` at `:324-393` increments the clock, performs the event-hour
   callback, and calls the event helpers.
4. The first helper at hour 5 (`ghostShipDisappear`, `:593-603`) and the first
   helper at hour 21 (`ghostShipAppear`, `:581-591`) each execute
   `weatherMu.RLock()` before checking `weatherWorld`. The other four helpers
   have the same lock sequence.

`sync.RWMutex` is not re-entrant. A goroutine that owns its write lock cannot
acquire its read lock. The blocked stacks from the focused test identify this
exact sequence rather than an unrelated setup timeout:

```text
sync.(*RWMutex).RLock
game.ghostShipDisappear or game.ghostShipAppear
game.AnotherHour
game.WeatherAndTime
```

The nil/live matrix establishes that the nil check is too late to prevent the
block. The non-event-hour and `mode=false` controls return, demonstrating that
the failure is gated by the `mode` event switch and the post-increment hours.

Under R5e, the finding is reachable through both production callers:

- `pkg/engine/gameloop.go:325-331` invokes the weather callback at the
  63-second weather pulse. `cmd/server/main.go:324-326` supplies the live
  callback `game.WeatherAndTime(true, manager.SendToOutdoor)`.
- `pkg/session/wiz_system.go:732-744` implements `tick` and calls
  `game.WeatherAndTime(true, s.manager.SendToOutdoor)` before its later affect
  and point updates.

The test dynamically exercises only the shared `WeatherAndTime` boundary. The
heartbeat and manual command routes are established by source inspection; no
telnet or full server vehicle was added because the subprocess test is the
smallest safe proof of a package-global lock deadlock.

## Six-helper audit

All six helpers have only one production caller: `AnotherHour`. Their direct
test calls are outside `weatherMu` and therefore do not cover the re-entry
boundary.

| helper | `AnotherHour` call/order | current lock pattern | current event-hour reachability |
|---|---|---|---|
| `ghostShipDisappear` | hour 5, first helper after outdoor callback | `RLock` → copy `weatherWorld` → `RUnlock` → nil check → `SendToAll` | dynamically blocked |
| `removeNightGate` | hour 5, after ghost-ship disappearance | same | statically confirmed sibling; not reached after first block |
| `ghostShipAppear` | hour 21, first helper after outdoor callback | same | dynamically blocked |
| `loadNightGate` | hour 21, after ghost-ship appearance | same | statically confirmed sibling; not reached after first block |
| `fullMoon` | hour 21, after gate load only when `timeInfo.Day+1` is 22–25 | same | statically confirmed sibling; not reached after first block |
| `lunarHunter` | immediately after `fullMoon` under the same day condition | same | statically confirmed sibling; not reached after first block |

Thus the two dynamic event cases prove the first failing helper in both
branches; source inspection proves the same ownership pattern in all six.
The read lock protects only acquisition of the world pointer. The subsequent
`World.SendToAll` call occurs after `RUnlock`, so it is not an additional
weather-lock re-entry in the current helper bodies.

## Callback audit

`AnotherHour` and `WeatherChange` invoke `sendToOutdoor` while the enclosing
`weatherMu` write lock is held. The production callback
`manager.SendToOutdoor` at `pkg/session/session_manager.go:107-123` enters
`sendToPlaying`, marshals the message, takes the session manager read lock,
and checks player position/room. Its room lookup is the snapshot-backed
`World.GetRoom` path at `pkg/game/snapshot.go:62-69`; no path inspected here
re-enters `weatherMu`.

The event helpers' `World.SendToAll` path at `pkg/game/world.go:720-738` takes
the world read lock only after the helper released its weather read lock, then
messages players. It also has no weather-lock re-entry in the current path.

No additional concrete re-entry was found in this bounded weather critical
section. The callback parameter remains an ownership boundary: a future
callback that calls `TimeWeatherSnapshot`, `SetWeatherWorld`, or another
weather mutator would re-enter `weatherMu` while the write lock is held. The
repair must retain the current callback order and document that callback
contract rather than moving callbacks opportunistically.

## C comparison under R1/R3/R4

The C weather call path is `src/comm.c:825-831` →
`weather_and_time(1)`, and the manual path is
`src/act.wizard.c:3501-3511` → the same function. C's
`src/weather.c:31-37` calls `another_hour(mode)` and then, for mode 1,
`weather_change()`. C has no corresponding Go `RWMutex` boundary.

Event order is the same in the current Go control flow as the C source, before
accounting for the lock defect:

| post-increment hour | order in C `src/weather.c:54-80` | order in Go `pkg/game/weather.go:328-357` |
|---|---|---|
| 5 | set sunrise → `send_to_outdoor` → `ghost_ship_disappear` → `remove_night_gate` | set `SunRise` → callback if non-nil → `ghostShipDisappear` → `removeNightGate` |
| 21 | set sunset → `send_to_outdoor` → `ghost_ship_appear` → `load_night_gate` → conditional `full_moon` → `lunar_hunter` | set `SunSet` → callback if non-nil → `ghostShipAppear` → `loadNightGate` → same conditional `fullMoon` → `lunarHunter` |

The moon condition is evaluated before the day rollover in both paths:
`timeInfo.Day+1 < 26 && timeInfo.Day+1 >= 22` in Go, matching C's
`time_info.day+1 <26 && time_info.day+1 >=22`. Therefore a pre-increment day
of 21–24 qualifies (the next day is 22–25). The focused hour-20 worker seeds
day 21, but the first ghost-ship helper blocks before the moon condition can
execute. This is a reachability fact, not a claim that the current Go moon
side effects match C.

The conditional random boundaries are also recorded without changing them:

- `WeatherChange` consumes the unconditional `dice(1,4)`, `dice(2,6)`, and
  `dice(2,6)` chain after `AnotherHour`, then only consumes its extra `1d4`
  draw in the pressure/sky branches whose conditions are true. C has the same
  boundaries at `src/weather.c:137-188`; Go is at `pkg/game/weather.go:415-480`.
- C `ghost_ship_appear` consumes `number(0,1)` only after `mini_mud` is false,
  at `src/new_cmds.c:2685-2695`. The current Go helper has no corresponding
  draw.
- C `lunar_hunter` evaluates the level gate first and consumes
  `number(0,5)` once per eligible connected player, at
  `src/new_cmds.c:2662-2681`. The current Go helper has no corresponding
  draw. `full_moon` and the two gate helpers have no C RNG draw in their
  bodies.

The current Go/C event discrepancies are separate findings and are not
repaired or blessed by this proof:

- C `full_moon` transforms eligible vampire/werewolf players and emits two
  player messages (`src/new_cmds.c:1310-1329`); Go `fullMoon` broadcasts one
  custom global message (`pkg/game/weather.go:535-543`).
- C `lunar_hunter` creates a mob, relocates it, and tells an eligible target;
  Go `lunarHunter` broadcasts one custom global message
  (`pkg/game/weather.go:545-555`).
- C ghost-ship helpers create/remove exits and send room-specific messages
  (`src/new_cmds.c:2685-2740`); Go helpers broadcast custom global messages.
- C night-gate helpers create/extract portal objects according to moon phase
  and send room messages (`src/gate.c:180-230`); Go gate helpers broadcast
  custom global messages.

These differences mean this PR does not claim full weather parity and does
not add an oracle vehicle. The direct lock proof is independent of those
separate C/Go behavior gaps.

## Smallest safe repair proposal — not implemented

Do not remove the locks from the six helpers in place. That would avoid the
observed self-deadlock only by making direct/future helper callers read the
process-global `weatherWorld` without synchronization. Do not move all
callbacks outside the lock: that would change the C event/sky ordering and
would allow a concurrent snapshot or mutator to observe an interleaved
half-tick.

The smallest safe production shape is an explicit lock-owning/internal split:

1. Make `WeatherAndTime` the owner of one `weatherMu.Lock()` spanning the
   existing `AnotherHour` event order and `WeatherChange` draw order. Replace
   its nested calls with `anotherHourLocked` and `weatherChangeLocked` bodies
   that document the caller-held write lock.
2. Make the exported `AnotherHour` and `WeatherChange` entry points acquire
   `weatherMu` and delegate to those lock-held bodies. This preserves the
   already-used direct test/API entry points without requiring callers to know
   the internal ownership rule.
3. At the start of the lock-held tick, snapshot `weatherWorld` while owning
   `weatherMu`. Have `anotherHourLocked` call six no-lock event bodies with
   that stable pointer. Each body keeps the existing nil check and
   `World.SendToAll` after the pointer snapshot; it does not acquire
   `weatherMu` again. Retain small synchronized wrapper functions for the
   direct helper tests if those names remain needed; wrappers take an `RLock`,
   copy the pointer, release it, and invoke the no-lock body.
4. Keep `sendToOutdoor` in its current positions: after setting sunrise/sunset
   and before the corresponding event helpers, and keep weather-change output
   under the same enclosing write lock. Keep `WeatherAndTime(true)`'s
   `AnotherHour` → `WeatherChange` order and all existing conditional draw
   sites unchanged.

Before/after ownership is therefore explicit:

| site | current | proposed |
|---|---|---|
| `WeatherAndTime` | owns write lock across `AnotherHour` and `WeatherChange` | remains sole owner of the combined tick lock and sequencing |
| `AnotherHour` / `WeatherChange` direct entry points | lock-free bodies | synchronized wrappers over `...Locked` bodies |
| six event helpers from `AnotherHour` | each re-enters `weatherMu.RLock` | receive the world pointer captured under the write lock; no re-entry |
| direct helper wrappers | read-lock snapshot, then send | unchanged synchronized snapshot behavior |
| outdoor/event callbacks | invoked inside the write lock in C order | same positions and lock boundary |

This preserves synchronization for canonical time/weather state and
serializes `SetWeatherWorld` with the tick. It avoids the re-entry without
changing the scheduler, event order, callback order, or RNG boundaries. The
separate C/Go event side effects must remain separate follow-up work.

## Follow-up regression gates

The characterization cases must be converted after the repair; they must no
longer expect a deadlock. The completion regression should require bounded
return for hours 4 and 20 with nil/live targets, retain the non-event and
`mode=false` controls, assert hour/sunlight progression, and assert the
outdoor-before-event ordering. Add a lock-held unit matrix for all six helper
entry paths and a focused `-race` run for the synchronized wrappers.

The repair PR must also rerun the relevant weather/time depth cases and the
repository gates. It must not claim the event-output, gate state, lunar
side-effect, or lunar RNG discrepancies above as fixed without their own
source-backed proof.

## Validation boundary for this proof PR

Production files, scenario files, fixture inputs, runner scripts, `src/`, and
`darkpawns-c-oracle/` are unchanged. The whole-corpus census is reused only
under the amended charter's checkpoint rule: the accepted provenance
checkpoint is `ab916e4b21d4b99a1740521846c21b9b90b7dd23`, and the verified
production/scenario/fixture/runner inputs are unchanged from that checkpoint.
The reusable aggregate is:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
elapsed=7066.739s
```

The recovered output preserves 856 status lines and the aggregate but not all
941 per-scenario identities or temporary attempt logs. It is therefore an
aggregate comparison, not fresh full-census proof; the only non-PASS baseline
is the established human-cleared `accuse-noarg-depth` unpinnable row. Fresh
focused characterization output is the proof for this defect.

Final validation on commit `aa742b617` plus the documentation-only follow-up
was:

| gate | result | durable output |
|---|---|---|
| `make fmt`; `gofumpt -l .` | PASS; no formatting changes remained | command output was empty after the formatter line |
| `git diff --check` | PASS | command output was empty |
| `/usr/local/go/bin/go build ./...` | PASS | `/home/zach/dp-weather-lock-build-2026-09-13.log` |
| `/usr/local/go/bin/go vet ./...` | PASS | `/home/zach/dp-weather-lock-vet-2026-09-13.log` |
| `/usr/local/go/bin/go test ./...` | PASS | `/home/zach/dp-weather-lock-test-all-2026-09-13.log` |
| `/usr/local/go/bin/go test ./pkg/game/...` | PASS | `/home/zach/dp-weather-lock-test-game-2026-09-13.log` |
| `golangci-lint run ./...` with `/usr/local/go/bin` on `PATH` | PASS; 0 issues | `/home/zach/dp-weather-lock-lint-2026-09-13.log` |
| `make fidelity-depth` | PASS; 4816 total, 4697 proven/delegated, 68 blocked, 51 excluded | `/home/zach/dp-weather-lock-fidelity-depth-2026-09-13.log` |
| `make expected-divergences-check` | PASS; 26 ledger rows across 10 scenarios; pins OK | `/home/zach/dp-weather-lock-expected-divergences-2026-09-13.log` |
| focused characterization test | PASS; 4 expected deadlocks, 4 controls | `/home/zach/dp-weather-lock-characterization-2026-09-13.log` |
| focused race characterization test | PASS; 4 expected deadlocks, 4 controls, no race reports | `/home/zach/dp-weather-lock-characterization-race-2026-09-13.log` |

The full-census aggregate was reused under the checkpoint rule rather than
rerunning the 2-hour-plus corpus. The exact comparison
`git diff --name-status ab916e4b21d4b99a1740521846c21b9b90b7dd23 HEAD` contains
only the prior Phase 6.3/6.4 documentation evidence, this handoff/evidence,
the roadmap, and `pkg/game/weather_lock_characterization_test.go`; a filtered
comparison over production, scenario, fixture, and runner-input paths is
empty. The reused durable result is
`docs/fidelity/evidence/2026-09-13-phase6-3-provenance/recovered-census-output.txt`
and its accepted aggregate is
`scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0`.
The recovered stream has 856 status lines rather than all 941 scenario
identities and no temporary attempt logs, so it is an aggregate comparison
with that limitation. Its only non-PASS baseline is the human-cleared
`accuse-noarg-depth` unpinnable row; no INFRA row is being waived.

Any unexpected or flaky failure is a stop condition. The four deliberately
reproduced, isolated deadlocks are the only expected characterization result.
