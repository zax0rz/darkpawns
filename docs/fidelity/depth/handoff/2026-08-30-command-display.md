# Depth handoff — display command

Date: 2026-08-30
Queue slice: `src/interpreter.c:417`, `display` / `do_display`
Starting main: `8ac86f4bc91e`

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The command
table sweep advanced from `dismount` to the next un-manifested family,
`display`. No source or C-oracle file was edited, and no Go behavior required
changing because the clean-main implementation already matched this path.

## C path and proof

The command table registers `display` at `src/interpreter.c:417` with
`POS_DEAD`, dispatching to `do_display` in `src/act.other.c:1024-1082`.
The player command is non-NPC-only, skips leading spaces, emits the exact
usage line for an empty argument, treats `on` and `all` as the five-flag
enable branch, clears all five flags before the general letter loop, returns
silently for `off`, and otherwise enables the recognized h/m/v/t/f letters
before sending `Okay.`. The later C `prompt` table entry is a separate queue
family: it shares the C handler, but the Go surface currently routes through a
custom prompt-string command and is explicitly not claimed by this `display`
manifest.

The existing `display-cluster` vehicle was run on clean main at seed 1 with
`--show-oracle`; it was GREEN and exposed the intended C blocks for no
argument, all, on, off, hmv, and none. The vehicle is deterministic because
the handler performs no RNG draws. Existing focused `TestDoDisplay` proves the
hidden five-bit state for all, off, and the letter mask. No divergence was
confirmed, so no Go behavior was invented or altered.

This follows R1/R2/R3/R4 and R5/R5e: exact player bytes, command-surface and
position fidelity, deterministic state parity, no invented behavior, and
verification of the actual C dispatch and branch path.

## Evidence and gates

Added or changed:

- `cmd/dp-oracle-diff/scenarios/display-cluster.txt`
- `docs/fidelity/depth/display.tsv`
- `docs/fidelity/depth/handoff/2026-08-30-command-display.md`

Focused proof:

```
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff -scenario display-cluster -seed 1 -show-oracle
go test ./pkg/game -run '^TestDoDisplay$' -count=1
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
un-manifested command-table family after `display`.
