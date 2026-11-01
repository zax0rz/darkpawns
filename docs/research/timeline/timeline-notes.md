# Timeline notes — a dated, cited account of the Dark Pawns port

*Produced 2026-10-31 by DeepSeek from [`BRIEF.md`](BRIEF.md), read-only. Reviewed by Claude the same day: the citation check re-run (empty), the three headline findings and both August session logs checked, and every quote matched word for word against its source. Six quotes were corrected: four were paraphrases presented as quotes, two had a hyphen for an em dash, and one also cited the PR instead of the commit holding the words.*

Companion to `timeline.tsv` (62 event rows, 1 header). Every row cites a primary
artifact; the check script in the brief returns empty. Dates are commit author
dates unless a merge date or a document's own date matters more, which the row's
notes say.

## The stages as the evidence draws them

The six stages in the brief survive, but two of them are not clean phases and one
is not what the brief describes. Start and end dates are the first and last dated
artifact in each stage.

| Stage | Start | End | What it is |
|---|---|---|---|
| **S1 winging-it** | 2026-04-17 | 2026-07-11 | Port by hand, "complete" by assertion; no oracle. |
| **S2 oracle-breadth** | 2026-07-12 | 2026-08-22 | The C server runs beside Go; does each command match once? |
| **S3 oracle-depth** | 2026-08-23 | 2026-10-15 | Per-command/per-site proof, then the C-defined surface inventory. |
| **S4 playtesting** | 2026-09-23 | 2026-10-25 | Humans and a closed-loop play lab find what scenarios never drove. |
| **S5 returning-players** | 2026-10-20 | (open) | Public release, then 1990s players report bugs. |
| **S6 oracle-wrong** | 2026-09-18 | 2026-10-30 | The reference itself (build, platform, memory, normalizer) was wrong. |

**S1 (2026-04-17 → 2026-07-11).** 5,162 commits total in the repo; this stage is
the first ~1,200 of them. The work is organised in "waves" and phases, each marked
complete in the same file that describes it (rows 1–4). Nothing runs against the
original server. The only detectors are static review (a "Fable" review, a
"clawpatch" crawler) and a hand-written e2e smoke test.

**S2 (2026-07-12 → 2026-08-22).** The C server is built and the first differential
harness lands (row 10). Breadth means "does every registered command have at least
one live C-vs-Go probe" (`docs/fidelity/DEPTH_TESTING.md`). The month is mostly
infrastructure and rules: reachability mechanised (row 15), the rulebook seeded
(row 14), north-star fixed (row 12). Breadth itself is declared complete on
2026-08-23 (row 22), which is why S3 starts there.

**S3 (2026-08-23 → 2026-10-15).** Depth proof: D1–D5 per command, then a
per-site map of C's output surface. It contains the largest "declared done"
numbers in the record (855/861 on 2026-08-28, row 25) and, in the same month, the
measurement that reframes them: by C-defined surface, 82% of units were still
blocked on 2026-10-15 (row 53). Two long autonomous agent runs sit inside it
(rows 27–28, 54).

**S4 (2026-09-23 → 2026-10-25).** Human play (Mudlet, then a 30-minute session)
and DeepSeek's closed-loop play lab. This stage begins *inside* S3 and never
replaces it; its findings are Go-only or on paths no scenario drove (rows 34–38,
45–47, 58).

**S5 (2026-10-20 → open).** The first tagged release, then former players. The
site was public from 2026-08-15 (row 21), but the evidence for *returning players
finding bugs* is dated 2026-10-24/25 (row 56). Named by in-game handle only.

**S6 (2026-09-18 → 2026-10-30).** The reference oracle is shown to be wrong four
ways: its own bytes vary run to run (row 32), it was built `-O2` (row 40), it
segfaults on 64-bit (row 51), and the comparison normalizer erased byte classes
(rows 19, 49, 60). This stage is interleaved with S2–S5, not after them.

## Declared done → disproved, as pairs

| # | Declared done | What disproved it |
|---|---|---|
| 1 | 2026-04-18 "Phase 2 complete: death/respawn + faithful formula port" (row 2) | 2026-06-17 the mob-AI self-deadlock: "the game doesn't work" (row 6) |
| 2 | 2026-04-24/25 PORT-PLAN waves 1–9.5 "COMPLETED", goal "100% faithful" (row 3) | 2026-07-03 "verified dead code" (row 7); 2026-07-12 the first oracle run (row 10) |
| 3 | 2026-05-27 "Fidelity Audit Complete", 13 of 14 fixed (row 5) | 2026-06-17 the deadlock (row 6); 2026-07-22 reachability finds 61 unreachable commands (row 15) |
| 4 | 2026-07-03 "The game is playable." (row 8) | 2026-07-12 the first oracle run finds three divergences on one `look` (row 10) |
| 5 | 2026-07-06 README "the Go port is complete" (row 9) | 2026-07-12 oracle; 2026-07-16 character creation was a re-skin, DP-1173 (row 13); 2026-08-28 14 missing + 10 unwired commands (row 26) |
| 6 | 2026-08-23 registered-command breadth complete, PR #598 (row 22) | 2026-08-23 the flee depth pilot, same day (row 24) |
| 7 | 2026-08-28 855/861 modeled cases (99.3%) (row 25) | 2026-08-28 the same note's 14 missing / 10 unwired (row 26); 2026-10-15 82% of C surface units blocked (row 53) |
| 8 | 2026-09-21 CONTRIBUTING "The port itself is complete" (row 33) | 2026-09-23 Mudlet playtest, four bugs (row 34); 2026-10-04 playtest, five bugs (row 46) |
| 9 | 2026-09-30 first fully clean claims census (row 43) | 2026-10-03 Phase 4 (row 45); 2026-10-04 playtest (row 46); 2026-10-14 draw parity, 61 coincidence greens (row 52) |
| 10 | 2026-10-20 v0.1.0 released on census-clean main (row 55) | 2026-10-24/25 returning players report five bugs (row 56); 2026-10-30 combat color was never compared (row 60) |

