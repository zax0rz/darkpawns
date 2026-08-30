# Depth-fidelity handoff: `troll`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `7d570d50c`

## Scope and source audit

This slice covered `SPECIAL(troll)` in `src/spec_procs3.c:347-370`. The C body has
these player-visible and state branches:

1. Nonempty command, sleeping/non-awake, and non-positive-HP gates return false.
2. An injured, non-fighting troll draws `number(0, 20)`; on zero it adds
   `GET_LEVEL * 2`, caps at `GET_MAX_HIT`, emits the exact room act, and returns
   handled.
3. A fighting troll draws `number(0, 10)`; on zero it applies the same capped
   regeneration and room act, and returns handled.
4. A full-HP idle troll and a failed draw fall through false. A fighting troll
   uses only the fighting draw arm.

The regeneration helper is the immediately preceding C `npc_regen` body at
`src/spec_procs3.c:340-345`. The first active registration is
`ASSIGNMOB(10029, troll)` in `src/spec_assign.c:318`; the other registrations
are vnums 14100, 14311, 14312, 19900, and 19901. The canonical fixture uses
mob 10029 from `lib/world/mob/100.mob` in room 10049, strips its authored Lua
script, and sets the C `SPEC` bit explicitly because the assignment table is
what supplies the callable special. C mobile dispatch is
`src/mobact.c:68-93`; the combat hook is `src/fight.c:1898-2032`, after the
ordinary NPC attack loop. Go's corresponding hook is wired by
`pkg/session/manager.go:608-625` and called from `pkg/combat/engine.go:582`.

## Vehicles and proof

- `cmd/dp-oracle-diff/scenarios/spec-proc-troll.txt` aligns both players away
  from the authored `AGGR_NEUTRAL` behavior, uses room 10049 with no exits,
  and runs the registered mob through a padded 40-second mobile pulse window.
  Seeds 1, 2, 3, 5, and 8 all normalized green; seeds 5 and 8 visibly emit the
  exact room-wide glow for both observers.
- `cmd/dp-oracle-diff/scenarios/spec-proc-troll-combat.txt` explicitly starts
  Targ's fight and runs ten bounded violence rounds through the native combat
  hook. Seed 1 normalized green. Seeds 2 and 3 also normalized green during
  investigation; seeds 5 and 8 hit unrelated existing combat transcript
  divergence (stun/death timing) before they are clean special-procedure
  evidence, so the manifest records the honest seed-1 vehicle only.
- `pkg/game/spec_troll_test.go` proves entry gates, the idle `0..20` draw arm,
  capped level-scaled regeneration, exact room audience, the fighting `0..10`
  draw arm, and the fighting room audience. `trollNumber` in
  `pkg/game/spec_procs3.go` is only a test seam; no production behavior change
  was confirmed.
- Six manifest rows were added to `docs/fidelity/depth/spec-procs.tsv` for the
  entry gates, idle state/audience, mobile dispatch, fighting state, and combat
  dispatch.

## Fidelity result

This was a pure-coverage round: the existing Go `specTroll` path already
matched the confirmed C behavior, so no `src/` or `darkpawns-c-oracle/` file was
edited and no divergence fix was invented. The discarded long-window probe
was contaminated by unrelated point-update/ambient activity and is not proof.

The evidence satisfies R1/R2/R3/R4/R5e: player bytes, commandless dispatch,
deterministic seeded draws, no invented surface, and verified C call paths are
covered. R5c is unchanged because this slice did not reveal a broader class
failure.

## Verification

All required gates pass on `glm/spec-troll`: `make fidelity-depth` reports
1,276 total cases, 1,227 proven/delegated, 14 blocked, and 35 excluded
(1,227/1,241 actionable, 98.9%); `go test ./pkg/game -run 'TestSpecTroll'
-count=1`; `go build ./...`; `go vet ./...`; `go test ./...`;
`golangci-lint run ./...` (`0 issues`); `gofumpt -l .` (no output); and
`git diff --check` all pass.

## Next queue item

Continue special-procedure source/registration order with `quan_lo`, first
registered as mob vnum 19405 at `src/spec_procs3.c:372` and
`src/spec_assign.c:444`.
