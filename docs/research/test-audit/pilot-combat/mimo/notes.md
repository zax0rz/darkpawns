# Test audit — pkg/combat — agent mimo

## Setup facts

- Commit under audit: `b03629cc8` (detached worktree `~/wt/mimo-test-audit`).
- Baseline `go test ./pkg/combat/...` at 2026-11-04: **PASS** (`ok github.com/zax0rz/darkpawns/pkg/combat 0.062s`).
- Inventory: 208 top-level Test/Benchmark/Fuzz/Example functions (`inventory.txt`).
- Mutation worktree: `~/wt/mimo-test-audit-mut`, 8 mutations recorded in `mutations.tsv`.

## Pre-reading guesses vs actual

My guess (written before reading any test):

| Mark | Guess | Actual |
|------|-------|--------|
| R    | 60% (~125) | 87.5% (182) |
| F    | 12% (~25)  | 3.4% (7)   |
| C    | 15% (~31)  | 7.2% (15)  |
| D    | 13% (~27)  | 1.9% (4)   |

I substantially overestimated the junk. I expected agent-written padding and
Go-generated goldens to dominate; instead, the pkg/combat suite is mostly
high-quality fidelity tests with C citations, exact byte assertions, and
regression IDs (DP-900, DP-1021, DP-1033, DP-1204, DP-1213, DP-1215, DP-1318,
DP-1330, PL-005). Claude's recorded guess (~75% R) was closer but still
underestimated R by 12 points.

## Counts

### By mark
| Mark | Count |
|------|-------|
| R    | 182   |
| C    | 15    |
| F    | 7     |
| D    | 4     |

### By junk code
| Code | Count | Tests |
|------|-------|-------|
| J2   | 8     | 6 F + 2 D (no-assertion or unfailable statistical checks) |
| J7   | 1     | 1 D (ScriptedRoller test-only seam) |
| J1   | 1     | 1 F (CombatTranscript Go-generated golden) |

### By author agent
| Agent | Count | Notes |
|-------|-------|-------|
| unknown | 131 | Early BRENDA69/Zach commits on `fix/`, `tier2/`, `qa/` branches without Co-Authored-By trailers |
| codex   | 26  | `codex/` and `luna/` branches |
| glm     | 25  | `glm/` branches |
| claude  | 15  | `claude/` branches or Co-Authored-By: Claude |
| sol     | 6   | `sol/` branches |
| deepseek | 5  | `deepseek/` branches |

The "unknown" bucket (63%) is dominated by early porting-phase commits by
BRENDA69 (git author) on generic `fix/`/`refactor/`/`tier2/` branch names that
don't match any known agent prefix. Per the brief's rules, these are `unknown`
with the author name recorded — never guessed from writing style.

### Born before oracle (2026-07-12)
| | Count |
|---|-------|
| Before oracle | 114 |
| After oracle  | 94  |

All 7 F marks and all 4 D marks are **pre-oracle**. Post-oracle tests are
overwhelmingly R (93/94). This supports the hypothesis that the oracle
discipline raised test quality: once there was a C reference to check against,
test authors wrote assertions that trace to it.

## Three clearest junk examples

### 1. `TestProcessCombatPair_PlayerDeath` (round_test.go:~618) — J2
**What:** Runs up to 5 rounds of combat between a strong attacker and weak
defender, then branches three ways — `t.Log` in every branch. No assertion
can fail. The test passes whether the defender dies, survives, or the
engine crashes after logging.

**Why it exists:** Written during the engine port (likely the initial
`processCombatPair` smoke test) to verify the round loop runs without
panicking. Never upgraded to assert death was detected. The comment
"goblin survived with %d HP (all misses — improbable but possible)" shows
the author was aware of nondeterminism and chose to log rather than
eliminate it with a scripted roller.

### 2. `TestChangeAlignment` (fight_core_test.go:~340) — J2
**What:** Wires full callbacks, calls `ChangeAlignment(paladin, evil)`, then
comments "Just verify it doesn't panic — actual value depends on
implementation." No assertion on the alignment value despite the function
having a clear deterministic formula.

