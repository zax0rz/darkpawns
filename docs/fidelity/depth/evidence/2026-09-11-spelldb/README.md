# Phase 6.3 spellDB keying evidence — 2026-09-11

This evidence belongs to the bounded spellDB slice on branch
`glm/modernize-spelldb-keys`. It is not a Phase 6.3 completion claim.

## Tested source state

- Origin base: `e961be906` (`origin/main`, PR #1444 merge tip).
- Code checkpoint: `e59809f5dbda1a0d3c078a6b28eb515fdce1c793`.
- Pre-keying `pkg/session/cast_cmds.go` SHA-256:
  `278136f01b8bee67e1d09d0246f034b486d401fdc8eb7fe605aa7d80cf9a8919`.
- Post-keying `pkg/session/cast_cmds.go` SHA-256:
  `17725ac9d90d62b145338883029c65343a60bcb4a3991c0f7fc13bed5a956b2d`.
- Independent fixture `pkg/session/spell_db_test.go` SHA-256:
  `49bf13c1455c2e932762838e9d6dabcb72f2a8dea2350cc617acf6d58d7b8738`.
- C reference SHA-256: `src/spell_parser.c`
  `dccbda4a1a3ee3edd4b38c449afabcb10d9b9de184cc096f89efc64ca086123b`;
  `src/spells.h`
  `ee59218b974865fbc4d1a9275915a174ecdec5922a3177662c18399267347510`.

## Complete table census

- `maxCastSpell`: 130, unchanged from C `MAX_SPELLS`.
- Range audited: every index `0..130` inclusive, 131 positions.
- Present records: 103, at indices `1..102` and `105`.
- Absent/default positions: 28 — `0`, `103`, `104`, and `106..130`.
- Every present record has six checked fields: `SpellNum`, `Name`,
  `ManaMax`, `ManaMin`, `ManaChange`, and `MinLevel[12]`.
- Every `MinLevel[12]` in the pre-keying table is the zero value. Absent map
  lookups remain nil; no sentinel records were inserted.

`TestSpellDBPreservesCompletePreKeyingTable` stores the complete 103-record
pre-keying table in an independent type and compares it against the production
map at every one of the 131 indices. It also checks every production key for
range. Thus it detects a shifted entry, wrong named key, omitted record, field
change, changed zero default, or altered sentinel/presence state without
deriving expected values from the post-edit initializer.

## Scope and readers

The production change is limited to `pkg/session/cast_cmds.go`: named existing
spell constants replace raw numeric keys, and named fields replace unkeyed
`spellData` literals. The map remains a map; there is no iteration/order
coupling to preserve. `SpellNum` remains explicit because `manaCost` uses it
to resolve `game.ClassSkillMinLevel`; removing that duplicate identity would be
a broader representation change.

Audited readers are:

- `pkg/session/cast_cmds.go:141-150` (`manaCost`) — uses `SpellNum`, the
  canonical class-level lookup, and the existing `MinLevel` fallback.
- `pkg/session/cast_cmds.go:345-355` (`cmdCastCommand`) — keyed lookup and
  nil behavior remain unchanged.
- `pkg/session/spell_level.go:20-43` (`grantClassSpells`) — keyed lookup and
  legacy lower-case `Name` population remain unchanged; calls occur at
  creation, login, and level-up/menu paths.
- `pkg/session/spell_db_test.go:126-168` — exhaustive Go-before/Go-after
  preservation proof.

## C comparison and boundaries

The C spell identifiers are declared in `src/spells.h:65-175`, including
player IDs 1–105 and `MAX_SPELLS 130`. C initializes all slots 1 through the
limit to unused defaults before applying `spello()` records in
`src/spell_parser.c:1225-1232`. `spello()` field meanings/default min-level
behavior are at `src/spell_parser.c:1148-1162`; unused-slot defaults are at
`:1165-1178`; C mana reads the spell-numbered record at `:257-266`. The C
catalog’s reserved/unused spell-name slots and terminator are visible at
`src/spell_parser.c:51-209`.

The preservation proof is deliberately distinct from C fidelity. Existing
differences were not corrected in this refactor:

- Go record 54 retains `Name: "calliope"` and mana `(100,50,10)`, although C
  declares 54 as `SPELL_LYCANTHROPY` and initializes it to `(1,1,1)` at
  `src/spell_parser.c:1432-1436`; C `SPELL_CALLIOPE` is 94 with `(100,50,10)`
  at `:1534-1535`. The discrepancy is called out inline in the production
  table and remains a separate fidelity follow-up.
- `spellData.Name` is the legacy `SpellMap` label, not a claim that every
  label equals C’s catalog spelling. Existing examples include `armor` vs
  `holy ward`, `teleport` vs `shift reality`, and `flame arrow` at Go ID 32
  versus C’s catalog name for the same ID. These values are preserved.

The independent existing spell-info golden test remains the C/Go metadata
boundary for `spello()` data. This slice does not change formulas, dispatch,
THAC0, saving throws, `LVL_IMMORT`, the save format, or any oracle source.

## Focused proof and gates

Environment for all runs:

```text
PATH=/usr/local/go/bin:$PATH
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
```

Focused unit proof passed:

```text
go test ./pkg/session -run 'TestSpellDBPreservesCompletePreKeyingTable|TestManaCostUsesClassMinimumLevel|TestCast' -count=1
```

Named oracle scenarios passed with no normalized divergence:

- `cast-depth` at seeds `1,2,3,5,8`: 5 scenario-seed runs, 0 failures.
- `cast-gating` at seed `1`: 1 run, 0 failures.
- `cast-mob-innate` at seeds `1,2,3,5,8`: 5 scenario-seed runs, 0 failures.

The seed-1 runs used `--show-oracle`; the normalized C blocks confirmed the
intended cast parser/target/gating path and the mob-innate no-effect path.

Repository gates before the checkpoint passed: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, and
`golangci-lint run ./...` (`0 issues`). `make fidelity-depth` passed with the
actual census `4811 total; 4692 proven/delegated; 68 blocked; 51 excluded`.
`make expected-divergences-check` passed with the repository ledger reporting
26 expected-divergence rows across 10 scenarios.

## Full final oracle census

`make oracle-regression` was run from the clean `bba25758b198990239f968cb08cbaf864f23261e`
state with the required C oracle and default four workers. The run started on
2026-09-11 and finished on 2026-09-12:

```text
oracle-regression: scenarios=940 passed=930 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7048.494s started=2026-09-11T23:04:33-0400 finished=2026-09-12T01:02:01-0400
```

The command exited 2 solely because of the specifically identified,
previously human-cleared `accuse-noarg-depth` baseline. The nine
ledger-pinned expected-divergence scenarios were `accuse-depth`, `force-mob`,
`medit-entry-depth`, `medit-session-depth`, `redit-entry-depth`,
`redit-session-depth`, `sedit-entry-depth`, `sedit-session-depth`, and
`shoot-target-depth`. No result was stale, failed, infrastructural, or timed
out. These are actual run counts, not historical corpus expectations.
