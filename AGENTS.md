# Dark Pawns — AI Agent Instructions

## Prime Directive — Fidelity Law

Dark Pawns is a **1:1 faithful port**: the Go server must emit the *same player-facing bytes* as the original C. "The game is the game." Before you change any player-observable behavior, read the port bible:

**[`docs/fidelity/RULEBOOK.md`](docs/fidelity/RULEBOOK.md)** — the C→Go translation law (R1–R5).

**Documentation map** — read these before touching the matching surface:

| Doc | Governs |
|---|---|
| [`docs/fidelity/RULEBOOK.md`](docs/fidelity/RULEBOOK.md) | C→Go port fidelity law (R1–R5) |
| [`docs/fidelity/oracle-differential-testing.md`](docs/fidelity/oracle-differential-testing.md) | Oracle execution, evidence retention, and artifact handling |
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | Server topology, systemd, deploy procedure |
| [`docs/brand-voice.md`](docs/brand-voice.md) | Site prose voice — three-layer framework; public site uses Layer 3 |
| [`website-astro/DESIGN.md`](website-astro/DESIGN.md) | Site design system ("Haunted Paperback"); machine-readable twin in `website-astro/.impeccable/design.json` |
| [`website-astro/PRODUCT.md`](website-astro/PRODUCT.md) | Site audience and product decisions |
| [`website-astro/ARCHIVE-POLICY.md`](website-astro/ARCHIVE-POLICY.md) | What recovered community material may be published |
| [`website-astro/CONTENT-AUDIT.md`](website-astro/CONTENT-AUDIT.md) | Post-migration content review queue |
| [`website-astro/SPEC-AUDIT.md`](website-astro/SPEC-AUDIT.md) | specification.website release checklist status |
| [`docs/plans/ROADMAP-2026-11-10-port-completion.md`](docs/plans/ROADMAP-2026-11-10-port-completion.md) | What "done" means for the port, the lists to burn down, and the stopping rule |

