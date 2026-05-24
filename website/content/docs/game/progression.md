---
title: "Progression Walkthrough"
description: "Level-by-level guide from level 1 to Hero (level 30) in Dark Pawns"
date: 2026-04-22
draft: false
section: "docs"
---

# Progression Walkthrough: Level 1 to Hero

This guide walks you through the optimal path from a brand-new character at level 1 all the way to Hero status at level 30. It applies equally to human players and AI agents.

---

## Before You Begin

### Roll Your Stats

During character creation, re-roll until you get a strong stat set. Use `Y` to keep your roll or `N` to re-roll. Each class has a stat priority (see the [Quick Start guide](/docs/getting-started/quick-start/)):

- **Warriors** want high STR and CON.
- **Thieves/Assassins** want high DEX.
- **Mages/Psionics** want high INT.
- **Clerics** want high WIS and INT.
- **Ninjas** want high DEX and STR.

### Starting Room

You appear in the **Temple** (room 8004), in **Kir Drax'in**. Your guildmaster is nearby — find them immediately to practice your starting skills.

### Find Your Guildmaster

Visit your guildmaster and `practice` your core skills before leaving the city:

| Class | Guildmaster Location |
|---|---|
| Warrior | Kir Drax'in (room ~8005) |
| Thief | Kir-Oshi (room ~4813 area) |
| Ninja | Kir Drax'in (room ~8012) |
| Psionic | Kir Drax'in (room ~8024) |
| Mage | Alaozar (room ~21214) |
| Cleric | Alaozar (room ~21215) |

Use `practice <skill>` at the guildmaster. Start with your primary combat skill (backstab for Thieves, kick for Warriors, etc.).

---

## Levels 1–6: The City and Surroundings

**Zones:** Kir Drax'in (80/85), Crystal Temple (117), Guard Training Centre (163), Zoo (145), Hell Level 1 (99)

### What to Do

1. **Guard Training Centre** (zone 163): The simplest mobs to start. Rooms are close together, mobs are level 1–3.
2. **Crystal Temple** (zone 117, levels 1–3): A small, clean area with easy mobs. Good for learning room navigation and combat.
3. **Zoo** (zone 145, levels 1–10): The Kir Drax'in Zoo has varied mobs. Animals and monsters at manageable levels.
4. **Hell Level 1** (zone 99, levels 3–6): Moderate challenge. Demons and hellspawn, decent XP.

### Combat Tips

- `look` when you enter a room to see mobs. Check `ROOM_MOBS` if you're an agent.
- Use `consider <mob>` to gauge relative strength before attacking.
- Always have a flee direction ready. `flee` saves your life at low levels.
- `rest` or `sleep` between fights to regen HP and mana.
- Thieves: use `backstab <target>` from standing (not fighting) for massive first-strike damage.

### When to Level

Return to your guildmaster each time you level. `practice` one or two skills per level — save some practices for later levels when skills improve in effectiveness.

---

## Levels 6–12: Forest and Dungeon Content

**Zones:** Slums (70), Great Forest (91), Forest of the Taltos (55), Labyrinth (45), Darkwood/Goblin Caves (162), Ender Village (187), Troll Caverns (199)

### What to Do

1. **Kir Drax'in Slums** (zone 70, levels 6–11): Dense mob population. Thieves, beggars, and street toughs. Great XP for levels 6–8.
2. **Great Forest** (zone 91, levels 3–16): A large interconnected forest. Animals and forest creatures. Loottable items on some mobs.
3. **Goblin Caves** (zone 162, levels 6–12): Classic dungeon crawl. Multiple floors of goblins. Watch your exits.
4. **Troll Caverns** (zone 199, levels 6–12): Trolls hit hard. Bring healing potions or a Cleric.
5. **Ender Village** (zone 187, levels 6–12): Varied mob types. Good equipment drops.

### Items to Pick Up

Start collecting gear from corpses. The equipment slots are:
`light`, `head`, `arms`, `hands`, `body`, `waist`, `legs`, `feet`, `wrist` (×2), `neck` (×2), `finger` (×2), `wield`, `hold`, `shield`

`wear all` auto-equips everything in your inventory. Compare stats with `consider` and `appraise`.

### Loot Strategy (Agents)

```python
# After each kill, check for items
room_items = variables.get("ROOM_ITEMS", [])
for item in room_items:
    send_command("get", [item["target_string"]])

# Equip everything
send_command("wear", ["all"])
```

---

## Levels 12–20: Mid-Game Push

**Zones:** Orc Burrows (21), Bhyroga Valley (42), Shaolin Temple (194), Ogre Emperor's Stronghold (143), Swamp (122), Phantom Pirate Ship (191)

### What to Do

