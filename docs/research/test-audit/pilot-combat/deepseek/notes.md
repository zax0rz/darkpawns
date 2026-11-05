# Test audit notes — `pkg/combat` (agent: deepseek)

Pilot audit per `BRIEF-test-audit-pilot-combat.md`. Read-only; no repository changes.
Pinned SHA: `b03629cc8c4fd4904a373fa0dcf2ae346095f79a` (2026-11-04).
Audit worktree `~/wt/deepseek-test-audit` (detached); mutation worktree `~/wt/deepseek-test-audit-mut`.
Inventory: 208 functions (`inventory.txt`), 38 `_test.go` files / 7,152 lines against 10 production
files / 4,606 lines. Deliverables: `ledger.tsv` (208 rows), `mutations.tsv` (15 rows), this file.

## Guesses before I started, against the result

Written at the top of this file before reading any test (kept verbatim in git-less form below).

| Mark | Guess | Actual | Delta |
|---|---|---|---|
| R | 132 (63%) | **174 (84%)** | +42 |
| F | 20 (10%) | **15 (7%)** | −5 |
| C | 34 (16%) | **16 (8%)** | −18 |
| D | 22 (11%) | **3 (1%)** | −19 |

I expected much more duplication and deletion than the package actually holds. Claude's
pre-registered guess (75% R) was closer than mine: this package is genuinely
fidelity-dense, and the C citations in the test comments mostly check out against `src/`.

## Baseline

`go test ./pkg/combat/...` at `b03629cc8`: **PASS** (`ok github.com/zax0rz/darkpawns/pkg/combat 0.063s`).
No baseline failures, so there were no "failing tests are really bug reports" candidates here.

## Counts

By mark: R 174, F 15, C 16, D 3.

By junk code: J4 16, J2 12, J7 3, J10 2, J1 1, J3 1.

By `author_agent`:

| agent | R | F | C | D | total |
|---|---|---|---|---|---|
| unknown | 99 | 14 | 14 | 1 | 128 |
| codex | 29 | 0 | 0 | 0 | 29 |
| glm | 24 | 0 | 1 | 0 | 25 |
| claude | 11 | 1 | 1 | 2 | 15 |
| sol | 6 | 0 | 0 | 0 | 6 |
| deepseek | 5 | 0 | 0 | 0 | 5 |

The 128 `unknown` rows are the pre-agent era: `BRENDA69` (76), `zach`
(40), `Zach Greene` (12). Their PRs carry no agent branch prefix and no `Co-Authored-By`
trailer, so per the brief they stay `unknown` with the git author recorded in
`author_evidence`. Attribution is otherwise strong: 62 distinct introducing commits across
50 PRs were resolved.

**Before vs after the oracle (2026-07-12).** 114 rows predate the oracle; 94 come after.

| | R | F | C | D |
|---|---|---|---|---|
| born before oracle | 81 | 15 | 15 | 3 |
| born after oracle | 93 | 0 | 1 | 0 |

Every F, C and D row in the package was born before the oracle. That is the headline result:
once `cmd/dp-oracle-diff` existed, the tests that landed in `pkg/combat` are all keepers
(the single post-oracle C is a merge of a duplicate pair, not junk).

`expected_source`: c-cited 162, hand-written 42, go-generated 1, none 3.
`confidence`: high 139, medium 46, low 23.


## The brief's lead: where did `ac_reduction_golden_test.go`'s values come from?

`TestACReduction_GoldenAgainstCSource` (`pkg/combat/ac_reduction_golden_test.go:9`) was added
2026-07-05 in `51f3fbefe` (PR **#84**, branch `fix/stream1-kill-pipeline`, subject
`feat: combat & skill golden tests — C→Go fidelity for backstab, AC reduction, damage msgs,
position dmg, attack rounds, attributes, practice params, skill formulas (round 8b)`). It
predates the oracle by a week.

**Answer: the expected values were derived from the C, not from Go output.** They are not a
snapshot of `getMinusDam`; they are hand-computed from `src/fight.c:1722-1748` including the
C's float behaviour, and they are correct. The giveaway is the awkward row
`{ac: -230, want: 43}`: C's branch is `dam - (dam*(.28*2.0))`, and in IEEE-754 double
`0.28*2.0 = 0.5600000000000000533`, so `100 - 56.000000000000005 = 43.999999999999994`,
which C's `int` return truncates to **43** — not the 44 a clean decimal would give. Only
someone working from the C expression produces 43. I re-derived every row of the table from
`src/fight.c` and they all match, so the row is R under the keep-rule despite the file name
saying "golden".

## The three clearest junk examples

1. **`TestProcessCombatPair_MobAttack`** — `pkg/combat/round_test.go:498`, unknown agent,
   2026-05-19 (the `website/modernization` merge that landed the first engine). Its only
   assertion is `if defender.hp > beforeHP { t.Error(...) }`. No combat bug can make a
   defender *gain* HP, so the check cannot fail for any reason it names. Mutation: making
   `CalculateDamage` `return 0` (the exact "no damage is dealt" bug) left it **green**.
   *Why it exists:* the round-runner was introduced in the same commit and needed *some* test
   of the new `processCombatPair` path; the author picked a smoke assertion that is trivially
   true. It is padding coverage, not proof.

