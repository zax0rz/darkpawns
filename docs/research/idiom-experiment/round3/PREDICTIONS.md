# Round-three population and predictions

Written before either package sample was drawn or any mutation census was
started. Source base for both package populations and every eventual probe:
Part 1 commit `aa867bf4dfa25c0fcafe337c11071d56ed7aee40`.

## Population correction from round two

The round-two collector treated selector and ordinary calls alike. A
one-argument selector such as `ctx.SendToChar(message)` carries the message in
argument 0; ordinary `SendToChar(ch, message)` carries it in argument 1. The
collector now distinguishes this call shape and has a self-test for both forms
and the corresponding `SendToRoom` selector.

The corrected historical population was generated against source commit
`5e9556a8789c4125bed8913d34d16e6f2ed68fac`, into
[`corrected-round2-5e9556a87/`](corrected-round2-5e9556a87/). It has 404
function rows, two more than the frozen 402-row population: `Dream` and
`DreamTravelFn` in `pkg/game/dreams.go`. Those rows add 29 eligible literals.
The frozen round-two `sample.tsv`, `probes.tsv`, and denominator of 40 remain
unchanged.

| Artifact | SHA-256 |
|---|---|
| Frozen round-two functions (402 rows) | `a790e7a2d87f334eb37495f1433714de54b388d8d7d9f865ce1f13c52451b631` |
| Corrected historical functions (404 rows) | `e442e3cfb2a7e9763ee173cecc00b98b098b3f2a947a4bcaf0b1f7a024fb04d6` |
| Corrected historical literals | `61316a3b33a6a3dd394c687e5736967aa644254de2ac08be6ad35379801f66da` |

## Round-three populations

The sorted function rows and their source-offset-sorted literal inventories are
stored separately per package under [`population/`](population/). The collector
uses the corrected round-two script. For `pkg/spells`, its output-call catalog
also includes the package's message-forwarding helpers (`sendToCaster`,
`sendAffectRoom`, `sendAct`, reagent messages, and related wrappers), so literals
passed through those helpers are in the package population.

| Package | Function rows | Function TSV SHA-256 | Literal TSV SHA-256 |
|---|---:|---|---|
| `pkg/session` | 154 | `3494855e2685510eee44cb1deeb435c0e0ee5260840b0e9d34469a7c9e602ecf` | `7063b2f178f29e26075cb10e4d2d790edff3d84f84531f2365f514cdfad6ec32` |
| `pkg/spells` | 42 | `bc5f237f34db3a5e5d57a77807446d60a9bc0438f0e1dace770f8c73db2b3fa0` | `49d4f28b01e62959d3ff3aea3cdf0bef769e70b08d3abf10ba6fc316360b5e00` |

The population remains the round-two syntactic, same-function measure: it
follows recognized output argument positions and string-like local assignments,
not arbitrary interprocedural flows. Reproduce the two round-three hashes from
the repository root with:

```bash
sha256sum \
  docs/research/idiom-experiment/round3/population/session/population.tsv \
  docs/research/idiom-experiment/round3/population/session/population-literals.tsv \
  docs/research/idiom-experiment/round3/population/spells/population.tsv \
  docs/research/idiom-experiment/round3/population/spells/population-literals.tsv
```

## Catch-rate predictions

Round two caught 23 of 40 probes. With a Jeffreys prior, that gives a pooled
`Beta(23.5, 17.5)` catch-probability posterior. I use its beta-binomial
posterior predictive distribution for each new 40-probe sample. There is no
prior package-specific catch evidence, so the two forecasts are the same.

| Package | Expected catch rate | 80% predictive interval | Expected caught probes |
|---|---:|---:|---:|
| `pkg/session` | 57.3% | 42.5%–70.0% (17–28 of 40) | 22.9 of 40 |
| `pkg/spells` | 57.3% | 42.5%–70.0% (17–28 of 40) | 22.9 of 40 |

Each sample will use `random.Random(20261106).sample(rows, 40)` for session and
`random.Random(20261107).sample(rows, 40)` for spells, with each sorted function
row drawn at most once. Probe `i` uses `random.Random(package_seed + i).choice`
over that function's literals in source-offset order. The mutation changes the
first ASCII letter to `Q`, or to `Z` if it is already `Q`. Function selections,
literal choices, and predicted scenarios will be written and committed before
the first census starts.
