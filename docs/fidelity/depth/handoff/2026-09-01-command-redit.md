# Depth-fidelity handoff — `redit`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `return` handoff. The special-procedure inventory remains exhausted, the
one blocked row `objmagic.sleep-entry-gates` remains queued after its single
cast-sleep vehicle, and the interpreter sweep advanced from `return` to
`redit`.

The source-order audit did not repick `retreat` (owned by `escape`),
`retrieve` (owned by `spec-procs`), or `ride`/`roomflags` (owned by `mount` and
`gen-tog`). `redit` is the second registered C subcommand of the currently
blocked `do_olc` family; `medit` was already blocked after its own two honest
attempts. The next unclaimed interpreter-table family is `reallyquit` at
`src/interpreter.c:657`.

Frontier before this slice: 2,982 total; 2,909 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,986 total; 2,909 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:656 */
{ "redit"    , POS_DEAD    , do_olc      , LVL_BUILDER, SCMD_OLC_REDIT},
```

`src/olc.c:80-275` first rejects NPC callers, parses two arguments, handles
`save`, selects the current room for an empty `redit`, rejects nonnumeric input
with `Yikes!  Stop that, someone will get hurt!`, allocates descriptor OLC
state, resolves the target zone, rejects an unknown zone with `Sorry, there is
no zone for that number!`, checks zone permission, and then enters
`CON_REDIT`. For a valid room, `redit_setup_existing`/`redit_setup_new` in
`src/redit.c:65-92` builds the editor state and `redit_disp_menu` emits the
full `REDIT_MAIN_MENU`; subsequent descriptor input routes through
`redit_parse` (`src/interpreter.c:1723-1731`, `src/redit.c:673-709`).

## Evidence and blocked verdict

Scenarios:

- `cmd/dp-oracle-diff/scenarios/redit-entry-depth.txt`
- `cmd/dp-oracle-diff/scenarios/redit-session-depth.txt`

Manifest: `docs/fidelity/depth/redit.tsv` (four blocked rows)

The first honest clean-main vehicle covered nonnumeric and unknown-zone inputs.
C emitted respectively `Yikes!  Stop that, someone will get hurt!` and
`Sorry, there is no zone for that number!`; Go has no registered `redit`/OLC
handler and emitted `Huh?!?` for both.

The second honest vehicle issued empty `redit`. C selected the actor's current
room and emitted the complete room editor menu, entering the descriptor-driven
`CON_REDIT` state; Go again emitted `Huh?!?`. This confirms a missing OLC
descriptor/state-machine surface, not a safe one-line response correction.

After exactly two genuine RED attempts, the four reachable cases are marked
`blocked` in `docs/fidelity/depth/redit.tsv` with exact C sites and the reason
no substitute was invented. No Go behavior was changed, and no `src/` or
C-oracle file was edited. The queue advances under the two-attempt rule.

## Verification and integration

All required local gates passed on the blocked-evidence branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-redit`

Feature commit: `81a4ac048` (`docs: record redit proof gap (R1/R2/R4/R5e)`)

Feature PR: #1115 — hosted lint, security, and test checks were green;
build-and-push and deploy were skipped by workflow conditions. It was
self-merged as main commit `7061fe0ab` only after the required hosted checks
were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R4 (no
invention), R5 (process discipline), and R5e (verify the actual C call path).
The shared `do_olc` boundary and source-order claim are maintained under R5b
and R5c.
