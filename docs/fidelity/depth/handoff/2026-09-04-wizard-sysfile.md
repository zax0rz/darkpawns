# Wizard `sysfile` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:3412-3443`, covering `do_sysfile` and its command-table
entry at `src/interpreter.c:752`. The existing manifest proves only the empty
argument refusal through `wizard-residual-depth`; this audit expands C's
abbreviated selector, case-folding, invalid-selector, missing-file, and pager
branches without inventing static files that are absent from the checked-in
world trees.

The C handler consumes the first `one_argument`, matches `bugs`, `ideas`,
`todo`, and `typos` through case-insensitive prefix `is_abbrev`, reads the
corresponding `misc/*` path, reports the exact missing-file response on read
failure, and sends successful contents through the descriptor pager. The
command table supplies the `LVL_GOD` gate. The successful file-read/pager
branch has no stable live vehicle while both the port and oracle trees lack
the four C `misc/*` files, so that branch remains explicitly blocked pending
a fixture or another C-first vehicle (R1/R2/R4/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:3412-3443`, `src/interpreter.c:752`,
  `src/db.h:63-66`, and the C `is_abbrev`/`one_argument` paths before
  changing Go.
- Create a C-first vehicle covering all four selectors, abbreviated and
  case-insensitive forms, invalid input, trailing-word parsing, and the
  missing-file response. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes; distinguish
  selector/error/path drift from the unreachable successful pager branch.
- Fix only confirmed divergences, preserve C's first-token parsing and exact
  player-facing bytes, and record the unavailable successful read/pager path
  as blocked rather than fabricating files (R1/R2/R4/R5e).
- Run `make fidelity-depth`, the focused oracle matrix, all repository build,
  vet, test, lint, formatting, and security gates before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_sysfile` calls `one_argument(argument, arg)`, selects the first matching
file by case-insensitive prefix in the order `bugs`, `ideas`, `todo`,
`typos`, and otherwise sends `That isn't a file!\r\n`. It reads `misc/bugs`,
`misc/ideas`, `misc/todo`, or `misc/typos` with `file_to_string_alloc`. A
negative read result sends `File does not exist.\r\n`; a successful non-null
buffer enters `page_string(ch->desc, readfile, 1)`.

## Pre-fix result

To be filled after the C-first vehicle is run on the fresh implementation at
`DP_SEED=1` and `DP_SEED=2`, before code changes.

## Implementation and proof

To be filled after the confirmed selector, path, error-byte, and pager-call
changes are implemented and the focused vehicle is green at both seeds.

## Verification

To be filled after the full fidelity, build, vet, test, lint, formatting, and
security gates complete.

