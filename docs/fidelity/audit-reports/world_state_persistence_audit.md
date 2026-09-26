# World-State Persistence Audit — the invented `data/world_state.json` snapshot

**Status:** resolved. The snapshot is gone; boot now reproduces C's reset contract.
**Scope:** all non-player world persistence in the Go server, checked against the
canonical C boot/reboot behavior.
**Rules applied:** R1 (player-facing bytes are law), R4 (no invention), R5c
(find one discrepancy, inspect the whole class), R5e (verify the call path).

---

## 1. What C actually does

The C server keeps **no transient world state across a restart**. The entire
reboot contract is:

| Where | C evidence | Effect |
|---|---|---|
| Boot: static world | `src/db.c:264-299` `boot_world()` → `boot_world_files()` → `index_boot()` on `.zon`/`.wld`/`.mob`/`.obj`/`.shp` | rooms, mobiles, objects, shops are re-read from the area files |
| Boot: gossip buffer | `src/db.c:250-261` `init_review_strings()`, called at `src/db.c:293` | all 25 `review[]` slots are blanked — `review` shows nothing after boot |
| Boot: zone reset | `src/db.c:304-392` `boot_db()`; `src/db.c:391` `reset_zone(i)` for every zone | mobiles (`'M'`, `src/db.c:2107`) and objects (`'O'`/`'P'`/`'G'`/`'E'`, `src/db.c:2149`…) are re-placed from the reset table, stale ones removed (`'R'`, `src/db.c:2220`), doors forced (`'D'`, `src/db.c:2247-2281`), and `zone_table[zone].age = 0` (`src/db.c:2285`) |
| Boot: time + weather | `src/db.c:311` `reset_time()`; `src/db.c:415-460` | clock derived from the wall clock, weather re-derived by `dice()`; only year/month/day comes from `etc/date_record` (`src/db.c:3197-3219`) |
| Boot: other subsystems | `src/db.c:332-405` | help, player index, mail (`scan_file()` 367), bans (373-374), dns (377), houses (`House_boot()` 402), clans (`init_clans()` 405) |
| Shutdown | `src/comm.c:288-293` | `save_clans()` (289), `close_whod()` (290), `write_mud_date_to_file()` (291), `fclose(player_fl)` (293) — **no world snapshot is written** |

Everything a player can observe about the *world* (ground objects, corpses,
loose money, ash, mob position and HP, door state, room secret marks, recent
gossip, zone ages) is either re-derived or destroyed by that sequence. Shops have
no save path at all (`src/shop.c` contains no `fopen`).

## 2. What the Go server did instead

`pkg/game/save.go` carried a second, invented persistence system:

- `SerializeWorld` / `DeserializeWorld` / `SaveWorld` / `LoadWorld` and the
  structs `saveWorldData`, `saveMobPosition`, `saveItemRef`, `saveGossipEntry`.
- Shutdown called `game.SaveWorld(gameWorld)` (`cmd/server/main.go`).
- Boot called `game.LoadWorld(gameWorld)` after zone resets (`cmd/server/main.go`).
- `POST /admin/save-world` (Huma op `save-world`, plus the already-dead
  `handleSaveWorld` helper) triggered the same write on demand; `admin-ui`
  exposed it as "Save World State".

`LoadWorld` replayed the file through a prototype lookup keyed on `parser.Obj`
VNum. Synthetic objects carry `VNum == -1` and have no prototype
(`pkg/game/death.go` money at `:887`, corpse at `:1001`), so every corpse, coin
pile and ash object produced
`slog.Warn("DeserializeWorld: unknown obj vnum", "vnum", ref.VNum)` — the
~3,200 startup warnings. The `gossip` section is why `review` showed pre-restart
gossip.

## 3. Classification

Class codes: **1** player data C persists · **2** world data C persists ·
**3** player-invisible operational data · **4** invented player-visible runtime
persistence (removed) · **5** ambiguous, insufficient C evidence.

