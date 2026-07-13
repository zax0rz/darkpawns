# Session command layer vs game core vs C oracle

**Date:** 2026-07-13

**Scope:** Tier-A player commands only (27 commands)

**Method:** static three-sided audit of the player registry, the reachable session handler, the closest game-core implementation, and the C command table/handler. C is treated as the behavioral oracle. RNG-result parity is excluded.

## Executive summary

The Tier-A surface contains 27 commands. Thirteen are reimplemented in `pkg/session` despite overlapping game-core logic, four already delegate to game code, and ten are session-only or absent.

| Classification | Commands | Count |
| --- | --- | ---: |
| Bucket 1 — reimplemented/duplicated | `look`, `examine`, `time`, `weather`, `drop`, `wear`, `remove`, `wield`, `hold`, `open`, `close`, `lock`, `unlock` | 13 |
| Bucket 2 — session delegates; game owns behavior | `get`, `put`, `give`, `pour` | 4 |
| Bucket 3 — session-only or missing | `exits`, `score`, `inventory`, `equipment`, `who`, `where`, `consider`, `drink`, `eat`, `fill` | 10 |

The systemic opportunity is real, but the safe headline is **consolidate, then delegate**, not “point every session command at the current game function.” Nine filed/new finding groups sit in Bucket 1: DP-1083, DP-1084, DP-1086, DP-1089, DP-1091, O16, O18, O19, and O23 below. A shared-core refactor plus the identified core fixes can close those nine groups and prevent two implementations from drifting again. A blind reroute to today's game functions safely closes **zero** of the nine in full:

- game `look` omits room extra descriptions and C's full adjacent-room directional rendering (`pkg/game/look.go:226-311`, `pkg/game/look.go:366-398`; C: `src/act.informative.c:1032-1037`, `src/act.informative.c:899-904`);
- both Go door implementations are door-only (`pkg/session/door_cmds.go:25-79`, `pkg/game/act_movement.go:608-676`), unlike C's object-or-door path (`src/act.movement.c:611-635`);
- game `drop` is closer, but still lacks C's coin form (`pkg/game/item_transfer.go:241-300`; C: `src/act.item.c:548-600`);
- game has no complete exported equivalents for examine, time/weather display, remove, wield, or hold.

Fourteen new candidate finding groups are recorded as O10-O23. They are deliberately grouped by one likely fix/root cause rather than by every differing string. The highest-impact residuals are a mortal-accessible `where` command that exposes all player room vnums, ineffective Tier-A position gates, broken take-bit handling in delegated `get`, `give` to mobs that does not transfer the object, and eat/drink helpers that are literal no-op stubs.

### Correction to the brief's starting thesis

`pkg/game/look.go` is substantially closer to C, but is not fully C-faithful. Its `lookAtTarget` searches characters and object extra descriptions but never `room.ExtraDescs` (`pkg/game/look.go:226-311`), whereas C checks the room first (`src/act.informative.c:1032-1037`). Its directional look sends the destination name plus contents (`pkg/game/look.go:366-398`), while C announces the direction and renders the destination through full `look_at_room` (`src/act.informative.c:851-904`). DP-1086 and DP-1089 therefore require a canonical observation fix, not merely changing the called function.

## Already filed — do not re-file

These are the nine findings named in the brief/repository. Linear was not available from this execution environment, so the list uses only titles established by the brief and local research; Claude should do the final Linear-side dedup.

