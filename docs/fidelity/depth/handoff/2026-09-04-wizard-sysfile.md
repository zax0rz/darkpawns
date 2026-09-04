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

## Pre-fix result — 2026-09-04

The C-first vehicle was red on the fresh `origin/main`-derived port at
`DP_SEED=1` and `DP_SEED=2`. The empty argument, full-word `BUGS`, invalid
selector, and trailing-word cases normalized green, while the C prefix forms
`b`, `i`, `t`, and `ty` reached the missing-file branch and the Go handler
rejected them as non-files:

```text
sysfile b   C: File does not exist.   Go: That isn't a file!
sysfile i   C: File does not exist.   Go: That isn't a file!
sysfile t   C: File does not exist.   Go: That isn't a file!
sysfile ty  C: File does not exist.   Go: That isn't a file!
```

The C and Go trees contain no `misc/bugs`, `misc/ideas`, `misc/todo`, or
`misc/typos` file, so the successful read/pager path was not reachable by this
vehicle. The confirmed pre-fix differences are selector matching and the
associated C missing-file branch; no pager behavior is inferred from the
absence of a fixture (R1/R2/R4/R5e).

## Implementation and proof — 2026-09-04

`cmdSysfile` now consumes the first argument with the shared C-compatible
`game.OneArgument` path, matches the four selectors in C's ordered
case-insensitive prefix order, derives `misc/*` from the configured lib
directory, emits the exact CRLF refusal bytes, and sends successful reads
through `PageString`. Its defense-in-depth handler gate now matches the
command-table `LVL_GOD` requirement. The helper unit test pins empty,
abbreviated, ordered, case-insensitive, and unknown selector behavior.

The C-first vehicle is green with `--show-oracle` at both `DP_SEED=1` and
`DP_SEED=2`:

```text
wizard-sysfile-depth: result: no normalized divergence
```

All reachable selector, missing-file, invalid-input, case-folding, and
trailing-word cases are proven. The successful read/pager branch remains
blocked because the authoritative C path files are absent from both checked-
in trees; adding a synthetic static file would violate R4 and would not prove
the live C call path (R1/R2/R4/R5b/R5c/R5e).

## Verification

Completed on 2026-09-04:

- `make fidelity-depth`: 4,272 total, 4,167 proven/delegated, 54 blocked,
  51 excluded.
- `go build ./...`: pass.
- `go vet ./...`: pass.
- `go test ./...`: pass.
- `golangci-lint run ./...`: pass with 0 issues.
- `gofumpt -l .`: clean.
- high-severity/high-confidence `gosec`: pass with 0 issues.

No C or oracle-tree files were changed. The untracked
`website/static/images/` directory remains outside this slice.
