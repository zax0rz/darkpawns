# Depth handoff — comfort command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:390`, `comfort` / `do_action`  
Starting main: `6a0f9baa5`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged color slice, the interpreter sweep selected
`comfort` at line 390 in table order.

## C path and proof

The command table registers `comfort` at `src/interpreter.c:390` for
`POS_RESTING`. The command reaches `do_action` in `src/act.social.c:102-151`.
The `comfort` record at `lib/misc/socials:121-129` has a minimum victim
position of `POS_RESTING` (5), with distinct no-argument, found, missing,
self-target, and proper-position messages. The successful target path emits
actor, room, and victim audiences; a sleeping target stops before those
messages.

The RED-or-GREEN vehicle on main used a primary actor, a standing room peer,
and a sleeping room peer. It proved all five reachable cases with
`--show-oracle`: no argument, successful target, missing target, self-target,
and sleeping-target position gate. All transcripts matched C exactly. No Go
divergence was confirmed, so no implementation change was made; neither
`src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/comfort-depth.txt` with the five audience
  and gate cases.
- Add `docs/fidelity/depth/comfort.tsv` with the five manifest rows.
- Add this dated handoff. The shared `do_action` implementation remains owned
  by `socials.tsv`; this slice records the registered `comfort` command and its
  distinctive social record.

`comfort-depth --seed 1 --show-oracle` is GREEN, and the updated manifest
reports `do_action: 16/16`.

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

The refreshed frontier is 1,444 total cases: 1,390 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,390/1,404 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the dispatch, audience, position
gate, and delegation proof follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `comfort`.