- **DP-1083:** telnet `look` leaks room vnum. Confirmed root cause: `RoomState.VNum` is unconditional (`pkg/session/protocol.go:104-114`), session always fills it (`pkg/session/cmd_look.go:74-100`), and telnet always renders it (`pkg/telnet/listener.go:693-696`).
- **DP-1084:** `examine` dumps an item stat block. Confirmed at `pkg/session/examine.go:102-135`; C uses ordinary look output and, for containers, contents (`src/act.informative.c:1137-1164`).
- **DP-1085:** new-character hometown/start-room mismatch. Outside this Tier-A command audit.
- **DP-1086:** room extra descriptions cannot be looked at. Confirmed in session and, contrary to the brief, also absent from game `lookAtTarget` (`pkg/session/cmd_look.go:173-236`, `pkg/game/look.go:226-311`; C: `src/act.informative.c:1032-1037`).
- **DP-1089:** directional look is one line/thin. Confirmed in session and still incomplete in game (`pkg/session/cmd_look.go:142-170`, `pkg/game/look.go:366-398`; C: `src/act.informative.c:841-912`).
- **DP-1090:** `exits` is missing. The info registry jumps from `score`/`who`/`where` onward without `exits` (`pkg/session/commands.go:105-115`); C registers and implements it (`src/interpreter.c:435`, `src/act.informative.c:683-721`).
- **DP-1091:** `open <container>` fails. Confirmed in both Go door paths (`pkg/session/door_cmds.go:46-78`, `pkg/game/act_movement.go:608-676`); C resolves inventory/room objects before exits (`src/act.movement.c:611-635`).
- **DP-1092:** `get` uses the wrong container name in a failure message. Owned by the delegated game path (`pkg/session/cmd_inventory.go:256-268`, `pkg/game/item_transfer.go:207-221`).
- **DP-1093:** `score` uses the wrong hometown table. Confirmed at `pkg/session/cmd_info.go:201-206`; C indexes `hometowns` (`src/act.informative.c:1281-1285`).

DP-1087 and DP-1088 are not among the nine IDs named by the brief and have no metadata in this checkout; no conclusion about them is inferred here.

## Cross-cutting registry divergence

Every registered Tier-A command uses minimum position `0` (`pkg/session/commands.go:61`, `pkg/session/commands.go:86-108`, `pkg/session/commands.go:174-177`, `pkg/session/commands.go:234-237`). The dispatcher enforces a position only when the registered value is greater than zero (`pkg/session/commands.go:609-616`). C permits only `inventory`, `score`, `time`, and `who` at `POS_DEAD`; the other Tier-A commands require sleeping, resting, sitting, or standing (`src/interpreter.c:382-443`, `src/interpreter.c:459-543`, `src/interpreter.c:589-674`, `src/interpreter.c:778-824`). This is O10 and affects all table rows except those four C-dead commands.

## Tier-A divergence table

“No command equivalent” means the game package has a useful primitive at most, not a player-command implementation with the C argument and message contract.

