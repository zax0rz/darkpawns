# Depth handoff — curtsey command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:402`, `curtsey` / `do_action`  
Starting main: `1606fbc12`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including active registration
tables, remains exhausted. The blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and remains
blocked; it was not repicked. The interpreter sweep selected `curtsey` at line
402, the first un-manifested command after the merged `curse` slice. `curtsey`
routes to the shared `do_action` handler; shared lookup and Act() behavior
remain owned by the existing social manifests, while this slice proves
curtsey's own no-argument record, argument-discard behavior, and gate.

## C path and proof

The command table registers `curtsey` at `src/interpreter.c:402` for
`POS_STANDING` and routes it to `do_action` in `src/act.social.c:102-151`.
The C social record at `lib/misc/socials:171-174` has `hide=0`, minimum victim
position zero, actor text `You curtsey to your audience.`, room text
`$n curtseys gracefully.`, and no `char_found` message. Consequently,
`do_action` skips target parsing and emits only the no-argument actor/room pair
even when an argument is typed. `PLR_NOSHOUT` is checked before that pair.

The C-first `curtsey-depth --seed 1 --show-oracle` vehicle uses the primary
Implementor and a room observer to prove the no-argument actor/room output and
typed-argument discard. The separate `curtsey-noshout --seed 1 --show-oracle`
vehicle mutes a named mortal and proves the pre-message emote refusal. Both
vehicles are GREEN with exact normalized output and audiences. No Go
divergence was confirmed; neither `src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/curtsey-depth.txt` with the no-argument
  and argument-discard vehicle.
- Add `cmd/dp-oracle-diff/scenarios/curtsey-noshout.txt` with the isolated
  `PLR_NOSHOUT` gate vehicle.
- Add the three rows in `docs/fidelity/depth/curtsey.tsv`.
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

The refreshed frontier is 1,496 total cases: 1,442 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,442/1,456 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the actual command dispatch,
social-record branch, and audience topology follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `curtsey`.
