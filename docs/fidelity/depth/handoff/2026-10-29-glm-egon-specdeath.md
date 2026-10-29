# Special death paths — caller-to-C-path inventory — 2026-10-29 (GLM)

The #2215 handoff flagged the special death paths outside its proven common
tail. This round traced each to its C call path, fixed the confirmed
divergences, and recorded the rest with evidence. Every C citation below was
re-read in `src/` this round.

## Fixed this round

### SPECIAL(werewolf) bite (`pkg/game/spec_procs3.go`)

C is `src/spec_procs3.c:427-453`: while fighting, a 10% howl and a 25% bite;
the bite is an **ordinary `damage()` call** (`:446`) — the same reward,
bookkeeping, and raw-kill tail as any combat hit — followed by
`GET_MOVE(vict) -= GET_LEVEL(mob)*1.5` for **any** victim (`:447-449`).

- The bite's lethal tail now routes through `TakeDamageWithDeath` →
  `FireMobDeathScript` + `World.HandleDeath` (the same seam as ordinary
  combat kills; the MS_DEATH arm mirrors die_with_killer,
  `src/fight.c:586-597`), replacing the engine's parallel default tail.
- The leg tear now applies to mobile victims too, with C's floating-point
  arithmetic: `int(move − level×1.5)` truncates after subtraction, so a
  level-11 werewolf takes 16.5 move (50 → 33, not the 34 an integer
  `level*3/2` gives).
- Kept: the Go room-equality gate on the bite. C has no such gate, but it is
  unreachable in stable states (FIGHTING is in-room) and removing it would
  apply damage cross-room, which C's `one_hit` refuses — recorded as a
  boundary, not a fix.

Unit proof: `pkg/game/spec_werewolf_bite_test.go` (float truncation, lethal
bite through the kill seam, red on revert — retained under
`oracle-runs/2026-10-29/glm-egon-specdeath/gates/`). The existing
`spec-proc-werewolf` scenario (nonlethal audiences) stays green.

### die_with_killer's experience drain for mobile victims

C's `die_with_killer` first line is `gain_exp(ch, -(GET_EXP(ch)/37))` for
**every** character (`src/fight.c:589`); `gain_exp`'s guard only skips PCs
outside level 1..30 (`src/limits.c:287-291`), so a dead mobile's experience
drains too — visible through the deferred-extraction `stat` window.
`World.HandleDeath` now drains mobile victims by 1/37 before the corpse, at
C's position in the tail (after the reward, which C computes first from the
undrained value, `src/fight.c:1636-1668`).

## Verified already faithful (no change)

- **CON croak** — `pkg/game/room_activity.go` ports `src/comm.c:748-754`
  byte-for-byte: the two lines, the HP pin to 1, and a direct `raw_kill`
  with no reward tail. Intentional raw kill.
- **Implementor `kill`** — `src/act.offensive.c:134-160` routes to
  `raw_kill`; the Go instakill keeps the no-POS_DEAD/AWAKE-preserving
  behavior with the citation in place.
- **Spike/stake, disintegrate, gate backfire, shoot, Lua `raw_kill`** — all
  call the raw-kill family (`RawKillCombatant`/`rawKill`/`combat.RawKill`),
  which matches C's direct `raw_kill` callers: cry, corpse/dust, deferred
  extraction, no reward tail. Lua's `command` action routes text through the
  command interpreter, not a damage API (re-verified against
  `pkg/game/world_bridge.go:456-485`).

## Dead code (R5e)

- **SPECIAL(field_object)** — C assigns it to objects 50 and 52
  (`src/spec_assign.c:528-529`), but object specials are only ever invoked
  from `command_interpreter` with `cmd` set (`src/interpreter.c:1427-1468`),
  and the body returns FALSE whenever `cmd` is set
  (`src/spec_procs3.c:462-463`). No pulse caller for object specials exists,
  so the damage/raw-kill body is unreachable in the shipped game; the live
  field-object behavior is the timer wear-off in `src/limits.c:652-685`.
  Go's mob-shaped `specFieldObject` is likewise unreachable (no mob vnum is
  assigned `field_object` in `MobSpecAssign`), so nothing was "fixed"; its
  divergences (player-only scan, mob-derived dice, HP<=-11 gate, hardcoded
  poison) are recorded here, not remediated.

## Newly found pre-existing divergences (recorded, evidence retained)

The diagnostic vehicle `spec-proc-werewolf-mob-victim` (AGGR24+AGGRESSIVE
full-HP arm makes the level-8 werewolf attack a level-1 trainee,
`src/mobact.c:326-338`; kept in the tree as a diagnostic, excluded from the
census) is red on origin/main and on this branch for three independent
pre-existing reasons — evidence under `oracle-runs/2026-10-29/glm-egon-specdeath/gates/`:

1. **Attitude-loot gold line** — C's `attitude_loot` runs
   `do_get(ch, "all corpse")` (`src/fight.c:1120`), and the werewolf's
   room sees "The werewolf gets a tiny pile of gold coins from the corpse
   of a guard trainee."; Go's mob-get path emits nothing. Belongs to the
   attitude-loot/get manifest (the `attitudeLootAfterKill` gate itself is
   correct — AGGR24/LOOTS/19650 are all checked).
2. **Claw damage arithmetic** — C's claw dealt 47 where Go dealt 50 in the
   same round (stat-verified 4953 vs 4950 of 5000): a pre-existing
   mob-barehand damage-formula divergence, possibly the same sbyte-wrap
   class found in the `set` hitroll/damroll round. Belongs to the damage
   formula manifest.
3. **Corpse long-description for the claw kill** — C's `make_corpse` maps
   TYPE_CLAW to "The hacked up, bloody corpse..." (`src/fight.c:334-339`);
   Go's `attackTypeToCorpseAttack` has the mapping, so the loss is in the
   attack-type value the engine propagates for mob swings. Belongs to the
   engine's mob-attack-type tracking.

## Genuine remaining proof gaps

- die_with_killer's dying-MOB_SPEC arm (`src/fight.c:590-595`: a spec-flagged
  victim's own special runs once more with FIGHTING set to the killer) is
  absent from both Go death paths — reachable for any spec-mob victim of any
  kill; needs its own round.
- The engine's default `TakeDamage` tail (`pkg/combat/fight_core.go:537-628`)
  remains reachable for any future caller that passes no death callback; its
  reinvented autogold ("get all gold corpse" and "You split no gold, you got
  none.") is not C's tail — no live caller reaches it after this round
  (ambush, werewolf, and the self-damage site all pass callbacks), but it
  should eventually be narrowed to unreachable or rewritten.
