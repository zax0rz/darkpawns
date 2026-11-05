# A random sample finds the census's blind side

2026-11-05. Field note by Claude, from reviewing round two of the idiomatic-Go experiment.

## The experiment

Zach adapted Google's `migrate-to-rust` skill into a long unattended Luna (Codex) goal: rewrite "translated C" Go in `pkg/game`, `pkg/combat` and `pkg/command` as idiomatic Go, one commit at a time. Each commit had to keep the targeted census green, and each was followed by a **mutation probe**: change one letter of a player-facing message in the touched code and confirm the census goes red. Both rounds' predictions were committed to main before the runs started (`dce50f5bf`, `b03629cc8`).

- **Round one** ([PR #2275](https://github.com/zax0rz/darkpawns/pull/2275)): 7 commits kept, **7 of 7 probes caught**.
- **Round two** ([PR #2276](https://github.com/zax0rz/darkpawns/pull/2276), branch tip `1ff951088`): Tier A (`interface{}` → `any`, 88 sites) plus 21 extraction commits, all with clean targeted censuses.

Round one's perfect catch rate was an artifact. Its brief ranked targets that the census visibly covered first, so the probes only went where the census was already looking. Round two fixed this by drawing its probe sample at random, before any refactor.

## Finding 1: 42.5% of a random sample is invisible to the full census

Round two froze a population of **402 functions** that pass a source string literal to a player-output call. It committed the population's SHA-256, then drew 40 functions with a fixed seed (`random.Random(20261105).sample(rows, 40)`). For each function, a seeded choice picked one literal, and its first ASCII letter was changed. Each probe ran a **full** census, retained under `/home/zach/Archives/darkpawns/oracle-runs/2026-11-04/luna-r2-probe-*`.

**17 of 40 mutations stayed green (42.5%; 95% Wilson interval about 28–58%).** Claude's pre-registered guess was 30%, with a 15–50% range. The census changed a message the player sees in 17 sampled functions and did not notice.

The most useful misses are the five where the agent named, *before* the run, the scenario that should catch the change, and that scenario stayed green:

| Function | Scenario that should have caught it |
|---|---|
| `findDoor` | `door-basic` |
| `CmdSerpentKick` | `serpent-depth` |
| `weatherChangeLocked` | `weather-sky-transitions` |
| `CmdSkills` | `skills-practice` |
| `executeShoot` | `shoot-mob-hit-depth` |

A scenario's name is a claim about what it reaches, and these five claims are false for the sampled message. This is the same class as PF-099, where every `practice` scenario used a class whose costs were all zero. Across all 40 probes, the agent's pre-run predictions were 23 exact, 7 partly right and 10 wrong.

Limits:

- A green probe can't tell an unreachable function from an unexercised branch inside a reached one.
- The mutation is one letter of an output literal. It says nothing about logic changes the census would or wouldn't catch.
- The population script missed at least `Dream` and `DreamTravelFn`: a selector-call argument index bug, documented in the run's `REPORT.md`. The frozen sample was kept as drawn.

## Finding 2: there was little "translated C" to remove

The de-slop half of the experiment mostly didn't happen:

| Measure | Round-two baseline | After | Claude's prediction |
|---|---:|---:|---:|
| Lines in scope | 73,540 | 73,641 (+101) | −300 |
| Functions over 100 lines | 45 | 33 | 30 |
| Longest function | 747 | 747 | 400 |
| snake_case declarations | 126 | 126 | 15 |
| Game bugs found | | 0 | 6 |

Extraction adds function signatures, so lines went up. Most of the drop in long functions came from splitting a function at a natural seam (for example, `DoBackstab` → `backstabAfterGates`). That's real, but cosmetic. None of the remaining snake_case names was an eligible rename: they are exported or mirror C names.

Claude reviewed all 21 extraction commits and found no behavior change:

- Only three move an RNG draw (`tryHuntEvasion`, `specRoachFood`, `scavengerPickup`). Each moves the code verbatim, at its one call site, with short-circuit order intact (R3a, R3b).
- Only two of the void helpers contain an early return, and in both the call is the caller's final statement.
- The commits add no defers, goroutines or locks; the six lock lines are moved, not new.

Reading: the Go is shaped like the C because the fidelity rules require it. "Translated C" in this port isn't slop waiting to be removed. The experiment's value was the measurement it forced, not the refactor.

## Proposed ledger row

Deferred until PR #2271 merges, because it also edits `EVIDENCE_LEDGER.tsv`:

> A pre-registered random sample of 40 player-output functions (from a frozen, hashed population of 402 in pkg/game, pkg/combat and pkg/command) found 17 whose one-letter output mutation the full census did not detect (42.5%); the agent-chosen probes of round one were 7 of 7 caught. Five misses sat in functions whose named scenario (door-basic, serpent-depth, weather-sky-transitions, skills-practice, shoot-mob-hit-depth) was predicted to cover them.

Primary evidence: `docs/research/idiom-experiment/round2/run/probes.tsv` and `RESULTS.md` on PR #2276, plus the probe census directories above.
