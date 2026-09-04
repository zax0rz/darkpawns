# Wizard `zlist` depth audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:3545-3574`, covering `do_zlist` and the C command-table
entry at `src/interpreter.c:847`. The existing manifest proves only the
current-zone no-argument path through `wizard-residual-depth`; this audit
expands numeric selection, decimal-prefix parsing, missing-file output,
current-zone defaulting, bounded file bytes, and pager behavior.

The C handler calls `one_argument`, converts the first token with `atoi`,
defaults an empty token to the current room's zone-table number, opens
`world/zon/<number>.zon`, reads at most `MAX_STRING_LENGTH-5` bytes, and sends
the result through `page_string`. A missing file sends the exact refusal
line. The shared pager implementation is not re-counted here; this slice
proves the command-to-pager boundary and delegates pager navigation to the
pager manifest (R1/R2/R3/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:3545-3574`, `src/interpreter.c:847`, the
  actual `one_argument` implementation, and the `page_string` call path before
  changing Go.
- Create a C-first vehicle covering no-argument current-zone selection,
  explicit numeric selection, signed/decimal-prefix `atoi`, trailing-token
  ignoring, and a missing zone file. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes. Distinguish
  command-file selection from shared pager output and do not infer file-read
  truncation from a short real fixture (R1/R2/R3/R5e).
- Fix only confirmed divergences, preserve C's file bytes and exact refusal,
  and record any pager-only continuation gap as delegated to the pager
  surface (R1/R2/R3/R5e).
- Run `make fidelity-depth`, the focused oracle matrix, and all repository
  build, vet, test, lint, formatting, and security gates before the
  implementation commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_zlist` uses `one_argument` and `atoi`; an empty argument selects
`zone_table[world[ch->in_room].zone].number`, while any nonempty first token
selects its signed decimal prefix. It constructs `world/zon/%d.zon`, reads
the file into the fixed C buffer, and calls `page_string(ch->desc, buf, 1)`.
If `fopen` fails, C sends `No zone file for that number.\r\n`. The C command
table gates `zlist` at `LVL_BUILDER` and `POS_DEAD` (R1/R2/R5e).

## Pre-fix result

The C-first vehicle was run on the fresh `origin/main`-derived port at
`DP_SEED=1` and `DP_SEED=2` before implementation changes. Missing-file,
decimal-prefix, explicit numeric, nonnumeric-`atoi` zero, trailing-argument,
and current-zone default blocks were green at both seeds. The long `zlist 27`
probe was red only at the shared pager prompt:

```text
zlist 27: C page 1/15; Go page 1/23
```

The first page bytes and `q` pager exit were otherwise equal. The disposable
`quiet-mobs` fixture prefixes reachable reset lines and expands the selected
file beyond C's fixed `MAX_STRING_LENGTH` buffer. C's `fread` therefore exposes
only its bounded prefix, while Go passed the complete file to `PageString` and
counted eight additional pages. This is a confirmed command read-boundary
divergence, not a pager navigation inference (R1/R3/R5e).

## Implementation and proof

To be filled after the confirmed C read bound is implemented and the focused
vehicle is green at both seeds.

## Verification

To be filled after the full fidelity, build, vet, test, lint, formatting, and
security gates complete.
