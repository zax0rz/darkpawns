# Settled ledger: `pkg/combat` test audit

2026-11-05, by Claude. Merges the three blind ledgers (MiMo, DeepSeek, Sol) into one decision per test: [`settled.tsv`](settled.tsv).

## How each mark was settled

- **Unanimous (163 tests):** the shared mark stands.
- **Two of three agree (20 tests):** the majority mark stands.
- **Claude ruled (25 tests):** every test with no majority, every majority F, and every test where any model's mutation stayed green. One rule decided most of them: **a mutation that breaks the behavior a test claims to protect, and stays green, beats a vote.** A mutation that breaks behavior *outside* the test's contract doesn't count. For example, a "no panic when the hook is missing" test is right to stay green when the computation is removed. Each ruling's reason is in the `reason` column. Two of Sol's minority claims (the alignment tests) were checked against the test source and hold.

## Result

| Mark | Tests |
|---|---:|
| R (keep) | 172 |
| F (fix the check) | 17 |
| C (merge) | 15 |
| D (delete) | 4 |

## What happens to the F tests

| Action | Tests | Who |
|---|---:|---|
| `strengthen`: tighten the assertion until the recorded mutation goes red | 10 | fix phase |
| `fold-into-keeper`: move the case into the stronger table test that owns it | 3 | fix phase |
| `with-bug`: replaced by the `counter_procs` fix PR | 2 | counter_procs brief |
| `needs-oracle`: expected bytes or behavior need an oracle run first | 2 | later: golden transcript rebuild, PF-105 |

C and D tests are on **hold**. Merging and deleting is mostly taste, and it risks losing coverage. Nothing moves there until the fixes land.

## Measuring the fix phase

For each of the 13 fix-phase tests: rerun the mutation recorded in the model ledgers against the strengthened test. It must now go **red**, and the unmutated code must stay green. The result is "N of 13 tests that could not fail now can", with one mutation as proof per test.
