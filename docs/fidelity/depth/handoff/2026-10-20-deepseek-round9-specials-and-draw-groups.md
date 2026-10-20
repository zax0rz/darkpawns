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
`steal`, the four `shopping_*` bodies, decline everything else). Evidence:
`spec-npc-shop-keeper-depth@1,2,3,5,8` + `TestSpecShopKeeperActorPortsC`.

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
4. **`list` dropped a drink container's liquid.** `list_object` appends
   `" of <drink>"` for an `ITEM_DRINKCON` holding liquid and then CAPs the line
   (`src/shop.c:858-862`), so C lists `A barrel of beer`; the port listed
   `A barrel`. Exposed by the shop scenario's final player `list` and fixed in
   `shopListLine` — the same scenario now pins the corrected bytes.

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
- **`dump`'s drop arm stays blocked on reachability, not on porting.** C's
  `do_drop(ch, …)` is descriptor-free, so the arm is portable; the reason it is
  still blocked is that the only charmed pet the fixture grammar can produce — a
  bought horse — can never hold the item. The stableboy presets
  `IS_CARRYING_W(horse) = 1000` / `IS_CARRYING_N(horse) = 100`
  (`src/spec_procs2.c`, "Be certain that pets can't carry items"), so C answers
  `give barrel horse` with `It can't carry that much weight.` (oracle run,
  2026-10-20). The arm needs a mobile `do_drop` **plus** a charmed non-horse pet
  vehicle; both belong to the NPC item-command surface, which is not ported.

### What the shop lane actually proves

The first draft of this round kept `list`/`buy`/`sell`/`value` blocked, claiming
they were undefined behaviour. Claude's review was right: `page_string` returns
on a NULL descriptor (`src/modify.c:430-433`), so a pet's `list` is a safe silent
no-op. The bodies are ported and the lane is proven by **draw parity** rather
than by transcript bytes: `shopping_list` and `shopping_buy` both call
`sort_keeper_objs`, whose `slide_obj` burns `number(0,5)` per unsorted item
(`src/shop.c:679`). The scenario gives the jeweler one unsorted barrel, and with
the NPC-side shop recovery disabled C is exactly one `Number(0,5)` ahead at the
pet's `list` step (`draw-parity MISMATCH … c=Number(0,5) … go=Number(0,3)`).

What stays invisible in both engines: `do_tell(keeper, …)` reaches nobody, and
the `steal` slap resolves `GET_NAME(ch)` through `one_argument`, which truncates
`"a horse"` to `"a"`. What *is* visible and matched: the keeper's closed-shop
say, the broke-temper emote, and `$n buys <item>.` (unreachable for the stable
horse only because of its carry state).

## Oracle evidence

Five scenarios, each diffed against the C oracle with `--draw-parity` on seeds
1, 2, 3, 5 and 8 — all `result: no normalized divergence` and
`draw-parity MATCH` (25/25):

```
spec-npc-quan-lo-depth                 seed 1   MATCH  entries=4082 stream=4139
spec-npc-quan-lo-depth                 seed 2   MATCH  entries=4077 stream=4134
spec-npc-quan-lo-depth                 seed 3   MATCH  entries=4078 stream=4135
spec-npc-quan-lo-depth                 seed 5   MATCH  entries=4078 stream=4135
spec-npc-quan-lo-depth                 seed 8   MATCH  entries=4077 stream=4134
spec-npc-elements-guardian-depth       seed 1   MATCH  entries=4114 stream=4176
spec-npc-elements-guardian-depth       seed 2   MATCH  entries=4116 stream=4178
spec-npc-elements-guardian-depth       seed 3   MATCH  entries=4116 stream=4178
spec-npc-elements-guardian-depth       seed 5   MATCH  entries=4109 stream=4171
spec-npc-elements-guardian-depth       seed 8   MATCH  entries=4116 stream=4178
spec-npc-elements-minion-depth         seed 1   MATCH  entries=4080 stream=4132
spec-npc-elements-minion-depth         seed 2   MATCH  entries=4079 stream=4131
spec-npc-elements-minion-depth         seed 3   MATCH  entries=4079 stream=4131
spec-npc-elements-minion-depth         seed 5   MATCH  entries=4080 stream=4132
spec-npc-elements-minion-depth         seed 8   MATCH  entries=4079 stream=4131
spec-npc-carrion-depth                 seed 1   MATCH  entries=4078 stream=4130
spec-npc-carrion-depth                 seed 2   MATCH  entries=4074 stream=4126
spec-npc-carrion-depth                 seed 3   MATCH  entries=4075 stream=4127
spec-npc-carrion-depth                 seed 5   MATCH  entries=4074 stream=4126
spec-npc-carrion-depth                 seed 8   MATCH  entries=4074 stream=4126
spec-npc-dump-depth                    seed 1   MATCH  entries=4074 stream=4108
spec-npc-dump-depth                    seed 2   MATCH  entries=4072 stream=4106
spec-npc-dump-depth                    seed 3   MATCH  entries=4073 stream=4107
spec-npc-dump-depth                    seed 5   MATCH  entries=4072 stream=4106
spec-npc-dump-depth                    seed 8   MATCH  entries=4073 stream=4107
spec-npc-shop-keeper-depth             seed 1   MATCH  entries=4087 stream=4154
spec-npc-shop-keeper-depth             seed 2   MATCH  entries=4087 stream=4154
spec-npc-shop-keeper-depth             seed 3   MATCH  entries=4087 stream=4154
spec-npc-shop-keeper-depth             seed 5   MATCH  entries=4087 stream=4154
spec-npc-shop-keeper-depth             seed 8   MATCH  entries=4087 stream=4154
```

