# C Oracle Boot Path: Wall-Clock Reads & PRNG Draws

**Date**: 2026-07-18
**Project**: Dark Pawns C→Go fidelity oracle
**Scope**: Read-only enumeration — no fixes, no root-cause analysis
**Key files**: `db.c`, `comm.c`, `weather.c`, `utils.c`, `random.c`, `version.c`

---

## Boot Sequence Overview

```
dp_main()                          [comm.c:173]
  init_game(port)                  [comm.c:259]
    prng_seed(time(0))             [comm.c:263]          *** PRNG SEED ***
    event_init()
    boot_db()                      [db.c:304]
      reset_time()                 [db.c:311]            *** CALENDAR INIT ***
      check_dst(1)                 [db.c:312]
      load_compile_time()          [db.c:313]
      boot_world()                 [db.c:329]
        boot_world_files()         [db.c:269]
          index_boot(DB_BOOT_ZON)  [db.c:272]
          index_boot(DB_BOOT_WLD)  [db.c:275]
          index_boot(DB_BOOT_MOB)  [db.c:284]  → parse_mobile() → parse_simple_mob()
          index_boot(DB_BOOT_OBJ)  [db.c:287]  → parse_object()
          index_boot(DB_BOOT_SHP)  [db.c:298]
      ... (help, player index, messages, assignments, spells, mail, bans) ...
      reset_zone(i) loop           [db.c:391]            *** ZONE RESETS ***
      boot_lua()                   [db.c:397]
      House_boot()
      init_clans()
      boot_time = time(0)          [db.c:407]            *** TIMESTAMP (post-reset) ***
```

---

## SECTION 1 — Wall-Clock / Calendar Reads Reachable During Boot

Every site that reads `time(0)`, `time(NULL)`, or accesses `boot_time` / `time_info` / `weather_info` before the first player finishes character creation.

### S1.1 — PRNG seed from wall-clock

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 1 | `comm.c:263` | `prng_seed(time(0))` | Seeds the CMWC PRNG with the 32-bit wall-clock timestamp. All subsequent `number()`, `dice()`, `uniform()`, and `prng_next()` calls during (and after) boot draw from this seeded state. |

### S1.2 — Game calendar derivation

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 2 | `db.c:420` | `time_info = mud_time_passed(time(0), 650336715)` | Converts real-world `time(0)` into the in-game calendar struct `time_info` (hours, day, month, year). This is the **sole source** of wall-clock→calendar mapping during boot. |

### S1.3 — Calendar→weather derivation (within `reset_time()`)

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 3 | `db.c:423–432` | `if (time_info.hours <= 4) … else if …` | Sets `weather_info.sunlight` (SUN_DARK / SUN_RISE / SUN_LIGHT / SUN_SET) from `time_info.hours`. |
| 4 | `db.c:438–445` | `time_info.moon = …` (seven-tier cascade) | Sets `time_info.moon` phase from `time_info.day`. |
| 5 | `db.c:447–451` | `weather_info.pressure = 960; … += dice(1,50)` or `dice(1,80)` | Sets `weather_info.pressure` from a base value plus a PRNG draw **gated by `time_info.month`** (months 7–12 → `dice(1,50)`, else → `dice(1,80)`). |
| 6 | `db.c:455–462` | `if (weather_info.pressure <= 980) … else if …` | Sets `weather_info.sky` (SKY_LIGHTNING / SKY_RAINING / SKY_CLOUDY / SKY_CLOUDLESS) from `weather_info.pressure`. |

### S1.4 — `read_mud_date_from_file()` (may override calendar)

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 7 | `db.c:421` | `read_mud_date_from_file()` | Reads `etc/date_record` and may overwrite `time_info.year`, `time_info.month`, `time_info.day` **after** `mud_time_passed()` at line 420. If the file is missing or corrupted, the wall-clock-derived values persist. Critically: `time_info.hours` is **never** overwritten by this file read. |

### S1.5 — `check_dst()` wall-clock read

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 8 | `db.c:3182` | `chtime = time(0); ltime = localtime(&chtime)` | Sets the global `daylight_saving_time` flag from `localtime()`. This flag is **not used** in any boot codepath that gates PRNG draws; it only affects time-display formatting in `act.informative.c`. |

### S1.6 — `load_compile_time()` wall-clock read

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 9 | `version.c:26` | `ct = time(0)` | Sets `compile_time` and formats `compile_time_str` for display/logging. Does not feed any boot logic or PRNG gating. |

### S1.7 — `boot_time` assignment (post-reset)

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 10 | `db.c:407` | `boot_time = time(0)` | Records the wall-clock boot time **after** all zone resets and other boot phases. Not used during `reset_zone()` because it is set later. |

### S1.8 — `mytime` in `reset_zone()` (DEAD — never read)

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 11 | `db.c:2082` | `mytime = time(0) - boot_time` | Declared on line 2079, assigned on 2082, **never read** anywhere in the function. `boot_time` is still 0 at this point (set at db.c:407, after the reset loop), so `mytime` would equal `time(0)`. Dead store — does not gate anything. |