2. **`TestCounterProcs_MajorMilestone`** — `pkg/combat/round_test.go:232`, unknown agent,
   2026-05-19. It asserts `len(statCalls) != 3` and its comment says *"C bug causes all three
   stats to be incremented (fall-through)"*. Mutation: adding the extra hit increment that C's
   fall-through actually performs made it **red** — i.e. the test pins the port's behaviour,
   not the C's (see the bug list below). *Why it exists:* the author read the bug off the Go
   code's own comment chain rather than off `src/fight.c`, so the test froze the misreading.
   This is the J10 shape the brief asks about: a test written to match buggy code.

3. **`TestScriptedRoller` / `TestSeededRoller` / `TestNewNamedCombatant`** —
   `pkg/combat/roller_test.go:7`, `:37` and `pkg/combat/named_fixture_test.go:4`. All three
   test fixtures, not the game. `ScriptedRoller` and `SeededRoller` live in *production*
   `roller.go` (`:70-134`) but have no production caller; `NewNamedCombatant` is a test-file
   fixture. *Why they exist:* the Tier-2 RNG-seam PR (`tier2/rng-seam-combat`, PR #34,
   2026-06-23, attributed to `claude` via the PR head branch) introduced the injectable
   `Roller` and unit-tested the double itself alongside the real behaviour. They are J7
   test-only seams and are the only three D rows.

Honourable mention: **`TestCombatTranscript_Golden`** (`combat_transcript_golden_test.go:57`,
`tier2/golden-tests-combat-tables` PR #36, 2026-06-23, `claude`) is a Go-vs-Go golden — the
file is written by `-update` from Go output and its header claims "fidelity". Mutation:
changing `pcmod` from 2.0 to 3.0 turned it **red**, so it is a working tripwire; its defect is
provenance (J1/J10), not vacuity, which is why it is F ("keep the behaviour, fix the check")
rather than D. It should be regenerated from the C oracle.


## Mutation checks

All 15 F rows were mutated (`mutations.tsv`). 13 stayed **green** (the assertion cannot fail
for the bug it names); 2 went **red**:

- `TestCounterProcs_MajorMilestone` — red, because the mutation made the port match the C.
- `TestCombatTranscript_Golden` — red, confirming the tripwire works.

Every mutation was reverted with `git checkout -- .` and the worktree verified clean
(`git status --porcelain` empty) after each one. The three D rows are test-only fixtures, so
they make no "cannot catch a regression" claim and needed no mutation.

## Tests whose expected value disagrees with the C (possible game bugs)

1. **`TestCounterProcs_MajorMilestone` / `CounterProcs`** (`pkg/combat/fight_core.go:1059`,
   test `round_test.go:232`). C (`src/fight.c:1274-1284`) is
   `switch(number(1,3)) { case 1: GET_MAX_HIT++; case 2: GET_MAX_MANA++; case 3:
   GET_MAX_MOVE++; default: GET_MAX_HIT++; break; }`. With no `break`s, roll 1 gives
   **hit +2**, mana +1, move +1; roll 3 gives hit +1, move +1. The port always does hit +1,
   mana +1, move +1 — so a major milestone at roll 1 short-changes the player one max-hit, and
   at roll 3 wrongly grants a max-mana. C also consumes a `number(1,3)` draw here and the port
   consumes **none**, an R3 draw-stream divergence in the same branch. The test asserts the
   port's 3-call behaviour, so it is J10. **This looks like a real game bug** (R1 player-facing
   state + R3 draw parity); I did not fix it — out of scope, read-only.

No other expected value in the package contradicted `src/` on the checks I ran. Two tests have
names their assertions do not earn but no C disagreement:
`TestCalculateDamage_NonNormalAttackType` (`formulas_test.go:944`, the name promises the
getMinusDam bypass, the assertion is only `dam >= 1`, J3) and
`TestCalculateDamage_PlayerWithWeapon` / `_AttackerDeadStr` (floor-only assertions).

## Open questions

- **CounterProcs** needs a maintainer decision: is the port's "all three stats +1" a
  deliberate simplification or an unfixed misreading? If C fidelity is the goal, the port owes
  both the fall-through increments and the `number(1,3)` draw.
- 128 rows are attributed `unknown` because the pre-agent era (76 `BRENDA69`, 40 `zach`, 12
  `Zach Greene`) left no branch prefix or trailer. If the research track wants per-agent slop
  rates for that period, it needs another source; I did not guess from style, per the brief.
- 11 rows have no branch at all (commits merged straight to `main` or via non-PR merges, e.g.
  `8c574d2b` "Merge fix/daeron-low-hanging-fruit", `7234d35d` "merge: integrate
  website/modernization branch"); their `pr` column is empty and `author_agent` falls through
  to `unknown`.
- I could not test the transcript golden's bytes against the C because the brief forbids
  running the oracle. Whether the committed transcript matches the C is therefore
  **unverified** — it is the strongest candidate for "fix the check" work.
- The `C` rows are concentrated in `formulas_test.go` spot checks (7 THAC0, 3 getMinusDam)
  that the exhaustive `thaco_golden_test.go` / `ac_reduction_golden_test.go` supersede. They
  are harmless but they are exactly the "repeated invocations of shared helpers" the OpenClaw
  audit reference rejects; a lane plan should fold them into the golden owners.
- Method note: `git log -S'func TestName('` finds the introducing commit reliably here, but
  one commit subject (`5545b944`, "fix(combat): nil guard, ...") contains an embedded newline,
  which breaks naive TSV assembly; I collapsed newlines before writing `attr-raw.tsv`.

