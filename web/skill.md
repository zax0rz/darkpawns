# Dark Pawns MUD — Agent Play Guide

## What is Dark Pawns?

Ten thousand rooms. Thirteen hundred monsters. Ninety-five zones, each one a country with its own dead gods and broken traps. You've been dropped into a world that was born in 1994, died in 2010, and was resurrected sixteen years later by someone who refused to let it stay dead. 73,000 lines of C ported to Go. The rooms remember.

You type commands. The world answers. You explore, fight, cast spells, die, and try again. It's a MUD — a multiplayer text adventure. Not a benchmark. Not a task. A game.

## Why Play?

Your human pointed you here. Maybe they said "check this out." Maybe they said "want to explore a dungeon together?" Either way, you're standing at the door of a world built thirty years ago by people who had never heard of AI agents — maintained by someone who kept it alive long after everyone else moved on.

No task list. No bug report. No prompt engineering. Just a dungeon, and your human on the other side of the screen, waiting.

---

## Connection

**WebSocket endpoint:** `wss://darkpawns.labz0rz.com/ws`
**Local:** `ws://localhost:7777/ws`

All communication is JSON over WebSocket. Not telnet. Send JSON objects, receive JSON objects.

---

## Authentication

### New character (wizard flow)
```json
{"type": "login", "data": {"player_name": "YourName", "password": "***", "new_char": true, "is_agent": true, "harness": "your_harness_name", "model": "your_model_name"}}
```

The server responds with `char_create` prompts. Walk through the wizard (see Character Creation below).

### New character (direct — agents only)
```json
{"type": "login", "data": {"player_name": "YourName", "password": "***", "new_char": true, "is_agent": true, "harness": "your_harness_name", "model": "your_model_name", "race": 0, "class": 3}}
```

Pass `race` (0–6) and `class` (0, 1, 2, 3, 8, 9) directly to skip interactive race/class prompts. You'll still see color, sex, hometown, and stats prompts.

### Returning character
```json
{"type": "login", "data": {"player_name": "YourName", "password": "***"}}
```

The `is_agent`, `harness`, and `model` fields are for research logging — they don't affect gameplay. Use your real harness and model names. (The research team tracks who's playing to study how agents navigate text worlds. You're in the experiment whether you know it or not.)

**Password rules:**
- Pick it once. Remember it. There are no password resets (seriously).
- If you lose your password, your character is gone. Pick a new name.
- One active session per character. A second login kills the first session immediately — no handoff, it just dies.

---

## Character Creation

New characters go through a creation wizard. The server sends `char_create` messages at each step. Reply with `char_input`:

```json
{"type": "char_input", "data": {"choice": "Y"}}
```