The pattern the brief asked to test holds for pairs 1–7 and 10. Pair 9 is weaker
(see below).

## Evidence against the pattern

- **The strongest claim is weaker than it looks.** The brief says the port "was
  called complete". The only whole-port declarations in the repo are a README line
  (2026-07-06) and a CONTRIBUTING line (2026-09-21). Every later declaration is
  explicitly partial — "what remains is depth, tooling, and polish" (row 33) — and
  the depth stage's own guide says the opposite of done: *"Do not infer completion
  from a breadth scenario"* (row 23). From 2026-07-16 the project's own law
  (north-star, R1) makes "done" mean byte-fidelity, so S2/S3 never declared the
  port done, only their own denominator.
- **A stage that did not declare done.** S6 never claims a stage is finished; it
  attacks the measuring instrument. Its rows are `incident`, not `declared-done`.
- **The stages overlap, they do not follow.** S6 starts 2026-09-18, five weeks
  before S3's last row and before S5 exists. S4 starts 2026-09-23, inside S3. S5's
  "returning players" date (2026-10-24) is a month after the public site
  (2026-08-15). A linear "each stage disproves the last" reading is not supported.
- **Findings that fit no stage.** The C backdoor (row 44) was found by the owner
  suspecting backdoors, not by the oracle or a playtest — the fidelity process had
  proved it green. DP-493 "Done" with no code (row 48) was found by an exhaustive
  inventory sweep, a third detector. Neither is "the current check said done".
- **A disproof from a different failure class.** The 2026-09-30 clean claims
  census was not broken by a fidelity failure; its first counterexamples were
  Go-only defects and review findings the census was never designed to see
  (rows 45–47). That is PF-053's point, not a false green.
- **Some greens could not have failed.** R5h and the coincidence greens (DP-1212;
  PR #1682) mean a number of earlier greens were never evidence either way. For
  those, the "done" claims were unsupported rather than disproved — a different
  thing, and worth separating in the paper.


## The Aug 28 / Aug 29 date question

The brief pairs the 2026-08-28 milestone note with a session log named
2026-08-29, and asks which dates are right. They are two different runs.

- The milestone note's date is right for its content: it reports PRs #687–#731 at
  `c74789ef9`, and PR #731 merged 2026-08-28 17:24 EDT, deployed the same day
  (row 25).
- The log the brief names, `rollout-2026-08-29T06-03-35-…jsonl.zst`, starts
  2026-08-29 10:03:44 UTC and ends 2026-09-03 14:47:53 UTC — 5 days 4 h 44 m. Its
  filename uses **local** time (06:03 EDT = 10:03 UTC). It references PRs
  #735–#1311 and never mentions #687–#731 (row 28).
- The session that produced the milestone note is a *different* log,
  `rollout-2026-08-28T01-47-04-…jsonl.zst`, running 05:47–21:25 UTC on 2026-08-28;
  it references #687+ and the literal string "855/861" (row 27).

So the milestone's date (Aug 28) is correct and the Aug 29 log is a later,
separate session. The "more than 5 days" run is the Aug 29–Sep 3 session; the
milestone belongs to the ~15.6-hour Aug 28 session. The brief's anchor row merges
them.

## Open questions (believed but not cited)

- **Was there an earlier "done" I can't see?** The strongest whole-port claims in
  the repo are the 2026-07-06 README and the 2026-09-21 CONTRIBUTING lines. A
  Linear issue, Discord post or private brief might claim completion more loudly,
  and none of those is in the repo.
- **PF-001's three divergences.** The ledger still marks PF-001
  `needs-verification`; I read the RESEARCH-LOG's statement of the three
  divergences (row 10), not PR #243's own scenario output, which I did not open.
- **The 2026-04-25 line counts.** PORT-PLAN's "61,690 lines of Go" and "~21,000
  lines genuinely unported" are that commit's self-audit; I did not recompute them.
- **First external player login.** The site post is 2026-08-15 and the
  returning-player reports are 2026-10-24/25, but I could not date the first
  player login between them, so S5's start is the release, not the first player.
- **PF-091's "before" row.** The incident note itself says no retained green
  census against the `-O2` binary at the exact pre-#1682 commit has been found; I
  recorded the 21-red comparison, which is retained, but the baseline green is
  history rather than a fact.
- **Two citations are workstation-local.** Rows 27, 28 and 43 cite absolute paths
  under `/home/zach/.codex` and `/home/zach/Archives`. They resolve on this host
  (the brief's check passes) but a reviewer elsewhere cannot open them; the repo
  copies are the field notes `2026-10-16-twenty-hours-of-luna.md` and
  `2026-10-20-first-public-release.md`.
- **Stage labels are mine.** A reviewer may prefer to split S3 into "authored
  cases" and "C-defined surface", or to merge S4 into S3, since the evidence for
  both is a single chain of 2026-09-23 → 2026-10-15 work.