| Go field / subsystem | Go save fn | Go restore fn | Player-visible effect | C boot/reboot behavior | Class | Action | Exact C evidence |
|---|---|---|---|---|---|---|---|
| `saveWorldData.DoorStates` | `SerializeWorld` | `DeserializeWorld` | a door a player left locked stays locked over a reboot | rooms re-read from `.wld`, then reset `'D'` forces the table's state | 4 | removed | `src/db.c:2247-2281` |
| `saveWorldData.Mobs` (`vnum`,`id`,`room_vnum`,`current_hp`,`max_hp`) | `SerializeWorld` | `DeserializeWorld` | a mob stays where a player left it, at the HP they left it | reset `'M'` respawns mobiles at the reset room at full HP | 4 | removed | `src/db.c:2107-2118`, `:2086` |
| `saveWorldData.RoomItems` prototype entries | `SerializeWorld` | `DeserializeWorld` | dropped objects survive a reboot | world re-read; reset `'O'`/`'R'` are authoritative | 4 | removed | `src/db.c:2149-2219`, `:2220-2246` |
| `saveWorldData.RoomItems` entries with VNum `-1` (corpse/money/ash) | `SerializeWorld` | `DeserializeWorld` (prototype miss → warn) | corpses and coin piles survive; ~3,200 warnings | synthetic objects have no prototype and are never persisted; a reboot loses them | 4 | removed, and the per-entry warning class with it | `src/fight.c` `make_corpse()`; `src/db.c` `read_object()` |
| `saveWorldData.Gossip` | `SerializeWorld` | `DeserializeWorld` | `review` shows pre-restart gossip | `init_review_strings()` blanks all 25 slots at boot | 4 | removed | `src/db.c:250-261`, `:293` |
| `saveWorldData.NextMobID` / `NextObjID` | `SerializeWorld` | `DeserializeWorld` | none (internal instance ids; C has no equivalent) | index/rnum tables rebuilt by `index_boot()` | 4 | removed (fresh counters each boot) | `src/db.c:266-299` |
| `POST /admin/save-world` (Huma `save-world`, `handleSaveWorld`) | `game.SaveWorld` | n/a | none — operator HTTP surface | no C equivalent exists | 4 | removed from router, Huma spec, admin-ui and tests | n/a — invented surface |
| `pkg/admin/data/world_state.json` (tracked orphan) | n/a | n/a | none | none | 4 | deleted (no code referenced it) | n/a |
| Player file / game store (`savePlayerData`, `SavePlayer`, `LoadPlayer`, `pkg/db`) | `SavePlayer`, game store | `LoadPlayer`, game store | level, stats, gold, inventory, equipment, room, affects, bank | `save_char()` → player file; `fclose(player_fl)` at shutdown; players are **not** reset by a reboot | 1 | retained | `src/db.c` `save_char`/`load_char`; `src/comm.c:293` |
| Player inventory & equipment (`SaveItemData`) | `SavePlayer` | `RestoreItemsFromSave` | carried/worn items persist | C saves carried + equipped objects per player | 1 | retained | `src/objsave.c` |
| Rent / crash-save | `objsave.go` | `objsave.go` | rented items restored | `update_obj_file()` and rent files at boot | 1 | retained | `src/db.c:379-385`, `src/objsave.c` |
| Clans (`clan.c`) | `clans.go` | `clans.go` | clan roster, banks, titles | `save_clans()` on shutdown and on change; `init_clans()` at boot | 2 | retained | `src/clan.c:808-809`, `src/db.c:405`, `src/comm.c:289` |
| Houses & house contents (`house.c`) | `house_save.go` | `HouseBoot` | house ownership and stored items | `House_boot()` at boot; `House_save`/`HCONTROL_FILE` writes | 2 | retained | `src/db.c:402`, `src/house.c:83,158,259` |
| Mail (`mail.c`) | `mail.go` | `mail.go` | mail messages | `scan_file()` at boot; direct `r+b` writes | 2 | retained | `src/db.c:367`, `src/mail.c:141,174,227` |
| Boards (`boards.c`) | `boards` package | `boards` package | board posts | binary board files read/written per board | 2 | retained | `src/boards.c:454,491` |
| Site bans / invalid names (`ban.c`) | `bans.go` | `bans.go` | connection refusal, name rejection | `load_banned()` + `Read_Invalid_List()` at boot; `ban.c` writes | 2 | retained | `src/db.c:373-374`, `src/ban.c:62,122,295` |
| Aliases (`alias.c`) | `aliases.go` | `aliases.go` | player aliases | per-player alias file written on change, read at login | 1 | retained | `src/alias.c:52,83` |
| DNS cache (`etc/dns`) | `dns.go` | `dns.go` | immortal `dns` output only | `boot_dns()` at boot; wizard add/delete rewrites the file | 3 | retained | `src/db.c:377,1780` |
| `whod` display flags (`whod.c`) | `whod.go` | `whod.go` | immortal `whod` output only | whod file read at boot, `close_whod()` at shutdown | 3 | retained | `src/comm.c:290`; `src/whod.c` |
| Zone reset state (`age`, `lifespan`, dispatcher timers) | none | none | respawn cadence only | `reset_zone()` sets `age = 0`; `zone_point_update()` ages zones in memory | conforms — never in the snapshot | unchanged | `src/db.c:2285`, `:1990-2022` |
| Room transient flags (`ROOM_SECRET_MARK`) | none (in-memory `room.Flags`) | none | a detected secret exit | reset `'D'` clears `ROOM_SECRET_MARK`; rooms re-parsed at boot | conforms — never in the snapshot | unchanged | `src/db.c:2252-2253` |
| Shop mutable state | none | `ShopManager` from `.shp` | prices, keeper stock | `.shp` read at boot; **no** shop write path exists in C | conforms — never in the snapshot | unchanged | `src/db.c:296-299`; `src/shop.c` has no `fopen` |
| Weather (`pressure`, `sky`, `change`) | none | derived at boot | `weather` output | `reset_time()` derives it fresh with `dice()` | conforms | unchanged | `src/db.c:446-460` |
| In-game date (year/month/day) | none | computed from the epoch | `time` output | C writes `etc/date_record` on shutdown and reads it at boot | 5 — pre-existing, documented divergence | **not changed** (see §5) | `src/comm.c:2625-2641`, `src/db.c:3197-3219` |

