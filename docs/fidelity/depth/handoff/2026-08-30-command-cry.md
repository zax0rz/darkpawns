# Depth handoff — cry command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:398`, `cry` / `do_action`  
Starting main: `a8784916b`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including active registration
tables, remains exhausted. The blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and remains
blocked; it was not repicked. The interpreter sweep selected `cry` at line
398, immediately after the merged `cringe` slice and in command-table order.

## C path and proof

The command table registers `cry` at `src/interpreter.c:398` for
`POS_RESTING` and routes it to `do_action` in `src/act.social.c:102-151`. The
record at `lib/misc/socials:146-155` has `hide=0`, minimum victim position
`POS_STANDING` (value 5), and all eight actor/room/victim/self/missing-target
messages. The C handler checks `PLR_NOSHOUT`, parses one target, emits the
no-argument pair without a target, then uses visible target lookup, self-target,
minimum-position, successful trio, and missing-target branches in that order.

The C-first `cry-depth --seed 1 --show-oracle` vehicle uses a primary actor, a
standing peer, and a sleeping peer. It proves no argument, successful peer,
missing target, self target, and the sleeping-target position gate. All five
blocks are GREEN with exact actor, room, victim, and refusal audiences. No Go
divergence was confirmed, so no implementation change was made; neither
`src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/cry-depth.txt` with the complete target and
  audience matrix.
- Add the five rows in `docs/fidelity/depth/cry.tsv`.
- Add this dated handoff.

## Gates and frontier

The following all pass on the final tree:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The refreshed frontier is 1,476 total cases: 1,422 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,422/1,436 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the social dispatch, visible target
resolution, minimum-position gate, and actor/room/victim audience topology
follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `cry`.