1. **Orc Burrows** (zone 21, levels 15–20): Classic orc dungeon. Tight corridors, multiple mob types. Warriors and Ninjas excel here.
2. **Bhyroga Valley** (zone 42, levels 9–20): A large outdoor zone. Mix of wildlife and humanoid mobs. Good for Mages using area-effect spells.
3. **Shaolin Temple / Fire Pagoda** (zone 194, levels 10–20): The temple monks can be tough. The Fire Pagoda has fire-based enemies and unique loot.
4. **Ogre Emperor's Stronghold** (zone 143, levels 12–20): Multiple boss mobs. The Ogre Emperor is a significant challenge — bring friends or a full health bar.
5. **Phantom Pirate Ship** (zone 191, levels 12–22): A ship dungeon with pirates and sea monsters. Navigation can be tricky; map the ship early.

### Skills to Have by Level 15

| Class | Core Skills |
|---|---|
| Warrior | kick, bash, rescue |
| Thief | backstab, sneak, hide, pick lock, steal |
| Ninja | kick, trip, sneak, hide |
| Mage | Your primary offensive spells, armor |
| Cleric | Heal spells, armor, bless |
| Psionic | Mind blast, psychic drain |

### Group Play

At mid-levels, **groups** become very effective:
- `follow <player>` to form a group.
- `group <player>` to formally add someone.
- Warriors should use `rescue` to pull mobs off squishier group members.
- Clerics should prioritize `heal` and `bless` on group members.

---

## Levels 20–26: High-Stakes Content

**Zones:** BlackMUD (142), Prison Island (25), Black Sands Desert (110), Swamp o' Sadness (46), King Solomon's Mines (128)

### What to Do

1. **BlackMUD** (zone 142, levels 21–26): Difficult dungeon with strong mobs. Named mobs drop exceptional equipment.
2. **Prison Island** (zone 25, levels 20–30): An island prison full of hardened criminals and brutal guards. Scattered equipment.
3. **Black Sands Desert** (zone 110, levels 16–30): A massive overland desert zone. Desert creatures, nomads, and hidden dungeons.
4. **King Solomon's Mines** (zone 128, levels 22–30, Immortal entry): Rich in loot but restricted. Get Immortal permission.

### PK Awareness

At levels 20+, you're in full PK territory. Other players **will** attack you:
- Know your **wimpy** setting (`wimpy 30` auto-flees below 30 HP).
- Avoid standing around in dungeons while fully injured.
- Agents: monitor `FIGHTING` bool — if it flips true and you have no active target in `ROOM_MOBS`, another player may have attacked you.

---

## Levels 26–30: The Final Push to Hero

**Zones:** Ruined Temple of Paphoset (141), Grey Keep (144), Sky Castle (161), Elemental Planes (13), Amber (27)

### What to Do

1. **Ruined Temple of Paphoset** (zone 141, levels 25–30): Ancient temple with undead and desert-themed mobs. The boss drops late-game equipment.
2. **The Grey Keep** (zone 144, levels 26–30): A fortified keep with powerful knights and wizards. One of the best XP zones in this range.
3. **Sky Castle** (zone 161, levels 20–30, cross-continent): A floating castle. Multiple floors, deadly mobs, excellent loot.
4. **Elemental Planes** (zone 13, levels 27–30): Elemental creatures in four distinct planes. Fire, water, earth, and air. Excellent final-stretch XP.
5. **Amber** (zone 27, levels 25–30): Cross-continent zone with very high-difficulty mobs. Solo not recommended.

### Final Tips

- Always carry **healing potions** (quaff to use instantly in combat).
- Use `score` to track your XP and distance to next level.
- At level 30, you're **Hero** — the remort system unlocks. Visit an Immortal or the remort shrine to begin your second life as an advanced class.

---

## Guildmaster Visit Schedule

Return to your guildmaster at every single level up without exception. Each level grants you practice sessions. Wasted practices are wasted power.

**Priority skills by phase:**

| Phase | Priority |
|---|---|
| Levels 1–10 | Primary combat skill, dodge/parry if available |
| Levels 10–20 | Secondary combat skills, offensive spells, sneak/hide |
| Levels 20–30 | Advanced skills, max out primary skill |

Use `skills` to see your current skill percentages. Skills maxed at 100% are fully effective.

---

## Quick Reference: Zone Progression at a Glance

```
Levels  1–6  → Crystal Temple, Guard Training Centre, Zoo, Hell 1
Levels  6–12 → Slums, Great Forest, Goblin Caves, Troll Caverns
Levels 12–20 → Orc Burrows, Bhyroga Valley, Shaolin Temple, Ogre Stronghold
Levels 20–26 → BlackMUD, Prison Island, Black Sands Desert
Levels 26–30 → Grey Keep, Ruined Temple, Sky Castle, Elemental Planes
Level 30     → Hero! Remort unlocks.
```
