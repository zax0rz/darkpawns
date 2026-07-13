# Session command layer vs game core vs C oracle — Tiers B and C

**Date:** 2026-07-13

**Scope:** the 242 primary command registrations not included in the 27-command Tier-A audit, plus the dynamically dispatched socials pathway

**Method:** static three-sided audit of the reachable session registry/handler, the closest game-core implementation, and the C command table/handler. C is treated as the behavioral oracle. Combat, skill, and spell RNG-result parity is excluded; existence, grammar, position/level gates, target selection, and deterministic messages are included.

## Executive summary

`pkg/session/commands.go` contains 264 primary registrations. Removing Tier A's exact 27-command set leaves 237 registrations. Five more reachable registrations are installed by other session-package `init` functions (`autoloot`, `cast`, `map`, `recite`, and `zap`), bringing this continuation's explicit surface to **242 commands** (`pkg/session/autoloot.go:32-34`, `pkg/session/cast_cmds.go:273-276`, `pkg/session/map_cmds.go:248-251`, `pkg/session/use_cmds.go:199-202`). Unknown commands also take a separate dynamic social route (`pkg/session/commands.go:598-603`), so socials are audited as a pathway rather than pretending they are ordinary registry rows.

The result supports Tier A's “consolidate, then delegate” conclusion. The live movement path calls the thinner of two game movement implementations; communications are split among session-only handlers, session duplicates, and delegated game handlers; and the current game copies are often incomplete too. A blind reroute would preserve or exchange bugs rather than restore C behavior.

The highest-impact findings are:

- mortal `summon` directly teleports another player, mortal `hcontrol` can create/destroy/pay/rekey houses, and mortal `page` reaches remote players (O42-O44);
- movement bypasses the closer game movement implementation and omits charm, mount, follower-position, hide, and room-trigger behavior (O25);
- the B/C registry collapses many C position/level gates, while several fighting commands use `POS_STANDING` where C uses `POS_FIGHTING` (O24/O40);
- communication delivery ignores recipient writing/channel/soundproof/position gates, while tells and same-room directed speech omit C eligibility/visibility rules (O29-O31);
- `reallyquit` is only an alias for ordinary `quit`, losing the C safe-room/equipment-loss contract (O33);
- `practice` changes skills anywhere although C only lists skills and directs named practice to a guild, `skills` is overwritten by a later alias, and `spells` is a literal stub (O36-O38);
- the extra `zap` command decrements `item.Prototype.Values[2]`, corrupting charges for every instance of the vnum (O41).

Twenty-two new candidate groups are recorded as O24-O45. Every group includes a bucket and oracle-provability tag. None is a re-filing of DP-1083..DP-1107/O1-O23.

## Already filed — do not re-file

DP-1083..DP-1093 cover O1-O9 from the pre-audit set, and DP-1094..DP-1107 cover O10-O23 from Tier A. This document starts at O24 and does not restate those findings. Linear is the source of truth for issue status; Claude should deduplicate these candidates there before filing.

## Classification key

- **Bucket 1 — reimplemented-and-drifted:** session owns a live command despite overlapping `pkg/game` command logic.
- **Bucket 2 — delegates-but-game-buggy:** session delegates to `pkg/game` or the shared `pkg/command`/game operation, but the effective behavior still diverges.
- **Bucket 3 — session-only/missing:** no game command equivalent exists, the C command is absent, or the behavior/gate is owned solely by the session/registry layer.

“No game command equivalent” does not mean there are no useful game primitives. It means there is no game-owned operation preserving the C command's grammar, validation, mutation, and message contract.

## Tier B — everyday play classification

This table exhaustively classifies the Tier-B families named by the brief. Related commands share a row where they share one implementation boundary.