**Why it exists:** Written as a nil-guard smoke test during the callback-
wiring refactor (the adjacent `TestChangeAlignment_NilGetAlignment` is a
proper nil-guard R test). The non-nil case was sketched but the assertion
was never filled in.

### 3. `TestCalculateHitChance_Natural20` (formulas_test.go:~648) — J2
**What:** Statistical test: 1000 iterations of `CalculateHitChance` with a
weak attacker (THAC0 20) vs defender (AC 100 → victimAC 10). Asserts
`hits == 0` fails. But rolls 10–19 all hit normally (20 − roll ≤ 10),
giving ~50% hit rate even if natural-20 auto-hit is broken. Mutation
confirmed: flipping `return true` to `return false` on the natural-20
branch leaves this test green.

**Why it exists:** Written before the author knew about (or chose to use)
the `ScriptedRoller` seam. Statistical testing was the default approach
for RNG-dependent code, producing an assertion too weak to catch the
named bug.

## The brief's lead: `ac_reduction_golden_test.go`

The brief flagged this test (added 2026-07-05 in `51f3fbefe`, pre-oracle,
titled "C→Go fidelity") as a possible J1 (Go-generated golden). **Finding:
the expected values trace to the C.** I verified all 60+ entries against
`src/fight.c:1722-1760` (`get_minusdam()`) by computing C double-truncation
semantics in Go. The values match exactly, including floating-point artifacts
that a naive Go-generated golden would not produce:

- ac=−230: C gives `100 − 100*(0.28*2.0)` = `43.999...` → truncates to **43**
  (not 44 from a clean −2 step).
- ac=−229: rounds to **46** (the float lands exactly on 46.0).

These truncation artifacts confirm the values were derived from C semantics,
not naively rounded. Per the keep-rule (R1): "if the expected value matches
`src/`, the test is R even though Go generated it." Marked **R, c-cited**.

## Tests whose expected values disagree with C

None found. All golden tables (`strApp`, `dexApp`, `thaco`, `ac_reduction`,
`position_damage`, `backstab_mult`, `damage_message_tiers`) were spot-checked
against the cited C source lines and match.

## Open questions

1. **63% "unknown" author.** The early porting-phase commits (BRENDA69 git
   author on `fix/`, `tier2/`, `qa/` branches) have no Co-Authored-By
   trailers and no agent-matching branch prefixes. Were these written by
   Zach directly, by Claude without trailers, or by an unidentified tool?
   The brief says "never guess from writing style" so they stay `unknown`.

2. **`TestCombatTranscript_Golden` (F, J1).** The golden is Go-generated
   (`-update` flag) and the test self-describes as "a Go-vs-Go regression
   tripwire." It's useful for catching formula drift but proves nothing
   about C fidelity. Should it be regenerated from an oracle transcript,
   or is its role as a non-fidelity tripwire acceptable with a clearer name?

3. **Subtest explosion in `TestFightMessageCorpus...`.** 332 subtests
   (83 variants × 4 outcomes) is excellent coverage but makes the test
   expensive to run and the failure output enormous. Worth considering a
   table-driven consolidation — though as R, it stays.

4. **`msgMockCombatant` and `waitStateMockCombatant`** are defined in test
   files but used across multiple test files. Not a production seam (J7
   doesn't apply), but worth noting for test-maintenance clarity.

5. **`TestPCGetAttacksPerRound_Statistical`** uses a seeded roller over 10K
   iterations with a ±0.02 tolerance on the mean. This is deterministic
   (seeded) but brittle: a change in the roller implementation could shift
   the mean slightly. Low risk given the seed is fixed.

## Mutation results summary

8 mutations recorded in `mutations.tsv`:
- 7 green (F and D claims proven: tests pass with the bug present)
- 1 red (R proof: `TestStrApp_GoldenAgainstCSource` catches strApp corruption)
