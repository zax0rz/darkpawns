# Depth handoff — con_seller — 2026-08-30

## Frontier and queue position

Started from clean `main` at `85f2bc543` after the merged
`teleport_victim` slice.  The full depth-testing guide and newest handoff
were read.  The initial `make fidelity-depth` checkpoint reported 1,262
cases: 1,214 proven/delegated, 14 blocked, and 34 excluded (98.9%
actionable).  The post-slice report is 1,269 total: 1,221 proven/delegated,
14 blocked, and 34 excluded (98.9% actionable).

The source-order inventory identifies `con_seller` immediately after
`teleport_victim`: `no_move_down` is the next unclaimed procedure in
`src/spec_procs3.c` and the next declaration/registration slot in
`src/spec_assign.c`.  No previously claimed procedure was repicked.

## C-first path audit

- The procedure is `src/spec_procs3.c:239-312`, declared at
  `src/spec_assign.c:150`, and registered as `ASSIGNMOB(21246, con_seller)` at
  `src/spec_assign.c:508`.  The registered world mobile is the shadow mage
  in room 21234, the dark “An Unlit Shop”; the disposable fixture strips its
  native script so the special dispatch is isolated.
- The entry path is exact: `cmd == 0`, a fighting actor, or an actor that is
  not awake returns `FALSE` before any visibility or command handling.  Only
  `list` and `buy` continue; all other commands fall through.  For the live
  command path, C calls `skip_spaces()` and then compares `buy`'s argument
  with exact case-insensitive `strcasecmp(argument, "con")`, so trailing
  whitespace is rejected.
- C's `CAN_SEE(mobile, ch)` gate emits the room-only exclamation
  `"$n exclaims, 'Who's there? I can't see you!'"` and returns handled.  The
  dark-shop path required verifying `LIGHT_OK` as well as the seller's
  infravision and the per-observer `PERS` substitution.
- `list` computes `GET_ORIG_CON(ch) - real_abils.con`, tells only the buyer
  either `You seem perfectly healthy!` or the exact pluralized purchase
  offer, and returns handled.  `buy` first rejects a non-`con` argument,
  then checks gold before the healthy branch, then deducts `level * 400`,
  directly tells the buyer the exact price sentence, sends two
  `TO_NOTVICT` room acts, increments base CON only below 18, calls
  `affect_total`, and sets the buyer stunned.  `orig_con` is the C persisted
  baseline; hard-coding 18 changes both list and purchase availability.

## RED and GREEN proof

Added `cmd/dp-oracle-diff/scenarios/spec-proc-con-seller.txt`.  It uses
`empty-players`, quiet mobs, registered mob 21246, a script-stripped
disposable spawn, and no room exits.  The peer must be teleported before the
actor enters the dark shop: otherwise C's visibility lookup correctly
rejects the remote arrival.  The probe proves the command surface, direct
list/tell bytes, rejected trailing argument, successful price and room
audience, and post-success stunned fallthrough.  The affordability branch
is pinned by the focused unit proof because the shared live stream preserves
the actor's state for the later success transcript.

The clean-main RED exposed three confirmed divergences: Go used a hard-coded
18 instead of C's `GET_ORIG_CON`, direct `do_tell` bodies retained the
recipient name, and `roomMessage` leaked or mis-cased room acts in the dark
shop instead of honoring C's target-only and `TO_NOTVICT` audiences.  The
special fix adds the runtime `OrigCon` baseline without changing the existing
Go save format, uses C's leading-only argument skip, routes direct tells to
the buyer, and supplies a narrow world-aware visibility/PERS room-act path.

The final oracle vehicle is green with `DP_ORACLE_BIN` set to
`/home/zach/darkpawns-c-oracle/bin/circle` on seeds 1, 2, 3, 5, and 8.  The
focused tests cover nil/command/fighting/asleep entry gates, dark-room
visibility, original-CON fallback and arithmetic, trailing-argument
rejection, affordability and healthy branch order, exact direct bytes,
actor/victim exclusions, gold/CON/position state, and post-stun fallthrough.

## Go changes and rules applied

- Added runtime-only `Player.OrigCon`, initialized at character creation and
  loaded-player construction; `GetOrigCon` falls back to current base CON
  for old/load-created players so the save format remains unchanged.
- Replaced the placeholder room broadcast with exact direct-tell and
  `TO_NOTVICT` behavior, including C's dark-room `LIGHT_OK`/infravision gate
  and whole-line capitalization/PERS casing.
- Added `pkg/game/spec_con_seller_test.go`, the oracle vehicle, and seven
  manifest rows covering D1 through D4 proof levels.  No con-seller branch is
  blocked or excluded.

Rules applied: R1 (player-facing bytes and audience), R2 (native command
surface), R3 (multiseed transcript proof), R4 (no invented recipient or room
messages), and R5/R5e (registration and live C call path verified before the
fix).  The runtime-only baseline is constrained by the repository's existing
save-format rule; no player-facing persistence format was invented or
changed.

## Gates

Passed on this branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check`

## Next action

Commit this slice, push `glm/spec-con-seller`, and open one PR.  Merge only
after every GitHub check is green; if CI does not fire, retry once with
`gh workflow run "Dark Pawns CI/CD" --ref glm/spec-con-seller`, then leave the
PR open and advance if it remains not-green.  After merge, reset to `main`,
pull, rerun `make fidelity-depth`, reread the depth guide and newest handoff,
and begin the next source-order procedure `no_move_down`.
