---
title: "Game Mechanics"
description: "Combat formulas, economy, PK rules, and the remort system for Dark Pawns"
date: 2026-04-22
draft: false
section: "docs"
---

# Game Mechanics Deep-Dive

This page documents the precise mechanical formulas used by the Dark Pawns server, compiled directly from `pkg/combat/` Go sources. All formulas are faithful ports of the original C `fight.c` and `class.c` logic.

---

## Combat

### THAC0 vs AC (Hit Resolution)

Hit resolution follows the original AD&D THAC0 system:

```
HitRoll = 1d20 + HitBonus
Hits if: HitRoll >= (THAC0 - TargetAC)
```

- **THAC0** ("To Hit Armor Class 0") starts at 20 and improves (decreases) as you level and gain Strength.
- **AC** ranges from 10 (naked) to −10 (heavily armored). Lower AC is harder to hit.
- DEX provides an AC bonus. Wearing armor, casting armor spells, and picking AC-granting equipment all lower your AC.
- **INT and WIS modifiers** apply a slight THAC0 reduction (bonus) for Mages, Clerics, and Psionics.
- **Blessed weapons** grant a THAC0 bonus. **Drunk characters** incur a THAC0 penalty.

### Attacks Per Round

The number of attacks a character gets per 2-second round scales with class and level (`GetAttacksPerRound()` from `fight.c`):

| Class | Base Attacks | Extra Attack |
|---|---|---|
| Warrior, Paladin, Ranger | 1 | +1 at level 10+ (60% + level% chance) |
| Ninja | 1 | +1 at level 10+ (60% + level% chance) |
| Thief, Assassin | 1 | +1 at level 15+ (30% + level% chance) |
| Mage, Cleric, Psionic | 1 | No extra attack from class |

`Haste` spell doubles attacks; `Slow` halves them.

### Damage Calculation

Base damage formula (`CalculateDamage()` from `fight.c`):

```
BaseDamage = WeaponDiceRoll + DamageRoll
FinalDamage = BaseDamage × PositionMultiplier × ParryReduction
```

- **DamageRoll**: Derived from STR for melee; also modified by level for combat skills.
- **Position multiplier**: Sleeping targets take 2× damage. Resting targets take 1.5×. Standing targets take 1×.
- **Parry reduction**: Reduces incoming damage by 20–29% (scaling with parry skill level), when the `parry` stance is active.

### Backstab Multiplier

The backstab skill (Thieves and Assassins) and disembowel use this multiplier from `class.c`:

```
BackstabMult(level) = level × 0.2 + 1.0
```

| Level | Multiplier |
|---|---|
| 1 | 1.2× |
| 5 | 2.0× |
| 10 | 3.0× |
| 20 | 5.0× |
| 25 | 6.0× |
| 30 | 7.0× |

Backstab requires a **piercing weapon** and must be initiated from a non-fighting stance. Disembowel (mid-combat piercing technique) uses `BackstabMult / 3.0`.

---

## Player Killing (PK)

Dark Pawns has a live PK system with the following rules:

### Restrictions

1. **Level gate:** Players at **level 10 or below** cannot be attacked by other players, nor can they attack other players. This protects new characters.
2. **Peaceful rooms:** Rooms with the `ROOM_PEACEFUL` flag block all PK damage (combat still engages but deals 0 damage).
3. **Shopkeepers:** Shopkeeper NPCs cannot be killed.

### Outlaw System

If you kill a player who is **not flagged as an outlaw**, you become an **outlaw** yourself:
- City guards (mob VNums 8102 and 8103) actively intercept and subdue outlaws who attack non-outlaw players.
- The outlaw flag persists until it expires or an Immortal clears it.
- Outlaws can be attacked freely by other players without triggering the outlaw flag on the attacker.

### Corpse Looting

When a player dies in PK:
- A corpse spawns in the room with all of the victim's inventory.
- **Any player** can loot the corpse (there is no loot protection in the base system).
- The corpse persists for a limited time before disappearing.
- The victim respawns at the Temple (room 8004) with the standard EXP penalty of `EXP / 3`.

---

## Economy

### Shopkeeper Pricing

Shopkeepers use per-shop **profit multipliers** defined in the area files:

```
BuyPrice  = ItemValue × ProfitBuy    # what you pay the shop
SellPrice = ItemValue × ProfitSell   # what the shop pays you
```

A typical shop has `ProfitBuy = 1.2` (you pay 120% of item value) and `ProfitSell = 0.8` (the shop pays you 80%). High-end or rare shops may have wider spreads.

Use `appraise <item>` to see an estimate of what a shopkeeper will pay before selling.

### Gold & Banking

- Gold is carried in your inventory and is dropped on death.
- The **Clan** system provides a shared clan bank vault (`clan bank deposit/withdraw`).
- Individual player banking is handled by in-game banker NPCs (`deposit`, `withdraw`).

---

## Remort System

After reaching **level 30 (Hero)** on your first character, you unlock the remort system. Remorti allow access to advanced classes unavailable to fresh characters.

### Base Classes (available at character creation)

| Class | Playstyle |
|---|---|
| Warrior | Exceptional frontline melee |
| Mage | AoE spells, high INT |
| Cleric | Healing, buffing, divine spells |
| Thief | Backstab, stealth, picking locks |
| Ninja | Agility, evasion, martial arts |
| Psionic | Mental spellcaster using Mind/Psi pool |

### Remort-Only Classes (unlocked after first Hero)

| Class | Based On | Specialization |
|---|---|---|
| Assassin | Thief | Maximum backstab damage, silent kill |
| Avatar | Any | Hybrid divine/martial |
| Magus | Mage | Arcane mastery, widened spell access |
| Paladin | Warrior | Holy warrior; lay on hands; guards city |
| Ranger | Warrior | Dual wield, nature affinity, shoot |
| Mystic | Psionic | Advanced mental techniques |

Remort characters retain their level 30 experience base and start the new class from level 1, preserving some of the original class's skill set.

---

## Experience & Leveling

Experience is gained from killing mobs and (in PK) from killing players. Each class/race combination has its own EXP table governing how much experience is required per level. The penalty for death is:

```
ExpLoss = TotalExp / 3
```

This is a significant penalty at high levels. Always `flee` if your health drops below your wimpy threshold.

To level up, visit your class **guildmaster** in your starting city after accumulating enough EXP. The guildmaster will grant you a new level and allow you to `practice` new skills.

---

## Useful Formulas Summary

| Formula | Source |
|---|---|
| Hit: `1d20 + HitBonus >= THAC0 − AC` | `fight.c:1783` |
| Backstab: `damage × (level × 0.2 + 1.0)` | `class.c:720` |
| Disembowel: `damage × (level × 0.2 + 1.0) / 3` | `fight.c:1108` |
| Death penalty: `ExpLoss = TotalExp / 3` | `fight.c:1675` |
| Buy price: `ItemValue × ProfitBuy` | `shop.c` |
| Sell price: `ItemValue × ProfitSell` | `shop.c` |
| Regen tick: every 30 seconds | `PointUpdate()` |
| Combat tick: every 2 seconds | `CombatEngine` goroutine |
