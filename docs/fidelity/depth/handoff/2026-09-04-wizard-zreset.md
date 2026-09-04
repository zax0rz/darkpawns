# Wizard `zreset` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:2035-2076`, covering `do_zreset` and its command-table entry
at `src/interpreter.c:848`. The existing manifest proves only the empty-
argument response through `wizard-residual-depth`; this audit expands the
reachable selector, reset, output, and parser branches without re-counting the
shared `reset_zone()` implementation owned by the world-reset surface.

The C handler has four observable branch families: missing argument, `*` world
reset, `.` current-zone reset, and a numeric zone-number lookup using C
`atoi`/`one_argument`. It also has invalid-zone output and exact zone index,
number, and name formatting. The reset itself can mutate mobs, objects, doors,
and RNG state, so state/draw parity belongs at the verified `reset_zone()` call
path and must be delegated or separately blocked rather than inferred from the
handler's acknowledgement (R1/R3/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:2035-2076`, `src/interpreter.c:848`, and the
  actual `src/db.c:2074-2195` reset call path before changing Go.
- Create a C-first vehicle that names every selector and parser boundary that
  can be safely observed, including `*`, `.`, numeric prefixes, invalid input,
  and trailing words. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes; distinguish
  acknowledgement-byte drift from shared reset state/RNG drift.
- Fix only confirmed divergences, preserve the shared reset ownership boundary,
  and record any reset-state or logging gap as delegated or blocked with a
  source-backed reason.
- Run `make fidelity-depth`, the focused oracle matrix, all repository build,
  vet, test, lint, formatting, and security gates before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_zreset` calls `one_argument`, refuses an empty first token with
`You must specify a zone.\r\n`, resets every table entry for `*` and reports
`Reset world.\r\n`, maps `.` through the actor's current room zone index, and
compares `atoi`'s result against each zone table number. A valid selected zone
reports `Reset zone %d (#%d): %s.\r\n`, where the first value is the zone-table
index. Any unmatched selector reports `Invalid zone number.\r\n`. The C handler
logs successful resets but does not expose the log to the issuing player.

## Pre-fix result — 2026-09-04

The C-first vehicle `cmd/dp-oracle-diff/scenarios/wizard-zreset-depth.txt`
was red on the fresh `origin/main`-derived port at `DP_SEED=1` and
`DP_SEED=2`. The invalid selector happened to match, but every reachable
successful acknowledgement diverged: Go invented `(async)`, reported zone
number `12` instead of C's zone-table index `3`, rejected C's numeric-prefix
`atoi` behavior for `12abc`, and emitted `Reset world (async).` instead of
`Reset world.`. The `.` branch exposed the same index drift. The red vehicle
also confirmed that these are handler-byte/parser findings; reset-world state
and RNG parity remain a separate shared `reset_zone()` boundary (R1/R2/R3/R5e).

## Implementation and proof — 2026-09-04

The handler now consumes the first C `one_argument`, uses C-compatible
`cAtoi` for numeric-prefix selectors, maps room zone numbers to the sorted
C-equivalent zone-table index, invokes the existing synchronous reset engine,
and preserves C's exact acknowledgement strings and CRLF. World-reset errors
are logged without inventing player-facing error bytes. The focused unit test
pins the sparse zone-number to table-index mapping used by both numeric and
dot selectors.

The C-first vehicle is green with `--show-oracle` at both `DP_SEED=1` and
`DP_SEED=2`:

```text
wizard-zreset-depth: result: no normalized divergence
```

It covers invalid, numeric, numeric-prefix, dot/current-zone, trailing-word,
and star/world selectors. The manifest records the handler branches as
`oracle-green-multiseed`; the full reset-state/RNG matrix remains explicitly
blocked because it is the shared `reset_zone()` surface rather than a proven
property of the acknowledgement handler.

Verification completed on this slice:

- `make fidelity-depth`: 4251 total, 4148 proven/delegated, 52 blocked, 51 excluded.
- `go build ./...`: pass.
- `go vet ./...`: pass.
- `go test ./...`: pass.
- `golangci-lint run ./...`: pass with 0 issues.
- `gofumpt -l .`: clean.
- high-severity `gosec`: pass.

No C or oracle-tree files were changed. The untracked
`website/static/images/` directory remains outside this slice.
