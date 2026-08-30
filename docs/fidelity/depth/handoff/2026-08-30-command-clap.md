# Depth handoff — clap command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:384`, `clap` / `do_action`  
Starting main: `330f4fa26`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `clan` slice, the interpreter sweep selected
`clap`, command-table sequence 67, at line 384. Its shared `do_action` matrix
remains owned by `docs/fidelity/depth/socials.tsv`; this slice proves the
previously un-manifested registered command name directly.

## C path and proof

The command table registers `clap` at `src/interpreter.c:384` with
`POS_RESTING`. The command reaches `do_action` in `src/act.social.c:102-151`.
The `clap` record at `lib/misc/socials:106-109` has only `char_no_arg` and
`others_no_arg`; its `char_found` field is NULL. C therefore emits the actor
and room no-argument pair for no input, a present target, a missing target, or
the actor's own name. It does not enter target lookup, not-found, self-target,
or victim-position branches. The Go command fallback and `DoAction` path are
`pkg/session/commands.go` and `pkg/game/act_social.go`.

The RED-or-GREEN proof on main used `--show-oracle` and a primary actor plus a
room peer. All four cases matched C exactly: no argument, a present target, a
missing target, and a self-named target. No Go divergence was confirmed, so no
implementation change was made; neither `src/` nor `darkpawns-c-oracle/` was
edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/clap-depth.txt` with four named cases for
  the registered command and its self-only record.
- Add `docs/fidelity/depth/clap.tsv` with the four proven rows.
- Add this dated handoff; shared social behavior remains delegated to the
  existing `socials.tsv` matrix under R5c.

`clap-depth --seed 1 --show-oracle` is GREEN, and the updated manifest reports
`do_action: 11/11`.

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

The refreshed frontier is 1,426 total cases: 1,372 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,372/1,386 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the command dispatch, social record,
audience proof, and delegation boundary follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table name after `clap`.
