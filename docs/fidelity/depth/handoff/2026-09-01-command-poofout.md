# Depth-fidelity handoff — `poofout`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `poofin` to
`poofout`.

The next unclaimed interpreter row is `pose` at `src/interpreter.c:615`.

Frontier before this slice: 2,842 total; 2,771 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,847 total; 2,776 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:614 */
{ "poofout"  , POS_DEAD    , do_poofset  , LVL_IMMORT, SCMD_POOFOUT },
```

`src/act.wizard.c:1711-1737` selects `POOFOUT(ch)` for `SCMD_POOFOUT`, calls
`skip_spaces(&argument)`, frees the prior value, stores `NULL` when the
remaining argument is empty, otherwise duplicates the complete remaining
string verbatim, and always sends the global `OK` acknowledgement. The
dispatcher therefore admits every position at `LVL_IMMORT` and rejects a
mortal before the handler. The configured room-audience consumer is shared
with the home poof-out path and remains delegated to `home.poof-audiences`.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/poofout-depth.txt`

Manifest: `docs/fidelity/depth/poofout.tsv` (5 rows)

Test: `pkg/session/poofout_depth_test.go`

The focused baseline state test was intentionally run against untouched main
and was RED: C preserved `a  swirl of departure` while Go stored
`a swirl of departure`. The final isolated scenario was GREEN at seed `1`, and
the matrix at seeds `1,2,3,5,8` was GREEN with no normalized divergence.

The focused test also proves that the bare command clears the prior poof-out
message and that both setter paths emit the exact `Okay.\r\n` acknowledgement.
The confirmed divergence was fixed in the Go command funnel by passing the
transport-preserved raw argument remainder to the shared poof setter, with
`poofout` selecting the `out` field. No C source or oracle checkout was
edited.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-poofout`

Feature commit: `a144f24e1` (`fix: align poofout depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1080 — merged; main commit `8601668d9`. Hosted lint, security,
and test checks were green after the one permitted exact workflow retry;
build-and-push and deploy were skipped by repository workflow conditions. The
PR was merged only after the required hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
