# Go-only string triage (the R4 review queue)

Every go-only segment in [`go-only.tsv`](go-only.tsv) classified, each with evidence a
reviewer can check in a minute. Generated from the census at `592abf00f`. Segments are the
key, not file:line, because lines move.

Reproduce the queue with `make string-census-update`; the ratchet baseline
[`go-only-baseline.json`](go-only-baseline.json) carries the reason and the evidence per row
(`internal/stringcensus/baseline_vocabulary_test.go` rejects a row that lacks them and keeps
the counts below in step with the baseline).

| reason | count |
|---|---|
| `bug:invented` | 64 |
| `bug:paraphrase` | 56 |
| `census:composed` | 2 |
| `data` | 0 |
| `surface:no-c` | 2 |
| `unsure` | 0 |
| **total** | **124** |

`unsure` is 0/124 (0.0%).

**This document classifies; it changes no game code.** Fix work is Claude's.

## Method, and how to re-check a row

1. `make string-census-update` produces `go-only.tsv` (the queue) and the baseline.
2. For each segment, the Go call site was read in full, then the C oracle was searched for the
   equivalent point (the command, spec proc, spell or event the Go code names) with `grep -rn` on
   `src/`; `lib/` was searched for data-driven text.
3. Every C citation was checked mechanically against the cited line, and every `nothing` claim against
   the whole C tree (`grep -i` for the segment's distinctive words). A row that failed either check was
   re-researched.

Two conventions keep the classes consistent:

- `bug:paraphrase` when C has a line at that slot in the same flow and the Go line re-words it;
  `bug:invented` when C's flow has no line there at all (a Go-only branch, command or feature), or C
  prints a line unrelated to the Go one. Both mean the same fix: emit C's bytes at the cited site, or
  delete the invented branch.
- `surface:no-c` only when **no telnet player can read the bytes** (agent/WebSocket sessions, webOLC,
  admin API, GMCP, TLS). A Go-only telnet command is *not* this class: its strings are `bug:invented`
  with the command's divergence named in the note, because R2 makes the command surface part of the
  game.

**Census blind spots found while triaging** (out of scope here, brief 07's tool owns them). Two kinds of
C player text are invisible to the matcher, and several rows below are them:

1. **String constants** such as `OK`, `NOPERSON` and `NOEFFECT` (`src/config.c:92-94`) are passed to
   `send_to_char` but live outside the census's table-file list (`constants.c`, `class.c`).
2. **`#define`d literals** such as `SUMMON_FAIL "You failed.
"` (`src/spells.c:216`) are skipped by the
   C lexer as preprocessor lines.
3. **A package-level twin of a method sink**: `actToRoom(w, roomVNum, format, exclude)` exists both as a
   package function (`pkg/game/other_helpers.go:71`) and as a `*World` method, but the sink table lists
   only the method, so literals passed to the function form (`pkg/game/graph.go:317`, `"%s opens the
   door."`) are invisible. Fix is one word in `internal/stringcensus/gosink.go` (`Kind: goSinkBoth`);
   reported rather than fixed here, because widening the census adds rows that would need their own
   triage.

## Root causes for the fix work

Groups that can each be fixed in one pass. They overlap — these are passes of work, not a partition of the rows. Indexes are the row numbers in the sections below (sorted go-only text, same order as `go-only.tsv`).

### Go-only command names (R2 command surface) (24 rows)

Indexes: 5, 6, 7, 8, 18, 19, 20, 32, 34, 35, 37, 44, 48, 58, 63, 64, 65, 66, 67, 71, 74, 84, 119, 120

The port registers commands C's cmd_info does not have, or under a name C reserves for something else: `password`, `heal` (C: restore), `vis` (C: visible), `broadcast` (C: gecho), `poofset` (C: poofin/poofout), `summon`, `autoloot` (C: auto loot), `channel`, `ignore`/`unignore`, and the guest login. Several are already documented as deliberate in pkg/session/command_gates.tsv (`heal`, `summon`, `vis`, `poofset`, `autoloot`, `password`); make reachability tracks the same surface. Their message strings have no C bytes to match, so every one lands here. Fix work is a policy call: either port C's text and name, or record the addition as an approved carve-out.

### Dead duplicate movement stubs (4 rows)

Indexes: 56, 77, 89, 109

cmdFleeMovement (pkg/session/movement_cmds.go) is an unregistered copy of do_flee, and the sneak stub beside it is superseded by the skill command. The live paths already print C's bytes (combat_cmds.go:406 for the PANIC line). Fix work is deletion, not re-wording.

### Combat damage numbers (3 rows)

Indexes: 107, 114, 115

pkg/combat/engine.go:971-974 and pkg/game/mob.go:360-370 print `You hit %s for %d damage!` / `%s hits you for %d damage!` / `%s attacks you for %d damage!`, and mob.go hardcodes `damage := 10`. C has no numeric damage text at all: damage() goes through skill_message()/dam_message() (src/fight.c:1524-1532), which send act() lines from the fight_messages table (src/fight.c:1033-1036). mob.go's Attack is a stub with no C call path.

### Weather and gate broadcasts (5 rows)

Indexes: 101, 102, 103, 104, 106

pkg/game/weather.go:570-635 broadcasts ghost-ship, night-gate and full-moon events globally, and spec_procs_missing.go:198 the moon-gate arrival. C has the ghost ship and the gates but tells specific rooms, with different bytes: `Suddenly a ghostly ship appears to the north!` (src/new_cmds.c:2703), `A shimmering portal of blue light suddenly appears in ...!` (src/gate.c:191), `The shimmering blue portal of light ... fades out of existence.` (src/gate.c:226), `With a sound like a raging river, a hole in space opens up ...` (src/gate.c:283). The full-moon broadcast has no C counterpart (src/new_cmds.c:1322 only force-transforms flagged vampires). So each row is a scope-and-wording divergence rather than a pure invention, except the full moon.

### spec_procs2 acted dialogue (10 rows)

Indexes: 2, 4, 43, 49, 50, 52, 86, 87, 123, 124

Ten strings in pkg/game/spec_procs2.go (bat, couch/mimic, boy, portals, key trade, guard refusals). Each needs its own C comparison, but they are one pass of work: src/spec_procs2.c holds the originals and several are reworded rather than invented.

### spec_procs_missing (port-only spec procs) (6 rows)

Indexes: 46, 47, 53, 80, 106, 111

pkg/game/spec_procs_missing.go implements the recharger, beholder and moon-gate behaviour. C's recharger is SPECIAL(recharger) in src/new_cmds2.c:415 (dialogue and price lines quoted in the rows); the moon gates live in src/gate.c. The bytes differ throughout.

### Port diagnostics and nil-guards (immortal-visible) (24 rows)

Indexes: 9, 10, 11, 15, 16, 17, 21, 25, 27, 28, 29, 33, 36, 38, 40, 41, 42, 45, 59, 60, 62, 69, 97, 105

Messages the port added around its own storage, world-parser and session plumbing: DB and file errors (`Error reading house file.`, `Player lookup not available.`), unavailable-parse guards (`No parsed world available.`), and wizard-diagnostic usage lines. C reads world data directly and has no equivalent branches, so these are R4 candidates rather than porting work; several sit on commands the port already documents as Go-only.

### Autoloot (4 rows)

Indexes: 6, 7, 78, 79

The `auto loot` toggle and the death-time loot lines. C's toggle bytes are `You will now automatically loot corpses.` / `You will no longer loot corpses.` (src/act.other.c:1363,1367); at a kill C calls do_get (`all corpse`), whose own per-item lines are `You get $p from $P.` (src/act.item.c:208) and `$p seems to be empty.` (src/act.item.c:259), which the Go autoloot lines re-word. C's wording, Go's bytes.


## bug rows, grouped by C file and function

One line each: Go site, the Go text, the C text (`nothing` when C prints nothing comparable), the C site.

### `lib/world/shp/42.shp` — `(no function)` (2)

- `bug:paraphrase` — Go `pkg/session/shop_cmds.go:407` — “The shopkeeper doesn't want that.” — C lib/world/shp/42.shp:14: "%s Hmmm, nah, I don't want one of those.~" — C tells the shop's configured do_not_buy line (lib/world/shp/42.shp:14), not a fixed sentence; C prints the shop's do_not_buy line via src/shop.c:616 + do_tell, so the bytes are data-driven, not the fixed string Go prints
- `bug:paraphrase` — Go `pkg/session/shop_cmds.go:365, pkg/session/shop_cmds.go:367, pkg/session/shop_cmds.go:417, pkg/session/shop_cmds.go:458, pkg/session/shop_cmds.go:460` — “gold pieces.” — C lib/world/shp/42.shp:18: "%s I'll give you %d coins for that.~" — C's shop tells the keeper's coins line (lib/world/shp/42.shp:18); Go says "gold pieces"; C prints the shop's message_sell line via src/shop.c:800 + do_tell; Go says "gold pieces" where the data and C say "coins", at five sites

### `src/act.comm.c` — `do_gen_comm` (1)

- `bug:invented` — Go `pkg/game/comm_channel.go:97` — “Unknown channel.” — C src/act.comm.c:1146: nothing — C do_gen_comm takes subcmd from cmd_info; no unknown-channel text (cmdChannel callers pass fixed names)

### `src/act.comm.c` — `do_qcomm` (1)

- `bug:paraphrase` — Go `pkg/session/act_comm.go:42` — “What is your question?” — C src/act.comm.c:1321: "%s?  Yes, fine, %s we must, but WHAT??\r\n" — Go already has qcommEmptyMsg (act_comm.go:139) for these bytes; cmdQcomm prints its own prompt

### `src/act.comm.c` — `do_tell` (2)

- `bug:invented` — Go `pkg/session/cmd_group.go:315, pkg/session/comm_cmds.go:120, pkg/session/comm_cmds.go:155, pkg/session/comm_cmds.go:364, pkg/session/comm_cmds.go:42, pkg/session/comm_cmds.go:57, pkg/session/comm_cmds.go:79` — “Your message was blocked.” — C src/act.comm.c:901: nothing — C do_tell has no word-filter gate; Go's moderation manager (pkg/moderation) is new
- `bug:invented` — Go `pkg/game/directed_speech.go:249, pkg/game/directed_speech.go:307` — “is ignoring you.” — C src/act.comm.c:901: nothing — C do_tell has no ignore list; Go-only ignore feature (commands.go:451)

### `src/act.comm.c` — `do_think` (1)

- `bug:paraphrase` — Go `pkg/game/comm_channel.go:223` — “You think: '” — C src/act.comm.c:1380: "You think . o O ( %s )\n\r" — do_think's self line; Go prints "You think: '%s'"

### `src/act.comm.c` — `do_write` (1)

- `bug:invented` — Go `pkg/session/comm_cmds.go:256` — “You write '” — C src/act.comm.c:1097: "Write your note.  End with '@' on a new line.\r\n" — C enters the string editor with this prompt; Go echoes the written text inline

### `src/act.informative.c` — `do_color` (1)

- `bug:paraphrase` — Go `pkg/session/act_informative.go:40` — “Your color is now” — C src/act.informative.c:2494: "Your %scolor%s is now %s.\r\n" — C wraps "color" in CCRED/CCNRM escapes; Go drops them

### `src/act.informative.c` — `do_gen_ps` (2)

- `bug:paraphrase` — Go `pkg/session/gen_ps_cmds.go:245` — “Built with:” — C src/act.informative.c:2173: "Compile Time: %s\r\n" — same immort-only build-info slot; Go swaps label and datum for the Go runtime version
- `bug:invented` — Go `pkg/session/gen_ps_cmds.go:132` — “That information is not available right now.” — C src/act.informative.c:2137: nothing — C do_gen_ps pages the boot-loaded file; an unreadable file prints nothing

### `src/act.item.c` — `get_from_container` (1)

- `bug:paraphrase` — Go `pkg/game/death.go:455` — “You autoloot the corpse but find nothing of value.” — C src/act.item.c:259: "$p seems to be empty." — C autoloot runs do_get all corpse; its empty-container line reworded

### `src/act.item.c` — `perform_drop` (1)

- `bug:invented` — Go `pkg/game/item_donate.go:38` — “Something went wrong. Your item was not donated.” — C src/act.item.c:510: nothing — C perform_drop moves the item unconditionally; no failure line

### `src/act.item.c` — `perform_drop_gold` (1)

- `bug:invented` — Go `pkg/game/item_donate.go:79` — “Something went wrong. Your gold was not donated.” — C src/act.item.c:453: nothing — C perform_drop_gold moves the coins unconditionally; no failure line

### `src/act.item.c` — `perform_get_from_container` (1)

- `bug:paraphrase` — Go `pkg/game/death.go:453` — “You autoloot:” — C src/act.item.c:208: "You get $p from $P." — C autoloot prints this per item via do_get; Go says "You autoloot: <names>."

### `src/act.movement.c` — `(no function)` (1)

- `bug:paraphrase` — Go `pkg/game/ai.go:172, pkg/game/graph.go:343` — “has arrived.” — C src/act.movement.c:238: "$n arrives from the %s." — C do_move direction-specific arrival line; Go substitutes a generic has-arrived line

### `src/act.movement.c` — `do_gen_door` (1)

- `bug:invented` — Go `pkg/session/door_cmds.go:43` — “Knock on what? Try north, south, east, west, up, or down.” — C src/act.movement.c:607: "%s what?\r\n" — No knock verb in C cmd_info; do_gen_door's no-argument branch prints "<Verb> what?"; do_gen_door's empty-argument branch; the "north, south, east, west" hint list is Go-only

### `src/act.offensive.c` — `do_flee` (3)

- `bug:paraphrase` — Go `pkg/session/movement_cmds.go:68` — “There's nowhere to flee!” — C src/act.offensive.c:415: "PANIC!  You couldn't escape!\r\n" — C's no-exit branch of do_flee prints the PANIC line; Go rewrote it; unreachable: cmdFleeMovement is not registered (only cmdFlee is, commands.go:94); the live path already prints C's PANIC line at combat_cmds.go:406
- `bug:invented` — Go `pkg/session/movement_cmds.go:50` — “You're not fighting anyone!” — C src/act.offensive.c:365: nothing — C do_flee has no fighting gate and prints nothing; string lives in an unregistered stub; unreachable: same unregistered cmdFleeMovement stub; the live flee path is cmdFlee (pkg/session/combat_cmds.go:335)
- `bug:invented` — Go `pkg/session/movement_cmds.go:122` — “experience points for fleeing.” — C src/act.offensive.c:405: nothing — C do_flee calls gain_exp silently; C has no flee XP-loss message; unreachable: same unregistered cmdFleeMovement stub

### `src/act.other.c` — `do_auto` (2)

- `bug:paraphrase` — Go `pkg/session/autoloot.go:27` — “Auto-loot disabled.” — C src/act.other.c:1363: "You will no longer loot corpses.\r\n" — C toggle loot-off line; Go's port-only autoloot command rewords it
- `bug:paraphrase` — Go `pkg/session/autoloot.go:25` — “Auto-loot enabled.” — C src/act.other.c:1368: "You will now automatically loot corpses.\r\n" — C toggle loot-on line; Go's port-only autoloot command rewords it

### `src/act.other.c` — `do_gen_tog` (1)

- `bug:invented` — Go `pkg/game/other_settings.go:249, pkg/game/other_settings.go:279` — “Unknown toggle.” — C src/act.other.c:1142: nothing — C's toggles are separate cmd_info entries into do_gen_tog; no unknown-subcmd text

### `src/act.other.c` — `do_save` (1)

- `bug:invented` — Go `pkg/game/other_session.go:15` — “Could not save your data. Contact an admin!” — C src/act.other.c:198: nothing — C do_save calls save_char with no failure text; Go adds a DB-failure line

### `src/act.other.c` — `do_sneak` (1)

- `bug:paraphrase` — Go `pkg/session/movement_cmds.go:171` — “You attempt to move silently.” — C src/act.other.c:225: "Okay, you'll try to move silently for a while.\r\n" — Unwired movement stub rewrites C do_sneak's line; unreachable stub: the live sneaking path is the skill command; C's line is act.other.c:225

### `src/act.other.c` — `do_use` (2)

- `bug:invented` — Go `pkg/session/use_cmds.go:154` — “You aren't holding that.” — C src/act.other.c:933: "You don't seem to have %s %s.\r\n" — No zap verb in C; do_use's not-found branch says this ("holding" variant commented out at :939)
- `bug:invented` — Go `pkg/session/use_cmds.go:133` — “Zap who?” — C src/act.other.c:903: "What do you want to %s?\r\n" — No zap verb in C; do_use's no-argument branch prints this with CMD_NAME

### `src/act.other.c` — `print_group` (1)

- `bug:invented` — Go `pkg/session/cmd_group.go:197` — “Your group leader is not online.” — C src/act.other.c:649: nothing — C print_group uses ch->master directly; no online check and no message

### `src/act.wizard.c` — `ACMD` (2)

- `bug:paraphrase` — Go `pkg/session/wiz_zone.go:187` — “No objects found.” — C src/act.wizard.c:3329: "Sorry, there are no objs in that zone.\r\n" — C do_olist's no-results line; Go rewords it
- `bug:invented` — Go `pkg/session/wiz_zone.go:169` — “Usage: olist <keyword>” — C src/act.wizard.c:3304: nothing — C do_olist has no usage line; its argument is a zone number, not a keyword

### `src/act.wizard.c` — `do_gecho` (1)

- `bug:invented` — Go `pkg/session/wiz_system.go:757` — “Broadcast what?” — C src/act.wizard.c:1697: "That must be a mistake...\r\n" — C has no broadcast command; its global-message cmd do_gecho prints this for an empty argument

### `src/act.wizard.c` — `do_invis` (1)

- `bug:invented` — Go `pkg/session/wiz_player.go:234` — “Vis whom?” — C src/act.wizard.c:1663: nothing — C vis/invis paths are self-only and take no player argument

### `src/act.wizard.c` — `do_restore` (4)

- `bug:paraphrase` — Go `pkg/session/wiz_player.go:20` — “Heal whom?” — C src/act.wizard.c:1591: "Whom do you wish to restore?\r\n" — Go heal duplicates do_restore (restores vitals) with a reworded no-argument prompt
- `bug:paraphrase` — Go `pkg/session/wiz_player.go:32` — “has healed you!” — C src/act.wizard.c:1616: "The hand of $N touches you, healing your wounds and leaving you " — C do_restore's victim line (adjacent literals), reworded
- `bug:paraphrase` — Go `pkg/session/wiz_player.go:249` — “is already visible.” — C src/act.wizard.c:1629: "You are already fully visible.\r\n" — C perform_immort_vis self line; Go reports it about a target instead
- `bug:paraphrase` — Go `pkg/session/wiz_player.go:247` — “is now visible to mortals.” — C src/act.wizard.c:1635: "You are now fully visible.\r\n" — C perform_immort_vis success line; Go says visible to mortals

### `src/act.wizard.c` — `do_rlist` (1)

- `bug:invented` — Go `pkg/session/wiz_zone.go:124, pkg/session/wiz_zone.go:165, pkg/session/wiz_zone.go:205` — “No parsed world available.” — C src/act.wizard.c:3336: nothing — Go nil-parsed-world guard in rlist/olist/mlist; C reads world data directly

### `src/act.wizard.c` — `do_shutdown` (1)

- `bug:invented` — Go `pkg/session/manager.go:1942` — “!! The MUD server is performing a graceful shutdown for maintenance. Your state has been saved. !!” — C src/act.wizard.c:1099: "Shutting down for maintenance.\r\n" — OS SIGINT/SIGTERM path cmd/server/main.go:874,971; in-game shutdown uses C do_shutdown's own bytes; trigger is the OS SIGINT/SIGTERM path, but a telnet player reads the bytes, so R4 applies; C's only shutdown text is the in-game do_shutdown line quoted here

### `src/act.wizard.c` — `do_vnum` (3)

- `bug:invented` — Go `pkg/session/wiz_stats.go:670` — “Parsed world data not available.” — C src/act.wizard.c:395: nothing — Go nil-parsed-world guard in vnum; C do_vnum reads mob/obj protos directly
- `bug:invented` — Go `pkg/session/wiz_stats.go:359` — “Room data not found.” — C src/act.wizard.c:422: "Room name: %s%s%s\r\n" — C do_stat_room prints the report unconditionally; no room-not-found branch
- `bug:invented` — Go `pkg/session/wiz_stats.go:353, pkg/session/wiz_stats.go:485, pkg/session/wiz_stats.go:664` — “World not available.” — C src/act.wizard.c:422: "Room name: %s%s%s\r\n" — Go nil-world guards in stat room/obj and vnum; C reads world[] and prints the report

### `src/act.wizard.c` — `do_wizlock` (1)

- `bug:invented` — Go `pkg/session/wiz_system.go:213` — “Cannot access manager state.” — C src/act.wizard.c:1788: "The game is %s completely open.\r\n" — C do_wizlock has no nil-manager guard; it prints the wizlock status line

### `src/act.wizard.c` — `do_wizutil` (3)

- `bug:invented` — Go `pkg/session/wiz_system.go:463` — “Unknown sub-command. Options: reroll, pardon, notitle, squelch, freeze, thaw, unaffect” — C src/act.wizard.c:2214: nothing — C has no wizutil command; the unknown-subcmd branch only logs, no player text
- `bug:paraphrase` — Go `pkg/session/wiz_system.go:447` — “Usage: reroll|pardon|notitle|squelch|freeze|thaw|unaffect <player>” — C src/act.wizard.c:2091: "Yes, but for whom?!?\r\n" — C do_wizutil's missing-argument line for reroll/pardon/notitle/etc.
- `bug:paraphrase` — Go `pkg/session/wiz_system.go:640` — “Usage: unaffect <player>” — C src/act.wizard.c:2091: "Yes, but for whom?!?\r\n" — C unaffect with no target prints this; Go prints a usage line instead

### `src/clan.c` — `do_clan_demote` (1)

- `bug:paraphrase` — Go `pkg/game/clan_membership.go:253` — “You've been demoted within your clan!” — C src/clan.c:487: "You've demoted within your clan!\r\n" — C omits "been"; Go silently corrects the typo

### `src/clan.c` — `do_clan_status` (1)

- `bug:invented` — Go `pkg/game/clan_info.go:33` — “You are Rank” — C src/clan.c:677: "You are %s (Rank %d) of %s\r\n" — rank>0 with no clan: C indexes clan[-1] (UB); "of a clan" line has no C source

### `src/comm.c` — `send_to_zone` (1)

- `bug:paraphrase` — Go `pkg/session/wiz_system.go:780` — “[Broadcast]” — C src/comm.c:2545: "&RBroadcast: %s&n" — C's broadcast prefix is red Broadcast: ; Go writes [Broadcast]

### `src/config.c` — `(no function)` (3)

- `bug:paraphrase` — Go `pkg/session/cmd_info.go:1067, pkg/session/wiz_player.go:26` — “No one by that name online.” — C src/config.c:93: "No-one by that name here.\r\n" — NOPERSON, C's wizard target-not-found line, reworded
- `bug:invented` — Go `pkg/session/wiz_info.go:580, pkg/session/wiz_info.go:585` — “Usage: poofset <in|out> [message]” — C src/config.c:92: "Okay.\r\n" — no C poofset command; do_poofset (poofin/poofout) only replies OK
- `bug:invented` — Go `pkg/session/wiz_player.go:31` — “You heal” — C src/config.c:92: "Okay.\r\n" — C do_restore replies OK to the healer; Go adds a You heal <name>. line instead

### `src/fight.c` — `dam_message` (3)

- `bug:paraphrase` — Go `pkg/game/mob.go:364` — “attacks you for” — C src/fight.c:931: "$n #W you hard." — C dam_message victim line for 7-10 damage, #W from attack type; Go prints a damage count
- `bug:invented` — Go `pkg/combat/engine.go:971` — “hits you for” — C src/fight.c:925: "$n #W you." — C dam_message uses its weapon-verb table, never for-N-damage; MessageFunc already wired (main.go:561); C's damage text is the dam_message verb table (src/fight.c:919-960), banded by damage; it never prints a damage number, which is the invented part
- `bug:paraphrase` — Go `pkg/game/damage_stubs.go:256` — “hits you, but it doesn't hurt!” — C src/fight.c:907: "$n tries to #w you, but misses." — C dam_message dam==0 victim line; Go re-words it as a hit that does not hurt

### `src/fight.c` — `damage` (1)

- `bug:paraphrase` — Go `pkg/combat/fight_core.go:180, pkg/combat/fight_core.go:494` — “You are incapacitated and will slowly die, if not aided.” — C src/fight.c:1572: "You are incapacitated an will slowly die, if not aided.\r\n" — C's POS_INCAP line (two adjacent literals) has typo "an will"; Go corrects it

### `src/fight.c` — `make_dust` (1)

- `bug:invented` — Go `pkg/game/death.go:1194` — “is disintegrated! Equipment lies scattered on the ground.” — C src/fight.c:433: nothing — C make_dust only moves gear and loads dust; there is no disintegrate room message

### `src/gate.c` — `SPECIAL` (1)

- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:198` — “arrives through a shimmering moon gate!” — C src/gate.c:283: "With a sound like a raging river, a hole in space opens up " — C continues and $n steps out.; Go re-words the gate arrival message

### `src/gate.c` — `load_night_gate` (1)

- `bug:paraphrase` — Go `pkg/game/weather.go:596` — “[ NIGHT GATE ] A shimmering gate materializes in the darkness...” — C src/gate.c:191: "A shimmering portal of blue light suddenly appears in " — C continues the darkness!; Go re-words it as a global [NIGHT GATE] broadcast

### `src/gate.c` — `remove_night_gate` (1)

- `bug:paraphrase` — Go `pkg/game/weather.go:609` — “[ NIGHT GATE ] The shimmering gate fades into nothingness.” — C src/gate.c:226: "The shimmering blue portal of light " — C continues fades out of existence.; Go says the gate fades into nothingness

### `src/handler.c` — `check_for_bad_stats` (2)

- `bug:paraphrase` — Go `pkg/game/char_mgmt.go:59` — “Your light source flickers and sputters.” — C src/handler.c:1057: "Your light begins to flicker and fade." — update_char_objects' value==1 line; Go rewrites it
- `bug:paraphrase` — Go `pkg/game/char_mgmt.go:61` — “Your light source has gone out.” — C src/handler.c:1062: "Your light sputters out and dies." — update_char_objects' value==0 line; Go rewrites it

### `src/house.c` — `(no function)` (3)

- `bug:invented` — Go `pkg/game/house_rent.go:23, pkg/game/house_rent.go:49` — “Error reading house file.” — C src/house.c:215: nothing — C House_listrent returns silently on ferror; no read-error text
- `bug:invented` — Go `pkg/game/house_rent.go:13` — “Invalid house vnum.” — C src/house.c:205: nothing — C House_get_filename failure returns silently; no invalid-vnum text
- `bug:invented` — Go `pkg/game/house_rent.go:46` — “Objects stored for house #” — C src/house.c:220: nothing — C sends only concatenated per-item lines; it has no Objects-stored header

### `src/house.c` — `do_house` (1)

- `bug:invented` — Go `pkg/game/house_player.go:148, pkg/game/house_player.go:61` — “Player lookup not available.” — C src/house.c:652: nothing — C get_id_by_name always answers; the unwired-lookup branch is port-only

### `src/interpreter.c` — `do_alias` (1)

- `bug:invented` — Go `pkg/session/act_social.go:104, pkg/session/act_social.go:61` — “Error saving aliases.” — C src/interpreter.c:1013: nothing — C do_alias prints only "Alias deleted."/"Alias added."; alias save errors are SYSERR-log only

### `src/interpreter.c` — `do_zreset` (7)

- `bug:invented` — Go `pkg/session/commands.go:697` — “Guest accounts are restricted from using this command.” — C src/interpreter.c:310: nothing — Guest login is port-only (session_login.go:84); C cmd_info has no account gate
- `bug:invented` — Go `pkg/session/cmd_account.go:19` — “Usage: password <old> <new>” — C src/interpreter.c:916: "Huh?!?\r\n" — No password verb in C cmd_info; the unknown command answers this
- `bug:invented` — Go `pkg/session/comm_cmds.go:393` — “You are ignoring:” — C src/interpreter.c:310: nothing — No ignore verb in C cmd_info; Go's player ignore list has no C lineage
- `bug:invented` — Go `pkg/session/affects_informative.go:16` — “You are not affected by any spells.” — C src/interpreter.c:310: nothing — No affects verb in C cmd_info (do_abils is stats); comment cites a nonexistent do_affects
- `bug:invented` — Go `pkg/session/comm_cmds.go:390` — “You are not ignoring anyone.” — C src/interpreter.c:310: nothing — No ignore verb in C cmd_info; Go-only ignore list
- `bug:invented` — Go `pkg/session/comm_cmds.go:408` — “is no longer ignored.” — C src/interpreter.c:310: nothing — No ignore verb in C cmd_info; Go-only ignore toggle
- `bug:invented` — Go `pkg/session/comm_cmds.go:411` — “is now ignored.” — C src/interpreter.c:310: nothing — No ignore verb in C cmd_info; Go-only ignore toggle

### `src/interpreter.c` — `nanny` (7)

- `bug:invented` — Go `pkg/session/cmd_account.go:45, pkg/session/cmd_account.go:65` — “An error occurred. Please try again later.” — C src/interpreter.c:2291: nothing — C nanny() already holds the loaded character; no DB load step and no error text
- `bug:invented` — Go `pkg/session/cmd_account.go:72` — “Failed to save new password. Please try again later.” — C src/interpreter.c:1983: nothing — C save_char after a password change has no failure text
- `bug:paraphrase` — Go `pkg/session/cmd_account.go:56` — “Old password is incorrect.” — C src/interpreter.c:2294: "\r\nIncorrect password.\r\n" — C nanny()'s wrong-old-password reply; Go rewords it
- `bug:paraphrase` — Go `pkg/session/cmd_account.go:76` — “Password changed successfully.” — C src/interpreter.c:1985: "\r\nDone.\n\r" — C's password-change completion line; Go rewords it
- `bug:paraphrase` — Go `pkg/session/cmd_account.go:37` — “Password is too long (max 72 characters).” — C src/interpreter.c:1946: "\r\nIllegal password.\r\n" — C rejects an over-long new password; its cap is MAX_PWD_LENGTH 10, not 72
- `bug:invented` — Go `pkg/session/cmd_account.go:49` — “Player record not found.” — C src/interpreter.c:2291: nothing — C has already loaded the character; no record-not-found branch at this step
- `bug:paraphrase` — Go `pkg/session/cmd_account.go:27` — “That's the same as your old password!” — C src/interpreter.c:1946: "\r\nIllegal password.\r\n" — C rejects a new password equal to the character name; Go rejects equal-to-old

### `src/modify.c` — `string_write` (9)

- `bug:paraphrase` — Go `pkg/session/menu.go:212` — “Description is too long; type @ or /s to save, /a to abort.” — C src/modify.c:147: "String too long.  Last line skipped.\r\n" — C string_add's over-length notice in the CON_EXDESC editor; Go rewords it
- `bug:paraphrase` — Go `pkg/session/menu.go:185` — “Description not changed.” — C src/modify.c:245: "Description aborted.\r\n" — C exdesc cleanup's abort line; Go rewords it as not changed
- `bug:invented` — Go `pkg/session/menu.go:205` — “Description saved.” — C src/modify.c:247: nothing — C exdesc cleanup on save writes only MENU; there is no confirmation line
- `bug:paraphrase` — Go `pkg/game/mail.go:594` — “Mail aborted (empty message).” — C src/modify.c:225: "Mail aborted.\r\n" — playing_string_cleanup's mail-abort line; Go appends "(empty message)"
- `bug:paraphrase` — Go `pkg/game/mail.go:592` — “Mail sent.” — C src/modify.c:223: "Message sent!\r\n" — C's mail-editor save line; Go says "Mail sent."
- `bug:paraphrase` — Go `pkg/game/note_write.go:86` — “Note limit reached. Type '@' on a new line to save.” — C src/modify.c:147: "String too long.  Last line skipped.\r\n" — C's note-editor over-length line; Go rewords it, calling C silent
- `bug:invented` — Go `pkg/game/note_write.go:67` — “Note recorded.” — C src/modify.c:218: nothing — C playing_string_cleanup speaks only for mail; a finished note prints nothing
- `bug:invented` — Go `pkg/session/menu.go:198` — “Unable to save description.” — C src/modify.c:240: nothing — C keeps the description in memory; no save-failure branch here
- `bug:invented` — Go `pkg/game/note_write.go:58` — “Your note was lost. (internal error)” — C src/modify.c:173: nothing — C's unknown-origin write abort is a SYSERR log, never player text

### `src/new_cmds.c` — `do_whois` (1)

- `bug:invented` — Go `pkg/session/cmd_info.go:1200` — “Error looking up player.” — C src/new_cmds.c:1415: "There is no such player.\r\n" — C do_whois loads the save file and prints this; the DB-error line is Go-only

### `src/new_cmds.c` — `full_moon` (1)

- `bug:invented` — Go `pkg/game/weather.go:570` — “[ FULL MOON RISES ] The full moon casts an eerie glow across the land.” — C src/new_cmds.c:1322: "The lunar light infuses your body, forcing you to " — C full_moon only force-transforms flagged vampires/werewolves; the global broadcast is Go-only

### `src/new_cmds.c` — `ghost_ship_appear` (1)

- `bug:paraphrase` — Go `pkg/game/weather.go:622` — “[ GHOST SHIP ] An eerie fog rolls in from the harbor... the ghost ship has been sighted!” — C src/new_cmds.c:2703: "Suddenly a ghostly ship appears to the north!\r\n" — C sends this to the dock room; Go re-words it as a global fog sighting

### `src/new_cmds2.c` — `SPECIAL` (5)

- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:83` — “That is not a wand or staff.” — C src/new_cmds2.c:455: "$n tells you, 'Ummm... does that look like a wand or staff to you?'" — C recharger sends a $n-prefixed tell; Go prints a bare descriptive line
- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:92` — “That item is already fully charged.” — C src/new_cmds2.c:481: "$n tells you, 'The item does not need recharging, stop wasting " — C refusal continues "my time!"; Go re-words it as already fully charged
- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:120` — “The recharger works their magic on” — C src/new_cmds2.c:476: "The item now has %d charges remaining.\r\n" — C sends only the charge count here; Go prepends invented works-their-magic/cost text
- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:128` — “You feel your magic dissipate in the beholder's presence!” — C src/new_cmds2.c:338: "A ray of light shoots out from one of $n's eyestalks, breaking" — C beholder breaks the caster's concentration; Go re-words the spell-block line
- `bug:paraphrase` — Go `pkg/game/spec_procs_missing.go:104` — “gold to recharge that, which you cannot afford.” — C src/new_cmds2.c:462: "$n tells you, 'You don't have enough gold!'" — C recharger affordability tell; Go re-words it as a computed cost you cannot afford

### `src/shop.c` — `get_selling_obj` (1)

- `bug:invented` — Go `pkg/session/shop_cmds.go:453` — “The shopkeeper doesn't want anything you have.” — C src/shop.c:616: nothing — C has no sell-all; its per-item refusal is the shop's data do_not_buy line told by the keeper

### `src/shop.c` — `shopping_sell` (1)

- `bug:paraphrase` — Go `pkg/session/shop_cmds.go:387` — “Sell what?” — C src/shop.c:753: "%s What do you want to sell??" — C tells "<name> What do you want to sell??" via the keeper; Go prints a bare fixed prompt

### `src/spec_procs.c` — `SPECIAL` (1)

- `bug:invented` — Go `pkg/game/spec_procs4.go:246` — “Something went wrong.” — C src/spec_procs.c:1901: "May you enjoy your pet.\r\n" — C pet_shops buys the pet unconditionally; the SpawnMob failure branch is Go-only

### `src/spec_procs2.c` — `SPECIAL` (11)

- `bug:paraphrase` — Go `pkg/game/spec_procs2.go:1935` — “A bat swoops down and attacks you!” — C src/spec_procs2.c:1780: "$n swoops down, narrowly missing your face." — C bat proc emotes this on random pulses; Go re-words it into an attack on look dripping
- `bug:paraphrase` — Go `pkg/game/spec_procs2.go:1779` — “A shimmering portal appears and sucks you in!” — C src/spec_procs2.c:1665: "\r\nWith a blinding flash of light and a crack of thunder, you are" — C portal_room tells the victim this on teleport; Go re-words it and teleports on movement
- `bug:paraphrase` — Go `pkg/game/spec_procs2.go:296` — “Starved and needing food to make more pillows, the couch attacks you!” — C src/spec_procs2.c:304: "Starved and needing food to make more pillows, the " — C says the mimic attacks you with \n\r\n\r; Go says couch and \r\n\r\n
- `bug:invented` — Go `pkg/game/spec_procs2.go:1913` — “The bats swarm around you, blocking your escape!” — C src/spec_procs2.c:1756: nothing — C bat_room only wakes sleeping bats (Your movements wake up $n!); no blocking branch
- `bug:invented` — Go `pkg/game/spec_procs2.go:1504` — “The boy smiles and hands you a small note.” — C src/spec_procs2.c:1398: "$n exclaims, 'Don't let the bad man hurt me, mommy!" — C little_boy only exclaims this at random; the give-flower-for-note exchange is Go-only
- `bug:invented` — Go `pkg/game/spec_procs2.go:1512` — “The little boy runs off!” — C src/spec_procs2.c:1398: "$n exclaims, 'Don't let the bad man hurt me, mommy!" — C little_boy never gives a note or leaves; the runs-off line is Go-only
- `bug:invented` — Go `pkg/game/spec_procs2.go:1416` — “You trade your key for” — C src/spec_procs2.c:1171: "%s eneeswseswseswseswsesw" — C eviltrade only answers a give of gold watch 13111 with this; Go invents trade/exp
- `bug:invented` — Go `pkg/game/spec_procs2.go:1789` — “You tumble out into a strange place...” — C src/spec_procs2.c:1672: "\r\nWith a blinding flash of light and a crack of thunder, $n " — C tells only the room $n appears and gives the victim do_look; tumble line is Go-only
- `bug:paraphrase` — Go `pkg/game/limits_condition.go:133` — “Your jail sentence is served. You are free!” — C src/spec_procs2.c:1485: "The guard throws you out of the cell!\r\n" — C limits.c tick only decrements; release text is SPECIAL(jail)'s, reworded by Go
- `bug:paraphrase` — Go `pkg/game/spec_procs2.go:1546` — “says 'I don't like you, and you'd better leave before I make you!'” — C src/spec_procs2.c:1417: "$n exclaims, 'Dirty Orange Scum!'" — C ira exclaims this before hit(); Go substitutes a leave-or-I-attack line
- `bug:invented` — Go `pkg/game/spec_procs2.go:1483` — “says 'You're an evil one! That won't be allowed here!'” — C src/spec_procs2.c:1342: nothing — C evillead only handles a give of cheese 13100 and teleports the giver; no alignment attack

### `src/spell_parser.c` — `mag_objectmagic` (1)

- `bug:invented` — Go `pkg/session/use_cmds.go:53` — “Nothing magical happens.” — C src/spell_parser.c:660: nothing — C's ITEM_SCROLL recite has no such pre-check; invalid spellnum is silently ignored (spell_parser.c:405)

### `src/spell_parser.c` — `say_spell` (1)

- `bug:paraphrase` — Go `pkg/spells/call_magic.go:48` — “A magical force prevents you from casting here.” — C src/spell_parser.c:419: "Your magic fizzles out and dies.\r\n" — NOMAGIC rejection slot of call_magic; Go invents wording and drops the psionic variant at :423

### `src/spells.c` — `ASPELL` (5)

- `bug:invented` — Go `pkg/session/cmd_info.go:1048` — “Summon who?” — C src/spells.c:218: nothing — C has no summon command (summon is a spell needing a cast target); no such prompt
- `bug:paraphrase` — Go `pkg/spells/affect_spells.go:2793` — “The force of the blast knocks you off your feet!” — C src/spells.c:739: "The fires of hell bring you to your knees!\r\n" — spell_hellfire's DEX-knockdown TO_VICT line; Go rewords it as a blast
- `bug:paraphrase` — Go `pkg/session/cmd_info.go:1062` — “You are summoned by” — C src/spells.c:345: "$n has summoned you!" — C spell_summon's victim line; Go rewords it for its own teleport command
- `bug:invented` — Go `pkg/game/world.go:1292` — “You see nothing but void.” — C src/spells.c:935: nothing — spell_mindsight returns silently on a bad location; look_at_room (act.informative.c:725) has no void branch
- `bug:paraphrase` — Go `pkg/spells/affect_spells.go:2964, pkg/spells/affect_spells.go:3044` — “just tried to summon you but failed.” — C src/spells.c:246: "%s just tried to summon you to: %s.\r\n%s failed.\r\n" — spell_summon's victim line (adjacent literals); Go drops room name and failed line

### `src/tattoo.c` — `use_tattoo` (2)

- `bug:invented` — Go `pkg/game/other_economy.go:140` — “Your tattoo fizzles...” — C src/tattoo.c:50: nothing — use_tattoo's TATTOO_SKULL case has no failure branch; read_mobile always succeeds
- `bug:invented` — Go `pkg/session/tattoo.go:79` — “Your tattoo flickers but nothing happens.” — C src/tattoo.c:50: nothing — C's TATTOO_SKULL case always emits the glow act; only Go has this failure branch

## census:composed (2)

- Go `pkg/session/wiz_info.go:386` — “Usage: checkload { obj | mob } <number>” — src/act.wizard.c:3825: "Usage: checkload { obj | mob } <number>\r\n" — C holds it in a local const char* usage, invisible to the census sink lexer
- Go `pkg/game/graph.go:267` — “auctions, '” — src/act.comm.c:1273: sprintf(buf, "$n %ss, '%s'", com_msgs[subcmd][1], argument); — C composes these bytes via this sprintf plus com_msgs[SCMD_AUCTION][1] = auction (line 1189)

## surface:no-c (2)

- Go `pkg/game/graph.go:316` — “You open the door.” — src/config.c:92: "Okay.\r\n" — C do_doorcmd (reached from graph.c) sends OK to the actor; Go prints a descriptive line; no player can read it: MobInstance.SendMessage discards the text (pkg/game/mob.go:832), so this is a sink-table artefact, not a C gap; the room line C also prints is do_doorcmd's "Okay."
- Go `pkg/session/cmd_group.go:79` — “You start following” — n/a: - — Sent only to agent sessions (cmd_group.go:75); is_agent comes from the WebSocket agent login (session_login.go:64)
