# Depth handoff — 2026-08-31 — `goto`

## Frontier and queue position

- Started from clean `main` at `d3255cea1` after merging the `glare`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-glare.md`.
- The starting frontier was 1,868 total, with 1,811 proven/delegated, 16
  blocked, and 41 excluded. The dedicated `goto` manifest adds 13
  proven/delegated cases. The post-slice frontier is 1,881 total, with 1,824
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,824/1,840 (99.1%).
- A fresh source-order audit confirms `goto` at `src/interpreter.c:467` is
  covered. `gossip` remains covered by `channels.tsv` and `gold` by
  `spec-procs.tsv`; the next un-manifested command-table family is `goose` at
  `src/interpreter.c:470`. The next session must return to clean `main`, pull,
  rerun the frontier check, reread this handoff, and begin `goose` while
  leaving glare PR #896 open.

## C call path and branch inventory

`src/interpreter.c:467` registers `goto` with `POS_SLEEPING` and
`LVL_IMMORT`, dispatching to `do_goto`. The dispatcher applies the entry gate
before `src/act.wizard.c:275-305` runs. `do_goto` first calls
`find_target_room` (`act.wizard.c:184-239`), whose C `one_argument` skips fill
words and lowercases the first non-fill token. A numeric token without a dot
uses `atoi` and `real_room`; otherwise C resolves a visible player/mob and
then a visible object. An object at `NOWHERE` emits
`That object is not available.`. Missing input, invalid rooms, and unresolved
targets each have distinct C messages.

The resolved destination then applies `ROOM_GODROOM` and occupied
`ROOM_PRIVATE` restrictions to actors below `LVL_GRGOD`. Success emits the
actor's poof-out through `act(TO_ROOM)`, calls `char_from_room` and
`char_to_room`, moves a mount if present, emits poof-in through `act(TO_ROOM)`,
and calls `look_at_room(ch, 0)`. The Go path now uses the canonical world
character/object resolvers, named room-flag semantics, `PlayerTransfer` for
the player/mount transfer, canonical `game.Act` for both room audiences, and
`cmdMovementLook` for the post-move room observation. It no longer invents an
actor-only “You go to room” line.

## Coverage proof

The unchanged-main `goto-depth --seed 1 --show-oracle` run was RED for the
invalid-room, named-target, object-target, unavailable-object, poof-audience,
and trailing-fill-word branches. The two permission vehicles were also RED:
the Go path entered both an occupied private room and a god room instead of
emitting the C refusals. After the fix, `goto-depth`,
`goto-private-depth`, and `goto-godroom-depth` reported no normalized
divergence for seeds 1, 2, 3, 5, and 8.

The implementation follows R1/R2/R4/R5e and R5c: the C bytes, command gate,
lookup order, room restrictions, audience order, and actual call path remain
authoritative; no target or movement behavior was invented; and shared room
flag, transfer, Act, and look seams are reused rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/goto-depth.txt`,
  `goto-private-depth.txt`, and `goto-godroom-depth.txt`.
- Added `docs/fidelity/depth/goto.tsv` with 13 explicit rows.
- Updated `pkg/session/wiz_movement.go` and added the narrow public room-flag
  read boundary in `pkg/game/room_flags.go`.
- Added `pkg/session/wiz_movement_test.go` coverage for the C command gate,
  fill-word parsing, and private/god-room restrictions.
- Local gates passed: `make fidelity-depth` — 1,881 total /
  1,824 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

Implementation PR #898 was self-merged only after hosted `lint`, `security`,
and `test` checks were green. Its `build-and-push` and `deploy` jobs were
skipped by policy. The glare implementation PR #896 remains open because its
hosted test failure is the unrelated pre-existing retry-based
`pkg/spells/TestMagAffects_Sleep`; it was not retried or fixed forward.
