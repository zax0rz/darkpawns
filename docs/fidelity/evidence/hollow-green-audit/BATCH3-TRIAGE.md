# Batch 3 triage — spec-procs cluster (DP-1511, 2026-11-08)

Continues the Part 2 worklist after the corrected screening
(120 counted suspects; worklist 108 rows). Verdicts below were each verified
against the census seed-1 dump, the row's claim, and — where a repair was
attempted — a live `--show-oracle` dev run.

## Repairs attempted: both exposed Go-vs-C divergences (stopped, not fixed, not pinned)

1. **`spec-proc-backstabber`** (`mob.backstabber-autonomous-dispatch`,
   `mob.backstabber-target-and-weapon-gates`). Root cause of the hollow:
   `quiet-mobs` strips authored `E` equip lines, so Daral (mob 9151) had no
   wielded weapon and `SPECIAL(backstabber)`'s `do_backstab` stopped silently
   at the `!GET_EQ(ch, WEAR_WIELD)` gate (`src/act.offensive.c:181-184`) —
   the gate's messages go to the descriptor-less mob, so both engines showed
   nothing. Repair attempted: `equip-object 9151 9101 1 16 91` (the zone's
   own authored weapon, zone 91 line 42). Result: C's transcript now shows
   `Daral tries to backstab you, but changes his mind.` and all pulse blocks
   match — but the **combat state diverges**: C's `score` says
   `You are fighting Daral.`, Go's says `You are standing.` First divergent
   block `score`, fingerprint `33d1f5ab687f48d5d5d6fa7e64b95c96d71bfa7beebc5f4558b5f17e4a16679e`.
   Retained red census: `~/Archives/darkpawns/oracle-runs/2026-11-08/glm-hollow-backstab`
   (NOT_CLEAN @9264dff0c, throwaway commit only, not for merge).

2. **`spec-proc-troll-combat`** (retained evidence completed per review).
   Re-run through `scripts/census.sh start --name glm-hollow-diag
   --scenarios spec-proc-troll-combat,wear-race-hate-pulse` at throwaway
   commit 90c94e87d: verdict NOT_CLEAN, failed=2 —
   `~/Archives/darkpawns/oracle-runs/2026-11-08/glm-hollow-diag`.
   Go's actor block is *shorter* (257 lines vs C's 316) while Go's peer
   block is *longer* (259 vs 234): in Go the peer keeps seeing the actor
   fight ("TrollCombatActor hits Targ hard."), but the actor stops receiving
   its own lines. Claude's review reads this as an output-delivery or capture
   gap, not a combat or RNG divergence. Filed as DP-1515; out of scope here.

3. **New divergence, retained in the same census**: `wear-race-hate-pulse`
   with a `stat racehate` probe — C prints
   `Racial Hatreds: Giant Ogre None None None`, Go prints
   `None None None None None`. Go's stat prints that line as a constant
   (`pkg/session/wiz_stats.go:358`), so this proves a stat stub, not a
   missing affect. Filed as DP-1514 (with the `PLR: CSH` vs `NOBITS` line). First divergent block `stat racehate`,
   fingerprint `9d7ec6114d194f351de15f72b77400ddddb017cd01075566d4fba83258cbf742`.

## Claim withdrawn

The earlier **"Go reads unindexed world files"** claim is withdrawn. The
unified diff for `stat giant` shows `(no normalized divergence)` — C and Go
both print `Nothing around by that name.`; Go never found mob 8552. The
misreading was mine (I read diff headers as divergence). Go's world load
follows the index files exactly as C does. The wear row's merged notes say
otherwise and are corrected on main by the follow-up commit.

## Triage verdicts (spec-procs worklist rows)

FP — the claimed bytes are in the dump (drop):
`mob.butler-container-open-close`, `mob.puff-emotes`,
`mob.dragon-breath-combat-transcript`, `mob.fighter-combat-action`,
`mob.guild-guard-move-block`, `mob.werewolf-combat-dispatch`,
`mob.medusa-look-target-gate` (the not-found fallthrough is the claim),
`mob.rescuer-7909-no-ally-fallthrough` (silence is the claim),
`mob.take-to-jail-invisible-victim` (silence is the claim),
`mob.spec-unknown-word` (movement fallthrough shown),
`mob.remorter-buy-list-tell`, `room.pray-return-false`,
`obj.npc-shop-keeper-values-and-sells-for-the-pet`,
`obj.npc-special-decline-obeys-orders`.

Genuine — the spec's output is absent from every block (unrepaired; the two
repair attempts above both hit Go combat-state divergences, and these are
the same attack-the-player class):
`mob.thief-periodic-steal`, `mob.thief-awake-audience` (pulse-40 windows
empty; the victim's coins unchanged across `score` blocks),
`mob.spec-recruiter-cast-will`, `mob.spec-recruiter-hit-kill` (only the
sign-up tell appears), `room.elements-galeru-alive-*` (Galeru never
responds), `mob.werewolf-bite-mob-victim` (claw bytes only, no bite).

Repaired earlier: `mob.troll-idle-room-audience`, `mob.troll-pulse-dispatch`
(#2395, census CLEAN). Stopped: `mob.troll-combat-dispatch` (above).
Downgraded: `wear.player-race-hate-mobile-dispatch` (#2396).

## Diagnosis worth routing (Go bugs, for Linear)

- DP-1513: a failed backstab on an immortal does not set fighting in Go
  (C's `damage()` does, `src/fight.c:1445`) — surfaced by the backstabber repair.
- DP-1514: Go's `stat` hardcodes the Racial Hatreds line and drops a PLR bit.
- DP-1515: in a long fight Go stops delivering the actor's own lines while
  the peer still sees the fight (troll-combat).
