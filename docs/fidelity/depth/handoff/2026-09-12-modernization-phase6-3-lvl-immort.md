# Phase 6.3 `LVL_IMMORT` single-source handoff — 2026-09-12

## Boundary and verdict

This handoff covers only the bounded Phase 6.3 `LVL_IMMORT` ownership slice.
It does not claim other level constants, level ladders, table redesign,
privilege-policy changes, spell dispatch, save-format changes, or whole-Phase
6.3 completion.

The isolated worktree is `/home/zach/darkpawns-lvl-immort`, branch
`glm/modernize-lvl-immort`, based on `origin/main` at
`dd2e4638a583d5c876f59af3aca732f37f90e7b9`. The primary checkout's
uncommitted `docs/specs/tui-setup-wizard.md` edit was not touched. PR #1447
was verified `MERGED` before fetching; its merge commit is included in the
base. The implementation checkpoint tested below is
`caf62b6f70d15e0174defbe9dfc4750fdc6e40a1`.

Verdict: complete for this bounded slice. The canonical value is still the
C value `31`; all confirmed Go aliases and direct consumers preserve their
existing type, comparison, call path, and player-facing result.

## C authority and dependency rationale

The authoritative declarations are `src/structs.h:620-621`:

```c
#define LVL_IMMORT  31
#define LEVEL_IMMORT    LVL_IMMORT
```

`pkg/combat` is the smallest existing dependency-neutral owner. It imports no
`pkg/game`; `pkg/game` already imports `pkg/combat`, and both `pkg/spells` and
`pkg/scripting` already import `pkg/combat`. `pkg/session` already imports
`pkg/game`. Therefore the change introduces no cycle, runtime variable,
configuration setting, or broad package migration. The canonical declaration
is `pkg/combat/fight_core.go:54`; its type remains an untyped compile-time
constant, so existing arithmetic, comparisons, integer arguments, and array
bounds retain their typing behavior.

## Before/after inventory and dispositions

The pre-edit census found one numeric Go production definition plus the
following aliases and direct consumers. The final named-definition search has
one numeric production definition and no remaining numeric `LVL_IMMORT`,
`LVLImmort`, or `lvlImmort` definition outside independent tests/tooling.

| Surface | Before | After / disposition | Actual consumer or call path |
|---|---|---|---|
| `pkg/combat/fight_core.go:54` | `LVL_IMMORT = 31` | **Canonical definition**, unchanged numeric value | Combat gates and damage/logging paths in `fight_core.go` and `skill_messages.go` |
| `pkg/game/limits.go:23` | `LVL_IMMORT = 31` | **Compatibility alias**: `combat.LVL_IMMORT` | Game-wide `LVL_IMMORT` consumers, including level/XP, movement, visibility, skills, clans, death, and command registration |
| `pkg/game/dreams.go:247` | `LVLImmort = 31` | **Compatibility alias**: `LVL_IMMORT` | Dream level switch and `WhodMinLevel` |
| `pkg/game/spec_procs4.go:17` | `lvlImmort = 31` | **Compatibility alias**: `LVL_IMMORT` | `outofjailguard`, `jailguard`, `pray_for_items`, and newbie-zone special gates; other game files consume this local name |
| `pkg/session/wizard_cmds.go:13` | `LVL_IMMORT = 31` | **Compatibility alias**: `game.LVL_IMMORT`; `LVL_GOD`/other authority literals retained | Session command gates, wizard commands, and `LVL_IMMORT+1` authority checks |
| `pkg/spells/call_magic.go:12` | `lvlImmort = combat.LVL_IMMORT` | **Already satisfied**; no edit | `affect_spells.go` and `callMagic`; spells already use the dependency-neutral owner |
| `pkg/scripting/engine.go:719` | `lua.LNumber(31)` | **Confirmed semantic replacement**: `lua.LNumber(combat.LVL_IMMORT)` | Go setup of the Lua `LVL_IMMORT` global after shared script loading |
| `pkg/session/cmd_info.go:24` | `for i := 1; i < 31; i++` | **Confirmed semantic replacement**: `i < LVL_IMMORT` | `cmdLevels`, matching C `do_levels`' 30-row loop |

