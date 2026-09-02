# Depth-fidelity handoff — `scoff`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `salute` handoff. The special-procedure inventory remains
exhausted. The single blocked row `objmagic.sleep-entry-gates` remains
blocked after its one cast-sleep vehicle using the outlaw/reagent arms; the
reachable portion is covered by the existing sleep-spell evidence and the
remaining entry-gate surface is still recorded as blocked. The interpreter
sweep advanced from `salute` to `scoff`; `score` is already owned by the
existing `info` family. The next un-manifested family is the next command
after `scoff` in `src/interpreter.c` table order.

Frontier before this slice: 3,094 total; 3,016 proven/delegated; 26 blocked;
52 excluded.

Frontier after this slice: 3,101 total; 3,023 proven/delegated; 26 blocked;
52 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:675 */
{ "scoff"    , POS_RESTING , do_action   , 0, 0 },
```

The shared handler is `src/act.social.c:102-151`. It resolves the `scoff`
record, checks `PLR_NOSHOUT`, and, because the record has no `char_found`
message, bypasses target parsing and emits only the no-argument actor/room
pair. The record in `lib/misc/socials:1451-1453` is `scoff 0 0` with exact
messages `You scoff at the idea.` and `$n scoffs at the idea.`; its `#`
terminator makes typed, missing, and self-looking targets repeat the same
no-argument path. The POS_RESTING command gate and shared `PLR_NOSHOUT`
refusal are covered by the command gate and do_action class evidence.

The clean-main RED vehicle found no confirmed Go divergence, so no behavior
change was warranted under R1/R4. No `src/` or oracle-tree file was edited.

## Evidence and confirmed parity

Scenario: `cmd/dp-oracle-diff/scenarios/scoff-depth.txt`

Manifest: `docs/fidelity/depth/scoff.tsv` (7 rows)

Focused test: `pkg/session/scoff_depth_test.go`

The oracle vehicle uses a room peer and covers no argument, typed-target
ignored, missing-target ignored, and self-target ignored. It reported no
normalized divergence at seeds 1, 2, 3, 5, and 8; seed 1 was run with
`--show-oracle` and showed the intended C actor/room blocks. The focused test
pins the POS_RESTING entry gate and all three parsed social message slots.

## Verification and integration

All required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-scoff`

Feature commit: `9bc0ffafa` (`test: prove scoff depth fidelity`)

Feature PR: #1139 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. The PR was self-merged only
after all required checks were green, as main commit `cea4a0c25`.

Open no-check PRs remain unmerged: plot #1064, purge #1095, qecho feature
#1096, qecho handoff #1097, and roll handoff #1130.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). Shared do_action gates and social
record behavior were handled as class-level evidence under R5b/R5c.
