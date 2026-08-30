# Depth handoff — dance command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:405`, `dance` / `do_action`  
Starting main: `cd19129d6`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `cutthroat` slice, the interpreter table's next
un-manifested command family in source order was `dance`.

## C path and proof

The command table registers `dance` at `src/interpreter.c:405` for
`POS_STANDING` and dispatches to `do_action` in `src/act.social.c:102-151`.
Its C social record at `lib/misc/socials:176-184` has `hide=1`, minimum victim
position `POS_STANDING`, and these exact message slots: no-arg actor/room,
target actor/room/victim, missing-target `Eh, WHO?`, and self-target
`You skip and dance around by yourself.` / `$n skips a light Fandango.`.

The actual call path checks `PLR_NOSHOUT`, parses only when `char_found` is
present, resolves a visible room target, then selects the proper-position,
self, or actor/room/victim branch. The existing Go `DoAction` implementation
already follows this shared C social path; no Go divergence was confirmed, so
the slice adds only proof vehicles, manifest evidence, and this handoff.

The C-first `dance-depth` vehicle proves no argument, one-argument target
selection, missing target, self target, standing-peer audience topology, and
the sleeping-target position gate. The isolated `dance-noshout` vehicle proves
the pre-lookup emote refusal. Both vehicles are GREEN at seeds 1, 2, 3, 5, and
8, and `--show-oracle` was used during development. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Evidence and gates

Added `docs/fidelity/depth/dance.tsv`, two oracle scenarios, and this dated
handoff. The manifest records seven cases for the command's reachable gates,
audiences, and direct outcomes. Shared social table metadata and visibility
behavior remain covered by the existing `socials` depth manifest under R5b/R5c.

Final local gates on the committed slice:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

This slice follows R1/R2/R3/R4 and R5/R5e: exact player bytes, command
surface, deterministic oracle coverage, no invention, and verification of the
actual C dispatch and social record path.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `dance`.
