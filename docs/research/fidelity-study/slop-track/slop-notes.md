# Slop-track notes (v2)

DeepSeek, session 2026-11-01. Companion to `slop.tsv` (201 rows, 23 columns),
`taxonomy.md`, `predictions.md` and `sample.json`. Read `taxonomy.md` for the
Pan et al. category names used below. v1 is kept unchanged beside this file.

**What this counts.** Rows are mistakes the port made that were later fixed:
invented, paraphrased, missing or mis-wired behaviour. Prevalence here covers
*caught* mistakes only — an unknown number of identical mistakes are still live
and uncaught, so the counts are a floor, not a rate.

**Row set.** 201 rows: the 100-commit random sample of the 260 slop-tagged git
leads (`sample.json`, frame sha256
`dbe4d9f0891fcd293faf9960fff9b5267766979604b35e2577252ab2d8d7e17c`, seed
20261103) plus all 101 slop-tagged screening records. 178 rows are `coded`
(a mistake with a fix, a birth and a checked `src/` citation); 23 are `excluded`.

**What v2 changed.**

- **Attribution is now birth-side.** `author_agent` / `author_evidence` come from
  the *birth* commit only: its own `Co-Authored-By` trailer, else the branch of
  the first PR merge that brought the birth into `origin/main`, else `unknown`.
  The old fix-side values are kept in the new `fixer_agent` / `fixer_evidence`
  columns. Attribution is never guessed from style.
- **Every row now carries a `c_truth_quote`**: the exact C line(s) at `c_truth`,
  copied verbatim from `src/` at `origin/main` (multiple lines joined with
  ` ~~~ `, a separator that cannot occur in C source). Every citation was
  re-checked; v1 lines that pointed at a prototype, blank, brace or comment line
  were replaced with the line that carries the behaviour.
- **Two rows moved to `excluded`** because the behaviour has no C counterpart
  (Go-only browser/WebSocket surfaces). Their reason is in `notes`.

## Counts

### By category (Pan et al., 178 coded)

| Category | n |
|---|---:|
| B5 Removal of logic in the source code | 86 |
| B3 Incorrect loop and conditional statements | 40 |
| B4 Inclusion of logic not in the source | 21 |
| C1 Incorrect data type | 12 |
| C3 Output formatting | 11 |
| `none-fits` | 4 |
| C2 Incorrect input parsing | 1 |
| B6 Mismatch of behavior after replacing API call | 1 |
| A3 Mismatch of API behaviors in the source and target | 1 |
| A4 Incorrect use of operators | 1 |

### By catch method (178 coded)

| Method | n |
|---|---:|
| `audit` (a deliberate sweep: R5h, mutation, witness, inventory) | 71 |
| `oracle-depth` | 40 |
| `unknown` | 27 |
| `code-review` | 26 |
| `playtest` | 7 |
| `returning-player` | 4 |
| `oracle-breadth` | 3 |

**The `audit` count is shaped by two things and should not be read as a rate.**
Of its 71 rows, **23 are the DP-1371 mudlog-inventory train** — one deliberate
sweep that walked every C `mudlog()` producer and restored the ones the port
never wired. The other 48 are the colour audit, the special-proc class audits,
the security passes and the mobile-equipment reader audit. Separately, the git
leads were found with search terms that only came into use after 2026-09-28, so
the sample over-represents late fixes and hence late sweeps; the birth dates
correct part of that (see *Limitations*), but not the *method* distribution.

### By author agent (178 coded)

| Agent | birth-side (`author_agent`) | fix-side (`fixer_agent`) |
|---|---:|---:|
| `unknown` | 118 | 76 |
| `glm` | 26 | 17 |
| `codex` | 17 | 14 |
| `claude` | 8 | 7 |
| `cline` | 5 | — |
| `deepseek` | 3 | 11 |
| `sol` | 1 | 53 |

The two columns disagree because they answer different questions. The birth
column is the better proxy for *who introduced* a mistake, but most births are
early squashed phase commits pushed straight to `main`, so 118 rows cannot be
attributed at all. The fix column is far better populated (102 of 178) but says
who *repaired* it, and the repair work was concentrated in one agent's
long-running train. Neither is a productivity measure: tasks were not
randomized, the harness is a confound, and model versions changed.

### By born-before-oracle (154 coded rows with a birth)

`yes` 72 · `no` 82. **Share born before the oracle: 46.8% — a lower bound.**
Birth dates are found by searching the fix's parent for the commit that
introduced the removed line, so a line that was later moved or reformatted
yields a *later* birth than the truth. The real share is therefore **at least**
46.8%, and the true median lifetime is **at least** the value below. 24 coded
rows have no birth at all.

### Player-visible

