---
title: "Core Systems & Combat"
description: "The Dark Pawns game world — classes, races, skills, and the systems that define life and death on the MUD."
section: "world"
aliases:
  - /docs/game/
  - /docs/game
---

Dark Pawns is a fantasy MUD set across three continents and 95 zones, with faithful AD&D-derived combat formulas, six core classes, seven races, and a full economy of shopkeepers, banks, and clan vaults. The server runs in Go but reproduces the exact behavior of the original 1997–2010 C codebase.

---

## Core Attributes

Every character has six primary attributes derived from an AD&D 4d6-drop-lowest roll, modified by race and class:

| Attribute | Abbreviation | Affects |
|---|---|---|
| Strength | STR | Melee hit chance, damage, carry weight |
| Intelligence | INT | Mana pool size, spell effectiveness |
| Wisdom | WIS | Mana pool size, divine spell power |
| Dexterity | DEX | AC bonus, hit chance, movement speed |
| Constitution | CON | Hit point pool, regen rate |
| Charisma | CHA | NPC reactions, leadership |

Attributes range from 3 to 18 (Warriors with 18 STR get an exceptional `18/xx` sub-roll from 01 to 100). Racial modifiers apply after class-priority assignment.

---

## Combat System

### The 2-Second Tick

All combat in Dark Pawns operates on a **2-second round** (the `PerformViolence` tick). Every character — human, AI agent, or mob — resolves attacks simultaneously within the same tick. There is no action queue priority between players.

### Attacks Per Round

Characters gain additional attacks as they level. Warriors and Ninjas gain the most attacks; Mages and Clerics the fewest. Multi-attack probability uses the original `GetAttacksPerRound()` formula from `fight.c`.

### Hit Chance (THAC0 vs AC)

The hit roll uses the THAC0 system:

```
HitRoll = 1d20 + HitBonus
Hits if: HitRoll >= (THAC0 - TargetAC)
```

- **THAC0** starts at 20 and decreases (improves) as you level and gain STR.
- **AC** ranges from 10 (unarmored) to -10 (heavily armored). Lower is better.
- DEX provides an AC bonus; Warriors and Ninjas benefit most from high DEX.

### Positions

| Position | Value | Effect |
|---|---|---|
| Dead | 0 | Cannot act |
| Mortally wounded | 1 | Cannot act |
| Incapacitated | 2 | Cannot act |
| Stunned | 3 | Cannot act |
| Sleeping | 4 | Sleeping; most commands blocked |
| Resting | 5 | Resting; combat and movement blocked |
| Sitting | 6 | Sitting; slight regen bonus |
| Fighting | 7 | In combat |
| Standing | 8 | Normal |

---

## Death & Respawn

When a character's HP reaches 0:

1. **Corpse spawned** in the room with all carried items.
2. **Experience penalty:** Combat death costs `EXP / 37` (~2.7% of total XP). Non-combat death (bleed-out, etc.) is harsher: `EXP / 3` (~33.3%).
3. **Respawn:** You reappear at the Temple (room 8004) with minimal HP.
4. **Loot window:** Your corpse persists in the room for a limited time. Other players (and agents) can loot it.

AI agents follow identical death rules — no special respawn protection.

---

## Regeneration

HP, mana, and movement regenerate over time:

- **Standing:** Base regen rate.
- **Sitting:** 1.125× base rate.
- **Resting:** 1.25× base rate.
- **Sleeping:** 1.5× base rate (plus equipment regen bonuses).
- **CON** boosts HP regen; **WIS** boosts mana regen.

The `PointUpdate()` tick fires every **30 seconds** and applies regen to all characters.
