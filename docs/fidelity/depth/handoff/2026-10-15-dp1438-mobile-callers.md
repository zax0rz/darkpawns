# DP-1438 part 2 — the mobile hit() callers

Date: 2026-10-15 · Branch: `glm/dp-1438-mobile-callers` (off the batch tip
`claude/batch-2026-10-15`, which carries #1982, #1969, #1984) · Rules: R1,
R3b, R5c, R5e, R5g, R5h

## What converted (commit per family, C read fresh this session)

Every mobile-side C `hit()` caller that still enrolled first now runs the
deferred initial attack (`PerformDeferredInitialAttack`, from part 1):

| Caller | Go site | C cite |
|---|---|---|
| mobact aggressive (both arms) | `mobact.go` via `mobFirstHit` | `mobact.c:223, 227` |
| race hate | same | `mobact.c:254` |
| memory | same | `mobact.c:279` |
| helper | same | `mobact.c:299` |
| `mobAssist` (charmed-master arm) | same | `act.offensive.c:95` |
| AGGR24 player arm | same | `mobact.c:320` |
| AGGR24 weaker-mobile arm | same | `mobact.c:334` (no peaceful check at selection — damage() refuses) |
| `mobHit` seam (7 spec_procs2 callers, shopkeeper-vs-dog greet, mob prey) | `spec_procs.go` | per-caller; e.g. `mobprog.c:85` |
| hunter arrival + mob `kill` action | `graph.go` | `graph.c:312, 340-345`; `interpreter.c:525` + `act.offensive.c:126` |
| no_get hand strike | `spec_procs_missing.go` | `new_cmds2.c:561-569` |
| elements guardian charmed-player hit | `spec_procs3.go` | `spec_procs3.c:1281` |
| cityguard bribe refusal | `mobprogs.go` | `mobprog.c:220-226` |

`mobFirstHit` (world_combat.go) is the world seam; engines without the
deferred entry keep the historical shape for focused tests. DeepSeek's acts
and selection orders are untouched.

### Leave-alones (audited against C this session)

- `do_rescue`'s success arm — pure `set_fighting`, `act.offensive.c:575-576`.
- The backstabber special — draws in the caller, gate then post-gate
  enrolment (`fight.c:1400-1408`, cited in-file).
- The spell-damage enrolment seam (`damage_spells.go`) enrols after its gate
  and before modifiers, matching C's `fight.c:1408` position; no draw or byte
  impact. Deeper refactoring stays with the spell-damage owner.
- `StartRoomCombat`'s eager body survives for engines without the deferred
  entry (its cityguard-bribe caller converted).

## Acceptance evidence

- **Room 8162** (the blocked row `mobact.peaceful-first-hit-enrolment`, now
  `oracle-green`): the withdrawn DP-1371 fixture is reborn as
  `mobact-aggr24-weaker-mobile-grin` — the lord grins, the swing draws
  `number(1,20)`, and the courier lives. R5h: with `pkg/` reverted to the
  batch tip the scenario diverges (the grin and the living courier vanish).
- **Corpus-2 DRAW rows**: all five `spec-proc-no-get*` scenarios and
  `sharpen-fighting-depth` are draw-CLEAN at seed 1. The five shop rows'
  first mismatch moved past the hit-roll class — pre-fix it was
  `c=Number(1,20) go=Number(0,18)` at stream 59247 (exactly this class:
  C's mobile hit() vs Go's wander-door draw), post-fix the streams align
  ~3,200 draws further to the next, different class (below).
- **The parked soaks stay parked.** On this batch tip their first mismatch is
  NOT this class — identical pre-fix and post-fix (verified on
  `living-world-haven-square`): the fight draws align, then at a mid-fight
  round C draws the position-degraded mob's swing (`Dice(1,1)`, the
  "trouble getting his body to fight" variant, lib/misc/messages:399) where
  Go draws the player's (`Number(0,20)`). At long pulses their bytes also
  fail from DP-1439 (Go loses ALL sessions' output for the whole long-pulse
  step — verified on all three observers), which is session/transport stop
  tier. Shortening cannot help: at `pulse 100` the bytes are clean but the
  same setup-phase draw divergence remains.

## Follow-up findings (new, not this class)

1. **God-vs-mob round order**: in long setup/settle fights (the shop rows at
   stream 62467+, the soaks), C draws the 1d1-dice mob's swing where Go draws
   the level-60 player's bare-hand `number(0,20)` — a combat round-order
   divergence that the enrolment-point change does not move (pre/post-fix
   identical on the soaks). Affects `shop-live-inventory-control`,
   `shop-stack-list-live`, `show-shops-detail` (62467) and
   `shop-live-inventory-filters`/`-gate` (62382). Byte verdicts stay clean;
   these remain DRAW rows.
2. **carve-depth warmup draws**: C draws the `kill fish`/`hit fish` swing
   (`Number(1,20)` + `Dice(1,6)`) during warmup where Go draws an extra
   `Number(0,3)` — Go's warmup ends the fight path without the swing draws
   (stream 4098). Still a DRAW row; bytes clean.
3. **`PerformUnenrolledInitialAttack`'s refusal arm** still enrols the
   attacker on a refused swing (carried from part 1's table); fold with the
   next engine pass and re-run the backstab corpus.

## Proof

Real-engine, seed-pinned-miss unit tests per family (mobact arms incl. the
8162 weaker-mobile shape, the hunter strike, the no_get strike, the guardian
hit, the bribe strike — each asserting the draw consumed, peaceful refusal
enrolling nobody, open room enrolling the pair), each verified failing on its
reverted production shape. Targeted censuses recorded under
`~/Archives/darkpawns/oracle-runs/2026-10-15/dp-1438-*`.
