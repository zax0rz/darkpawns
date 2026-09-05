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

## Pre-fix result — 2026-09-04

The first C-first vehicle is red on the fresh main-derived port at
`DP_SEED=1`. C emits `No such character.`, the self-target joke, and the
in-use-body refusal, while Go emits its invented/incorrect alternatives and
accepts the live peer body. The resulting state cascade also prevents the Go
run from reaching the same mob-success and switched-NPC-gate positions as C;
the vehicle remains the falsifiable RED baseline rather than a partial claim.

The high-level mob fixture confirms an important call-path boundary: after C
successfully switches into the level-40 NPC, the next `switch` is stopped by
`interpreter.c:915-916`'s NPC/immortal gate before `do_switch` can reach its
`ch->desc->original` branch. The `do_switch` already-switched body remains a
real but separate blocked case requiring a switched, descriptor-less
high-level player target; it is not relabeled excluded (R5e/R5b).

## Result — 2026-09-04

The corrected `wizard-switch-depth` vehicle is green with `--show-oracle` at
`DP_SEED=1` and `DP_SEED=2`. It proves the exact missing-character, self,
in-use-player, mob-success, one-argument, and switched-NPC-gate cases. The
successful mob path is state-checked by the following C interpreter gate; no
room audience or save side effect is invented (R1/R2/R3/R4/R5e).

The Go handler now follows the C branch order and bytes: `No such character.`,
the self-target joke, the descriptor-in-use refusal, the Implementor-only
mortal-player check, and `Okay.` for successful body attachment. The previous
M-16 return toggle and pre-switch player saves were removed because neither is
part of `do_switch`'s C call path. The descriptor-less player success and
already-switched player body branches remain explicitly blocked because the
current vehicle cannot construct those C states (R4/R5e).

`make fidelity-depth` reports 4218 total cases, 4118 proven/delegated, 49
blocked, and 51 excluded. The full repository gates and the implementation
commit are still pending at this handoff point.
