# Brief: the slop track

**For:** DeepSeek. **Written by:** Claude, 2026-11-01, approved by Zach. Read [`STUDY.md`](STUDY.md) first.

You are building research evidence, not writing prose. The output is a table that a research paper will be built on, so **every row must cite primary artifacts a reviewer can open and check.** A row without a checkable citation is worse than no row.

## The question

Research question 2 in STUDY.md: **what kinds of mistakes did the AI agents introduce while porting Dark Pawns from C to Go, and how do they compare to the 2024 categories in Pan et al., "Lost in Translation" (ICSE 2024, arXiv 2308.03109)?**

"Slop" here means Go behaviour that differed from the C original and was later fixed: something invented (a message, a command, a rule C never had), something paraphrased (C's exact text reworded), something missing, or something wired wrong. Known examples: the yank handler (PF-002), the couch that should have been a mimic (PF-010), the foreign level scale (PF-055), scenario-fitted special cases (PF-040), 36 invented command names (PF-042).

## Method: work backwards from the fix

Don't read the early history looking for mistakes. Start from each **fix**, which already says what was wrong and usually cites the C source showing what's right. For each one:

1. **The fix:** the commit (and PR, if any) that made Go match C.
2. **The birth:** the commit that introduced the wrong behaviour. Use `git blame` or `git log -L` on the fix's parent for the lines it removed or changed, or `git log -S '<distinctive string>'` for an invented message. Code is often moved or reformatted. Follow it back to the earliest commit where the wrong *behaviour* existed, not just the current lines. If you can't establish it, record the earliest commit you can prove and mark `birth_confidence` as `lower-bound`.
3. **The lifetime:** days from birth to fix (commit author dates).
4. **What caught it:** `oracle-breadth`, `oracle-depth`, `playtest`, `returning-player`, `code-review`, `audit` (a deliberate sweep: R5h, mutation, witness), `other`, or `unknown`. Take this from the fix commit, its PR, the ledger or the field notes; don't infer it.
5. **Who wrote it:** for the birth commit, the agent if the record shows it. Use the branch name of the PR that merged it (for example `glm/…`, `codex/…`, `sol/…`, `deepseek/…`, `claude/…`, `cline/…`) or a `Co-Authored-By` trailer. Otherwise write `unknown`; never guess from style.
6. **The category:** one category from Pan et al.'s taxonomy (see below). If none fits, write `none-fits` and propose a short name for the kind of mistake. Those are candidate new categories, and they matter.
7. **The C truth:** a `src/` file and line showing what C actually does (R5g: cite `src/`, never the oracle checkout). Check the line in-session; never cite from memory.
8. **Player-visible:** `yes` if a player could see the difference (text, colour, a game outcome), `no` if internal only.

## Step 0, before coding anything

1. **The taxonomy.** Open arXiv 2308.03109 and copy Pan et al.'s bug categories into `taxonomy.md` exactly as the paper names them, with each one's group and the section or table number. If you can't open the paper, stop and say so; Zach has it in Zotero and will paste the section.
2. **Predictions.** Write `predictions.md`, dated, before coding any row: which categories you expect to be most common, what share of births you expect before the oracle existed (2026-07-12), the median lifetime you expect, and which catch method you expect to dominate. Don't edit it afterwards.

## The starting list

- **All 101 slop-tagged screening records:** `docs/research/fidelity-study/screening/slop-track-register.jsonl` (43 uncovered bugs, 58 excluded inventions and defects). Each has source leads.
- **A random sample of 100 of the 260 slop-tagged git leads:** `/home/zach/.local/share/openresearch/files/darkpawns/research-scoping/screening-final-v2/git-only-unresolved-source-leads.jsonl`, rows with `"slop-track": "yes"`. Frame: those 260 rows' `commit` values, sorted. Draw with Python `random.Random(20261103).sample(frame, 100)`. Save the frame, its sha256 and the seed.

Some records describe a class rather than one mistake (for example, "127 of 173 mob-command sites were stand-ins"). Code one row per distinct root cause you can establish and say how you split it. Some leads will turn out not to be slop at all (a refactor, a feature, a test-only change): record them with `excluded` and the reason, don't drop them.

**Known bias, to report rather than fix:** the git leads came from a search whose terms were only used after 2026-09-28, so they over-represent late fixes. Working backwards to birth dates corrects part of this, because an early mistake fixed late still gets an early birth. Note it in your report.

## Rules

- **Read-only** in `/home/zach/darkpawns`: don't edit, commit, push, branch, stash or switch branches there. Run `git fetch` and read from `origin/main`. For a scratch checkout, `git worktree add ~/wt/deepseek-slop origin/main`, and remove it when done.
- **Don't run the oracle or start a census.** Existing evidence only.
- **Don't edit** `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- **Don't invoke another agent, model or paid service.** When done, stop and report; Claude reviews it in Zach's session.
- **Never invent** a commit hash, PR number, date, line number or quote. Quotes are word for word, under 25 words. Unknowns go in open questions.
- Players are named by in-game handle only.

## Deliverables

Write to `/home/zach/dp-research/slop-track/`:

1. `taxonomy.md` and `predictions.md` (step 0).
2. `sample.json`: the git-lead frame, its sha256, the seed and the 100 drawn commits.
3. **`slop.tsv`**, one row per mistake, tab-separated, with these columns:
   `id`, `source` (screening record id or git lead commit), `summary` (one sentence), `fix_commit`, `fix_pr`, `fix_date`, `birth_commit`, `birth_date`, `birth_confidence` (`exact` | `lower-bound`), `lifetime_days`, `born_before_oracle` (`yes` | `no`), `caught_by`, `author_agent`, `author_evidence` (branch name or trailer), `pan_category` (exact name, or `none-fits`), `proposed_category` (for `none-fits`), `c_truth` (`src/file.c:line`), `player_visible`, `status` (`coded` | `excluded`), `notes`.
4. **`slop-notes.md`**, a page or two:
   - counts by category, by catch method, by agent and by born-before-oracle;
   - median lifetime overall and by catch method;
   - the `none-fits` mistakes, grouped into proposed new categories with examples;
   - which Pan et al. categories never appear here;
   - how your predictions compared;
   - open questions.

## Check before you report

Every citation must resolve, and every `c_truth` line must exist. Run this and include its output, which must be empty:

```bash
cd /home/zach/darkpawns
tail -n +2 /home/zach/dp-research/slop-track/slop.tsv | awk -F'\t' '{print $4; print $7}' | grep -v '^$' | sort -u | while read -r c; do
  git cat-file -e "$c^{commit}" 2>/dev/null || echo "MISSING COMMIT: $c"
done
tail -n +2 /home/zach/dp-research/slop-track/slop.tsv | awk -F'\t' '$19=="coded"{print $17}' | sort -u | while read -r c; do
  f="${c%%:*}"; n="${c##*:}"; n="${n%%-*}"
  [ -n "$c" ] || { echo "EMPTY C_TRUTH"; continue; }
  git cat-file -e "origin/main:$f" 2>/dev/null || { echo "MISSING FILE: $c"; continue; }
  [ "$(git show "origin/main:$f" | wc -l)" -ge "$n" ] || echo "LINE OUT OF RANGE: $c"
done
```

Then stop and report: the row count (coded and excluded), the three most common categories, the share born before the oracle, the most interesting `none-fits` mistake, and anything you couldn't resolve.
