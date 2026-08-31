# Depth-fidelity handoff: `house`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `hop` handoff. The pre-slice
frontier was 2,191 total cases: 2,131 proven/delegated, 16 blocked, and 44
excluded. The `house` manifest adds 20 cases, producing 2,211 total: 2,151
proven/delegated, 16 blocked, and 44 excluded (2,151 of 2,167 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `house`; the next unclaimed family
is `howl` at `src/interpreter.c:503`.

The slice was PR #950, branch `glm/depth-house`, merged to `main` as
`89f4dba2d`. GitHub initially reported no checks, so the single permitted
retry was run with `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-house`.
The asynchronously materialized test, lint, and security checks then passed;
build and deploy were skipped by repository policy. No non-green PR was
merged.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:502: { "house", POS_RESTING, do_house, 0, 0 }
```

`src/house.c:603-718` is the complete command path. `do_house` first gates
on `ROOM_HOUSE`, resolves the controlling house record, and requires the
primary owner (or an immortal). It parses the first two command words using
the C `two_arguments` path, which delegates to `one_argument`, skips fill
words, lowercases tokens, and ignores the remainder.

The audited player-visible branches are:

- room and owner entry gates, including the malformed-control diagnostic;
- no-argument usage text;
- guest list with empty, populated, and missing-player cleanup states;
- guest add, duplicate/delete toggle, full-capacity refusal, and unknown
  player lookup;
- fill-word and trailing-argument boundaries;
- transfer usage, unknown-player refusal, successful transfer, and its exact
  historical spelling `House transfered.`;
- silent unknown subcommands;
- in-memory ownership change without a `House_save_control` call.

R1/R2/R4/R5e apply. There is no RNG draw in this procedure. The room,
position, and owner gates are shared/typed behavior and are represented by
the direct entry-gate and focused cases rather than invented extra branches;
R5b/R5c keep the proof attached to the actual `do_house` call path.

## RED/ GREEN result

The first house vehicle included `empty-players` and failed to reach the
command because the C actor entered its yes/no prompt. That setup was
discarded and is not counted as a proof attempt. The corrected vehicle
retained the seeded C `Testor`, created an online `Houseguest`, installed a
house-control record, and then exercised the command. It initially exposed
confirmed divergences on clean `main`:

- Go held `World.mu` while sending/calling back into player state, deadlocking
  after the first guest-list operation;
- Go used `strings.Fields` rather than C's two-`one_argument` parsing;
- Go invented output for unknown subcommands;
- Go emitted `House transferred.` and persisted transfer state, unlike C's
  `House transfered.` and no-save path;
- no-DB players with ID zero aliased guest lookups to the first player.

The Go fix snapshots guest state under the world lock and performs callbacks,
cleanup persistence, and player output after unlocking; uses the C argument
boundary; preserves the silent unknown path and exact transfer typo; leaves
transfer unsaved; and assigns unique positive runtime IDs to subsequent
zero-ID players while preserving the fixture owner's ID. No file under `src/`
or `darkpawns-c-oracle/` was edited.

## Verification

`house-depth.txt` was GREEN with `--show-oracle` at seed 1 and also at seeds
2, 3, 5, and 8. `house-entry-gates-depth.txt` was GREEN with `--show-oracle`
at seed 1. The focused house tests passed, including malformed controls,
missing-guest cleanup, capacity, transfer save-boundary, parser boundaries,
and runtime-ID lookup.

The local gates all passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. PR #950's hosted test, lint, and security checks were
green before merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only the
unclaimed `howl` family at `src/interpreter.c:503`. Continue the
command-table sweep in source order with one slice/one PR and the non-green-
check safety rule.
