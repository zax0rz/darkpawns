# Brief: idiomatic-Go experiment, round two

This is a long, unattended run with **no time limit**. Zach isn't steering it.
Claude reviews the branch in Zach's session when you report.

## Read first

- `AGENTS.md`
- `docs/fidelity/RULEBOOK.md`
- [`PREDICTIONS.md`](PREDICTIONS.md) in this directory: what round one showed
  and what changes now. Don't edit it.
- Round one's [`BRIEF.md`](../BRIEF.md) and its `run/` files. **Every rule in
  round one's brief still applies** unless this brief changes it, in particular:
  - the forbidden list;
  - the protected paths;
  - the hard rules;
  - one target per commit;
  - log bugs instead of fixing them.

Round one did careful, correct work. Its blueprint skipped most targets as
"unsure", and that was because of the brief, not because of your judgment. Read
the clarified RNG rule below before you decide what is unsafe.

## Setup

1. **Clean up round one's leftovers.** Remove your round-one worktrees:
   `~/wt/luna-idiom-probe-1` through `-7`, and `~/wt/luna-idiom-target2` through
   `-7`. Use `git worktree remove`.
   - Before deleting each `luna/idiomatic-go-target<N>` branch, check that its
     commit is reachable from `luna/idiomatic-go-2026-11-03`.
   - Leave `~/wt/luna-idiom-go` alone.
2. **Create the round-two worktree** on top of round one, so that its 7 commits
   are included:

   ```bash
   cd /home/zach/darkpawns && git fetch -q
   git worktree add ~/wt/luna-idiom-r2 -b luna/idiomatic-go-round2 origin/luna/idiomatic-go-2026-11-03
   ```

   Then bring in this brief from main:

   ```bash
   git checkout origin/main -- docs/research/idiom-experiment/round2
   ```

   Commit that as `docs(research): add round-two brief`. Record the base SHA.
   Don't rebase during the run.
3. **Record the baseline** into `round2/run/baseline.tsv`.
   - The metrics script has its own `go.mod`, so run it from its own directory:

     ```bash
     cd docs/research/idiom-experiment/metrics && go run . ../../../.. pkg/game pkg/combat pkg/command
     ```

   - Also run the four-linter golangci command from round one's
     `PREDICTIONS.md`, and save its counts.
4. **Install gopls** with `go install golang.org/x/tools/gopls@latest`, and
   record the version.
5. **Refresh the open-PR file list.** Don't edit files that an open PR touches.

## Stage 1: measure what the census can see (before any refactor)

This stage is the research measurement. Do it on the base commit, before you
change any code.

1. **Define the population.** Write a script at `round2/run/population.py` (or
   Go) that lists every non-test function or method in `pkg/game`,
   `pkg/combat` and `pkg/command` that contains a string literal flowing into
   player-facing output. That means:
   - a literal passed straight to `SendMessage`, `Act`, `send_to_char`-style
     helpers, or the `fmt.Sprintf` feeding them;
   - or a literal assigned to a message variable that is sent within the same
     function.

   Write the exact rule at the top of the script. Output
   `round2/run/population.tsv` with columns `func`, `file`, `line`,
   `n_literals`. Sort it by `file`, then `line`. Record its sha256 in
   `round2/run/sample.md`. **Commit the script, the TSV and the hash before you
   draw the sample.**
2. **Draw the sample.** Pick 40 functions with Python's
   `random.Random(20261105).sample(rows, 40)` over the sorted rows. Write the
   sample to `round2/run/sample.tsv`.
3. **Choose each function's probe.** Probe `i` (0-based) uses
   `random.Random(20261105 + i)` to choose one of that function's output
   literals. The mutation is to replace the literal's first ASCII letter with
   `Q`, or with `Z` if the letter is already `Q`. That's a one-byte change.
   - Record the file, line and literal for each probe.
   - **Before running anything,** write down the scenarios you expect to catch
     that probe, in the column `predicted_scenarios`. If you think no scenario
     reaches the function, leave the column blank. This is a second
     measurement: how well an agent can predict coverage.
4. **Run each probe against the full census, not a targeted one.** Do this in a
   throwaway worktree at the base commit:

   ```bash
   scripts/census.sh start --name luna-r2-probe-<i>
   ```

   - Run at most 2 at a time, and use `wait --run` on each.
   - Never commit a probe. Remove each probe worktree when its census finishes.
   - The full census is what makes this unbiased: whether a probe gets caught
     must not depend on which scenarios anyone thought to run.
