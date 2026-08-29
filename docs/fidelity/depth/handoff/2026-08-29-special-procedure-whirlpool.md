# 2026-08-29 — `whirlpool` depth slice

## Frontier and queue

- Started from clean `main`, pulled, ran `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- The pre-slice frontier was 973 total cases; 946 proven/delegated, 6 blocked,
  and 21 excluded. This slice records six blocked cases, yielding 979 total,
  946 proven/delegated, 12 blocked, and 21 excluded; actionable completion is
  946/958 (98.7%).
- The next active, unclaimed source-order special is `couch` at
  `src/spec_procs2.c:287`, registered for mob vnum 12204 at
  `src/spec_assign.c:343`.

## C call path and branch census

- `SPECIAL(whirlpool)` is defined at `src/spec_procs2.c:244-275` and assigned
  to vnum 12200 at `src/spec_assign.c:342`.
- Autonomous dispatch is `src/mobact.c:68-93`: the awake, non-fighting
  `MOB_SPEC` mobile calls the procedure as `(ch, ch, 0, "")`. Player command
  dispatch is `src/interpreter.c:1407-1456`, but the procedure has no command
  gate and therefore can intercept any command in the room.
- The procedure skips mini-mud, null callers, `PRF_NOHASSLE` players, the mob
  itself, and NPCs; for each remaining victim it rejection-samples
  `number(real_room(4600), real_room(4699))` until the room is not private,
  godroom, death, or nomob, transfers the victim, sends two victim-only lines,
  calls `look_at_room`, and returns whether a victim moved.

## Vehicle evidence and disposition

- The required disposable `spawn-mob` vehicle was attempted repeatedly with
  vnum 12200, both in the authored zone-122 reset and an injected zone-183
  reset, with the peer walking through the ordinary start-room path to room
  8105. The same zone/room reset with known vnum 18305 materialized and listed
  the cleaner, proving the fixture harness is functioning.
- `vnum mob whirlpool` sees the C prototype `[12200] a magical whirlpool`, but
  no C instance is present after either reset vehicle. The authored reset also
  remains absent in the direct room probe. No player-visible whirlpool branch
  can therefore be paired across engines.
- A command-time `load mob 12200` check was not used as proof: C's command
  authority/vehicle behavior differed from the Go helper and does not satisfy
  the required registered-mob reset vehicle. No Go fix is inferred from that
  unpaired path.
- `whirlpool` is marked blocked after two honest vehicle attempts in the
  manifest. The six rows are `mob.whirlpool-autonomous-entry`,
  `mob.whirlpool-nohassle-gate`, `mob.whirlpool-random-destination`,
  `mob.whirlpool-victim-output`, `mob.whirlpool-destination-look`, and
  `mob.whirlpool-state-transition`.

## Verification and integration

- `make fidelity-depth` passes after the six blocked rows are added.
- No `src/` or `darkpawns-c-oracle/` file was edited, and no Go behavior was
  changed because no confirmed divergence was reachable.
- The disposable scenario is
  `cmd/dp-oracle-diff/scenarios/spec-proc-whirlpool.txt`.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

This disposition applies R1/R2 (no player-facing or command-surface claim
without a live vehicle), R3 (no random-draw claim without a paired C draw),
R4 (no invented output), and R5/R5e (actual C call path and vehicle evidence).

## Next queue item

Continue the active special-procedure inventory with `couch` in source and
registration order. Do not repick `whirlpool` or earlier claimed procedures.
