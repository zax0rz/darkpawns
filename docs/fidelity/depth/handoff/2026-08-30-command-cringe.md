# Depth handoff — cringe command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:397`, `cringe` / `do_action`  
Starting main: `6f73f913e`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including the active
registration tables, remains exhausted. The blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked; it was not repicked. After the
merged `cough` slice, the interpreter sweep selected `cringe` at line 397 in
table order. The preceding `compact` alias is owned by `gen-tog.tsv`, and
`cough` was just proven in its own source-order social slice.

## C path and proof

The command table registers `cringe` at `src/interpreter.c:397` for
`POS_RESTING` and routes it to `do_action` in `src/act.social.c:102-151`. The
record at `lib/misc/socials:136-144` has `hide=1`, minimum victim position
zero, and a complete actor/room/victim/self/missing-target matrix. The C
handler checks `PLR_NOSHOUT`, parses
one target, emits the no-argument pair without a target, then uses visible
target lookup, self-target, successful trio, and missing-target branches in
that order. A sleeping target is accepted because the record's minimum
position is zero; `hide=1` controls invisible-actor room filtering.

The C-first `cringe-depth --seed 1 --show-oracle` vehicle uses a primary actor,
a standing peer, and a sleeping peer. It proves no argument, successful peer,
missing target, self target, and successful sleeping target. All five blocks
are GREEN with exact actor, room, and victim audiences. No Go
divergence was confirmed, so no implementation change was made; neither
`src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/cringe-depth.txt` with the complete target
  and audience matrix.
- Add the five rows in `docs/fidelity/depth/cringe.tsv`.
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

The refreshed frontier is 1,471 total cases: 1,417 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,417/1,431 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the social dispatch, visible target
resolution, hide policy, and actor/room/victim audience topology follow
R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `cringe`.