| Command(s) | Bucket | C oracle | Player session | Game equivalent | Audit result |
| --- | ---: | --- | --- | --- | --- |
| `north`, `east`, `south`, `west`, `up`, `down` | 1 | Registry/handler at `src/interpreter.c:314-319`, `src/act.movement.c:95-365` | Live wrapper/handler at `pkg/session/commands.go:52-58`, `pkg/session/cmd_movement.go:9-125` | Thin `MovePlayer` at `pkg/game/world.go:884-977`; closer unused `performMove` at `pkg/game/act_movement.go:210-404` | Drifted duplicate; O25 |
| `enter` | 3 | Registered/implemented at `src/interpreter.c:432`, `src/act.movement.c:642-686` | No registration among movement rows (`pkg/session/commands.go:52-83`) | No game command equivalent | Missing; O28 |
| `stand`, `sit`, `rest`, `sleep`, `wake` | 3 | `src/act.movement.c:696-880` | Session-only state changes at `pkg/session/movement_cmds.go:17-174` | Position primitives only (`pkg/game/player_affects.go:57-68`) | Mount and magical-sleep drift; O26 |
| `follow`, `group`, `ungroup`, `gtell` | 3 | `src/act.movement.c:883-950`, `src/act.other.c:624-794`, `src/act.comm.c:824-870` | Session-only at `pkg/session/cmd_group.go:10-273` | Follower queries/primitives at `pkg/game/party.go:18-45`, `pkg/game/world.go:1306-1351` | `follow` omits charm/cycle rules; O27. No additional group-local mutation finding beyond registry gates |
| `say`, `tell`, `reply`, `emote` | 3 | `src/act.comm.c:759-973`; emote registry at `src/interpreter.c:429-430` | Session-only handlers at `pkg/session/comm_cmds.go:42-155`, `pkg/session/comm_cmds.go:285-375` | No game command equivalents | Eligibility, visibility, recipient, intoxication, and norepeat drift; O29 |
| `shout`, `gossip`, `think`, `whisper`, `ask`, `qcomm` | 1 | `src/act.comm.c:976-1018`, `src/act.comm.c:1204-1320`, `src/act.comm.c:1356-1386` | Duplicate handlers at `pkg/session/comm_cmds.go:162-278`, `pkg/session/comm_cmds.go:381-405`, `pkg/session/act_comm.go:30-175` | Second copies at `pkg/game/comm_channel.go:18-233`; bridges at `pkg/game/act_comm_bridge.go:20-44` | Both copies have gaps; O30/O31 |
| `auction`, `gratz`, `newbie`, `ctell`, `race_say` | 2 | Generic/racial/clan communication in `src/act.comm.c:635-755`, `src/act.comm.c:1204-1298`, `src/act.comm.c:1328-1354` | Thin preprocessing/delegation at `pkg/session/comm_cmds.go:526-597`, `pkg/session/act_comm.go:14-27` | `pkg/game/comm_channel.go:125-202`, `pkg/game/comm_channel.go:235-261`, `pkg/game/comm_say.go:13-170` | Keep delegated; generic channel recipient rules remain incomplete as part of O30 |
| `insult`, `dream` | 2 | `src/act.social.c:153-216` | Delegates at `pkg/session/act_social.go:14-31` | Game implementations at `pkg/game/act_social.go:115-205` | Structurally delegated; no additional deterministic divergence found |
| dynamic socials | 2 | Each social is a command-table entry with actor position, e.g. `src/interpreter.c:390-405`; `do_action` at `src/act.social.c:102-149` | Unknown-command fallback bypasses registry at `pkg/session/commands.go:598-603` | `DoAction` at `pkg/game/act_social.go:42-113`; metadata at `pkg/game/socials.go:3-14` | Actor position/level metadata is not enforced; O32 |
| `quit`/`reallyquit` | 3 | Distinct subcommands at `src/interpreter.c:630`, `src/interpreter.c:657`; contract at `src/act.other.c:72-181` | One handler/alias at `pkg/session/commands.go:245-250`, `pkg/session/cmd_inventory.go:12-44` | No game command equivalent; manager cleanup saves at `pkg/session/manager.go:821-852` | Safe-room/equipment-loss distinction absent; O33 |
| `save` | 2 | `src/act.other.c:186-203` | Delegates at `pkg/session/cmd_misc.go:7-10` | `pkg/game/other_session.go:7-18` | Delegated and persists; registry position is wrong under O24 |
| `title`, `describe`, `description` | 3 | C `title` validates/sanitizes the entire argument at `src/act.other.c:595-619`; no player `describe`/`description` registry row | Two session implementations at `pkg/session/informative_cmds.go:26-43`, `pkg/session/act_informative.go:83-95` | No game command equivalent | First-word truncation, missing validation, and inconsistent duplicate commands; O34 |
| `diagnose` (`glance`) | 3 | Qualitative target/fighting-target result at `src/act.informative.c:353-383`, `src/act.informative.c:2433-2455` | Session-only at `pkg/session/act_informative.go:97-170` | No game command equivalent | Defaults to self and exposes exact player HP; O35 |
| `color`, `toggle` | 3 | `src/act.informative.c:2462-2548` | Session formatters at `pkg/session/act_informative.go:11-41`, `pkg/session/act_informative.go:174-237` | No game command equivalent | Deterministic structure is substantially present; only registry/format drift, no standalone finding |
| `brief`, `compact`, `notell`, `noauction`, `noshout`, `nogossip`, `nograts`, `quest`, `norepeat`, `nonewbie`, `noctell`, `nobroadcast`, `nosummon`; immortal toggles `nohassle`, `nowiz`, `roomflags`, `holylight` | 2 | `do_gen_tog` registrations across `src/interpreter.c:366-834` | Shared wrapper at `pkg/session/commands.go:328-344`, `pkg/session/commands.go:409-417` | `ExecGenTog`/game toggle implementation in `pkg/game/act_other_bridge.go:70-72`, `pkg/game/other_settings.go:187-343` | Delegated; retain oracle scenarios per toggle |
| `autoloot`, `autoexit`, `lines`, `infobar`, `password`, `prompt`, `ignore`, `alias`, `afk` | 3 | Mixed C/Go extension surface; closest C rows are in `src/interpreter.c:320-850` | Session-owned registrations/handlers at `pkg/session/autoloot.go:17-34`, `pkg/session/commands.go:238-242`, `pkg/session/commands.go:295-303`, `pkg/session/commands.go:369-382` | No complete game command equivalents (AFK has a game bridge) | Classified as extensions/session UI; no new high-impact group beyond O24 |

## Tier C — structural sweep

### Combat and skill/spell invocation

RNG results and damage formulas were not compared. The sweep covers reachability, registry gates, argument/target structure, and deterministic precondition messages.