The confirmed direct C paths are: `src/act.informative.c:2322-2326` for the
strict `i < LVL_IMMORT` levels loop; `src/spec_procs3.c:1289-1301` for the
strict `GET_LEVEL(ch) > LVL_IMMORT` fly-exit pass-through; and
`src/act.informative.c:1272` plus `src/limits.c:327-363` for the XP/level
cap path. The existing Go consumers retain their original strict/inclusive
operators; only the named value source changed.

## Deliberately retained values

Equivalent integers were classified by call path rather than mechanically
replaced:

| Value / surface | Disposition |
|---|---|
| `src/fight.c:1912-1913` and `pkg/combat/formulas.go:570-574` level `31`/`30` | **Intentionally unrelated**: C's NPC attack scheduler uses explicit thresholds, not `LVL_IMMORT`; the existing 30/31 attack tests remain independent. |
| `cmd/dp-command-gates/main.go:48-64,293-295` and embedded `pkg/session/command_gates.tsv` rows | **Intentionally retained independent C/tooling data**: the generator evaluates C command-table expressions and stores numeric gate evidence; changing it to a production alias would couple the independent oracle/tool model and is outside this refactor. |
| `lib/world/scripts/globals.lua:6` and `tests/test_scripts/globals.lua:6` | **Independent script/test inputs** frozen during runs. The engine's Go setup then exports the canonical value; the source fixtures were not rewritten. |
| `pkg/game/spec_procs4.go:400,404` level assignments and numeric test fixtures | **Independent authored result/expectation literals**, not threshold definitions. |
| Spell IDs, affect bits, array dimensions, vnums, table cells, random bounds, and unrelated test data containing `31` | **Unrelated/deferred** under R5c; no matching policy meaning was established. |

`LEVEL_IMMORT` is a C preprocessor alias, not a separate Go definition. The
remaining `LVL_GOD`, `LVL_LEGEND`, `LVL_HIGOD`, `LVL_GRGOD`, `LVL_IMPL`, and
other authority values remain numeric and package-local because this milestone
is only `LVL_IMMORT`; no privilege-policy redesign is implied.

## Independent numeric and type proof

`pkg/combat/lvl_immort_test.go` uses an independent `const cLVLImmort = 31`
from `src/structs.h:620`; it does not import or compare an alias. It proves
the canonical value and uses `const mortalRows = LVL_IMMORT - 1` plus
`var _ [LVL_IMMORT]int` to exercise compile-time arithmetic and an array bound.
`pkg/session/cmd_info_test.go` independently expects 30 rows and checks that
`[30]` is present while `[31]` is absent. `pkg/scripting/lvl_immort_test.go`
independently expects the Lua value to be a `lua.LNumber(31)`. Existing tests
also remain independent where their expected C literals are behavioral
fixtures; no alias-to-alias equality is used as fidelity proof.

Boundary behavior was reused and focused at the actual consumers:

- `TestSpecFlyExitUp_EntryGatesAndAudience` exercises above (`32`) pass,
  exactly (`31`) fly/pass and non-flying block, and below (`30`) block, while
  preserving the strict C `>` comparison and exact actor/peer bytes.
- `TestGainExp_MortalCannotReachImmortal` exercises level 29 → 30 promotion
  and the level-30 cap at `LVL_IMMORT-1`.
- `TestCmdLevels` exercises the full 30-row player-facing result on the
  structured-data path; the existing pager test continues to own the plain
  22-line page boundary.

## Focused oracle proof

Environment:

```text
PATH=/usr/local/go/bin:$PATH
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
```

The frozen focused matrix was 13 unique scenario/seed executions:

