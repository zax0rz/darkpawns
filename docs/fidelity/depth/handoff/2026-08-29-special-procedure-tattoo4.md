# Dated Handoff: 2026-08-29 — `tattoo4` depth slice

## Frontier and queue

- Started from synchronized `main` after the required checkout, fast-forward
  pull, `make fidelity-depth`, and reread of `docs/fidelity/DEPTH_TESTING.md`
  plus the newest prior handoff, `2026-08-29-special-procedure-identifier.md`.
- The pre-slice frontier was 1061 total cases: 1023 proven/delegated, 12
  blocked, and 26 excluded. The `tattoo4` slice adds seven proven cases,
  yielding 1068 total: 1030 proven/delegated, 12 blocked, and 26 excluded.
  Actionable completion is 1030/1042 = 98.8%.
- `SPECIAL(tattoo4)` is assigned to mob vnum 2766 at `src/spec_assign.c:213`.
  The next actual registered source-order procedure is `take_to_jail` at
  `src/spec_procs2.c:1427`, assigned to mob vnums 8027, 8059, 8001, 8002,
  and 8020 at `src/spec_assign.c:285-291`.

## C call path and branch census

- `SPECIAL(tattoo4)` is defined at `src/spec_procs2.c:1282-1340`. The
  registered vnum-2766 mobile is the dispatch vehicle; its script was stripped
  in disposable fixtures so the C special is the exercised path.
- `list` emits the direct tattoo catalog for source-selected tattoo numbers
  5, 1, and 3. `buy` skips spaces, rejects bare, nonnumeric, and out-of-range
  arguments, rejects an already-owned tattoo, rejects insufficient gold, and
  otherwise enters shared `give_tat`, which owns the pain, room, shout,
  blackout, gold, and tattoo-state behavior.
- The slice added exact entry/list/owned/price/fallthrough rows, a live
  actor/peer audience proof, and focused state proofs for all three offers and
  the shared mutation path. This follows R5e through `give_tat` rather than
  inventing a second tattoo implementation.

## RED/GREEN evidence and port result

- Main RED showed the old Go placeholder implementing unrelated tattoo removal
  instead of the registered C artist's list/buy surface. The C-first vehicles
  also established the exact catalog, trailing-space owned tell, affordability
  tell, and funded actor/peer transcript.
- GREEN vehicles are:
  `cmd/dp-oracle-diff/scenarios/spec-proc-tattoo4.txt`,
  `spec-proc-tattoo4-owned.txt`, `spec-proc-tattoo4-price.txt`, and
  `spec-proc-tattoo4-success.txt`. Each reports no normalized divergence with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`; the funded vehicle
  was inspected with `--show-oracle`.
- Focused coverage is in `pkg/game/spec_tattoo4_test.go`, including every
  offered tattoo number/effect, exact entry/price/owned gates, audience, gold,
  and state transition.

## Verification and integration

- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .` (the local high-severity gosec scan also passed).
- PR #764 (`glm/spec-tattoo4`) passed all GitHub checks and was squash-merged
  to `main` as `21cf6f5f6`. Build-and-push and deploy were skipped by workflow
  policy.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Inventory correction and next item

- The source/registration diff after `tattoo4` found three declared but
  unregistered bodies: `evillead` (`spec_procs2.c:1342-1388`), `little_boy`
  (`:1391-1403`), and `ira` (`:1405-1425`). Their declarations in
  `assign_mobiles` are not dispatch registrations; no ASSIGNMOB/ASSIGNOBJ/
  ASSIGNROOM entry exists. They are now durable D5 `excluded` rows in
  `docs/fidelity/depth/spec-procs.tsv`, under R2/R4/R5e, with no synthetic
  vehicle claimed.
- Continue with registered `take_to_jail`, in C source and registration order.
  Do not repick `tattoo4`, `identifier`, `eviltrade`, `couch`, `whirlpool`, or
  any earlier claimed procedure.

## Manifest

The durable rows added by the tattoo4 slice are:

- `mob.tattoo4-list`
- `mob.tattoo4-entry-gates`
- `mob.tattoo4-owned-gate`
- `mob.tattoo4-price-gate`
- `mob.tattoo4-success-audience`
- `mob.tattoo4-success-state`
- `mob.tattoo4-fallthrough`

The inventory correction adds:

- `mob.evillead-unassigned`
- `mob.little-boy-unassigned`
- `mob.ira-unassigned`

This handoff applies R1 (exact bytes), R2 (registered command surface), R3
(gold/tattoo state), R4 (no invented unregistered vehicles), and R5/R5e
(actual registration, C call path, and shared `give_tat` ownership).
