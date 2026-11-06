# Brief: read-only test audit, `pkg/spells`

You are building a research ledger, not cleaning up code. **Change nothing in the repository.** For every test in `pkg/spells`, decide whether it protects real behavior, and record who wrote it and when. Deleting or fixing tests comes later, in a separate reviewed step that is not yours.

This is the second package audit. The first was `pkg/combat`. Don't read its ledgers, notes or settled results (`docs/research/test-audit/pilot-combat/`, `/home/zach/dp-research/test-audit/`, or the 2026-11-05 combat field note) before you finish: the point is an independent measurement.

The method is adapted from OpenClaw's `test-audit` skill (https://github.com/openclaw/openclaw/tree/main/.agents/skills/test-audit). Read `SKILL.md`, `CAMPAIGN.md` steps 1–3 and `references/audit.md` there before starting. Where this brief and that skill disagree, this brief wins.

## Why

Dark Pawns has about 196,000 lines of tests against about 165,000 lines of Go, and agents wrote most of them. Two questions:

1. Which tests can actually fail when the game breaks, and which only look like proof?
2. For the ones that only look like proof: what kind of junk is it, which agent wrote it, when, and why does it seem to exist?

The answers feed the research "slop track": what did the AI invent when nothing was checking it? Spells is a second package, so the results can be compared with the first.

## Ground truth (checked 2026-11-05 at `origin/main` = `c85ab41be`)

**The package:**

- **Tests:** 30 `_test.go` files (7,126 lines) and 191 top-level `Test`/`Benchmark`/`Fuzz`/`Example` functions, with about 116 `t.Run` calls. Your `inventory.txt` is the authoritative list.
- **Production code:** 9 files (6,062 lines):
  - `act_delivery.go`, `affect_spells.go`, `call_magic.go`;
  - `damage_spells.go`, `saving_throws.go`, `say_spell.go`;
  - `skill_catalog_names.go`, `spell_info.go`, `spells.go`.
- There are no subpackages. `go test ./pkg/spells/...` passes at this commit.

**Dates:**

- The first oracle harness landed 2026-07-12 (`ba05d9906`). Tests added before that date had no C oracle to check against.
- 14 of the 30 test files were added before that date.

**The C:**

- The C is in `src/`, read-only ground truth.
- Spells live mainly in `src/magic.c`, `src/spell_parser.c` and `src/spells.c`. Damage is also in `src/fight.c`, and affects in `src/handler.c`.
- Cite C as `src/<file>:<line>` from your own grep, never from memory.

**Code outside this package:**

- Some spell behavior lives outside `pkg/spells`, for example `pkg/game/spell_mana.go` and `pkg/game/world_bridge.go`.
- Read that code when a test's path goes there, but audit only the tests in `pkg/spells`.

**A lead, not a conclusion.**

- `saving_throws_golden_test.go` was added 2026-06-23 in `71d74913b` ("tier-2: add golden tests for dex_app, saving throws, XP curve, regen"), before the oracle.
- Find out where its expected values came from.
- Read its history with `git log -p`, looking for any later correction. A golden table can be corrected after the oracle arrived, so its current values alone don't tell you who derived them.

## Setup

**Two agents run this brief at the same time:** DeepSeek and MiMo.

- **Your name:** every `<agent>` below is fixed. DeepSeek uses `deepseek`; MiMo uses `mimo`. That name keeps your worktrees and output folder apart from the other agent's.
- **Your paths:** touch only paths that carry your own name.
- **No fetch:** commit `c85ab41be` is already present locally, so don't run `git fetch`.
- **Locks:** if `git worktree add` reports a lock held by the other agent, wait a minute and retry. Never delete a lock file.

```bash
git -C /home/zach/darkpawns worktree add ~/wt/<agent>-spells-audit c85ab41be --detach
cd ~/wt/<agent>-spells-audit
mkdir -p /home/zach/dp-research/test-audit-spells/<agent>
go test ./pkg/spells/... 2>&1 | tail -5     # record the baseline: pass, or each failing test
go test -list . ./pkg/spells/... | grep -E '^(Test|Benchmark|Fuzz|Example)' | sort > /home/zach/dp-research/test-audit-spells/<agent>/inventory.txt
```

Write everything to `/home/zach/dp-research/test-audit-spells/<agent>/`.

**Write your guesses before reading any test.** Put them at the top of `notes.md`:

- how many of the inventory's functions you expect to mark R, F, C and D;
- what share of the F, C and D marks you expect to fall on tests written before the oracle.

Claude's guess, recorded here before the run:

- about 78% R, and F, C and D together 22%;
- at least 85% of the F, C and D marks on tests written before the oracle.

**Another agent is auditing the same package independently.** Don't read anything under `/home/zach/dp-research/test-audit-spells/` except your own directory.

## The marks

Read every test in full: its table rows, the production code it calls, and the C it claims to match. Judge a test by its assertions, not its name.

- **R** keep: names a real behavior and a believable bug that would make it fail.
- **F** keep the behavior, fix the check: for example, an assertion that passes even when the bug is present.
- **C** merge: another test already covers this. Name it.
- **D** delete: name the test that still covers the behavior, or say why there's no real behavior to protect.

Mark a whole table-driven test once, unless its rows need different marks. Then use one row per subtest.

**Look at the fixture, not only the assertion.** A test can't fail when its input sits outside the branch it claims to test. For spells, check for:

- a saving throw that can't fail at the chosen level;
- a victim who already has the affect;
- a caster level that skips the scaling arm;
- a dice roller that ignores the bounds it's asked for.

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

**The keep-rule overrides the patterns.** A test that pins player-facing bytes, a C formula or an RNG draw order is valuable even if it looks static or tedious, *provided its expected values trace to the C*. That's the project's rule R1: player-facing bytes are law.

When J1 and a C-faithful expectation collide, check the C. If the expected value matches `src/`, the test is R, even though Go generated it.

**Duplicate production code.** Some Go functions have a second, separate implementation elsewhere, for example one in `pkg/spells` and one in `pkg/game`. If a test exercises one of these:

- say so in `notes.md`;
- trace the call sites, and say which copy the live game calls.

## Mutation checks (allowed, in a separate scratch worktree)

Prove a mark with a mutation wherever you can:

- every F;
- every D that claims a test **cannot** catch a regression.

To prove one, make one small deliberate change to the production code the test covers, run that test, and record whether it stays green.

Use a second worktree, never the audit one:

```bash
git -C /home/zach/darkpawns worktree add ~/wt/<agent>-spells-audit-mut c85ab41be --detach
# edit one line, run: go test ./pkg/spells/ -run '^TestName$'
git -C ~/wt/<agent>-spells-audit-mut checkout -- .      # restore byte for byte after each mutation
```

Record each mutation in `mutations.tsv`.

When you're done, remove your own two worktrees and no others:

```bash
git -C /home/zach/darkpawns worktree remove ~/wt/<agent>-spells-audit
git -C /home/zach/darkpawns worktree remove ~/wt/<agent>-spells-audit-mut
```

## Attribution (git only)

For each test function, find the commit that introduced it:

```bash
git log -S'func TestName(' --format=%h --reverse -- pkg/spells | head -1
```

Then find the agent, trying each of these in order:

1. **The PR and branch:**
   ```bash
   gh api repos/zax0rz/darkpawns/commits/<sha>/pulls --jq '.[] | "\(.number) \(.head.ref)"'
   ```
   If that prints nothing, use the first-parent merge that brought the commit in:
   ```bash
   git log c85ab41be --first-parent --merges --ancestry-path <sha>..c85ab41be --format='%h %s' | tail -1
   ```
2. **The branch prefix:**

   | Prefix | Agent |
   |---|---|
   | `glm/` | glm |
   | `codex/`, `luna/` | codex |
   | `sol/` | sol |
   | `deepseek/`, `cline/` | deepseek |
   | `mimo/` | mimo |
   | `kimi/` | kimi |
   | `claude/` | claude |

3. **A `Co-Authored-By` trailer** on the commit.
4. **Otherwise `unknown`.** Put the git author name, and anything else you saw, in `author_evidence`.

Never guess the agent from writing style.

## Deliverables

### `ledger.tsv`

One row per test function, or per subtest where the marks differ. It's tab-separated, with this header (the same 18 columns as the combat audit, so the two packages can be compared):

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

### `mutations.tsv`

One row per mutation, with these columns: `test	file	line	mutation	result(green|red)	restored(yes)`.

### `notes.md`

A page or two, covering:

- your guesses from before you started, against the actual counts;
- the baseline `go test` result;
- counts by mark, by junk code, by `author_agent`, and before versus after the oracle;
- the three clearest junk examples, with file:line, and **why each seems to exist**: padding coverage, proving a guess, a test written to match buggy code? Cite the PR or commit message as evidence;
- the saving-throw golden table: where its values came from, and when;
- every test whose expected value disagrees with the C. Each one is a possible game bug, so list them separately, with the C line and the Go line;
- every duplicate production implementation you found, and which copy is live;
- open questions.

## Hard rules

- **Read-only.** Don't edit, commit, push, branch or stash in `/home/zach/darkpawns`, and never switch its branch: other agents work there. Your two worktrees are detached and temporary.
- Never run `git stash` anywhere.
- Don't start a census or run the oracle. Unit tests only.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Don't invoke another agent, model or paid service (`claude`, `codex`, other CLIs or APIs).
- Don't open the private transcript archives. Attribution comes from git only.
- Never invent a commit hash, PR number, date, C line or quote. If something is probably true but you can't cite it, put it in open questions.
- There's no time limit. Finish the whole inventory, because a partial ledger can't be compared.

## Check before you report

Every inventory test must be in the ledger, every mark must be valid, and every C or D mark must name its remaining proof. This must print nothing:

```bash
cd /home/zach/dp-research/test-audit-spells/<agent>
cut -f4 ledger.tsv | tail -n +2 | sed 's#/.*##' | sort -u > ledger-tests.txt
comm -23 inventory.txt ledger-tests.txt | sed 's/^/MISSING FROM LEDGER: /'
awk -F'\t' 'NR>1 && $5 !~ /^[RFCD]$/ {print "BAD MARK: " $1}
            NR>1 && $5 ~ /^[CD]$/ && $10 == "" {print "NO REMAINING PROOF: " $1}
            NR>1 && NF != 18 {print "WRONG COLUMN COUNT: " $1 " (" NF ")"}' ledger.tsv
```

Then stop and report:

- row counts by mark;
- your guesses against the results;
- the three clearest junk examples;
- every test that disagrees with the C;
- every duplicate implementation;
- what you couldn't resolve.

Claude reviews the ledger in Zach's session.