| Command family | Bucket | Structural disposition |
| --- | ---: | --- |
| `hit`, `kill`, `flee`, `assist` | 3 | Session command implementations feed the combat engine (`pkg/session/combat_cmds.go:16-270`, `pkg/session/cmd_combat_basic.go:10-88`); C entry points are at `src/interpreter.c:337`, `src/interpreter.c:445`, `src/interpreter.c:495`, `src/interpreter.c:525`. Position metadata is covered by O40; no RNG conclusions |
| `backstab`, `spike`, `stake`, `bash`, `circle`, `charge`, `kick`, `trip`, `headbutt`, `rescue`, `sneak`, `hide`, `steal`, `berserk`, `rin`, `kyo`, `toh`, `kai`, `jin`, `retsu`, `zai`, `zhen`, `sha` | 2 | Registry delegates through `wrapSkill` (`pkg/session/commands.go:144-168`, `pkg/session/commands.go:420-424`) into shared command/game operations; C positions/levels are in `src/interpreter.c:320-850`. Structural gate drift is O40 |
| `disembowel`, `dragonkick`/`dragon`, `tigerpunch`/`tiger`, `shoot`, `subdue`, `sleeper`, `neckbreak`, `ambush`, `bearhug`, `behead`, `bite`, `carve`, `compare`, `cutthroat`, `search`, `detect`, `disarm`, `groinrip`, `mindlink`, `mold`, `palm`, `point`, `scrounge`, `sharpen`, `slug`, `smackheads`, `strike`, `tag`, `turn`, `aid`, `alter`, `serpent`, `scan` | 2 | Same shared command/game boundary (`pkg/session/commands.go:252-290`, `pkg/command/skill_commands.go:520-1040` and sibling skill files). Go-only aliases/extensions are not assumed C-fidelity; deterministic positions are part of O40 |
| `practice` | 2 | Delegates to `pkg/command`, but its grammar/mutation contract differs from C (`pkg/session/commands.go:130-133`, `pkg/command/skill_commands.go:79-150`; C `src/act.other.c:543-553`); O36 |
| `skills`, `learn`, `listskills`, `forget`, `confirm`, `skillinfo` | 2 | Shared Go skill-system commands at `pkg/session/commands.go:129-142`, `pkg/command/skill_commands.go:18-76`, `pkg/command/skill_commands.go:153-499`; `skills` routing collision is O37. Go-only learn/forget/catalog semantics are classified but not assigned C parity |
| `cast` | 3 | Registered outside `commands.go` and implemented in session around `spells.Cast` (`pkg/session/cast_cmds.go:153-275`); C command/target grammar at `src/interpreter.c:373`, `src/spell_parser.c:916-1113`; O39 |
| `spells` | 3 | Session-only literal stub (`pkg/session/commands.go:242`, `pkg/session/informative_cmds.go:46-48`); no C command-table equivalent; O38 |
| `use` | 2 | Routes magic items to safe instance-value game code and otherwise to shared skills (`pkg/session/commands.go:715-750`, `pkg/game/other_economy.go:163-302`); C at `src/act.other.c:895-970` |
| `pick` | 3 | Session door/lock handler registered at `pkg/session/commands.go:168`; no complete game command equivalent. C structural entry is `src/interpreter.c:604` and `src/act.movement.c:598-638` |
| `recite` | 1 | Separate session copy (`pkg/session/use_cmds.go:12-105`, `pkg/session/use_cmds.go:199-202`) overlaps game `DoUse` scroll handling (`pkg/game/other_economy.go:267-302`); C at `src/interpreter.c:639`, `src/act.other.c:895-970`. No prototype write found |
| `zap` | 1 | Separate session copy mutates shared prototype state (`pkg/session/use_cmds.go:107-194`) while game `DoUse` uses `GetValue`/`SetValue` (`pkg/game/other_economy.go:177-240`); O41 |
| `order` | 3 | Session-only mob command dispatch (`pkg/session/cmd_combat_special.go:8-40`); C structural handler/registry at `src/act.other.c:271-335`, `src/interpreter.c:589` |

### Remaining non-Tier-A player/utility surface

These commands are included so the “everything beyond Tier A” enumeration is closed even where the brief calls for only a structural sweep.

| Bucket | Commands | Boundary / evidence |
| ---: | --- | --- |
| 1 | `junk`, `donate`, `taste`, `sip`, `quaff`, `knock`, `bashdoor`, `write`, `gossip` | Session-owned/reimplemented operations in the item/door/communication families (`pkg/session/commands.go:96-103`, `pkg/session/commands.go:178-179`, `pkg/session/commands.go:364-372`). Tier-A O21/O22 already own eat/drink prototype corruption; it is not re-filed here |
| 2 | `report`, `split`, `wimpy`, `display`, `transform`, `ride`, `dismount`, `yank`, `peek`, `recall`, `stealth`, `appraise`, `scout`, `roll`, `visible`, `inactive`, `auto`, `bug`, `typo`, `idea`, `todo`, `clan`, `house` | Thin delegates in `pkg/session/cmd_misc.go:12-147`, `pkg/session/cmd_misc.go:150-165`; game bridges at `pkg/game/act_other_bridge.go:12-110` and implementations in `pkg/game/other_*.go` |
| 3 | `coins`, `abilities`, `levels`, `review`, `whois`, `help`, `credits`, `news`, `policy`, `handbook`, `future`, `whoami`, `version`, `affects`, `autoexit`, `commands`, `lines`, `infobar`, `list`, `buy`, `sell`, `password`, `prompt`, `ignore`, `alias`, `autoloot` | Session view/UI/economy implementations registered at `pkg/session/commands.go:109-122`, `pkg/session/commands.go:135-142`, `pkg/session/commands.go:238-242`, `pkg/session/commands.go:295-324`, `pkg/session/commands.go:369-382`; no full game command equivalents |

### Immortal/wizard commands — lower-priority structural subsection

This surface is not ordinary player-facing fidelity. It is listed separately because under-gating can change server or persistent world state.