| Command | Bucket | C oracle | Player session | Game equivalent | Divergence / impact | Recommended fix |
| --- | ---: | --- | --- | --- | --- | --- |
| `where` | 3 | Immortal-only/resting in `src/interpreter.c:820`; no-arg and target searches in `src/act.informative.c:2244-2307` | Mortal/dead and no arguments at `pkg/session/commands.go:108`; prints every session's room vnum at `pkg/session/cmd_info.go:476-505` | No command equivalent; room lookup primitive at `pkg/game/world.go:794-800` | **Critical:** exposes exact player locations/vnums to mortals; target form is unreachable (O11) | Correct registry level/position; implement both C forms with visibility rules |
| Tier-A position gates | 1/2/3 | Per-command positions in `src/interpreter.c:382-443`, `src/interpreter.c:459-543`, `src/interpreter.c:589-674`, `src/interpreter.c:778-824` | All Tier-A registrations pass `0`; dispatcher skips a zero gate at `pkg/session/commands.go:609-616` | Some game functions have local checks, e.g. look at `pkg/game/look.go:9-42`, but wrappers do not consistently reach them | **High:** dead/stunned/sleeping players can execute commands C rejects (O10) | Fix registry metadata; retain domain checks for non-registry callers |
| `who` | 3 | Parses filters and applies visibility/no-who-room gates at `src/act.informative.c:1681-1774` | `wrapNoArgs` discards filters at `pkg/session/commands.go:107`; lists every live session at `pkg/session/cmd_info.go:426-463` | No player equivalent; unrelated whod export is `pkg/game/whod.go:212-279` | **High:** hidden/no-who-room players are exposed; C filters, titles, and status output are absent (O13) | Implement in session or shared query service with viewer-aware visibility |
| `get` | 2 | Take is a bit test and container forms are handled at `src/act.item.c:158-175`, `src/act.item.c:346-427` | Correctly delegates at `pkg/session/cmd_inventory.go:256-268` | `DoGet` at `pkg/game/item_transfer.go:81-223`; take check compares a whole flag word to `1` at `pkg/game/item_transfer.go:21-31` | **High:** normal take+wield objects can be rejected; `get all <container>` is consumed by the room-all branch (O17). DP-1092 is also here | Fix in game: bit-test TAKE and parse container before room-all dispatch |
| `give` | 2 | Object transfer works for characters/mobs and supports dot modes at `src/act.item.c:669-705`, `src/act.item.c:767-830` | Correctly delegates at `pkg/session/cmd_inventory.go:271-283` | Mob path invokes a script/refuses but never moves the object at `pkg/game/item_transfer.go:451-475`; player dot modes begin at `pkg/game/item_transfer.go:483-490` | **High:** scripted mob gives do not deliver the item; C's full `all`/`all.x` contract is incomplete (O20) | Fix game transfer first, then trigger scripts; restore full argument grammar |
| `eat` | 3 | Applies fullness/poison and race/type branches at `src/act.item.c:1035-1156` | Reimplements at `pkg/session/eat_cmds.go:15-113`; calls `EatFood` at lines 62-69 and mutates prototype values at lines 95-105 | `EatFood` is a literal `return 1, nil` stub at `pkg/game/act_item_stubs.go:3-9` | **High:** no fullness gain; taste mutates every instance sharing the prototype; C branches are absent (O21) | Implement canonical game consumable operation using instance values; make session render its result |
| `drink` | 3 | Computes amount, changes DRUNK/FULL/THIRST, poison, and container state at `src/act.item.c:895-1032` | Reimplements at `pkg/session/eat_cmds.go:117-259`; calls `DrinkLiquid` at lines 211-217 and mutates prototype values at lines 247-255 | `DrinkLiquid` is a literal fixed-result stub at `pkg/game/act_item_stubs.go:11-15` | **High:** conditions do not change correctly and one drink mutates all instances of a prototype (O21) | Same shared consumable operation as `eat`; use `GetValue`/`SetValue` |
| `time` | 1 | Reads the global time/weather state and prints moon only at `SUN_DARK` in `src/act.informative.c:1498-1543` | Uses process-start elapsed time and a separate clock at `pkg/session/time_weather.go:80-147` | Canonical time/weather state advances at `pkg/game/weather.go:57-180`; moon getter at `pkg/game/weather.go:422-426` | **High:** displayed time/moon can disagree with darkness and world events (O16) | Add read-only game snapshot and a transport-neutral formatter/result |
| `weather` | 1 | Outdoor gate plus sky/change text at `src/act.informative.c:1546-1563` | Uses a separate ten-minute random cache, with no indoor gate, at `pkg/session/time_weather.go:154-257` | Canonical weather at `pkg/game/weather.go:67-110`; world darkness consumes it at `pkg/game/world.go:766-791` | **High:** command can report weather unrelated to active world weather; indoor players get outdoor reports (O16) | Read canonical game weather snapshot; preserve C outdoor test |
| `look` | 1 | Position/blind gates, vnum privilege, room extras, and full directional look at `src/act.informative.c:725-912`, `src/act.informative.c:1005-1134` | Structured reimplementation at `pkg/session/cmd_look.go:13-236`; unconditional vnum at lines 74-100 | Closer port at `pkg/game/look.go:9-116`, `pkg/game/look.go:226-311`, `pkg/game/look.go:366-457`, but still misses room extras/full directional rendering | **High:** confirms DP-1083/1086/1089; game is not yet safe as a direct replacement | Build canonical observation model, fix it once, then WS/text renderers |
| `open`/`close`/`lock`/`unlock` | 1 | Generic object-or-door operations at `src/act.movement.c:577-638`; sitting gate in `src/interpreter.c:382`, `src/interpreter.c:543`, `src/interpreter.c:589`, `src/interpreter.c:791` | Direction-only duplicate at `pkg/session/door_cmds.go:25-79`, handlers at lines 342-363 | Second door-only implementation at `pkg/game/act_movement.go:608-676` | **High:** confirms DP-1091; all four also have wrong position gates | Consolidate canonical door/container mutation first; then adapt structured door events |
| `fill` | 3 | Registered standing and handled by `do_pour` at `src/interpreter.c:443`, `src/act.item.c:1186-1224` | Missing from item registry (`pkg/session/commands.go:85-103`) | `ExecPour` hard-codes `"pour"` at `pkg/game/act_other_bridge.go:83-84`; no fill branch in `pkg/game/item_consumable.go:8-110` | **High:** command does not exist (O22) | Implement canonical fill/pour operation and register `fill` |
| `pour` | 2 | Fill/pour, puddle creation, value/weight/name updates at `src/act.item.c:1159-1335` | Correctly delegates at `pkg/session/cmd_misc.go:90-94` | `doPour` at `pkg/game/item_consumable.go:8-110` clears liquid without a puddle and mutates `Prototype.Values` | **High:** shared prototype corruption; missing puddle and state updates (O22) | Fix in game using instance `GetValue`/`SetValue` (`pkg/game/object.go:432-455`) |
| `examine` | 1 | Ordinary target look, then container/drink contents at `src/act.informative.c:1137-1164` | Separate resolver and debug-like stats at `pkg/session/examine.go:24-193`; does not search equipped items or show contents | Closest shared primitives are `lookAtTarget`/object look at `pkg/game/look.go:226-311`; no full examine command | **Medium-high:** confirms DP-1084; also misses C's contents/equipped targeting and overexposes player/mob data (O23) | Shared look/examine result model; C-faithful renderer |
| `drop` | 1 | Coin, individual, `all`, `all.x`, and cursed checks at `src/act.item.c:529-666` | Single substring item, no curse/coins/dot modes at `pkg/session/cmd_inventory.go:301-342` | Closer dot-mode/curse implementation at `pkg/game/item_transfer.go:225-300`, but no coin branch | **Medium-high:** command grammar and NODROP protection differ (O18) | Export/consolidate game drop, add C coin path, preserve semantic event adapter |
| `wear` | 1 | `all`, `all.x`, explicit body location and equipment checks at `src/act.item.c:1584-1658` | Single item and generic `EquipForPlayer` at `pkg/session/cmd_inventory.go:86-122` | Closer `doWear`/`performWear` at `pkg/game/item_equipment.go:85-156`, `pkg/game/item_equipment.go:195-282` | **Medium-high:** session omits C grammar and wield/two-hand/occupied-slot behavior (O19) | Export corrected game operation; return a mutation/event result |
| `remove` | 1 | Individual/all removal with NODROP checks at `src/act.item.c:1713-1789` | One map-order substring match at `pkg/session/cmd_inventory.go:124-161` | Partial `performRemove` only at `pkg/game/item_equipment.go:284-310` | **Medium:** missing C grammar; nondeterministic ambiguous matching (O19) | Complete one game command around `performRemove`, then delegate |
| `wield` | 1 | Type, weight, occupied-slot, and two-hand rules at `src/act.item.c:1661-1681` | Pre-unequips existing weapon and calls generic equip at `pkg/session/cmd_inventory.go:163-211` | Wield constraints exist only inside game wear at `pkg/game/item_equipment.go:117-142` | **Medium-high:** session bypasses the canonical constraints and can remove the current weapon before replacement succeeds (O19) | Shared explicit-slot equipment operation; atomic result |
| `hold` (`grab`) | 1 | Hold/light behavior at `src/act.item.c:1685-1709`; resting in `src/interpreter.c:472`, `src/interpreter.c:497` | Pre-unequips hold, then generic auto-slot equip at `pkg/session/cmd_inventory.go:213-254` | Hold/two-hand constraint only in game wear at `pkg/game/item_equipment.go:133-141` | **Medium-high:** requested hold is not guaranteed to use hold slot and can destructively pre-unequip (O19) | Same explicit-slot atomic equipment operation |
| `score` | 3 | Full display contract at `src/act.informative.c:1168-1452` | Session-only at `pkg/session/cmd_info.go:142-352` | No command equivalent; closest data model is `pkg/game/player.go:55-82` | **Medium:** DP-1093 plus omitted chosen/equipment-affect lines and wrong mana/fighting/drunk/flesh-alter presentation (O14) | Fix session formatter or move to shared character-view formatter |
| `consider` | 3 | Missing target uses “Consider killing who?” and output is TO_CHAR only at `src/act.informative.c:2330-2429` | Session-only at `pkg/session/consider.go:15-219`; wrong sex mapping at lines 252-273 and room broadcast at lines 283-298 | No command equivalent; shared target resolver only | **Medium:** wrong pronouns, wrong not-found message, and broadcasts a private evaluation (O15) | Correct session messages/pronouns; remove broadcast; later share calculation |
| `inventory` | 3 | Fixed header and `list_obj_to_char` formatting at `src/act.informative.c:1460-1467` | Simple short-desc list at `pkg/session/cmd_inventory.go:47-66` | No command equivalent; room-only listing helper at `pkg/game/look.go:122-169` | **Medium:** empty/output/grouping semantics differ (O12) | Implement C-style inventory view/formatter |
| `equipment` | 3 | Fixed slot order/labels, visibility, and “Nothing” at `src/act.informative.c:1470-1495` | Ranges a map and says “wearing” at `pkg/session/cmd_inventory.go:68-84` | No command equivalent; typed equipment access at `pkg/game/equipment.go:175-217` | **Medium:** nondeterministic order and changed labels/visibility semantics (O12) | Render canonical slot order from shared view |
| `put` | 2 | Container/dot-mode grammar and capacity at `src/act.item.c:77-153` | Correctly delegates at `pkg/session/cmd_inventory.go:286-299` | `DoPut` at `pkg/game/item_container.go:9-134` | No additional command-local divergence found in this pass; container openability remains DP-1091 | Keep delegated; add oracle scenarios before edits |
| `exits` | 3 | Blind gate, visible destinations, immortal vnums at `src/act.informative.c:683-721`; registered at `src/interpreter.c:435` | Missing at `pkg/session/commands.go:105-115` | Auto-exit helper only at `pkg/game/look.go:432-457` | Confirms DP-1090 | Implement a structured exits view plus WS/telnet renderers |