Targeted census (the smallest census that can fail on this change: the five new
scenarios plus round 8's two, which share the NPC dispatch path):

```
deepseek-r9-specials @07e908821: scenarios=7 passed=7 expected=0 unpinnable=0
  stale=0 failed=0 infra=0 timed_out=0 unstable=0 verdict=CLEAN
```

Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-20/deepseek-r9-specials/`.

Revert controls, each run to a red report:

| control | result |
|---|---|
| comment out all eleven `RegisterActorSpec` calls | `spec-npc-quan-lo-depth`, `spec-npc-elements-guardian-depth`, `spec-npc-elements-minion-depth`, `spec-npc-carrion-depth` and `spec-npc-dump-depth` all red |
| disable the NPC-side shop-index recovery | `spec-npc-shop-keeper-depth` red on draw parity: C is one `Number(0,5)` ahead at the pet's `list` |

`make fidelity-depth` passes: 6,708 cases, 6,617 proven/delegated, 0 errors.

## Gates

Run individually, each checked for its own exit status:
`make fmt` · `go build ./...` · `go vet ./...` · `go test ./...` (43 packages,
0 failures) · `golangci-lint run ./...` (0 issues) · `make deadcode-ratchet` ·
`make fidelity-depth` · `make fidelity-units` · `make string-census` ·
`gosec -severity=high -quiet ./...` — all exit 0.

## Part 2 — the coincidence greens

The 2026-10-18 batch census
(`~/Archives/darkpawns/oracle-runs/2026-10-18/batch-z/draw-results.tsv`) has 36
DRAW rows over 18 scenarios. Per the brief they were grouped by first divergent
draw before any fix (`--draw-parity`, which prints the C call, the Go call and
the step each engine was in).

| first divergent stream index | scenarios (seeds) | what the first divergence is |
|---|---|---|
| 106 | `carve-depth` (1,2,3,5,8) | C `Number(1,20)=9` in warmup vs Go `Number(0,3)=1` in probe step 1 |
| 54 / 55 | `house-depth` (1,2,3,5,8) | C `Number(0,3)=0` in warmup vs Go `Number(1,6)=1` in setup |
| 40 | `circle-aware-depth`, `checkload-depth` (2,3,5,8) | same index in two unrelated scenarios |
| 19–24 | `peek-gates-depth` (19), `sedit-session-depth` (22), `sedit-entry-depth` (23), `idlist-depth` (23/24) (3,8) | a tight cluster one to five draws apart |
| 37 | `will-zen-meditation-depth` (3,8) | not yet diagnosed |
| 64 / 65 | `mudlog-crash-load-entry` (3,8) | not yet diagnosed |
| 54 | `shop-buy-weight-sell-steal` (1) | not yet diagnosed |
| 22 | `shop-live-inventory-gate`, `shop-live-inventory-filters` (1) | C `Number(0,900)=759` vs Go `Number(0,20)=17`, both in setup |
| 42 / 20 / 21 | `spec-proc-key-seller` (1), `spec-proc-key-seller-non-owner` (1), `spell-charm-shopkeeper-depth` (1) | not yet diagnosed |
| 30 | `spec-proc-beholder-command` (1) | not yet diagnosed |
| 35 | `spec-proc-cityguard` (5) | not yet diagnosed |

### carve-depth: the first divergence is a combat attack roll

`carve-depth` diverges at index 106 on every seed it runs, the strongest signal
in the set. C's draw is `number(1, 20)` — `hit()`'s attack roll
(`src/fight.c:1815`) — and Go's draw at the same position is `number(0, 3)`, the
hide-clear roll that precedes the next command (`src/interpreter.c:883-949`,
`pkg/game/npc_command.go`). C therefore rolls one more attack roll during the
warmup's ten `~dpclock pulse 20` rounds than the port does. `hit()` makes that
roll unconditionally once attacker and victim share a room: its only pre-roll
early return is the different-room stop, which also ends the fight
(`src/fight.c:1774-1780`). The port's round skipped one swing.

The cause is in the combat round's call accounting, not in the carving handler
and not in a special proc. Owner for the fix: `pkg/combat` round / `hit`
accounting; `carve-depth` is the vehicle.

### house-depth and the shop rows: the first divergence is inside setup

Both diverge during the setup phase — `Number(0,3)` vs `Number(1,6)` for
`house-depth`, `Number(0,900)` vs `Number(0,20)` for the shop pair — i.e. before
the probed commands run. Setup draws are login, room reset and shop stock
generation, which is the stop tier the brief names (`DP_CLOCK` accounting, login,
persistence). They are written up here rather than chased, with the exact call
pairs and stream indices, so the next round can re-run one scenario per pair and
name the call site.

### Row ownership

| scenario | rows | state | owner |
|---|---|---|---|
| `carve-depth` | 5 | open — diagnosed to `hit()`'s attack roll | combat round accounting |
| `house-depth` | 5 | open — setup-phase draw substitution | setup / login lane |
| `circle-aware-depth` | 4 | open | to diagnose |
| `checkload-depth` | 2 | open (index 40, same as circle-aware) | to diagnose |
| `peek-gates-depth` | 2 | open (index 19) | to diagnose |
| `sedit-session-depth`, `sedit-entry-depth` | 4 | open (indices 22, 23) | to diagnose |
| `idlist-depth` | 2 | open (indices 23, 24) | to diagnose |
| `will-zen-meditation-depth` | 2 | open (index 37) | to diagnose |
| `mudlog-crash-load-entry` | 2 | open (index 64/65) | to diagnose |
| `shop-buy-weight-sell-steal`, `shop-live-inventory-gate`, `shop-live-inventory-filters` | 3 | open — setup-phase stock generation | shop stock generation |
| `spec-proc-key-seller`, `spec-proc-key-seller-non-owner`, `spell-charm-shopkeeper-depth` | 3 | open | to diagnose |
| `spec-proc-beholder-command` | 1 | open | to diagnose |
| `spec-proc-cityguard` | 1 | open | to diagnose |

No DRAW row was cleared this round and none was claimed: the one group diagnosed
to a call site (`carve-depth`) points outside this round's lane, and the two
setup-phase groups fall in the stop tier.

## Not taken this round

- `receptionist` and `cryogenicist`: stop tier (rent), unchanged.
- The 32 specials C never assigns: out of scope (R5e).

## Decisions for Zach

- **One PR (three group commits plus the handoff) instead of one PR per special
  group.** The twelve
  lanes share one helper file (`cRoomPeople`, `elementTeleportPerson`) and one
  scenario family; separate PRs would each carry the same helper and collide on
  `docs/fidelity/depth/spec-procs.tsv`. Alternative: three stacked PRs.
- **`start_room` is not given an oracle vehicle.** Alternative: fix the
  extra-description trailing-space carry-over first, which is an output/render
  change touching every room with an extra description.
- **`whirlpool` is unit-proven only.** Alternative: a `set-mob-flag … SENTINEL`
  fixture directive, which is stop tier.
- **`dump`'s drop arm stays blocked on reachability.** Alternative: port a
  mobile `do_drop` and add a charmed non-horse pet vehicle (the NPC item-command
  surface).
