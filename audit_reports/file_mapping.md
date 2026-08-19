# Port Fidelity Audit: File Mapping Plan

This document serves as the comprehensive execution plan and mapping system for the C-to-Go port fidelity audit. It maps each legacy C file in `src/` to its corresponding Go implementations, notes the architecture relationship, specifies the code complexity, and provides high-level context.

## Summary Stats

- **Total C Files mapped:** 64
- **High Complexity (>1000 lines C):** 28
- **Medium Complexity (200-1000 lines C):** 24
- **Low Complexity (<200 lines C):** 12
- **Unported Files:** 10

---

## File Mapping Table

| # | C File | Lines (C) | Go File(s) / Package(s) | Mapping Type | Complexity | Description / Port Context |
|---|--------|-----------|-------------------------|--------------|------------|----------------------------|
| 1 | `act.comm.c` | 1,566 | `pkg/session/comm_cmds.go`<br>`pkg/session/cmd_group.go`<br>`pkg/game/act_comm.go`<br>`pkg/game/world_scriptable.go` | 1:N | HIGH | Communication commands (say, tell, shout, gossip, gsay). |
| 2 | `act.display.c` | 717 | `pkg/game/act_display.go`<br>`pkg/session/display_cmds.go` | 1:N | MEDIUM | Screen/color settings, prompt customization, display modes. |
| 3 | `act.informative.c` | 2,803 | `pkg/game/act_informative.go`<br>`pkg/game/look.go`<br>`pkg/game/help.go`<br>`pkg/session/act_informative.go`<br>`pkg/session/informative_cmds.go`<br>`pkg/session/cmd_look.go`<br>`pkg/session/examine.go` | 1:N | HIGH | Info/look/help/score/equipment/inventory/examine commands. |
| 4 | `act.item.c` | 1,789 | `pkg/game/act_item_stubs.go`<br>`pkg/game/inventory.go`<br>`pkg/game/item_consumable.go`<br>`pkg/game/item_container.go`<br>`pkg/game/act_movement.go` (doors)<br>`pkg/game/item_equipment.go`<br>`pkg/game/item_helpers.go`<br>`pkg/game/item_transfer.go`<br>`pkg/session/cmd_inventory.go`<br>`pkg/session/use_cmds.go` | 1:N | HIGH | Item interaction (get, drop, give, wear, take off, junk, use). |
| 5 | `act.movement.c` | 951 | `pkg/game/act_movement.go`<br>`pkg/game/movement.go`<br>`pkg/session/cmd_movement.go`<br>`pkg/session/movement_cmds.go` | 1:N | MEDIUM | Movement commands (north/south/east/west, open/close, enter, leave). |
| 6 | `act.offensive.c` | 1,510 | `pkg/game/combat_helpers.go`<br>`pkg/game/combat_wire.go`<br>`pkg/game/player_combat.go`<br>`pkg/game/skill_combat.go`<br>`pkg/game/skill_c10_combat.go`<br>`pkg/game/combat_skill_names.go`<br>`pkg/session/act_offensive.go`<br>`pkg/session/cmd_combat_basic.go`<br>`pkg/session/cmd_combat_special.go`<br>`pkg/session/combat_cmds.go` | 1:N | HIGH | Combat action commands (kill, hit, kick, bash, rescue, flee). |
| 7 | `act.other.c` | 1,947 | `pkg/game/act_other_bridge.go`<br>`pkg/game/other_character.go`<br>`pkg/game/other_economy.go`<br>`pkg/game/other_helpers.go`<br>`pkg/game/other_mount.go`<br>`pkg/game/other_session.go`<br>`pkg/game/other_settings.go`<br>`pkg/game/other_status.go`<br>`pkg/game/skill_stealth.go`<br>`pkg/game/other_utility.go`<br>`pkg/session/cmd_misc.go` | 1:N | HIGH | Miscellaneous commands (quit, sleep, wake, visible, steal, practice). |
| 8 | `act.social.c` | 305 | `pkg/game/act_social.go`<br>`pkg/game/player_social.go`<br>`pkg/game/socials.go`<br>`pkg/session/act_social.go` | 1:N | MEDIUM | Social commands (smile, grin, laugh, cry, etc.) driven by tables/social files. |
| 9 | `act.wizard.c` | 3,863 | `pkg/session/wizard_cmds.go`<br>`pkg/session/wiz_communication.go`<br>`pkg/session/wiz_info.go`<br>`pkg/session/wiz_movement.go`<br>`pkg/session/wiz_object.go`<br>`pkg/session/wiz_player.go`<br>`pkg/session/wiz_stats.go`<br>`pkg/session/wiz_system.go`<br>`pkg/session/wiz_zone.go`<br>`pkg/command/admin_commands.go` | 1:N | HIGH | Wiz/Immortal commands (goto, force, load, shutdow, restore, set). |
| 10 | `alias.c` | 110 | `pkg/game/aliases.go` | 1:1 | LOW | Command alias parsing, saving, and reading system. |
| 11 | `ban.c` | 313 | `pkg/game/bans.go` | 1:1 | MEDIUM | Ban loading, saving, IP/site blocking system. |
| 12 | `boards.c` | 551 | `pkg/game/boards.go` | 1:1 | MEDIUM | In-game bulletin boards (reading, writing, deleting notes). |
| 13 | `circle.c` | 30 | `pkg/engine/gameloop.go` | 1:1 | LOW | legacy main() entrypoint, ported into modern game loop runner. |
| 14 | `clan.c` | 1,574 | `pkg/game/clans.go`<br>`pkg/game/clan_admin.go`<br>`pkg/game/clan_bank.go`<br>`pkg/game/clan_command.go`<br>`pkg/game/clan_economy.go`<br>`pkg/game/clan_info.go`<br>`pkg/game/clan_membership.go`<br>`pkg/game/clan_settings.go` | 1:N | HIGH | Clan management, ranking, bank, commands. |
| 15 | `class.c` | 1,191 | `pkg/game/class_tables.go`<br>`pkg/game/character.go`<br>`pkg/game/level.go` | 1:N | HIGH | Class definitions, XP levels, stat calculations, skill gains. |
| 16 | `comm.c` | 2,637 | `pkg/session/manager.go`<br>`pkg/session/protocol.go`<br>`pkg/session/session_manager.go`<br>`pkg/session/session_pump.go`<br>`pkg/session/session_login.go`<br>`pkg/telnet/listener.go` | 1:N | HIGH | Connection management, game loops, Telnet handshakes, nanny input parser. |
| 17 | `config.c` | 287 | `pkg/game/constants.go`<br>`pkg/session/char_creation.go` | 1:N | MEDIUM | MUD constants, limits, default rooms, start variables. |
| 18 | `constants.c` | 1,450 | `pkg/game/constants.go`<br>`pkg/game/spec_procs3.go` | 1:N | HIGH | Constant string arrays (flags, items, slots, conditions, positions). |
| 19 | `db.c` | 3,219 | `pkg/game/world.go`<br>`pkg/game/world_zone.go`<br>`pkg/game/spawner.go`<br>`pkg/game/serialize.go`<br>`pkg/parser/` | 1:N | HIGH | World loading (rooms, objects, mobs, zones), initialization, database bootstrap. |
| 20 | `dream.c` | 223 | `pkg/game/dreams.go`<br>`pkg/dreaming/dream.go` | 1:N | MEDIUM | AI/narrative dreaming engine. |
| 21 | `events.c` | 125 | `pkg/events/queue.go`<br>`pkg/events/bus.go`<br>`pkg/events/types.go` | 1:N | LOW | Event queue, micro-second timer scheduling. |
| 22 | `fight.c` | 2,033 | `pkg/combat/`<br>`pkg/game/deferred_fight_fns.go`<br>`pkg/game/party.go`<br>`pkg/game/death.go` | 1:N | HIGH | Combat ticker, violence loops, hits, damage, death, group exp. |
| 23 | `file-edit.c` | 199 | `pkg/game/note_write.go` | 1:1 | LOW | In-game line editor for mail, boards, descriptions. |
| 24 | `gate.c` | 397 | `pkg/game/gates.go` | 1:1 | MEDIUM | Inter-zone gates and portals. |
| 25 | `graph.c` | 405 | `pkg/game/graph.go`<br>`pkg/dreaming/graph.go` | 1:N | MEDIUM | Pathfinding (BFS), track command logic, dream graphs. |
| 26 | `handler.c` | 1,616 | `pkg/game/inventory.go`<br>`pkg/game/location.go`<br>`pkg/game/char_mgmt.go`<br>`pkg/game/world.go`<br>`pkg/game/death.go`<br>`pkg/engine/affect.go`<br>`pkg/game/affect_update.go` | 1:N | HIGH | Low-level element manipulation (affect apply, inventory transfer, list lookups). |
| 27 | `house.c` | 744 | `pkg/game/houses.go`<br>`pkg/game/house_boot.go`<br>`pkg/game/house_control.go`<br>`pkg/game/house_player.go`<br>`pkg/game/house_rent.go`<br>`pkg/game/house_save.go` | 1:N | MEDIUM | Player housing, saving items inside houses, renting, access lists. |
| 28 | `ident.c` | 277 | `NONE` | NONE | MEDIUM | **UNPORTED** (RFC 1413 Ident protocol. Legacy security check not needed in modern server). |
| 29 | `improved-edit.c` | 627 | `pkg/session/pager.go`<br>`pkg/session/wiz_player.go`<br>`pkg/game/note_write.go` | 1:1 | MEDIUM | Improved string editor for descriptions. |
| 30 | `interpreter.c` | 2,365 | `pkg/session/commands.go`<br>`pkg/command/registry.go`<br>`pkg/game/aliases.go` | 1:N | HIGH | Command parser, command lookup, aliases, subcommands, command validation. |
| 31 | `limits.c` | 686 | `pkg/game/limits.go`<br>`pkg/game/limits_condition.go`<br>`pkg/game/limits_exp.go`<br>`pkg/game/limits_gain.go`<br>`pkg/game/limits_misc.go` | 1:N | MEDIUM | Player HP/Mana/Move regen, level gain, age, hunger, thirst. |
| 32 | `luaedit.c` | 58 | `NONE` | NONE | LOW | **UNPORTED** (Legacy C in-game Lua editor. Go scripts are managed externally). |
| 33 | `magic.c` | 1,999 | `pkg/spells/`<br>`pkg/game/affect_update.go` | 1:N | HIGH | Spell effects (bless, armor, cure, fly, invis, poison), casting checks. |
| 34 | `mail.c` | 596 | `pkg/game/mail.go` | 1:1 | MEDIUM | MUD postal system (sending, receiving, storing MUD mail). |
| 35 | `mapcode.c` | 226 | `pkg/session/map_cmds.go` | 1:N | MEDIUM | Ascii/Constellation MUD map rendering. |
| 36 | `medit.c` | 1,126 | `NONE` | NONE | HIGH | **UNPORTED** (Legacy OLC mob editor. Go uses modern web/JSON/file editing). |
| 37 | `mobact.c` | 408 | `pkg/game/mobact.go`<br>`pkg/combat/engine.go`<br>`pkg/game/ai.go`<br>`pkg/game/deferred_fight_fns.go` | 1:N | MEDIUM | Mob autonomous behavior (moving, aggressive attacks, spec-procs triggers). |
| 38 | `mobprog.c` | 646 | `pkg/game/mobprogs.go` | 1:1 | MEDIUM | MobProgram triggers and script executions. |
| 39 | `modify.c` | 869 | `pkg/session/pager.go`<br>`pkg/session/wiz_player.go`<br>`pkg/game/note_write.go` | 1:1 | MEDIUM | String modification, page formatting, writing notes. |
| 40 | `new_cmds.c` | 2,792 | `pkg/game/skills.go`<br>`pkg/game/skills2.go`<br>`pkg/game/skill_combat.go`<br>`pkg/game/skill_stealth.go`<br>`pkg/game/limits_exp.go` | 1:N | HIGH | Special modern commands and skills (parry, dodge, stealth, etc.). |
| 41 | `new_cmds2.c` | 1,027 | `pkg/game/skills2.go`<br>`pkg/game/skill_advanced.go`<br>`pkg/game/skill_special.go` | 1:N | HIGH | Additional character skill commands. |
| 42 | `objsave.c` | 1,250 | `pkg/game/objsave.go`<br>`pkg/game/save.go` | 1:N | HIGH | Auto-save of player equipment and inventory (rent/crash files). |
| 43 | `oc.c` | 180 | `NONE` | NONE | LOW | **UNPORTED** (Legacy C-level main wrapper / wrapper shell. Dead infrastructure). |
| 44 | `oedit.c` | 1,564 | `NONE` | NONE | HIGH | **UNPORTED** (Legacy OLC object editor. Obsoleted by modern tooling). |
| 45 | `olc.c` | 524 | `NONE` | NONE | MEDIUM | **UNPORTED** (Legacy OLC main menu. Obsoleted by modern tooling). |
| 46 | `poof.c` | 102 | `pkg/session/wizard_cmds.go` | 1:1 | LOW | Wizard teleport messages (poofin/poofout). |
| 47 | `queue.c` | 175 | `pkg/events/queue.go` | 1:1 | LOW | Priority queue for scheduled events. |
| 48 | `random.c` | 73 | `pkg/combat/formulas.go`<br>`pkg/game/other_helpers.go` | 1:N | LOW | Custom random number generator (RNG) formulas, bounds. |
| 49 | `redit.c` | 1,078 | `NONE` | NONE | HIGH | **UNPORTED** (Legacy OLC room editor. Obsoleted by modern tooling). |
| 50 | `scripts.c` | 2,115 | `pkg/game/scripts.go`<br>`pkg/game/world_scriptable.go`<br>`pkg/scripting/engine.go` | 1:N | HIGH | Lua scripting bridge for triggers on mobs/rooms/objects. |
| 51 | `sedit.c` | 1,178 | `NONE` | NONE | HIGH | **UNPORTED** (Legacy OLC shop editor. Obsoleted by modern tooling). |
| 52 | `shop.c` | 1,445 | `pkg/game/shop.go`<br>`pkg/game/systems/shop.go`<br>`pkg/game/systems/shop_manager.go`<br>`pkg/session/shop_cmds.go` | 1:N | HIGH | Shopkeeper transactions, inventory buying/selling, pricing. |
| 53 | `spec_assign.c` | 642 | `pkg/game/spec_assign.go` | 1:1 | MEDIUM | Assigning special procedures to mob/room/object templates. |
| 54 | `spec_procs.c` | 2,420 | `pkg/game/spec_procs.go`<br>`pkg/game/postmaster.go` | 1:N | HIGH | Core special procedures (guildmasters, postmasters, cityguards, puff). |
| 55 | `spec_procs2.c` | 2,300 | `pkg/game/spec_procs2.go` | 1:1 | HIGH | Additional special procedures. |
| 56 | `spec_procs3.c` | 1,301 | `pkg/game/spec_procs3.go`<br>`pkg/game/player.go` | 1:N | HIGH | Zone specific special procedures (receptionists, cryogenics, guards). |
| 57 | `spell_parser.c` | 1,626 | `pkg/spells/spells.go`<br>`pkg/spells/spell_info.go`<br>`pkg/spells/say_spell.go`<br>`pkg/spells/call_magic.go` | 1:N | HIGH | Spell list, casting cost, reagents, mana checks, target verification. |
| 58 | `spells.c` | 1,218 | `pkg/spells/spells.go`<br>`pkg/spells/affect_spells.go`<br>`pkg/spells/damage_spells.go`<br>`pkg/game/affect_update.go` | 1:N | HIGH | Spell implementation of magical damage, affects, teleports. |
| 59 | `tattoo.c` | 186 | `pkg/session/tattoo.go`<br>`pkg/game/deferred_fight_fns.go` | 1:N | LOW | Character tattoo effects, timers, applications. |
| 60 | `tedit.c` | 98 | `NONE` | NONE | LOW | **UNPORTED** (Legacy OLC text editor. Obsoleted by modern web/JSON/file editing). |
| 61 | `utils.c` | 980 | `pkg/game/other_helpers.go`<br>`pkg/game/aliases.go`<br>`pkg/combat/formulas.go`<br>`pkg/game/deferred_fight_fns.go`<br>`pkg/spells/affect_spells.go` | 1:N | MEDIUM | Core utility functions (string formatting, location checks, grouping). |
| 62 | `weather.c` | 233 | `pkg/session/time_weather.go`<br>`pkg/game/weather.go` | 1:N | MEDIUM | Weather tick simulation (pressure, sky change, wind, time updates). |
| 63 | `whod.c` | 532 | `pkg/game/whod.go` | 1:1 | MEDIUM | WHO daemon command & server status information loop. |
| 64 | `zedit.c` | 1,276 | `NONE` | NONE | HIGH | **UNPORTED** (Legacy OLC zone editor. Obsoleted by modern tooling). |

---

## Macro & Header Files (Reference Context)

These structural headers define crucial MUD structures, bitvectors, and macro operations. They do not have dedicated behavioral Go implementations but are mapped implicitly into types and helper macros across `pkg/game/`, `pkg/session/`, and `pkg/spells/`. They are kept open as a macro glossary:
- `src/structs.h` → Reference for game structs (attributes, points, characters, objects, rooms).
- `src/utils.h` → Reference for utility macros (`GET_MAX_HIT`, `IS_NPC`, `AFF_FLAGGED`, `PLR_FLAGGED`).
- `src/spells.h` → Reference for magical spells and spell structures.
