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

## Pre-fix result — 2026-09-04

The C-first vehicle `cmd/dp-oracle-diff/scenarios/wizard-wiznet-depth.txt`
was red on this fresh `origin/main`-derived port at `DP_SEED=1` (the same
branch family is deterministic at seed 2). The existing Go handler matched
usage, ordinary/emote entry, invalid and empty-prefix refusals, but diverged
on the player-visible branches that the vehicle made reachable:

- C emits `<32>` for a numeric directed wizline; Go omitted the level tag.
- C removes the escape backslash before `@`; Go broadcast the backslash.
- C preserves the sender's name for visible recipients; Go always used
  `Someone` for peers.
- C's `@` branch listed no gods in the chosen/mortal setup; Go invented an
  online list from every session.

The red vehicle therefore established confirmed parser, recipient, and
listing gaps under R1/R2/R3/R5e. A separate qualifying `@` vehicle was also
run, with the same empty C block; that branch is retained as explicitly
blocked in the manifest pending a stable vehicle for the source's online/
offline scratch-buffer path.

## Implementation and proof — 2026-09-04

The implementation preserves C's raw argument remainder, `one_argument` /
`is_number` / `half_chop` prefix sequence, exact level-tagged message bodies,
backslash escaping, chosen-recipient rule, `PRF_NOWIZ` sender refusal,
recipient filters, visibility shadowing, `PRF_NOREPEAT` acknowledgement, and
deterministic descriptor-order snapshots. The focused sender-entry unit test
proves that an unchosen mortal is refused while a `PLR_CHOSEN` mortal reaches
the C handler despite the command-table level of zero.

The C-first vehicles are green with `--show-oracle` at both `DP_SEED=1` and
`DP_SEED=2`:

```text
wizard-wiznet-depth: result: no normalized divergence
wizard-wiznet-gates-depth: result: no normalized divergence
```

The main vehicle covers usage, ordinary and emote broadcasts, numeric and
nonnumeric prefixes, above-owner and empty payload refusals, escaped `@`,
chosen-recipient filtering, and no-repeat state. The gate vehicle covers the
post-prefix `PRF_NOWIZ` refusal. The qualifying `@` branch remains blocked as
recorded above; no green claim is made for its unstable C transcript.

Verification completed on this slice:

- `make fidelity-depth`: 4244 total, 4142 proven/delegated, 51 blocked, 51 excluded.
- `go build ./...` and `go vet ./...`: pass.
- `go test ./...`: pass.
- `golangci-lint run ./...`: pass with 0 issues.
- `gofumpt -l .`: clean.
- high-severity `gosec`: pass.

No C or oracle-tree files were changed. The untracked
`website/static/images/` directory remains outside this slice.
