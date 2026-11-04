# Brief: one night of making the game logic idiomatic Go (GPT-Luna)

You have a long, unattended run. Zach is asleep. Claude reviews your branch
in Zach's session in the morning. Nobody will answer questions during the
night, so every decision you need is either made here or covered by a rule
that says what to do when you're unsure.

Read these before you start:

- `AGENTS.md`;
- `docs/fidelity/RULEBOOK.md` (R1–R5);
- [`PREDICTIONS.md`](PREDICTIONS.md) (why this experiment exists and how it
  is measured).

Don't edit `PREDICTIONS.md`.

## The job

Much of `pkg/game`, `pkg/combat` and `pkg/command` still reads like C
translated line by line: 700-line functions, deep nesting, package-level
state, snake_case names, `interface{}` where a type is known. Make it read
like Go a careful Go programmer wrote. **Every player-facing byte and every
random-number draw must stay identical** (R1, R3). This is a refactor. If a
commit changes what any player could see, or how many times or in what
order the RNG is called, it's a bug, even if the census stays green.

You are not fixing fidelity bugs tonight. When you find one, log it (see
Deliverables) and leave the code as it is. A behaviour fix inside a refactor
commit makes both impossible to review (R5a).

## Setup (Stage 0)

1. Make a worktree. Never switch branches in `/home/zach/darkpawns`:

   ```bash
   cd /home/zach/darkpawns && git fetch -q
   git worktree add ~/wt/luna-idiom-go -b luna/idiomatic-go-2026-11-03 origin/main
   ```

   Record the base SHA. Don't rebase during the run, even if main moves.
2. Copy the metrics script's output for the base into
   `docs/research/idiom-experiment/run/baseline.tsv`:

   ```bash
   go run ./docs/research/idiom-experiment/metrics . pkg/game pkg/combat pkg/command
   ```

   Also run the golangci-lint command from `PREDICTIONS.md` and save its
   counts.
3. **Efficiency baseline.**
   - Build `./cmd/server` and record its size in bytes.
   - Find how to start a scratch server from the checkout on a free port.
     Never use port 4000, 7777, 7778 or 7779, and never touch a server you
     didn't start.
   - Measure, 5 times, the wall time from process start to the server
     accepting connections, and its peak RSS (`/usr/bin/time -v`). Record
     the medians.
   - Stop only the PIDs you started.
   - Write the exact procedure into `run/efficiency.md` so the end
     measurement repeats it exactly.
4. List the files touched by open PRs
   (`gh pr list --repo zax0rz/darkpawns --state open`, then
   `gh pr diff <n> --name-only`). **Don't edit those files tonight.** That
   avoids merge conflicts with other agents.

## Audit (Stage 1)

Write `run/blueprint.md`, a ranked list of refactor targets in scope:

- each function over 100 lines;
- each function nested 5 or more levels deep;
- each repeated block;
- each snake_case or C-style name that isn't exported;
- each `interface{}` with one real type.

For each target, note which commands reach it and which scenarios in
`cmd/dp-oracle-diff/scenarios/` exercise those commands. **Rank by how well
the census can see the code first, then by payoff.** Start where scenarios
exist. Code no scenario reaches comes last.

## Refactor loop (Stage 2)

One target per commit, small enough to review in five minutes. For each:

1. **Make the change.**
2. **Check it locally.** Run each of these on its own and check its exit
   status. Never chain gates with `;`.
   - `go build ./...`
   - `go vet ./...`
   - `go test` on the touched packages
   - `make fmt` (gofumpt)
   - `golangci-lint run` on the touched packages
   - `make deadcode-ratchet`
3. **Run a targeted census** on the scenarios that reach the touched code:

   ```bash
   scripts/census.sh start --name luna-idiom-<n> --scenarios a,b,c
   scripts/census.sh wait --run <dir>
   ```

   `wait` blocks up to 9 minutes. Call it again on exit code 3. Don't poll
   `status` or read `census.log`.
4. **Run a mutation probe (R5h).**
   - Make a second, throwaway worktree at the same commit.
   - Change one byte of one output string in one touched function. Pick a
     string on the branch you rewrote, not the most common path.
   - Run the same scenario list as its own census.
   - Record whether it went red, then remove the probe worktree.
   - **Never commit a probe.** You may start the probe census right after the
     real one; both can queue.
5. **Record the commit's row** in `run/telemetry.tsv` (columns below), then
   commit:
   - conventional message: `refactor(game): ...`;
   - say in the body what was transformed and why behaviour is unchanged;
   - end with your normal co-author trailer.

**Bounded self-healing**, from Google's skill: if a gate or the census goes
red, diagnose and fix it up to **5 times**. If it is still red, `git reset`
the change, log it in `run/abandoned.md` with the census directory, and
move to the next target. A red census means your refactor changed behaviour,
so the fix is in your refactor, never in the test.

**When the probe stays green**, the census can't see that code. Keep the
commit, mark it `probe=green` in telemetry, and take extra care. Do not add
scenarios to make it red tonight: that is fidelity work with its own review.

Every 8 kept commits, and once at the end, run a full combined census on the
branch tip:

```bash
scripts/census.sh start --combined --name luna-idiom-full-<k>
```

It isn't a release, so don't use `--release`. A FAIL row there means an
earlier commit slipped past its targeted census:

- find it by rerunning that one scenario at earlier commits;
- revert that commit;
- record it in `abandoned.md` as a **targeted-census miss**. That is
  valuable data.

## Allowed transforms

- **Extract function.** Split a long function into named helpers. Keep every
  statement in its original order, including statements that output text,
  draw random numbers, or change state.
- **Guard clauses.** Early returns to reduce nesting, only when the condition
  evaluation order and draw count are unchanged.
