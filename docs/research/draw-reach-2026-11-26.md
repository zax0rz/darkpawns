# C RNG Draw-Site Inventory and Draw Reach (2026-11-26)

Phase 3 step 1 of `docs/plans/ROADMAP-2026-11-10-port-completion.md`. Measured on
2026-11-26 against the unchanged C reference binary
(`/home/zach/darkpawns-c-oracle/bin/circle`, built `-g -O0 -fcommon` from
`dp-oracle-seam`).

## Runs and provenance

| Artifact | Path / Value |
|---|---|
| Reference census run | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-26/nightly-20261126` |
| Reference `go-head.txt` | `0b54a21431618d806c070b83192a023dfed1be7e` |
| Attributed census run | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-26/neon-draw-attribution` |
| Attributed `go-head.txt` | `aeba1ac7cc8f0b17f7090754a9aa146df0c17025` (dirty worktree adding tooling only; no game code changes) |
| Combined census verdict | `CLEAN` (`full=CLEAN`, `claims=CLEAN`, `pairs=7137`, `deduplicated=2318`, `draw_divergent=0`, `elapsed=6074.228s`) |
| Generated inventory & reach | `docs/fidelity/draws/draw-inventory.tsv`, `docs/fidelity/draws/draw-coverage.tsv`, `docs/fidelity/draws/draw-coverage-summary.txt` |

## Headline numbers (beside output reach)

| Metric | Seed 1 | All seeds (22 seeds) | Source |
|---|---:|---:|---|
| Output reach — C segments | 4,332 / 4,902 (88.4%) | — | PF-188 (`nightly-20261126` dump) |
| Output reach — C sites | 3,626 / 5,454 (66.5%) | — | PF-188 (`nightly-20261126` dump) |
| **Draw reach — C top-level draw sites** | **285 / 361 (78.9%)** | **288 / 361 (79.8%)** | **this report (`neon-draw-attribution`)** |
| Verified pairs included | 2,521 | 7,120 (of 7,137; 17 excluded) | `draw-coverage-summary.txt` |
| Verified top-level draws | 13,324,285 | 33,653,904 | `draw-coverage-summary.txt` |

The 3 sites reached only outside seed 1 (+0.8 pp multi-seed gain) are:

- `src/class.c:482` (`roll_real_abils`, `number`, 3 draws in `entry-stats-reroll-values`)
- `src/fight.c:606` (`die_with_killer`, `number`, 3 draws in `circle-skill-message-death-depth`, `trip-skill-message-death-depth`)
- `src/new_cmds.c:437` (`do_headbutt`, `number`, 2 draws in `spec-proc-combined-guard-arms`)

## Method

### 1. Binary inventory (`internal/drawcoverage.ExtractInventory`)

`cmd/dp-draw-coverage` inspects the compiled reference binary
`/home/zach/darkpawns-c-oracle/bin/circle` directly via Go's `debug/elf` and
`debug/dwarf` packages:

- Locates the ELF symbols `number`, `dice`, `uniform`, and `prng_uniform` and
  all `STT_FUNC` symbol ranges.
- Scans `.text` for every x86-64 direct near `call` (`0xE8 <rel32>`) targeting
  one of those four symbols, verifying via `objdump -d` that no indirect or
  tail-call references exist outside internal jumps.
- Maps each call instruction address to its enclosing function and DWARF
  `file:line`, normalizing paths to `src/<file>`.
- Separates the **3 internal primitive-to-primitive calls** in `src/utils.c`
  (`src/utils.c:90` `number -> uniform`, `src/utils.c:109` `dice -> number`,
  `src/utils.c:121` `uniform -> prng_uniform`) from the **361 top-level call
  sites** (`294` `number`, `66` `dice`, `1` `uniform` at `src/db.c:2081` in
  `percent_load`, `0` direct `prng_uniform`).
- Where multiple call instructions share a single source line (`src/weather.c:137`
  has three `dice()` calls in `weather_change`), each call instruction gets a
  distinct `#1`/`#2`/`#3` suffix in ascending instruction-address order so
  `ptrace` return-address attribution maps 1:1 to every call instruction.

Note on line numbers: DWARF line numbers come from the `dp-oracle-seam`
checkout (`/home/zach/darkpawns-c-oracle/src/`). Because the oracle seam adds
instrumentation to `act.wizard.c`, `comm.c`, `db.c`, and `utils.c`, 28 sites in
those four files have small line-number offsets relative to the repository's
`src/` copy (e.g. `src/db.c:2081` in the oracle checkout).

### 2. Inventory vs `grep` cross-check

