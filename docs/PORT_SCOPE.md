# Dark Pawns — Port Scope & Remaining Work

**Date:** 2026-04-26  
**Method:** Function-by-function comparison of every C source file against Go counterparts. Automated name matching + manual verification.

---

## Summary

**~2,400 C lines remaining** (~1,700-2,400 Go lines to write).

| Status | Systems |
|--------|---------|
| ✅ Done | mobact, clans, shops, mail, dreams, tattoos, whod, spec_assign |
| 🟡 Small gaps | boards, weather, graph, informative, display, movement |
| 🔴 Significant work | handler helpers, objsave, houses, limits, wizard commands, events/queue, socials, scripts |

---

## Fully Ported Systems
- **mobact.c** — Mob AI (wander, aggro, memory, scavenging, spec_proc dispatch)
- **clan.c** — Full clan system
- **shop.c** — Buy/sell/list with charisma modifiers
- **mail.c** — Read/write/index/delete, postmaster NPC
- **dream.c** — Dream travel
- **tattoo.c** — Full tattoo system
- **whod.c** — Who list
- **spec_assign.c** — Mob/obj/room spec assignment
- **ban.c** — Basic ban checking (load/write missing)

---

## Remaining Work (by priority)

### Priority 1: Core gameplay blockers (~800 C lines)

#### limits.c — Leveling & resource gains (208 C lines)
| Function | What it does |
|----------|-------------|
| `exp_needed_for_level` | XP table for leveling |
| `find_exp` | Level from XP |
| `mana_gain` | Mana regen per tick (based on class, stats, position) |
| `hit_gain` | HP regen per tick |
| `move_gain` | Movement point regen per tick |
**Go location needed:** `pkg/game/limits.go` (exists, needs these functions)

#### handler.c — Missing core helpers (171 C lines)
| Function | What it does |
|----------|-------------|
| `remove_follower` | Remove from follow chain |
| `set_hunting` | Set mob hunting target |
| `obj_from_room` | Remove object from room |
| `obj_to_obj` | Put object inside container |
| `obj_from_obj` | Remove object from container |
| `get_number` | Parse leading number from string |
| `affect_to_char2` | Master affect application variant |
| `get_obj_num` | Find object by prototype VNum |
| `get_char_num` | Find character by ID |
| `free_char` | Full character cleanup |
**Go location needed:** `pkg/game/inventory.go`, `pkg/game/movement.go`

#### Shared small helpers (~100 C lines)
`remove_follower`, `set_hunting`, `stop_follower`, `can_speak`, `add_follower`, `num_followers`, `circle_follow`, `die_follower` — these are used across fight.c, mobact.c, handler.c, etc. Some exist as methods on MobInstance; need centralized versions in follow.go.

### Priority 2: Persistence & infrastructure (~750 C lines)

#### objsave.c — Binary save format (394 C lines)
| Function | What it does |
|----------|-------------|
| `Obj_to_store` | Serialize object to binary |
| `Obj_to_store_from` | Serialize equipped object |
| `Crash_load` | Load saved objects on login |
| `Crash_delete_file` | Delete crash save file |
| `Crash_delete_crashfile` | Delete crash backup |
| `Crash_write_rentcode` | Write rent info header |
| `Crash_restore_weight` | Restore object weight from save |
| `Crash_extract_objs` | Extract saved objects |
| `Crash_extract_norents_from_equipped` | Remove no-rent items |
| `Crash_extract_expensive` | Remove expensive items |
**Note:** Go may use JSON instead of binary format. The logic is what matters.
**Go location needed:** `pkg/game/objsave.go` (partially done)

#### house.c — File I/O (349 C lines)
| Function | What it does |
|----------|-------------|
| `House_load` | Load house from file |
| `House_save` | Save house to file |
| `House_crashsave` | Save on crash/idle |
| `House_delete_file` | Delete house file |
| `House_save_control` | Save house control records |
| `hcontrol_list_houses` | List all houses |
| `hcontrol_build_house` | Build new house |
| `hcontrol_destroy_house` | Destroy house |
| `hcontrol_pay_house` | Pay house rent |
**Go location needed:** `pkg/game/houses.go` (1086 lines, structure exists, I/O missing)

