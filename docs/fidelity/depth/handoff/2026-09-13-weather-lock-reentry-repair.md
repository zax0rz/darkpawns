# Weather lock re-entry repair handoff — 2026-09-13

## Disposition

This is one bounded production repair for the deadlock proven by merged PR
#1456. It is a reviewable, unmerged PR. It does not implement Phase 6.4
global-to-struct injection, change scheduler timing, change the RNG algorithm,
repair C event side effects, or edit `src/` or `darkpawns-c-oracle/`.

The isolated worktree started from fresh `origin/main` at
`52b17db9b8b52240d5ede4704c3e2c15a40f5ad5`, which contains merged #1456. The
primary checkout's pre-existing `docs/specs/tui-setup-wizard.md` edit was
preserved. The open-PR overlap scan found no open weather/time/lock/scheduler/
RNG PR, and nothing was merged.

## Production delta

The smallest safe ownership repair is in `pkg/game/weather.go`:

1. `WeatherAndTime` retains one write lock across `AnotherHour` and
   `WeatherChange`, using `anotherHourLocked` and `weatherChangeLocked` so the
   combined tick does not re-enter `weatherMu`.
2. Exported `AnotherHour` and `WeatherChange` are synchronized wrappers for
   direct callers and tests.
3. The lock owner captures `weatherWorld` and passes it to six no-lock event
   bodies. Retained direct event helpers snapshot `weatherWorld` under
   `RLock()` before invoking those bodies.
4. Outdoor callbacks remain after sunrise/sunset state updates and before the
   event calls. Weather-change callbacks and all existing clock, event, and RNG
   order/conditions remain in place. The callback contract is documented:
   callbacks invoked under the write lock must not call a weather accessor or
   mutator that attempts to acquire `weatherMu`.

No weather-world struct injection was added. The production delta is the
single repair commit `eac85ac30` (`fix: prevent weather lock re-entry`).

## Completion coverage

`TestWeatherLockReentryRegression` retains bounded subprocess cleanup from the
characterization vehicle but now requires return. It covers:

- hour 4→5 and 20→21 with nil and live worlds;
- hour/sunlight progression, non-event-hour and `mode=false` controls;
- a live observer receiving exact existing Go event output in
  outdoor-before-event order;
- moon days 21 and 24 that qualify, plus days 20 and 25 that do not;
- all six event helper paths through the owner-passed bodies.

`TestWeatherEvents_SynchronizedDirectEntryPoints` covers the six synchronized
direct helper wrappers and exact broadcast strings. `TestWeatherSynchronizationRace`
exercises those wrappers, `AnotherHour`, `WeatherChange`, `WeatherAndTime`, and
concurrent `SetWeatherWorld` updates under focused `-race`. The independent
`TestWeatherAndTimePreservesWeatherDrawOrder` reference checks the three
unconditional weather dice calls, conditional `number(1,4)` branch arguments,
and next-stream position. The event-hour test distinguishes the existing
weather draws that become reachable after the deadlock is removed from any new
draw site: the repair adds none.

The existing command-level weather/time cases remain useful but bounded:
`info-basic` and `info-pulse-variants` may not reach event hours, so they do not
claim full weather lifecycle parity.

## Fidelity boundaries retained

The known C/Go weather-event differences remain separate debts: Go's custom
ghost-ship, night-gate, full-moon, and lunar-hunter output/state behavior does
not claim C parity here. The separate C/Go lunar and ghost-ship RNG/event-side
effects are not changed. Phase 6.4 weather-world injection remains
unimplemented. These are R1/R3/R4 debts, not reasons to widen this locking
repair.

## Changed files and validation record

Changed files are exactly:

- `pkg/game/weather.go` — production locking ownership and event-body split;
- `pkg/game/weather_lock_characterization_test.go` — completion,
  synchronization, and independent draw-order regressions;
- `pkg/game/weather_test.go` — synchronized direct-helper/output preservation;
- `docs/modernization/06-roadmap.md` — bounded repair status and remaining
  Phase 6.4/fidelity boundaries;
- `docs/fidelity/evidence/2026-09-13-weather-lock/README.md` — characterization
  plus repair evidence;
- this dated handoff.

The production/test checkpoint is `eac85ac30`; the documentation-only
follow-up after the final tested checkpoint must leave production, tests,
scenarios, fixtures, and runner inputs identical. The final branch head is
reported with the unmerged PR for human review.

Named coverage citations:

- source callers: `pkg/engine/gameloop.go:325-331`,
  `cmd/server/main.go:324-326`, and `pkg/session/wiz_system.go:732-744`;
- C order: `src/weather.c:31-80,137-188` (read-only reference);
- Go implementation: `pkg/game/weather.go`;
- before-fix characterization: `docs/fidelity/evidence/2026-09-13-weather-lock/README.md`;
- depth/testing boundary: `docs/fidelity/DEPTH_TESTING.md` and the
  characterization handoff `2026-09-13-weather-lock-reentry.md`.

Gate output paths and the fresh full-census reconciliation are appended to the
evidence README and this handoff after the frozen-input live runs. Any
unexpected content-red or flaky-red result remains a stop condition.

## Final gate and full-census record — 2026-09-13

The tested production checkpoint is `eac85ac30`. Formatting, build, vet, full
tests, game tests, lint, focused race checks, fidelity-depth, and
expected-divergences-check all passed. The named weather/time oracle matrix
(`info-basic` and `info-pulse-variants`, seeds `1,2,3,5,8`) had 10/10
`no normalized divergence` results. The depth gate reported 4,816 cases,
4,697 proven/delegated, 68 blocked, 51 excluded; expected-divergence pins
reported `OK` for 26 rows across 10 scenarios.

The fresh frozen-input full census completed 941/941 scenarios:

```text
passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

Its exit 2 is the established human-cleared `accuse-noarg-depth` unpinnable
baseline only. The nine expected rows are the established pinned shapes for
`accuse-depth`, `force-mob`, `medit-entry-depth`, `medit-session-depth`,
`redit-entry-depth`, `redit-session-depth`, `sedit-entry-depth`,
`sedit-session-depth`, and `shoot-target-depth`.

The frozen manifest listed 941 scenario inputs; comparison against result
identities after removing the manifest `.txt` suffix found missing 0,
unexpected 0, and duplicate 0. Complete output and durable attempt evidence
are preserved at
`/home/zach/weather-lock-fix-evidence-2026-09-13/full-census-52b17db9b-production-eac85ac30/`.
The run preserved 941 attempt-1 logs and 17 attempt-2 logs. Seven bounded
infrastructure-shaped retries (`disarm-no-weapon-depth`, `fwap-depth`,
`mount-depth`, `sing-depth`, `spank-depth`, `spec-proc-conjured-charmed`, and
`spit-depth`) recovered with no normalized divergence; each attempt-1 log
records `SYSERR: bind: Address already in use`. The other ten attempt-2 logs
are the nine pinned expected rows and the one run-varying unpinnable row.

The driver, scenario inputs, fixtures, and ledger inputs were frozen for the
live run. The only changes after `eac85ac30` are the documentation updates in
this handoff, the evidence README, and the roadmap; the final review head is
reported with the unmerged PR.