Cross-checking the 364 binary call instructions (362 unique `(file, line,
primitive)` tuples) against `grep -E '\b(number|dice|uniform|prng_uniform)\s*\('`
over `/home/zach/darkpawns-c-oracle/src/*.{c,h}` (380 unique `(file, line,
primitive)` tuples) yields **zero unexplained differences**:

- **Multiple calls on one line (2 extra instructions vs unique lines):**
  `src/weather.c:137` (`weather_change`) contains three `dice()` calls on one
  line (`dice(1, 4) * diff + dice(2, 6) - dice(2, 6)`), giving 364 instructions
  across 362 unique `(file, line, primitive)` tuples.
- **Only in binary (1):** `src/graph.c:355` (`hunt_victim`, `number`). Paired
  with grep-only `src/graph.c:356`: a two-line `sprintf(...)` call starting on
  line 355 whose `number(1000, 2000)` argument is on line 356; GCC `-g -O0`
  attributes the `call number` instruction to the statement's starting line 355.
- **Only in grep (19):**
  - Multi-line statement counterpart (1): `src/graph.c:356` (`number`).
  - String literal (1): `src/comm.c:1111` (`"number (%s%d%s/%s%d%s) ]%s",`).
  - Function definitions (4): `src/random.c:32` (`prng_uniform`),
    `src/utils.c:79` (`number`), `src/utils.c:100` (`dice`), `src/utils.c:120`
    (`uniform`).
  - Function prototypes / declarations (5): `src/magic.c:63` (`dice`),
    `src/random.h:26` (`prng_uniform`), `src/utils.h:52` (`number`),
    `src/utils.h:53` (`dice`), `src/utils.h:54` (`uniform`).
  - Comments (8): `src/structs.h:818` (`number`), `src/structs.h:825`
    (`number`), `src/utils.c:54` (`number`), `src/utils.c:55` (`dice` and
    `uniform`), `src/utils.c:57` (`dice` and `number`), `src/utils.c:82`
    (`number`).

### 3. Attribution under `ptrace` (`cmd/dp-draw-tracer`)

Nothing in `src/` or `/home/zach/darkpawns-c-oracle` was modified, and no C
binary was rebuilt. `cmd/dp-draw-tracer` runs the **unchanged reference binary**
under Linux `ptrace`:

- To preserve direct PID ownership for `dp-oracle-diff`'s listener-ownership
  check and `<CRASH>`/`stop()` signal handling without modifying
  `scripts/census.sh` or `cmd/dp-oracle-diff`, the launcher spawns a detached
  `--attach <pid>` helper, grants `PR_SET_PTRACER`, waits for `PTRACE_ATTACH` +
  `PTRACE_O_TRACEEXEC`, and then `execve`s `/home/zach/darkpawns-c-oracle/bin/circle`
  in place.
- On `PTRACE_EVENT_EXEC`, the helper reads `/proc/<pid>/maps` for the PIE base
  and installs `INT3` breakpoints on `number`, `dice`, and every external
  `call uniform` site outside `number` (`0x26945`, `src/db.c:2081`).
- On each trap, it records `<ret-vaddr>\t<primitive>` to
  `<DP_ORACLE_EVIDENCE_ROOT>/<DP_ORACLE_EVIDENCE>/c-draw-sites.log` only for
  logged top-level calls (`dice` with `num > 0 && size > 0` per `utils.c:104`;
  `number` whose return address is outside `dice`; external `call uniform`).
- The combined run executed through `scripts/census.sh start --combined
  --draw-parity --name neon-draw-attribution` with
  `CENSUS_ALLOW_NONREFERENCE=1` inside the standard 36-worker admission budget
  (6,074s vs 5,772s for the untraced `nightly-20261126` run).

## Parity check and exclusions

`cmd/dp-draw-coverage` checks every `(seed, scenario)` capture in
`neon-draw-attribution` against `nightly-20261126`:

1. Byte-for-byte equality of `c-draws.log` against the reference run: **7,137 of
   7,137 pairs matched (0 `draw_stream_mismatch`)**.
2. Exact 1:1 line count and per-line primitive alignment between `c-draws.log`
   and `c-draw-sites.log`, plus resolution of every return address in the
   inventory: **7,120 pairs passed; 17 pairs (5 scenarios) were excluded** for
   `line_count_mismatch` and not counted.

In all 17 excluded pairs, `c-draws.log` is byte-for-byte identical to the
reference run and the full prefix of `c-draw-sites.log` matches line-for-line,
with 1 or 2 trailing entries (`src/fight.c:1035` `skill_message` `dice`, or
`src/fight.c:603` `die_with_killer` `number`) recorded as `circle` shut down.
None of the 5 excluded scenarios hit any of the 73 unreached sites.

