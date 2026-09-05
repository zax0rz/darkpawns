# Depth handoff — dig command

Date: 2026-08-30
Queue slice: `src/interpreter.c:413`, `dig` / `do_dig`
Starting main: `67e83cb86`
Working branch: `glm/depth-dig`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After `dc`, the interpreter table's next un-manifested command
family was `dig`.

## C path and proof

The command table registers `dig` at `src/interpreter.c:413` with
`POS_RESTING` and `LVL_BUILDER` (31), dispatching to `do_dig` in
`src/new_cmds2.c:818-881`. The handler uses `two_arguments`, C `atoi` and
`real_room`, checks missing and unknown rooms, applies the saved
`GET_OLC_ZONE`/target-zone builder gate, accepts both cases of the six
direction initials, emits the invalid-direction warning while continuing with
the default north/south pair, creates bare reciprocal exits, and confirms with
the original direction argument.

The pre-fix `dig-red-depth` run on main reached `Huh?!?` for every probe while
the oracle reached the C format, invalid-direction-plus-success,
missing-room, success, and trailing-argument branches. The fixed branch
proved `dig-depth`, `dig-permission-depth`, and `dig-current-zone-depth` with
no normalized divergence (seed 1, with `--show-oracle`). The permission
vehicle covered same-zone builder success, target-zone refusal, and current
zone refusal. No `src/` or `darkpawns-c-oracle/` file was edited.

The Go changes register the C OLC command, preserve the unrelated foraging
skill only at its game-layer API, add the saved OLC zone field and fixture
setter, create bare reciprocal exits, and match the C parser and bytes.

## Gate blocker

The ordinary Go gates passed:

```
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

`make fidelity-depth` did not complete because the existing manifest row
`at.nested-movement-not-restored` reported a missing unit-test symbol
`TestCmdAtPreservesNestedMovementLocation` in `docs/fidelity/depth/at.tsv`.
The apparent blocker was caused by this slice's first focused-test addition
accidentally replacing the existing `pkg/session/wiz_movement_test.go`; it was
not a main failure. No `dig` PR was opened or merge attempted in that pass.
The existing `at` test was restored, and the branch retained the proven `dig`
implementation, scenarios, manifest rows, and focused helper tests for the
successful resumed pass.

Before the validator stopped, the expected frontier after eight `dig` rows
was 1543 total, 1489 proven/delegated, 14 blocked, and 40 excluded (99.1%
actionable). The validator failure means this snapshot was not accepted as a
fresh report; the resumed handoff records the accepted frontier.

This handoff follows R1/R2/R3/R4 and R5/R5e: exact player bytes, command
surface, deterministic proof, no invention, and verification of the actual C
dispatch and call path. R5b/R5c required the test-file replacement to be
corrected before claiming the queue could advance.

## Next action

Use the resumed `dig` handoff as the accepted record: the existing `at` proof
symbol is restored and all local gates pass. The next queue item is the
un-manifested `disarm` command family.