## New candidate findings (O-finding format)

### O10 — Tier-A registry collapses C position gates to POS_DEAD

- **C:** Only inventory/score/time/who in this tier use `POS_DEAD`; the other commands require sleeping/resting/sitting/standing (`src/interpreter.c:382-443`, `src/interpreter.c:459-543`, `src/interpreter.c:589-674`, `src/interpreter.c:778-824`).
- **Go-session:** Every Tier-A registration passes min-position `0` (`pkg/session/commands.go:61`, `pkg/session/commands.go:86-108`, `pkg/session/commands.go:174-177`, `pkg/session/commands.go:234-237`); zero bypasses enforcement (`pkg/session/commands.go:609-616`).
- **Go-game:** Local checks are inconsistent; game look checks position (`pkg/game/look.go:9-42`), while many delegated mutations assume the caller gated them.
- **Divergence:** The live registry admits command execution below C's minimum position.
- **Player impact:** High; dead, incapacitated, stunned, or sleeping players can inspect or mutate state when C refuses.
- **Fix sketch:** Port each C registry position. Keep critical game-side preconditions because spec-procs/mobprogs bypass the player registry.

### O11 — `where` is mortal-accessible and exposes every player's exact room

- **C:** `where` is `LVL_IMMORT`, `POS_RESTING` (`src/interpreter.c:820`), with no-arg and target search forms (`src/act.informative.c:2244-2307`).
- **Go-session:** Registered level/position zero through `wrapNoArgs` (`pkg/session/commands.go:108`) and prints all names, room vnums, and room names (`pkg/session/cmd_info.go:476-505`).
- **Go-game:** No player command; only room query primitives (`pkg/game/world.go:794-800`).
- **Divergence:** Privilege and argument gates are absent.
- **Player impact:** Critical privacy/gameplay leak; any mortal can track every connected character precisely.
- **Fix sketch:** Make it immortal/resting, restore both forms, and apply visibility rules.

