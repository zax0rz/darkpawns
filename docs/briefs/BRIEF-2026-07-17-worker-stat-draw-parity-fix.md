# BRIEF (codex / glm-5.2 / k3 — whoever's free) — newbie stat-roll draw parity + observation co-location

**Owner:** any available worker. **Gate:** Claude runs the oracle red→green (`hunger-thirst`, `guild-practice`, `character-view`-partial, `observation`) — workers have no `DP_ORACLE_BIN`. **Branch off `main`, one PR.**
Diagnosis source (measured, by mimo): `docs/briefs/REPORT-2026-07-17-mimo-newbie-stat-draw-parity.md`. Root cause verified by Claude in-tree. This closes the newbie-stat divergence that reddens `hunger-thirst` (HP 21 vs 24) + `guild-practice` (practices 4 vs 3), plus the `observation` co-location.

## Root cause (measured, DP_SEED=1)
C consumes **~8904 more PRNG draws** than Go during boot zone resets, so the shared stream is desynced by the time `roll_real_abils` fires → Go rolls different CON/WIS. At the stat roll: C=59394 draws (con=12,wis=12), Go=50492 draws (con=13,wis=13). The newbie-stat algorithm, tables, and assignment are already byte-identical to C — this is purely a **boot-reset draw-count** gap. Two fixes close the bulk of it:

## Fix A — copy the 2 missing zones into the Go world (the dominant ~8900 draws)
The C oracle world has zones **150** and **165** that Go's `lib/world/` is missing entirely → 179 fewer M/O/G/E reset commands in Go → ~8900 fewer HP-dice / gold-variance / percent_load / init_rare draws. Copy the full zone definition for each from the C oracle tree (`~/.openclaw/workspace/darkpawns-c-oracle/lib/world/`) into `lib/world/`, byte-for-byte:
- `zon/150.zon`, `wld/150.wld`  (150 has **no** mob/obj file — rooms only)
- `zon/165.zon`, `wld/165.wld`, `mob/165.mob`, `obj/165.obj`
Confirm the Go world loader picks them up (index/renumber). **Verify the copied files parse and don't collide** with existing vnum ranges. This is the faithful fix — the Go port must carry the same world as C.

## Fix B — remove Go's spawn-time mob stat-boost double-draw (6 draws per level>15 mob)
`pkg/game/mob.go:113-129` (in `NewMob`, spawn time) rolls the 6 "random stat boosts for mobs above level 15" — but C (`db.c` `read_mobile`, ~1053-1062) does these boosts **at parse time only**, and Go **already does them at parse time too** (in the mob parser). So Go double-applies them, over-drawing 6 per high-level mob at spawn. **Remove the spawn-time boost block in `mob.go:113-129`** (the `if proto.Level > 15 { … 6× dprng.Number(0, statmod) … }`), keeping the values from the (already-boosted) prototype. Verify the parser IS the single boost site and matches C's draw order/position; do not remove the boosts from both places.

## Fix C — observation co-location (scenario file, trivial)
`cmd/dp-oracle-diff/scenarios/observation.txt`: the `[setup:port:target]` block is missing the `look` and `recall` lines that `[setup:oracle:target]` has, so the port's target peer never recalls to the temple and ends up in a different room than the actor. **Add `look` then `recall` after the `1` (enter-game) line in `[setup:port:target]`**, mirroring `[setup:oracle:target]`. (Recall itself works — this is a scenario-block asymmetry, not a game bug.)

## Caveats (why the gate is decisive)
- Mimo also noted a THIRD, smaller contributor: C's random-room-selection uses a do-while retry loop that can draw more than Go's 5-attempt cap. Fixes A+B may not fully zero the gap if that residual matters. If `hunger-thirst`/`guild-practice` are still red after A+B, report the residual `drawsBefore` delta (do NOT add a compensating draw-burn) — Claude will pin the remainder.
- Fix A changes Go's boot world (2 new zones) — this is the highest-risk change. It must not regress any currently-green scenario.

## Acceptance (Claude-gated, from a PR-branch worktree, corrected `result:`-line check)
1. `hunger-thirst`, `guild-practice` → `no normalized divergence` (the stat-derived values now match).
2. `observation` → `no normalized divergence`.
3. `character-view` — the score block stays green; its remaining `time` divergence is a SEPARATE bug (not in scope here).
4. Full committed sweep stays green — especially anything touching zones 150/165's vnum ranges or high-level mobs.
5. `DP_SEED`/`DP_CLOCK` unset ⇒ no behavioral change beyond the (now-present) zones + single mob boost.
