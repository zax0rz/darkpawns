# DP-1489 trace — group scroll member order, and `damage()`'s `stop_follower`

Linear: DP-1489 · Rules: R1, R3b, R5b/R5c, R5e, R5g, R5h, R5j
Baseline: `origin/main` `b7fdd6ef5` (same commit as the rehearsal baseline
`~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107`).
Run: `~/Archives/darkpawns/oracle-runs/2026-11-07/mimo-dp1489-trace`
(`scripts/census.sh start --name mimo-dp1489-trace --scenarios spellmatrix-scroll-olc-02-depth`),
verdict `oracle-regression: … expected=1 … verdict=CLEAN` — the pinned shape
fired again on today's baseline. All C cites below were read from `src/` in
this session (R5g). Transcript quotes are from the run's retained capture
`initial-logs/attempt.Hot1JA/spellmatrix-scroll-olc-02-depth.capture2/blocks.json`.

## Question

Gemini read the divergence at `recite book ItemPeer [actor]` as two causes.
Which are real on today's baseline, and is anything else going on?

**Both are real.** The fall-through flag raised in the brief is real in the
source but **unreachable** (see §3).

## 1. Cause A is REAL — `mag_groups` member order (R1, R3b)

### What the transcript shows

`spellmatrix-scroll-olc-02-depth` probe block 1 recites the OLC scroll with
spell values `20, 49, 56, 57` = `SPELL_DETECT_MAGIC`, `SPELL_GROUP_RECALL`
(`src/spells.h:84,113`), `SPELL_SOBRIETY`, `SPELL_GROUP_INVIS`
(`src/spells.h:121-122`). Both group spells walk the group through
`mag_groups`. The C [actor] (ItemMage) block reads, line by line:

```
You recite the Book of Common Prayer which dissolves.
ItemPeer disappears.                 <- group recall, follower first
At the Temple Altar                  <- actor's own recall landing (look)
   You stand before the temple altar. …
 ItemPeer the Warrior is standing here.
ItemPeer slowly fades out of existence.   <- group invis, follower first
You vanish.                              <- group invis, self last
```

Go matches the recall half and **swaps the invis half**:

```
…
 ItemPeer the Warrior is standing here.
You vanish.
ItemPeer slowly fades out of existence.
```

Unified diff (the run's only shape on this block):

```
--- [recite book ItemPeer [actor]] c-oracle
+++ [recite book ItemPeer [actor]] go-port
@@ -7,5 +7,5 @@
  ItemPeer the Warrior is standing here.
-ItemPeer slowly fades out of existence.
 You vanish.
+ItemPeer slowly fades out of existence.
```

### Why the recall half matches and the invis half does not

C's walk is stable across spells; Go's walk is not.

- **C order** — `mag_groups` (`src/magic.c:1466-1501`): `k = ch->master` or
  `ch` (`:1477-1478`); then every follower of `k` in `k->followers` list
  order, skipping `ch` (`:1482-1484`), other rooms (`:1485-1486`) and
  non-`AFF_GROUP` (`:1487-1488`); then `k` itself if in the room and grouped
  (`:1492-1494`); then `ch` last if `k != ch` (`:1496-1498`).
- **List order** — `add_follower` and `add_follower_quiet` both **prepend**:
  `k->next = leader->followers; leader->followers = k;`
  (`src/utils.c:474-475`, `src/utils.c:491-492`). The followers list is
  newest-first.
- **Go order** — `MagGroups` (`pkg/spells/affect_spells.go:873-925`) iterates
  `world.GetAllCharsInRoom(roomVNum)`, a room-occupancy order (newest entry
  first: `char_to_room` prepends at `src/handler.c:537-538`, mirrored by
  `RoomEntrySequence` desc in `pkg/game/world_movement.go:101-116`), and
  includes the caster wherever they sit in that order rather than last.

In probe block 1 the two orders coincide at first — the room order happens to
put ItemPeer (the newest occupant of the grouped set) first, so `SpellGroupRecall`
emits `ItemPeer disappears.` before the actor's own recall, matching C. But
`spell_recall` re-enters both bodies (`char_from_room`/`char_to_room`), so by
the time `SpellGroupInvis` walks the room again the **actor is now the newest
occupant**: Go processes the actor first (`You vanish.`) then ItemPeer, while C
still walks the follower list (ItemPeer first). That is exactly the observed
swap. The room-ordered walk is wrong even without the mutation (a caster that
entered last is processed first), and the mid-walk mutation makes it visible
in a two-member group.

Note also that C's walk is one level deep: only `k`'s **direct** followers are
processed (a follower-of-follower of `k` is never visited when `ch != k`).
Go's room walk visits every grouped occupant. Not reachable in this scenario
(only one follower level), recorded in `member-order.tsv`.

## 2. Cause B is REAL — `stop_follower` missing on the spell-damage path (R1)

### What the transcript shows

Probe block 2 recites spell values `61, 62, 67` = `SPELL_MINDPOKE`,
`SPELL_MINDBLAST`, `SPELL_VITALITY` (`src/spells.h:126-127,132`). The first
two are damaging: `mag_damage` → `damage()`. C's [actor] block:

```
You recite the Book of Common Prayer which dissolves.
ItemPeer stops following you.
ItemPeer screams in agony as you prod her mind.
ItemPeer screams in agony as you blast her mind.
```

Go's [actor] block omits the `stop_follower` line entirely at the hit and
instead emits it one pulse later, when melee runs (probe `~dpclock pulse 20`
blocks carry `ItemPeer stops following you.` on Go only; C is silent there).
The same displacement shows in [builder] (`ItemPeer stops following ItemMage.`)
and [peer] (`You stop following someone.`) — this is the pinned
`~dpclock pulse 20 [actor|builder|peer]` shape.

### The call path (R5e)

- **C** — `damage()` (`src/fight.c:1314`) runs
  `if (victim->master == ch) stop_follower(victim);` at
  `src/fight.c:1457-1458` on **every** damage() call, after the fighting
  setup (`:1400-1455`) and before the inline hide fade (`:1460-1464`),
  whatever the attack type. `stop_follower` emits the "You stop following $N."
  act trio (`src/utils.c:417-421`), unlinks from the leader's list, and clears
  `AFF_CHARM|AFF_GROUP` (`src/utils.c:436-442`).
- **Go melee** already has it — `processCombatPair` (`pkg/combat/engine.go:1048-1052`)
  and `takeDamageFrom` (`pkg/combat/fight_core.go:520-524`) both call
  `cbStopFollowerOfMaster` at the C position, then `fadeHiddenAttacker`.
  That is what delays the Go bytes to `~dpclock pulse 20`: the spell hit never
  breaks the relation, the next violence round does.
- **Go spell damage does not** — `inflictDamage` (`pkg/spells/damage_spells.go:357-460`),
  the `mag_damage` twin the mind poke/blast hit takes, enrolls fighting and
  applies modifiers but never calls the follower break or the hide fade. The
  brief's note "#2325 (`damage()`'s default arm) has merged" is true but
  orthogonal: #2325 ported the surviving-victim default arm
  (`src/fight.c:1587-1614`, `ApplySurvivingVictimDefaultArm`) — the
  pain/scream/bleed/wimpy block — not the `stop_follower` at `:1457-1458`.

