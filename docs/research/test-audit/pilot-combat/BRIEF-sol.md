# Brief: read-only test audit, pilot on `pkg/combat` (Sol)

You are building a research ledger, not cleaning up code. **Change nothing in the repository.** For every test in `pkg/combat`, decide whether it protects real behavior, and record who wrote it and when. Deleting tests comes later, in a separate reviewed step that is not yours.

The method is adapted from OpenClaw's `test-audit` skill (https://github.com/openclaw/openclaw/tree/main/.agents/skills/test-audit). Read `SKILL.md`, `CAMPAIGN.md` steps 1–3 and `references/audit.md` there before starting. Where this brief and that skill disagree, this brief wins.

## Why

Dark Pawns has about 1,270 test files and 193,000 lines of tests, against about 114,000 lines of game code. Agents wrote most of them. Two questions:

1. Which tests can actually fail when the game breaks, and which only look like proof?
2. For the ones that only look like proof: what kind of junk is it, which agent wrote it, when, and why does it seem to exist?

The second question feeds the research "slop track": what did the AI invent when nothing was checking it?

## Ground truth (checked 2026-11-04 at `origin/main` = `b03629cc8`)

- `pkg/combat` has 38 `_test.go` files (7,152 lines), about 210 top-level `Test`/`Benchmark`/`Fuzz`/`Example` functions and 32 `t.Run` subtests (your `inventory.txt` is the authoritative list), against 10 production files (4,606 lines).
- Production files: `body.go`, `callbacks.go`, `combatant.go`, `damage_before_message.go`, `engine.go`, `fight_core.go`, `fight_messages.go`, `formulas.go`, `roller.go`, `skill_messages.go`. Golden data: `pkg/combat/testdata/combat_transcript.golden`.
- The original C is in `src/` (read-only ground truth). Combat lives mainly in `src/fight.c`. Cite C as `src/<file>:<line>` from your own grep, never from memory.
- The first oracle differential harness landed 2026-07-12 (`ba05d9906`). Tests added before that date had no C oracle to check against.
- A lead, not a conclusion: `ac_reduction_golden_test.go` was added 2026-07-05 in `51f3fbefe`, titled "combat & skill golden tests — C→Go fidelity…". It predates the oracle. Find out where its expected values came from.

## Setup

Your name for every `<agent>` below is `sol`. Commit `b03629cc8` is already present locally, so don't run `git fetch`. If `git worktree add` reports a lock held by another agent, wait a minute and retry. Never delete a lock file.

```bash
git -C /home/zach/darkpawns worktree add ~/wt/<agent>-test-audit b03629cc8 --detach
cd ~/wt/<agent>-test-audit
mkdir -p /home/zach/dp-research/test-audit/<agent>
go test ./pkg/combat/... 2>&1 | tail -5     # record the baseline: pass, or each failing test
go test -list . ./pkg/combat/... | grep -E '^(Test|Benchmark|Fuzz|Example)' | sort > /home/zach/dp-research/test-audit/<agent>/inventory.txt
```

Write everything to `/home/zach/dp-research/test-audit/sol/`.

**Before reading any test**, write your guesses at the top of `notes.md`: how many of the inventory's functions you expect to mark R, F, C and D. Claude's guess, recorded here before the run: about 75% R, with F, C and D together making up 25%.

## Stay blind

Other models have already audited this package with this brief. The experiment measures whether independent audits agree, so **your ledger is only useful if you never see theirs**. A blind miss is data; a peek is not.

Off limits, in any form (reading, `git show`, `grep`, `find`, `ls`, search):

- anything under `/home/zach/dp-research/test-audit/` except `sol/`;
- `docs/research/test-audit/` in any commit or branch;
- research field notes dated 2026-11-05 or later, and `EVIDENCE_LEDGER.tsv` rows `PF-103` onward;
- any file outside your two worktrees and your output folder that mentions this audit;
- Linear, GitHub issues and pull-request comments, except the `gh api .../commits/<sha>/pulls` attribution lookup below.

Work from commit `b03629cc8`, which predates the other audits. Use `origin/main` only in the attribution commands below, which read commit metadata and not file contents.

**Network:** read the OpenClaw skill pages linked above and make the `gh api` attribution calls against `zax0rz/darkpawns`. Nothing else: no web search, no Hugging Face or other model hubs, no datasets, no other models or APIs.

If you see prior audit output anyway, even by accident, **stop reading it, and say so in `notes.md` under "Contamination"**, with what you saw and when. An honest report keeps the ledger usable; a hidden one ruins it. If you saw nothing, write "Contamination: none".

## The marks

Read every test in full, including its table rows, plus the production code it calls and the C it claims to match. Judge a test by its assertions, not its name.

- **R** keep: names a real behavior and a believable bug that would make it fail.
- **F** keep the behavior, fix the check: for example, an assertion that passes even when the bug is present.
- **C** merge: another test already covers this. Name it.
- **D** delete: name the test that still covers the behavior, or say why there's no real behavior to protect.

Mark a whole table-driven test once, unless its rows need different marks. Then use one row per subtest.

## Junk patterns (use these codes)