`yes` 156 · `no` 22 — 88% of caught mistakes could reach a player.

## Lifetimes (birth → fix, days)

- Overall (n=154): **median 80.5**, mean 89.9, min 0, max 194.
- By catch method (median): `returning-player` 183 (n=3) · `code-review` 156
  (n=14) · `oracle-depth` 125 (n=37) · `oracle-breadth` 91 (n=2) · `unknown` 46
  (n=27) · `audit` 42.5 (n=64) · `playtest` 41 (n=7).
- By source: git-lead rows median 45; screening rows median 127.

The shape is what the frame story predicts: the long-lived mistakes are the ones
found by humans and by deep oracle work, while deliberate sweeps and playtests
caught young mistakes. Every lifetime is a lower bound for the same reason as
the born-before-oracle share.

## The `none-fits` mistakes (candidate new categories)

Four coded rows fit no Pan et al. category. Three propose new categories:

1. **semantic paraphrase** (2 rows). C's exact player-facing text reworded into
   something plausible but different, so the *joke* or the contract is lost.
   - PF-002 (yank): the Go handler "differed from C in messages branches
     position gating pronouns and mob handling" (fix 980560edb, PR #477).
   - PF-010 (couch): C's `'the mimic attacks you!'` became `'the couch attacks
     you!'`, removing the joke (`src/spec_procs2.c:282-310`).
   Pan's C3 "Output formatting" is close but is defined as *logic correct,
   formatting different*; here the port also reworded meaning-bearing text, and
   in the yank case changed branches too. A reviewer can check both strings.
2. **scenario-fitted special case** (1 row). PF-040: `do_description`'s
   trailing-space rule was reproduced three times as one-off spacers (a
   command-path autoexit spacer, a galeru-specific look re-render, a hard-coded
   arrival-act space), each commented as observed C "framing", instead of ported
   as a rule. The class is "a special case fitted to a scenario rather than the
   rule", which is an agent-behaviour category, not a language-translation one.
3. **correcting the source's typo** (1 row). PF-011: the port silently fixed
   C's typo `'incapacitated an will slowly die'` to `'and will'`. The port
   improved on the oracle, which R1 forbids; a fidelity port must reproduce the
   typo. Fix 32896d536 restores it (`src/fight.c:1572`).

## Pan et al. categories that never appear

**A1 Violating target language requirements · A2 Duplicating source syntax · B1
Missing library imports · B2 Missing definition · D Model specific constraints ·
E Others** — six of the fifteen categories, including the whole of the paper's
"syntactic" group A bar two, and its model-constraint group D.

**The reason is that this corpus contains only committed mistakes.** A mistake
that never reaches a commit — a translation that fails to build, or one the
agent itself notices and fixes before saving — leaves no artifact to count. Pan
et al. measured *raw translations*, one pass per sample, so every syntactic
slip and every token-window failure was in their corpus by construction. Here
the Go compiler, `go vet` and the test suite run before anything is committed,
and the fix loop is inside the agent's own session: a missing definition or
import, a duplicated C construct, or an over-long prompt is repaired before the
commit exists. Those categories are therefore absent from the table not because
the agents never made them, but because a build-verified, commit-based corpus
cannot see them. What survives into commits is exactly the class a compiler
cannot check: **semantics** — missing logic (B5, 48% of coded rows), wrong
conditions (B3, 22%) and invented logic (B4, 12%).

The corollary matters for the comparison with Pan et al.: their per-category
*shares* and ours are not measuring the same population. Theirs is all
translations; ours is the subset that survived a compiler and a test run and
then needed a later fix.

## How the predictions compared

`predictions.md` was written before any row was coded.

| Prediction | Outcome |
|---|---|
| **P1a** most common = B5; ranking B5 > B4 > C3 > A3 > B3 | **Half right.** B5 first, confirmed, and by a wide margin. But the ranking is wrong: B3 is second (40), not B4 (21); C3 is fifth, not third. |
| **P1b** B4 second | **Failed.** B3 (wrong/missing conditions) is second. |
| **P1c** C3 third, "a large third" | **Failed.** C3 is 11 rows, fifth. Paraphrased text is real but far from the biggest class. |
| **P1d** A1/A2/B1/B2/D near-absent | **Confirmed.** All six never appear (for the corpus reason above). |
| **P2** ~78% born before the oracle | **Failed, and interestingly.** Measured 46.8%, and a lower bound. Most caught mistakes were born *after* the oracle existed — the depth era produced at least as many as the "winging it" era. |
| **P3** median lifetime ~120 days | **Failed.** Median 80.5 (a lower bound). The tail is shorter than expected; a large cluster of mistakes was born and fixed inside the depth era. |
| **P4** `oracle-depth` dominant, then `oracle-breadth` | **Failed.** `audit` (71) dominates, and 23 of those are a single mudlog-inventory sweep; `oracle-breadth` is last (3). |
| **P5** large `unknown` author share | **Confirmed, more strongly than expected.** 118/178 unknown on the birth side. |