| Effective result | Commands | C vs Go evidence |
| --- | --- | --- |
| **Mortal-accessible administrative operation** | `summon`, `hcontrol`, `page` | Go level zero at `pkg/session/commands.go:171`, `pkg/session/commands.go:363`, `pkg/session/commands.go:373`; C has no `summon` command and gates `hcontrol`/`page` at `src/interpreter.c:491`, `src/interpreter.c:598`. O42-O44 |
| **Go less restrictive than C** | `at` 31 vs 38; `restore` 31 vs 33; `last` 31 vs 33; `reload` 34 vs 39; `shutdown` 38 vs 39; `sysfile` 31 vs 34; `sethunt` 31 vs 38; `tick` 31 vs 39 | Registry at `pkg/session/commands.go:182-230`; matching internal checks at `pkg/session/wiz_movement.go:38-42`, `pkg/session/wiz_player.go:40-45`, `pkg/session/wiz_system.go:14-18`, `pkg/session/wiz_system.go:62-66`, `pkg/session/wiz_system.go:180-185`, `pkg/session/wiz_system.go:338-347`, `pkg/session/wiz_zone.go:231-237`, `pkg/session/wiz_zone.go:279-285`; C rows at `src/interpreter.c:323`, `src/interpreter.c:535`, `src/interpreter.c:638`, `src/interpreter.c:651`, `src/interpreter.c:683`, `src/interpreter.c:699`, `src/interpreter.c:752`, `src/interpreter.c:774`. O45 |
| **Delegated Go less restrictive than C** | `ban`, `unban` 34 vs 38 | Go registry/delegation at `pkg/session/commands.go:352-353`, `pkg/session/cmd_misc.go:168-179`, `pkg/game/act_other_bridge.go:95-110`; C at `src/interpreter.c:345`, `src/interpreter.c:793`. O45 |
| **Go more restrictive than C** | `force` 38 vs 34; `purge` 34 vs 32; `switch` effective 38 vs 32; `return` 31 vs registry level 0; `map` 31 vs mortal; `idlist` 40 vs 38; `wizlock` 40 vs 39; `zreset` 34 vs 31 | Go registrations at `pkg/session/commands.go:182-230`, `pkg/session/map_cmds.go:248-251`; C rows at `src/interpreter.c:438`, `src/interpreter.c:512`, `src/interpreter.c:548`, `src/interpreter.c:624`, `src/interpreter.c:655`, `src/interpreter.c:751`, `src/interpreter.c:834`, `src/interpreter.c:848` |
| **Equivalent or locally stricter after internal gate** | `goto`, `load`, `teleport`, `invis`, `vis`, `gecho`, `echo`, `send`, `snoop`, `advance`, `stat`, `vnum`, `vstat`, `dc`, `home`, `date`, `reroll`, `unaffect`, `show`, `dark`, `syslog`, `checkload`, `poofset`, `wiznet`, `rlist`, `olist`, `mlist`, `zlist`, `users` | Go registry family at `pkg/session/commands.go:182-231`, `pkg/session/commands.go:303`; C command table across `src/interpreter.c:323-848`. No new security under-gate found in this static pass |
| **Go extensions/no exact C registry row** | `heal`, `wizutil`, `newbiegive`, `whod` | Registered at `pkg/session/commands.go:187-231`, `pkg/session/commands.go:356`; classified as Bucket 3 except delegated `whod` (Bucket 2) |

All wizard implementations are Bucket 3 unless they explicitly delegate (`ban`, `unban`, `hcontrol`, `whod` are Bucket 2). The table records effective gates, not just registry literals: for example, `set` is registered at 31 but its handler requires 38 (`pkg/session/commands.go:189`, `pkg/session/wiz_player.go:51-55`), so it is not reported as a privilege leak.

## New candidate findings (O-finding format)

### O24 — Tier-B registry collapses C position and level gates

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** Everyday commands carry distinct minimum positions/levels: `say`/`shout`/`emote` require resting (emote level 1), `tell` permits mortally wounded, `follow` requires resting, `group`/`gtell` allow sleeping, and `save` allows sleeping (`src/interpreter.c:429-484`, `src/interpreter.c:670-690`, `src/interpreter.c:672`, `src/interpreter.c:755`).
- **Go-session:** The corresponding registrations mostly pass level/position zero (`pkg/session/commands.go:64-71`, `pkg/session/commands.go:79-83`, `pkg/session/commands.go:125-133`, `pkg/session/commands.go:306-310`, `pkg/session/commands.go:364-378`). Zero bypasses the dispatcher position and level checks (`pkg/session/commands.go:609-623`).
- **Go-game:** Local preconditions are inconsistent and cannot restore an omitted entry gate; for example game channel functions check selected flags but not the registry position (`pkg/game/comm_channel.go:30-69`, `pkg/game/comm_channel.go:125-202`).
- **Divergence:** Dead/incapacitated/sleeping characters reach commands C rejects, and level-1 entries are exposed at level zero.
- **Impact:** High structural drift; invalid-position state mutation and communication are possible, and command failure text differs globally.
- **Fix sketch:** Port the C metadata command by command and retain game-side domain checks for non-registry callers.

### O25 — Live movement uses the thinner duplicate and skips C movement invariants

**Bucket:** 1

**Oracle-provability:** `tier1`

- **C:** Movement checks charm/master, mount state/exhaustion/indoors, follower standing position, hide clearing, sneak-gated greetings, and room enter scripts (`src/act.movement.c:95-190`, `src/act.movement.c:192-305`, `src/act.movement.c:311-350`).
- **Go-session:** `cmdMove` calls `World.MovePlayer`, then independently broadcasts, greets, and moves every follower collected in the old room without a position check or hide clear (`pkg/session/cmd_movement.go:9-29`, `pkg/session/cmd_movement.go:61-125`). It replaces any detailed game failure with a second generic failure (`pkg/session/cmd_movement.go:25-28`).
- **Go-game:** The called `MovePlayer` handles doors/boat/tunnel/cost/death but not charm, mounts, follower position/hide, or room-enter scripts (`pkg/game/world.go:884-977`). A second, unused `performMove` already has charm and follower-position/hide behavior (`pkg/game/act_movement.go:210-218`, `pkg/game/act_movement.go:367-404`) but still lacks full mount/room-trigger parity.
- **Divergence:** The live route is not the closer route, and three movement implementations now own overlapping rules/messages.
- **Impact:** High; charmed followers can leave masters, sleeping/incapacitated followers can be dragged, hidden followers stay hidden, mounts desynchronize, room scripts do not fire, and failures may double-message.
- **Fix sketch:** Create one game-owned movement transaction covering C invariants and semantic events; adapt WS/telnet renderers rather than layering session movement over it.