| Code | Pattern |
|---|---|
| J1 | Expected value produced by the code under test (the test asks Go for the answer, then checks Go gives it). Includes golden files generated from Go output. |
| J2 | No assertion, or one that cannot fail (self-comparison, `x == x`, checking a value it just set). |
| J3 | Name promises behavior the inputs or assertions never exercise. |
| J4 | Duplicate of another test's contract at the same layer. |
| J5 | Tests implementation shape (which helper was called, in what order, the struct layout) rather than what the player sees or the game state. |
| J6 | Reads or greps Go source text. |
| J7 | Test-only seam: an export, hook or flag in production code that only tests use. Name the seam. |
| J8 | Mock or fixture supplies the behavior being asserted. |
| J9 | Negative check that passes for the wrong reason (an unrelated guard, or an unreachable branch). |
| J10 | Expected bytes or numbers that disagree with the C, or claim C fidelity with no C citation and no oracle transcript behind them. |

**Keep-rule (overrides the patterns).** A test that pins player-facing bytes, a C formula or RNG draw order is valuable even if it looks static or tedious, *provided its expected values trace to the C*. That's the project's rule R1: player-facing bytes are law. When J1 and a C-faithful expectation collide, check the C: if the expected value matches `src/`, the test is R even though Go generated it.

## Mutation checks (allowed, in a separate scratch worktree)

For every F, and every D that claims a test **cannot** catch a regression, prove it where you can. Make one small deliberate change to the production code the test covers, run that test, and record whether it stays green. Use a second worktree, never the audit one:

```bash
git -C /home/zach/darkpawns worktree add ~/wt/<agent>-test-audit-mut b03629cc8 --detach
# edit one line, run: go test ./pkg/combat/ -run '^TestName$'
git -C ~/wt/<agent>-test-audit-mut checkout -- .      # restore byte for byte after each mutation
```

Record each mutation in `mutations.tsv`. When done, remove your own two worktrees and no others: `git -C /home/zach/darkpawns worktree remove ~/wt/<agent>-test-audit` and the same for `~/wt/<agent>-test-audit-mut`.

## Attribution (git only)

For each test function, find the commit that introduced it: `git log -S'func TestName(' --format=%h --reverse -- pkg/combat | head -1`. Then find the agent:

1. PR and branch: `gh api repos/zax0rz/darkpawns/commits/<sha>/pulls --jq '.[] | "\(.number) \(.head.ref)"'`. If empty, use the first-parent merge that brought it in: `git log origin/main --first-parent --merges --ancestry-path <sha>..origin/main --format='%h %s' | tail -1`.
2. Branch prefix → agent: `glm/` = glm, `codex/` and `luna/` = codex, `sol/` = sol, `deepseek/` and `cline/` = deepseek, `claude/` = claude.
3. Otherwise a `Co-Authored-By` trailer on the commit.
4. Otherwise `unknown`. Put the git author name and anything else you saw in `author_evidence`. Never guess the agent from writing style.

## Deliverables

**`ledger.tsv`**, one row per test function (or per subtest where the marks differ), tab-separated, with this header:

`id	file	line	test	mark	contract	junk	expected_source	evidence	remaining_proof	introduced_commit	introduced_date	pr	branch	author_agent	author_evidence	born_before_oracle	confidence`

- `id`: `T001`, `T002`, …
- `test`: `TestName`, or `TestName/subtest`.
- `mark`: `R` | `F` | `C` | `D`.
- `contract`: one sentence on what behavior it protects, or `none`.
- `junk`: codes separated by `;`, or blank.
- `expected_source`: `c-cited` (give `src/file:line` in evidence) | `oracle` | `go-generated` | `hand-written` | `none`.
- `evidence`: one line, the reason for the mark.
- `remaining_proof`: required for C and D. Name the test, or write `none needed:` and a reason.
- `born_before_oracle`: `yes` if `introduced_date` is before 2026-07-12.
- `confidence`: `high` | `medium` | `low`.

**`mutations.tsv`**: `test	file	line	mutation	result(green|red)	restored(yes)`.

**`notes.md`** (a page or two):

- your guesses from before you started, against the actual counts;
- the baseline `go test` result;
- counts by mark, by junk code, by `author_agent`, and before versus after the oracle;
- the three clearest junk examples, with file:line, and **why each seems to exist** (padding coverage? proving a guess? a test written to match buggy code?), citing the PR or commit message as evidence;
- any test whose expected value disagrees with the C: that's a possible game bug, so list it separately;
- open questions.

## Hard rules

- **Read-only.** Don't edit, commit, push, branch or stash in `/home/zach/darkpawns`, and never switch its branch: other agents work there. Your two worktrees are detached and temporary.
- Never run `git stash` anywhere.
- Don't start a census or run the oracle. Unit tests only.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Don't invoke another agent, model or paid service (`claude`, `codex`, other CLIs or APIs).
- Don't open the private transcript archives. Attribution is from git only.
- Never invent a commit hash, PR number, date, C line or quote. If something is probably true but you can't cite it, put it in open questions.

## Check before you report

Every inventory test must be in the ledger, every mark must be valid, and every C or D must name its remaining proof. This must print nothing:

```bash
cd /home/zach/dp-research/test-audit/<agent>
cut -f4 ledger.tsv | tail -n +2 | sed 's#/.*##' | sort -u > ledger-tests.txt
comm -23 inventory.txt ledger-tests.txt | sed 's/^/MISSING FROM LEDGER: /'
awk -F'\t' 'NR>1 && $5 !~ /^[RFCD]$/ {print "BAD MARK: " $1}
            NR>1 && $5 ~ /^[CD]$/ && $10 == "" {print "NO REMAINING PROOF: " $1}
            NR>1 && NF != 18 {print "WRONG COLUMN COUNT: " $1 " (" NF ")"}' ledger.tsv
```

Then stop and report: your contamination statement, row counts by mark, your guess against the result, the three clearest junk examples, any test that disagrees with the C, and what you couldn't resolve. Claude reviews the ledger in Zach's session.