### The same class, swept (R5c)

Grep of every Go `damage()` twin (entry points called from live code):

| Go twin | call sites | `stop_follower` @ C pos | hide fade @ C pos |
|---|---|---|---|
| `combat.processCombatPair` (`engine.go:1048-1052`) | melee round | yes | yes |
| `combat.takeDamageFrom` (`fight_core.go:520-524`) | `TakeDamage*`, `TakeDamageAfterGate`, `DoSpellDamage`, `Do*Damage` skill seams | yes | yes |
| `spells.inflictDamage` (`damage_spells.go:357`) | `MagDamage` (every damaging spell: mind poke/blast, acid blast, …) | **no** | **no** |
| `game.ApplySkillDamageWithMessage` (`damage_stubs.go:171`) | skill command tail (`skill_commands.go:1412,1579`) | **no** | **no** |
| `game.mobSkillDamageAfterGate` (`spec_fighter.go:22`) | fighter/paladin/soul-leech/psiblast mob specials | **no** | **no** |
| `game.selfDamage` (`room_activity.go:256`) | room damage, tick damage | n/a (ch==victim; `victim->master == ch` impossible) | yes |
| `game.doDamage` (`damage_stubs.go:370`) | **no callers** — dead twin | (untouched; listed for Claude) | |

All three live gaps get the `src/fight.c:1457-1464` block (follower break
then fade) at the same position: after the fighting setup, before
`ApplyDamageModifiers` (`src/fight.c:1465` onward).

## 3. `perform_mag_groups` fall-through — real in source, unreachable (R5e)

`perform_mag_groups` (`src/magic.c:1431-1450`) has `case SPELL_MASS_DOMINATE:
spell_charm(...)` with **no `break`** (`:1438-1439`), falling through into
`case SPELL_GROUP_INVIS: mag_affects(..., SPELL_INVISIBLE, ...)` (`:1440-1441`).

It cannot fire:

- `mag_groups` is dispatched only under `IS_SET(SINFO.routines, MAG_GROUPS)`
  (`src/spell_parser.c:488-489`).
- `SPELL_MASS_DOMINATE` is registered `TAR_IGNORE … MAG_AREAS` and nothing
  else (`src/spell_parser.c:1480-1481`); its live body is the
  `spell_charm` arm of `mag_areas` (`src/magic.c:1617-1618`).
- Go mirrors both: `RoutineAreas` registration
  (`pkg/spells/affect_spells.go:1717`), `RoutineGroups`-gated dispatch
  (`pkg/spells/call_magic.go:161-163`), and the `mag_areas` charm arm
  (`pkg/spells/affect_spells.go:1054-1058`).

The Go `case SpellMassDominate` in `MagGroups` (`affect_spells.go:921-923`) is
likewise unreachable. Per the brief ("port the fall-through") the case is
ported so the function twin matches the C source byte for byte, with a unit
test and revert proof — but it is a **dead-code twin, not a player-observable
fix**. Flagged in the PR's Decisions section rather than claimed as a
divergence fix.

## 4. Verdict

| claim | verdict | live bytes |
|---|---|---|
| A: `mag_groups` walk order | **real** | recite 1 `[actor]` — invis line swap |
| B: `stop_follower` in `damage()` | **real** | recite 2 `[actor]/[builder]/[peer]` + `~dpclock pulse 20` [×3] |
| fall-through `MASS_DOMINATE`→`GROUP_INVIS` | source-real, **unreachable** | none |

Nothing else diverges in this scenario: 236 blocks compared, 7 divergent
fingerprints (the pin set), all accounted for by A and B. #2325 removed
nothing of B (different arm of `damage()`); B's blocks still pin-match, so the
pin stays until this fix lands and is then removed (STALE policy).