### O12 — Inventory/equipment views do not preserve C formatting or order

- **C:** Inventory uses `list_obj_to_char`; equipment walks fixed `where[]` order with visibility handling (`src/act.informative.c:1460-1495`).
- **Go-session:** Inventory prints one short description per slice entry; equipment ranges a Go map (`pkg/session/cmd_inventory.go:47-84`).
- **Go-game:** No player equivalent; typed equipment access exists (`pkg/game/equipment.go:175-217`).
- **Divergence:** Equipment order is nondeterministic; headers, empty states, visibility, grouping, and object formatting differ.
- **Player impact:** Medium; unstable and materially less informative routine output.
- **Fix sketch:** Produce a canonical inventory/equipment view in fixed slot order and transport-neutral lines.

### O13 — `who` bypasses C filters and visibility/privacy gates

- **C:** Parses level/name/class/quest/outlaw/short filters and excludes invisible/no-who-room players (`src/act.informative.c:1681-1774`).
- **Go-session:** `wrapNoArgs` makes filters unreachable and the handler lists every session (`pkg/session/commands.go:107`, `pkg/session/cmd_info.go:426-463`).
- **Go-game:** No in-game equivalent; whod output is a separate facility (`pkg/game/whod.go:212-279`).
- **Divergence:** Viewer-aware filtering and C's title/status contract are absent.
- **Player impact:** High privacy leak and broken command forms.
- **Fix sketch:** Accept args and query a visibility-aware roster; render C fields from a DTO.

