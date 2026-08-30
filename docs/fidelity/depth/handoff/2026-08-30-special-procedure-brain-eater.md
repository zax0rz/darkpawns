# Depth handoff — brain_eater — 2026-08-30

## Frontier and queue position

Started from refreshed clean `main` at `bb3752df1` after the prior
`fly_exit_up` slice.  `make fidelity-depth` before this slice reported 1,245
cases: 1,198 proven/delegated, 13 blocked, and 34 excluded (98.9% actionable).
The required depth-testing guide and newest prior handoff were read.  The
special-procedure inventory and registration census identify `brain_eater` as
the first active, unclaimed definition in source order.  After this slice,
`teleport_victim` (mob 14405) is next; `con_seller` follows it (mob 21246).

## C-first path audit

- Definitions and entry gates: `src/spec_procs3.c:198-223`.  The procedure is
  commandless and is called autonomously with the mob as both `ch` and `me`.
  It returns FALSE when fighting, when `cmd` is nonzero, when asleep, or when
  `GET_HIT(ch) < 0`.
- Active registrations: `src/spec_assign.c:386` (`14420`) and `:389`
  (`14432`); declaration at `:148`.  The live vehicle used the first
  registration, mob 14420.
- The procedure scans room contents for a container with nonzero value[3], a
  literal `corpse` keyword substring, and no literal `headless` substring.  It
  then calls `do_behead(mobile, "corpse", 0, 0)` and unconditionally emits its
  multiline room Act and updates the spawned mob: level +1 below 30, otherwise
  damroll +2.
- `do_behead` is `src/new_cmds.c:222-365`; its room object lookup and linked
  object mutations are `src/handler.c:1341-1374,939-967`.  It emits the NPC
  room rip/behead Act, creates proto 16 head and proto 17 headless corpse,
  transfers contents, and extracts the original.  Native corpse creation used
  by the vehicle is `src/fight.c:534-633`.  Autonomous dispatch is
  `src/mobact.c:68-93`; Act rendering is `src/comm.c:2392-2555`.

## RED and GREEN proof

Added `cmd/dp-oracle-diff/scenarios/spec-proc-brain-eater.txt`.  The vehicle
uses `empty-players`, `quiet-mobs`, registered mob 14420, a disposable trainee
corpse, a same-room peer, and a disposable-world-only clear of
`MOB_AGGRESSIVE` so unrelated combat cannot consume the pulse.  The initial
main implementation was RED on seed 1: C emitted the rip and brain-eating
Acts and left a beheaded corpse; Go emitted neither and left the native corpse
unchanged.  The disposable fixture parser was extended to support clearing
`AGGRESSIVE`; no authoritative C data was changed.

The final vehicle is GREEN on seeds 1, 2, 3, 5, and 8.  It proves both
player-visible room Acts and the autonomous registered-VNum pulse dispatch.
Focused tests additionally prove all entry gates, literal/object predicate
branches, shared behead state and content transfer, instance-local level
growth, and the level-30 damroll branch.

## Go changes

- Refactored the existing player `DoBehead` mutation into shared
  `performBehead`, then used it for the NPC `brain_eater` call path.  The NPC
  path preserves C's first visible `corpse` resolution after the outer
  predicate, NPC room audience, proto 16/17 lifecycle, C list-prepend order,
  and unconditional continuation after the void `do_behead` call.
- Added instance-safe `AddDamrollBonus` and included the bonus in
  `MobInstance.GetDamroll`; level growth now uses `MobInstance.SetLevel` and
  does not mutate the shared prototype.
- Added `pkg/game/spec_brain_eater_test.go` and claimed seven manifest rows in
  `docs/fidelity/depth/spec-procs.tsv`.

## Gates and counts

Passed on this branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check`

Final depth report: 1,252 total; 1,205 proven/delegated; 13 blocked; 34
excluded; actionable completion 1,205/1,218 = 98.9%.

Rules applied: R1/R2/R3/R4/R5e, with the shared behead audit preserving the
R5b/R5c class boundary.

## Next action

Commit this slice, open `glm/spec-brain-eater`, and merge only after all GitHub
checks are green.  The next queue item is active `teleport_victim` in
`src/spec_procs3.c` / mob 14405; do not repick any claimed handoff row.
