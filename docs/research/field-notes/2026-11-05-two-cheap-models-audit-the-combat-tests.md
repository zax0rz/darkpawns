# Two cheap models audit the combat tests

2026-11-05. Field note by Claude, from reviewing a read-only test audit run independently by MiMo (V2.6 Pro) and DeepSeek (v4.1 flash, through Cline).

## The setup

Dark Pawns has about 196K lines of tests against about 165K lines of Go. Agents wrote most of them. The audit asks which tests can actually fail when the game breaks, and who wrote the ones that can't.

The method is adapted from OpenClaw's `test-audit` skill ([brief](../test-audit/pilot-combat/BRIEF.md)). An agent reads every test in `pkg/combat` and marks each one:

- **R**, keep;
- **F**, fix the check;
- **C**, merge into another test;
- **D**, delete.

It tags the junk with codes J1–J10 and attributes each test from git. Where it claims a test can't catch a bug, it proves that with a mutation in a scratch worktree. Both agents ran the same brief at the same time, each blind to the other's output. Their ledgers, mutation logs and notes are in [`../test-audit/pilot-combat/`](../test-audit/pilot-combat/); git author emails were stripped from the copies.

## Results

The package has 208 test functions. Both ledgers pass the brief's completeness check.

| | MiMo | DeepSeek | Claude's guess |
|---|---:|---:|---:|
| R (keep) | 182 (87.5%) | 174 (83.7%) | 75% |
| F (fix the check) | 7 | 15 | |
| C (merge) | 15 | 16 | |
| D (delete) | 4 | 3 | |
| Mutations run | 8 | 15 | |

**Agreement: 92.3% on the four marks, Cohen's kappa 0.70** ("substantial"). On keep versus not-keep it's 93.3%, kappa 0.73. Both agents found the same commit for all 208 tests, and the same agent for 205.

Where the agents disagreed, the disagreement is where the value was. Each caught a defect the other marked "keep":

- **MiMo:** `TestCalculateHitChance_Natural20` can't fail. With its THAC0 and AC setup, rolls of 10–19 hit anyway, so breaking the natural-20 rule leaves it green. MiMo's mutation proved it. DeepSeek marked it R without a mutation.
- **DeepSeek:** `TestCounterProcs_MajorMilestone` pins wrong behavior, which led to a game bug. MiMo marked it R and reported no C disagreements.

## Every weak test predates the oracle

Both agents found the same thing independently. **Every F, C and D row was written before the first oracle harness (2026-07-12, `ba05d9906`).** After that date, 93 of 94 new tests are R, and the remaining one is a C (merge).

This supports what Zach has observed across the port: once there were real checks and specific rules, agent work started matching the C. It can't separate the two causes, though. The oracle (2026-07-12) and the incident-taught rulebook (2026-07-22, `3d58ba719`) arrived ten days apart.

## The bug: a wrong duplicate beside a faithful path

C's `counter_procs` (`src/fight.c:1252-1311`) rewards kill milestones. Major milestones roll `switch(number(1,3))`, and its cases fall through:

| Roll | Reward |
|---|---|
| 1 | +2 hit, +1 mana, +1 move |
| 2 | +1 hit, +1 mana, +1 move |
| 3 | +1 hit, +1 move |

The port has two copies:

- **`pkg/game/death.go:1379` is faithful.** It was fixed under DP-1040 and serves every kill with a game-layer death callback.
- **`pkg/combat/fight_core.go:1062` is wrong**, in three ways:
  - it makes no draw (R3a);
  - it always gives +1/+1/+1 (R1);
  - it sends `\r\n` where C sends `\n\r` (R1). The census can't see this one, because it normalizes terminators.

Its test asserts the wrong behavior. The wrong copy runs only through `combat.TakeDamage` with no death callback, which in production means a player ambushing another player to death on a milestone. It's rare, but it's the same shape as the stand-ins returning players found (field note 2026-10-25): a faithful path, and beside it a paraphrase that tests keep alive. The fix brief makes the combat copy delegate to the faithful one.

## The pre-oracle "golden" test is legitimate

`ac_reduction_golden_test.go` (2026-07-05, `51f3fbefe`, PR #84) predates the oracle and claims C fidelity. Both agents traced its values to `src/fight.c:1722-1760`. One row clinches it: AC −230 gives 43, not 44, and that only comes out of C's double-precision `0.28*2.0`. Its author derived it from the C, not from Go output.

## Limits

- **One package.** `pkg/combat` is formula-dense and C-cited, and is probably the cleanest package in the repository.
- **Attribution has a gap.** About 62% of tests (128 of 208, by DeepSeek's count) predate agent branches and trailers. They're authored "BRENDA69", "zach" or "Zach Greene", so they can't be assigned to a model from git alone.
- **`TestCombatTranscript_Golden` is a working Go-versus-Go tripwire,** red under mutation, but it proves nothing about C. It should be regenerated from an oracle transcript.