### O14 — `score` has residual C-visible omissions beyond DP-1093

- **C:** Uses `Mind/Psi` for psionic/mystic, names the fighting target, gates drunk at `>10`, shows chosen/flesh weapon/equipment effects (`src/act.informative.c:1192-1201`, `src/act.informative.c:1319-1366`, `src/act.informative.c:1383-1452`).
- **Go-session:** Always says Mana, fighting is generic, drunk is `>0`, chosen is explicitly skipped, and flesh alter is generic (`pkg/session/cmd_info.go:115-162`, `pkg/session/cmd_info.go:258-300`).
- **Go-game:** No command equivalent.
- **Divergence:** Several player-visible score fields still differ after the hometown issue.
- **Player impact:** Medium; incorrect status/class presentation and missing effects.
- **Fix sketch:** Port the residual formatter branches and add golden output fixtures.

### O15 — `consider` uses wrong pronouns and broadcasts a private action

- **C:** Missing/unseen target gets “Consider killing who?” and the result is `TO_CHAR` only (`src/act.informative.c:2341-2349`, `src/act.informative.c:2429`).
- **Go-session:** Not-found says “They aren't here,” then broadcasts to the room (`pkg/session/consider.go:30-36`, `pkg/session/consider.go:210-216`, `pkg/session/consider.go:283-298`). Its pronoun mapping says 0 neutral/1 male/2 female (`pkg/session/consider.go:252-273`).
- **Go-game:** Player sex is actually 0 male/1 female/2 neutral (`pkg/game/player.go:68-72`); no consider command exists.
- **Divergence:** Sex constants are interpreted incorrectly and an extra social event is emitted.
- **Player impact:** Medium; visibly wrong pronouns and unintended disclosure.
- **Fix sketch:** Use shared sex constants/pronoun helper, restore exact not-found output, and remove the broadcast.

### O16 — `time` and `weather` report a simulation disconnected from the world

- **C:** Both commands read the global `time_info`/`weather_info`; weather is outdoor-gated (`src/act.informative.c:1498-1563`).
- **Go-session:** Owns a process-start clock and separate random weather cache (`pkg/session/time_weather.go:80-123`, `pkg/session/time_weather.go:154-257`).
- **Go-game:** Owns the canonical advancing time/weather (`pkg/game/weather.go:57-180`), consumed by darkness (`pkg/game/world.go:766-791`).
- **Divergence:** Display state and gameplay state have independent clocks/weather.
- **Player impact:** High; the command can contradict darkness, moon, and weather events.
- **Fix sketch:** Expose a locked read-only game snapshot; delete session state and render the snapshot.

### O17 — Delegated `get` misreads TAKE bit and mishandles container-all grammar

- **C:** `CAN_WEAR(obj, ITEM_WEAR_TAKE)` is a bit test, and second-argument container modes are honored (`src/act.item.c:158-175`, `src/act.item.c:346-427`).
- **Go-session:** Correctly delegates (`pkg/session/cmd_inventory.go:256-268`).
- **Go-game:** Treats TAKE as `wearFlagWord == 1` and executes room `get all` before considering arg2 (`pkg/game/item_transfer.go:21-31`, `pkg/game/item_transfer.go:92-123`).
- **Divergence:** Combined wear flags fail TAKE and `get all container` targets the room.
- **Player impact:** High; ordinary loot can be unpickable and a common container form does the wrong action.
- **Fix sketch:** Bit-test TAKE and parse/resolve the optional container before dispatching dot modes.