### O26 — Position commands ignore mounts and magical sleep

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** Standing while mounted dismounts; sitting/resting/sleeping while mounted is refused; magical `AFF_SLEEP` prevents waking and emits toss/turn messages (`src/act.movement.c:696-705`, `src/act.movement.c:734-745`, `src/act.movement.c:770-782`, `src/act.movement.c:808-822`, `src/act.movement.c:841-878`).
- **Go-session:** The five session-only handlers change position without mount checks, and `wake` unconditionally changes a sleeping target/self to sitting (`pkg/session/movement_cmds.go:17-120`, `pkg/session/movement_cmds.go:123-173`).
- **Go-game:** Only generic position/mount state primitives exist (`pkg/game/player_affects.go:57-68`, `pkg/game/deferred_fight_fns.go:139-145`); there is no canonical position command.
- **Divergence:** Mount and spell-affect invariants are absent.
- **Impact:** Medium-high; players can remain mounted while sitting/sleeping and bypass magical sleep.
- **Fix sketch:** Move position transitions into a game operation that validates mount/affect state and returns actor/room messages.

### O27 — `follow` permits charm overrides and follower cycles

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** A charmed character cannot change masters, and `circle_follow` rejects loops before changing the follower link (`src/act.movement.c:904-926`).
- **Go-session:** `cmdFollow` resolves only players and directly replaces `Following`/group state without charm or cycle checks (`pkg/session/cmd_group.go:10-66`).
- **Go-game:** Follower setters/queries do not supply the missing player-command validation (`pkg/game/world.go:1306-1351`, `pkg/game/party.go:18-45`).
- **Divergence:** Master authority and acyclic follower topology are not enforced.
- **Impact:** Medium-high; charm can be escaped and A↔B follower loops can be created, complicating group/movement recursion.
- **Fix sketch:** Add a canonical follower transition with charm, visibility, target-kind, and cycle validation; have group/movement consume it.

### O28 — `enter` is absent

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** `enter` is standing/mortal and resolves a named door or the sole indoor destination (`src/interpreter.c:432`, `src/act.movement.c:642-686`).
- **Go-session:** Movement registration jumps from directions to look/communication/positions; no `enter` row exists (`pkg/session/commands.go:52-83`).
- **Go-game:** No player-command equivalent exists; only exits/movement primitives are present (`pkg/game/act_movement.go:210-404`).
- **Divergence:** A C movement grammar entry point is unreachable.
- **Impact:** Medium; named-entry workflows and the one-indoor-exit convenience fail with “Unknown command.”
- **Fix sketch:** Implement `enter` as argument resolution feeding the canonical movement operation.

### O29 — `say`, `tell`, and `reply` omit C communication eligibility/recipient rules

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** `say` checks zero INT/WIS, mute, drunk speech, and norepeat (`src/act.comm.c:759-820`). `tell`/`reply` enforce sender notell, visibility, linkless/writing targets, recipient notell/soundproof, and stable player identity (`src/act.comm.c:873-973`).
- **Go-session:** `say` handles punctuation/moderation only (`pkg/session/comm_cmds.go:323-375`). `tell`/`reply` use online session names and ignore several recipient gates; reply also skips tell's sanitization/filter path (`pkg/session/comm_cmds.go:42-155`).
- **Go-game:** There are no game-owned say/tell/reply command equivalents; only adjacent channel/racial-say code exists (`pkg/game/comm_channel.go:18-261`, `pkg/game/comm_say.go:13-170`).
- **Divergence:** Muted/stupid/drunk/norepeat behavior and private-message availability/privacy differ.
- **Impact:** High; players can reach recipients C considers unavailable and communications disclose presence across visibility/soundproof/writing states.
- **Fix sketch:** Build a shared communication eligibility/delivery service, persist last-tell by player ID, and render channel-specific messages from its result.

### O30 — Channel duplicates ignore recipient gates; both copies remain incomplete

**Bucket:** 1

**Oracle-provability:** `tier1`

- **C:** Generic channels reject muted/soundproof/under-level/off-channel senders and skip recipients who disabled the channel, are writing, or are soundproof; shout also limits zone and recipient position (`src/act.comm.c:1204-1298`).
- **Go-session:** `shout` says recipient checks are “simplified” and delivers to every session in-zone; `gossip` delivers to every online session (`pkg/session/comm_cmds.go:162-229`, `pkg/session/comm_cmds.go:232-278`).
- **Go-game:** The second shout copy respects some preferences/soundproof state but is not zone/position scoped; generic channels omit writing/soundproof gates (`pkg/game/comm_channel.go:30-69`, `pkg/game/comm_channel.go:125-202`).
- **Divergence:** Routing to either current copy is insufficient; live shout/gossip and delegated auction/gratz/newbie share incomplete recipient policy.
- **Impact:** High privacy/consent drift; opted-out, writing, soundproof, or invalid-position players receive channel traffic.
- **Fix sketch:** Consolidate sender/recipient selection in game, fix it to C once, and return recipients plus semantic channel events for transport rendering.

### O31 — `whisper`/`ask` cannot target mobs and allow self-directed speech

**Bucket:** 1

**Oracle-provability:** `tier1`

- **C:** Resolves any visible room character, rejects self, honors norepeat, and emits distinct actor/victim/observer messages (`src/act.comm.c:976-1018`).
- **Go-session:** Searches only player sessions in the room, does not reject self, and hard-codes ANSI victim text (`pkg/session/act_comm.go:69-175`).
- **Go-game:** The duplicate uses a broader room resolver but likewise omits self/norepeat and hard-codes ANSI (`pkg/game/comm_channel.go:72-123`).
- **Divergence:** Valid NPC targets fail in the live route and invalid self targets succeed; color/norepeat contracts differ in both copies.
- **Impact:** Medium; common directed-roleplay behavior and spec-proc interaction differ visibly.
- **Fix sketch:** One canonical directed-speech operation with viewer-aware resolution and semantic actor/victim/observer outputs.

