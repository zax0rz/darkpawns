# Wizard `wnewbie` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:3516-3544`, covering `do_newbie` and the C command-table
entry at `src/interpreter.c:835`. The existing manifest proves only the empty
argument refusal through `wizard-residual-depth`; this audit expands target
resolution, object creation, inventory state, actor acknowledgement, target
audience, and first-token parsing.

The C handler calls `one_argument`, resolves a visible character with
`get_char_vis`, creates prototypes 8019, 8062, 8063, and 8023 in that order,
prepends each object to the target's carrying list through `obj_to_char`,
acknowledges the wizard, and emits a target-only `act()` gesture line. The
shared object movement/inventory implementation is not re-counted here; this
slice proves the command's call boundary, exact bytes, target classes, and
state ordering (R1/R2/R3/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:3516-3544`, `src/interpreter.c:835`, the
  actual `src/handler.c:get_char_vis` path, and the object creation/placement
  call path before changing Go.
- Create a C-first vehicle with a visible peer covering empty, missing,
  successful player target, case-insensitive/abbreviated target resolution,
  and trailing-word parsing. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes; distinguish
  target/output drift from shared object movement or prototype behavior.
- Fix only confirmed divergences, preserve C's object order and exact actor /
  target audiences, and record any deeper shared object movement gap as
  delegated rather than inferring it from the command output (R1/R2/R3/R5e).
- Run `make fidelity-depth`, the focused oracle matrix, all repository build,
  vet, test, lint, formatting, and security gates before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_newbie` uses `one_argument`; an empty token sends
`Whom do you wish to newbie?\r\n`, and an unresolved `get_char_vis` target
sends `NOPERSON`, defined as `No-one by that name here.\r\n`. A successful
target receives four `obj_to_char` objects in source-array order, which means
the final carrying-list order is club, skin, bread, tunic because C prepends
each object. The issuing wizard receives `Newbied.\r\n`; the target alone
receives `The wizard makes a magickal gesture, creating a bunch of equipment,
and hands it to you!\r\n` after C `act()` substitution/capitalization.

## Pre-fix result

To be filled after the C-first vehicle is run on the fresh implementation at
`DP_SEED=1` and `DP_SEED=2`, before code changes.

## Implementation and proof

To be filled after the confirmed target, object, audience, and parser changes
are implemented and the focused vehicle is green at both seeds.

## Verification

To be filled after the full fidelity, build, vet, test, lint, formatting, and
security gates complete.