### O18 — Session `drop` bypasses C grammar and NODROP checks

- **C:** Handles coins, individual, `all`, `all.x`, and cursed/NODROP objects (`src/act.item.c:529-666`).
- **Go-session:** Finds one substring item and moves it directly (`pkg/session/cmd_inventory.go:301-342`).
- **Go-game:** Closer all/dot/NODROP path exists (`pkg/game/item_transfer.go:225-300`) but lacks coins.
- **Divergence:** Live command bypasses game checks and most C forms.
- **Player impact:** Medium-high; cursed items can be dropped and bulk/coin forms fail.
- **Fix sketch:** Export one corrected game drop operation and return semantic event data to session.

### O19 — Session equipment commands bypass the closer game equipment rules

- **C:** Wear/remove/wield/grab have command-specific grammar and slot/weight/two-hand/NODROP checks (`src/act.item.c:1584-1789`).
- **Go-session:** Four independent handlers call generic equipment APIs and pre-unequip wield/hold (`pkg/session/cmd_inventory.go:86-254`).
- **Go-game:** `doWear`/`performWear` carry the closer rules, with only partial remove and no complete wield/hold entry points (`pkg/game/item_equipment.go:85-156`, `pkg/game/item_equipment.go:195-310`).
- **Divergence:** Live handlers omit grammar and command-specific constraints and are not atomic.
- **Player impact:** Medium-high; wrong slot selection, bypassed constraints, or loss of current hand setup on failed replacement.
- **Fix sketch:** One explicit-slot atomic game equipment operation, complete C grammar, and a result DTO for session events.

### O20 — Delegated `give` does not transfer objects to mobs

- **C:** `perform_give` transfers the object to the victim, including NPCs, then triggers behavior (`src/act.item.c:669-705`, `src/act.item.c:767-830`).
- **Go-session:** Correctly delegates (`pkg/session/cmd_inventory.go:271-283`).
- **Go-game:** Mob path finds the object and calls `ongive`, but never moves it; without a script it refuses (`pkg/game/item_transfer.go:451-475`).
- **Divergence:** Script notification substitutes for the transfer.
- **Player impact:** High; quests/spec-procs can observe a give while the player retains the item, or ordinary NPC gives fail.
- **Fix sketch:** Perform validated transfer first (with rollback), then invoke the mob hook; restore all/dot forms.

### O21 — Eat/drink condition helpers are stubs and consumables mutate prototypes

- **C:** Consumption updates FULL/THIRST/DRUNK and instance container/food values (`src/act.item.c:895-1156`).
- **Go-session:** Calls helpers and then mutates `item.Prototype.Values` (`pkg/session/eat_cmds.go:62-109`, `pkg/session/eat_cmds.go:211-255`).
- **Go-game:** Helpers return constants without changing the player (`pkg/game/act_item_stubs.go:3-15`); instance-safe `SetValue` exists (`pkg/game/object.go:432-455`).
- **Divergence:** Core effects are absent and mutable state is written to a shared prototype.
- **Player impact:** High; hunger/thirst/drunk gameplay is wrong and using one instance can alter every copy of that vnum.
- **Fix sketch:** Implement one canonical consumable transaction using player condition APIs and copy-on-write object values.

### O22 — `fill` is missing; `pour` corrupts prototype state and omits C effects

- **C:** `fill` shares `do_pour`; pouring out creates a puddle and maintains liquid state/weight/name (`src/act.item.c:1159-1335`).
- **Go-session:** Registers only pour and delegates it (`pkg/session/commands.go:98-103`, `pkg/session/cmd_misc.go:90-94`).
- **Go-game:** Bridge hard-codes pour and implementation writes `Prototype.Values`; no puddle/fill branch (`pkg/game/act_other_bridge.go:83-84`, `pkg/game/item_consumable.go:8-110`).
- **Divergence:** A C command is absent and existing liquid mutations are shared across instances/incomplete.
- **Player impact:** High; missing gameplay and cross-instance liquid corruption.
- **Fix sketch:** Canonical fill/pour transaction using `GetValue`/`SetValue`, puddle creation, and result events.

### O23 — `examine` omits C contents/equipped targeting while adding non-C stats

