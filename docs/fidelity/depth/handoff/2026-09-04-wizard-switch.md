# Wizard `switch` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at `src/act.wizard.c`
`do_switch` (1175–1204), after the completed VNUM/STAT/VSTAT report slice.
The command-table entry is `interpreter.c:751` and the registered Go handler is
`pkg/session/wiz_player.go:110`. The neighboring `do_return` path already has
its own manifest and is not re-counted here.

The audit covers the reachable C branch families: already-switched, missing
argument, missing visible character, self-target, in-use body, mortal-player
level gate, successful mob switch, and successful player switch. C's
descriptor/body reassignment and the exact `OK` success byte are part of the
player-visible/state contract; existing Go switch state is explicitly
residual until the C-first vehicle proves its call path (R1/R2/R3/R5e).

## Required evidence

- Read and cite the C `do_switch` branches before changing Go; do not use the
  current M-16 behavior as authority.
- Add a C-first named-peer vehicle with disposable mob/player fixtures and
  `# depth-case:` annotations for each branch that executes.
- Run the pre-fix vehicle on this fresh main-derived branch to establish the
  RED/GREEN boundary, then run `--show-oracle` at seeds 1 and 2.
- Record exact C ranges, audience/state scope, and any intentionally blocked
  descriptor lifecycle behavior in `docs/fidelity/depth/wizard.tsv`.
- Run `make fidelity-depth`, the focused vehicle, the full build/vet/test/lint
  gate, and `gofumpt -l .` before the implementation commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.
