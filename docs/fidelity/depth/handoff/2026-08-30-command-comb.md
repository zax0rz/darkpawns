# Depth handoff — comb command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:391`, `comb` / `do_action`  
Starting main: `0ed13a431`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `comfort` slice, the interpreter sweep selected
`comb` at line 391 in table order.

## C path and proof

The command table registers `comb` at `src/interpreter.c:391` for
`POS_RESTING`. The command reaches `do_action` in `src/act.social.c:102-151`.
The `comb` record at `lib/misc/socials:111-119` has a complete actor/room/
victim record, missing-target message, self-target message, and minimum victim
position zero. The successful path therefore accepts a sleeping target and
emits all three target audiences.

The RED-or-GREEN vehicle on main used a primary actor, a standing room peer,
and a sleeping room peer. It proved all five reachable cases with
`--show-oracle`: no argument, successful standing target, missing target,
self-target, and successful sleeping target. All transcripts matched C
exactly. No Go divergence was confirmed, so no implementation change was made;
neither `src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/comb-depth.txt` with the five target and
  audience cases.
- Add `docs/fidelity/depth/comb.tsv` with the five manifest rows.
- Add this dated handoff. The shared `do_action` implementation remains owned
  by `socials.tsv`; this slice records the registered `comb` command and its
  complete target record.

`comb-depth --seed 1 --show-oracle` is GREEN, and the updated manifest reports
`do_action: 21/21`.

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

The refreshed frontier is 1,449 total cases: 1,395 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,395/1,409 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the dispatch, audience, position
state, and delegation proof follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `comb`.
