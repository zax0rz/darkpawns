# Weather lock re-entry characterization — 2026-09-13

Status: characterization plus bounded repair evidence. The repair removes only
the weather lock re-entry hang; it does not change weather output, alter the
scheduler or RNG stream, or implement Phase 6.4 global-to-struct injection.
The original deadlock observations below are retained as the before-fix proof;
the completion and gate evidence is recorded after them.

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

The six event helpers and their callers were audited statically. The repaired
`TestWeatherEvents_SynchronizedDirectEntryPoints` now exercises all six helper
wrappers with a live observer and exact existing Go output expectations. The
weather event output/state differences from C, the lunar RNG path, and the
gate/ghost-ship world side effects are recorded in the dated handoff, but are
not repaired or blessed here.

## Repair evidence — tested checkpoint `eac85ac30`

The production repair keeps one write lock across `WeatherAndTime`'s combined
tick and splits the lock-held bodies from synchronized direct-entry wrappers.
The tick captures `weatherWorld` under that owning lock and passes the pointer
to no-lock event bodies. Direct helper wrappers still snapshot the pointer
under `weatherMu.RLock()` before broadcasting. Outdoor callbacks remain inside
the write-locked C order; they must not call a weather accessor or mutator that
tries to acquire `weatherMu`.

The completion subprocess matrix is
`TestWeatherLockReentryRegression`: 12 cases cover the four nil/live event
hours, two non-event controls, two `mode=false` controls, and moon days
20/21/24/25. Live cases use a player-backed `World.MessageSink`, and assert
exact output and outdoor-before-event ordering. No completion case expects a
deadlock. `TestWeatherSynchronizationRace` covers synchronized wrappers and
concurrent `SetWeatherWorld` under `-race`. The six direct helper paths are
also covered by `TestWeatherEvents_SynchronizedDirectEntryPoints`.

`TestWeatherAndTimePreservesWeatherDrawOrder` independently models the C
`weather_change` draw sequence. It verifies the three unconditional dice calls,
the conditional `number(1,4)` branches, branch arguments, and the next stream
value. The event-hour case makes clear that these are existing draws which
now execute because the old deadlock no longer prevents `WeatherChange`; no
new production draw site was introduced.

The repair remains bounded. Existing C/Go event discrepancies are separate
debts, and the named `info-basic` / `info-pulse-variants` oracle cases are
command-surface checks that may not reach event hours. They must not be read as
full weather lifecycle parity.

## Final gate and census evidence — 2026-09-13

The repair branch was tested from production checkpoint `eac85ac30` with the
following gates green:

- formatting (`make fmt`, `gofumpt -l .`, `git diff --check`), build, vet,
  full tests, game tests, and `golangci-lint run ./...` (`0 issues`);
- focused weather tests and `-race` coverage for the completion matrix,
  synchronized wrappers/direct helpers, and independent draw-order model;
- `make fidelity-depth`: 4,816 total cases, 4,697 proven/delegated, 68
  blocked, 51 excluded; actionable completion 98.6%;
- `make expected-divergences-check`: 26 ledger rows across 10 scenarios,
  pins `OK`;
- the named oracle matrix: `info-basic` and `info-pulse-variants`, seeds
  `1,2,3,5,8` (10 runs), all `no normalized divergence`.

The named command matrix is intentionally limited: these cases may not reach
the event hours exercised by the lifecycle regressions. No pins, exclusions,
or normalization changes were made for this repair.

The fresh full `make oracle-regression` used frozen inputs from
`52b17db9b8b52240d5ede4704c3e2c15a40f5ad5` through production checkpoint
`eac85ac30`, with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`,
`/usr/local/go/bin/go`, timeout `240s`, seed `1`, and four jobs. It completed
all 941 scenarios with the aggregate:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The exit status was 2 solely for the established human-cleared
`accuse-noarg-depth` unpinnable baseline. The nine `EXPECTED` rows were the
existing pinned ledger shapes: `accuse-depth`, `force-mob`,
`medit-entry-depth`, `medit-session-depth`, `redit-entry-depth`,
`redit-session-depth`, `sedit-entry-depth`, `sedit-session-depth`, and
`shoot-target-depth`.

The frozen manifest listed 941 scenario inputs. After removing the manifest's
`.txt` suffix for comparison, the durable result directory contained 941
matching identities: missing 0, unexpected 0, duplicates 0. The preserved
full output and attempt logs are outside self-cleaning directories at:

- `/home/zach/weather-lock-fix-evidence-2026-09-13/full-census-52b17db9b-production-eac85ac30/frozen-input-manifest.txt`;
- `/home/zach/weather-lock-fix-evidence-2026-09-13/full-census-52b17db9b-production-eac85ac30/full-run.log`;
- `/home/zach/weather-lock-fix-evidence-2026-09-13/full-census-52b17db9b-production-eac85ac30/live-run-snapshot/`.

There were seven bounded infrastructure-shaped first attempts, all recovered
on attempt 2 with `result: no normalized divergence`:
`disarm-no-weapon-depth`, `fwap-depth`, `mount-depth`, `sing-depth`,
`spank-depth`, `spec-proc-conjured-charmed`, and `spit-depth`. Their preserved
attempt-1 server logs all show `SYSERR: bind: Address already in use`; no
content-red or unrecovered infrastructure result occurred. The ten additional
attempt-2 logs belong to the nine pinned expected rows and the one
run-varying `accuse-noarg-depth` unpinnable row, and were retained for manual
reconciliation.

The census driver, scenario inventory, fixtures, and ledger inputs remained
frozen during the live run. The post-checkpoint changes are documentation only;
the production and regression checkpoint remains `eac85ac30`.