#### events.c + queue.c — Event queue system (155 C lines)
The C code uses a custom event queue for delayed actions (mob AI ticks, combat rounds, etc.). Go uses goroutines and time.Ticker. The missing functions are:
- Event scheduling, cancellation, processing
- Queue dequeue, head, key operations
**Go location needed:** May already be handled by Go concurrency. Needs verification.

### Priority 3: Player commands & features (~550 C lines)

#### act.wizard.c — Immortal commands (345 C lines)
| Function | What it does |
|----------|-------------|
| `do_stat_room` | Show room details |
| `do_stat_object` | Show object details |
| `stop_snooping` | Stop watching a player |
| `perform_immort_vis` | Toggle immortal visibility |
| `perform_immort_invis` | Toggle immortal invisibility |
| `print_zone_to_buf` | Zone info for output |
**Go location needed:** `pkg/game/wizard_cmds.go`

#### act.social.c — Social command system (107 C lines)
| Function | What it does |
|----------|-------------|
| `fread_action` | Parse social from file |
| `find_action` | Find social by name |
| `boot_social_messages` | Load all socials at boot |
**Go location needed:** `pkg/game/act_social.go` (structure exists, file loading missing)

#### ban.c — Ban persistence (72 C lines)
| Function | What it does |
|----------|-------------|
| `load_banned` | Load ban list from file |
| `isbanned` | Check if site is banned |
| `write_ban_list` | Save ban list |
**Go location needed:** `pkg/game/bans.go`

#### modify.c — String editor helpers (93 C lines)
| Function | What it does |
|----------|-------------|
| `next_page` | Pagination helper |
| `count_pages` | Count pages for display |
| `quad_arg` | Parse 4-argument string |
**Go location needed:** Various

### Priority 4: Polish & secondary systems (~350 C lines)

#### act.informative.c — Missing display helpers (59 C lines)
`find_target_room`, `find_class_bitvector`, `exp_needed_for_level`, `find_exp`, `print_object_location`

#### oc.c — Object editor commands (119 C lines)
`oc_onlist`, `oc_show_list`, `oc_dispose_list`

#### mapcode.c — ASCII map (86 C lines)
`map`, `do_map` — render ASCII map of surrounding rooms

#### graph.c — Pathfinding helpers (50 C lines)
`bfs_enqueue`, `bfs_dequeue`, `bfs_clear_queue`, `is_intelligent`

#### scripts.c — Lua table operations (33 C lines)
`table_to_char`, `table_to_obj`, `table_to_room`, `char_to_table`, `obj_to_table`, `room_to_table`, `clear_stack`

#### boards.c — Board persistence (19 C lines)
`Board_save_board`, `Board_load_board`, `Board_reset_board`, `Board_write_message`

#### weather.c — Weather effects (9 C lines)
`weather_change`, `another_hour`

---

## NOT needed (Go handles differently)
- **OLC editors** (oedit, redit, zedit, medit, sedit) — separate tooling
- **String utilities** (str_dup, str_cmp, etc.) — Go stdlib
- **Random numbers** (prng_*) — Go math/rand
- **Networking primitives** (init_socket, new_descriptor, etc.) — Go net package
- **Logging** (mudlog, basic_mud_log) — Go slog
- **File editors** (improved-edit, file-edit) — different UX pattern

---

## Estimated effort
| Priority | C Lines | Go Lines | Sessions |
|----------|---------|----------|----------|
| P1: Core gameplay | ~480 | ~340-480 | 1-2 |
| P2: Persistence | ~750 | ~525-750 | 1-2 |
| P3: Commands | ~550 | ~385-550 | 1 |
| P4: Polish | ~350 | ~245-350 | 1 |
| **Total** | **~2,130** | **~1,500-2,130** | **3-6** |

The architecture is solid. The interfaces are clean. This is straightforward implementation work — no redesign needed.
