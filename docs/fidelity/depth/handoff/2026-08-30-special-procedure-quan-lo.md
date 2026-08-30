# Depth-fidelity handoff: `quan_lo`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `fb99cbec5`

## Scope and source audit

This slice covered `SPECIAL(quan_lo)` in `src/spec_procs3.c:372-392`. The
registered procedure is attached to mob vnum 19405 by
`ASSIGNMOB(19405, quan_lo)` at `src/spec_assign.c:444`; the source table has no
other active assignment for this procedure. The authored vehicle is
`lib/world/mob/194.mob` in room 19424 with `quanlo.lua`; the proof fixture
strips that script and sets the native `SPEC` flag so only the C-assignment
surface is exercised.

The C body has these player-visible branches and gates:

1. A nonempty command and an awake mob are required; commandless, sleeping,
   and all other non-awake calls return false silently.
2. `flee`, `retreat`, and `escape` each call `do_gen_comm` with the exact
   mobile-authored gossip text, then the special returns false so the ordinary
   command continues.
3. `look` and `examine` trim the argument and use C `isname()` against the
   mobile keyword list. An exact keyword emits the exact room-wide speech and
   then falls through; a non-keyword is silent at the special layer.

The command path was verified through `src/interpreter.c:889-947` and
`1407-1456`: the special is checked before ordinary command dispatch and its
false return preserves fallthrough. The gossip implementation is C
`do_gen_comm` in `src/act.comm.c:1146-1240`, whose NPC sender uses the global
descriptor audience and filters no-gossip, writing, and soundproof recipients.
The look/examine room audience is C `act()` through the ordinary room
communication path.

## Confirmed port divergence and fix

The first RED vehicle showed two confirmed Go divergences: Quan Lo's gossip
was routed through an existing same-room helper, omitting a remote eligible
player, and the look/examine predicate used a short-description substring
check instead of C's exact case-insensitive keyword token match. The slice
adds the dedicated `World.mobGlobalGossip` path in
`pkg/game/comm_channel.go` and the exact `isCName` predicate used by
`specQuanLo` in `pkg/game/spec_procs3.go`. The shared graph helper was not
changed; R5c was considered and the unrelated command behavior remains
outside this slice.

The ordinary `retreat` and `escape` command handlers are not present in the
current Go command registry. They remain owned by the later
`src/interpreter.c` command-table sweep; this slice proves the special's three
alias branches and its false-return contract without inventing those commands.
Likewise, generic look/examine output after the special falls through remains
separate ordinary-command coverage; this handoff claims only the special's
room response and exact predicate.

## Vehicles and proof

- `cmd/dp-oracle-diff/scenarios/spec-proc-quan-lo.txt` spawns registered mob
  19405, strips its script, sets `SPEC`, removes exits from room 19424, and
  moves a peer to remote room 10049. The `flee` vehicle normalized green for
  seeds 1, 2, 3, 5, and 8, including the global gossip seen by the remote
  peer and the canonical no-exit flee fallthrough. Seed 1 was also inspected
  with `--show-oracle`.
- `pkg/game/spec_quan_lo_test.go` proves the awake/command gate, all three
  command aliases, false return, remote global-gossip delivery, exact
  `look`/`examine` keyword matching, and C recipient filters for no-gossip,
  writing, and soundproof players.
- Seven manifest rows were added to `docs/fidelity/depth/spec-procs.tsv`.

No `src/` or `darkpawns-c-oracle/` file was edited. The evidence satisfies
R1/R2/R3/R4/R5e: player bytes, command surface, seeded vehicle parity, no
invented ordinary commands, and verified call paths are covered. R5c is
explicitly preserved by keeping the generic same-room helper and the later
ordinary command sweep separate.

## Verification

All required gates pass on `glm/spec-quan-lo`: `make fidelity-depth` reports
1,283 total cases, 1,234 proven/delegated, 14 blocked, and 35 excluded
(1,234/1,248 actionable, 98.9%); `go test ./pkg/game -run
'TestSpecQuanLo|TestMobGlobalGossip' -count=1`; `go build ./...`; `go vet
./...`; `go test ./...`; `golangci-lint run ./...` (`0 issues`); `gofumpt -l
.` (no output); and `git diff --check` all pass.

## Next queue item

Continue special-procedure source order with `SPECIAL(alien_elevator)` at
`src/spec_procs3.c:401`. A complete registration search found only its
declaration in `src/spec_assign.c:589` and no active `ASSIGNMOB`,
`ASSIGNOBJ`, or `ASSIGNROOM` entry, so the next session should record the
unreachable body as a D5 excluded procedure under R2/R4/R5e and advance.
