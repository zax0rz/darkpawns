---
title: "Commands"
description: "Complete command reference for the Dark Pawns game"
date: 2026-04-22
draft: false
section: "world"
aliases:
  - /docs/game/commands/
  - /docs/game/commands
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
| [`recall`](/help/commands/recall/) | — | Recall to your home city | — |

---

## Observation

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| [`look`](/help/commands/look/) | `l` | Look around the room | — |
| [`examine`](/help/commands/examine/) | `exa` | Examine something in detail | — |
| [`consider`](/help/commands/consider/) | `con` | Compare yourself to a target | — |
| `scan` | — | Scan adjacent rooms | — |
| [`scout`](/help/commands/scout/) | — | Scout ahead for danger | — |
| [`diagnose`](/help/commands/diagnose/) | `diag` | Diagnose health status of a target | — |
| [`peek`](/help/commands/peek/) | — | Peek at another player's inventory | — |

---

## Information

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| [`score`](/help/commands/score/) | `sc` | Show your character stats | — |
| [`who`](/help/commands/who/) | — | List all online players | — |
| `where` | — | Show player locations | — |
| `whois` | — | Look up a player's info | — |
| [`time`](/help/commands/time/) | — | Show the current in-game time | — |
| [`weather`](/help/commands/weather/) | — | Show the current weather | — |
| `affects` | — | Show active spells and affects | — |
| [`inventory`](/help/commands/inventory/) | `i`, `inv` | Show your inventory | — |
| [`equipment`](/help/commands/equipment/) | `eq` | Show your equipped items | — |
| `skills` | `sk` | Show your learned skills | — |
| `spells` | — | List known spells | — |
| [`report`](/help/commands/report/) | — | Show a report of your surroundings | — |
| [`commands`](/help/commands/commands/) | `cmds` | List available commands | — |
| [`help`](/help/commands/help/) | — | Show help for a command or topic | — |

---

## Communication

| Command | Aliases | Description |
|---|---|---|
| `say` | — | Say something to the room |
| `tell` | — | Send a private message to a player |
| [`reply`](/help/commands/reply/) | `r` | Reply to the last tell you received |
| [`emote`](/help/commands/emote/) | `me` | Perform a roleplay action |
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
| [`flee`](/help/commands/flee/) | — | Attempt to flee from combat | Fighting |
| [`assist`](/help/commands/assist/) | — | Assist a target in combat | Fighting |
| [`rescue`](/help/commands/rescue/) | — | Rescue someone from combat | Standing |

---

## Combat Skills

These commands require the relevant skill to be practiced. See `/help/<skill>/` for level and class requirements.

| Command | Aliases | Description | Min Position |
|---|---|---|---|
| [`backstab`](/help/commands/backstab/) | `bs` | Backstab with a piercing weapon (surprise attack) | Standing |
| [`bash`](/help/commands/bash/) | — | Bash a target, potentially stunning them | Fighting |
| [`kick`](/help/commands/kick/) | — | Kick a target for bonus damage | Fighting |
| [`trip`](/help/commands/trip/) | — | Trip a target, knocking them down | Fighting |
| [`headbutt`](/help/commands/headbutt/) | — | High-damage headbutt | Fighting |
| [`disembowel`](/help/commands/disembowel/) | `gut` | Disembowel with a piercing weapon | Fighting |
| `dragonkick` | `dkick` | Dragon-style kick | Fighting |
| `tigerpunch` | `tpunch` | Tiger-style punch (bare hands) | Fighting |
| `shoot` | — | Shoot with a ranged weapon | Standing |
| [`ambush`](/help/commands/ambush/) | — | Ambush from hiding | Standing |
| [`subdue`](/help/commands/subdue/) | — | Subdue non-lethally | Standing |
| [`sleeper`](/help/commands/sleeper/) | — | Apply a sleeper hold | Standing |
| [`neckbreak`](/help/commands/neckbreak/) | — | Break neck (bare hands) | Standing |
| [`parry`](/help/commands/parry/) | — | Toggle parry stance to deflect attacks | Standing |
| [`sneak`](/help/commands/sneak/) | — | Move silently | Standing |
| [`hide`](/help/commands/hide/) | — | Hide in shadows | Resting |
| [`steal`](/help/commands/steal/) | — | Steal from a target | Standing |
| `pick` | `pick lock` | Pick a door lock | Standing |

---

## Position & Rest

| Command | Aliases | Description |
|---|---|---|
| `stand` | — | Stand up |
| `sit` | — | Sit down |
| `rest` | — | Rest |
| [`sleep`](/help/commands/sleep/) | — | Go to sleep |
| `wake` | — | Wake up or wake someone else |

---

## Items & Inventory

