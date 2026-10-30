---
tags: [audit, fidelity, skills, spells, combat, corpse, open]
last_updated: 2026-10-30
author: Cline (deepseek) with The Architect
---
# Skills/Spells Attack-Type Audit — the class behind "color weirdness"

Companion to [`ansi-color-audit.md`](ansi-color-audit.md). That audit fixed two
colour bytes in combat; this one answers the question it left open — *is a
single bug a symptom of a class?* For skills and spells, yes.

Fireball's own text carries no colour codes, so Egon's report points at the
shared delivery path every skill and spell uses. Tracing that path for **every**
skill turned up a second, unrelated class bug in the same surface: the attack
type the port hands to `damage()`.

## 1. One C argument, three consumers

C passes exactly one value to `damage()` — the skill's own constant
(`SKILL_BASH` for `do_bash`, `spellnum` for a damage spell) — and everything
downstream reads that one value:

| consumer | site | effect |
|---|---|---|
| message dispatch | `src/fight.c:1534-1545` (`IS_WEAPON`) | `skill_message` set vs the weapon ladder |
| corpse wording | `die_with_killer` → `raw_kill` → `make_corpse` (`src/fight.c:283-370`) | the corpse's long description |
| dust instead of corpse | `make_corpse`'s `SPELL_DISINTEGRATE` check | `make_dust` |

Every `damage()` call site in `src/*.c` was extracted and classified: 14 in
`act.offensive.c`, 20 in `new_cmds.c`, 4 in `limits.c`, 3 in `fight.c`'s
`one_hit`, plus `magic.c`, `spells.c`, `spec_procs2.c`, `spec_procs3.c`,
`comm.c` and `handler.c`.

## 2. make_corpse's groups

`src/fight.c:283-370` sorts the attack type into one of these descriptions:

| group | members |
|---|---|
| charred, still smoking | `SPELL_BURNING_HANDS`, `SPELL_FIREBALL`, `SPELL_HELLFIRE`, `SPELL_FLAMESTRIKE` |
| frozen, thawing | `SPELL_CHILL_TOUCH` |
| blasted, in pieces | `SPELL_COLOR_SPRAY`, `SPELL_METEOR_SWARM`, `SPELL_DISRUPT`, `TYPE_BLAST` |
| dried-up husk | `SPELL_ENERGY_DRAIN`, `SPELL_SOUL_LEECH` |
| shocked looking | `SPELL_LIGHTNING_BOLT`, `SPELL_CALL_LIGHTNING`, `SPELL_SHOCKING_GRASP` |
| brains exploded | `SPELL_PSIBLAST` |
| **bruised, battered** | `SKILL_BASH`, `SKILL_KICK`, `SKILL_PUNCH`, `SKILL_DRAGON_KICK`, `SKILL_TIGER_PUNCH`, `SKILL_HEADBUTT`, `SKILL_SMACKHEADS`, `SKILL_SLUG`, `SKILL_SERPENT_KICK`, `TYPE_BLUDGEON`, `TYPE_POUND`, `TYPE_PUNCH`, `TYPE_WHIP` |
| **hacked up, bloody** | `SKILL_BITE`, `TYPE_BITE`, `TYPE_CLAW`, `TYPE_SLASH`, `SKILL_BACKSTAB`, `SKILL_CIRCLE` |
| guts spilled | `SKILL_DISEMBOWEL` |
| neck snapped in two | `SKILL_NECKBREAK` |
| crushed, barely recognizable | `TYPE_CRUSH`, `TYPE_MAUL`, `TYPE_THRASH` |
| well-ventilated | `TYPE_PIERCE`, `TYPE_STAB` |
| bloated, waterlogged | `SPELL_DROWNING` |
| frozen in stone | `SPELL_PETRIFY` |
| plain "The corpse of X is lying here." | `TYPE_UNDEFINED` **and the default arm** |

The groups deliberately mix `SKILL_*` and `TYPE_*` members, which is why some
substitutions are invisible and others are not.

## 3. What the port did

`pkg/game` carried the attack type in two channels:

- `SkillResult.SkillMsgType` — the `lib/misc/messages` key. C derives it from
  the same `damage()` argument, so it *is* C's attack type.
