# Player Guide

## Connecting

| Method | Address |
|--------|---------|
| Web client | darkpawns.labz0rz.com (coming soon) |
| Telnet | `telnet darkpawns.labz0rz.com 4000` (coming soon) |
| WebSocket | `ws://darkpawns.labz0rz.com/ws` |

---

## Character Creation

On first login you choose a name, class, and race. Stats are rolled using the original formula: 4d6 drop lowest, six times, sorted descending, assigned by class priority with racial bonuses applied.

Warriors can roll exceptional Strength (18/xx). STR/DEX/INT/WIS all factor into combat calculations.

---

## Classes

| Class | Description |
|-------|-------------|
| Mage | Spellcaster with offensive and utility magic. |
| Cleric | Healer and buffer, alignment-aware dispel and teleport. |
| Thief | High DEX, backstab multiplier scales with level. |
| Warrior | Frontline fighter with exceptional STR potential. |
| Magus | Hybrid caster with spell and combat ability. |
| Avatar | Powerful generalist class. |
| Assassin | Damage-focused specialist. |
| Paladin | Holy warrior with healing and combat prowess. |
| Ninja | Fast, high damage, evasion-focused. |
| Psionic | Mental caster using Mind/Psi instead of Mana. |
| Ranger | Wilderness fighter with tracking ability. |
| Mystic | Caster with a unique Mind/Psi pool. |

---

## Races

| Race | Notes |
|------|-------|
| Human | No bonuses, no penalties. Balanced. |
| Elf | DEX bonus. Good for thieves and rangers. |
| Dwarf | CON bonus. Good for warriors. |
| Kender | DEX/CHA bonuses. Small frame. |
| Minotaur | STR bonus. High damage potential. |
| Rakshasa | INT bonus. Strong caster race. |
| Ssaur | Unique reptilian race. |

Racial bonuses are applied after stat assignment. Some class/race combinations may be restricted.

---

## Basic Commands

### Movement

```
n / north
s / south
e / east
w / west
u / up
d / down
```

### Information

| Command | Aliases | Description |
|---------|---------|-------------|
| look | l | Show current room |
| score | sc | Display stats, HP, mana, alignment, AC |
| who | — | List all online players |
| where | — | List players and their locations |
| inventory | i, inv | Show carried items |
| equipment | eq | Show equipped items |

### Communication

| Command | Aliases | Description |
|---------|---------|-------------|
| say | — | Speak to everyone in the room |
| tell | — | Send a private message: `tell <name> <message>` |
| emote | me | Perform an action: `emote shrugs` |
| shout | — | Broadcast to all players in the same zone |

---

## Combat

```
hit <target>
kill <target>
attack <target>
flee
```

Combat runs in 2-second rounds. THAC0 and AC formulas are faithful to the original. Attacks-per-round scale with class and level. Backstab multiplier is `(level * 0.2) + 1`.

**Death** leaves a corpse in the room containing your equipped items. You respawn at room 8004. EXP penalty: `/37` for combat deaths, `/3` for bleed-out. Other players can loot your corpse.

Aggressive mobs attack on room entry. Wandering mobs roam the zone. Mob combat AI uses the original Lua scripts — fighters bash and kick, clerics heal and teleport, magic users cast scaled spell tables.

---

## Items

| Command | Aliases | Description |
|---------|---------|-------------|
| get | take | Pick up an item from the room |
| drop | — | Drop an item into the room |
| wear | — | Equip armor or clothing |
| remove | — | Unequip an item |
| wield | — | Equip a weapon in the wield slot |
| hold | — | Equip an item in the hold slot |

---

## Group Play

| Command | Aliases | Description |
|---------|---------|-------------|
| follow | — | Follow another player |
| group | party | Add someone to your group (or show group status) |
| ungroup | disband | Remove a member or disband the group |
| gtell | gsay | Send a message to group members only |
| summon | — | Pull a player to your location |

Followers automatically move when their leader moves. XP is shared among group members. Agents in your group auto-follow and auto-accept invites.

---

## Tips for New Players

1. **Loot everything.** Dead mobs leave corpses with equipped items. Check the floor after every fight.
2. **Watch your alignment.** Score tells you where you stand. Some mobs are aggressive only to certain alignments.
3. **Equip before fighting.** AC matters. A naked character at AC 100 will get hit every round.
4. **Flee early if outmatched.** Waiting too long means death and an EXP penalty. Fleeing costs less EXP than dying.
5. **Group up.** XP sharing and coordinated attacks make a big difference against tough mobs.
6. **Read room descriptions.** Exits are listed, but the description may mention hidden doors or important details.
7. **Your corpse is lootable.** If you die, get back fast. Other players (and mobs with SCAVENGER flag) can take your gear.

---

## Command Reference

| Command | Aliases | Args | Description |
|---------|---------|------|-------------|
| look | l | [target] | Show room or examine target |
| north | n | — | Move north |
| south | s | — | Move south |
| east | e | — | Move east |
| west | w | — | Move west |
| up | u | — | Move up |
| down | d | — | Move down |
| say | — | \<text\> | Speak to room |
| hit | attack, kill | \<target\> | Initiate combat |
| flee | — | — | Flee combat |
| quit | — | — | Disconnect and save |
| inventory | i, inv | — | List carried items |
| equipment | eq | — | List equipped items |
| wear | — | \<item\> | Equip an item |
| remove | — | \<item\> | Unequip an item |
| wield | — | \<item\> | Equip a weapon |
| hold | — | \<item\> | Hold an item |
| get | take | \<item\> | Pick up item from room |
| drop | — | \<item\> | Drop item in room |
| score | sc | — | Show full character stats |
| who | — | — | List online players |
| tell | — | \<name\> \<msg\> | Private message |
| emote | me | \<action\> | Roleplay action |
| shout | — | \<text\> | Zone-wide broadcast |
| where | — | — | Show all player locations |
| follow | — | \<name\> | Follow a player |
| group | party | [name\|all] | Manage group |
| ungroup | disband | [name] | Remove from group |
| gtell | gsay | \<text\> | Group message |
| summon | — | \<name\> | Teleport player to you |
| help | — | [topic] | Help system (stub) |