- **C:** Calls target look, searches equipment too, and shows contents for containers/drink containers/fountains (`src/act.informative.c:1137-1164`).
- **Go-session:** Searches room/inventory but not equipped objects, prints raw-ish stats, and never looks inside (`pkg/session/examine.go:24-136`).
- **Go-game:** Has partial look primitives but no complete examine command (`pkg/game/look.go:226-311`).
- **Divergence:** In addition to DP-1084, valid targets/contents are missing and extra player/mob/item data is exposed.
- **Player impact:** Medium-high; core inspection behavior fails while implementation details leak.
- **Fix sketch:** Fold examine into the shared look observation model; render only C-visible fields and contents.

## Delegation candidates and transport design

### Recommended shape

The stable boundary is a **game-owned query/mutation result**, not game-owned text:

1. Game core validates C rules and returns an observation or mutation result (room view, object view, equipment change, door/container change, transfer).
2. WebSocket session renders the result as structured `ServerMessage`/`MsgState`.
3. Telnet renders the same result as C-compatible text.
4. Mobprogs/spec-procs may use a text adapter, but do not get a second rules implementation.

This matters because `Player.SendMessage` reaches session only as an `EventData{Type: "text"}` (`pkg/game/player_affects.go:237-253`, `pkg/session/manager.go:273-294`). Direct delegation therefore preserves bytes but loses session's semantic event types such as `drop` (`pkg/session/cmd_inventory.go:327-339`) and cannot construct the room `MsgState` expected by WS clients (`pkg/session/protocol.go:104-121`).

| Candidate | Canonical work before routing | WS/telnet obstacle |
| --- | --- | --- |
| `look` | Add room-extra lookup, full adjacent-room view, C gates, and viewer-authorized metadata to one observation model | Highest: WS needs structured room/doors/items/occupants; telnet must not render vnum for mortals. `RoomState.VNum` currently cannot express omission (`pkg/session/protocol.go:104-114`) |
| `examine` | Build on corrected look target resolution; add equipped lookup and C contents behavior | Return structured object/character/contents sections; do not use raw game text or debug fields |
| `time`, `weather` | Expose canonical locked snapshots from `pkg/game/weather.go` and remove session simulations | Low: result DTO can feed one or two text lines; no current semantic WS dependency |
| `drop` | Export game drop; add coins; retain NODROP/all forms | Medium: session currently emits semantic `drop`; game direct messages become generic text |
| `wear`, `remove`, `wield`, `hold` | Complete one atomic equipment API with explicit requested slot and C grammar | Medium: return moved item/old slot/new slot/action so session can mark dirty and emit equipment events |
| `open`, `close`, `lock`, `unlock` | First unify `systems.DoorManager`, parser exit state, and container state; both current implementations are wrong | High: WS clients need structured door state and reciprocal-exit updates, while game currently emits text |

### Residual work that delegation cannot solve

- Bucket 2 must be fixed in game: get (O17 plus DP-1092), give (O20), pour half of O22. Put showed no additional command-local divergence.
- Bucket 3 needs real implementation/session or shared-view work: exits (DP-1090), score (DP-1093/O14), inventory/equipment (O12), who (O13), where (O11), consider (O15), eat/drink (O21), fill (O22).
- Registry position/level metadata (O10/O11) remains a player-entry concern even after shared game operations exist.

## Suggested implementation order

1. Immediately gate `where` and restore Tier-A registry positions (O11/O10).
2. Fix delegated game correctness: get, give, and instance-safe consumables/pour (O17/O20/O21/O22).
3. Introduce the shared observation/result DTO boundary for look/examine, with dual renderers; close DP-1083/1084/1086/1089 and O23 together.
4. Move time/weather views onto canonical game state (O16).
5. Consolidate item/equipment/drop operations, then doors/containers; preserve semantic events (O18/O19/DP-1091).
6. Implement the residual information commands/views (DP-1090/DP-1093/O12-O15).

## Audit limitations

- Static audit only; no C oracle scenarios were executed for these candidates.
- The local checkout contains no Linear metadata/connector, so final issue dedup must be done in Linear before filing.
- Tiers B and C were not divergence-diffed.
- RNG outcomes inside `consider` were not assessed; only target/message/gating structure was in scope.
