# Depth handoff — daydream command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:407`, `daydream` / `do_action`  
Starting main: `86eded2b7`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `date` slice, the interpreter table's next
un-manifested command family in source order was `daydream`.

## C path and proof

The command table registers `daydream` at `src/interpreter.c:407` for
`POS_SLEEPING` and dispatches to `do_action` in `src/act.social.c:102-151`.
Its social record at `lib/misc/socials:186-189` is `daydream 1 0` with only
the no-argument actor/room messages:
`You dream of better times.` and
`$n looks absent-minded, $s eyes staring into space.`. The `char_found` slot
is `#`, so the actual C path never parses or looks up an argument; every
typed target follows the same actor/room branch, with `hide=1`.

The C-first `daydream-depth` vehicle proves no argument, visible target,
missing target, and self-named target. All four probes are GREEN at seed 1;
the social has no RNG. The existing Go shared `DoAction` path matched the C
behavior, so no Go divergence was confirmed and no implementation change was
needed. No `src/` or `darkpawns-c-oracle/` file was edited.

## Evidence and gates

Added `docs/fidelity/depth/daydream.tsv`, the oracle scenario, and this dated
handoff. Shared social dispatch, visibility, and message-table behavior remain
covered by the existing `socials` depth manifest under R5b/R5c.

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
surface, deterministic proof, no invention, and verification of the actual C
dispatch and social record path.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `daydream`.
