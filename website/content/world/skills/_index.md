---
title: "Skills"
description: "Core combat, stealth, and utility skills that define character capabilities and progression."
aliases:
  - /world/skills/skills/
  - /world/skills/skills
date: 2026-05-24
---

# Skills & Spells System

Dark Pawns features a rich, dual-layered skill and magic model. Mortals grow stronger through direct combat experience, mental study, and dedicated training near guildmasters.

---

## The Dual-Layer Architecture

To maintain high mechanical fidelity with legacy game engines while adding depth, character skills operate on two distinct, synchronized layers:

### Layer 1: Core CircleMUD Skills
The baseline game mechanics. Every combat skill and stealth check is calculated directly through this layer. Skills are represented as raw proficiencies ranging from **0 to 100**. Level requirements are strictly gated by your class.

### Layer 2: The SkillManager
A modern progression overlay. Characters possess a pool of **Skill Points** and a maximum number of **Skill Slots** (default 10). You must use skill points to unlock skills in the SkillManager before they can be practiced or used in the game command loop.

---

## Learning & Practice

### Unlocking a Skill
To unlock a new skill, you must visit a class guildmaster. Learning requires:
1.  **Character Level:** Your level must meet the skill's difficulty ranking.
2.  **Stat Thresholds:** Combat skills require $\ge 10$ Strength/Dexterity; Magic skills require $\ge 12$ Intelligence/Wisdom; Utility skills require $\ge 8$ Dexterity/Intelligence.
3.  **Skill Points:** You must spend points equal to the skill's difficulty.
4.  **Available Slots:** You must have an empty slot (maximum of 10 slots used, unless expanded).

### Improving Your Proficiency
There are two distinct paths to mastery:
*   **Use-Based Improvement:** Successfully executing a skill in the world (e.g., scoring a hit with `backstab` or hiding from sight) grants a chance to improve. The improvement check is driven by a combination of your **Intelligence and Wisdom** stats. As your proficiency grows higher, the window for automatic improvement becomes smaller.
*   **Guildmaster Practice:** You can deliberately practice at a class guildmaster. Typing `practice <skill>` near a guildmaster adds practice points. Once practice points cross 100, your character attempts a level-up check based on stats and level versus the skill's difficulty. A success levels up the skill.

### Proficiency Titles

Your level of mastery (0 to 100) displays with the following titles:

| Level Scale | Title | Description |
| :--- | :--- | :--- |
| **0** | *Unlearned* | The character has no knowledge of this skill. |
| **1–24** | *Novice* | Basic understanding; success rate is highly unpredictable. |
| **25–49** | *Apprentice* | Developing muscle memory; moderate success in combat. |
| **50–74** | *Journeyman* | Highly competent; able to reliably teach others. |
| **75–89** | *Expert* | Exceptional speed and recovery in high-stakes situations. |
| **90–99** | *Master* | Renowned specialist; near-flawless execution. |
| **100** | *Grandmaster* | Peak mortal capability; legendary speed and precision. |

---

## Core Skill Registries

Below are the default skills wired into the Dark Pawns game engine, organized by type:

### 1. Combat & Offensive Skills

| Skill Command | Gated Classes | Minimum Level | Description |
| :--- | :--- | :--- | :--- |
| `backstab` | Thief, Assassin | 1 | A highly lethal strike executed from stealth behind a target. |
| `bash` | Warrior, Paladin, Ranger | 3 | A heavy shield or body blow designed to knock a target down. |
| `kick` | Most Classes | 1 | A quick, tactical foot strike dealing minor damage. |
| `trip` | Thief, Assassin | 9 | Sweeping a target's legs, incapacitating them for several combat ticks. |
| `headbutt` | Warrior, Paladin, Ranger | 5–7 | A forceful head strike dealing moderate damage with a minor stun. |
| `rescue` | Warrior, Paladin, Ranger | 3–5 | Divert a monster's attention to protect a fragile party member. |

### 2. Stealth & Utility Skills

| Skill Command | Gated Classes | Minimum Level | Description |
| :--- | :--- | :--- | :--- |
| `sneak` | Thief, Assassin | 2 | Move silently through rooms, evading aggressive monster triggers. |
| `hide` | Thief, Assassin, Ranger | 5–10 | Melt into the shadows, making yourself completely invisible to onlookers. |
| `steal` | Thief, Assassin | 3 | Filch gold or items from another humanoid's inventory. |
| `pick_lock` | Thief, Assassin | 4 | Bypass locked doors or chest security mechanisms without a key. |

### 3. Advanced Combat (C-10 Spec)
These represent highly advanced combat options unlocked by seasoned combatants:

*   `disembowel` — A brutal, bleeding slash dealing massive ongoing damage.
*   `dragon_kick` / `tiger_punch` — Special high-tier unarmed martial arts styles.
*   `subdue` / `sleeper` / `neckbreak` — Advanced chokeholds and finishing strikes.
*   `ambush` — Strategic offensive initiation from hiding.
*   `parry` — Dynamic weapon deflection to block incoming attacks.
*   `escape` / `retreat` — Controlled disengagement from active combat.

### 4. Magic Domains
Spellcasters study eight distinct fields of magical spellcraft, gating their spell selection:

*   **Evocation** (Direct combat damage, fireballs, lightning)
*   **Abjuration** (Wards, shields, protection spells)
*   **Conjuration** (Summoning entities and physical resources)
*   **Divination** (Identity checks, scanning, locating objects)
*   **Enchantment** (Target buffs, weapon charging)
*   **Illusion** (Invisibility, mirror images)
*   **Necromancy** (Lifetap, raising corpses, rot curses)
*   **Transmutation** (Flesh altering, speed buffs)
