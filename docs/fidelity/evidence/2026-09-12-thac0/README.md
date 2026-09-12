# Phase 6.3 THAC0 keying evidence — 2026-09-12

## Boundary and tested state

This is the bounded THAC0 table-keying slice only. The isolated worktree is
`/home/zach/darkpawns-thac0`, branch `glm/modernize-thac0-keys`, based on
`origin/main` at `2ea3474002e5f055395ac9d90251e22bb84eabf6`. The primary
checkout's uncommitted `docs/specs/tui-setup-wizard.md` edit was not touched.

PR #1445 (`glm/modernize-spelldb-keys`) was verified merged before this work;
its spellDB changes were already in the base commit and were not incorporated
as unmerged work.

## Audit

The positional risk was two unkeyed copies of the same C-shaped table:

| Go table | Actual reader | Representation retained |
|---|---|---|
| `pkg/combat/formulas.go:62` | `pkg/combat/formulas.go:294-309`, called by `CalculateHitChance` | `[12][41]int`, keyed rows |
| `pkg/game/player.go:458` | `pkg/game/player.go:423-425` in `newCharacter` | `[12][41]int`, keyed rows |

C declares the authoritative table at `src/class.c:297-371`; the class IDs and
ordering are `src/structs.h:115-128`, and the live C combat reader is
`src/fight.c:1763-1786`. The table has 12 rows, 41 columns, level 0 sentinel
`100`, and valid levels 1–40. Go's existing reader behavior remains: NPCs use
flat 20; player levels below 1 clamp to 1 and above 40 clamp to 40; invalid
classes fall back to `ClassWarrior`. The constructor reader only initializes
level 1 and leaves invalid-class construction at its existing default 20.
Mob-file THAC0 (`pkg/game/mob.go`) and all derived hit modifiers/formulas are
outside the table-keying change.

## Independent preservation proof

The pre-edit source commit is `2ea3474002e5f055395ac9d90251e22bb84eabf6`.
Before editing, both Go table literals were captured from that commit and
their semantic numeric payloads were compared independently with the C
declaration. The final semantic census is:

```text
c:     cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
combat before: cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
combat after:  cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
game before:   cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
game after:    cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
total Go cells=984; before==after for both copies; after==C for both copies
```

The complete independent game fixture is
`pkg/game/thaco_table_test.go:5-22`; the combat C fixture is
`pkg/combat/thaco_golden_test.go:7-27`. The tests compare all 492 stored
cells, including zero/default boundaries (the level-0 sentinel), without
calling the production lookup or reading the new table for expected values.
They also pin valid level-1/40 endpoints, below/above bounds, NPC behavior,
invalid-class fallback, all 12 named class IDs, and the constructor's level-1
reader. These checks detect swapped rows, shifted/omitted levels, wrong named
keys, changed defaults or dimensions, and fallback changes separately from
the C-fidelity assertion.

Exact semantic census command:

```text
python3 - <<'PY'  # extract the numeric initializer payloads from HEAD, worktree, and src/class.c
... compare 492-cell payloads and SHA-256 values ...
PY
```

The production diff changes only row syntax: existing numeric row literals are
assigned to `ClassMage` through `ClassMystic` (combat) and
`ClassMageUser` through `ClassMystic` (game). No container, dimension, lookup,
formula, initialization, or save format changed.

## Focused oracle proof

Environment:

```text
PATH=/usr/local/go/bin:$PATH
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
```

Command used for each matrix cell:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /tmp/dp-oracle-diff-thac0 --scenario <scenario> --seed <seed>
```

The input matrix was frozen as four named scenarios × seeds `1,2,3,5,8`:

| scenario | runs | passed | failed | missing | duplicates |
|---|---:|---:|---:|---:|---:|
| `combat-swing` | 5 | 5 | 0 | 0 | 0 |
| `combat-hit-weapon` | 5 | 5 | 0 | 0 | 0 |
| `combat-hit-sleeping` | 5 | 5 | 0 | 0 | 0 |
| `hit-depth` | 5 | 5 | 0 | 0 | 0 |
| **total** | **20** | **20** | **0** | **0** | **0** |

Seed 1 was rerun with `--show-oracle` for every vehicle. The intended C
blocks were reached and matched: `[hit trainee]` miss in `combat-swing`,
`[hit orc]` weapon-hit in `combat-hit-weapon`, `[hit orc]` sleeping-victim
path in `combat-hit-sleeping`, and `[hit the trainee]` synchronous command path
in `hit-depth`. The varied seeds exercise the affected reader under different
RNG streams; the result is not presented as a standalone draw-parity proof.

## Required gates

Focused unit baseline and post-edit checks passed:

```text
go test ./pkg/combat ./pkg/game
```

The complete repository gates, `make fidelity-depth`,
`make expected-divergences-check`, and the full `make oracle-regression` census
are recorded in the dated handoff after the final checkpoint. The only
permitted unpinnable row, if present in the full census, is the established
human-cleared `accuse-noarg-depth` baseline; `failed`, `infra`, `timed_out`,
and `stale` must all be zero.
