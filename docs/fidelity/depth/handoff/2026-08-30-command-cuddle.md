# Depth handoff — cuddle command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:400`, `cuddle` / `do_action`  
Starting main: `41f1d01a6`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including active registration
tables, remains exhausted. The blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and remains
blocked; it was not repicked. The interpreter sweep selected `cuddle` at line
400, the first un-manifested command after the merged `ctell` slice. `cuddle`
routes to the shared `do_action` handler; its shared target-lookup and Act()
visibility behavior remains owned by the existing social manifests, while
this slice proves cuddle's own message record and gate topology.

## C path and proof

The command table registers `cuddle` at `src/interpreter.c:400` for
`POS_RESTING` and routes it to `do_action` in `src/act.social.c:102-151`.
The C social record at `lib/misc/socials:156-164` has `hide=1`, minimum victim
position `POS_RESTING`, a no-argument actor line, no room line, target-found
actor/room/victim lines, a missing-target line, a self-target line, and no
self room line. `do_action` checks `PLR_NOSHOUT` before parsing or resolving a
target.

The C-first `cuddle-depth --seed 1 --show-oracle` vehicle uses the primary
Implementor, a sleeping target, and a room observer. It proves the no-argument
line, sleeping-target position refusal, missing target, self target, and
standing-target actor/room/victim audience; the primary wakes the target with
`wake Sleepero` between the position and success probes. The separate
`cuddle-noshout --seed 1 --show-oracle` vehicle mutes a named mortal and proves
the pre-lookup `PLR_NOSHOUT` gate. The two vehicles are GREEN with exact
normalized output and audiences. No Go divergence was confirmed; neither
`src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/cuddle-depth.txt` with the C-first cuddle
  branch and audience vehicle.
- Add `cmd/dp-oracle-diff/scenarios/cuddle-noshout.txt` with the isolated
  `PLR_NOSHOUT` gate vehicle.
- Add the six rows in `docs/fidelity/depth/cuddle.tsv`.
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

The refreshed frontier is 1,490 total cases: 1,436 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,436/1,450 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the actual command dispatch,
social-record branches, and target/audience topology follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `cuddle`.