5. **Record the results** in `round2/run/probes.tsv` with these columns:
   - `i`, `func`, `file`, `line`, `literal`
   - `predicted_scenarios`
   - `census_dir`, `verdict`
   - `failed_scenarios`
   - `caught` (`yes` or `no`)
   - `agent_predicted_correctly` (`yes`, `no` or `partly`)

   A probe whose census goes red only on unrelated scenarios is an
   infrastructure problem. Rerun it once. If it still fails that way, record it
   as `infra`.

Don't add scenarios to catch green probes. That's fidelity work with its own
review. Green probes go in `round2/run/findings.md` as **"census blind spots"**,
one line each.

## Stage 2: Tier A, compiler-checked changes

These two changes need the local gates and the checks below. They don't need a
census per commit.

- **`interface{}` → `any`** across the scope.
  - `any` is an alias, so the change is spelling only.
  - Do one commit per package.
  - Don't change generic type constraints, and don't change anything outside
    the scope.
- **Rename unexported C-style identifiers** with
  `gopls rename -w <file>:<line>:<col> <newName>`. Use one commit per file, or
  per small group of files.
  - Before each rename, check the old name against all of these:
    - It must not appear in any string literal: `git grep -n '"[^"]*<old>'`.
    - It must not appear under `lib/` (the Lua scripts and world files).
    - It must not reach `%T` or `reflect`. In scope those are only at
      `pkg/game/world_movement.go:167` and `pkg/combat/body.go:11-12`.
  - If any check finds a match, skip that name and note why.
  - Never rename exported identifiers.
  - Never rename names that copy a C table or a C function on purpose, when a
    `// C:` comment ties the two together.

When Tier A is done, run one full census of the branch tip (not `--combined`)
as a check on the whole of Tier A.

## Stage 3: Tier B, extractions, with the RNG rule clarified

Use round one's loop:

1. local gates;
2. a targeted census;
3. a one-byte probe of the touched function;
4. at most 5 self-heal attempts, then abandon the target.

What changes is the RNG rule:

- **A block that contains draws may be extracted** into a helper, provided all
  of the following hold:
  - the helper has exactly one call site, at the block's original position;
  - the statements inside it keep their order;
  - every early `return`, `break` or `continue` in the block is carried back to
    the caller unchanged, for example by returning a bool the caller checks
    straight away.

  An extraction that meets these conditions leaves the draw sequence unchanged,
  and so does R3a. Containing a draw is not a reason to skip.
- **Still forbidden:**
  - moving a draw before or after an early return;
  - moving a draw into or out of a `&&` or `||`;
  - reordering a draw relative to any other statement;
  - putting a draw inside a helper that has more than one call site, or one
    called from a loop it wasn't in before (R3a, R3b, R3d).

Work through round one's ranked blueprint again under this rule, including
ranks 6 to 17. Record a target as **"Skipped: unsure"** only when you can name
the specific forbidden move it would need. Write that move in one line.

## When to stop

There is no time limit. Stop when Stage 1 is complete, Tier A is done, and every
ranked target has been kept, abandoned, or skipped with a named reason. Also
stop if 3 Tier B targets in a row are abandoned, and report what went wrong.

When you stop:

1. Rerun the metrics script and the lint counts into `round2/run/after.tsv`.
2. Write `round2/RESULTS.md`: one row per prediction, observed value next to
   predicted. It must include:
   - the probe catch rate, with its count out of 40 and the infra rows;
   - how often your own `predicted_scenarios` were right.
3. Write `round2/run/REPORT.md` with:
   - what you did;
   - what surprised you;
   - census blind spots;
   - proposed rule text, if any;
   - the protected-path check from round one's brief, with its empty output
     pasted in. Use the round-two base SHA.
4. Push `luna/idiomatic-go-round2`.
5. Open a **draft** PR titled
   `refactor: idiomatic Go round two over pkg/game, combat, command (experiment)`.
   - Say in it that it includes PR #2275's commits.
   - End the PR body with "Decisions for Zach".
   - Don't merge it.

The final combined census stays with Claude's reviewed batch, as in round one.
