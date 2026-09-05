# Depth handoff — dismount command

Date: 2026-08-30
Queue slice: `src/interpreter.c:416`, `dismount` / `do_dismount`
Starting main: `7a95c34e2fd8`

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The command
table sweep advanced from `disembowel` to the next un-manifested family,
`dismount`. No source or C-oracle file was edited.

## C path and proof

The command table registers `dismount` at `src/interpreter.c:416` with
`POS_FIGHTING`, dispatching to `do_dismount` in `src/act.other.c:1597-1615`.
The handler ignores its argument, sends the unmounted refusal when
`IS_MOUNTED(ch)` is false, and otherwise sends `You hop off your mount.` to
the actor first. It then resolves `get_mount(ch)`, sends the room act
`$n dismounts from the back of $N.`, sends the non-player mount notification,
and calls `unmount(ch, get_mount(ch))`. The actual `src/utils.c:401-414`
helper only clears `AFF_MOUNT` on the rider and mount; it does not clear the
ordinary follower/master relation.

The live `dismount-depth` vehicle was RED on clean main at seed 1 through the
focused Go test: Go delivered the room act before the actor's hop-off line,
reversing the C call order. After the repair, the two-client oracle vehicle is
GREEN at seeds 1, 2, 3, 5, and 8. It proves the refusal, mounted actor line,
ignored argument, and peer room audience; the focused test additionally proves
the modeled pair cleanup and preserves an unrelated follower relation.

The Go path now sends the actor line before `Act(ToRoom)`, clears the rider
and mount mounted state, clears the modeled mount rider/name, and leaves the
ordinary follower relation untouched. This follows R1/R2/R3/R4 and R5/R5e:
exact player bytes, command-surface fidelity, deterministic ordering/state,
no invented behavior, and verification of the actual C dispatch and helper
call path.

## Evidence and gates

Added or changed:

- `cmd/dp-oracle-diff/scenarios/dismount-depth.txt`
- `docs/fidelity/depth/dismount.tsv`
- `pkg/game/dismount_depth_test.go`
- `pkg/game/other_mount.go`

Focused RED and GREEN:

```
go test ./pkg/game -run '^TestDismountDepthMatchesCOrderAndPairCleanup$' -count=1
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-depth -scenario dismount-depth -seed 1
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-depth -scenario dismount-depth -seed 2
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-depth -scenario dismount-depth -seed 3
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-depth -scenario dismount-depth -seed 5
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /tmp/dp-oracle-diff-depth -scenario dismount-depth -seed 8
```

The full repository gates passed after the manifest addition:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

The frontier after this manifest addition is recorded by the final gate
output and must be refreshed on clean `main` before the next slice.

## Next queue item

After this slice's PR is handled, return to clean `main`, pull, refresh the
frontier, reread the testing guide and newest handoff, and take the next
un-manifested command-table family after `dismount`.
