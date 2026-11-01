# Predictions (written before coding any row)

**Author:** DeepSeek (session 2026-11-01). **Status:** frozen at the moment of
writing; not edited afterwards.

Written after reading `STUDY.md`, `BRIEF-slop-track.md`, the screening register
counts, and the Pan et al. taxonomy, but **before** opening any individual fix
commit or coding any row of `slop.tsv`.

## Context I am predicting against

- Port begins 2026-04-17 (first commit). The reference C oracle is dated
  2026-07-12. The bulk of the port predates any differential check.
- The git leads come from a search whose terms were only used after 2026-09-28,
  so the *fixes* in the sample are late-biased (604/606 ≥ 2026-09-28). Births
  are the correcting variable.
- Known exemplars named in the brief: yank handler (PF-002), a couch that should
  have been a mimic (PF-010), a foreign level scale (PF-055), scenario-fitted
  special cases (PF-040), 36 invented command names (PF-042).

## Predictions

**P1 — Which Pan categories dominate.** I expect the port's mistakes to cluster
in the *semantic* groups, not the *syntactic* ones, because Go is statically
typed and the compiler is a free A1/B1/B2 filter — an agent that produced code
that did not compile would have noticed immediately.

- **P1a:** the single most common category will be **B5 "Removal of logic from
  source"** (omission: a branch, a check, a whole command path C has and Go
  does not). Ranking prediction: B5 > B4 > C3 > A3 > B3.
- **P1b:** **B4 "Inclusion of logic not in the source"** (invention) will be
  second, driven by invented messages and invented command names.
- **P1c:** **C3 "Output formatting"** will be a large third: the game is a text
  surface, and paraphrased strings are exactly the kind of thing a compiled port
  hides.
- **P1d:** the syntactic categories **A1, A2, B1, B2** will be near-absent
  (the compiler catches them); **D "Model specific constraints"** will be absent
  (no token-window failure can survive a build).

**P2 — Share of births before the oracle (2026-07-12).** I expect a **large
majority, ~70–85%**, of mistakes to be *born* before the oracle existed. The
port ran for ~3 months before the oracle and "felt right"; the whole story is
that winging it left a large backlog that only later checks found. Working
backwards to birth dates is exactly what un-biases the late-fix sample, so I
expect the born-before-oracle share to be much higher than the (late) fix-date
distribution would suggest. Point estimate: **78%**.

**P3 — Median lifetime (birth → fix).** I expect a **median of roughly 100–150
days**, point estimate **~120 days**, because most births are in the May–July
"winging it" phase and most fixes are Sep–Oct. I also expect a long right tail
(bugs born in the first weeks of the port) and a small cluster of short
lifetimes (bugs born during depth work and caught within days).

**P4 — Dominant catch method.** I expect **`oracle-depth`** to dominate (long
agent runs that prove behaviour site by site), with **`oracle-breadth`** second
(one green per command), and **`playtest`/`returning-player`** a smaller but
high-precision tail that catches things no scenario claimed. Point prediction:
`oracle-depth` > `oracle-breadth` > `playtest` > `returning-player` > `audit` >
`code-review`. I expect `unknown` to be non-trivial because many fix commits do
not say how the bug was noticed.

**P5 — Author.** Given the port's workflow, I expect the births to be spread
across several agents (glm, codex, deepseek, sol, claude, cline, gemini). I
expect a large `unknown` share, because birth commits are often squashed
"phase" commits authored by Zach rather than a labelled agent branch, and I will
not guess from style.

## Things I explicitly do **not** predict (to avoid post-hoc fitting)

- I do not predict which categories never appear; I will report whatever is
  empty rather than guess.
- I do not predict the `none-fits` cases beyond expecting **at least a few**,
  since the port's "the game is the game" fidelity failures (paraphrase,
  invented-but-plausible content) may not map cleanly onto a taxonomy built for
  algorithm translation.

## Falsifiability

These are recorded now so the report can score them. Any of P1–P5 that fails is
kept as a failure in `slop-notes.md`, not quietly revised.
