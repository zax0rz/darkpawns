# DeepSeek — round 8: the NPC special lane (DP-1448)

Date: 2026-10-16 · Branch: `deepseek/round8-specials` (off `origin/main` `58e7c503a`) ·
Rules: R1, R1a, R3, R5c, R5e, R5g, R5h

Round 7's follow-up (PR #2069, merged) fixed the NPC follower loop and gave 41
specials a faithful NPC answer. This round classifies **every** remaining
`SPECIAL(...)` body in `src/` from its C source and gives each one its faithful
NPC behaviour, its recorded gap, or its stop-tier write-up.

The problem it closes, in Egon's words (2026-10-16, our most regular player):
horses "don't follow" and `order` "doesn't seem to work either, at least not
consistently". C's `special()` (`src/interpreter.c:1407-1476`) offers every typed
command and every step a mobile takes to the room's room/mob/object specials. A
special that C proves returns `FALSE` for an unnamed command must therefore
**decline**, not block; the port answered `SpecialUnsupported`, which its callers
treat as "blocked" (`movementSpecialBlocksMob`, `runCharacterSpecials`).

## Method

Every `SPECIAL(name)` in `src/*.c` (122 bodies) was read and put in one class:

| class | meaning | count added this round |
|---|---|---|
| 1a | `if (cmd || ...) return FALSE;` opener — acts only on the pulse (`cSpecialDeclinesEveryCommand`) | 1 (`roach`) |
| 1b | acts only on named commands, every unnamed path returns `FALSE` with no side effect (`cSpecialInterceptsOnly`) | 28 |
| 1c | needs the descriptor or the object receiver, so it gets an actor-object port | 1 (`gen_board`) |
| 2 | acts on or gates an NPC actor — needs an actor port with C's exact conditions | 12 |
| 2b | acts for any body in a state the actor does not control (charm), so only part of the body is portable now | 1 (`conjured`) |
| 3 | acts before it declines, or acts for every command regardless of the actor | 11 — recorded, not ported |
| 4 | stop tier (rent/persistence) | 2 — blocked |
| 5 | declared in C but never assigned, so no live body carries it | 32 — out of scope |

The mechanical scan used in round 7 anchored on a line holding **only**
`SPECIAL(name)`. Two definitions put the brace on the same line
(`SPECIAL(roach) {`, `SPECIAL(conjured) {`) and were invisible to it. Both are
classified here.

## Class 1b — `cSpecialInterceptsOnly` (1 → 29 entries)

Each entry records the commands C answers `TRUE` for. Every other command an NPC
issues is declined, which is what lets a pet move and obey. Every entry's command
set is pinned twice: against a recorded golden map, and against the cited C
function by requiring each command to appear there as `CMD_IS("<cmd>")`
(`TestSpecialInterceptTableMatchesRecordedCommandSets`), so a typo in a command
name fails the gate.

Town and shops (29 total, in the order the brief sets):

