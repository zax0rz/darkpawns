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

## The pre-oracle "golden" test: corrected after the oracle

`ac_reduction_golden_test.go` (2026-07-05, `51f3fbefe`, PR #84) predates the oracle and claims C fidelity. Both smaller models traced its current values to `src/fight.c:1722-1760`. They took the row where AC −230 gives 43, not 44, as proof that the author derived the values from C's double-precision `0.28*2.0`.

**That inference was wrong, and Sol caught it.** The original 2026-07-05 file had the naive values: `{ac: -230, want: 44}`, plus three neighbors that were also one too high. They were corrected to the C values on 2026-07-17 in `649d5deae` ("align combat death path with C oracle"), five days after the oracle landed. So the test's C-faithful float artifact came from the oracle era. The original pre-oracle literals were plausible paraphrase. An earlier version of this note repeated the smaller models' claim; this section replaces it.

## Update: a frontier model on the same brief

Sol (gpt-6.1-sol, at medium reasoning effort) ran the same blind brief later the same day and reported no contamination. Its ledger is in [`../test-audit/pilot-combat/sol/`](../test-audit/pilot-combat/sol/), and the scored predictions, Claude's and Zach's, are in [`PREDICTIONS-sol.md`](../test-audit/pilot-combat/PREDICTIONS-sol.md).

| | MiMo | DeepSeek | Sol |
|---|---:|---:|---:|
| R (keep), by function | 182 | 174 | 159 |
| F / C / D | 7 / 15 / 4 | 15 / 16 / 3 | 16 / 28 / 5 |
| Mutations | 8 | 15 | 23 |
| Time | 34 min | (timed out, then nudged) | 8 min |

- **Agreement.** Sol agrees less with each smaller model (kappa 0.49 with MiMo, 0.44 with DeepSeek) than they agree with each other (0.70). Fleiss' kappa across all three is 0.53, and all three give the same mark to 165 of 208 tests. Most of the gap is Sol's merge marks: it proposed 28 consolidations where the others proposed 15–16.
- **What only Sol found:**
  - the AC golden history above;
  - **a wait-state divergence.** C keeps two wait counters: `ch->wait`, which `WAIT_STATE` writes (`src/utils.h:462-464`), and `GET_MOB_WAIT`, which only the Lua bridge writes (`src/scripts.c:2017`). C's mob attack, flee and cast gates read only the second one (`src/fight.c:1977-1982`, `src/act.offensive.c:377`, `src/spell_parser.c:845`). Go's `MobInstance.WaitState` holds both (`pkg/game/mob.go:1001-1015`). Claude followed the lead: a player skill that waits its target writes the mob's field (`pkg/command/skill_commands.go:1792-1794`), and Go's flee and cast gates read that field (`pkg/game/flee.go:62`, `pkg/game/world_bridge.go:1193`). So a mob a player has just bashed can't flee or cast in Go, while in C it can. This is a static reading, not oracle-proven. Sol reported only the narrower symptom, that the combat engine ignores script-set mob timers.
- **What Sol missed:** it marked the natural-20 test C (merge) rather than F, and it left the `counter_procs` reachability open, though it found the divergence. Its own notes say some keep marks are provisional, because it didn't read every fixture line by line.
- **The pre-oracle split weakens.** Sol put 3 F and 3 C on post-oracle tests, so "every weak test predates the oracle" holds for the two smaller models but not for all three. The direction is the same: 93% of Sol's post-oracle marks are R, against 62% before the oracle.

## Update: the fix phase

DeepSeek then took the 13 tests the settled ledger ([`SETTLED.md`](../test-audit/pilot-combat/SETTLED.md)) marked F for a test-only fix, under a brief that required expected values derived from `src/`, dice forced with `ScriptedRoller`, and a three-part proof per row: the old test green under the audit's recorded mutation, the new test green on real code, the new test red under the same mutation. PR #2278 merged as `ac73fe8a1`; the per-row record is [`fix-phase/results.tsv`](../test-audit/pilot-combat/fix-phase/results.tsv).

- **13 of 13 tests that could not fail now can.** 10 strengthened, 3 folded into the keeper the audit named. No row was `baseline-red`, so every recorded F mutation really did pass the old test. No row was `possible-bug`: every C-derived expected value matched the port.
- **Replicated.** Claude reran rows 1, 9, 10 and 13 independently: the old test green and the new test red under each mutation, the new test green unmutated. Claude also checked every expected value against the cited C lines.
- **The setup hid the guard more often than the assertion did.** In 9 of the 10 strengthened rows the fixture changed, not only the assertion. The usual hider was an input outside the branch under test: AC 100 sits above every `get_minusdam` bracket (`src/fight.c:1727`), so two damage tests could not see the reduction or its bypass; a neutral victim returns before the NPC-killer guard (`src/fight.c:488-493`); and the default fixture's race 0 matched all five race-hate slots of 0 (`src/fight.c:1466-1468`), silently adding 5 × level to a swing. The tenth row (the parry range) needed the harness instead: a scripted roller ignores the bounds it is asked for, so the test now records the draw.

## Limits

- **One package.** `pkg/combat` is formula-dense and C-cited, and is probably the cleanest package in the repository.
- **Attribution has a gap.** About 62% of tests (128 of 208, by DeepSeek's count) predate agent branches and trailers. They're authored "BRENDA69", "zach" or "Zach Greene", so they can't be assigned to a model from git alone.
- **`TestCombatTranscript_Golden` is a working Go-versus-Go tripwire,** red under mutation, but it proves nothing about C. It should be regenerated from an oracle transcript.
