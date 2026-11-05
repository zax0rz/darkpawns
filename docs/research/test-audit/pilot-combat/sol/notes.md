# Combat test audit

Pre-reading prediction (top-level functions): R 158, F 23, C 20, D 9, assuming 210 functions.

Contamination: none

## Baseline and counts

Pinned commit `b03629cc8`. Both `go test ./pkg/combat/...` and the uncached JSON baseline passed. The package has 38 test files (7,152 lines) and 10 production Go files (4,606 lines). The Go inventory contains 208 top-level tests and the baseline emitted 467 runtime subtest pass events, including dynamically generated corpus cases. `TestMain` is package setup and does not appear in the Go inventory. One table was split where its subtests earned different marks, producing 209 ledger rows. CI routes this package through the repository-wide Go test job in `.github/workflows/ci.yml:44`.

| | R | F | C | D |
|---|---:|---:|---:|---:|
| Pre-reading prediction, functions | 158 | 23 | 20 | 9 |
| Actual, ledger rows | 160 | 16 | 28 | 5 |
| Introduced before July 12 | 71 | 13 | 25 | 5 |
| Introduced July 12 or later | 89 | 3 | 3 | 0 |

R accounts for 76.6% of ledger rows. Junk-code counts can overlap. J1 1, J2 4, J3 16, J4 28, J5 2, J6 0, J7 2, J8 2, J9 4, J10 3. Both J7 rows retain shared test infrastructure.

Attribution by row was unknown 128, codex 29, glm 26, claude 15, sol 6, and deepseek 5. Every function has an introduction commit and date. The commit-to-PR endpoint returned associations for 54 of 62 unique introduction commits. First-parent or trailer evidence was used where the API was empty. An unrecognized branch prefix or a human git author alone leaves the agent unknown. The introduction attribution is distinct from authorship of subsequent changes.

The 23 scratch mutation runs yielded 20 green and three red. Every F and four of five D rows have a production mutation. The fifth D exercises a fixture defined exclusively in a test file, so there is no production line to mutate. For the red NPC-wait probe the mutation moved the behavior toward C. For the red AC control the mutation deliberately broke a C threshold. Both worktrees were verified clean before removal.

## Three clearest junk examples

- `pkg/combat/formulas_test.go:760`, `TestCalculateHitChance_LowACAlwaysHits`. The loop drops all 100 return values. A mutation that makes every awake non-natural-20 roll miss stays green. The introducing commit `8c574d2b5` calls this a testing push, counts 55 formula tests and cites combat-formula coverage. This looks like a coverage or smoke probe that acquired a stronger name than its assertions warrant.
- `pkg/combat/round_test.go:572`, `TestProcessCombatPair_PlayerDeath`. The victim is a Goblin NPC. Every outcome, including survival, merely logs. Suppressing every round swing stays green. The introducing commit `7234d35d0` says “test: add combat round execution tests”. The body reads like a diagnostic experiment carried into the suite without converting its observations to a failure condition. The commit provides no more specific motive.
- `pkg/combat/combat_transcript_golden_test.go:57`, `TestCombatTranscript_Golden`. The committed bytes come from `simulateCombat` via `-update`. The introduction `b78a3df20` explicitly calls it a “Go-vs-Go regression tripwire” and says the aim was to demonstrate the seeded RNG seam. That gives it formula-regression value. It cannot certify C parity or live round and death dispatch, as the live-engine death mutation illustrates.

## Expected values that diverge from C

- `pkg/combat/round_test.go:232`, major milestones. The test requires exactly three stat callbacks without forcing a roll. `src/fight.c:1280` draws 1–3. Roll 1 increments HP twice and also increments mana and move. Roll 2 increments HP, mana and move once each. Roll 3 increments HP and move. `pkg/combat/fight_core.go:1074` always applies one of each and skips the draw. The world death implementation at `pkg/game/death.go:1375` has the C roll and fall-through effects. Reachability of a player reaching a milestone through the combat package's separate fallback remains a follow-up.
- `pkg/combat/engine_test.go:1115` and `pkg/combat/engine_standup_test.go:145`, positive NPC wait. The first expects the waited mob to stand and its wait to remain unchanged. The second also demands a to-hit draw. `src/fight.c:1977` consumes the separate mob wait and zeroes attacks after the count and defense draws. Its stand-up gate also reads that timer. The introducing commit `07d1e4e8a` (#479) argued that Go's scripting layer wrote no mob wait. At the pinned revision `pkg/game/world_bridge.go:251` writes the script timer through `MobInstance.SetWaitState`, and `pkg/game/npc_command_test.go:211` explicitly identifies the field as C mob wait. The engine ignores it. The two fixtures do not distinguish generic wait from script wait by origin. A mutation suppressing attacks on positive NPC wait failed the test that demands a hit draw and passed the test whose name alone promises a swing.

## AC golden and open questions

`51f3fbefe` describes the AC literals as C thresholds, and the original test compares against a literal table. Its four original expectations at AC -230, -249, -250, and -269 were respectively 44, 44, 43, and 43. The commit does not record whether those numbers were calculated by hand or copied from the port. `649d5deae` changed them to 43, 43, 42, and 42. Independently evaluating the completed C double expression in `src/fight.c:1722–1758` and truncating at return matched all 64 current rows. Moving the AC 90 cutoff made the golden fail.

The inventory is complete. Source review used an assertion inventory with targeted full-body and call-path reads. Some R rows remain provisional because a full line-by-line review of every fixture and every production caller was not completed. The script-timer finding is a static call-path finding with a focused unit mutation. Its prevalence in actual Lua combat was not measured. No C oracle or census was run. The ledger is outside the repository as directed by the brief.