## 4. What changed

1. **Deleted** the invented persistence: `SerializeWorld`, `DeserializeWorld`,
   `SaveWorld`, `LoadWorld` and their save structs (`pkg/game/save.go`).
2. **Boot** (`cmd/server/main.go`): the `game.LoadWorld` call is replaced by
   `game.IgnoreLegacyWorldState()` (`pkg/game/world_state.go`), which recognises
   a leftover snapshot, logs one structured INFO line with the ignored section
   counts, and applies nothing.
3. **Shutdown** (`cmd/server/main.go`): the `game.SaveWorld` call is deleted, so
   the heartbeat-quiescence guard around it goes too. Nothing world-shaped is
   written; player profiles are still drained by `manager.ShutdownGracefully*`
   before exit, matching `save_char` + player-file close in C.
4. **Admin surface**: `POST /admin/save-world` removed end to end — router
   registration, the Huma operation, the already-dead `handleSaveWorld` helper,
   `spec_sanity_test.go`'s route table, the router tests, and the admin-ui
   `saveWorld` client method / "Save World State" button. The C-faithful
   operator action that remains is `POST /admin/reset-all-zones` → `reset_zone()`.
5. **Orphan fixture** `pkg/admin/data/world_state.json` (added by an old test
   commit, referenced by nothing) deleted.
6. **No new format.** Nothing replaces the file; a boot writes no
   `data/world_state.json` at all (`TestBootAndShutdownWriteNoWorldSnapshot`).

## 5. Ambiguities and intentionally retained persistence

