# Depth-fidelity handoff — `poofin`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `ponder` to
`poofin`.

The next unclaimed interpreter row is `poofout` at `src/interpreter.c:614`.

Frontier before this slice: 2,836 total; 2,765 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,842 total; 2,771 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:613 */
{ "poofin"   , POS_DEAD    , do_poofset  , LVL_IMMORT, SCMD_POOFIN },
```

`src/act.wizard.c:1711-1737` selects `POOFIN(ch)` for `SCMD_POOFIN`, calls
`skip_spaces(&argument)`, frees the prior value, stores `NULL` when the
remaining argument is empty, otherwise duplicates the complete remaining
string verbatim, and always sends the global `OK` acknowledgement. The
dispatcher therefore admits every position at `LVL_IMMORT` and rejects a
mortal before the handler. The stored value is later consumed by the shared
wizard movement paths; that destination-room audience is delegated to the
existing `goto` depth matrix.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/poofin-depth.txt`

Manifest: `docs/fidelity/depth/poofin.tsv` (6 rows)

Test: `pkg/session/poofin_depth_test.go`

The first combined setter-plus-goto vehicle was intentionally used as a RED
probe. It confirmed that C preserved an internal double space in the stored
message while the Go token join collapsed it. That vehicle also exposed an
unrelated leading-space difference in the already-owned goto room renderer;
the goto command was removed from this setter scenario rather than fixed
forward outside the current slice. The final seed-1 `--show-oracle` vehicle was
GREEN for both exact `Okay.\r\n` acknowledgements and the bare-command clear.

The focused test proves the hidden state transition: an empty remainder clears
the old poof-in string, while a raw remainder such as `a  swirl of arrival`
retains its internal spacing. The matrix at seeds `1,2,3,5,8` was GREEN with
no normalized divergence.

The confirmed divergence was fixed in the Go command funnel by passing the
transport-preserved raw argument remainder to the poof-in setter. No C source
or oracle checkout was edited. The shared setter helper is ready for the next
`poofout` slice, whose command row remains unclaimed.

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

Feature branch: `glm/depth-poofin`

Feature commit: `6a0aa07e6` (`fix: align poofin depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1078 — merged; main commit `01728fb64`. Hosted lint, security,
and test checks were green; build-and-push and deploy were skipped by repository
workflow conditions. The PR was merged only after the required hosted checks
were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
