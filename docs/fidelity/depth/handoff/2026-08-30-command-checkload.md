# Depth handoff — checkload command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:381`, `checkload` / `do_checkload`  
Starting main: `1ad1db6ea`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including the active
registration tables, remains exhausted. The blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
remains blocked; it was not repicked. The interpreter sweep therefore selected
`checkload`, the next un-manifested family after the merged `circle` slice.

## C path and proof

The command table registers `checkload` for `POS_DEAD`, level `LVL_IMMORT`, at
`src/interpreter.c:381`. `src/act.wizard.c:3847-3870` uses `two_arguments`,
requires a first-byte decimal vnum, dispatches on the first type byte (`M/m`
or `O/o`), and emits C's exact usage and missing-prototype messages. C's
`atoi` accepts a numeric prefix. `check_load` at
`src/act.wizard.c:3681-3817` scans reset commands while retaining the last room,
object, and mob context, then reports M, O, P, E, G, and R branches with the
prototype load percentage/max values. Mob names come from the NPC short
description, and object load percentage comes from `GET_OBJ_LOAD`.

The RED vehicle on main exposed invented usage/prototype text, strict rather
than C-prefix parsing, a concise count instead of the exact multi-line reset
report, and missing type aliases. A no-`quiet-mobs` vehicle then exercised the
positive M/O/P/E/G/R rows and the no-load tail. The C call path and stateful
reset scan were mapped before changing Go; neither `src/` nor
`darkpawns-c-oracle/` was edited.

## Changes

- `pkg/session/wiz_info.go`: implement the C argument gates, decimal-prefix
  parsing, exact errors, and ordered mob/object reset reports.
- `pkg/session/wiz_info_checkload_test.go`: cover C-prefix parsing and synthetic
  M/O/P/E/G/R report state, including legacy object removal.
- `cmd/dp-oracle-diff/scenarios/checkload-depth.txt`: durable vehicle for all
  manifest cases, with the registered mob/object prototypes and live reset
  branches.
- `docs/fidelity/depth/checkload.tsv`: 17 depth cases covering argument gates,
  report branches, aliases, prefix parsing, and the no-load result.

`checkload-depth` is GREEN at seeds `1,2,3,5,8`; `--show-oracle` was used to
inspect the exact C bytes and branch coverage. The focused state test is
`TestCheckloadReportsMobAndObjectResetBranches`.

## Gates and frontier

The following all pass on the slice:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The refreshed frontier is 1,372 total cases: 1,318 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,318/1,332 = 98.9%.

This work follows R1/R2/R3/R4 and R5/R5e; the call-path, reset-state, and
draw/proof audit follow R5b/R5c. The command surface and player-facing bytes
remain grounded in the C oracle.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take `clan` at `src/interpreter.c:383` in table order.
