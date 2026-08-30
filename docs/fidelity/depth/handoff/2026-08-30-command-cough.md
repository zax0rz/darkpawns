# Depth handoff — cough command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:395`, `cough` / `do_action`  
Starting main: `17691c1f1`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including the active
registration tables, remains exhausted. The blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked; it was not repicked. After the
merged `compare` slice, the interpreter sweep selected `cough` at line 395 in
table order. The `compact` command immediately before this row is already
owned by `gen-tog.tsv`; the shared social matrix remains owned by
`socials.tsv`, but this registered social record was selected for its own
source-order proof.

## C path and proof

The command table registers `cough` at `src/interpreter.c:395` for
`POS_RESTING` and routes it to `do_action`. The C social loader reads the
record at `lib/misc/socials:131-134`: hide is zero, minimum victim position is
zero, `char_no_arg` is `Yuck, try to cover your mouth next time!`,
`others_no_arg` is `$n coughs loudly.`, and `char_found` is `#`/NULL. In
`src/act.social.c:102-151`, the `PLR_NOSHOUT` gate runs first; because the
record has no `char_found`, every argument is ignored and the handler emits the
no-argument actor/room pair. There is no self-target, missing-target, victim,
position, or RNG branch for this record.

The C-first `cough-depth --seed 1 --show-oracle` vehicle uses a primary actor
and a same-room peer, then compares no argument, a visible peer, a missing
target, and the actor's own name. All four blocks are GREEN and show the same
actor/room pair, proving the NULL `char_found` argument behavior and audience
routing. No Go divergence was confirmed, so no implementation change was
made; neither `src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/cough-depth.txt` with the four argument
  and audience cases.
- Add the four rows in `docs/fidelity/depth/cough.tsv`.
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

The refreshed frontier is 1,466 total cases: 1,412 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,412/1,426 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the registered social dispatch,
NULL target branch, and actor/room audience path follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `cough`.