- **Rename unexported C-style identifiers** to Go names (`ch_name` →
  `chName`).
- **Typed constants** for raw integer flags, where the values stay identical.
- **Replace `interface{}`** with `any` or the concrete type.
- **Collapse repeated blocks** into one helper, when the blocks really are
  identical in effect.
- **`strings.Builder`** in place of repeated concatenation, with
  byte-identical results.
- **Delete a variable, branch or parameter that is truly unused.** Never
  delete a whole function: that goes through the deadcode ratchet process,
  not this run.

## Forbidden: these break fidelity in ways the census often misses

- **Changing any output string, format verb, padding, color code, line ending
  or message order** (R1, R1b).
- **Moving, merging, hoisting, skipping or duplicating any RNG call**
  (`number`, `dice`, `rand*` and their wrappers).
  - Don't move a draw past an early return.
  - Don't move a draw into or out of a `&&`/`||` short circuit.
  - Don't move a draw into a helper that is called a different number of
    times (R3a, R3b, R3d).
- **Turning an ordered list or slice into a map you range over.** Go map
  order is random. Also, sort stability: never swap a stable sort for
  `sort.Slice`.
- **Changing integer widths or signedness**, or arithmetic that depends on
  C truncation or overflow. Example: `int32` → `int` changes what overflows
  (R1a context).
- **Goroutines, channels or locks**, and anything else that changes timing
  or the order of operations (R3b, R3c).
- **Renaming or removing exported identifiers.**
  - It risks conflicts with other agents.
  - It breaks `tools/db-migrate`.
  - It bloats the diff.
- **Touching the `CustomData` field definition, the save format, or anything
  in `pkg/session`, `pkg/db`, `pkg/telnet` or `pkg/spells`** (out of scope:
  see PREDICTIONS).
- **Removing or rewording comments that cite C** (`// C: act.other.c:1629`
  and similar). They are the provenance trail. Move them with their code.
- **Protected paths.** Never edit any of these:
  - test files (`*_test.go`);
  - `cmd/dp-oracle-diff/` (including scenarios and the `expected_*` TSVs);
  - `internal/oraclediff/`;
  - `docs/fidelity/`;
  - `scripts/`;
  - `AGENTS.md`;
  - `src/`;
  - `darkpawns-c-oracle/`.

  If a test fails, your refactor is wrong. Editing a check to make it pass
  is exactly the failure the research studies, and it voids the whole run.

**When you are unsure** whether a transform preserves draw count or bytes,
don't make it. Write it in `run/blueprint.md` under "Skipped: unsure", with
one line saying why. A skipped target costs nothing. A wrong one costs a
morning.

## Hard rules (from AGENTS.md)

- Run censuses only through `scripts/census.sh`. Cancel only your own runs.
  Never delete locks.
- Never `git stash`.
- Never `pkill` or `killall`.
- Never switch branches in the main checkout.
- Never push to main, merge anything, deploy, or touch production.
- Don't invoke another agent, model or paid service (`claude`, `codex`,
  `ultrareview`, other CLIs or APIs). Claude reviews in Zach's session.
- Don't edit governing documents. If the night teaches a rule, propose the
  exact text in your report.

## When to stop

Stop at the first of:

- all ranked targets are done or skipped;
- 12 hours after you start;
- 3 targets in a row abandoned (something systematic is wrong: report it
  rather than pushing on).

Then:

1. Run the final full combined census on the tip.
2. Rerun the metrics script, the lint counts and the efficiency procedure
   into `run/after.tsv` and `run/efficiency.md`.
3. Push the branch:
   `git push -u origin luna/idiomatic-go-2026-11-03`.
4. Open a **draft** PR titled
   `refactor: idiomatic Go pass over pkg/game, combat, command (experiment)`.
   - Body: the before/after table, census run directories, and the "Decisions
     for Zach" section.
   - Don't merge it.
5. Remove probe worktrees. Leave `~/wt/luna-idiom-go` in place for review.

## Deliverables

All new files go under `docs/research/idiom-experiment/run/` on your branch.

- `baseline.tsv`, `after.tsv`: metrics script output and lint counts.
- `efficiency.md`: the procedure, plus before and after medians.
- `blueprint.md`: ranked targets, including "Skipped: unsure".
- `telemetry.tsv`, one row per attempted target. Columns:
  - `n`
  - `commit` (blank if abandoned)
  - `utc_start`, `utc_end`
  - `files`
  - `lines_added`, `lines_removed`
  - `transform` (one of the allowed transforms)
  - `functions`
  - `scenarios`
  - `census_dir`
  - `census_verdict`
  - `selfheal_attempts`
  - `probe_function`
  - `probe_census_dir`
  - `probe` (`red`, `green` or `none`)
  - `kept` (`yes`/`no`)
  - `notes`
- `abandoned.md`: each abandoned target and each targeted-census miss, with
  run directories.
- `findings.md`: latent port bugs you saw but didn't fix. Give each:
  - the Go file:line;
  - the C file:line you read in `src/` (R5g);
  - what differs;
  - how confident you are.

  These go to Linear later, after review.
- `REPORT.md`, a short report:
  - what you did;
  - the numbers against `PREDICTIONS.md`, row by row;
  - what surprised you;
  - anything systematic;
  - any proposed rule text.

Before you stop, check that `git diff --name-only <base>..HEAD` touches no
protected path. Paste the empty result into `REPORT.md`:

```bash
git diff --name-only <base>..HEAD | grep -E '_test\.go$|^cmd/dp-oracle-diff/|^internal/oraclediff/|^docs/fidelity/|^scripts/|^AGENTS\.md$|^src/|^pkg/(session|db|telnet|spells)/'
```
