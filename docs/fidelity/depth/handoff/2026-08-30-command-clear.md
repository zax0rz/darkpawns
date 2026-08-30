# Depth handoff — clear-screen command family

Date: 2026-08-30  
Queue slice: `src/interpreter.c:385-386`, `clear` / `cls` / `do_gen_ps`  
Starting main: `09ae1386d`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `clap` slice, the interpreter sweep selected the
clear-screen family in table order: `clear` at line 385 followed immediately
by its `cls` alias at line 386.

## C path and proof

Both command rows dispatch to `do_gen_ps` with `SCMD_CLEAR`. The C branch is in
`src/act.informative.c:2120-2170` and sends exactly `"\033[H\033[J"`, ignoring
the command argument. The Go handler is `pkg/session/gen_ps_cmds.go:108-112`,
registered with `cls` as the alias in `pkg/session/commands.go:146`.

The RED-or-GREEN proof on main used `--show-oracle` and exercised both names
with no argument and with an ignored argument. All four transcripts matched
the C oracle with no extra bytes. No Go divergence was confirmed, so no
implementation change was made; neither `src/` nor `darkpawns-c-oracle/` was
edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/clear-depth.txt` with four durable cases
  for the `clear` and `cls` rows.
- Add `docs/fidelity/depth/clear.tsv` with the exact alias and argument cases.
- Add this dated handoff. No shared `do_gen_ps` implementation was duplicated;
  the slice records the two table entries and their shared C branch.

`clear-depth --seed 1 --show-oracle` is GREEN, and the updated manifest reports
`do_gen_ps: 4/4`.

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

The refreshed frontier is 1,430 total cases: 1,376 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,376/1,390 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the dispatch, shared-handler
boundary, exact terminal bytes, and proof audit follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table family after `cls`.