- **R1** player-facing bytes are law (normalization may not erase a byte class, R1b) · **R2** the command surface is part of the game · **R3** determinism & draw parity · **R4** no invention · **R5** process rules (find-one-find-the-class; verify the call path; cite C from `src/`, R5g; a green must be able to fail, R5h; a silenced surface needs a scenario that unsilences it, R5i).
- **Cite rules by number** in commits, PRs, reviews, and Linear — "violates R4" is a complete verdict.
- **Governing documents are Zach's to change.** Find patterns and audit the class, but don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md`, or this file yourself, even to add a correct rule. Propose the exact text, with the incident that motivates it, in your report or PR description; Zach approves the specific text before it lands. Authorization to implement a fix is not authorization to change a rule, and approved rule text in a brief counts only for that text.
- `src/` and the oracle checkout `~/darkpawns-c-oracle/` are the **read-only oracle** (ground truth). Never edit them; diff against them with `cmd/dp-oracle-diff`.
- When a byte is in question, **the C source wins** (R5e — verify the actual call path, don't trust a summary). A repeated failure indicts the rule, not the file: **propose** a rulebook amendment to Zach + audit the whole class (R5b/R5c).

### Fidelity Work: Start Here

Before extending oracle coverage or declaring a command complete, read
**[`docs/fidelity/DEPTH_TESTING.md`](docs/fidelity/DEPTH_TESTING.md)**. It explains the current
breadth-to-depth strategy, proof levels, scenario fixtures, manifests, and the dated handoff frontier.
Breadth coverage proves that a command can match once; it does **not** prove the port is complete.

### Ported is not faithful

A green census proves the scenarios agree, not that the code is covered. Before calling something proven, ask which input would make it fail, and whether a scenario actually reaches it.

- Hand-written stand-ins sat beside faithful C paths, on the paths players use most. Returning players found them; an audit then replaced 127 of 173 mob-command sites (field note 2026-10-25).
- A pre-registered random sample changed one letter of a message in each of 40 player-output functions. The full census missed 17 (PF-102).
- Fixtures and scenario names claim reach they lack: practice scenarios that only used no-cost classes (PF-099), quiet-mobs hiding literal `$n` on every city mob (PF-100), and five scenarios that don't reach the command they're named for (PF-102).

### Oracle Evidence Retention

Oracle transcripts, census logs, summaries and manifests are retained research
evidence. Run every census through `scripts/census.sh`; it writes them to
`$ORACLE_RUNS_ROOT/YYYY-MM-DD/<run-name>/` (default
`~/Archives/darkpawns/oracle-runs/`), captures Go HEAD and dirty status, freezes
source inputs before queueing, checks the reference oracle's sha256, and writes
`MANIFEST.md` itself. Every attempt retains both engines' normalized blocks,
draw logs when enabled, and engine logs; retries never overwrite a red attempt.

```bash
scripts/census.sh start --name <issue-or-run>            # full census, returns after registration
scripts/census.sh start --name <run> --scenarios a,b,c   # targeted
scripts/census.sh start --combined --name <run>          # full seed-1 corpus plus claimed pairs
scripts/census.sh start --combined --release --draw-parity --name <run> # exclusive release gate after draining
scripts/census.sh wait --run <returned-directory>        # bounded wait; ONE line
```

The host/user shares at most **72 scenario workers**, at most **36 per run**.
`--jobs N` sets the run's maximum across all seeds; it does not allocate another
host budget. Registration queues by default. Memory admission stops below
4 GiB MemAvailable and resumes at 4.5 GiB; a separate serialized build slot
reserves build headroom. Running scenarios finish without preemption.
`--release` requires `--combined`, drains active ordinary scenarios/builds,
then gives the release exclusive use. Ordinary combined runs are not releases.
`--no-queue` refuses with exit 5 when immediate admission is unavailable.
`--exclusive` is available for exclusive ordinary diagnosis/rollback.

When a brief requires both a full census and a claims census, use `--combined`.
It executes their union once, retains the full and claims result files and
separate verdicts, and is clean only when both verdicts are CLEAN or
CLEAN_AFTER_RECHECK. Separate full and `--claims` runs remain available for
diagnosis and comparisons. A release ships only from its exact-commit clean
combined gate. A release census must pass `--draw-parity` and leave a
`draw-results.tsv`; confirm the file exists before reading the verdict (R3).
The `v0.1.9` release census omitted the flag and had no draw log.

Save the directory printed by `start` and use `--run` for every wait/status or
cancellation. Bare reports refuse to choose among multiple unfinished runs.
`wait` exits 0 (clean), 1 (not clean), 3 (queued/running: call `wait` again),
or 4 (dead/missing/ambiguous). Its one line includes any draw tally.

**Waiting costs tokens; the census does not.** While a census runs:
- Do not poll in a loop, `sleep`-and-check, or re-run `status` repeatedly. Call
  `wait --run` (each call blocks up to 9 minutes), or end your turn and say the
  census is queued/running; Zach gets a desktop notification when it finishes.
- Never `cat`, `tail` or `grep` a running `census.log`. When it has finished,
  read `summary.txt`, and read `results.tsv` only for the rows you need.
- Concurrent registrations are allowed through the updated wrapper. Do not
  change runtime variables, use test namespaces, launch duplicate jobs, or
  call the underlying runner directly to evade admission. A queued line with
  `eta=unknown` is not an instruction to retry start.
- Cancel only your own run, with `census.sh cancel --run <returned-directory>`.
  Never delete runtime locks, kill another run, or reclaim a live worker's
  permit. Upgrade older worktrees before expecting concurrent starts; their
  legacy exclusive lock remains respected during migration.

**Choose the smallest census that can fail on your change** (R5h):
- **Full census**: before every release, on the exact commit being tagged; a
  release ships only from a CLEAN or CLEAN_AFTER_RECHECK combined census, and
  any fix after a red census reruns it in full. Between releases, PRs changing
  player-facing output or game behavior run targeted scenarios covering it.
- **Targeted** (`--scenarios`): tests, comments, docs, or one statement whose
  effect a named set covers. Name the scenarios and why. A brief can require more.
- **None**: docs-only or website-only changes.

**The final full census is batched, and Claude runs it.** Coding agents run
necessary targeted censuses and open their PRs without a final full/combined
census. Claude stacks reviewed PRs over current `origin/main` and runs one
`--combined --release` census for the batch. Attribute a FAIL row by rerunning
only that scenario against each PR. Do not start a separate full census per PR.
Start the required target, then continue independent work in another worktree;
retain the exact run directory instead of waiting idle or following another
agent's result.

**Infrastructure rows.** The tool re-runs INFRA/TIMEOUT rows alone once;
CLEAN_AFTER_RECHECK counts as clean. Do not rerun the whole census for them.
FAIL, STALE and UNPINNABLE are never retried by the census wrapper: investigate.
EXPECTED scenarios remain ledger-backed known divergences. A concurrency
acceptance proof additionally compares first attempts, both transcripts and
draw rows: matching final verdicts after new flakes are not sufficient (R3/R5h).

### Reference Oracle Binary

The shared `DP_ORACLE_BIN` (`~/darkpawns-c-oracle/bin/circle`) is built
`-g -O0 -fcommon -ffile-prefix-map=$PWD=.` from `dp-oracle-seam` (the prefix map
makes any checkout build identical bytes). Optimization level changes transcript
bytes through the C source's undefined behaviour (R1a), so a binary built any
other way is not the reference. Never replace the shared binary except by
[`docs/DEV-SETUP.md`](docs/DEV-SETUP.md) → *Promoting a rebuilt reference
binary*: keep the old binary and both SHA-256s in the run's evidence, and run
the full census green against the candidate first. Taught 2026-09-28: the
reference had silently been an `-O2` build, and the port had copied its
truncated reports.

### Research Continuity

Dark Pawns is also an open, ongoing research artifact. Before making paper-level
claims or running a research-writing task, read **[`docs/research/README.md`](docs/research/README.md)**
and update **[`docs/research/EVIDENCE_LEDGER.tsv`](docs/research/EVIDENCE_LEDGER.tsv)**. Treat cron and
agent prose as field notes until its citations survive contact with the repository, oracle output,
Git history, or another named primary artifact.

## Build & Verify

```bash
go build ./...          # Full build — must pass before committing
go vet ./...            # Static analysis — must pass before committing
go test ./...           # All tests — must pass before committing
go test ./pkg/game/...  # Game package tests specifically
golangci-lint run ./... # Full lint (uses .golangci.yml; CI pins the version)
make modernize-check    # CI enforces the reviewed go fix modernizers
make deadcode-ratchet   # CI enforces it; the baseline only shrinks
```

**Toolchain.** The Go version is whatever `go.mod` says on `origin/main` (Go
1.27.4 as of 2026-11-11); never quote it from memory or from the shared
checkout. Every `go` command needs `GOEXPERIMENT=nojsonv2`. The Makefile
exports it; a direct `go test` must export it too, because Go 1.27's default
JSON encoder changes invalid-UTF-8 bytes and the world text is Latin-1 (R1).
`internal/buildcompat` fails without it. Run golangci-lint at the version
`.github/workflows/ci.yml` pins: an older binary built with an older Go refuses
to load the config, and that is not a lint pass.

`tools/db-migrate` is a separate Go module: the root `go build ./...`, `go vet ./...`
and `go test ./...` don't compile it. Any change to an exported symbol in a package it
imports (such as `pkg/game`) must also pass `make db-migrate-test`. Its appcompat tests
skip unless `DP_MIGRATE_TEST_DB_URL` points at a disposable PostgreSQL, so a green run
without it proves compilation only.

**NEVER commit without running all four.** Subagents that self-report passing builds have lied before.

Run each gate on its own and check its exit status. Never chain them with `;` or
end a chain with `; echo "passed"`: the line reports success when a gate failed,
and that has hidden real breakage twice.

### Worktrees

Never switch branches in the main checkout (`/home/zach/darkpawns`): other agents
work there. Do all branch work in a worktree under `~/wt/`, named
`<agent>-<issue>` (e.g. `~/wt/glm-dp-1435`):

```bash
git -C /home/zach/darkpawns fetch origin
git worktree add ~/wt/<agent>-<issue> -b <branch> origin/main
```

**Fetch before you read or start.** The shared checkout at
`/home/zach/darkpawns` is not kept current, and other agents leave it wherever
they leave it. Before reading any doc, brief or rule, and before branching, run
`git fetch origin`, then read from your worktree or with
`git show origin/main:<path>`. Rebase on `origin/main` again right before you
report. Taught 2026-11-10, when a new agent reported the previous Go version
from the shared checkout.

Remove it with `git worktree remove ~/wt/<agent>-<issue>` once its PR merges.
Don't put worktrees in `~` itself (230 accumulated there by 2026-10-14) or in
`/tmp`: systemd-tmpfiles deletes files older than 30 days one at a time, which
silently corrupts a long-lived worktree. Taught 2026-10-13, when a
`git checkout main` in the shared checkout put an agent's commit on local main.

Never run `git stash` in this repository. Stash refs are shared by every worktree, so
`git stash pop` can apply another agent's stash in your tree. To set work aside, commit it
to your branch (a `wip:` commit is fine). Taught 2026-10-24, when an agent's revert-proof
experiment popped a codex agent's stash into its own worktree.

Never stop processes by name (`pkill`, `killall`, `pkill -f`). Stop only the
PIDs you started, or use `census.sh cancel --run <dir>` for your own census.
Other agents' servers and censuses run on the same host. Taught 2026-10-25,
when a play-lab cleanup killed servers belonging to the v0.1.5 release census.

**Handoffs.** When a brief says another agent or Zach will review, approve or
decide something, stop and report to Zach. Never invoke another agent, model or
paid service yourself (`claude`, `ultrareview`, other CLIs or APIs) to obtain
it. Taught 2026-10-26, when an agent read "Claude reviews" as a command to run.

## Project Overview

Dark Pawns is a Go MUD server, ported from the C original: CircleMUD 3.0, of the DikuMUD family (`src/config.c:249`). About 165K lines of Go and 196K lines of tests; the 66 C files in `src/` are the reference. Every C file has a Go counterpart, so don't start a fresh port of a whole file. But ported is not faithful: see *Ported is not faithful* under Fidelity Work.

### Architecture

- `cmd/server/` — MUD server entrypoint
- `pkg/game/` — Core game logic (combat, skills, objects, rooms, mobs, world state)
- `pkg/session/` — Player session handling, commands, wizard commands
- `pkg/parser/` — Zone/obj/mob/world file parsing (Diku format)
- `pkg/combat/` — Combat formulas and damage calculation
- `pkg/command/` — skill and admin command handlers
- `pkg/dprng/` — the C-compatible `number()`/`dice()` stream; every game draw goes through it (R3)
- `pkg/spells/` — Spell system (saving throws, damage, affect spells)
- `pkg/telnet/` — Telnet protocol handling
- `pkg/db/` — Player persistence (SQLite only; the legacy PostgreSQL converter is isolated under `tools/db-migrate/`)
- `pkg/scripting/` — Lua mob, object and room scripts (`lib/world/scripts`)
- `cmd/dp-oracle-diff/`, `internal/oraclediff/` — the oracle differential harness

### Key Conventions

- **ObjectLocation system**: Objects track location via tagged union `ObjectLocation` in `location.go`. Use `LocInventoryPlayer()`, `LocEquipped()`, `LocNowhere()` constructors. Never manipulate `Location` fields directly.
- **Runtime state**: `ObjectRuntimeState` and `MobRuntimeState` replace `CustomData map[string]interface{}`. Use typed fields. `CustomData` is the escape hatch for truly dynamic keys.
- **Equipment**: `Equipment` struct with typed `EquipmentSlot` constants. Use `Equip()`/`Unequip()`/`UnequipItem()` — never manipulate `Slots` map directly.
- **Error handling**: Player-facing operations (item transfer, equipment, shop) MUST check error returns. Use `slog.Error()` on rollback failures. Never use `#nosec G104` to suppress error checks in player-facing code.
- **`fmt.Fprintf` to `io.Writer`**: These are user-facing output (MUD text). Do NOT convert to `slog` — they stay as formatted writes.
- **Formatting is gofumpt, not gofmt**: CI runs `gofumpt -l .` and fails on any diff. gofumpt is a strict superset of gofmt, so it's the *only* formatter to run — `gofmt` alone will pass locally but still fail CI. Run `make fmt` before committing, and `make hooks` once per clone to install the pre-push hook that enforces it.

### What NOT to Do

- Do not start a fresh port of a whole C file. When you find a hand-written stand-in for a C path, replace it with the real C path (R4, R5c).
- Do not convert `fmt.Fprintf(player.Writer, ...)` to slog
- Do not remove `CustomData` field (it's the escape hatch)
- Do not change save file format
- Do not use `#nosec G104` on new code — handle the error

### #nosec Annotations

The codebase has about 370 `#nosec` annotations. Most are:
- **G404 (weak random)** — about 270 intentional `math/rand` uses for dice, combat and MUD mechanics. Leave these alone.
- **G703 (taint)** — about 40, the oracle harness's trusted scratch-path boundaries.
- **G304 (file path)** — about 16. Use `filepath.Clean()` and explicit permissions.
- **G101, G115, G104** — small counts. G104 (errcheck) has 5 left, and they are the cleanup targets; handle the error or log it.

## Commit Messages

Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

## Testing

Tests live alongside source files as `*_test.go`. Run `go test ./...` after any change.
The game package has object movement tests that validate the ObjectLocation system — keep those green.

## Website Development & Deployment

The Dark Pawns website is a static site built using **Astro** and served via Caddy.

### Core Codebase Location
* Authored website source lives in **`website-astro/`**, including page scripts under `website-astro/src/scripts/`.
* **`website/` is not a second site.** Hugo was removed once the migration completed; what remains there is shared, non-Astro material: `static/` (Astro's public directory: generated map and database assets, `llms.txt`, `.well-known/`), `scripts/` (the world and database parsers the build runs), `deploy/` (the Caddy configuration), and `data/history.toml`. Do not add pages, layouts or templates there.
* **NEVER** edit or build website source on the production server. `/srv/darkpawns/` is only the deployed document root (the `website/` directory name is historical).

### Design Aesthetics & Philosophy
* **"Haunted Paperback" style** (see `website-astro/DESIGN.md`): worn cream paper backgrounds, charcoal ink text, one oxblood accent, flat ink, serif typography.
* **Asset Pipeline**: Authored JavaScript lives in `website-astro/src/scripts/` and must be imported by Astro so Vite fingerprints it. Generated assets under `website/static/` keep stable public URLs and are served as-is.

### Automated Deployment Pipeline
To prevent compiling from the wrong directory or syncing outdated files, **always use the Makefile target** in the root of the repository:

```bash
make deploy-site
```

This target automatically executes the complete, secure deployment sequence:
1. Regenerates map and database assets from the authoritative world files.
2. Runs voice, content-inventory, and Astro build checks.
3. Generates the Caddy permanent-redirect table.
4. Syncs `website-astro/dist/` to the configured production document root. `DEPLOY_USER` and `DEPLOY_HOST` are required explicitly.

**Always dry-run first and read the deletions.** The sync uses `--delete`, so anything production serves that your branch does not build is removed:

```bash
rsync -azn --delete --itemize-changes \
  website-astro/dist/ "$DEPLOY_USER@$DEPLOY_HOST:$DEPLOY_PATH" | grep '^\*deleting'
```

Only stale build artifacts should appear: superseded `_astro/*.css` hashes, Markdown twins for routes that no longer emit them, old `.bak` files. If a real page or image is listed, stop: its source is missing from the branch you are deploying. Commit the source and dry-run again rather than dropping `--delete`. This check caught a published blog post and its illustrations on 2026-09-04, whose only copy was untracked files in another worktree.
