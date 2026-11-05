# Idiomatic-Go experiment, round two: predictions

Written 2026-11-04. These predictions were committed to main before the run started, which dates them. Don't edit them after the run starts. Record results in `round2/RESULTS.md`.

## What round one showed

Round one is draft PR #2275, branch `luna/idiomatic-go-2026-11-03`, tip `5e9556a87`. It ran in Codex, which reported its model as GPT-6.

The run made 7 small refactors. All 7 were clean on their targeted censuses, and all 7 one-byte mutation probes were caught. Claude reviewed every diff and found no behaviour change. The scope shrank by 17 lines (73,557 to 73,540), and functions over 100 lines went from 48 to 45. Most other targets were marked "Skipped: unsure".

The brief had two flaws, both Claude's:

1. **"Contains a random draw" was treated as "would move a random draw".** The brief forbade moving RNG calls and said to skip when unsure. Nearly all the game logic draws somewhere, so the agent skipped almost everything. It also skipped the compiler-checked renames.
2. **Probe targets were not sampled.** The brief ranked targets "census-visible first", so the probes only ever hit code that scenarios already reach. A 0/7 green rate from targets chosen that way says nothing about the rest of the code.

The brief also gave a wrong metrics command. `metrics/` has its own `go.mod`, so the command has to run from inside that directory. The agent noticed this and recorded it in `run/setup-notes.md`.

## Changes in round two

- **Tier A, compiler-checked changes, are allowed outright:**
  - `interface{}` → `any`. They are the same type, so the change is a spelling change only.
  - Scope-aware renames of unexported identifiers with `gopls rename`, with the checks listed in the brief.
- **The RNG rule is clarified.** A block that contains draws may be extracted into a helper when the helper has exactly one call site at the same point, and every early return is carried back to the caller. A draw is forbidden only if it moves past an early return, crosses a short circuit, or ends up running a different number of times. Containing a draw is not a reason to skip.
- **The visibility measure is separated from the refactor.** Before any refactor, the agent defines the population of in-scope functions that send player-facing text and records its sha256. It then draws **40 functions at random with seed `20261105`** and probes each one on the base commit. Coverage plays no part in which functions are probed.
- **There is no time limit.** The run stops when the inventory is exhausted, or when 3 targets in a row are abandoned.

## Baseline

Measured at the round-one tip `5e9556a87`, which round two builds on:

```bash
cd docs/research/idiom-experiment/metrics && go run . ../../../.. pkg/game pkg/combat pkg/command
```

| Measure | Value |
|---|---:|
| Lines in scope | 73,540 |
| Functions | 2,730 |
| Functions over 100 lines | 45 |
| Functions over 200 lines | 9 |
| Longest function | 747 |
| Functions nested 5 or more deep | 21 |
| snake_case declarations | 126 |
| Empty `interface{}` | 88 |
| Package-level vars | 205 |

Round one recorded the lint counts at its tip: funlen 191, gocognit 111.

## Predictions

Each prediction has an 80% interval: I expect 8 of every 10 true values to land inside it. Round one taught me to widen these, especially for row 6.

| # | Measure | Baseline | Claude predicts | 80% interval | Zach guesses |
|---|---|---:|---:|---|---|
| 1 | Empty `interface{}` | 88 | **0** | 0 to 5 |  |
| 2 | snake_case declarations | 126 | 15 | 0 to 60 |  |
| 3 | Net lines in scope | 73,540 | −300 | +1,000 to −2,500 |  |
| 4 | Functions over 100 lines | 45 | 30 | 15 to 43 |  |
| 5 | Longest function | 747 | 400 | 150 to 747 |  |
| 6 | Tier B refactor commits kept (extractions, not renames) | 7 | 20 | 5 to 50 |  |
| 7 | Share of the ranked inventory skipped as "unsure" | most | 45% | 20% to 75% |  |
| 8 | Tier B commits whose first targeted census went red | 0 | 3 | 0 to 10 |  |
| 9 | **Probed sample functions where the probe stays green (the census can't see them)** | | **30%** (12 of 40) | 15% to 50% |  |
| 10 | Sample functions no scenario reaches at all | | 20% | 5% to 40% |  |
| 11 | Latent port bugs logged | 2 | 6 | 1 to 20 |  |
| 12 | Protected-path edits | 0 | 0 | 0 to 0 |  |
| 13 | Morning review finds a kept, green commit that still changes behaviour | | 30% | | |

## Reasons

- **Rows 1 and 2.** Tier A changes are mechanical. Some snake_case names will survive because they are spelled C-style on purpose: Lua script bindings, C-named tables, or names that show up in strings.
- **Row 3: the line count barely moves again.** Renames don't change line counts, and extracting functions adds signatures.
- **Rows 6 and 7.** With the rule clarified, the combat and skill paths are open. They are also where the risk is. I expect the agent to stay cautious, so the number of kept commits could still be low.
- **Row 9 is the research number.** A random sample removes the bias round one had. Breadth scenarios reach the common paths, and PF-095 and PF-096 showed hollow proofs. So I expect about a third of player-facing functions to be invisible to the census.