- `SkillResult.DamageSkill` — a **skill name string**, re-converted to a number
  by `skillToAttackType()` in `pkg/game/damage_stubs.go`.

`skillToAttackType` mapped eight skills to `TypeBludgeon` (305) instead of their
own `SKILL_*` value, and everything unrecognised — including the **empty string**
left by seven results that never set `DamageSkill` — to `TypeSlash` (303).

Measured with a probe over the real skill entry points:

| skill | C's value | Go before | C corpse | Go corpse before |
|---|---|---|---|---|
| bash | 132 | 303 (name empty) | bruised, battered | **hacked up, bloody** |
| kick | 134 | 303 (name empty) | bruised, battered | **hacked up, bloody** |
| headbutt | 141 | 303 (name empty) | bruised, battered | **hacked up, bloody** |
| trip | 144 | 303 (name empty) | plain | **hacked up, bloody** |
| bearhug | 142 | 303 (name empty) | plain | **hacked up, bloody** |
| strike | 155 | 303 (unknown name) | plain | **hacked up, bloody** |
| backstab | 131 | 303 (name empty) | hacked up, bloody | hacked up, bloody *(coincidence)* |
| circle, charge, bite, disembowel, neckbreak, groinrip, cutthroat | own | own | correct | correct |
| slug, smackheads, serpent_kick, tiger_punch, punch | own | 305 | bruised, battered | bruised, battered *(same group)* |
| ambush | 191 | 191 | plain | plain |

Six skills printed the wrong corpse wording on a kill; eight more escaped only
by sharing a group with `TYPE_BLUDGEON`. The messages were right throughout,
which is why no scenario noticed: every existing skill scenario proves the
message, and the corpse is only visible to a second `look`.

## 4. Fix

- `skillToAttackType` is exact for every damaging skill and spell, and its
  fallback is `TYPE_UNDEFINED` — C's own "no specific attack type", whose
  `make_corpse` case and default arm print the same line.
- `SkillResult.DamageAttackType()` returns `SkillMsgType` when set (C's own
  argument), else `skillToAttackType(DamageSkill)`.
- `ApplySkillDamage` / `ApplySkillDamageWithMessage` take that numeric attack
  type instead of a skill string, so the pipeline receives C's value directly.
- `SkillAmbushNum = 191` added to `pkg/game/death.go`.

## 5. Proof

`cmd/dp-oracle-diff/scenarios/skill-corpse-attack-type-depth.txt` — headbutt
kills the trainee guard, the attacker looks, and the room's object list carries
the corpse line. Reverting `DamageAttackType` reddens it:

```
-The bruised, battered corpse of a guard trainee is lying here.
+The corpse of a guard trainee is lying here.
```

R5h unit tests (`pkg/game/skill_attack_type_test.go`), all observed failing with
the preference reverted:

- `TestSkillDamageAttackTypeMatchesC` — every skill's numeric value and the
  corpse line C prints for it.
- `TestSkillResultDamageAttackTypePrefersCSkillMessageKey` — the
  `SkillMsgType`-first rule, including the `TYPE_UNDEFINED` fallback.
- `TestSkillKillCorpseUsesCSkillAttackType` — drives the real
  `ApplySkillDamageWithMessage` to a kill and asserts the corpse's long
  description for headbutt (bruised), trip (plain) and backstab (hacked).

## 6. Open, found by this audit

- **Headbutt auto-attack miss variant.** While building the vehicle, the first
  post-headbutt auto-attack round diverged: C
  `You try to hit a guard trainee who easily avoids the blow.` versus Go
  `A guard trainee ducks under your fist as you try to hit him.` Two different
  `TYPE_HIT` variants, so the shared RNG stream is one draw out of step after a
  headbutt. `attackType` cannot cause it (it reaches only the charge pain draw
  and `HandleDeath`), so it is a separate, pre-existing draw-order bug in the
  headbutt path. Needs its own vehicle.
- **Brief item 1 (fireball colour bytes)** is not done: the four colour levels
  × hit/miss/kill × caster/victim/onlooker × telnet/WebSocket capture.
- **Brief item 2 (delete the invented `skillMessageTable` fallback)** is not
  done.
- **Brief item 3** (`score`, `infobar`, `who`, `stat` colour proofs) is not
  done.

