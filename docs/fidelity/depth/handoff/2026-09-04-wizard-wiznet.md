# Wizard `wiznet` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:1912-2034`, covering `do_wiznet` and its command-table alias
`;` at `src/interpreter.c:830-831`. The existing manifest proves only the
empty-argument usage branch; this audit expands the reachable direct branches,
recipient filtering, and exact command bytes without re-counting the already
owned communication helpers.

The C surface includes the chosen-or-immortal entry gate, empty-argument usage,
`*` emotes, `#<level>` directed wizlines, `@` online/offline god listing,
backslash escaping, ordinary broadcasts, `PRF_NOWIZ` refusal, empty payload
refusal after a prefix, recipient visibility and level filters, writing/mail
filters, and `PRF_NOREPEAT` self-output. C `skip_spaces`,
`delete_doubledollar`, `one_argument`, `half_chop`, and `is_number` behavior
are parser and byte authority (R1/R2/R5e). Room/player audience ordering and
the no-repeat state are shared communication behavior and must be delegated or
blocked at the verified call-path boundary (R3/R5b/R5c).

## Required evidence

- Read and cite `src/act.wizard.c:1912-2034` and `src/interpreter.c:830-831`
  before changing Go; the current handler is not authority.
- Add a C-first vehicle with named passive peers and, where reachable, a
  linkless peer to exercise audience filtering. Annotate every depth case.
- Run the pre-fix vehicle on this fresh `origin/main`-derived branch at seeds
  1 and 2 with `--show-oracle`, recording exact red branches before any fix.
- Fix only confirmed divergences, preserving existing communication ownership
  boundaries and recording exclusions/blocks rather than inventing coverage.
- Run `make fidelity-depth`, the focused vehicle at both seeds, the full
  build/vet/test/lint/security gate, and `gofumpt -l .` before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## Pre-fix result

To be filled after the vehicle establishes the fresh-main RED boundary.
