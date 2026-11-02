# The slop track, v2 (2026-11-01)

The second study: what kinds of mistakes AI agents introduced while porting Dark Pawns, labelled with Pan et al.'s taxonomy ("Lost in Translation", ICSE 2024). It answers research question 2 in [`../STUDY.md`](../STUDY.md). DeepSeek built it from [`../BRIEF-2026-11-01-slop-track.md`](../BRIEF-2026-11-01-slop-track.md); Claude reviewed it in Zach's session.

## Headline numbers

- **201 rows: 178 coded mistakes, 23 excluded** (2 Go-only, 12 with no single C line establishing them, 9 not mistakes).
- **Top categories:** B5 Removal of logic (86), B3 Incorrect loop and conditional statements (40), B4 Inclusion of logic not in the source (21).
- **Pan et al.'s syntactic categories (A1, A2, B1, B2) and model-constraint group (D) don't appear** among committed mistakes. That's partly the compiler and partly method: a mistake fixed before it was committed never reaches git, while Pan et al. measured raw one-pass translations.
- **"None fits":** semantic paraphrase (yank PF-002, the couch PF-010), a scenario-fitted special case (PF-040), and correcting the source's typo (PF-011). These are candidate new categories.
- **Born before the oracle (2026-07-12): at least 46.8%** (72 of 154 rows with a birth). Median lifetime **at least 80.5 days**. Both are lower bounds, because 99 birth dates are "no later than" dates.
- **Who introduced them**, from the birth commit only: unknown 118, GLM 26, Codex 17, Claude 8, DeepSeek 8 (5 through the Cline harness, which Zach used only for DeepSeek, plus 3 direct), Sol 1. **Who fixed them** is a separate column (Sol 53, DeepSeek 11, ...). Report attribution descriptively; it shows five agents, built on models from four companies (Codex and Sol both run OpenAI models), introducing mistakes, not a ranking.

## What the review checked

- The 100-lead sample: the frame matches the 260 slop-tagged leads, its hash matches, and seed `20261103` draws the same 100 commits.
- The taxonomy against the paper (arXiv 2308.03109v3, Table 4): same codes and names.
- The brief's citation checker and `check.py` both pass on this copy: every commit resolves, and every `c_truth_quote` equals its cited `src/` lines.
- Birth-side authorship, re-derived independently: same answer on every row. (v1 had taken it from the fix commit; v2 corrects that.)
- Nine random rows read for relevance: seven quotes show the claimed behaviour; two are weak (S006 quotes only a function's first line; S058 quotes a log line rather than the rent code).

## Limits

- One coder (DeepSeek) assigned every category; no second coder yet.
- Birth dates and lifetimes are lower bounds; 25 coded rows have no birth.
- The git leads came from search terms used only after 2026-09-28, which shapes the catch-method counts (the `audit` count includes 23 mudlog-inventory rows).
- Only mistakes someone caught are visible, so category shares describe caught mistakes, not all of them.

## Changes from DeepSeek's copy

Corrected 2026-11-02: S001 (yank) was dated to `f00c268c3`, a refactor that only moved the code; its paraphrased messages first appear in `070dfe54e` (2026-04-24). OpenClaw transcripts recovered that day attribute that session to deepseek-chat (PF-098); `author_agent` stays git-only, so the attribution is in the row's notes. Three other births that are refactor commits (G007, G059, S007) are relabelled `lower-bound`. Headline numbers are unchanged.

Two notes naming a returning player by handle now say "Returning-player report" (`slop.tsv` S067, S069, and the matching lines in `build.py`), pending that player's permission. `check.py` reads the `slop.tsv` beside it. Source: `~/dp-research/slop-track/v2/` (`slop.tsv` sha256 `31e895b915b58d8803f3a222c269725b8bbde5fd0ff5e84fed24a8c8a474fe8c` before those edits).

## Files

`slop.tsv` (the table), `slop-notes.md` (DeepSeek's analysis), `taxonomy.md`, `predictions.md` (written before coding), `sample.json` and `frame-260-slop-git-leads.txt`, `check.py` (run from the repository root), `build.py` (DeepSeek's generator; paths point at its workspace).
