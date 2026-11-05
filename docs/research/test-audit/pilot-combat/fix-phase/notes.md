# Fix phase: the 13 `pkg/combat` tests that could not fail

2026-11-05, by Cline. Companion to [`results.tsv`](results.tsv). Every row was
measured, not asserted: baseline mutation run on the old test, then the same
mutation against the new test.

## Result

**13 of 13 tests that could not fail now can.** 10 strengthened, 3 folded into
the keeper the audit named, 0 `baseline-red`, 0 `possible-bug`, 0 `not-fixed`.

| outcome | rows |
|---|---:|
| `fixed` | 13 |
| `baseline-red` | 0 |
| `possible-bug` | 0 |
| `not-fixed` | 0 |

## Method

Two worktrees off `origin/main` (`bb0fadaa0`): `~/wt/deepseek-test-fix`
(branch `cline/combat-test-fix`) and `~/wt/deepseek-test-fix-mut` (detached,
the scratch tree). For every row, in the scratch tree:

1. applied the mutation to production code from the audit ledgers,
2. ran the **original** test — all 13 green, so the audit's F marks were right
   and no row is `baseline-red`,
3. restored the scratch tree (`git checkout -- .`, `git status --porcelain`
   empty after every row).

Then in the work tree the test was strengthened/folded and run unmutated (all
green), and finally the five changed test files were copied into the scratch
tree with the mutation still applied and the test run again — all 13 red. The
scratch tree was restored byte for byte afterwards and then removed.

Mutations used (one per row, all from the model ledgers):

| row | mutation | file |
|---:|---|---|
| 1 | delete the `GET_STR(ch) == 0` override | `formulas.go` |
| 2 | `if attackType == AttackNormal` -> `if true` | `formulas.go` |
| 3 | weapon arm `else if weaponDamage.Num > 0 && ...` -> `else if false` | `formulas.go` |
| 4 | `if diceroll == 20` -> `if diceroll == -1` | `formulas.go` |
| 5 | delete `victimAC = MAX(-10, victimAC)` | `formulas.go` |
| 6 | `awake && (diceroll == 1 \|\| ...)` -> `diceroll == 1 \|\| (awake && ...)` | `formulas.go` |
| 7 | `cbSetAlignment(killer, newAlign)` -> `_ = newAlign` | `fight_core.go` |
| 8 | `if killer.IsNPC()` -> `if true` | `fight_core.go` |
| 9 | `if killer.IsNPC()` -> `if false` | `fight_core.go` |
| 10 | `Number(0, 10000)` -> `Number(0, 100)` | `formulas.go` |
| 11 | `level > 30 \|\| ...` -> `level > 99 \|\| ...` | `formulas.go` |
| 12 | warrior gate `level > 10` -> `level > 0` | `formulas.go` |
| 13 | `CalculateDamage` `return dam` -> `return 0` | `formulas.go` |

Row 6 needed a narrower mutation than my first attempt. `awake := true` turns
*every* roll into a miss for a sleeping defender, which breaks behaviour outside
the test's contract (and turned the old test red, which the audit recorded as
green). Sol's recorded mutation — let a natural 1 miss regardless of AWAKE — is
the one used, and it reproduces the recorded green baseline exactly.

## `possible-bug` rows

None. Every C-derived expected value I wrote agrees with the port:

- `change_alignment` (rows 7, 8, 9): 62, 412 and the NPC early return all match
  `src/fight.c:488-501` exactly.
- `get_minusdam` (rows 1, 2, 3, 13): the .11 and .06 `pcmod` brackets and the
  AC 100 passthrough match `src/fight.c:1727-1738`; the 11 -> 9 reduction in
  row 13 is C's `int(11 - 11*0.12)`.
- `hit()` (rows 4, 5, 6): `MAX(-10, victim_ac)` and the `diceroll < 20 &&
  AWAKE` gate match `src/fight.c:1822-1826`.
- `perform_violence` (rows 11, 12): the class gate, the `level > 30 ||
  !number(0,500)` guarantee and the parry probe's `number(0,10000)` match
  `src/fight.c:1927-1949`.

The `counter_procs` divergence DeepSeek and Sol both flagged is real but is not
one of these 13 rows; it belongs to the `counter_procs` brief.

## What I had to judge

- **Row 10 cannot be caught by a value.** `ScriptedRoller.Number` returns its
  scripted value and ignores `(from, to)`, so no scripted roll can distinguish a
  0-10000 draw from a 0-100 one. The contract in `src/fight.c:1949` *is* the
  bounds, so the strengthened test wraps the scripted roller in the package's
  existing `recordingRoller` (engine_standup_test.go) and asserts the recorded
  draw was `number(0,10000)`. This is the one row where the assertion is about
  the draw rather than its result.
- **Row 3 needs the draw shape, not just the value.** Dropping the weapon
  contribution falls through to the bare-hands arm, which draws
  `number(0, level/3)` — and a scripted roller answers that probe with the same
  value as the weapon's `dice(1,8)`, so the total is unchanged. The test asserts
  the exact 15 *and* that a `Dice(1,8)` draw happened and no bare-hands probe
  did.
- **Row 13's fixture carried an unrelated +50.** `defaultCombatCallbacks`
  answers race 0 for the victim and 0 for every race-hate slot, and the C loop
  at `src/fight.c:1466-1468` adds `GET_LEVEL(ch)` per match, so the old round
  dealt 5 x 10 bonus damage. The strengthened test sets `GetRace` to a real race
  so the swing carries no race-hate bonus and the asserted 9 points are `hit()`
  arithmetic alone. The production wiring (`pkg/game/combat_wire.go:42-67`)
  returns real race ids and `-1` for absent entries, so this is a fixture
  artifact, not a port bug.
- **Row 8's name stays wrong.** The test is called `..._NPCKiller` but its
  killer is a PC (`npc: false`). Renaming it would break the audit's row
  identity, so the name is kept and the comment records the fact.
- **Folds are moves, then deletes.** Rows 4, 11 and 12 add a case to the named
  keeper and delete the old test. The old bodies are gone from
  `formulas_test.go`; a comment there points at the keeper so the deletion is
  not silent.
- **No callback cleanup was added.** `TestChangeAlignment_NPCKiller` and
  `TestChangeAlignment_NPCKillerNPC` leak their `SetCallbacks` fixture, as they
  did before. Adding `defer SetCallbacks(orig)` would change the global state
  later tests in the package observe, which is outside this brief's scope.
- **No `C` (merge) or `D` (delete) row was touched**, per the brief.

## Gates (each run on its own; all exit 0)

`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`
(0 issues), `make fmt` (no reformat), `make deadcode-ratchet`,
`make fidelity-depth` (regenerated artifacts byte-identical, `git status` clean),
`make fidelity-units`.

No census was run: the diff touches no production file, so no scenario can
observe it. The mutation runs above are the proof (R5h).