| Command | Aliases | Description |
|---|---|---|
| `get` | `take` | Pick up an item from the room, container, or corpse |
| [`drop`](/help/commands/drop/) | — | Drop an item from your inventory |
| [`put`](/help/commands/put/) | — | Put an item into a container |
| [`give`](/help/commands/give/) | — | Give an item or gold to another character |
| [`wear`](/help/commands/wear/) | — | Wear an item from your inventory |
| [`wield`](/help/commands/wield/) | — | Wield a weapon |
| `hold` | — | Hold an item |
| [`remove`](/help/commands/remove/) | — | Remove an equipped item |
| `eat` | — | Eat some food |
| `drink` | — | Drink from a container |
| `quaff` | `q` | Quaff a potion |
| [`appraise`](/help/commands/appraise/) | — | Appraise an item's value |

---

## Shops

| Command | Aliases | Description |
|---|---|---|
| [`list`](/help/commands/list/) | — | List items for sale at a shop |
| [`buy`](/help/commands/buy/) | — | Buy an item from a shop |
| [`sell`](/help/commands/sell/) | — | Sell an item to a shop |

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
| [`follow`](/help/commands/follow/) | — | Follow another player |
| [`group`](/help/commands/group/) | `party` | Manage your group |
| [`ungroup`](/help/commands/ungroup/) | `disband` | Disband or leave a group |
| [`split`](/help/commands/split/) | — | Split gold with your group |

---

## Doors

| Command | Aliases | Description |
|---|---|---|
| `open` | — | Open a door: `open north` |
| `close` | — | Close a door: `close east` |
| `lock` | — | Lock a door with your key |
| `unlock` | — | Unlock a door with your key |
| `knock` | — | Knock on a door |

---

## Clans & Houses

| Command | Aliases | Description |
|---|---|---|
| `clan` | `clans` | Clan management (join, leave, ranks, bank) |
| [`house`](/help/commands/house/) | — | House management |

---

## Character Preferences

| Command | Aliases | Description |
|---|---|---|
| `color` | — | Toggle ANSI color on/off |
| [`toggle`](/help/commands/toggle/) | — | Toggle a player preference |
| [`title`](/help/commands/title/) | — | Set your character title |
| `describe` / `description` | `desc` | Set your character description |
| `prompt` | — | Set your command prompt |
| [`wimpy`](/help/commands/wimpy/) | — | Set your wimpy (auto-flee) HP threshold |
| `auto` | — | Toggle auto-attack mode |
| `autoexit` | — | Toggle auto-exit display |
| `alias` | — | Manage command aliases |
| `afk` | — | Toggle away-from-keyboard status |
| [`inactive`](/help/commands/inactive/) | — | Toggle inactive status |
| [`visible`](/help/commands/visible/) | — | Make yourself visible |
| `password` | — | Change your password |

---

## Miscellaneous

| Command | Aliases | Description |
|---|---|---|
| [`save`](/help/commands/save/) | — | Save your character to the database |
| [`quit`](/help/commands/quit/) | — | Quit the game |
| `roll` | — | Roll a random number |
| `summon` | — | Summon a player to your room |
| [`review`](/help/commands/review/) | — | Show recent gossip history |
| `todo` | — | Submit a todo suggestion |
| `description` | `desc` | Set your character description (alias for `describe`) |
| `display` | — | Set display preferences |
| [`transform`](/help/commands/transform/) | — | Transform your appearance |
| `ride` | — | Ride a mount |
| [`dismount`](/help/commands/dismount/) | — | Dismount from a mount |
| [`yank`](/help/commands/yank/) | — | Yank someone from a mount or chair |
| [`stealth`](/help/commands/stealth/) | — | Enter stealth mode |
| `hcontrol` | — | Admin house control |
| `bug` | — | Report a bug |
| `typo` | — | Report a typo |
| `idea` | — | Submit a suggestion |
| `ignore` | — | Ignore or un-ignore a player |
| [`write`](/help/commands/write/) | — | Write on an object |
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
| `switch` | Enter another character's body | IMMORT |
| `return` | Return from switched body | IMMORT |
| `reload` | Reload world data | GOD |
| `send` | Send a message to another character | GOD |
| `wizlock` | Toggle wizard-only login | IMPL |
| `idlist` | Dump object ID list to file | IMPL |
| `checkload` | Check zone load info for a mob/obj | IMMORT |
| `poofset` | Set poof in/out messages | IMMORT |
| `zlist` | List zones matching a filter | IMMORT |
| `rlist` | List rooms matching a keyword | IMMORT |
| `olist` | List objects matching a keyword | IMMORT |
| `mlist` | List mobiles matching a keyword | IMMORT |
| `sysfile` | Show system file path | IMMORT |
| `sethunt` | Set hunt target for a character | IMMORT |
| `tick` | Show current tick info | IMMORT |
| `users` | Show connected players | IMMORT |
| `whod` | Toggle WHOD display mode | IMMORT |
| `dark` | Stop combat in a room | IMMORT |
| `syslog` | Toggle system logging level | IMMORT |
| [`newbie`](/help/commands/newbie/) | Give newbie gear to a player | IMMORT |
| [`order`](/help/commands/order/) | Order a pet or follower | IMMORT |