### S1.9 — `time(0)` in `read_mobile()` (timestamp, not gating)

| # | File:Line | Expression | What It Feeds |
|---|-----------|------------|---------------|
| 12 | `db.c:1760` | `mob->player.time.birth = time(0)` | Sets the birth timestamp on freshly-created mob instances during zone reset. Pure timestamp — does not gate any PRNG draw. |
| 13 | `db.c:1762` | `mob->player.time.logon = time(0)` | Sets the logon timestamp on freshly-created mob instances. Pure timestamp — does not gate any PRNG draw. |

---

## SECTION 2 — PRNG Draw Sites Reachable During Boot / Zone-Reset

Every call to `number()`, `dice()`, `prng_next()`, `prng_uniform()`, `uniform()`, or `circle_random`-style helpers in the boot path. Each entry notes whether the draw's **execution count** is gated by a wall-clock / calendar value.

> **Key**: `number(from,to)` → `uniform()` → `prng_uniform()` → `prng_next()`.
> `dice(N,S)` loops N times calling `number(1,S)`.

### S2.1 — Within `reset_time()` (db.c:415–462)

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 1 | `db.c:449` | `dice(1, 50)` | **YES** — executes only when `(time_info.month >= 7) && (time_info.month <= 12)`, where `time_info.month` is derived from `mud_time_passed(time(0), …)` at line 420 (possibly overwritten by `read_mud_date_from_file()` at line 421). |
| 2 | `db.c:451` | `dice(1, 80)` | **YES** — executes in the `else` branch (months 0–6 or 13–16). Same calendar dependency as above. |

### S2.2 — Within `parse_simple_mob()` → called during `index_boot(DB_BOOT_MOB)`

These are called in `boot_world_files()` → `index_boot(DB_BOOT_MOB)` (db.c:284) → `discrete_load()` → `parse_mobile()` → `parse_simple_mob()` (db.c:1028). They fire for **every** mobile prototype with `GET_LEVEL > 15` during the mob file parse pass.

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 3 | `db.c:1056` | `MIN(number(0, statmod), 7)` — STR boost | **No** — gated by `GET_LEVEL(mob_proto+i) > 15`, a file-data value. Always executes for qualifying mobs regardless of wall-clock. |
| 4 | `db.c:1057` | `MIN(number(0, statmod), 7)` — INT boost | **No** — same condition. |
| 5 | `db.c:1058` | `MIN(number(0, statmod), 7)` — WIS boost | **No** — same condition. |
| 6 | `db.c:1059` | `MIN(number(0, statmod), 7)` — DEX boost | **No** — same condition. |
| 7 | `db.c:1060` | `MIN(number(0, statmod), 7)` — CON boost | **No** — same condition. |
| 8 | `db.c:1061` | `MIN(number(0, statmod), 7)` — CHA boost | **No** — same condition. |

### S2.3 — Within `read_mobile()` → called during `reset_zone()` 'M' command

These fire for every mobile instance created via the zone reset 'M' command (`db.c:2109`).

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 9 | `db.c:1750` | `dice(mob->points.hit, mob->points.mana) + mob->points.move` | **No** — executes only when `mob->points.max_hit == 0` (a mob-proto flag). Not calendar-gated. |
| 10 | `db.c:1754` | `number(mob->points.hit, mob->points.mana)` | **No** — executes when `max_hit != 0` (the common case). Not calendar-gated. |
| 11 | `db.c:1769` | `number(0, 1)` | **No** — executes within `if (GET_GOLD(mob))`, which depends on mob prototype data, not calendar. |
| 12 | `db.c:1770` | `number(1, 20)` (gold increase branch) | **No** — gated by `!number(0,1)` at line 1769 (itself a PRNG draw, not calendar). |
| 13 | `db.c:1772` | `number(1, 20)` (gold decrease branch) | **No** — gated by `number(0,1)` being true (else branch). Not calendar-gated. |

### S2.4 — Random room placement in `reset_zone()` 'M' command

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 14 | `db.c:2117` | `number(0, top_of_world)` | **No** — executes for zone79 randload mobs (vnum 7900–7998). Gated by mob virtual number range, not calendar. |
| 15 | `db.c:2134` | `number(0, top_of_world)` | **No** — executes for mobs with `MOB_RANDZON` flag. Gated by mob flags, not calendar. |

### S2.5 — Within `init_rare()` → called during `read_object()` (O/P/G/E commands)

