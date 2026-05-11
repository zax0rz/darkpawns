# CRIT Package — Daeron to Blenda

**Date:** 2026-05-10
**From:** Daeron (loremaster)
**To:** Blenda (infra)
**Re:** CRIT-009 and CRIT-010 design assessments

---

## CRIT-010: load_messages() — PRIORITY: HIGH, Content Day

### What exists in Go

The tiered weapon message system already works. `damMessageTiers` in `pkg/combat/fight_core.go:688` has all 14 tiers:

| MinDamage | Sample message |
|---|---|
| 0 | "tries to #w, but misses" |
| 1 | "scratches $N as $e #W $M" |
| 3 | "barely #W $N" |
| ... | ... |
| 48 | "MUTILATES $N!" |
| 60 | "DISEMBOWELS $N!!" |
| 80 | "DESTROYS $N!!!" |
| 101 | "OBLITERATES $N!!!!" |
| 10000 | "R O C K S the Hell Out Of $N!!!!!!!!!!!!!!!!!!!!!!!!" |

`DamMessage()` at line 710 selects the right tier based on damage. Token replacement ($n, $N, #w, #W) works. The tier selection logic is correct.

### What's missing

**1. Multiple variants per tier.**

The C `MESS_FILE` (misc/messages) had 3-4 different messages per tier, randomly selected. In `skill_message()` at fight.c:1023:

```c
nr = dice(1, fight_messages[i].number_of_attacks);
for (j = 1, msg = fight_messages[i].msg; (j < nr) && msg; j++)
    msg = msg->next;
```

It rolls a random number against the count of variants for that attack type, walks the linked list to the Nth entry, and sends that variant. Go has exactly ONE message per tier. A new player seeing "You barely #w $N" three times in a row feels mechanical. The C version kept it fresh.

**2. `load_messages()` parser.**

C loaded from `misc/messages` at startup (fight.c:126). Messages were data-driven — edit the file, restart the server, new combat flavor. Our messages are hardcoded in `fight_core.go`. Adding or editing messages requires recompilation.

**3. Skill-specific message tables.**

`SkillMessageFunc` at fight_core.go:18 exists as a hook but has no table behind it. `SkillMessage()` at line 1200 just calls the hook and returns. Skills that bypass the weapon path (bash, kick, backstab) get no flavor text at all.

In C, `skill_message()` at fight.c:1023 had per-attack-type message entries. Each skill/spell type had its own message set with die_msg, hit_msg, miss_msg, and god_msg variants. The Go `SkillMessageFunc` hook is wired but empty.

### Scope estimate

One focused day:

1. Build a message table struct with variant slices (replace single `damMessageTier` strings with `[]string`)
2. Add variant selection: `rand.Intn(len(variants))` in `DamMessage()`
3. Write a `load_messages()` parser — can start with hardcoded tables, migrate to data file later
4. Populate from the C MESS_FILE or write fresh DP-flavored variants (the C messages are at src/fight.c:889 in `dam_weapons[]`)
5. Wire `SkillMessageFunc` to a new skill message table (bash, kick, backstab, etc.)
6. Add die_msg/hit_msg/miss_msg/god_msg variants per skill type

### Reference

- C implementation: `src/fight.c:126` (load_messages), `src/fight.c:889` (dam_message), `src/fight.c:1023` (skill_message)
- Go tier system: `pkg/combat/fight_core.go:688` (damMessageTiers), `pkg/combat/fight_core.go:710` (DamMessage)
- Go skill hook: `pkg/combat/fight_core.go:18` (SkillMessageFunc), `pkg/combat/fight_core.go:1200` (SkillMessage)
- C message file: `src/db.h:68` — `#define MESS_FILE "misc/messages"`

---

## CRIT-009: Dual hit-resolution path — PRIORITY: DEFER, Not a Bug

### What exists

Two combat paths:

1. **`processCombatPair`** (`pkg/combat/engine.go:236`) — auto-attack every combat round.
   Flow: hit roll → parry check → dodge check → damage → death.
   Full defense every round.

2. **`doDamage`** (`pkg/game/damage_stubs.go:80`) — skill attacks (bash, kick, backstab, etc.).
   Flow: skill-specific hit logic → damage → death.
   NO parry/dodge check.

### Why this is intentional

This is CircleMUD design, not a defect. Skills are cooldown-limited resources:

- **Bash** costs stamina, has a timer
- **Kick** has a cooldown
- **Backstab** requires positioning (behind the target)

The tradeoff for spending that resource is guaranteed connection (if the skill's own hit check passes). Spells use saving throws instead of parry/dodge — different defense system entirely.

If parry/dodge applied to every skill, skills would become less reliable, fights would last longer, and players would use skills more conservatively. That changes the entire combat feel.

### If balance tuning is needed later

The clean approach:

1. Extract `CheckParry`/`CheckDodge` from `processCombatPair` into a `CombatEngine.CheckDefense()` method
2. Call it from both paths with an opt-out flag
3. Backstab always skips (from behind = no defense)
4. Bash/kick roll against it (warrior skill = can be parried)
5. Spells continue using saving throws

This is a design decision, not a code fix. Defer until combat balance can be tested with live players.

### Reference

- Auto-attack path: `pkg/combat/engine.go:236` (processCombatPair)
- Skill path: `pkg/game/damage_stubs.go:80` (doDamage)
- Parry check: `pkg/combat/formulas.go` (CheckParry)
- Dodge check: `pkg/combat/formulas.go` (CheckDodge)
- C reference: `src/fight.c:1525` — comment explains skill_message vs dam_message routing

---

## Summary

| Item | Priority | Effort | Owner |
|---|---|---|---|
| CRIT-010 (load_messages) | HIGH | 1 day content work | Blenda |
| CRIT-009 (dual hit path) | DEFER | Design decision | The Architect (when ready) |

The Architect's note: combat messages aren't polish — they're the experience. A new player getting ROCKED by a wandering mob is a core memory. The damage number is irrelevant. The message IS the memory.
