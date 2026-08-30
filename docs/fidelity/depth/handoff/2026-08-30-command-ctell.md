# Depth handoff — ctell command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:399`, `ctell` / `do_ctell`  
Starting main: `f471ccb4d`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including active registration
tables, remains exhausted. The blocked `objmagic.sleep-entry-gates` row was
already attempted through the cast-sleep outlaw/reagent vehicle and remains
blocked; it was not repicked. The interpreter sweep selected `ctell` at line
399, immediately after the merged `cry` slice and in command-table order. The
earlier `group-clan-channels` evidence owns only the clanless-mortal rejection;
this slice covers the remaining command path without repicking that case.

## C path and proof

The command table registers `ctell` at `src/interpreter.c:399` for
`POS_SLEEPING` and routes it to `do_ctell` in `src/act.comm.c:1451-1565`. C
uses a distinct syntax for immortals: `ctell <clan-number> <message>`, with
invalid clan numbers rejected first. Mortals use their own clan and rank; a
leading `#<rank>` filters recipients after the channel and noshout gates. The
remaining branches are the valid/invalid clan-number gates, empty message,
rank-prefix parse and range gates, sender echo versus `PRF_NOREPEAT`, and
same-clan recipient rank/channel/visibility filtering.

The C-first `ctell-depth --seed 1 --show-oracle` vehicle creates two two-rank
clans with online rank-two members; the second clan makes the C `clan[c]`
rank-prefix lookup reachable with its actual source indexing. The first
three-client RED run confirmed that Go rejected the immortal as clanless and
missed the immortal clan-number, rank-prefix, sender-echo, and no-repeat
branches. The Go path was then aligned to the C gates and broadcast topology;
the corrected vehicle proves the
immortal no-argument and invalid-number gates, valid-number empty-message
prompt, invalid and over-high rank prefixes, rank-filtered delivery, normal
sender/recipient delivery, and the no-repeat sender branch. All eight blocks
are GREEN with exact actor and recipient audiences. Neither `src/` nor
`darkpawns-c-oracle/` was edited.

## Changes

- Add `cmd/dp-oracle-diff/scenarios/ctell-depth.txt` with the remaining
  immortal clan-tell matrix and a three-client setup that reaches the C
  rank-index path without a fourth-client setup dependency.
- Add the eight rows in `docs/fidelity/depth/ctell.tsv`.
- Align `pkg/session/comm_cmds.go` and `pkg/game/comm_channel.go` with the
  confirmed C immortal syntax, sender gates, rank-prefix handling, visibility,
  recipient filtering, and `PRF_NOREPEAT` behavior.
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

The refreshed frontier is 1,484 total cases: 1,430 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,430/1,444 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the actual immortal/mortal dispatch,
rank-prefix parser, sender echo, and recipient topology follow R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `ctell`.