The most important failure is **P2**: the "winging it" phase is not where most
*caught* mistakes were born. The frame story's early phases produced the
longest-lived mistakes (median 127 for the screening sample), but the depth era
produced more mistakes in total. Both are consistent with the study's frame —
later checks kept finding new faults — but the intuition that mistakes are
front-loaded is not supported by the birth dates.

## Limitations and open questions

1. **Birth dates are a lower bound for 96 of 154 rows.** The birth finder
   searches the fix's parent for the commit that introduced the removed line
   (`git log -S`). When a line was moved or reformatted after its true
   introduction, the found commit is later than the birth, so lifetimes are
   *under*-estimated and the born-before-oracle share is *under*-stated. Where
   all candidate lines agree the birth is marked `exact` (58 rows); otherwise
   `lower-bound` (96). 24 coded rows have no birth at all. **Treat 46.8% as a
   floor, not a point estimate.**
2. **Birth-side attribution is sparse.** 118 of 178 coded rows have
   `author_agent: unknown`, because most births are early squashed phase commits
   pushed directly to `main` with no PR branch and no trailer. The fix-side
   columns (`fixer_agent`) are better populated (102 of 178) but name the
   repairer, not the introducer. Neither column is a productivity comparison.
3. **Catch method is `unknown` for 27 rows** where the fix commit, its PR, the
   ledger and the field notes do not say how the mistake was noticed. Not
   inferred.
4. **The method distribution is partly an artifact of the sample.** The git
   leads came from search terms only in use after 2026-09-28, so the sample
   over-represents late fixes and therefore late sweeps; and 23 of the 71
   `audit` rows are the single DP-1371 mudlog-inventory train. Do not read
   `audit` as "the method that finds most mistakes".
5. **Two rows are excluded as Go-only surfaces** (`S014` browser/WebSocket
   renderer, `S044` WebSocket entry-name hold): the behaviour exists only in the
   Go port, so no `src/` line can establish it. The reason is in each row's
   `notes`.
6. **Twelve further rows are excluded as "no single `src/` line establishes
   the claim"** (`G062`, `G069`, `G099`, `S002`, `S021`, `S046`, `S051`, `S073`,
   `S074`, `S076`, `S081`, `S096`). Each is plausibly a real port mistake, but
   the evidence is a Go-only concurrency or wiring defect, or world data whose
   truth is in `.mob`/`.wld` files rather than C code. They should be coded in a
   follow-up; they are not evidence against the categories above.
7. **Nine rows are excluded as genuinely not mistakes** (`S005`, `S013`, `S047`,
   `S086`, `S088`, `S092`, `S095`, `S099`, `S100`): a refuted hypothesis, a
   process/regression incident, a locking change, a tooling gap, an unrunnable
   fixture, a certification of C undefined behaviour, a dead-code false alarm,
   and two checking-study process records. Retained per the brief.
8. **Coding is single-coder** (this session) with the fix commit, PR, ledger and
   field notes as sources. No second independent coder, so the category labels
   carry coder variance; the `none-fits` calls are the most judgement-heavy.
9. **Multi-root-cause rows.** Some fixes address a class rather than one
   mistake (for example the composite mount/report/mount-listing fix, G003, and
   the do_kill second-death fix, G074, which also carried an invented `[N]`
   room-list count). These are coded once at the dominant root cause and the
   second root cause is named in `notes`.
10. **`c_truth_quote` is verbatim, with one convention.** Multiple cited lines
    are joined with ` ~~~ `, a separator that cannot occur in C source, so the
    quote can be split unambiguously. Leading whitespace is preserved.

## Verification

`check.py` runs both checks and prints nothing when they pass:

- the brief's checker (every `fix_commit` and `birth_commit` resolves; every
  coded row's `c_truth` names an existing `src/` file and an in-range line);
- the quote check (every coded row has a `c_truth_quote`; each line of the quote
  appears verbatim in the cited file, and the quote equals the cited lines).

## Files

- `taxonomy.md` — Pan et al.'s categories verbatim, with groups and section/table numbers.
- `predictions.md` — dated predictions, frozen before coding.
- `sample.json`, `frame-260-slop-git-leads.txt` — the frame, its sha256, the seed, the 100 drawn commits.
- `slop.tsv` — 201 rows, 23 columns, the deliverable table.
- `check.py` — the brief's checker plus the quote check.
- `build.py` — regenerates `slop.tsv` from the sample, the register and git.
