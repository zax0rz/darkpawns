# Depth handoff — date command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:406`, `date` / `do_date`  
Starting main: `70c0449c8`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `dance` slice, the interpreter table's next
un-manifested command family in source order was `date`.

## C path and proof

The command table registers `date` at `src/interpreter.c:406` with
`POS_DEAD`, `LVL_IMMORT`, and `SCMD_DATE`, dispatching to `do_date` in
`src/act.wizard.c:1802-1830`. The handler selects `time(0)` for `SCMD_DATE`,
formats local time with `asctime(localtime(...))`, and sends
`Current machine time: <timestamp>`. It does not parse `argument`; the
separate `uptime` row at `src/interpreter.c:795` owns `SCMD_UPTIME` and the
`Up since ...` branch.

The C-first `date-depth` vehicle proves the immortal machine-time command and
the argument-bearing invocation. Its timestamp is volatile and therefore
normalized by the Tier-1 harness. The focused `TestCmdDateIgnoresArguments`
proof catches the confirmed Go divergence: the old Go handler invented
`date boot` as an uptime alias. The Go fix now always emits the SCMD_DATE
machine-time line, leaving uptime to its own command. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Evidence and gates

Added `docs/fidelity/depth/date.tsv`, the `date-depth` oracle scenario, the
focused regression test, and this dated handoff. The existing
`date.mortal-gate` row in `info.tsv` continues to own the mortal registry gate;
this slice adds the immortal handler branches without duplicating it.

The local gates for this slice are recorded after running:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

This slice follows R1/R2/R3/R4 and R5/R5e: exact player bytes, command
surface, deterministic proof boundaries, no invention, and verification of
the actual C dispatch and subcommand path.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `date`.