### O32 — Dynamic socials bypass actor position and social minimum-level metadata

**Bucket:** 2

**Oracle-provability:** `tier1`

- **C:** Socials are ordinary command-table entries with actor positions (for example resting `comfort` and standing `dance`) before `do_action` checks victim position (`src/interpreter.c:390-405`, `src/act.social.c:102-149`).
- **Go-session:** Socials are attempted only after registry lookup fails and call `game.DoAction` directly, bypassing registry gates (`pkg/session/commands.go:598-603`).
- **Go-game:** Social data carries `MinLevel`/`MinVictimPosition`, but `DoAction` enforces only mute and victim position, not actor level/position (`pkg/game/socials.go:3-14`, `pkg/game/act_social.go:42-113`).
- **Divergence:** Actor eligibility metadata is never applied.
- **Impact:** Medium; sleeping/dead/under-level characters can execute socials C rejects.
- **Fix sketch:** Register generated social entries with C actor metadata or make one dispatcher explicitly enforce both actor and victim gates.

### O33 — `quit` and `reallyquit` are collapsed into one unrestricted logout

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** Ordinary quit is safe only in specific temple/home rooms (or for immortals); elsewhere it directs the player to `reallyquit`, which loses equipment, then saves/extracts and dismounts (`src/act.other.c:72-181`; registrations `src/interpreter.c:630`, `src/interpreter.c:657`).
- **Go-session:** `reallyquit` is an alias for one handler; the source comment explicitly notes the missing split. The handler blocks fighting/death rooms only, then unregisters/closes (`pkg/session/commands.go:245-250`, `pkg/session/cmd_inventory.go:12-44`). Cleanup does save the character (`pkg/session/manager.go:821-852`).
- **Go-game:** No canonical quit operation exists.
- **Divergence:** Safe-room gating, explicit destructive confirmation, equipment loss, duplicate-socket handling, and dismount behavior are absent.
- **Impact:** High gameplay/economy drift; players retain equipment while quitting anywhere, eliminating the original rent/safe-logout constraint.
- **Fix sketch:** Restore separate subcommands around one game-owned logout transaction; make the destructive path explicit and test saved inventory/equipment.

### O34 — `title` truncates input and skips C validation; description commands conflict

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** `title` consumes/sanitizes the full argument and enforces notitle, parentheses, and maximum length (`src/act.other.c:595-619`). There is no player `describe`/`description` command-table row.
- **Go-session:** `title` and `describe` store only `args[0]`; `description` stores the joined argument, producing two inconsistent description commands. None applies the C title rules (`pkg/session/informative_cmds.go:26-43`, `pkg/session/act_informative.go:83-95`, registrations `pkg/session/commands.go:240-242`, `pkg/session/commands.go:295-299`).
- **Go-game:** No command equivalents exist.
- **Divergence:** Multiword titles/descriptions are truncated on two paths, title abuse controls are bypassed, and a second description path behaves differently.
- **Impact:** Medium; player-visible identity text is malformed and moderation flags/length limits do not work.
- **Fix sketch:** Port full title parsing/validation; choose one intentional description editing workflow and remove alias ambiguity.

### O35 — `diagnose` exposes exact player HP and changes no-argument targeting

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** With an argument, diagnose emits qualitative bands; without one it diagnoses the current opponent or asks “Diagnose who?” (`src/act.informative.c:353-383`, `src/act.informative.c:2433-2455`).
- **Go-session:** No argument diagnoses self, and player targets include exact current/max HP; its qualitative labels/text also differ (`pkg/session/act_informative.go:97-170`).
- **Go-game:** No command equivalent exists.
- **Divergence:** Default target and information disclosure differ.
- **Impact:** Medium; exact player HP becomes visible where C exposes only a condition band, and combat glance targets the wrong character.
- **Fix sketch:** Restore opponent/default grammar and a shared qualitative health-view formatter with viewer-aware naming.

### O36 — `practice` mutates skills anywhere instead of preserving the C guild contract

**Bucket:** 2

**Oracle-provability:** `tier1`

- **C:** No-argument `practice` lists learned skills; any named argument says skills can only be practiced in the guild. Guild specials own actual learning (`src/act.other.c:543-553`, `src/spec_procs.c:157-218`).
- **Go-session:** Registry delegates `practice` with level/position zero (`pkg/session/commands.go:130-133`).
- **Go-game/shared command:** `CmdPractice` requires a named learned skill and immediately calls `PracticeSkill`, producing progress/level messages in any room (`pkg/command/skill_commands.go:79-150`, `pkg/engine/skill_manager.go:74-89`).
- **Divergence:** Grammar, location authority, and mutation ownership are inverted.
- **Impact:** High progression drift; players can train anywhere while the C listing form is unavailable.
- **Fix sketch:** Restore C `practice` grammar and guild-owned mutation, or explicitly document a redesigned system rather than claiming a faithful port.

### O37 — `listskills` overwrites the primary `skills` route

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** Learned-skill listing is reached through `practice`/guild logic (`src/act.other.c:543-553`, `src/spec_procs.c:157-218`); there is no matching pair of `skills` and `listskills` commands.
- **Go-session:** `skills` is registered to `CmdSkills`, then `listskills` is registered with alias `skills` three lines later (`pkg/session/commands.go:130-133`). Registry aliases overwrite existing keys without collision detection (`pkg/command/registry.go:51-78`).
- **Go-game/shared command:** The two handlers are materially different: learned skills at `pkg/command/skill_commands.go:18-76`, available catalog at `pkg/command/skill_commands.go:219-270`.
- **Divergence:** Typing `skills` reaches `CmdListSkills`; the intended primary learned-skills handler is reachable only through `sk`.
- **Impact:** Medium; the canonical command silently invokes the wrong view and help/registry introspection disagrees with dispatch.
- **Fix sketch:** Reject registry collisions and assign unique primary/alias names; add a lookup test for every advertised alias.

