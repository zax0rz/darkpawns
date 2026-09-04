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

