# 2026-08-29 — `rescuer` depth slice

## Frontier and queue

- Started from pulled `main` at `c9706d646`, after running
  `make fidelity-depth` and rereading `docs/fidelity/DEPTH_TESTING.md` plus
  the newest handoff.
- The pre-slice frontier was 1002 total cases: 967 proven/delegated, 12
  blocked, and 23 excluded. This slice adds seven proof cases and one
  registration exclusion, yielding 1010 total, 974 proven/delegated, 12
  blocked, and 24 excluded; actionable completion remains 974/986 (98.8%).
- `rescuer` was the next active source-order special after `stableboy`.
  Continue with `pissedalchemist` at `src/spec_procs2.c:546`, assigned to mob
  vnum 15814 at `src/spec_assign.c:422`. Do not repick `rescuer`, `stableboy`,
  `tipster`, `couch`, `whirlpool`, or earlier claimed procedures.

## C call path and branch census

- `SPECIAL(rescuer)` is defined at `src/spec_procs2.c:523-544` and assigned
  first to mob vnum 7909 at `src/spec_assign.c:263`, then to 15808 at
  `src/spec_assign.c:421`. The autonomous caller is
  `src/mobact.c:68-93`: it invokes the special with command zero after its
  active/awake/fighting filters and skips the next mobile when the special
  returns TRUE.
- The special rejects nonzero command, sleeping/non-awake, negative HP, and
  reciprocal existing combat. It scans the room in C list order for an NPC
  ally fighting a non-NPC and whose mob special is not `rescuer`; it calls
  `do_rescue(ch, GET_NAME(ally), 0, 1)` and returns TRUE after the first match.
- `src/act.offensive.c:501-581` supplies the callee path. `one_argument` uses
  the ally short description, then C applies the NPC subcommand behavior:
  skill absence is told only to the NPC and does not stop the special,
  target/self/current-fight/mounted/peaceful/no-attacker gates can reject,
  and `number(1,101)` still consumes a draw even though `subcmd=1` makes the
  rescue probability 100 (101 is the failure edge). Success emits the
  `TO_NOTVICT` `$n heroically rescues $N!` room line, stops the three native
  fight states, interposes the rescuer, synchronously calls `hit(vict,tmp_ch)`
  for the ally, and sets `WAIT_STATE(vict, 2 * PULSE_VIOLENCE)`. The direct
  `TO_CHAR` messages target NPCs in this registered surface and are not
  player-visible.

## RED/GREEN evidence and port result

- The first valid C-first vehicle used the real assigned random-load mob 7909
  in its fixed-seed landing room 21210, stripped its unrelated Lua script,
  and used indexed aggressive mob 4610 (Aatxe) as the real ally fight. A peer
  observed the rescue act. Clean `main` RED output showed Go's invented
  `Fear not!` room line and the rescuer attacking the player, while C emitted
  only `The elven avenger heroically rescues Elrik!` before the native ally
  hit.
- Source tracing exposed two shared combat divergences during this vehicle:
  C's high-level NPC switcheroo scan includes the current defender and
  consumes its `number(0,80)` draw; and C reaches that redirect from
  `damage()` after `hit()` has consumed the to-hit and weapon-damage draws,
  with the redirect branch calling synchronous `hit()` on the new target.
  `pkg/combat/engine.go` now follows that confirmed call order, and
  `TestMobRedirect_HighLevelSwitcheroo` preserves the regression. This is a
  shared combat fix under R5b/R5c, not a rescuer-only workaround.
- Go now mirrors the native rescuer path in `pkg/game/spec_procs2.go`: exact
  reciprocal gate, deterministic room scan, ally special exclusion, C-style
  short-description target resolution, the 1-in-101 edge, native audience,
  three-way teardown/interposition, canonical synchronous ally hit, and
  wait-state assignment. The focused tests in
  `pkg/game/mobact_spec_procs_test.go` cover entry gates, reciprocal combat,
  no-ally fallthrough, canonical ally hit, and wait state.
- `spec-proc-rescuer-7909` reports `result: no normalized divergence` for
  autonomous entry, rescue transition, observer audience, and downstream
  combat ordering. `spec-proc-rescuer-7909-no-ally` independently reports no
  divergence for the eligible-ally scan returning FALSE. Both use
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and seed 1; the
  intended C branch was checked with the live vehicle during RED/GREEN work.
- Mob vnum 15808 is recorded as `mob.rescuer-15808-no-spec` excluded. Its
  prototype `lib/world/mob/158.mob` has no `MOB_SPEC` action flag, so the
  C `mobile_activity` special dispatch cannot reach the assigned procedure;
  no synthetic 15808 behavior vehicle is valid under R2/R5e.
- No `src/` or `darkpawns-c-oracle/` file was edited. The unrelated untracked
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains
  preserved.

## Verification and integration

- Local gates are green: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- This slice applies R1 (player-facing bytes), R2 (autonomous return and
  registered command surface), R3 (shared RNG/state/wait parity), R4 (no
  invented output), and R5/R5b/R5c/R5e (verify the actual C call path and
  audit the shared combat class). The required PR/check/merge record will be
  added here when the GitHub review completes.

## Manifest

The durable rows are appended in `docs/fidelity/depth/spec-procs.tsv`:

- `mob.rescuer-entry-gates`
- `mob.rescuer-7909-no-ally-fallthrough`
- `mob.rescuer-target-gates`
- `mob.rescuer-7909-audience`
- `mob.rescuer-7909-rescue-transition`
- `mob.rescuer-7909-combat-swing`
- `mob.rescuer-7909-autonomous-entry`
- `mob.rescuer-15808-no-spec` (excluded)