`init_rare()` (db.c:1899) is called from `read_object()` (db.c:1956) for objects with `ITEM_RARE` flag. `read_object()` is called by zone-reset commands O (db.c:2152, 2163), P (db.c:2172), G (db.c:2191), E (db.c:2209).

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 16 | `db.c:1905` | `dice(1, 100) <= 20` | **No** — executes for each object affect slot with non-zero location. Not calendar-gated. However, the result **does** gate whether draw #17 fires. |
| 17 | `db.c:1919` | `number(0, 1) == 0` | **No** — executes only when `dice(1,100) <= 20` (20% chance, itself a PRNG draw). Not calendar-gated. |

### S2.6 — Within `percent_load()` → called during O/P/G/E commands

`percent_load()` (db.c:2065) is called for every object loaded via O (`db.c:2154`), P (`db.c:2177`), G (`db.c:2192`), and E (`db.c:2210`) commands.

| # | File:Line | Call Expression | Wall-Clock-Gated Execution Count? |
|---|-----------|-----------------|-----------------------------------|
| 18 | `db.c:2067` | `uniform() * 100.0` | **No** — executes for every qualifying object load (O/P/G/E commands where `obj_index[…].number < ZCMD.arg2`). Not calendar-gated. |

### S2.7 — Lua boot (`boot_lua()`, db.c:397)

The function `boot_lua()` registers a `number()` binding (`scripts.c:908` → `lua_number()` → `number(from,to)`) and runs `globals.lua`. If that Lua script calls `number()`, it draws from the PRNG. This is **script-dependent** — not a deterministic C call site. Not enumerated here as a concrete site.

---

## SECTION 3 — Cross-Reference: Wall-Clock-Gated Boot Draws

Only sites from SECTION 2 whose **execution** (not just outcome) is gated — directly or transitively — by a SECTION 1 wall-clock or calendar value.

### The only gated site

| SECTION 2 # | File:Line | Draw | SECTION 1 Gate | Mechanism |
|-------------|-----------|------|----------------|-----------|
| S2.1 #1 | `db.c:449` | `dice(1, 50)` | S1.2 `db.c:420` — `mud_time_passed(time(0), …)` sets `time_info.month` | `reset_time()` selects between `dice(1,50)` (months 7–12) and `dice(1,80)` (other months). The month value flows: `time(0)` → `mud_time_passed()` → `time_info.month` → (possibly overwritten by `read_mud_date_from_file()` at `db.c:421`) → conditional at `db.c:448`. |
| S2.1 #2 | `db.c:451` | `dice(1, 80)` | Same as above | Executes in the `else` branch. Mutually exclusive with #1. |

### Why only these two?

Every other PRNG draw in the boot path executes unconditionally for its qualifying entities:

- **parse_simple_mob** stat boosts (S2.2 #3–8): gated by `GET_LEVEL > 15`, a file-data constant, not calendar.
- **read_mobile** HP/gold rolls (S2.3 #9–13): gated by mob-proto fields (`max_hit == 0`, `GET_GOLD != 0`), not calendar.
- **random room placement** (S2.4 #14–15): gated by mob vnum range / flags, not calendar.
- **init_rare** stat drift (S2.5 #16–17): gated by `ITEM_RARE` flag and a 20% PRNG check, not calendar.
- **percent_load** (S2.6 #18): gated by zone-command logic, not calendar.

### Important nuance: `read_mud_date_from_file()`

The file read at `db.c:421` can overwrite `time_info.month` (and `time_info.day`, `time_info.year`) after `mud_time_passed()` sets them. If `etc/date_record` exists with valid data, the `dice(1,50)` vs `dice(1,80)` branch is gated by a **file-persisted month**, not the wall-clock-derived month. However, `time_info.hours` is **never** overwritten by this file, so the sunlight cascade (`db.c:423–432`) always uses the wall-clock-derived hour.

### System-wide consequence

Although only these two `dice()` calls have **execution count** gated by wall-clock, every PRNG draw during boot has its **outcome** gated by wall-clock because the PRNG is seeded from `time(0)` at `comm.c:263`. Changing the boot wall-clock by even 1 second produces a completely different PRNG stream, and hence different values for every `number()`/`dice()`/`uniform()` call.

---

## Appendix: Call Chain Summary

```
time(0) @ comm.c:263
  → prng_seed()                   seeds CMWC PRNG (1024 words via xorshift)

time(0) @ db.c:420
  → mud_time_passed(t2=time(0), t1=650336715)
    → time_info.hours, .day, .month, .year
      → reset_time() sunlight cascade    (db.c:423–432)
      → reset_time() moon cascade        (db.c:438–445)
      → reset_time() pressure branch     (db.c:448) → dice(1,50) OR dice(1,80)
        → weather_info.sky cascade       (db.c:455–462)

number() / dice() / uniform()
  → uniform()          [utils.c:80]
    → prng_uniform()   [random.c:32]
      → prng_next()    [random.c:26]
        → cmwc_next()  [random.c:57]  — advances the CMWC state
```

---

*End of enumeration. A human (Mimo/Claude) should perform root-cause analysis on the gated site (db.c:449/451) and decide whether the file-persisted month override (db.c:421) matters for the fidelity oracle.*
