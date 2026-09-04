# Wizard `syslog` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:3079-3105`, covering `do_syslog` and its command-table entry
at `src/interpreter.c:753`. The existing manifest proves only the default
no-argument status through `wizard-residual-depth`; this audit expands C's
selector, state, and refusal branches without claiming the separate logging
consumer matrix.

The C handler has three observable families: no-argument status derived from
the two preference bits, a `search_block(..., FALSE)` selector for Off, Brief,
Normal, and Complete, and an invalid-selector usage response. C's non-exact
search accepts case-insensitive prefixes, then clears both log bits before
setting the selected bit combination. The player-facing state is the two-bit
contract; downstream filtering of internal log records belongs to the shared
logging call sites and must be delegated or blocked at that boundary
(R1/R2/R3/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:3076-3105`, `src/interpreter.c:753`, and
  `src/interpreter.c:1150-1178` before changing Go.
- Create a C-first vehicle covering off/brief/normal/complete status and
  mutation, case-insensitive and abbreviated selectors, invalid input, and
  trailing-word parsing. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes; distinguish
  selector/state drift from the shared log-consumer behavior.
- Fix only confirmed divergences, preserve unrelated preference bits, and
  record any internal logging consumer gap as delegated or blocked.
- Run `make fidelity-depth`, the focused oracle matrix, all repository build,
  vet, test, lint, formatting, and security gates before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_syslog` calls `one_argument`, reports the two-bit `PRF_LOG1`/`PRF_LOG2`
state as `off`, `brief`, `normal`, or `complete`, then uses case-insensitive
prefix `search_block` matching. Unknown selectors receive
`Usage: syslog { Off | Brief | Normal | Complete }\r\n`. Valid mutations clear
both bits, set bit 0 for Brief, bit 1 for Normal, both for Complete, and report
`Your syslog is now <level>.\r\n`.

## Pre-fix result — 2026-09-04

The C-first vehicle `cmd/dp-oracle-diff/scenarios/wizard-syslog-depth.txt`
was red on the fresh `origin/main`-derived port at `DP_SEED=1` and
`DP_SEED=2`. The no-argument path, full-word
mutations, invalid usage, and trailing-word handling matched, but C's
case-insensitive prefix selectors `b`, `c`, and `o` were rejected by Go. After
the vehicle set the Go player to Brief with the full word and then selected
Normal, Go retained both log bits and reported Complete while C reported
Normal. These are confirmed selector/state divergences in the handler, not
evidence about the downstream internal log consumer (R1/R2/R3/R5e).

## Implementation and proof — 2026-09-04

The handler now mirrors C's case-insensitive prefix `search_block` behavior,
consumes only the first `one_argument`, clears both log preference bits before
applying the selected Off/Brief/Normal/Complete state, and preserves the exact
usage/status/mutation bytes. Focused unit tests pin the prefix parser and the
Brief-to-Normal bit transition.

The C-first vehicle is green with `--show-oracle` at both `DP_SEED=1` and
`DP_SEED=2`:

```text
wizard-syslog-depth: result: no normalized divergence
```

It covers all four states, abbreviated and case-insensitive selectors, state
queries, invalid input, and trailing-word parsing. The manifest records the
handler cases as `oracle-green-multiseed`; the downstream `mudlog` audience and
log-type consumer matrix remains explicitly blocked as a shared surface.

Verification completed on this slice:

- `make fidelity-depth`: 4264 total, 4160 proven/delegated, 53 blocked, 51 excluded.
- `go build ./...`: pass.
- `go vet ./...`: pass.
- `go test ./...`: pass.
- `golangci-lint run ./...`: pass with 0 issues.
- `gofumpt -l .`: clean.
- high-severity `gosec`: pass.

No C or oracle-tree files were changed. The untracked
`website/static/images/` directory remains outside this slice.
