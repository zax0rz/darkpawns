# Phase 6.4 disposition — 2026-09-15

Base: `ebb5feff8` (merged #1466). Zach authorized recording the dispositions
and moving to a larger bounded Phase 6.5 slice on 2026-09-15.

Phase 6.4 is **ruled under the standing charter**, with all four ownership
families retained/deferred. No global-to-struct completion is claimed.
The roadmap's dated disposition lists each family, evidence, and reopening
condition. Weather and mail correctness repairs are separate achievements,
not an implementation of the ownership refactors. No reduction in production
lines is claimed by this documentation-only disposition.

Sources:

- [Original four-family audit](2026-09-13-modernization-phase6-4-audit.md).
- [Weather lock characterization](2026-09-13-weather-lock-reentry.md) and merged #1457 repair.
- [Mail initialization repair](2026-09-14-mail-initialization-repair.md).
- [Mail reload proof](2026-09-15-mail-reload.md), following merged #1465 text repair.

#1466 records its tested implementation checkpoint as
`cbe87c8b7e67b000d0c116b22e87d094b8946294`, with a census of 941 scenarios:
931 PASS, 9 EXPECTED, 1 established accuse-noarg-depth UNPINNABLE;
failed/infra/timed_out/stale all zero. This is cited prior validation, not a
new run. This disposition changes no production, fixture, scenario, or runner
inputs and does not require another census.

Corrected stale next-step guidance: #1451 already landed the board-removal
authority fix. It is not an open next slice.

Next task: Phase 6.5 production ignored-error handling. Inventory once, select
and finish a coherent low-risk batch across files, defer behavior-sensitive
sites with exact reasons, and run one full census after freezing the batch.
Keep the mail/free-list/concurrency and other Phase 6.4 proof debts explicit.

Validation for this documentation change: make fmt, full build, vet, all Go
tests, game tests, full lint (0 issues), fidelity-depth (4,816 total;
4,697 proven/delegated; 68 blocked; 51 excluded), and
expected-divergences-check (26 rows / 10 scenarios; pins OK) passed.
No new census was run because all census inputs are unchanged.