### O38 — `spells` always says the character knows none

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** There is no separate `spells` command; learned spells appear in the skill/practice catalog (`src/act.other.c:543-553`, `src/spec_procs.c:157-218`).
- **Go-session:** `spells` is advertised/registered but its handler is a literal `s.Send("You know no spells.")` stub (`pkg/session/commands.go:242`, `pkg/session/informative_cmds.go:46-48`).
- **Go-game:** The spell system has a populated catalog and casting implementation (`pkg/spells/spells.go:45-130`, `pkg/spells/spells.go:205-260`).
- **Divergence:** An advertised command contradicts actual learned/castable spell state.
- **Impact:** Medium; spell users receive false progression information.
- **Fix sketch:** Remove the unsupported extension or render learned spell state from the same canonical data used by cast/practice.

### O39 — `cast` bypasses the C structural target and eligibility contract

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** `cast` is level 1/sitting, requires quoted names, checks class minimum/learned percentage/peaceful violent spells, resolves character and object target flags across room/world/inventory/equipment, and defaults from target flags/current combat (`src/interpreter.c:373`, `src/spell_parser.c:916-1077`).
- **Go-session:** Registered level/position zero outside the main registry block. It accepts an unquoted first word, checks only `SpellMap`, defaults no target to self, resolves only an in-room character, deducts mana before calling `spells.Cast`, and emits unconditional confirmation (`pkg/session/cast_cmds.go:153-275`).
- **Go-game:** `spells.Cast`/`CallMagic` owns spell effects and a peaceful check but cannot repair the already-chosen wrong target/grammar or pre-deducted mana (`pkg/spells/spells.go:205-260`, `pkg/spells/call_magic.go:13-51`).
- **Divergence:** Structural targeting, position/level/class gates, failure timing, and messages differ independently of RNG.
- **Impact:** High; violent/self/object/world/default targets behave incorrectly and mana can be charged around invalid C casts.
- **Fix sketch:** Move cast resolution into a canonical spell invocation operation driven by `SpellInfo` target flags; let it return deterministic failure/success events before RNG resolution.

### O40 — Combat/skill registry positions block valid fighting-state commands and under-gate level-1 skills

**Bucket:** 2

**Oracle-provability:** `tier1`

- **C:** `hit`, `kill`, `rescue`, and `neckbreak` use `POS_FIGHTING`; `assist`, `flee`, `rescue`, `practice`, `cast`, and many learned skills are level 1 (`src/interpreter.c:337`, `src/interpreter.c:373`, `src/interpreter.c:445`, `src/interpreter.c:495`, `src/interpreter.c:525`, `src/interpreter.c:564`, `src/interpreter.c:618`, `src/interpreter.c:650`). A minimum of fighting permits both fighting and higher standing position.
- **Go-session:** `hit`, `kill`, `rescue`, and `neckbreak` are registered `POS_STANDING`; combat/skill registrations overwhelmingly use min level zero (`pkg/session/commands.go:73-76`, `pkg/session/commands.go:145-168`, `pkg/session/commands.go:252-290`). Dispatcher compares numeric position directly (`pkg/session/commands.go:609-623`).
- **Go-game/shared command:** Skill handlers perform learned-skill/domain checks but receive no call when the session rejects a fighting character first; examples are rescue and neckbreak (`pkg/command/skill_commands.go:741-775`, `pkg/command/skill_commands.go:997-1018`).
- **Divergence:** A character currently fighting is rejected from commands C allows, while level-zero metadata is looser than C.
- **Impact:** High structural combat regression; rescue/retarget/neckbreak paths can be unreachable in the state they are designed for.
- **Fix sketch:** Generate/verify registry metadata against the C command table and add deterministic position-matrix tests; keep skill-learned checks in game.

### O41 — `zap` decrements shared prototype charges

**Bucket:** 1

**Oracle-provability:** `tier1`

- **C:** Wand/staff current charges are instance object values and are decremented on the used object (`src/spell_parser.c:724-810`).
- **Go-session:** The extra `zap` handler reads and decrements `item.Prototype.Values[2]` (`pkg/session/use_cmds.go:144-163`) and is separately registered (`pkg/session/use_cmds.go:199-202`).
- **Go-game:** Canonical `DoUse` reads/writes instance values through `GetValue`/`SetValue` (`pkg/game/other_economy.go:177-240`); copy-on-write accessors are at `pkg/game/object.go:432-455`.
- **Divergence:** Using one wand/staff through `zap` changes the shared prototype and therefore every instance of that vnum.
- **Impact:** Critical data corruption; charges leak across players/objects and may affect future instances.
- **Fix sketch:** Remove the duplicate `zap` path or route it through canonical `DoUse`; never mutate `Prototype.Values` at runtime.

### O42 — `summon` gives every mortal an unsynchronized remote-player teleport

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** There is no player/admin `summon` command-table row; summon is a spell with its own spell rules (`src/spell_parser.c:916-1113`).
- **Go-session:** An “Admin / debug” command is registered at level/position zero and directly assigns the target's `RoomVNum` (`pkg/session/commands.go:170-171`, `pkg/session/cmd_info.go:508-531`).
- **Go-game:** The command bypasses movement/teleport operations and their room/light/events/scripts; even thin movement centralizes some of those effects (`pkg/game/world.go:884-977`).
- **Divergence:** Any authenticated mortal can relocate any named online player without consent, spell checks, privilege, or movement bookkeeping.
- **Impact:** Critical security/gameplay issue; harassment, escape/capture, room-state desynchronization, and trigger bypass are trivial.
- **Fix sketch:** Remove the debug command from production or gate it to the intended administrator level and route relocation through a canonical teleport transaction.

### O43 — `hcontrol` is mortal-accessible and mutates persistent house control