| Scenario | Seeds | Executions | Pass | Failed | Infra | Timed out | Stale | Unpinnable |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| `levels-depth` | `1` | 1 | 1 | 0 | 0 | 0 | 0 | 0 |
| `levels-arguments-depth` | `1` | 1 | 1 | 0 | 0 | 0 | 0 | 0 |
| `levels-npc-depth` | `1` | 1 | 1 | 0 | 0 | 0 | 0 | 0 |
| `spec-proc-fly-exit-up-high` | `1,2,3,5,8` | 5 | 5 | 0 | 0 | 0 | 0 | 0 |
| `spec-proc-fly-exit-up-block` | `1,2,3,5,8` | 5 | 5 | 0 | 0 | 0 | 0 | 0 |
| **Total** | | **13** | **13** | **0** | **0** | **0** | **0** | **0** |

The governing existing manifests are `docs/fidelity/depth/levels.tsv` and
`docs/fidelity/depth/spec-procs.tsv`; no duplicate manifest rows or new
scenario inputs were needed. The existing `levels.output`,
`levels.arguments-ignored`, `levels.npc-branch`, `levels.pager`, and
`room.fly-exit-up-*` rows remain the durable coverage records. Seed 1 was run
with `--show-oracle` for all five vehicles. The blocks inspected were the 30
`levels` rows, the ignored-argument table, the hound-dog NPC response, the
exact fly-exit refusal plus peer line, and the allowed level-40 movement plus
origin-room line. Every focused run reported `result: no normalized divergence`.

Focused summary and show-oracle logs are preserved outside the harness at
`/home/zach/lvl-immort-evidence-2026-09-12/`. The focused summary SHA-256 is
`b2dc0774982c40d4a47706e702bac37a612724bad49fdf5b1e2c653c5d27b47d`.

## Required gates and full census

At checkpoint `caf62b6f7`, all required non-oracle gates passed:

```text
make fmt                         PASS
go build ./...                   PASS
go vet ./...                     PASS
go test ./...                    PASS
go test ./pkg/game/...           PASS
golangci-lint run ./...          PASS
make fidelity-depth              PASS: 4811 total, 4692 proven/delegated,
                                      68 blocked, 51 excluded;
                                      actionable 4692/4760 = 98.6%
make expected-divergences-check  PASS
```

The full command was run with seed `1`, timeout `240s`, four workers, and
frozen inputs:

```text
ORACLE_REGRESSION_GO=/usr/local/go/bin/go \
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
ORACLE_REGRESSION_TIMEOUT=240s ORACLE_REGRESSION_SEED=1 \
ORACLE_REGRESSION_JOBS=4 make oracle-regression
```

The final harness line was:

```text
scenarios=940 passed=930 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The command exited 2 solely because `accuse-noarg-depth` is the specifically
identified, previously human-cleared run-varying baseline. Independent
reconciliation found 940 unique scenario names, no missing names, no
unexpected names, and no new unpinnable/stale/failed/infra/timed-out result.
The raw result stream has 941 lines because the harness prints the one
`accuse-noarg-depth` unpinnable result once during worker output and once in
its aggregate report; that is duplicate aggregate display, not duplicate
execution. The durable full-log SHA-256 is
`842d68ac75bc651fb50f07dbcabd4567ffec20413431d4895f4e49361eac8fd2`.

## Measured delta and next action

Relative to `origin/main`, the implementation checkpoint measures **+20/-15
production lines** (aliases, one loop, one conversion, and comments) and
**+53/-0 test lines** (independent canonical, Lua, and `levels` boundary
proof). Documentation is limited to this dated handoff and the single
LVL_IMMORT-only roadmap tracking entry. No `src/` or oracle checkout file,
save format, generator, runner, pin, website, or deployment file changed.

The next action is human review of the single PR. Do not merge it from this
handoff. Remaining Phase 6.3 candidates must be reconciled separately before
any whole-phase claim.
