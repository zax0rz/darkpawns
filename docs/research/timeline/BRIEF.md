# Brief: a dated, cited timeline of the Dark Pawns port

You are building research evidence, not writing prose. The output is a timeline table that a research paper will later be built on, so **every row must cite a primary artifact that a reviewer can open and check**. A row without a checkable citation is worse than no row.

## The story the timeline has to test

Zach's account of the port, which the timeline should confirm, correct or complicate:

1. **Winging it.** "Let's port Dark Pawns." No oracle. It reached a point where the game ran and resembled Dark Pawns, and was called complete.
2. **The oracle, breadth.** The original C server started running side by side with the Go port and the output was compared. Breadth: does each command match at least once?
3. **The oracle, depth.** Long agent runs (Luna) proved command behaviour site by site, not just once per command.
4. **Playtesting.** Humans playing found bugs the green census missed.
5. **Returning players.** After the public launch, former players of the 1990s game found more bugs the census missed.
6. **The oracle itself was wrong.** A wrongly built C binary, a C memory bug, a 64-bit crash, and a comparison configured to ignore color.

The pattern being tested: **at each stage, the current way of checking said "done", and the next one proved it wasn't.** Find the evidence for that pattern, and also any evidence *against* it (a stage that didn't declare done, a finding that doesn't fit, stages that overlap rather than follow one another). Disconfirming evidence is just as valuable. Don't bend the facts to fit the story.

## Where to look (all read-only)

- **Git history** of `/home/zach/darkpawns`: use `git log origin/main` and `git show`. Run `git fetch` first.
- **PRs:** `gh pr view <n> --repo zax0rz/darkpawns`, `gh pr list --state merged --search ...`.
- **Field notes:** `docs/research/field-notes/*.md`. They're dated and cite their own artifacts.
- **Evidence ledger:** `docs/research/EVIDENCE_LEDGER.tsv`. Use rows marked `verified` freely; treat `needs-verification` rows as leads to check.
- **The fidelity rulebook's amendment log:** the table at the end of `docs/fidelity/RULEBOOK.md`. Each rule was added in response to a dated incident.
- **Fidelity depth docs:** `docs/fidelity/DEPTH_TESTING.md` and `docs/fidelity/depth/handoff/*.md`.
- **Oracle run archives:** `~/Archives/darkpawns/oracle-runs/YYYY-MM-DD/*/MANIFEST.md` and `summary.txt`.
- **Briefs:** `docs/briefs/` (dated filenames).

## Anchors already checked (verify them again anyway)

