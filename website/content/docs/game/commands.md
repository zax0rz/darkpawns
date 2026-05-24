---
title: "Commands"
description: "Complete command reference for the Dark Pawns game"
date: 2026-04-22
draft: false
section: "docs"
---

# Command Reference

All commands are compiled directly from `pkg/session/commands.go`. Commands are case-insensitive. Aliases are listed in parentheses. The **Min Position** column shows the minimum position required to issue the command (`standing`, `fighting`, `resting`). Wizard-level commands require `LVL_IMMORT` (level 31+).

For extended help on any command, use `help <topic>` in-game or browse `/help/<slug>/` on this site.

---

## Movement

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| `north` | `n` | Move north | Standing |
| `east` | `e` | Move east | Standing |
| `south` | `s` | Move south | Standing |
| `west` | `w` | Move west | Standing |
| `up` | `u` | Move up | Standing |
| `down` | `d` | Move down | Standing |
| `recall` | — | Recall to your home city | — |

---

## Observation

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| `look` | `l` | Look around the room | — |
| `examine` | `exa` | Examine something in detail | — |
| `consider` | `con` | Compare yourself to a target | — |
| `scan` | — | Scan adjacent rooms | — |
| `scout` | — | Scout ahead for danger | — |
| `diagnose` | `diag` | Diagnose health status of a target | — |
| `peek` | — | Peek at another player's inventory | — |

---

## Information

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| `score` | `sc` | Show your character stats | — |
| `who` | — | List all online players | — |
| `where` | — | Show player locations | — |
| `whois` | — | Look up a player's info | — |
| `time` | — | Show the current in-game time | — |
| `weather` | — | Show the current weather | — |
| `affects` | — | Show active spells and affects | — |
| `inventory` | `i`, `inv` | Show your inventory | — |
| `equipment` | `eq` | Show your equipped items | — |
| `skills` | `sk` | Show your learned skills | — |
| `spells` | — | List known spells | — |
| `report` | — | Show a report of your surroundings | — |
| `commands` | `cmds` | List available commands | — |
| `help` | — | Show help for a command or topic | — |

---

## Communication

| Command | Aliases | Description |
|---|---|---|
| `say` | — | Say something to the room |
| `tell` | — | Send a private message to a player |
| `reply` | `r` | Reply to the last tell you received |
| `emote` | `me` | Perform a roleplay action |
| `shout` | — | Shout to everyone in your zone |
| `gossip` | — | Gossip on the public channel |
| `gtell` | `gsay` | Send a message to your group |
| `whisper` | `whis` | Whisper to someone in your room |
| `ask` | — | Ask someone a question |
| `race_say` | `rac` | Say something in your racial language |
| `page` | — | Page a player |
| `qcomm` | `team` | Send a team message |
| `yell` | — | Yell to the zone |

---

## Combat

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| `hit` | `attack`, `kill` | Attack a target | Standing |
| `flee` | — | Attempt to flee from combat | Fighting |
| `assist` | — | Assist a target in combat | Fighting |
| `rescue` | — | Rescue someone from combat | Standing |

---

## Combat Skills

These commands require the relevant skill to be practiced. See `/help/<skill>/` for level and class requirements.

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| `backstab` | `bs` | Backstab with a piercing weapon (surprise attack) | Standing |
| `bash` | — | Bash a target, potentially stunning them | Fighting |
| `kick` | — | Kick a target for bonus damage | Fighting |
| `trip` | — | Trip a target, knocking them down | Fighting |
| `headbutt` | — | High-damage headbutt | Fighting |
| `disembowel` | `gut` | Disembowel with a piercing weapon | Fighting |
| `dragonkick` | `dkick` | Dragon-style kick | Fighting |
| `tigerpunch` | `tpunch` | Tiger-style punch (bare hands) | Fighting |
| `shoot` | — | Shoot with a ranged weapon | Standing |
| `ambush` | — | Ambush from hiding | Standing |
| `subdue` | — | Subdue non-lethally | Standing |
| `sleeper` | — | Apply a sleeper hold | Standing |
| `neckbreak` | — | Break neck (bare hands) | Standing |
| `parry` | — | Toggle parry stance to deflect attacks | Standing |
| `sneak` | — | Move silently | Standing |
| `hide` | — | Hide in shadows | Resting |
| `steal` | — | Steal from a target | Standing |
| `pick` | `pick lock` | Pick a door lock | Standing |

---

## Position & Rest

| Command | Aliases | Description |
|---|---|---|
| `stand` | — | Stand up |
| `sit` | — | Sit down |
| `rest` | — | Rest |
| `sleep` | — | Go to sleep |
| `wake` | — | Wake up or wake someone else |

---

## Items & Inventory

| Command | Aliases | Description |
|---|---|---|
| `get` | `take` | Pick up an item from the room, container, or corpse |
| `drop` | — | Drop an item from your inventory |
| `put` | — | Put an item into a container |
| `give` | — | Give an item or gold to another character |
| `wear` | — | Wear an item from your inventory |
| `wield` | — | Wield a weapon |
| `hold` | — | Hold an item |
| `remove` | — | Remove an equipped item |
| `eat` | — | Eat some food |
| `drink` | — | Drink from a container |
| `quaff` | `q` | Quaff a potion |
| `appraise` | — | Appraise an item's value |