Excluded pairs (17):

| Seed | Scenario | Reason |
|---:|---|---|
| 1 | `dp1443-save-re-equip-bad-stats-depth` | `line_count_mismatch` |
| 1 | `l1-act-movement-1-depth` | `line_count_mismatch` |
| 1, 2, 3, 5, 8 | `shoot-mob-miss-depth` | `line_count_mismatch` |
| 1, 2, 3, 5, 8 | `shoot-pc-miss-depth` | `line_count_mismatch` |
| 1, 2, 3, 5, 8 | `suffering-message-outcomes-depth` | `line_count_mismatch` |

## Least-reached C files

| File | Seed 1 | All seeds | Draws (all seeds) |
|---|---:|---:|---:|
| `src/objsave.c` | 0/2 (0.0%) | 0/2 (0.0%) | 0 |
| `src/spec_procs.c` | 14/35 (40.0%) | 14/35 (40.0%) | 6,264 |
| `src/mobact.c` | 4/8 (50.0%) | 4/8 (50.0%) | 1,809,860 |
| `src/act.wizard.c` | 1/2 (50.0%) | 1/2 (50.0%) | 1,610 |
| `src/limits.c` | 1/2 (50.0%) | 1/2 (50.0%) | 18 |
| `src/class.c` | 24/37 (64.9%) | 25/37 (67.6%) | 413,352 |
| `src/spec_procs2.c` | 12/17 (70.6%) | 12/17 (70.6%) | 76,597 |
| `src/act.informative.c` | 5/7 (71.4%) | 5/7 (71.4%) | 371 |
| `src/new_cmds2.c` | 13/18 (72.2%) | 13/18 (72.2%) | 266 |
| `src/dream.c` | 10/13 (76.9%) | 10/13 (76.9%) | 602 |
| `src/spells.c` | 9/11 (81.8%) | 9/11 (81.8%) | 198 |
| `src/spec_procs3.c` | 14/16 (87.5%) | 14/16 (87.5%) | 4,838 |
| `src/magic.c` | 38/43 (88.4%) | 38/43 (88.4%) | 5,386 |
| `src/graph.c` | 8/9 (88.9%) | 8/9 (88.9%) | 82,329 |
| `src/new_cmds.c` | 26/30 (86.7%) | 27/30 (90.0%) | 1,401 |
| `src/mobprog.c` | 10/11 (90.9%) | 10/11 (90.9%) | 909 |
| `src/act.offensive.c` | 20/21 (95.2%) | 20/21 (95.2%) | 1,492 |
| `src/db.c` | 21/22 (95.5%) | 21/22 (95.5%) | 31,049,905 |
| `src/fight.c` | 21/23 (91.3%) | 22/23 (95.7%) | 35,767 |

Twelve files are at 100.0% draw-site reach (`src/act.other.c` 12/12,
`src/weather.c` 9/9, `src/act.movement.c` 3/3, `src/act.item.c` 2/2,
`src/act.social.c` 1/1, `src/comm.c` 1/1, `src/gate.c` 1/1, `src/handler.c` 1/1,
`src/interpreter.c` 1/1, `src/scripts.c` 1/1, `src/shop.c` 1/1,
`src/spell_parser.c` 1/1).

## Unreached draw sites (73, grouped by file)

Listed without classification (classification is the next step):

### `src/act.informative.c` (2)
- `src/act.informative.c:2360` — `do_consider` (`dice`)
- `src/act.informative.c:2371` — `do_consider` (`dice`)

### `src/act.offensive.c` (1)
- `src/act.offensive.c:1437` — `ambush_event` (`dice`)

### `src/act.wizard.c` (1)
- `src/act.wizard.c:3491` — `random_room_in_zone` (`number`)

### `src/class.c` (12)
- `src/class.c:487` — `roll_real_abils` (`number`)
- `src/class.c:636` — `advance_level` (`number`)
- `src/class.c:637` — `advance_level` (`number`)
- `src/class.c:639` — `advance_level` (`number`)
- `src/class.c:644` — `advance_level` (`number`)
- `src/class.c:645` — `advance_level` (`number`)
- `src/class.c:647` — `advance_level` (`number`)
- `src/class.c:665` — `advance_level` (`number`)
- `src/class.c:666` — `advance_level` (`number`)
- `src/class.c:691` — `advance_level` (`number`)
- `src/class.c:692` — `advance_level` (`number`)
- `src/class.c:694` — `advance_level` (`number`)

