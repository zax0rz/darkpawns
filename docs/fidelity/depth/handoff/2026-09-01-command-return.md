# Depth-fidelity handoff — `return`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `reroll` handoff. The special-procedure inventory remains exhausted, the
one blocked row `objmagic.sleep-entry-gates` remains queued after its single
cast-sleep vehicle, and the interpreter sweep advanced from `restore` to
`return`.

The source-order audit did not repick `retreat` (owned by `escape`),
`retrieve` (owned by `spec-procs`), or `ride`/`roomflags` (owned by `mount` and
`gen-tog`). The next unclaimed interpreter-table family is `redit` at
`src/interpreter.c:656`, sharing C's `do_olc` handler with the already-blocked
`medit` family.

Frontier before this slice: 2,977 total; 2,904 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,982 total; 2,909 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:655 */
{ "return"   , POS_DEAD    , do_return   , 0, 0 },
```

`src/act.wizard.c:1205-1221` has no handler-level authorization or argument
gate. It checks only `ch->desc && ch->desc->original`; an ordinary player
descriptor therefore reaches a silent return, and all arguments are ignored.
For a switched descriptor, C sends exactly `You return to your original
body.\r\n` to the current body, closes a descriptor already attached to the
original body if present, reattaches the descriptor to `original`, clears
`descriptor->original`, and leaves the switched body's descriptor null.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/return-depth.txt`

Manifest: `docs/fidelity/depth/return.tsv` (five rows)

Focused tests: `pkg/session/return_depth_test.go`

The clean-main scenario was RED: both the no-argument and trailing-input
probes emitted Go's invented `You aren't switched.` while C emitted no bytes.
The corrected Go path removes the invented level gate, silent-branch response,
and persistence writes (C has no save path), and matches C's switched-body
message and state clearing. No `src/` or C-oracle file was edited.

The corrected scenario was GREEN with `--show-oracle` and across seeds 1, 2, 3,
5, and 8. The switched-body output/state branch is pinned by focused unit
proof because the Go session's switched-body bookkeeping is an in-process
state seam rather than a descriptor-issued oracle fixture. The manifest keeps
the broader `switch` call path separate under R5b/R5c.

## Verification and integration

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-return`

Feature commit: `50d1a5b29` (`fix: match return depth fidelity (R1/R2/R4/R5e)`)

Feature PR: #1113 — hosted lint, security, and test checks were green;
build-and-push and deploy were skipped by workflow conditions. It was
self-merged as main commit `b72afdeb5` only after the required hosted checks
were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism across the oracle matrix), R4 (no invention), R5 (process
discipline), and R5e (verify the actual C call path). The separate switching
boundary and source-order claim are maintained under R5b/R5c.
