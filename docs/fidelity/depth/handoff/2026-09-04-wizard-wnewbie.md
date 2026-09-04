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

## Pre-fix result — 2026-09-04

The C-first vehicle was red on the fresh `origin/main`-derived port at
`DP_SEED=1` and `DP_SEED=2`. The no-argument path normalized green, but the
missing-target text differed and every valid C target lacked the target-only
gesture line in Go:

```text
wnewbie Nobody [actor]       C: No-one by that name here.
                             Go: No one by that name online.
wnewbie Wnewbieglobal        C: Newbied. + target gesture
                             Go: Newbied. only
wnewbie Wnewbiepeer / prefix C: Newbied. + target gesture
                             Go: No one by that name online.
wnewbie self                 C: Newbied. + self gesture
                             Go: No one by that name online.
```

The C global-player case proves that `get_char_vis` falls back from the room
scan to an exact visible player lookup; the in-room abbreviation and
case-folding cases prove the `isname_with_abbrevs` room path. The missing
target and target-only audience are direct handler divergences. Object state
and list order were not inferred from the transcript and remain a focused
state proof item (R1/R2/R3/R5e).

## Implementation and proof — 2026-09-04

`cmdNewbie` now consumes the first C-compatible argument, resolves the target
through the shared `get_char_vis`-equivalent world resolver, creates object
prototypes 8019, 8062, 8063, and 8023 in C's source order, and places them
through the canonical world object movement paths. Player placement uses the
C-compatible prepend behavior, so the resulting order is club, skin, bread,
tunic. Mob placement uses the corresponding front insertion path. The
handler emits the exact actor acknowledgement and routes the target-only
gesture through the canonical `Act` substitution path; missing and empty
target responses now preserve C's NOPERSON and CRLF bytes.

The focused unit tests prove registration, player state/order and audience
separation, mob state/order, and empty/missing target bytes. The C-first
vehicle is green with `--show-oracle` at both `DP_SEED=1` and `DP_SEED=2`:

```text
wizard-wnewbie-depth: result: no normalized divergence
```

It covers the existing empty-argument context, missing target, global exact
player lookup, in-room player lookup, abbreviation, case folding, trailing
words, and self targeting. The live world has no mob at the selected vehicle
location, so the mob target is recorded with focused state proof rather than a
misclassified missing-target transcript (R1/R2/R3/R5e).

## Verification

The required verification completed on 2026-09-04:

- `make fidelity-depth` — 4280 total, 4175 proven/delegated, 54 blocked, 51
  excluded; 98.7% actionable completion.
- `go test ./pkg/session -run 'Test(Wnewbie|CmdNewbie)' -count=1` — passed.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The focused `wizard-wnewbie-depth` oracle matrix also produced no normalized
divergence at `DP_SEED=1` and `DP_SEED=2`. No files under `src/` or
`darkpawns-c-oracle/` were changed.
