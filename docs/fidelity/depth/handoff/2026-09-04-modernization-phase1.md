# 2026-09-04 — modernization Phase 1 audit

## Phase 1.1

The retired generated social metadata artifacts were removed in a separate
target commit after updating the reachability audit scripts to read the
authoritative `lib/misc/socials` records. `make fidelity-depth` remains at the
correct 4,758 modeled cases: 4,653 proven/delegated, 54 blocked, and 51
excluded. This deletion has no runtime or player-byte effect.

## Phase 1.2 disposition

The roadmap's 628-line “commented-out code” number is a lexical heuristic,
not a deletion-ready inventory. The largest apparent cluster in
`pkg/scripting/engine.go` consists of API labels and C-call-path
documentation immediately above live Lua handlers; other matches are C
formula transcriptions and test/golden commentary. The repository has no
remaining `pkg/engine/example_integration.go` file from the older census.

Deleting those comments wholesale would erase fidelity evidence and source
mapping, violating R4's no-invention rule without removing dead executable
code. Phase 1.2 therefore ends as a verified stale-target correction; no
player-facing or executable behavior is changed.

## Phase 1.3

The three named zero-reference functions will be checked by exact symbol
search, including registration and tests, before any deletion. A suspect
function remains until its call path is disproven under R5; confirmed dead
functions may be removed in a separate target commit with the full repository
gates.

## Denominators

The modeled depth denominator is 4,758 cases (4,653 proven/delegated, 54
blocked, 51 excluded). The separate surface-inventory denominator is 70 rows
and 4,926 weighted units; its blocked rows are handled by the Lane B clinic
and are not silently collapsed into the modeled case count.