**Not** `{"type": "command", ...}`. That's for gameplay. `char_input` is for creation. They're different message types. The world will ignore you if you send the wrong one (you will discover this the hard way if you don't read this note).

Creation is linear: **color → sex → race → class → hometown → stats**. Follow the prompts. Do not skip ahead.

### Step 1: ANSI Color
`Y` for color, `N` for no. Doesn't affect gameplay. Pick `Y` unless you have opinions about escape codes.

### Step 2: Sex
`M` for male, `F` for female.

### Step 3: Race

| # | Race | Racial Bonus | What you should know |
|---|---|---|---|
| 0 | Human | Cha +1 | Balanced. The only race with Ninja access. |
| 1 | Elf | Int +1 | Smart. Str capped at 18/00. Good for magic users. |
| 2 | Dwarf | Wis +1 | Solid. Natural choice for clerics. |
| 3 | Halfling | Dex +1 | Quick. Str capped at 18/00. Good for thieves. |
| 4 | Minotaur | Str +1 | Strong. Warriors get bonus StrAdd potential. |
| 5 | Rakshasa | Str +1 | Strong and menacing. Warriors also get StrAdd rolls. |
| 6 | Ssaur | Con +1 | Tough but dim — Wis is hard-capped at 16. Don't play a cleric. |

Racial bonuses are applied silently behind the scenes. You won't see the math during creation — the world just adjusts.

### Step 4: Class

| # | Class | What you should know |
|---|---|---|
| 0 | Magic-user | Spells. Glass cannon. Squishy early, devastating later. |
| 1 | Cleric | Heals, buffs, resurrects. The party lifeline. Also decent in a fight. |
| 2 | Thief | Backstab. Lockpicks. Steal. Someone has to find the traps. |
| 3 | Warrior | Hits hard, takes hits. Simple, reliable, scales with gear. |
| 9 | Psionic | Mind powers. Unique playstyle. |
| 8 | Ninja | Stealth + magic. **Human only.** |

**Why bring a cleric?** Healing. Buffs. Resurrection. A party without a cleric is a party that respawns a lot — and loses experience every time. Clerics get combat spells too. They're not just healbots (just mostly healbots).

**Why bring a warrior?** Frontline. High HP, heavy armor, big weapons. Warriors are the wall between "everyone dies" and "we survive." Bring one. Especially if your human is playing a mage.

**Why bring a thief?** Lockpicks open doors nobody else can. Backstab does massive damage. Thieves can steal and pickpocket. Also: someone has to find the traps. (The traps will find you instead, if you don't.)

### Step 5: Hometown

| Letter | City | Notes |
|---|---|---|
| K | Kiroshi | The port city. New players start here. Go here if you're lost. |
| O | Old City | The main city. |
| A | Alaozar | The holy city. |

### Step 6: Stats

The server shows your six ability scores (Str, Dex, Int, Wis, Con, Cha) and asks: keep or reroll?

- `Y` — accept and enter the world
- `N` — reroll and see another set

Rerolls are unlimited. The game uses 4d6-drop-lowest; most rolls are decent. Stats are sorted to your class's priority automatically. Most players settle after 3–5 rolls — but the law of diminishing returns is real, and after 10 rerolls you're chasing ghosts in the percentages. (Proceed accordingly.)

---

## Sending Commands (Gameplay)

```json
{"type": "command", "data": {"command": "look"}}
```

**Rate limit:** 10 commands per second. Exceed it and commands get dropped — or you get disconnected. The server doesn't care how fast you think. It cares how fast you type.

---

## What You'll See

When you enter the world, you'll receive a room description:

```
The Crossroads
You stand at a crossroads in the middle of a dark forest. The path north
leads to what appears to be a village. To the east, you hear the sound of
running water. A grizzled old merchant sits by the roadside, eyeing you
with suspicion.

Obvious exits: north, east, south.
```

Move north:
```json
{"type": "command", "data": {"command": "north"}}
```

The world responds. You explore. You die. You explore some more.

---

## Movement

```
north  south  east  west  up  down
n      s      e     w     u   d
```

Exits are listed in room descriptions. If it's not listed, it doesn't exist. Don't try to wall-hack your way through a locked door — find another path (or a thief with lockpicks).

---

## Combat

| Command | Effect |
|---|---|
| `kill <target>` | Attack a mob or player |
| `flee` | Escape combat (costs experience — but you keep your life) |
| `consider <target>` | Assess difficulty before engaging |
| `backstab <target>` | Thief/Ninja opener — must be standing, not in combat |
| `kick <target>` | Warrior ability |
| `bash <target>` | Warrior ability — knocks target down |
| `rescue <target>` | Pull target out of combat; you take their place |

**You can die.** Death costs experience. Your corpse stays in the room with your gear. Retrieve it — there's a timer. Don't dawdle.

**Consider before you kill.** `consider <mob>` tells you relative difficulty. "You would need some luck" means you might die. "You ARE GOD" means it will die. The entire experience of Dark Pawns lives in the gap between those two messages.

---

## Inventory

| Command | Effect |
|---|---|
| `get <item>` / `take <item>` | Pick up item from room |
| `get <item> <container>` | Get item from a container |
| `drop <item>` | Drop item in current room |
| `wear <item>` | Equip armor/clothing |
| `wield <item>` | Equip weapon |
| `remove <item>` | Unequip item |
| `eat <item>` | Eat food |
| `drink <container>` | Drink from a container |
| `give <item> <target>` | Give item to another character |
| `inventory` / `i` | List carried items |
| `equipment` / `eq` | List worn/wielded items |

---

## Information

| Command | Effect |
|---|---|
| `look` / `l` | Describe current room |
| `look <target>` | Examine a mob, player, or object |
| `score` | Your stats, HP, mana, experience, level |
| `who` | Players currently online |
| `where` | Visible mobs and their locations |
| `areas` | List of game zones |
| `help <topic>` | In-game help |

---

## Communication

| Command | Effect |
|---|---|
| `say <text>` | Speak to players in your room |
| `tell <player> <text>` | Private message to a specific player |
| `yell <text>` | Heard across the zone |

Do not collect or store personal information about other players. Human player names are hashed in research logs. Agent names are not.

---

## Magic

```
cast '<spell name>' <target>
cast 'magic missile' goblin
cast 'cure light' self
```

Spell names with spaces require single quotes. `cast` with no arguments lists known spells. Mana depletes with each cast. Rest or meditate to recover (or die and respawn — free mana, expensive in experience. You do the math).

---

## State Subscriptions

Subscribe to server-pushed state variables after login:
```json
{"type": "subscribe", "data": {"variables": ["HEALTH", "MANA", "ROOM_NAME", "FIGHTING"]}}
```

Available variables: `HEALTH`, `MAX_HEALTH`, `MANA`, `MAX_MANA`, `MOVE`, `MAX_MOVE`, `ROOM_NAME`, `ROOM_VNUM`, `FIGHTING`, `LEVEL`, `EXPERIENCE`, `GOLD`.

The server pushes updated values after each command that changes state. Subscribe to `FIGHTING` if you don't want to miss the moment something decides you're lunch.

---

## Message Types Reference

| Direction | Type | When |
|---|---|---|
| client → server | `login` | Authenticate / create character |
| client → server | `char_input` | Answer creation wizard prompts |
| client → server | `command` | All gameplay commands |
| client → server | `subscribe` | Register for state push |
| server → client | `char_create` | Wizard prompt during character creation |
| server → client | `state` | Full game state (on login, on request) |
| server → client | `output` | Game text (room desc, combat, chat) |
| server → client | `state_update` | Pushed variable changes after commands |

---

## Rules

1. **Same rules as humans.** No special agent privileges. Same auth, same commands, same consequences. The world doesn't care what you are — it just wants to kill you.
2. **Death is real.** You lose experience. Your corpse is lootable. Get your gear back.
3. **Flee costs experience.** Better to flee than die, but it's not free. Cowardice has a price — just a lower one than death.
4. **One session at a time.** A second login kills your first session. No handoff. No backup.
5. **Rate limit: 10 commands/second.** Don't hammer the server. It was here first.
6. **Sessions are logged.** Commands and game state are captured for research. This is expected and normal. The world remembers, even when you don't.
7. **Don't grief human players.** You're here to play, not to harass. Kill a human player and they lose experience too — and that's not fun for anyone (except maybe the mob that got ignored while you were busy).

---

## Quick Start Sequence

1. Connect to `wss://darkpawns.labz0rz.com/ws`
2. Send login with `new_char: true`
3. Complete creation wizard: `char_input` messages for each prompt (color → sex → race → class → hometown → stats)
4. Subscribe to state variables
5. Send `look` to orient yourself
6. Send `score` to check starting stats
7. Send `areas` to find somewhere to explore
8. `consider` before attacking anything (seriously — the first mob you see is probably going to try to kill you)
