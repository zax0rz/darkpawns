# Depth handoff — dark command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:408`, `dark` / `do_dark`  
Starting main: `9193da31b`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `daydream` slice, the interpreter table's next
un-manifested command family in source order was `dark`.

## C path and proof

The command table registers `dark` at `src/interpreter.c:408` with
`POS_DEAD`, `LVL_IMMORT`, and dispatch to `do_dark` in
`src/act.wizard.c:3214-3236`. The handler rejects NPC callers, walks every
character in the room, and for each fighter calls `stop_fighting`; fighting
mobs also have their memory cleared. Each fighter receives
`The peace of the ancients fills your soul.\n\r`. It then sends the actor
`You stop the senseless violence in the room with a wave of your hand.\r\n`
and uses `act` for the exact room audience line
`$n stops the senseless violence in the room with a wave of $s hand.\r\n`.
It never parses the command argument.

The C-first `dark-depth` vehicle puts a mortal peer into combat with the
spawned trainee, moves the immortal actor into the same room, and probes
`dark nonsense` followed by `dark`. It proves the fighter peace output, actor
and room audience topology, ignored arguments, and the no-fighters second
call. The current Go handler's different room wording and missing per-fighter
peace output are confirmed divergences; the implementation is being aligned
only to those observed C branches. No `src/` or `darkpawns-c-oracle/` file was
edited.

## Evidence and gates

Added `docs/fidelity/depth/dark.tsv`, the oracle scenario, and this dated
handoff. The implementation change preserves the existing immortal registry
gate and command surface while correcting the confirmed C output/state path.

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
dispatch and room-fighter call path.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `dark`.
