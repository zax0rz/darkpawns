# 2026-08-29 — `assassin` depth slice

## Frontier and queue

- This slice began from synchronized `main` after the required
  `git checkout main`, `git pull --ff-only`, `make fidelity-depth`, and reread
  of `docs/fidelity/DEPTH_TESTING.md` plus the newest dated handoff.
- The pre-slice frontier was 1019 total cases: 982 proven/delegated, 12
  blocked, and 25 excluded. The slice adds eleven assassin cases, yielding
  1030 total: 993 proven/delegated, 12 blocked, and 25 excluded. Actionable
  completion is 993/1005 = 98.8%.
- The preceding remorter handoff left assassin's registration audit pending.
  The complete dispatch audit found `ASSIGNROOM(8114, assassin)` at
  `src/spec_assign.c:608` and no `ASSIGNMOB` registration. Room 8114 is
  reachable and therefore assassin is not excluded; the next queue item is
  `tattoo1` at `src/spec_procs2.c:945`, registered to mob vnum 8086 at
  `src/spec_assign.c:296`.

## C call path and branch census

- `SPECIAL(assassin)` is defined at `src/spec_procs2.c:845-924` and is
  dispatched as a room special through the room command path in
  `src/interpreter.c:1407-1415`. The procedure derives its roster room as
  `ch->in_room + 1`; for room 8114 that is room 8115.
- `list` emits the direct instructions, the available-assassin header, and
  one `%8d - %s` row per roster occupant. `hire` handles, in C order, bare
  hire, a missing roster member, a roster player, a missing victim, gold
  insufficiency, a victim below level 5, and successful hiring. Unrelated
  commands return FALSE to the normal interpreter. The player-roster branch
  sends the expulsion line to the selected player and the refusal only to the
  hiring actor. Success deducts level*1000 gold, clones the selected mob
  prototype, places it in the actor's room, sets `MOB_HUNTER`, assigns the
  victim, tells the actor the security line, and emits the room-only hire act.
- The disposable oracle vehicle uses the real roster mob vnum 8070 (street
  urchin) reset into room 8115, with `quiet-mobs`, `spawn-mob 8070 1 8115 80`,
  and `replace-room-exits 8115 none` to prevent movement during setup. The
  fixture changes only disposable C/Go world copies. C's `GET_NAME` for this
  NPC is its short description, matching Go's `MobInstance.GetName()`.

## RED/GREEN evidence and port result

- RED on `main` with the registered room vehicle showed Go falling through to
  generic `Sorry, but you cannot do that here.` while C emitted the assassin
  list and hire branches.
- GREEN oracle vehicles cover list and FALSE fallthrough, bare-hire and
  missing-roster gates, player-roster selection and audience split, the
  insufficient-gold gate, the level-below-five victim gate, the invisible or
  unknown victim branch, and successful hiring. The success vehicle proves
  actor gold 1000 -> 0, silent assassin placement, the room-only hire act, and
  the hunter/target state. All five scenario vehicles report no normalized
  divergence with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and
  seed 1.
- Focused unit proof in `pkg/game/spec_assassin_test.go` covers room list and
  entry gates, hire gates and state, invisible-victim rejection, and player
  roster rejection. Scenario manifests are in
  `docs/fidelity/depth/spec-procs.tsv` and use the
  `spec-proc-assassin*.txt` vehicles.

## Verification

- Green local gates: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, clean `gofumpt -l .`, and clean
  `git diff --check`.
- PR #759 (`glm/spec-assassin`) passed every applicable GitHub check and was
  squash-merged into `main` as `6e897d38a`. No CI retry was required. No
  `src/` or `darkpawns-c-oracle/` file was edited.
- This slice applies R1 (exact bytes), R2 (command surface and FALSE
  fallthrough), R3 (gold and spawned-state parity), R4 (no invented output),
  and R5/R5e (complete registration audit and actual C call path).

## Manifest

The durable rows are in `docs/fidelity/depth/spec-procs.tsv`:

- `room.assassin-list`
- `room.assassin-hire-entry-gates`
- `room.assassin-low-level-victim`
- `room.assassin-gold-gate`
- `room.assassin-victim-not-found`
- `room.assassin-player-roster`
- `room.assassin-player-rejection`
- `room.assassin-success-audience`
- `room.assassin-success-state`
- `room.assassin-invisible-victim`
- `room.assassin-fallthrough`

## Next queue item

Continue the special-procedure inventory with `tattoo1` in
`src/spec_procs2.c:945`, assigned to mob vnum 8086 at
`src/spec_assign.c:296`. First map its complete C call path, including the
tattoo table and `give_tat`, then build the registered-mob oracle vehicle in
source/registration order.