### `src/db.c` (1)
- `src/db.c:1768` — `read_mobile` (`number`)

### `src/dream.c` (3)
- `src/dream.c:75` — `dream` (`number`)
- `src/dream.c:82` — `dream` (`number`)
- `src/dream.c:90` — `dream` (`number`)

### `src/fight.c` (1)
- `src/fight.c:1932` — `perform_violence` (`number`)

### `src/graph.c` (1)
- `src/graph.c:217` — `do_track` (`number`)

### `src/limits.c` (1)
- `src/limits.c:517` — `point_update` (`number`)

### `src/magic.c` (5)
- `src/magic.c:648` — `mag_damage` (`dice`)
- `src/magic.c:670` — `mag_damage` (`dice`)
- `src/magic.c:686` — `mag_damage` (`dice`)
- `src/magic.c:702` — `mag_damage` (`dice`)
- `src/magic.c:713` — `mag_damage` (`dice`)

### `src/mobact.c` (4)
- `src/mobact.c:210` — `mobile_activity` (`number`)
- `src/mobact.c:212` — `mobile_activity` (`number`)
- `src/mobact.c:214` — `mobile_activity` (`number`)
- `src/mobact.c:249` — `mobile_activity` (`number`)

### `src/mobprog.c` (1)
- `src/mobprog.c:400` — `get_bad_guy` (`number`)

### `src/new_cmds.c` (3)
- `src/new_cmds.c:530` — `do_bearhug` (`number`)
- `src/new_cmds.c:2369` — `do_parry` (`number`)
- `src/new_cmds.c:2639` — `do_groinrip` (`number`)

### `src/new_cmds2.c` (5)
- `src/new_cmds2.c:299` — `do_mindlink` (`number`)
- `src/new_cmds2.c:379` — `beholder` (`number`)
- `src/new_cmds2.c:391` — `beholder` (`number`)
- `src/new_cmds2.c:393` — `beholder` (`number`)
- `src/new_cmds2.c:641` — `black_horn` (`number`)

### `src/objsave.c` (2)
- `src/objsave.c:1135` — `gen_receptionist` (`number`)
- `src/objsave.c:1136` — `gen_receptionist` (`number`)

### `src/spec_procs.c` (21)
- `src/spec_procs.c:458` — `magic_user` (`number`)
- `src/spec_procs.c:549` — `paladin` (`number`)
- `src/spec_procs.c:959` — `dragon_breath` (`number`)
- `src/spec_procs.c:1081` — `mini_thief` (`number`)
- `src/spec_procs.c:1094` — `mini_thief` (`number`)
- `src/spec_procs.c:1098` — `mini_thief` (`number`)
- `src/spec_procs.c:1107` — `mini_thief` (`number`)
- `src/spec_procs.c:1156` — `black_undead_knight` (`number`)
- `src/spec_procs.c:1195` — `black_undead_knight` (`number`)
- `src/spec_procs.c:1217` — `red_undead_knight` (`number`)
- `src/spec_procs.c:1254` — `red_undead_knight` (`number`)
- `src/spec_procs.c:1282` — `mickey` (`number`)
- `src/spec_procs.c:1387` — `mallory` (`number`)
- `src/spec_procs.c:1472` — `cleric` (`number`)
- `src/spec_procs.c:1553` — `cleric` (`number`)
- `src/spec_procs.c:1559` — `cleric` (`number`)
- `src/spec_procs.c:1565` — `cleric` (`number`)
- `src/spec_procs.c:1571` — `cleric` (`number`)
- `src/spec_procs.c:1622` — `conductor` (`number`)
- `src/spec_procs.c:1640` — `conductor` (`number`)
- `src/spec_procs.c:2193` — `fearface` (`number`)

### `src/spec_procs2.c` (5)
- `src/spec_procs2.c:472` — `tipster` (`number`)
- `src/spec_procs2.c:1396` — `little_boy` (`number`)
- `src/spec_procs2.c:1504` — `eq_sound` (`number`)
- `src/spec_procs2.c:1761` — `bat_room` (`number`)
- `src/spec_procs2.c:1776` — `bat` (`number`)

### `src/spec_procs3.c` (2)
- `src/spec_procs3.c:477` — `field_object` (`dice`)
- `src/spec_procs3.c:719` — `roach` (`number`)

### `src/spells.c` (2)
- `src/spells.c:921` — `spell_mindsight` (`number`)
- `src/spells.c:1001` — `spell_control_weather` (`dice`)
