# Acknowledged Gap Sweep — Train 11: `do_turn`

## Checkout and scope

Train 11 was implemented in the fresh worktree
`/home/zach/.codex/worktrees/luna-gap-sweep-11/darkpawns`, branch
`codex/luna-acknowledged-gap-sweep-11`, based on `origin/main` at
`d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`. The root checkout was not edited.

This train closes the acknowledged `do_turn` room-scan and name-guessing gap
from the [2026-10-13 comment inventory](2026-10-13-acknowledged-gap-sweep-comments.tsv)
and [special crosswalk](2026-10-13-acknowledged-gap-sweep-specials.tsv). The
full sweep inventory and earlier train evidence remain in the
[2026-10-13 handoff](2026-10-13-acknowledged-gap-sweep.md). The new depth rows
are in [`turn.tsv`](../turn.tsv); direct flee's uncovered NPC lane is recorded
there and in [`flee.tsv`](../flee.tsv).

## C evidence checked

- `src/new_cmds2.c:960-1030`: arguments are unused; the skill and good-alignment
  gates precede one `number(1,55)` mana cost; every room body is visited in
  `world[room].people` order; race IDs 8 and 9 select the shiver, no-effect,
  flee, and `raw_kill(SPELL_DISINTEGRATE)` branches.
- `src/structs.h:139-140`: `RACE_VAMPIRE=8`, `RACE_UNDEAD=9`.
- `src/handler.c:532-550`: `char_to_room` prepends each body to the room list.
- `src/act.offensive.c:360-415` and `src/act.movement.c:95-305`: direct flee
  wait gates, six independent direction draws, movement checks, audiences,
  scripts, and destination behavior.
- `src/mobprog.c:62-88`: the actual `mp_greet` dog, shopkeeper, and vnum 19406
  branches reached by successful movement.
- `src/interpreter.c:1393-1479`: C's room, equipment, inventory, mobile, and
  floor-object special ordering and PC-only Lua lanes.
- `src/fight.c:470-476,534-581`: disintegration dust and `raw_kill` ordering.

`src/` and the shared C oracle binary were read-only.

## Changes

- `CmdTurn` now sends no target lookup and delegates to a room-wide
  `World.DoTurn`, matching C's ignored arguments and gate ordering.
- Turn selects numeric body race, uses C's IDs, charges one seeded cost, clamps
  the low-mana deduction, and preserves the room prepend order through
  `RoomEntrySequence`.
- The shiver, flee, and disintegration arms use their authored room/private
  messages. Player targets use canonical player movement; NPC targets use a
  direct flee path that preserves the C wait gate, direction draws, movement
  text, entry/greet hooks, and room-enter hook.
- `MpGreetActor` ports the C trigger for both player and NPC movers; the prior
  Go body was not the C `mp_greet` behavior.
- The direct NPC movement path blocks registered legacy-only special handlers
  when dispatch returns `SpecialUnsupported`, rather than silently treating
  that lane as parity. This is still a visible behavior gap and is kept
  blocked in the manifest.

## Proof

- `go run ./cmd/dp-oracle-diff --scenario turn-depth --show-oracle --seed 1`:
  `result: no normalized divergence`; C emits
  `You are not holy enough to turn away the Undead!` for the supplied trailing
  words. The stock mortal setup cannot set good alignment through its
  non-wizard commands, so this oracle vehicle claims only the ignored-argument
  and neutral-alignment gates. The deeper room outcomes are direct world tests.
- Focused `go test` covers mixed player/mobile prepend order, numeric race vs.
  display-name guesses, exact messages, no-effect and shiver bands, mana draw
  and deduction, both dust races, player flee, NPC flee and its wait gate.
- R5h controls were run with the fix temporarily reverted, then restored:
  - Set `RACE_UNDEAD` back to 3: `TestDoTurnScansCOrderedRoomBodiesByNumericRace`
    failed because the lich shiver line was missing.
  - Restored the target prompt in `CmdTurn`: `TestCmdTurn_NoSkillIgnoresArguments`
    failed with `Turn who?` instead of C's `Huh?!?`.
  - Narrowed the cost draw to `number(1,54)` with seed 2:
    `TestDoTurnScansCOrderedRoomBodiesByNumericRace` failed (`mana = 887,
    want 886`).
  - Reversed the room sequence comparator:
    `TestDoTurnScansCOrderedRoomBodiesByNumericRace` failed on target order.
  - Omitted the direct `DoFlee` call:
    `TestDoTurnFleeUsesNpcDirectMovementAndPreservesDirectionDraw` failed
    because the mob did not reach the destination.

The final code gates passed: `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `go test -race ./pkg/game/...`,
`golangci-lint run ./...` (0 issues), `make deadcode-ratchet` (52 baselined,
0 new), `make fidelity-depth` (5,534 cases; 5,443 proven/delegated, 12 blocked,
67 excluded, 12 divergent-approved), `make fidelity-units` (1,620 proof claims
passed), `make string-census` (ratchet passed; it warned that generated reports
are stale, and no report update was made), and `gosec -severity=high -quiet ./...`.
The first full-suite run hit the unrelated 3-second timeout in
`pkg/admin/TestMudlogStreamReplaysAndTails` while the other gates were running;
the standalone `go test ./...` retry passed. Unit evidence is retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-units-064739`.

The final combined census, commit, and PR are pending. No governing document
or oracle source was edited. No rulebook amendment is proposed by this train.

## Remaining blocked work

The NPC `special(ch, dir+1, "")` path does not reproduce assigned legacy-only
room, object, or mobile special callbacks and their side effects or text. The
base NPC movement gates exist, but the boat, charm, tunnel, mounted, greet, and
script matrix still needs a full depth proof. `turn.npc-flee-special-lane` and
`flee.transitive-movement-failures` remain blocked; do not count them as
complete or merge a later train that claims them without evidence.