| special | C site | commands |
|---|---|---|
| `assassin` | `src/spec_procs2.c:845-924` | list, hire |
| `bank` | `src/spec_procs.c:2345-2399` | balance, deposit, withdraw |
| `pet_shops` | `src/spec_procs.c:1844-1908` | list, buy |
| `recharger` | `src/new_cmds2.c:415-486` | list, help, recharge |
| `clerk` | `src/spec_procs3.c:75-142` | list, buy |
| `con_seller` | `src/spec_procs3.c:239-314` | list, buy |
| `prostitute` | `src/spec_procs3.c:670-705` | list, buy |
| `tattoo1`-`tattoo4` | `src/spec_procs2.c:945-1008,1010-1072,1075-1137,1282-1340` | list, buy |
| `identifier` | `src/spec_procs2.c:1193-1280` | list, value, give |
| `mortician` | `src/spec_procs3.c:807-856` | list, retrieve |
| `horn` | `src/spec_procs.c:2401-2420` | use |
| `moon_gate` | `src/gate.c:242-326` | enter, look |
| `beholder` | `src/new_cmds2.c:328-398` | cast, recite |
| `cuchi` | `src/spec_procs.c:1034-1071` | pat |
| `dracula` | `src/spec_procs.c:1798-1834` | look |
| `medusa` | `src/spec_procs2.c:1552-1581` | look, examine |
| `no_get` | `src/new_cmds2.c:552-574` | get, palm, take |
| `pray_for_items` | `src/spec_procs.c:2068-2168` | pray |
| `portal_to_temple` | `src/spec_procs3.c:515-540` | say, `'` |
| `recruiter` | `src/spec_procs3.c:905-930` | kill, hit, cast, will |
| `suck_in` | `src/spec_procs.c:2279-2299` | look |
| `oro_quarters_room` | `src/spec_procs.c:2302-2318` | *(none — `IS_MOB(ch)` returns FALSE first)* |
| `oro_study_room` | `src/spec_procs.c:2321-2337` | *(none)* |
| `werewolf` | `src/spec_procs3.c:427-454` | *(none — every arm is behind `!cmd`)* |
| `elements_load_cylinders` | `src/spec_procs3.c:1058-1135` | get, drop |
| `stableboy` | `src/spec_procs2.c:315-461` | buy, list, stable, collect *(added by #2069)* |

Two C bodies need a note, not a wider entry:

- `clerk`'s zone switch runs before its command gate and its `default:` arm writes
  to `ch` for a zone where no clerk stands (mob 18228 is zone 182; the switch
  arms are 80, 182 and 212). In every zone a clerk actually occupies the write
  cannot fire.
- `tattoo1`'s buy arm has an `IS_NPC(ch)` guard that returns FALSE; the other
  three do not. None of the four reaches a write or a draw for an unnamed
  command, which is the property the table records.

## Class 1c — the object lane

`DispatchCharacterObjectSpecial` required an actor-aware callback and answered
`SpecialUnsupported` otherwise, so **every** object special without one blocked
every NPC command in its room — the KD and Kir-Oshi ATMs (obj 8034/18224,
`bank`), the Silent Tower horn (obj 14415), the moon gates (obj 4001-4002,
19604-19611) and the boards. The object lane now consults the same two tables as
the character lane, and `gen_board` gets its own port for the one gate a table
cannot express:

- `gen_board`, `src/boards.c:179-210`: C's first statement is `if (!ch->desc)
  return 0;`, so a descriptor-less body declines every command outright; a
  switched body keeps C's `CMD_WRITE/CMD_LOOK/CMD_EXAMINE/CMD_READ/CMD_REMOVE`
  gate and the named commands stay an explicit gap.

## Class 2 — actor ports (12 added; `no_move_east` from #2069)

Each port runs C's command-time body against the real actor.
`pkg/game/actor_specials_guards.go` holds the movement guards; the player lane
keeps its own callback.

| special | C site | NPC behaviour ported |
|---|---|---|
| `castle_guard_down` | `src/spec_procs2.c:2134-2174` | the guarded direction blocks a body that owns no house; C's grouped-owner arm passes with its own acts and still declines; every other command declines |
| `castle_guard_up` | `src/spec_procs2.c:2176-2216` | as above; C's `+1`/`+0` house offsets preserved |
| `castle_guard_north` | `src/spec_procs2.c:2218-2258` | as above |
| `chosen_guard` | `src/spec_procs2.c:2113-2132` | `IS_CHOSEN` is `!IS_NPC(ch) && ...` (`src/utils.h:422`), so only the move/awake/immortal gates and the south block carry over |
| `jailguard` | `src/spec_procs.c:1781-1796` | room 8118 and north only; C's act is rooted at the actor |
| `outofjailguard` | `src/spec_procs.c:1763-1778` | room 8117 and south only |
| `guild_guard` | `src/spec_procs.c:569-608` | the flee arm writes and consumes; the guild_info loop blocks a mobile because C's wrong-class test is `IS_NPC(ch) \|\| GET_CLASS(ch) != guild_info[i][0]`; a fighting guard still takes `fighter(guard, guard, 0, NULL)` |
| `wall_guard_ns` | `src/spec_procs2.c:2260-2300` | every typed command takes `return (fighter(ch, ch, 0, NULL))`, which declines for a body that is not fighting |
| `no_move_west` | `src/spec_procs2.c:2033-2052` | the C gate pair, the immortal/hunting gate, then the westerly block |
| `no_move_south` | `src/spec_procs2.c:2092-2111` | as above, south |
| `no_move_north` | `src/spec_procs2.c:2054-2073` | as above, north. C declares it at `src/spec_assign.c:143` with no `ASSIGNMOB` row, so no live mobile carries it; the port keeps the class complete for the same reason the player lane does |
| `newbie_zone_entrance` | `src/spec_procs.c:2262-2274` | the actor's own level decides: below `NEWBIE_LEVEL` passes, at or above it is turned back, an immortal passes |

`conjured`, `src/spec_procs3.c:859-892` (class 2b): a charmed conjured creature
returns `FALSE` for every command and does nothing — the state a summoned
elemental is normally in — so its room now declines instead of blocking. The
uncharmed branch (the act/extract pair at `:875-888`) is **not** ported and stays
an explicit unsupported gap rather than a silent decline.


## Class 3 — recorded, not ported (11)

Each of these acts for an actor it does not select, or acts *before* it declines,
so a command-set entry would be a fabrication. They remain `SpecialUnsupported`
for an NPC actor, which keeps the port's block honest and the gap visible.

| special | C site | why it needs more |
|---|---|---|
| `carrion` | `src/spec_procs2.c:1726-1754` | the gate is the command's **argument** (`strstr(argument, "corpse"/"corpses"/"pile")`), not its name; a matching argument spawns mob 14308 and attacks |
| `dump` | `src/spec_procs.c:254-293` | destroys every object in the actor's room on **any** command, then declines the non-drop ones |
| `start_room` | `src/spec_procs.c:2204-2258` | walks the room's people and teleports every mortal, then declines |
| `whirlpool` | `src/spec_procs2.c:244-275` | no `cmd` test at all: teleports every non-NPC, non-NOHASSLE body and returns TRUE when it moved someone |
| `quan_lo` | `src/spec_procs3.c:372-397` | gossips on flee/retreat/escape and acts on `look`/`examine <name>`, then always returns FALSE |
| `elements_platforms` | `src/spec_procs3.c:1004-1024` | teleports everyone to 1314 on every call, returns TRUE |
| `elements_master_column` | `src/spec_procs3.c:936-1002` | inspects carried talismans and teleports, on every call |
| `elements_galeru_column` | `src/spec_procs3.c:1137-1182` | checks four talisman rooms and teleports every non-NPC, on every call |
| `elements_galeru_alive` | `src/spec_procs3.c:1184-1215` | teleports everyone when mob 1315 is absent, on every command |
| `elements_guardian` | `src/spec_procs3.c:1242-1287` | charms players through song and makes them hit each other, on every command |
| `elements_minion` | `src/spec_procs3.c:1217-1240` | destroys its own talismans on every call — the `act` and `extract_obj` are the whole body |

Damage, teleports and object destruction for a *mobile* actor are exactly the
changes that need their own scenario and review, and none is on the playtest
path (a pet does not walk into the Elemental Temple's column rooms).

## Class 4 — stop tier (2)

`receptionist` (`src/objsave.c:1199-1202`) and `cryogenicist`
(`src/objsave.c:1205-1208`) are the rent/persistence surfaces. They stay blocked
with the surface named: rent, the saved-object list and the cryo store are the
port's `pkg/db` boundary.

## Class 5 — declared in C, never assigned (32 + `hisc`)

No `ASSIGNMOB`/`ASSIGNOBJ`/`ASSIGNROOM` row carries them, and those macros plus
the two boot-time loops (`assign_the_shopkeepers` `src/shop.c:1226-1243` and
`dts_are_dumps` `src/spec_assign.c:638-641`) are the only assignment mechanisms
in the C tree, so no live body ever runs them. They are out of scope for an NPC
lane (R5e): `alien_elevator`, `bat`, `bat_room`, `black_horn`,
`black_undead_knight`, `brass_dragon`, `castle_guard_east`, `conductor`,
`elemental_room`, `elevator`, `enter_circle`, `evillead`, `eviltrade`,
`fearface`, `ira`, `itoh`, `little_boy`, `mallory`, `mayor`, `mickey`, `mirror`,
`ninelives`, `no_move_down`, `portal_room`, `red_undead_knight`, `summoner`,
`tipster`, `turn_undead`, and `hisc` (whose row is commented out at
`src/spec_assign.c:382`). Ten of them are also inert entries in
`cSpecialDeclinesEveryCommand`; that table is only consulted on the dispatch
path, so an unassigned name changes nothing.

`shop_keeper` (`src/shop.c:942-1002`) is assigned **at boot** rather than by a
macro, so it is absent from the port's static `MobSpecAssign`; the player lane
recovers it from the shop index at dispatch (`pkg/session/commands.go:977-981`)
and the NPC lane does not. C therefore runs `shop_keeper` for an ordered pet and
the port skips it. That is a missing-C-behaviour gap rather than a block: C
declines every command but `steal`/`buy`/`sell`/`value`/`list`, which is what the
skip does. Recorded for a follow-up round — it is a shop surface, not a movement
one.

## Truth found this round (R5c class audits)

Three findings came out of the classification itself, each fixed with the class
it belongs to:

1. **The round-7 mechanical scan had a hole.** It anchored on a line holding only
   `SPECIAL(name)`, so the two same-line definitions were invisible. `roach`
   (`src/spec_procs3.c:707-805`, `if (cmd || !AWAKE(ch)) return (FALSE);`) joined
   `cSpecialDeclinesEveryCommand`; `conjured` is class 2b above. The pulse-only
   gate `TestPulseOnlyTableHasCCommandGate` now re-derives the shape from the C
   body for every entry — a bare `cmd` operand in an `if` whose statement returns
   FALSE/0, with no `act`/`number`/`send_to_char`/state change before it.
2. **The player lane for `no_move_east`/`west`/`south` was missing C's gates.**
   C's body is `if (!cmd || mini_mud || !IS_MOVE(cmd) || !AWAKE(mobile)) return
   FALSE ;` then `if (GET_LEVEL(ch) >= LVL_IMMORT || HUNTING(ch)) return FALSE;`.
   The port's player callbacks had neither, so an **immortal or hunting leader,
   or anyone beside a sleeping guard, was refused where C lets them pass**. The
   new guard scenario found it: C walked the immortal leader south and the port
   printed "A burly guard blocks your way." Both lanes now share one body per
   direction (`noMoveGuardLegacy` / `specNoMoveGuardActor`), and
   `TestNoMoveGuardPlayerLaneMatchesC` pins the immortal, hunting, sleeping-guard,
   wrong-direction and non-movement cases.
3. **The object lane blocked every registered object special for an NPC.** Any
   object whose C body declines unnamed commands (the ATMs, the horn, the moon
   gates) refused every command a pet was given. Fixed in
   `DispatchCharacterObjectSpecial` and proven by `TestObjectLaneConsultsDeclineTables`
   plus the `spec-npc-decline-depth` scenario.

`no_move_north`'s player callback also printed an invented line ("You try to go
north but are blocked by a heavy object."); it now uses C's block pair. The
special is unassigned in C (class 5), so no live path changed.


## Oracle evidence

Two new scenarios, each diffed against the C oracle with `--draw-parity` on seeds
1, 2 and 3. All six runs report `result: no normalized divergence` and
`draw-parity MATCH`.

```
spec-npc-decline-depth  seed 1  MATCH  entries=4080 stream=4146
                        seed 2  MATCH  entries=4080 stream=4146
                        seed 3  MATCH  entries=4079 stream=4145
spec-npc-guard-depth    seed 1  MATCH  entries=4082 stream=4120
                        seed 2  MATCH  entries=4085 stream=4123
                        seed 3  MATCH  entries=4082 stream=4120
```

`spec-npc-decline-depth` (`cmd/dp-oracle-diff/scenarios/spec-npc-decline-depth.txt`)
buys a horse, orders it, and walks it out and back in room 8105 with a recruiter
mob (mob 16300, `SPECIAL(recruiter)`) and the KD ATM (obj 8034, `SPECIAL(bank)`)
present. C's transcript:

```
--- [order horse emote neighs]        --- [north]
Okay.                                 ...continues to the north and south.
A horse neighs                         A horse arrives from the south.
--- [order followers say hello]       --- [look]
A horse says, 'hello'                 A horse stands here, flicking its tail.
Okay.
```

`spec-npc-guard-depth` (`.../spec-npc-guard-depth.txt`) buys a horse beside mob
16308 (`SPECIAL(no_move_south)`). The primary is immortal, so C's
`GET_LEVEL(ch) >= LVL_IMMORT` gate lets the leader walk south alone while the
horse is refused, and the same guard declines every other command, so the horse
obeys an order and follows north:

```
--- [south]                           --- [north]        (back in 8105)
The Avenue of the Gods                ...A horse stands here, flicking its tail.
  ...avenue continues to your         A big, burly guard dressed in all black
  east and west.                        guards the door.
(no horse follows)                    --- [north]          (out again)
--- [look]                            ...continues to the north and south.
The Avenue of the Gods                A horse arrives from the south.
  ...avenue continues to your         --- [look]
  east and west.                      A horse stands here, flicking its tail.
```

Revert controls, each run to a red report:

| control | result |
|---|---|
| remove the object lane's table consult | red — the horse neither obeys nor follows (`/tmp/dp-r8-run/revert1.log`, kept out of the repo) |
| remove the `recruiter` table entry | red — the same two steps drop |
| remove the `no_move_south` actor registration | red — the ordered `emote`/`say` are dropped and the horse fails to follow north |
| the pre-fix player lane (before finding 2) | red — C walks the immortal leader south; the port blocks them |

`make fidelity-depth` passes: 41 maps, 0 errors, 5,966 cases, 5,880 proven.
The four new oracle rows and ten unit rows are in
`docs/fidelity/depth/spec-procs.tsv`; the four new case ids are annotated in the
two scenario files, which the depth generator cross-checks.

## Gates

Run individually, each checked for its own exit status:
`make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`,
`make fidelity-units`, `make string-census`, `gosec -severity=high -quiet ./...`.

## Not taken this round

- The 11 class-3 specials (teleports, object destruction and charm-song for a
  mobile actor). Each needs its own scenario and a decision about whether a pet
  should be able to trigger them.
- `shop_keeper`'s NPC lane (the boot-time assignment), recorded above.
- `receptionist`/`cryogenicist` (stop tier).
- The 32 unassigned bodies: out of scope until C assigns them (R5e).