**Bucket:** 2

**Oracle-provability:** `tier1`

- **C:** `hcontrol` is `LVL_GRGOD` and dispatches build/destroy/pay/show/key (`src/interpreter.c:491`, `src/house.c:579-598`).
- **Go-session:** Registered level zero and delegates without an internal session gate (`pkg/session/commands.go:361-363`, `pkg/session/cmd_misc.go:156-159`).
- **Go-game:** `World.Hcontrol` has no level check and dispatches the same persistent mutations, whose helpers save house control (`pkg/game/house_control.go:331-360`; e.g. key save at `pkg/game/house_control.go:320-328`).
- **Divergence:** The sole authorization check is absent on both sides of the delegation.
- **Impact:** Critical authorization failure; mortals can create, destroy, pay, enumerate, or rekey houses.
- **Fix sketch:** Gate the registry to GRGOD and add defense-in-depth authorization in the game operation before any persistent mutation.

### O44 — `page` is mortal-accessible and its parser truncates messages

**Bucket:** 3

**Oracle-provability:** `tier1`

- **C:** `page` is immortal-only; it parses one target plus the remaining message, supports `all` only above GOD, and uses visibility/connection checks (`src/interpreter.c:598`, `src/act.comm.c:1107-1139`).
- **Go-session:** Registered level zero. It treats every argument except the last as a target and only the last word as the message, then sends to exact online session names (`pkg/session/commands.go:373`, `pkg/session/comm_cmds.go:477-515`).
- **Go-game:** No command equivalent exists.
- **Divergence:** Privilege, grammar, all-target authorization, visibility, and message text differ.
- **Impact:** High; mortals can send bell-prefixed urgent messages remotely, while ordinary multiword pages misparse.
- **Fix sketch:** Restore immortal gating and C target/message grammar; centralize remote-player visibility/connection selection.

### O45 — Remaining wizard effective-level matrix contains multiple under-gates

**Bucket:** 3 (session-owned rows); 2 for delegated `ban`/`unban`

**Oracle-provability:** `tier1`

- **C:** Effective levels are GRGOD for `at`, `ban`, `unban`, and `sethunt`; GOD for `sysfile`; GOD-1 for `restore`/`last`; IMPL-1 for `reload`/`shutdown`/`tick` (`src/interpreter.c:323`, `src/interpreter.c:345`, `src/interpreter.c:535`, `src/interpreter.c:638`, `src/interpreter.c:651`, `src/interpreter.c:683`, `src/interpreter.c:699`, `src/interpreter.c:752`, `src/interpreter.c:774`, `src/interpreter.c:793`).
- **Go-session:** Effective gates are lower for every command listed in the less-restrictive matrix after registry plus internal checks (`pkg/session/commands.go:182-230`, `pkg/session/commands.go:352-353`; handler checks cited in the immortal matrix above).
- **Go-game:** `ban`/`unban` delegate to game managers without a level check (`pkg/game/act_other_bridge.go:95-110`); the other commands are session-owned and call lower-level state operations directly.
- **Divergence:** Several lower immortal tiers receive authority reserved by C for higher tiers. Separate over-gates are also documented in the matrix but are not security leaks.
- **Impact:** High administrative least-privilege drift, though lower priority than mortal O42-O44.
- **Fix sketch:** Define one authoritative command privilege table, align effective (not merely advertised) gates to C, and assert every wizard command at boundary levels.

## Systemic threads and fix-roadmap implications

### Duplicate session/game paths

The new instances are movement (O25), generic/directed communication (O30/O31), and magic-item use (`recite`/`zap`, O41). In each case the desired endpoint is one canonical game operation plus transport adapters. Direct delegation to today's game copy is not automatically safe:

- game `performMove` is closer than `MovePlayer`, but still lacks full C mount/room-trigger behavior;
- game generic channels preserve more preferences than session shout/gossip, but still miss writing/soundproof/zone/position rules;
- game directed speech still omits self/norepeat behavior;
- game `DoUse` is instance-safe and is the clear consolidation target for `zap`, but its target/message parity still needs oracle coverage.

### Prototype mutation

O41 is the new B/C instance. Tier-A O21/O22 already own the `eat`/`drink`/`pour` prototype writes and are not repeated. The safe runtime boundary is `ObjectInstance.GetValue`/`SetValue` (`pkg/game/object.go:432-455`); `Prototype.Values` is read-only template data after instantiation.

### Recommended implementation order

1. Gate/remove mortal `summon`, `hcontrol`, and `page`; align the remaining wizard privilege matrix (O42-O45).
2. Stop `zap` prototype mutation by consolidating onto `DoUse` (O41).
3. Restore B/C registry position/level metadata, including fighting-vs-standing thresholds (O24/O40).
4. Consolidate movement and restore C invariants (O25-O28).
5. Build one communication eligibility/delivery core and route all session/game copies through it (O29-O32).
6. Restore quit/title/diagnose deterministic contracts (O33-O35).
7. Decide and document the intended skill-system model, then fix practice/skills/spells/cast structure against that decision and the oracle (O36-O39).

## Audit limitations

- Static audit only; no C oracle scenarios were executed. `tier1` tags mean the behavior is deterministic and suitable for a future red→green oracle-diff scenario, not that such a scenario already exists.
- Combat/skill/spell RNG outcomes were deliberately excluded. No finding depends on comparing hit chance, damage, saving throws, or random skill advancement results.
- Go-only commands and redesigned skill-system concepts have no direct C behavioral oracle. They are classified structurally; where a candidate is based on Go-internal reachability (such as O37/O38), the deterministic Go half is still testable even though product intent may need manual confirmation.
- Linear issue contents were not queried from this environment. The explicit DP-1083..DP-1107 dedup boundary from the brief was honored; Claude should perform final title/body dedup before filing O24-O45.