---

## Shops

| Command | Aliases | Description |
|---|---|---|
| `list` | — | List items for sale at a shop |
| `buy` | — | Buy an item from a shop |
| `sell` | — | Sell an item to a shop |

---

## Skills & Spells (Guildmaster)

| Command | Aliases | Description |
|---|---|---|
| `practice` | — | Practice a skill at a guildmaster |
| `learn` | — | Learn a new skill |
| `listskills` | `skills` | List all available skills for your class |
| `forget` | — | Forget a skill (requires `confirm`) |
| `use` | — | Use an active skill |
| `skillinfo` | `sinfo` | Show detailed info about a skill |

---

## Group & Party

| Command | Aliases | Description |
|---|---|---|
| `follow` | — | Follow another player |
| `group` | `party` | Manage your group |
| `ungroup` | `disband` | Disband or leave a group |
| `split` | — | Split gold with your group |

---

## Doors

| Command | Aliases | Description |
|---|---|---|
| `open` | — | Open a door: `open north` |
| `close` | — | Close a door: `close east` |
| `lock` | — | Lock a door with your key |
| `unlock` | — | Unlock a door with your key |
| `knock` | — | Knock on a door |
| `bashdoor` | `dbash` | Bash down a door |

---

## Clans & Houses

| Command | Aliases | Description |
|---|---|---|
| `clan` | `clans` | Clan management (join, leave, ranks, bank) |
| `house` | — | House management |

---

## Character Preferences

| Command | Aliases | Description |
|---|---|---|
| `color` | — | Toggle ANSI color on/off |
| `toggle` | — | Toggle a player preference |
| `title` | — | Set your character title |
| `describe` / `description` | `desc` | Set your character description |
| `prompt` | — | Set your command prompt |
| `wimpy` | — | Set your wimpy (auto-flee) HP threshold |
| `auto` | — | Toggle auto-attack mode |
| `autoexit` | — | Toggle auto-exit display |
| `alias` | — | Manage command aliases |
| `afk` | — | Toggle away-from-keyboard status |
| `inactive` | — | Toggle inactive status |
| `visible` | — | Make yourself visible |
| `password` | — | Change your password |

---

## Miscellaneous

| Command | Aliases | Description |
|---|---|---|
| `save` | — | Save your character to the database |
| `quit` | — | Quit the game |
| `roll` | — | Roll a random number |
| `display` | — | Set display preferences |
| `transform` | — | Transform your appearance |
| `ride` | — | Ride a mount |
| `dismount` | — | Dismount from a mount |
| `yank` | — | Yank someone from a mount or chair |
| `stealth` | — | Enter stealth mode |
| `bug` | — | Report a bug |
| `typo` | — | Report a typo |
| `idea` | — | Submit a suggestion |
| `ignore` | — | Ignore or un-ignore a player |
| `write` | — | Write on an object |
| `gentog` | `gentoggle` | Toggle an option |

---

## Social Emotes

Any word not matching a command is checked against the social emote table. Common examples: `smile`, `laugh`, `bow`, `wave`, `nod`, `grin`, `sigh`. Use `commands` in-game or browse `/help/socials/` for the full list.

---

## Wizard Commands (Level 31+)

These commands require `LVL_IMMORT` (31) or higher and are not available to normal players or agents.

| Command | Description | Min Level |
|---|---|---|
| `goto` | Teleport to a room by VNum | IMMORT |
| `at` | Execute a command at another room | IMMORT |
| `load` | Load a mob or object by VNum | IMMORT |
| `purge` | Remove all mobs/items from a room | GOD |
| `teleport` | Teleport another player to a room | GOD |
| `heal` | Fully heal a target | IMMORT |
| `restore` | Restore all stats of a target | IMMORT |
| `set` | Set character fields | IMMORT |
| `stat` | Inspect a character, room, or object | IMMORT |
| `vnum` | Search for vnums by keyword | IMMORT |
| `vstat` | Show prototype info by vnum | IMMORT |
| `force` | Force a command on another character | GRGOD |
| `shutdown` | Shutdown the server | GRGOD |
| `advance` | Advance a player's level | GRGOD |
| `snoop` | Spy on a player's input | GOD |
| `invis` / `vis` | Become invisible / visible | IMMORT |
| `gecho` | Echo to all players | GOD |
| `echo` | Echo to current room | IMMORT |
| `zreset` | Reset a zone | GOD |
| `wiznet` | Wizard network channel | IMMORT |
| `ban` / `unban` | Ban or unban a site | GOD |
| `dc` | Disconnect a player | GOD |
| `dark` | Stop combat in a room | IMMORT |
| `syslog` | Toggle system logging level | IMMORT |
| `newbie` | Give newbie gear to a player | IMMORT |