| Date | Event | Citation |
|---|---|---|
| 2026-04-17 | First commit: world-file parsers and scaffold | `3e11aca83` |
| 2026-07-12 | First oracle differential harness | `ba05d9906` (PR #242 area); ledger PF-001 says the first live oracle scenario found three divergences (PR #243) |
| 2026-08-28 | Depth milestone: 855 of 861 *modeled* cases proven (99.3%) | `docs/research/field-notes/2026-08-28-fidelity-depth-milestone.md` |
| late Aug to 2026-09-04 | Luna's run of more than 5 days, breadth to depth | Session log `~/.codex/sessions/2026/08/29/rollout-2026-08-29T06-03-35-*.jsonl.zst`; ledger PF-075. **The milestone note above is dated Aug 28 but the log file is named Aug 29: work out which dates are correct and say how.** |
| 2026-09-18 | Accuse's corrupted C output declared "run-varying" with a new EXPECTED_UNSTABLE verdict class | `d7e816b03` |
| 2026-09-23 | Mudlet playtest; harness parity framing | `docs/research/field-notes/2026-09-23-mudlet-playtest-and-harness-parity.md` |
| 2026-09-28 | The C reference turns out to be an `-O2` build; 21 scenarios go red with Go held fixed | PR #1682 (`7b59e3faa`); ledger PF-091 |
| 2026-10-04 | 30-minute playtest finds 5 bugs a census of ~3,200 scenario/seed pairs missed | `docs/research/field-notes/2026-10-04-playtest-and-the-foreign-level-scale.md` |
| 2026-10-11 | Accuse root cause found: C social-loader overflow | `docs/research/field-notes/2026-10-11-accuse-oracle-heap-overflow.md` |
| 2026-10-13 | `load mob` crash: the 64-bit oracle segfaults | `docs/research/field-notes/2026-10-13-load-mob-32-vs-64-bit.md`; ledger PF-066 |
| 2026-10-15 to 10-16 | Luna's 26-hour surface-mapping run | `docs/research/field-notes/2026-10-16-twenty-hours-of-luna.md` |
| 2026-10-23/24 | Public launch | `docs/research/field-notes/2026-10-20-first-public-release.md` and release tags |
| 2026-10-24 to 10-25 | Returning players report 5 bugs the census missed; 127 of 173 mob-command sites turn out to be stand-ins | `docs/research/field-notes/2026-10-25-players-find-the-stand-ins.md` |
| 2026-10-30 | ANSI audit: combat color never compared; rule R1b | PR #2231 (`ee12a71c2`), `eb6c70c2a` |

Stage 1 has the thinnest record: the discipline came later. Expect to rebuild April to July from commit messages, early briefs and early PRs. Look especially for **the moment the port was first called complete or working**, in the words used at the time.

## Decisions already made

- **"Declared done"** means any artifact claiming completion or a percentage complete: commit messages, PR titles, field notes, handoff docs, release notes. Quote the exact words, in under 25 words.
- **"Disproved"** means the first dated artifact showing that the claim missed something real and player-visible. A refactor doesn't count, and neither does a new feature.
- **Use the six stages above as a starting point.** If the evidence says a stage should be split, merged or renamed, do it and explain why in the notes.
- **Dates:** commit author date (`%ad`, short form) unless a merge date or a document's own date matters more; say which.
- Players are named only by in-game handle, never by real name. Sai is Zach.

## Hard rules

- **Read-only.** Don't edit, commit, push, branch or stash in `/home/zach/darkpawns`, and don't switch its branch: other agents work in that checkout. If you need a scratch checkout, `git worktree add ~/wt/deepseek-timeline origin/main` and remove it when done.
- **Don't start a census or run the oracle.** The timeline uses existing evidence only.
- **Don't edit** `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- **Don't invoke another agent, model or paid service** (`claude`, `codex`, other CLIs or APIs). When you're done, stop and report; Claude reviews the result in Zach's session.
- **The August session log is large.** Decompress it with `zstd -dc ... | grep` for specific dates, PR numbers and phrases. Don't read it whole.
- **Never invent a commit hash, PR number, date or quote.** If something is probably true but you can't cite it, put it in the open-questions list.

## Deliverables

Write both files to `/home/zach/dp-research/timeline/`:

1. **`timeline.tsv`**, one row per event, with these tab-separated columns:
   `date`, `stage`, `event_type` (`declared-done` | `disproved` | `stage-start` | `rule-added` | `incident` | `other`), `summary` (one sentence), `quote` (exact words, under 25, or blank), `citation` (commit hash, `PR #n`, a repo-relative file path, or an absolute path such as `/home/zach/Archives/...` for files outside the repo, never `~`; add line numbers after a colon; separate several citations with `;`), `confidence` (`verified` = you opened the artifact and it says this; `inferred` = follows from artifacts but none states it outright), `notes`.
2. **`timeline-notes.md`**, a page or two:
   - the stages as the evidence draws them, each with start and end dates;
   - each "declared done" moment and what disproved it, as pairs;
   - evidence against the pattern;
   - open questions: things you believe but couldn't cite.

## Check before you report

Every `citation` must resolve. Run this and include its output, which must be empty, in your report:

```bash
cd /home/zach/darkpawns
tail -n +2 /home/zach/dp-research/timeline/timeline.tsv | cut -f6 | tr ';' '\n' | sed 's/^ *//' | while read -r c; do
  case "$c" in
    "PR #"*) gh pr view "${c#PR #}" --repo zax0rz/darkpawns --json number >/dev/null 2>&1 || echo "MISSING PR: $c" ;;
    [0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]*) git cat-file -e "$c^{commit}" 2>/dev/null || echo "MISSING COMMIT: $c" ;;
    "") echo "EMPTY CITATION" ;;
    *) f="${c%%:*}"; git cat-file -e "origin/main:$f" 2>/dev/null || [ -e "$f" ] || echo "MISSING FILE: $c" ;;
  esac
done
```

Then stop and report: the row count, the stages you ended up with, the three most surprising findings, and anything you couldn't resolve.
