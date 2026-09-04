# Wizard valid-report slice — 2026-09-04

## Scope

This slice continues the verified source-order path in `src/act.wizard.c`.
The prior wizard slice proved the registered entry, parser, gate, and missing
target branches for `transfer`, `teleport`, `vnum`, `stat`, and `vstat`, but it
left the valid report bodies explicitly unproven. This slice targets the live
mobile/object lookup reports for `vnum`, the room/player/mobile/object report
dispatch for `stat`, and the prototype report bodies for `vstat`.

The C definitions and command-table registrations will be rechecked before
each vehicle. Shared report/rendering behavior will be assigned to the
appropriate owning manifest rather than counted twice (R2/R5b/R5c). The
valid-report output is a large byte surface, so a green error vehicle will not
be promoted to body coverage (R1/R4).

## Required evidence

- Create a C-first valid-report vehicle with stable world fixtures and named
  actor/observer peers where `stat room` exposes room topology.
- Run every selected vehicle with `--show-oracle` at seeds 1 and 2, confirming
  that the intended C report blocks execute.
- Record each command branch in a manifest with the exact C source range and
  any remaining shared or state gap.
- Run `make fidelity-depth` and all four repository gates before committing the
  implementation, then merge a green self-PR before the next source-order
  slice.

No C or oracle-tree files are to be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## Result — 2026-09-04

The new `wizard-valid-reports-depth` vehicle is green with `--show-oracle` at
`DP_SEED=1` and `DP_SEED=2`. It executes all ten annotated cases: both VNUM
lists, the room report, live mob/player/file reports, typed and fallback object
reports, and both valid prototype reports. The C blocks are non-empty and the
normalized transcripts match byte-for-byte on both seeds (R1/R2/R3/R5e).

The Go changes replace the previous abbreviated stat output with the shared C
character/object report shapes, add C-specific flag/race/position renderers,
load the `stat file` player record, and route `vstat mob` through the same
character report path. Shared `do_stat_character` and `do_stat_object` ownership
is recorded as delegated for the corresponding `vstat` rows (R5b/R5c).

The required repository checks for this slice passed before the implementation
commit: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, and `gofumpt -l .`. The pre-existing untracked
`website/static/images/` directory remains outside the slice.
