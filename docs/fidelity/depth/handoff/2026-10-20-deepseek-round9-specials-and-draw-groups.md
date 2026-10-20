# DeepSeek — round 9: the last specials, and the coincidence green groups

Date: 2026-10-20 · Branch: `deepseek/round9` (off `origin/main` `205da67c7`) ·
Rules: R1, R1a, R3, R5c, R5e, R5g, R5h

Round 8 gave 41 specials a faithful NPC answer and classified the rest. This
round ports the eleven class-3 bodies (they act before they decline, or act on
every call) and `shop_keeper`'s boot-time lane for NPC actors, then starts on the
36 coincidence-green draw rows from the 2026-10-18 batch census.

## Part 1 — the eleven class-3 specials

Each one was read in full from `src/` and ported as **one body with two entry
points**: the player callback keeps its signature and now delegates, and a new
`ActorSpecFunc` runs the same body with the real actor. That is what keeps R5b
honest: the body a pet runs is the body a player runs.

| special | C site | actor lane | evidence |
|---|---|---|---|
| `carrion` | `src/spec_procs2.c:1726-1754` | loads mob 14308 at the actor's room with the actor's level and damroll, then attacks the actor | `spec-npc-carrion-depth@1,2,3,5,8` + `TestSpecCarrionActorSpawnsForTheActor` |
| `dump` | `src/spec_procs.c:254-293` | clears the actor's room before it reads the command, declines everything but `drop` | `spec-npc-dump-depth@1,2,3,5,8` + `TestSpecDumpActorClearsRoomAndDeclines` |
| `start_room` | `src/spec_procs.c:2204-2258` | walks the people list, moves mobiles, stops at the first immortal | `TestSpecStartRoomMovesMobilesUpToTheImmortalStop` (no oracle vehicle — see the finding below) |
| `whirlpool` | `src/spec_procs2.c:244-275` | declines when no mortal victim is left; the pull itself is player-only | `TestSpecWhirlpoolActorDeclinesWithoutAVictim` |
| `quan_lo` | `src/spec_procs3.c:372-397` | both messages name the actor; every arm declines | `spec-npc-quan-lo-depth@1,2,3,5,8` + `TestSpecQuanLoActorNamesTheActor` |
| `elements_master_column` | `src/spec_procs3.c:936-1002` | shared body, mobiles included | manifest `excluded`; actor dispatch unreachable |
| `elements_platforms` | `src/spec_procs3.c:1004-1024` | shared body, mobiles included | `TestSpecElementsPlatformsMovesMobiles`; actor dispatch unreachable |
| `elements_galeru_column` | `src/spec_procs3.c:1137-1182` | shared body (C skips NPC occupants) | manifest `excluded`; own pass unreachable |
| `elements_galeru_alive` | `src/spec_procs3.c:1184-1215` | shared body, mobiles included | manifest `excluded`; own pass unreachable |
| `elements_guardian` | `src/spec_procs3.c:1242-1289` | runs on the pet's command, damaging the players again | `spec-npc-elements-guardian-depth@1,2,3,5,8` |
| `elements_minion` | `src/spec_procs3.c:1217-1240` | scans its own inventory on the pet's command, always declines | `spec-npc-elements-minion-depth@1,2,3,5,8` + `TestSpecElementsMinionActorPortsC` |

`shop_keeper` (`src/shop.c:942-1002`, installed at boot by
`assign_the_shopkeepers`, `src/shop.c:1226-1243`) now reaches an NPC actor too:
`pkg/session/character_dispatch.go` recovers the boot-time assignment from the
world's shop index for an NPC actor exactly as the player lane already did, and
`pkg/game/actor_specials_shop.go` runs C's gates (shop room, awake keeper,
`steal`, decline everything else). Evidence: `TestSpecShopKeeperActorPortsC`;
the transcript cannot distinguish it (see "Unprovable by bytes" below).

### Reachability: which actor lanes a pet can actually reach

C's `special()` offers a command to the room special, the worn/carried objects,
the room's mobiles and the room's objects (`src/interpreter.c:1407-1478`), and a
followers' move is offered to the room special in the room it is leaving
(`src/act.movement.c:347-349`). An ordered pet can only issue a command in a room
its leader is standing in, so the leader's own `order …` command runs the same
body first. That splits the eleven:

- **Both passes act, and the bytes differ** — `carrion`, `dump`, `quan_lo`,
  `elements_guardian`, `elements_minion` (and `shop_keeper`'s decline). Proven by
  the oracle scenarios above.
- **The first pass consumes the command** — `elements_platforms`,
  `elements_master_column` (always `TRUE`), `elements_galeru_column` and
  `elements_galeru_alive` (true once armed), `whirlpool` (true whenever a mortal
  victim stands there). No pet command can reach their actor lane, so each is
  registered (the body must never *block* a pet) and recorded as `excluded`
  with the argument rather than claimed as green.
- **Only the immortal gate is reachable** — `start_room`: with an immortal in
  the room both passes stop at it, and the pet's command continues.

### Truth found while porting (R5c class audits)

1. **`roomCleanup` invented its own rendering.** The dump special's object line
   was built with `obj.GetShortDesc()+" vanishes in a puff of smoke!"`, but C
   runs `act("$p vanishes in a puff of smoke!", …)`, which renders `$p` through
   `OBJS` and CAPs the first character (`src/comm.c:2477`). The oracle prints
   `A barrel vanishes in a puff of smoke!`; the port printed `a barrel …`. Fixed
   by routing the line through `Act`, so both lanes share the corrected bytes.
2. **The element room walk skipped mobiles.** `elements_platforms`,
   `elements_master_column`, `start_room` and `elements_galeru_alive` iterated
   players only, while C walks `world[room].people`. A pet standing there was
   left behind with no acts. All four now walk one list (`cRoomPeople`), in C's
   front-inserted order.
3. **The people-list order was a Go convenience.** The old player lane forced
   the command actor to the head of the list. C's list is newest-entrant first;
   the oracle vehicles already satisfy that because the peer is teleported in
   before the actor walks in, but the unit fixtures did not and were corrected
   to mirror the vehicle (peer added first).

### Findings this round could not close

- **A room with extra descriptions loses a trailing space.** When
  `world[room].ex_description` is non-empty, `do_description` sends
  `sprintf(buf2, "%s ", description)` (`src/act.informative.c:2783`), so C's next
  act begins with that pending space. Room 8099 (the Burning Hut, `start_room`'s
  room) carries an extra description, and the oracle transcript shows
  ` A horse arrives from the south.` where the port prints it without the leading
  space. That pending space is a descriptor-buffer carry-over the port does not
  model anywhere; fixing it changes the bytes of every room with an extra
  description and belongs to the output/render surface, not to a special. It is
  why `start_room` has no oracle vehicle this round.
- **A wandering fixture mob proves nothing.** Mob 1313 and mob 12200 carry no
  `MOB_SENTINEL`, so a `spawn-mob` fixture in an open room can leave before the
  probe. The minion vehicle closes room 8106 with `replace-room-exits 8106 none`; the whirlpool has no such vehicle that still lets a pet act, so its lane is
  unit-proven only.
- **`dump`'s drop arm stays a gap.** It calls `do_drop` for the actor, and the
  port has no mobile `do_drop` (the `drop` command itself has no NPC actor
  handler). `specDumpActor` returns `SpecialUnsupported` for `drop` — an explicit
  block, never a silent partial run.
- **`shop_keeper`'s buy/sell/value/list arms stay a gap.** They are
  descriptor-facing: `shopping_list` ends in `page_string(ch->desc, …)`
  (`src/shop.c:925`) and a pet has no descriptor, so C dereferences NULL there.
  That is undefined state (R1a) and is pinned, not copied.

### Unprovable by bytes

`shop_keeper`'s NPC lane changes no transcript. Every arm writes to the
descriptor-less pet: `do_tell` reaches nobody, and the `steal` arm's
`do_action(keeper, GET_NAME(ch), cmd_slap, 0)` parses its target with
`one_argument`, which truncates `"a horse"` to `"a"`, so the slap resolves to
nobody in both engines. The lane is pinned by `TestSpecShopKeeperActorPortsC`;
no scenario is claimed for it.
