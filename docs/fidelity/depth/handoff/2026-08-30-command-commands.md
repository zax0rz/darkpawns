# Depth handoff — commands command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:392`, `commands` / `do_commands`  
Starting main: `251ae8087`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including the active
registration tables, remains exhausted. The blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked; it was not repicked. After the
merged `comb` slice, the interpreter sweep selected `commands` at line 392 in
table order.

## C path and proof

The command table registers `commands` at `src/interpreter.c:392` and routes it
to `do_commands`. The implementation is in `src/act.informative.c:2585-2675`:
`sort_commands` orders `cmd_info[]` lexically, starts at index 1 to omit
`RESERVED`, and marks both `do_action` entries and the special `insult` entry
as socials. `do_commands` parses one optional target, rejects missing/NPC
targets with `Who is that?`, rejects a target above the actor's level, applies
the mortal command/social filters, formats seven 11-column entries per line,
and sends the result through `page_string`.

The RED vehicle initially exposed a real divergence: Go used its runtime
registry, emitted a different header and five-column layout, omitted the C
pager, and included `insult` in the normal command list. The confirmed fix
uses the checked-in C `cmd_info[]` extraction (`pkg/session/command_order.tsv`)
for level filtering and lexical ordering, excludes socials plus C's explicit
`insult` special case, reproduces the seven-column bytes, and calls the Go C
pager. The target lookup retains the C case-insensitive visible-player gate.

The final `commands-depth --seed 1 --show-oracle` vehicle is GREEN and proves
four cases: no argument, explicit self, a visible same-level peer, and a
missing target. The peer path also proves that the target's level controls the
listing while the requester remains the pager audience. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Changes

- Add `pkg/session/c_command_list.go` to parse the C command-order artifact and
  build the faithful mortal command list.
- Align `cmdCommands` in `pkg/session/act_informative.go` with the C header,
  layout, social exclusion, and pager path.
- Add `cmd/dp-oracle-diff/scenarios/commands-depth.txt` and the four rows in
  `docs/fidelity/depth/commands.tsv`.
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

The refreshed frontier is 1,453 total cases: 1,399 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,399/1,413 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the dispatch, command-table
ordering, social special case, target gate, audience, and pager path follow
R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `commands`.