- **`etc/date_record` (in-game date)** — C persists year/month/day across a
  reboot (`write_mud_date_to_file()` at `src/comm.c:2625`, read by
  `read_mud_date_from_file()` at `src/db.c:3198`). The Go port does not, and says
  so at `pkg/game/weather.go:186-188`: the epoch-derived calendar is the default
  C itself falls back to when the file is missing. Adding it is a *missing*
  persistence, not an invented one; out of scope here and left unchanged so this
  change stays a removal.
- **Zone `age` / reset cadence** — the Go `Spawner.resetEmptyZones()` cadence is
  a known pre-existing divergence (`docs/fidelity/audit-reports/db_audit.md`,
  finding 3). It is not persistence and was not touched.
- **Snapshot as crash recovery** — no operational reliance was found: nothing
  reads the file except the removed boot path; `DEPLOYMENT.md` names no
  `world_state.json` backup or restore procedure; `cmd/dp-oracle-diff` only
  isolates the file's directory so one scenario cannot poison another (comment at
  `cmd/dp-oracle-diff/main.go:181`). Player-durable data lives in the game store
  and the player files, both untouched.
- **Legacy files on disk** — deliberately **not deleted**. A leftover
  `lib/data/world_state.json` is ignored and reported once per boot; an operator
  may remove it at leisure. Deleting operator data at boot is not something C
  would do, and the file's absence is not required for correctness.

## 6. Regression tests

`pkg/game/save_world_test.go` (rebuilt) and `pkg/game/world_test.go`:

| Requirement | Test |
|---|---|
| dropped prototype-backed room objects do not survive restart | `TestRestartDoesNotRestoreLegacyWorldSnapshot/dropped_prototype_object_is_gone` |
| synthetic `VNum == -1` room objects do not survive restart | `TestRestartDoesNotRestoreLegacyWorldSnapshot/synthetic_vnum_-1_objects_are_gone` |
| corpses / money / ash follow C restart behavior | `TestCorpsesMoneyAndAshDoNotSurviveRestart` |
| mobs and zone state reset per C | `…/mob_is_at_its_reset_room`, `…/reset_table_still_places_its_own_objects` |
| temporary door/room state follows C reboot behavior | `…/door_state_comes_from_the_.wld_file` |
| recent gossip is empty after restart | `…/review_buffer_is_blank` |
| `review` cannot display pre-restart gossip | `TestReviewCannotShowPreRestartGossip` (through `DoReview`) |
| legacy snapshots with `room_items` accepted safely | `TestInspectLegacyWorldStateCountsTransientSections`, `TestInspectLegacyWorldStateToleratesBadInput` |
| legacy synthetic entries do not warn per object | `TestLegacySnapshotWithManySyntheticObjectsLogsOnce` (3,200 entries → 1 line) |
| boot survives an obsolete/corrupt snapshot | `TestIgnoreLegacyWorldStateIsNotFatalAndKeepsTheFile` |
| legitimate player persistence remains intact | `TestPlayerPersistenceIsSeparateFromWorldSnapshot` |
| legitimate separately persisted systems remain intact | unchanged suites: `clans*`, `mail*`, `house*`, `bans*`, `aliases*`, `pkg/boards`, `pkg/admin` |
| startup completes without panic; zone reset stays deterministic | `TestZoneResetInitializationIsDeterministic` |
| no new save-file schema is introduced | `TestBootAndShutdownWriteNoWorldSnapshot` |

The main restart test was verified to be **load-bearing**: reintroducing a
`DeserializeWorld`-style `room_items` replay locally makes
`dropped_prototype_object_is_gone` fail with
`room 5001 items = 1, want 0: a dropped object survived the restart`.

## 7. Operator note

A production instance that ran the old build holds
`lib/data/world_state.json`. It is harmless and ignored; boots report it once:

```
INFO ignoring legacy world snapshot; transient world state resets on boot
     path=./data/world_state.json save_version=2 mobs=N rooms_with_items=N
     room_items=N rooms_with_door_state=N door_states=N gossip_entries=N
```

Deleting it after the upgrade is optional cleanup; leaving it in place changes
nothing.

