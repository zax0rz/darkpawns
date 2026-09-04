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
