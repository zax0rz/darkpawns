# Phase 6.3 THAC0 keying handoff — 2026-09-12

## Boundary

This handoff covers only the THAC0 table-keying slice. It does not claim
saving throws, `LVL_IMMORT`, other literal ladders, spellDB, spell dispatch,
combat-ticker work, or whole-Phase 6.3 completion.

The worktree is `/home/zach/darkpawns-thac0`, branch
`glm/modernize-thac0-keys`, based on `origin/main` commit
`2ea3474002e5f055395ac9d90251e22bb84eabf6`. PR #1445 was merged before the
base was selected. The primary checkout's uncommitted TUI setup-wizard edit
was left untouched.

## C call path and risk removed

C declares `thaco[NUM_CLASSES][LVL_IMPL+1]` in `src/class.c:297-371`, with
class IDs/order in `src/structs.h:115-128`. The combat reader is
`src/fight.c:1763-1786`: players index by class and level, while NPCs use
flat 20. Go has two live copies. `pkg/combat/formulas.go:62-135` feeds
`getTHAC0` at `:294-309`, then `CalculateHitChance`; `pkg/game/player.go:458-471`
feeds the level-1 initialization at `:423-425` in `newCharacter`.

The refactor keys each of the 12 rows with the existing class constants while
retaining both `[12][41]int` containers. This removes the risk that inserting
or reordering a class row silently changes a different class's THAC0 values.
No formula, combat ticker, RNG call, mob handling, save field, dimension,
sentinel, default, or consumer changed.

## Preservation and fidelity evidence

The complete pre-edit Go baseline is the two 12×41 tables at the base commit:
984 cells total, including the level-0 sentinel and every level 1–40 entry.
The independent numeric fixture in `pkg/game/thaco_table_test.go:5-22` and
the existing C fixture in `pkg/combat/thaco_golden_test.go:7-27` are frozen
numeric data, not generated from the keyed initializer. Their semantic payload
checksum is identical for C, Go-before, and Go-after:

```text
cells=492 sha256=38562ecc51aa1043b365442250e3cf8684b26c9d055ee07dc781e12ea1d85b9b
```

`TestTHAC0_TableCellsMatchCSource` covers all combat table cells;
`TestTHAC0TablePreservesEveryCell` covers all game table cells;
`TestTHAC0_GoldenAgainstCSource` covers every valid combat lookup;
`TestTHAC0_Clamps` covers NPCs, levels -1/0/1/40/41/99, and invalid classes;
`TestNewCharacterTHAC0ReaderPreservesLevelOneLookup` covers the actual game
constructor reader for all classes and invalid constructor classes. This keeps
Go-before/Go-after preservation distinct from C fidelity and preserves the
existing Go-only invalid-input fallback boundary under R5c.

## Focused oracle coverage

Environment: `PATH=/usr/local/go/bin:$PATH`,
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`.

The affected reader paths were exercised by four named vehicles:
`combat-swing`, `combat-hit-weapon`, `combat-hit-sleeping`, and `hit-depth`.
The frozen matrix used seeds 1, 2, 3, 5, and 8 for each vehicle: 20 unique
runs, 20 passes, 0 missing, 0 duplicates, 0 failed, 0 infrastructure, 0
timed out, and 0 stale. Seed-1 `--show-oracle` runs established the intended
C blocks before the matrix: miss arm, weapon-hit arm, sleeping-victim arm,
and synchronous command-boundary arm. Multiple seeds are recorded as varied
execution evidence, not as a claim that one green combat outcome proves draw
parity.

## Validation and next state

The focused table/reader checks passed before this handoff was written. The
final checkpoint must also pass every `AGENTS.md` gate:

```text
make fmt
go build ./...
go vet ./...
go test ./...
go test ./pkg/game/...
golangci-lint run ./...
make fidelity-depth
make expected-divergences-check
make oracle-regression
```

For the full census, require `failed=0 infra=0 timed_out=0 stale=0`; the only
allowed unpinnable result is the specifically identified, previously
human-cleared `accuse-noarg-depth` baseline. The final PR description records
the exact tested checkpoint commit, complete census tally, changed-file list,
measured deltas, and review URL. Leave the PR unmerged and stop for review.
